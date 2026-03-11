import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtempSync, rmSync, mkdirSync, writeFileSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';
import { lintDirectory } from '../../src/commands/lint.js';

const validSkillYaml = `
skill: good-skill
version: "1.0.0"
context:
  consumes: []
  produces: []
  memory: short-term
strategy:
  tools: [read_file]
  approach: sequential
guardrails:
  - max_tokens: 1000
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

const badSkillYaml = `
skill: bad-skill
version: "1.0.0"
context:
  consumes: []
  produces: [output]
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
  filesystem: full
  network: none
  secrets: []
negotiation:
  file_conflicts: yield
  priority: 0
`;

describe('forgent lint command', () => {
  let tempDir: string;

  beforeEach(() => {
    tempDir = mkdtempSync(join(tmpdir(), 'forgent-lint-'));
    mkdirSync(join(tempDir, 'skills'));
  });

  afterEach(() => {
    rmSync(tempDir, { recursive: true, force: true });
  });

  it('should report no issues for a well-configured skill', () => {
    writeFileSync(join(tempDir, 'skills', 'good.skill.yaml'), validSkillYaml);
    const result = lintDirectory(join(tempDir, 'skills'));
    expect(result.totalFiles).toBe(1);
    expect(result.errors).toBe(0);
  });

  it('should report errors for a badly configured skill', () => {
    writeFileSync(join(tempDir, 'skills', 'bad.skill.yaml'), badSkillYaml);
    const result = lintDirectory(join(tempDir, 'skills'));
    expect(result.totalFiles).toBe(1);
    expect(result.errors).toBeGreaterThan(0);
    expect(result.totalIssues).toBeGreaterThan(0);
  });

  it('should handle multiple skill files', () => {
    writeFileSync(join(tempDir, 'skills', 'good.skill.yaml'), validSkillYaml);
    writeFileSync(join(tempDir, 'skills', 'bad.skill.yaml'), badSkillYaml);
    const result = lintDirectory(join(tempDir, 'skills'));
    expect(result.totalFiles).toBe(2);
  });

  it('should report parse errors for invalid YAML', () => {
    writeFileSync(join(tempDir, 'skills', 'broken.skill.yaml'), 'not: valid: yaml: [[[');
    const result = lintDirectory(join(tempDir, 'skills'));
    expect(result.errors).toBeGreaterThan(0);
  });

  it('should handle empty directory', () => {
    const result = lintDirectory(join(tempDir, 'skills'));
    expect(result.totalFiles).toBe(0);
    expect(result.totalIssues).toBe(0);
  });

  it('should ignore non-skill files', () => {
    writeFileSync(join(tempDir, 'skills', 'readme.md'), '# Hello');
    writeFileSync(join(tempDir, 'skills', 'good.skill.yaml'), validSkillYaml);
    const result = lintDirectory(join(tempDir, 'skills'));
    expect(result.totalFiles).toBe(1);
  });
});
