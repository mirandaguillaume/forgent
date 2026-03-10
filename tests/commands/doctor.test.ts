import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtempSync, rmSync, mkdirSync, writeFileSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';
import { runDoctor } from '../../src/commands/doctor.js';

const goodSkill = `
skill: good
version: "1.0.0"
context:
  consumes: []
  produces: []
  memory: short-term
strategy:
  tools: [read_file]
  approach: sequential
guardrails:
  - max_tokens: 1000
depends_on: []
observability:
  trace_level: standard
  metrics: [tokens]
security:
  filesystem: read-only
  network: none
  secrets: []
negotiation:
  file_conflicts: yield
  priority: 0
`;

const dangerousSkill = `
skill: dangerous
version: "1.0.0"
context:
  consumes: [data]
  produces: [data]
  memory: long-term
strategy:
  tools: []
  approach: sequential
guardrails: []
depends_on:
  - skill: nonexistent
    provides: something
observability:
  trace_level: minimal
  metrics: []
security:
  filesystem: full
  network: none
  secrets: []
negotiation:
  file_conflicts: yield
  priority: 0
`;

describe('ax doctor command', () => {
  let tempDir: string;

  beforeEach(() => {
    tempDir = mkdtempSync(join(tmpdir(), 'ax-doctor-'));
    mkdirSync(join(tempDir, 'skills'));
  });

  afterEach(() => {
    rmSync(tempDir, { recursive: true, force: true });
  });

  it('should report high score for well-configured skills', () => {
    writeFileSync(join(tempDir, 'skills', 'good.skill.yaml'), goodSkill);
    const report = runDoctor(join(tempDir, 'skills'));
    expect(report.skills).toHaveLength(1);
    expect(report.parseErrors).toHaveLength(0);
    expect(report.dependencyIssues).toHaveLength(0);
    expect(report.score).toBeGreaterThanOrEqual(80);
  });

  it('should detect all issues in a dangerous skill', () => {
    writeFileSync(join(tempDir, 'skills', 'dangerous.skill.yaml'), dangerousSkill);
    const report = runDoctor(join(tempDir, 'skills'));
    expect(report.skills).toHaveLength(1);
    expect(report.dependencyIssues.length).toBeGreaterThan(0);
    expect(report.loopRisks.size).toBeGreaterThan(0);
    expect(report.lintIssues.size).toBeGreaterThan(0);
  });

  it('should handle parse errors gracefully', () => {
    writeFileSync(join(tempDir, 'skills', 'broken.skill.yaml'), 'invalid: yaml: [[[');
    const report = runDoctor(join(tempDir, 'skills'));
    expect(report.parseErrors).toHaveLength(1);
    expect(report.skills).toHaveLength(0);
  });

  it('should aggregate issues from multiple skills', () => {
    writeFileSync(join(tempDir, 'skills', 'good.skill.yaml'), goodSkill);
    writeFileSync(join(tempDir, 'skills', 'dangerous.skill.yaml'), dangerousSkill);
    const report = runDoctor(join(tempDir, 'skills'));
    expect(report.skills).toHaveLength(2);
  });

  it('should return score of 100 for empty directory', () => {
    const report = runDoctor(join(tempDir, 'skills'));
    expect(report.skills).toHaveLength(0);
    expect(report.score).toBe(100);
  });
});
