#!/usr/bin/env node
import { Command } from 'commander';
import chalk from 'chalk';
import { initProject } from './commands/init.js';
import { createSkill } from './commands/skill-create.js';

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

program.parse();
