import { readdirSync, readFileSync } from 'fs';
import { join } from 'path';
import chalk from 'chalk';
import { parseSkillYaml } from '../utils/yaml-loader.js';
import { lintSkill, type LintResult } from '../linters/rules.js';
import { checkDependencies, type DependencyIssue } from '../analyzers/dependency-checker.js';
import { detectLoopRisks, type LoopRisk } from '../analyzers/loop-detector.js';
import type { SkillBehavior } from '../model/skill-behavior.js';

export interface DoctorReport {
  skills: SkillBehavior[];
  parseErrors: Array<{ file: string; error: string }>;
  lintIssues: Map<string, LintResult[]>;
  dependencyIssues: DependencyIssue[];
  loopRisks: Map<string, LoopRisk[]>;
  score: number;
}

export function runDoctor(skillsDir: string): DoctorReport {
  const files = readdirSync(skillsDir).filter((f) => f.endsWith('.skill.yaml'));
  const skills: SkillBehavior[] = [];
  const parseErrors: DoctorReport['parseErrors'] = [];
  const lintIssues = new Map<string, LintResult[]>();
  const loopRisks = new Map<string, LoopRisk[]>();

  // Parse all skills
  for (const file of files) {
    const content = readFileSync(join(skillsDir, file), 'utf-8');
    const parsed = parseSkillYaml(content);
    if (parsed.success) {
      skills.push(parsed.data);
    } else {
      parseErrors.push({ file, error: parsed.error });
    }
  }

  // Lint each skill
  for (const skill of skills) {
    const issues = lintSkill(skill);
    if (issues.length > 0) lintIssues.set(skill.skill, issues);
  }

  // Check dependencies across all skills
  const dependencyIssues = checkDependencies(skills);

  // Check loop risks per skill
  for (const skill of skills) {
    const risks = detectLoopRisks(skill);
    if (risks.length > 0) loopRisks.set(skill.skill, risks);
  }

  // Calculate health score
  const totalIssues =
    parseErrors.length +
    [...lintIssues.values()].flat().filter((i) => i.severity === 'error').length +
    dependencyIssues.length +
    [...loopRisks.values()].flat().filter((r) => r.severity === 'error').length;

  const maxScore = Math.max(files.length * 10, 100);
  const score = Math.max(0, Math.round(100 - (totalIssues / maxScore) * 100));

  return { skills, parseErrors, lintIssues, dependencyIssues, loopRisks, score };
}

export function printDoctorReport(report: DoctorReport): void {
  console.log(chalk.bold('\n=== AX Doctor Report ===\n'));
  console.log(`Skills found: ${report.skills.length}`);

  if (report.parseErrors.length > 0) {
    console.log(chalk.red(`\nParse Errors (${report.parseErrors.length}):`));
    for (const err of report.parseErrors) {
      console.log(`  ${chalk.red('x')} ${err.file}: ${err.error}`);
    }
  }

  if (report.lintIssues.size > 0) {
    console.log(chalk.yellow(`\nLint Issues:`));
    for (const [skill, issues] of report.lintIssues) {
      for (const issue of issues) {
        const icon = issue.severity === 'error' ? chalk.red('x') : chalk.yellow('!');
        console.log(`  ${icon} ${skill}: ${issue.message}`);
      }
    }
  }

  if (report.dependencyIssues.length > 0) {
    console.log(chalk.red(`\nDependency Issues (${report.dependencyIssues.length}):`));
    for (const issue of report.dependencyIssues) {
      console.log(`  ${chalk.red('x')} ${issue.skill}: ${issue.message}`);
    }
  }

  if (report.loopRisks.size > 0) {
    console.log(chalk.yellow(`\nLoop Risks:`));
    for (const [skill, risks] of report.loopRisks) {
      for (const risk of risks) {
        const icon = risk.severity === 'error' ? chalk.red('x') : chalk.yellow('!');
        console.log(`  ${icon} ${skill}: ${risk.message}`);
      }
    }
  }

  const scoreColor = report.score >= 80 ? chalk.green : report.score >= 50 ? chalk.yellow : chalk.red;
  console.log(`\nHealth Score: ${scoreColor(`${report.score}/100`)}\n`);
}
