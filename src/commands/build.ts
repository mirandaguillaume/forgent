import { readdirSync, readFileSync, writeFileSync, mkdirSync, existsSync } from 'fs';
import { join } from 'path';
import chalk from 'chalk';
import { parseSkillYaml, parseAgentYaml } from '../utils/yaml-loader.js';
import { lintSkill } from '../linters/rules.js';
import { generateSkillMd, countWords } from '../generators/skill-generator.js';
import { generateAgentMd } from '../generators/agent-generator.js';
import { checkSkillOrdering } from '../analyzers/skill-ordering.js';
import type { SkillBehavior } from '../model/skill-behavior.js';

const WORD_LIMIT = 500;

export type BuildTarget = 'claude';

const TARGET_DEFAULTS: Record<BuildTarget, string> = {
  claude: '.claude',
};

export function getOutputDir(target: BuildTarget, override?: string): string {
  return override ?? TARGET_DEFAULTS[target];
}

export interface BuildResult {
  success: boolean;
  error?: string;
  target: BuildTarget;
  outputDir: string;
  skillsGenerated: number;
  agentsGenerated: number;
  warnings: string[];
}

export function runBuild(skillsDir: string, agentsDir: string, outputDir: string, target: BuildTarget = 'claude'): BuildResult {
  const warnings: string[] = [];

  // 1. Parse all skills upfront (single pass)
  const skillFiles = existsSync(skillsDir)
    ? readdirSync(skillsDir).filter((f) => f.endsWith('.skill.yaml'))
    : [];

  const agentFiles = existsSync(agentsDir)
    ? readdirSync(agentsDir).filter((f) => f.endsWith('.agent.yaml'))
    : [];

  const skillMap = new Map<string, SkillBehavior>();
  let hasLintErrors = false;

  for (const file of skillFiles) {
    const content = readFileSync(join(skillsDir, file), 'utf-8');
    const parsed = parseSkillYaml(content);
    if (!parsed.success) {
      return { success: false, error: `Parse error in ${file}: ${parsed.error}`, target, outputDir, skillsGenerated: 0, agentsGenerated: 0, warnings };
    }
    skillMap.set(parsed.data.skill, parsed.data);
    const lintResults = lintSkill(parsed.data);
    const errors = lintResults.filter((r) => r.severity === 'error');
    if (errors.length > 0) {
      hasLintErrors = true;
    }
  }

  if (hasLintErrors) {
    return { success: false, error: 'Build failed: lint errors found. Fix errors before building.', target, outputDir, skillsGenerated: 0, agentsGenerated: 0, warnings };
  }

  // 2. Generate skills from already-parsed data
  let skillsGenerated = 0;
  for (const skill of skillMap.values()) {
    const md = generateSkillMd(skill);
    const wordCount = countWords(md);
    if (wordCount > WORD_LIMIT) {
      warnings.push(`Skill "${skill.skill}" generates ${wordCount} words (limit: ${WORD_LIMIT}). Consider simplifying.`);
    }

    const skillOutDir = join(outputDir, 'skills', skill.skill);
    mkdirSync(skillOutDir, { recursive: true });
    writeFileSync(join(skillOutDir, 'SKILL.md'), md);
    skillsGenerated++;
  }

  // 3. Generate agents with resolved skill tools
  let agentsGenerated = 0;
  for (const file of agentFiles) {
    const content = readFileSync(join(agentsDir, file), 'utf-8');
    const parsed = parseAgentYaml(content);
    if (!parsed.success) {
      warnings.push(`Agent ${file}: ${parsed.error}`);
      continue;
    }

    // Resolve referenced skills for tool mapping
    const resolvedSkills = parsed.data.skills
      .map((name) => skillMap.get(name))
      .filter((s): s is SkillBehavior => s !== undefined);

    if (resolvedSkills.length < parsed.data.skills.length) {
      const missing = parsed.data.skills.filter((name) => !skillMap.has(name));
      warnings.push(`Agent "${parsed.data.agent}": unresolved skills [${missing.join(', ')}]. Tool list may be incomplete.`);
    }

    // Check skill ordering for sequential agents
    const orderingIssues = checkSkillOrdering(parsed.data, skillMap);
    for (const issue of orderingIssues) {
      warnings.push(`Agent "${parsed.data.agent}": ${issue.message}`);
    }

    const md = generateAgentMd(parsed.data, resolvedSkills, outputDir);
    const agentOutDir = join(outputDir, 'agents');
    mkdirSync(agentOutDir, { recursive: true });
    writeFileSync(join(agentOutDir, `${parsed.data.agent}.md`), md);
    agentsGenerated++;
  }

  return { success: true, target, outputDir, skillsGenerated, agentsGenerated, warnings };
}

export function printBuildResult(result: BuildResult): void {
  if (!result.success) {
    console.log(chalk.red(`Build failed: ${result.error}`));
    return;
  }

  console.log(chalk.green(`Build complete (target: ${result.target}):`));
  console.log(`  Output: ${result.outputDir}`);
  console.log(`  Skills generated: ${result.skillsGenerated}`);
  console.log(`  Agents generated: ${result.agentsGenerated}`);

  if (result.warnings.length > 0) {
    console.log(chalk.yellow('\nWarnings:'));
    for (const w of result.warnings) {
      console.log(`  ${chalk.yellow('!')} ${w}`);
    }
  }
}
