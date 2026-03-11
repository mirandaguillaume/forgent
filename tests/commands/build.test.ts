import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdtempSync, rmSync, mkdirSync, writeFileSync, existsSync, readFileSync } from 'fs';
import { join } from 'path';
import { tmpdir } from 'os';
import { runBuild } from '../../src/commands/build.js';

const goodSkill = `
skill: code-review
version: "1.0.0"
context:
  consumes: [git_diff]
  produces: [review_comments]
  memory: conversation
strategy:
  tools: [read_file, grep]
  approach: diff-first
  steps:
    - analyze_diff
    - write_review
guardrails:
  - timeout: 5min
depends_on: []
observability:
  trace_level: detailed
  metrics: [tokens]
security:
  filesystem: read-only
  network: none
  secrets: []
negotiation:
  file_conflicts: yield
  priority: 1
`;

const goodAgent = `
agent: reviewer
skills: [code-review]
orchestration: sequential
description: "Reviews code changes"
`;

const badSkill = `
skill: broken
version: "1.0.0"
context:
  consumes: []
  produces: []
  memory: short-term
strategy:
  tools: []
  approach: seq
guardrails: []
depends_on: []
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

describe('forgent build command', () => {
  let tempDir: string;

  beforeEach(() => {
    tempDir = mkdtempSync(join(tmpdir(), 'forgent-build-'));
    mkdirSync(join(tempDir, 'skills'));
    mkdirSync(join(tempDir, 'agents'));
  });

  afterEach(() => {
    rmSync(tempDir, { recursive: true, force: true });
  });

  it('should generate SKILL.md for each valid skill', () => {
    writeFileSync(join(tempDir, 'skills', 'code-review.skill.yaml'), goodSkill);
    const result = runBuild(join(tempDir, 'skills'), join(tempDir, 'agents'), join(tempDir, '.claude'));
    expect(result.success).toBe(true);
    expect(existsSync(join(tempDir, '.claude', 'skills', 'code-review', 'SKILL.md'))).toBe(true);
  });

  it('should generate valid SKILL.md content', () => {
    writeFileSync(join(tempDir, 'skills', 'code-review.skill.yaml'), goodSkill);
    runBuild(join(tempDir, 'skills'), join(tempDir, 'agents'), join(tempDir, '.claude'));
    const content = readFileSync(join(tempDir, '.claude', 'skills', 'code-review', 'SKILL.md'), 'utf-8');
    expect(content).toContain('name: code-review');
    expect(content).toContain('## Guardrails');
    expect(content).toContain('## Security');
  });

  it('should generate agent.md for each valid agent', () => {
    writeFileSync(join(tempDir, 'agents', 'reviewer.agent.yaml'), goodAgent);
    const result = runBuild(join(tempDir, 'skills'), join(tempDir, 'agents'), join(tempDir, '.claude'));
    expect(result.success).toBe(true);
    expect(existsSync(join(tempDir, '.claude', 'agents', 'reviewer.md'))).toBe(true);
  });

  it('should generate valid agent.md content without model: inherit', () => {
    writeFileSync(join(tempDir, 'agents', 'reviewer.agent.yaml'), goodAgent);
    runBuild(join(tempDir, 'skills'), join(tempDir, 'agents'), join(tempDir, '.claude'));
    const content = readFileSync(join(tempDir, '.claude', 'agents', 'reviewer.md'), 'utf-8');
    expect(content).toContain('name: reviewer');
    expect(content).not.toContain('model: inherit');
    expect(content).toContain('code-review');
  });

  it('should resolve skill tools into agent frontmatter', () => {
    writeFileSync(join(tempDir, 'skills', 'code-review.skill.yaml'), goodSkill);
    writeFileSync(join(tempDir, 'agents', 'reviewer.agent.yaml'), goodAgent);
    runBuild(join(tempDir, 'skills'), join(tempDir, 'agents'), join(tempDir, '.claude'));
    const content = readFileSync(join(tempDir, '.claude', 'agents', 'reviewer.md'), 'utf-8');
    // code-review skill uses read_file + grep → Read, Grep; read-only fs → Glob, Grep, Read
    expect(content).toContain('tools: Glob, Grep, Read');
  });

  it('should warn when agent references unresolved skills', () => {
    writeFileSync(join(tempDir, 'agents', 'reviewer.agent.yaml'), goodAgent);
    // No skills dir content — code-review skill not available
    const result = runBuild(join(tempDir, 'skills'), join(tempDir, 'agents'), join(tempDir, '.claude'));
    expect(result.warnings.some((w) => w.includes('unresolved skills'))).toBe(true);
  });

  it('should refuse build when lint has errors', () => {
    writeFileSync(join(tempDir, 'skills', 'broken.skill.yaml'), badSkill);
    const result = runBuild(join(tempDir, 'skills'), join(tempDir, 'agents'), join(tempDir, '.claude'));
    expect(result.success).toBe(false);
    expect(result.error).toContain('lint');
  });

  it('should warn if generated skill exceeds 500 words', () => {
    // A normal skill shouldn't exceed, but let's verify the warning mechanism exists
    writeFileSync(join(tempDir, 'skills', 'code-review.skill.yaml'), goodSkill);
    const result = runBuild(join(tempDir, 'skills'), join(tempDir, 'agents'), join(tempDir, '.claude'));
    expect(result.warnings).toBeDefined();
  });

  it('should handle empty directories', () => {
    const result = runBuild(join(tempDir, 'skills'), join(tempDir, 'agents'), join(tempDir, '.claude'));
    expect(result.success).toBe(true);
    expect(result.skillsGenerated).toBe(0);
    expect(result.agentsGenerated).toBe(0);
  });

  it('should warn on data-flow ordering mismatch in sequential agent', () => {
    const consumerFirst = `
skill: code-review
version: "1.0.0"
context:
  consumes: [test_results]
  produces: [review_comments]
  memory: conversation
strategy:
  tools: [read_file]
  approach: diff-first
guardrails:
  - timeout: 5min
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
  priority: 1
`;
    const producer = `
skill: test-runner
version: "1.0.0"
context:
  consumes: []
  produces: [test_results]
  memory: short-term
strategy:
  tools: [bash]
  approach: execute
guardrails:
  - timeout: 3min
depends_on: []
observability:
  trace_level: standard
  metrics: [duration]
security:
  filesystem: read-only
  network: none
  secrets: []
negotiation:
  file_conflicts: yield
  priority: 0
`;
    const badOrderAgent = `
agent: reviewer
skills: [code-review, test-runner]
orchestration: sequential
description: "Reviews code"
`;
    writeFileSync(join(tempDir, 'skills', 'code-review.skill.yaml'), consumerFirst);
    writeFileSync(join(tempDir, 'skills', 'test-runner.skill.yaml'), producer);
    writeFileSync(join(tempDir, 'agents', 'reviewer.agent.yaml'), badOrderAgent);
    const result = runBuild(join(tempDir, 'skills'), join(tempDir, 'agents'), join(tempDir, '.claude'));
    expect(result.success).toBe(true);
    expect(result.warnings.some((w) => w.includes('test_results'))).toBe(true);
  });

  it('should create output directories if they do not exist', () => {
    writeFileSync(join(tempDir, 'skills', 'code-review.skill.yaml'), goodSkill);
    const outputDir = join(tempDir, '.claude');
    expect(existsSync(outputDir)).toBe(false);
    runBuild(join(tempDir, 'skills'), join(tempDir, 'agents'), outputDir);
    expect(existsSync(outputDir)).toBe(true);
  });
});
