import { readdirSync, readFileSync, writeFileSync, mkdirSync, existsSync } from 'fs';
import { join } from 'path';
import chalk from 'chalk';
import { parseSkillYaml, parseAgentYaml } from '../utils/yaml-loader.js';
import { lintSkill } from '../linters/rules.js';
import { generateSkillMd, countWords } from '../generators/skill-generator.js';
import { generateAgentMd } from '../generators/agent-generator.js';

const WORD_LIMIT = 500;

export interface BuildResult {
  success: boolean;
  error?: string;
  skillsGenerated: number;
  agentsGenerated: number;
  warnings: string[];
}

export function runBuild(skillsDir: string, agentsDir: string, outputDir: string): BuildResult {
  const warnings: string[] = [];

  // 1. Parse and lint all skills
  const skillFiles = existsSync(skillsDir)
    ? readdirSync(skillsDir).filter((f) => f.endsWith('.skill.yaml'))
    : [];

  const agentFiles = existsSync(agentsDir)
    ? readdirSync(agentsDir).filter((f) => f.endsWith('.agent.yaml'))
    : [];

  let hasLintErrors = false;

  for (const file of skillFiles) {
    const content = readFileSync(join(skillsDir, file), 'utf-8');
    const parsed = parseSkillYaml(content);
    if (!parsed.success) {
      return { success: false, error: `Parse error in ${file}: ${parsed.error}`, skillsGenerated: 0, agentsGenerated: 0, warnings };
    }
    const lintResults = lintSkill(parsed.data);
    const errors = lintResults.filter((r) => r.severity === 'error');
    if (errors.length > 0) {
      hasLintErrors = true;
    }
  }

  if (hasLintErrors) {
    return { success: false, error: 'Build failed: lint errors found. Fix errors before building.', skillsGenerated: 0, agentsGenerated: 0, warnings };
  }

  // 2. Generate skills
  let skillsGenerated = 0;
  for (const file of skillFiles) {
    const content = readFileSync(join(skillsDir, file), 'utf-8');
    const parsed = parseSkillYaml(content);
    if (!parsed.success) continue;

    const md = generateSkillMd(parsed.data);
    const wordCount = countWords(md);
    if (wordCount > WORD_LIMIT) {
      warnings.push(`Skill "${parsed.data.skill}" generates ${wordCount} words (limit: ${WORD_LIMIT}). Consider simplifying.`);
    }

    const skillOutDir = join(outputDir, 'skills', parsed.data.skill);
    mkdirSync(skillOutDir, { recursive: true });
    writeFileSync(join(skillOutDir, 'SKILL.md'), md);
    skillsGenerated++;
  }

  // 3. Generate agents
  let agentsGenerated = 0;
  for (const file of agentFiles) {
    const content = readFileSync(join(agentsDir, file), 'utf-8');
    const parsed = parseAgentYaml(content);
    if (!parsed.success) {
      warnings.push(`Agent ${file}: ${parsed.error}`);
      continue;
    }

    const md = generateAgentMd(parsed.data);
    const agentOutDir = join(outputDir, 'agents');
    mkdirSync(agentOutDir, { recursive: true });
    writeFileSync(join(agentOutDir, `${parsed.data.agent}.md`), md);
    agentsGenerated++;
  }

  return { success: true, skillsGenerated, agentsGenerated, warnings };
}

export function printBuildResult(result: BuildResult): void {
  if (!result.success) {
    console.log(chalk.red(`Build failed: ${result.error}`));
    return;
  }

  console.log(chalk.green(`Build complete:`));
  console.log(`  Skills generated: ${result.skillsGenerated}`);
  console.log(`  Agents generated: ${result.agentsGenerated}`);

  if (result.warnings.length > 0) {
    console.log(chalk.yellow('\nWarnings:'));
    for (const w of result.warnings) {
      console.log(`  ${chalk.yellow('!')} ${w}`);
    }
  }
}
