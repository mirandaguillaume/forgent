#!/usr/bin/env node
import { Command } from 'commander';
import chalk from 'chalk';
import { initProject } from './commands/init.js';

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

program.parse();
