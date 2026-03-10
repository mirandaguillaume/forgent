#!/usr/bin/env node
import { Command } from 'commander';

const program = new Command();

program
  .name('ax')
  .description('AX — Agent Experience CLI')
  .version('0.1.0');

program.parse();
