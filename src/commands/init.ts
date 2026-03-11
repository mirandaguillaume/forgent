import { mkdirSync, writeFileSync, existsSync, copyFileSync } from 'fs';
import { join, dirname } from 'path';
import { fileURLToPath } from 'url';

const __dirname = dirname(fileURLToPath(import.meta.url));

export interface InitResult {
  alreadyInitialized: boolean;
  path: string;
}

export function initProject(targetDir: string): InitResult {
  const configPath = join(targetDir, 'forgent.yaml');

  if (existsSync(configPath)) {
    return { alreadyInitialized: true, path: targetDir };
  }

  // Create directories
  mkdirSync(join(targetDir, 'skills'), { recursive: true });
  mkdirSync(join(targetDir, 'agents'), { recursive: true });

  // Write config
  const config = `# Forgent Project Configuration
version: "0.1.0"
skills_dir: skills
agents_dir: agents
`;
  writeFileSync(configPath, config);

  // Copy example skill
  const templatePath = join(__dirname, '..', '..', 'templates', 'skill.yaml');
  if (existsSync(templatePath)) {
    copyFileSync(templatePath, join(targetDir, 'skills', 'example.skill.yaml'));
  } else {
    // Inline fallback if running from dist
    const fallback = `skill: example
version: "0.1.0"
context:
  consumes: []
  produces: []
  memory: short-term
strategy:
  tools: []
  approach: sequential
guardrails: []
depends_on: []
observability:
  trace_level: minimal
  metrics: []
security:
  filesystem: none
  network: none
  secrets: []
negotiation:
  file_conflicts: yield
  priority: 0
`;
    writeFileSync(join(targetDir, 'skills', 'example.skill.yaml'), fallback);
  }

  return { alreadyInitialized: false, path: targetDir };
}
