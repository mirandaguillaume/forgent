import { describe, it, expect } from 'vitest';
import { lintSkill, type LintResult, type LintSeverity } from '../../src/linters/rules.js';
import type { SkillBehavior } from '../../src/model/skill-behavior.js';

const makeSkill = (overrides: Partial<SkillBehavior> = {}): SkillBehavior => ({
  skill: 'test-skill',
  version: '1.0.0',
  context: { consumes: [], produces: [], memory: 'short-term' },
  strategy: { tools: [], approach: 'sequential' },
  guardrails: [],
  depends_on: [],
  observability: { trace_level: 'minimal', metrics: [] },
  security: { filesystem: 'none', network: 'none', secrets: [] },
  negotiation: { file_conflicts: 'yield', priority: 0 },
  ...overrides,
});

describe('Linter rules', () => {
  it('should warn when skill has no tools', () => {
    const results = lintSkill(makeSkill());
    expect(results.some((r) => r.rule === 'no-empty-tools')).toBe(true);
  });

  it('should warn when skill has no guardrails', () => {
    const results = lintSkill(makeSkill());
    expect(results.some((r) => r.rule === 'has-guardrails')).toBe(true);
  });

  it('should warn when skill produces data but has no observability metrics', () => {
    const skill = makeSkill({
      context: { consumes: [], produces: ['output'], memory: 'short-term' },
    });
    const results = lintSkill(skill);
    expect(results.some((r) => r.rule === 'observable-outputs')).toBe(true);
  });

  it('should warn when skill has full filesystem access and no guardrails', () => {
    const skill = makeSkill({
      security: { filesystem: 'full', network: 'none', secrets: [] },
    });
    const results = lintSkill(skill);
    expect(results.some((r) => r.rule === 'security-needs-guardrails')).toBe(true);
    expect(results.find((r) => r.rule === 'security-needs-guardrails')!.severity).toBe('error');
  });

  it('should also flag read-write filesystem access without guardrails', () => {
    const skill = makeSkill({
      security: { filesystem: 'read-write', network: 'none', secrets: [] },
    });
    const results = lintSkill(skill);
    expect(results.some((r) => r.rule === 'security-needs-guardrails')).toBe(true);
  });

  it('should pass clean for a well-configured skill', () => {
    const skill = makeSkill({
      strategy: { tools: ['read_file'], approach: 'diff-first' },
      guardrails: ['max_tokens: 1000'],
      observability: { trace_level: 'detailed', metrics: ['tokens'] },
    });
    const results = lintSkill(skill);
    const errors = results.filter((r) => r.severity === 'error');
    expect(errors).toHaveLength(0);
  });

  it('should not flag observable-outputs when skill has metrics', () => {
    const skill = makeSkill({
      context: { consumes: [], produces: ['output'], memory: 'short-term' },
      observability: { trace_level: 'standard', metrics: ['quality'] },
    });
    const results = lintSkill(skill);
    expect(results.some((r) => r.rule === 'observable-outputs')).toBe(false);
  });

  it('should include facet info in lint results', () => {
    const results = lintSkill(makeSkill());
    for (const r of results) {
      expect(r.facet).toBeDefined();
      expect(r.facet.length).toBeGreaterThan(0);
    }
  });
});
