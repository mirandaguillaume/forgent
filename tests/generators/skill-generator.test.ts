import { describe, it, expect } from 'vitest';
import { generateSkillMd, countWords } from '../../src/generators/skill-generator.js';
import type { SkillBehavior } from '../../src/model/skill-behavior.js';

const makeSkill = (overrides: Partial<SkillBehavior> = {}): SkillBehavior => ({
  skill: 'code-review',
  version: '1.0.0',
  context: { consumes: ['git_diff', 'test_results'], produces: ['review_comments', 'risk_score'], memory: 'conversation' },
  strategy: { tools: ['read_file', 'grep', 'search'], approach: 'diff-first', steps: ['analyze_diff', 'check_coverage', 'write_review'] },
  guardrails: ['no_approve_without_tests', { max_comments: 15 }, { timeout: '5min' }],
  depends_on: [{ skill: 'test-runner', provides: 'test_results' }],
  observability: { trace_level: 'detailed', metrics: ['tokens', 'latency'] },
  security: { filesystem: 'read-only', network: 'none', secrets: [] },
  negotiation: { file_conflicts: 'yield', priority: 2 },
  ...overrides,
});

describe('Skill generator', () => {
  it('should generate valid SKILL.md frontmatter', () => {
    const md = generateSkillMd(makeSkill());
    expect(md).toMatch(/^---\nname: code-review\n/);
    expect(md).toContain('description:');
    expect(md).toContain('---');
  });

  it('should include skill name as title', () => {
    const md = generateSkillMd(makeSkill());
    expect(md).toContain('# Code Review');
  });

  it('should put guardrails first in body (primacy bias)', () => {
    const md = generateSkillMd(makeSkill());
    const bodyStart = md.indexOf('# Code Review');
    const guardrailsPos = md.indexOf('## Guardrails', bodyStart);
    const contextPos = md.indexOf('## Context', bodyStart);
    const strategyPos = md.indexOf('## Strategy', bodyStart);
    expect(guardrailsPos).toBeLessThan(contextPos);
    expect(guardrailsPos).toBeLessThan(strategyPos);
  });

  it('should put security last in body (recency bias)', () => {
    const md = generateSkillMd(makeSkill());
    const securityPos = md.lastIndexOf('## Security');
    const strategyPos = md.lastIndexOf('## Strategy');
    const contextPos = md.lastIndexOf('## Context');
    expect(securityPos).toBeGreaterThan(strategyPos);
    expect(securityPos).toBeGreaterThan(contextPos);
  });

  it('should include context details', () => {
    const md = generateSkillMd(makeSkill());
    expect(md).toContain('git_diff');
    expect(md).toContain('test_results');
    expect(md).toContain('review_comments');
    expect(md).toContain('conversation');
  });

  it('should include strategy with steps', () => {
    const md = generateSkillMd(makeSkill());
    expect(md).toContain('diff-first');
    expect(md).toContain('read_file');
    expect(md).toContain('1. analyze_diff');
    expect(md).toContain('2. check_coverage');
  });

  it('should include guardrail rules', () => {
    const md = generateSkillMd(makeSkill());
    expect(md).toContain('no_approve_without_tests');
    expect(md).toContain('max_comments: 15');
    expect(md).toContain('timeout: 5min');
  });

  it('should include security constraints', () => {
    const md = generateSkillMd(makeSkill());
    expect(md).toContain('read-only');
    expect(md).toContain('none');
  });

  it('should include dependencies', () => {
    const md = generateSkillMd(makeSkill());
    expect(md).toContain('test-runner');
    expect(md).toContain('test_results');
  });

  it('should stay under 500 words for a typical skill', () => {
    const md = generateSkillMd(makeSkill());
    expect(countWords(md)).toBeLessThan(500);
  });

  it('should generate concise output for a minimal skill', () => {
    const minimal = makeSkill({
      skill: 'simple',
      context: { consumes: [], produces: [], memory: 'short-term' },
      strategy: { tools: [], approach: 'sequential' },
      guardrails: [],
      depends_on: [],
      observability: { trace_level: 'minimal', metrics: [] },
      security: { filesystem: 'none', network: 'none', secrets: [] },
    });
    const md = generateSkillMd(minimal);
    expect(countWords(md)).toBeLessThan(150);
  });

  it('should format guardrails as objects correctly', () => {
    const skill = makeSkill({
      guardrails: [{ timeout: '10min' }, { max_retries: 3 }],
    });
    const md = generateSkillMd(skill);
    expect(md).toContain('timeout: 10min');
    expect(md).toContain('max_retries: 3');
  });
});
