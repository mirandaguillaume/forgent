import { watch, existsSync, type FSWatcher } from 'fs';
import chalk from 'chalk';
import { runBuild, printBuildResult, type BuildTarget } from './build.js';
import { debounce } from '../utils/debounce.js';

export interface WatchOptions {
  skillsDir: string;
  agentsDir: string;
  outputDir: string;
  target: BuildTarget;
  debounceMs?: number;
}

export interface WatchController {
  stop(): void;
}

export function formatTimestamp(): string {
  const now = new Date();
  return `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}:${String(now.getSeconds()).padStart(2, '0')}`;
}

export function isRelevantFile(filename: string | null): boolean {
  if (!filename) return false;
  return filename.endsWith('.skill.yaml') || filename.endsWith('.agent.yaml');
}

export function createWatcher(options: WatchOptions): WatchController {
  const { skillsDir, agentsDir, outputDir, target, debounceMs = 300 } = options;
  const watchers: FSWatcher[] = [];

  const rebuild = () => {
    const ts = formatTimestamp();
    console.log(chalk.blue(`[${ts}] Rebuilding...`));
    const result = runBuild(skillsDir, agentsDir, outputDir, target);
    printBuildResult(result);
    if (result.success) {
      console.log(chalk.green(`[${formatTimestamp()}] Watching for changes...`));
    }
  };

  // Initial build
  rebuild();

  const debouncedRebuild = debounce(rebuild, debounceMs);

  // Watch skills directory
  if (existsSync(skillsDir)) {
    try {
      const w = watch(skillsDir, { recursive: true }, (_event, filename) => {
        if (isRelevantFile(filename as string | null)) debouncedRebuild();
      });
      watchers.push(w);
    } catch {
      console.log(chalk.yellow(`Warning: could not watch ${skillsDir}`));
    }
  }

  // Watch agents directory
  if (existsSync(agentsDir)) {
    try {
      const w = watch(agentsDir, { recursive: true }, (_event, filename) => {
        if (isRelevantFile(filename as string | null)) debouncedRebuild();
      });
      watchers.push(w);
    } catch {
      console.log(chalk.yellow(`Warning: could not watch ${agentsDir}`));
    }
  }

  return {
    stop() {
      debouncedRebuild.cancel();
      for (const w of watchers) w.close();
    },
  };
}
