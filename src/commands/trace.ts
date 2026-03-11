import { readFileSync, existsSync } from 'fs';
import chalk from 'chalk';
import { parseTrace, summarizeTrace } from '../analyzers/trace-parser.js';

export function traceFile(tracePath: string): void {
  if (!existsSync(tracePath)) {
    console.log(chalk.red(`Trace file not found: ${tracePath}`));
    process.exit(1);
  }
  const content = readFileSync(tracePath, 'utf-8');
  const events = parseTrace(content);
  const summary = summarizeTrace(events);

  console.log(chalk.bold('\n=== Forgent Trace Summary ===\n'));
  console.log(`Total duration: ${summary.totalDuration_ms}ms`);
  console.log(`Total tokens: ${summary.totalTokens}`);
  console.log(`Tool calls: ${summary.toolCalls}`);
  console.log(`Decisions: ${summary.decisions}`);

  if (summary.toolFrequency.size > 0) {
    console.log(chalk.bold('\nTool usage:'));
    for (const [tool, count] of summary.toolFrequency) {
      console.log(`  ${tool}: ${count}x`);
    }
  }

  if (summary.warnings.length > 0) {
    console.log(chalk.yellow('\nWarnings:'));
    for (const w of summary.warnings) {
      console.log(`  ${chalk.yellow('!')} ${w}`);
    }
  }

  console.log('');
}
