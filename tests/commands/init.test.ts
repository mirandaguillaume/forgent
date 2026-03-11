import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtempSync, rmSync, existsSync, readFileSync, writeFileSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';
import { initProject } from '../../src/commands/init.js';

describe('forgent init', () => {
  let tempDir: string;

  beforeEach(() => {
    tempDir = mkdtempSync(join(tmpdir(), 'forgent-test-'));
  });

  afterEach(() => {
    rmSync(tempDir, { recursive: true, force: true });
  });

  it('should create forgent.yaml config file', () => {
    initProject(tempDir);
    expect(existsSync(join(tempDir, 'forgent.yaml'))).toBe(true);
  });

  it('should create skills/ directory', () => {
    initProject(tempDir);
    expect(existsSync(join(tempDir, 'skills'))).toBe(true);
  });

  it('should create agents/ directory', () => {
    initProject(tempDir);
    expect(existsSync(join(tempDir, 'agents'))).toBe(true);
  });

  it('should create example skill file', () => {
    initProject(tempDir);
    const examplePath = join(tempDir, 'skills', 'example.skill.yaml');
    expect(existsSync(examplePath)).toBe(true);
    const content = readFileSync(examplePath, 'utf-8');
    expect(content).toContain('skill:');
  });

  it('should write valid forgent.yaml config', () => {
    initProject(tempDir);
    const content = readFileSync(join(tempDir, 'forgent.yaml'), 'utf-8');
    expect(content).toContain('version:');
    expect(content).toContain('skills_dir: skills');
    expect(content).toContain('agents_dir: agents');
  });

  it('should not overwrite existing forgent.yaml', () => {
    writeFileSync(join(tempDir, 'forgent.yaml'), 'existing: true');
    const result = initProject(tempDir);
    expect(result.alreadyInitialized).toBe(true);
    const content = readFileSync(join(tempDir, 'forgent.yaml'), 'utf-8');
    expect(content).toBe('existing: true');
  });

  it('should return path in result', () => {
    const result = initProject(tempDir);
    expect(result.path).toBe(tempDir);
    expect(result.alreadyInitialized).toBe(false);
  });
});
