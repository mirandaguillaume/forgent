import { readdirSync, readFileSync } from 'fs';
import { join } from 'path';
import chalk from 'chalk';
import { parseSkillYaml } from '../utils/yaml-loader.js';
import { lintSkill, type LintResult } from '../linters/rules.js';

export interface LintCommandResult {
  totalFiles: number;
  totalIssues: number;
  errors: number;
  warnings: number;
  results: Map<string, LintResult[]>;
}

export function lintDirectory(skillsDir: string): LintCommandResult {
  const files = readdirSync(skillsDir).filter((f) => f.endsWith('.skill.yaml'));
  const results = new Map<string, LintResult[]>();
  let totalIssues = 0;
  let errors = 0;
  let warnings = 0;

  for (const file of files) {
    const content = readFileSync(join(skillsDir, file), 'utf-8');
    const parsed = parseSkillYaml(content);

    if (!parsed.success) {
      results.set(file, [{
        rule: 'valid-schema',
        severity: 'error',
        message: `Invalid skill file: ${parsed.error}`,
        facet: 'schema',
      }]);
      errors++;
      totalIssues++;
      continue;
    }

    const lintResults = lintSkill(parsed.data);
    if (lintResults.length > 0) {
      results.set(file, lintResults);
      for (const r of lintResults) {
        totalIssues++;
        if (r.severity === 'error') errors++;
        if (r.severity === 'warning') warnings++;
      }
    }
  }

  return { totalFiles: files.length, totalIssues, errors, warnings, results };
}

export function printLintResults(result: LintCommandResult): void {
  if (result.totalFiles === 0) {
    console.log(chalk.yellow('No skill files found.'));
    return;
  }

  for (const [file, issues] of result.results) {
    console.log(chalk.bold(`\n${file}`));
    for (const issue of issues) {
      const icon = issue.severity === 'error' ? chalk.red('x') : chalk.yellow('!');
      console.log(`  ${icon} [${issue.facet}] ${issue.message}`);
    }
  }

  console.log(`\nScanned ${result.totalFiles} skills: ${result.errors} errors, ${result.warnings} warnings`);

  if (result.errors > 0) {
    console.log(chalk.red('\nLint failed.'));
  } else if (result.warnings > 0) {
    console.log(chalk.yellow('\nLint passed with warnings.'));
  } else {
    console.log(chalk.green('\nLint passed.'));
  }
}
