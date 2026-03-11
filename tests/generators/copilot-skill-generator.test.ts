import { describe, it, expect } from 'vitest';
import { generateCopilotSkillMd } from '../../src/generators/copilot-skill-generator.js';
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

describe('Copilot skill generator', () => {
  it('should generate frontmatter with name and description', () => {
    const md = generateCopilotSkillMd(makeSkill());
    expect(md).toMatch(/^---\nname: code-review\n/);
    expect(md).toContain('description:');
    expect(md).toContain('---');
  });

  it('should include H1 title', () => {
    const md = generateCopilotSkillMd(makeSkill());
    expect(md).toContain('# Code Review');
  });

  it('should put guardrails first in body (primacy bias)', () => {
    const md = generateCopilotSkillMd(makeSkill());
    const bodyStart = md.indexOf('# Code Review');
    const guardrailsPos = md.indexOf('## Guardrails', bodyStart);
    const contextPos = md.indexOf('## Context', bodyStart);
    const strategyPos = md.indexOf('## Strategy', bodyStart);
    expect(guardrailsPos).toBeLessThan(contextPos);
    expect(guardrailsPos).toBeLessThan(strategyPos);
  });

  it('should put security last in body (recency bias)', () => {
    const md = generateCopilotSkillMd(makeSkill());
    const securityPos = md.lastIndexOf('## Security');
    const strategyPos = md.lastIndexOf('## Strategy');
    const contextPos = md.lastIndexOf('## Context');
    expect(securityPos).toBeGreaterThan(strategyPos);
    expect(securityPos).toBeGreaterThan(contextPos);
  });

  it('should include strategy steps as numbered list', () => {
    const md = generateCopilotSkillMd(makeSkill());
    expect(md).toContain('1. analyze_diff');
    expect(md).toContain('2. check_coverage');
    expect(md).toContain('3. write_review');
  });

  it('should include dependencies', () => {
    const md = generateCopilotSkillMd(makeSkill());
    expect(md).toContain('test-runner');
    expect(md).toContain('test_results');
  });

  it('should truncate description to 1024 chars max', () => {
    const longConsumes = Array.from({ length: 200 }, (_, i) => `very_long_context_field_name_${i}`);
    const skill = makeSkill({ context: { consumes: longConsumes, produces: ['output'], memory: 'conversation' } });
    const md = generateCopilotSkillMd(skill);
    // Extract description from frontmatter
    const descMatch = md.match(/description: (.+)/);
    expect(descMatch).not.toBeNull();
    expect(descMatch![1].length).toBeLessThanOrEqual(1024);
  });

  it('should skip guardrails section when empty', () => {
    const md = generateCopilotSkillMd(makeSkill({ guardrails: [] }));
    expect(md).not.toContain('## Guardrails');
  });

  it('should skip dependencies section when empty', () => {
    const md = generateCopilotSkillMd(makeSkill({ depends_on: [] }));
    expect(md).not.toContain('## Dependencies');
  });
});
