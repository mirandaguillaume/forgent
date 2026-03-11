import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtempSync, rmSync, mkdirSync, writeFileSync, existsSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';
import { createWatcher, isRelevantFile, formatTimestamp } from '../../src/commands/watch.js';

const goodSkill = `
skill: test-skill
version: "1.0.0"
context:
  consumes: [input]
  produces: [output]
  memory: short-term
strategy:
  tools: [read_file]
  approach: sequential
guardrails:
  - timeout: 5min
depends_on: []
observability:
  trace_level: minimal
  metrics: [tokens]
security:
  filesystem: read-only
  network: none
  secrets: []
negotiation:
  file_conflicts: yield
  priority: 1
`;

describe('Watch mode', () => {
  let tempDir: string;

  beforeEach(() => {
    tempDir = mkdtempSync(join(tmpdir(), 'forgent-watch-'));
    mkdirSync(join(tempDir, 'skills'));
    mkdirSync(join(tempDir, 'agents'));
  });

  afterEach(() => {
    rmSync(tempDir, { recursive: true, force: true });
  });

  it('should perform initial build on start', () => {
    writeFileSync(join(tempDir, 'skills', 'test.skill.yaml'), goodSkill);
    const controller = createWatcher({
      skillsDir: join(tempDir, 'skills'),
      agentsDir: join(tempDir, 'agents'),
      outputDir: join(tempDir, '.claude'),
      target: 'claude',
    });
    expect(existsSync(join(tempDir, '.claude', 'skills', 'test-skill', 'SKILL.md'))).toBe(true);
    controller.stop();
  });

  it('should return a controller with stop()', () => {
    const controller = createWatcher({
      skillsDir: join(tempDir, 'skills'),
      agentsDir: join(tempDir, 'agents'),
      outputDir: join(tempDir, '.claude'),
      target: 'claude',
    });
    expect(controller.stop).toBeDefined();
    expect(typeof controller.stop).toBe('function');
    controller.stop();
  });

  it('should not crash when watching non-existent directory', () => {
    expect(() => {
      const controller = createWatcher({
        skillsDir: '/nonexistent/path/skills',
        agentsDir: '/nonexistent/path/agents',
        outputDir: join(tempDir, '.claude'),
        target: 'claude',
      });
      controller.stop();
    }).not.toThrow();
  });
});

describe('isRelevantFile', () => {
  it('should match .skill.yaml files', () => {
    expect(isRelevantFile('test.skill.yaml')).toBe(true);
  });

  it('should match .agent.yaml files', () => {
    expect(isRelevantFile('reviewer.agent.yaml')).toBe(true);
  });

  it('should reject non-YAML files', () => {
    expect(isRelevantFile('test.ts')).toBe(false);
    expect(isRelevantFile('README.md')).toBe(false);
    expect(isRelevantFile('config.yaml')).toBe(false);
  });

  it('should handle null filename', () => {
    expect(isRelevantFile(null)).toBe(false);
  });
});

describe('formatTimestamp', () => {
  it('should return HH:MM:SS format', () => {
    const ts = formatTimestamp();
    expect(ts).toMatch(/^\d{2}:\d{2}:\d{2}$/);
  });
});
