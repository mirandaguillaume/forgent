import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtempSync, rmSync, existsSync, readFileSync, writeFileSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';
import { initProject } from '../../src/commands/init.js';

describe('ax init', () => {
  let tempDir: string;

  beforeEach(() => {
    tempDir = mkdtempSync(join(tmpdir(), 'ax-test-'));
  });

  afterEach(() => {
    rmSync(tempDir, { recursive: true, force: true });
  });

  it('should create ax.yaml config file', () => {
    initProject(tempDir);
    expect(existsSync(join(tempDir, 'ax.yaml'))).toBe(true);
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

  it('should write valid ax.yaml config', () => {
    initProject(tempDir);
    const content = readFileSync(join(tempDir, 'ax.yaml'), 'utf-8');
    expect(content).toContain('version:');
    expect(content).toContain('skills_dir: skills');
    expect(content).toContain('agents_dir: agents');
  });

  it('should not overwrite existing ax.yaml', () => {
    writeFileSync(join(tempDir, 'ax.yaml'), 'existing: true');
    const result = initProject(tempDir);
    expect(result.alreadyInitialized).toBe(true);
    const content = readFileSync(join(tempDir, 'ax.yaml'), 'utf-8');
    expect(content).toBe('existing: true');
  });

  it('should return path in result', () => {
    const result = initProject(tempDir);
    expect(result.path).toBe(tempDir);
    expect(result.alreadyInitialized).toBe(false);
  });
});
