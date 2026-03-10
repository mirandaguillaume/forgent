#!/usr/bin/env node
import { Command } from 'commander';
import chalk from 'chalk';
import { initProject } from './commands/init.js';
import { createSkill } from './commands/skill-create.js';
import { lintDirectory, printLintResults } from './commands/lint.js';
import { runDoctor, printDoctorReport } from './commands/doctor.js';
import { traceFile } from './commands/trace.js';

const program = new Command();

program
  .name('ax')
  .description('AX — Agent Experience CLI')
  .version('0.1.0');

program
  .command('init')
  .description('Initialize an AX project in the current directory')
  .argument('[path]', 'target directory', '.')
  .action((path: string) => {
    const result = initProject(path);
    if (result.alreadyInitialized) {
      console.log(chalk.yellow('AX project already initialized.'));
    } else {
      console.log(chalk.green('AX project initialized at'), result.path);
      console.log('  Created: ax.yaml, skills/, agents/');
    }
  });

const skill = program.command('skill').description('Manage skills');

skill
  .command('create')
  .description('Create a new skill')
  .argument('<name>', 'skill name')
  .option('-t, --tools <tools...>', 'tools the skill can use')
  .option('-m, --memory <type>', 'memory type', 'short-term')
  .action((name: string, opts: { tools?: string[]; memory?: string }) => {
    const result = createSkill('.', name, opts);
    if (result.success) {
      console.log(chalk.green(`Skill "${name}" created at ${result.path}`));
    } else {
      console.log(chalk.red(result.error));
    }
  });

program
  .command('lint')
  .description('Lint skill files for AX best practices')
  .argument('[path]', 'skills directory', 'skills')
  .action((path: string) => {
    const result = lintDirectory(path);
    printLintResults(result);
    process.exit(result.errors > 0 ? 1 : 0);
  });

program
  .command('doctor')
  .description('Run full diagnostic on all skills')
  .argument('[path]', 'skills directory', 'skills')
  .action((path: string) => {
    const report = runDoctor(path);
    printDoctorReport(report);
    process.exit(report.score < 50 ? 1 : 0);
  });

program
  .command('trace')
  .description('Analyze a JSONL trace file')
  .argument('<file>', 'trace file path (JSONL format)')
  .action((file: string) => {
    traceFile(file);
  });

program.parse();
