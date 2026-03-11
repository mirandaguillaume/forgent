import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtempSync, rmSync, existsSync, readFileSync, mkdirSync, writeFileSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';
import { createSkill } from '../../src/commands/skill-create.js';
import { parseSkillYaml } from '../../src/utils/yaml-loader.js';

describe('forgent skill create', () => {
  let tempDir: string;

  beforeEach(() => {
    tempDir = mkdtempSync(join(tmpdir(), 'forgent-test-'));
    mkdirSync(join(tempDir, 'skills'));
    writeFileSync(join(tempDir, 'forgent.yaml'), 'version: "0.1.0"\nskills_dir: skills');
  });

  afterEach(() => {
    rmSync(tempDir, { recursive: true, force: true });
  });

  it('should create a skill YAML file', () => {
    const result = createSkill(tempDir, 'code-review');
    expect(result.success).toBe(true);
    expect(existsSync(join(tempDir, 'skills', 'code-review.skill.yaml'))).toBe(true);
  });

  it('should populate skill name in the file', () => {
    createSkill(tempDir, 'code-review');
    const content = readFileSync(join(tempDir, 'skills', 'code-review.skill.yaml'), 'utf-8');
    expect(content).toContain('skill: code-review');
  });

  it('should not overwrite existing skill', () => {
    createSkill(tempDir, 'code-review');
    const result = createSkill(tempDir, 'code-review');
    expect(result.success).toBe(false);
    expect(result.error).toContain('already exists');
  });

  it('should accept optional tools', () => {
    createSkill(tempDir, 'searcher', { tools: ['grep', 'find'] });
    const content = readFileSync(join(tempDir, 'skills', 'searcher.skill.yaml'), 'utf-8');
    expect(content).toContain('grep');
    expect(content).toContain('find');
  });

  it('should accept optional memory type', () => {
    createSkill(tempDir, 'persistent', { memory: 'long-term' });
    const content = readFileSync(join(tempDir, 'skills', 'persistent.skill.yaml'), 'utf-8');
    expect(content).toContain('long-term');
  });

  it('should produce a valid skill YAML that passes schema validation', () => {
    createSkill(tempDir, 'validated', { tools: ['read_file'] });
    const content = readFileSync(join(tempDir, 'skills', 'validated.skill.yaml'), 'utf-8');
    const parsed = parseSkillYaml(content);
    expect(parsed.success).toBe(true);
    if (parsed.success) {
      expect(parsed.data.skill).toBe('validated');
      expect(parsed.data.strategy.tools).toContain('read_file');
    }
  });

  it('should return the created file path', () => {
    const result = createSkill(tempDir, 'my-skill');
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.path).toContain('my-skill.skill.yaml');
    }
  });
});
