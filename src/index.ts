#!/usr/bin/env node
import { Command } from 'commander';
import chalk from 'chalk';
import { initProject } from './commands/init.js';
import { createSkill } from './commands/skill-create.js';
import { lintDirectory, printLintResults } from './commands/lint.js';
import { runDoctor, printDoctorReport } from './commands/doctor.js';
import { traceFile } from './commands/trace.js';
import { runBuild, printBuildResult, getOutputDir } from './commands/build.js';
import { getAvailableTargets, type BuildTarget } from './generators/target-generator.js';
import { createWatcher } from './commands/watch.js';
import { runScore, printScoreReport } from './commands/score.js';

const program = new Command();

program
  .name('forgent')
  .description('Forgent — Forge agents from composable skill specs')
  .version('0.1.0');

program
  .command('init')
  .description('Initialize a Forgent project in the current directory')
  .argument('[path]', 'target directory', '.')
  .action((path: string) => {
    const result = initProject(path);
    if (result.alreadyInitialized) {
      console.log(chalk.yellow('Forgent project already initialized.'));
    } else {
      console.log(chalk.green('Forgent project initialized at'), result.path);
      console.log('  Created: forgent.yaml, skills/, agents/');
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
  .description('Lint skill files for best practices')
  .argument('[path]', 'skills directory', 'skills')
  .action((path: string) => {
    const result = lintDirectory(path);
    printLintResults(result);
    process.exit(result.errors > 0 ? 1 : 0);
  });

program
  .command('doctor')
  .description('Run full diagnostic on all skills and agents')
  .argument('[path]', 'skills directory', 'skills')
  .option('-a, --agents <dir>', 'agents directory', 'agents')
  .action((path: string, opts: { agents: string }) => {
    const report = runDoctor(path, opts.agents);
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

program
  .command('build')
  .description('Generate skills and agents for a target framework')
  .option('-t, --target <target>', 'target framework', 'claude')
  .option('-s, --skills <dir>', 'skills directory', 'skills')
  .option('-a, --agents <dir>', 'agents directory', 'agents')
  .option('-o, --output <dir>', 'output directory (overrides target default)')
  .option('-w, --watch', 'watch for changes and rebuild automatically')
  .action((opts: { target: string; skills: string; agents: string; output?: string; watch?: boolean }) => {
    const target = opts.target as BuildTarget;
    const available = getAvailableTargets();
    if (!available.includes(target)) {
      console.log(chalk.red(`Unknown target "${opts.target}". Available: ${available.join(', ')}`));
      process.exit(1);
    }
    const outputDir = getOutputDir(target, opts.output);

    if (opts.watch) {
      const controller = createWatcher({
        skillsDir: opts.skills,
        agentsDir: opts.agents,
        outputDir,
        target,
      });
      process.on('SIGINT', () => {
        controller.stop();
        console.log('\nStopped watching.');
        process.exit(0);
      });
      return;
    }

    const result = runBuild(opts.skills, opts.agents, outputDir, target);
    printBuildResult(result);
    process.exit(result.success ? 0 : 1);
  });

program
  .command('score')
  .description('Score design quality of skills and agents')
  .argument('[path]', 'skills directory', 'skills')
  .option('-a, --agents <dir>', 'agents directory', 'agents')
  .action((path: string, opts: { agents: string }) => {
    const report = runScore(path, opts.agents);
    printScoreReport(report);
  });

program.parse();
