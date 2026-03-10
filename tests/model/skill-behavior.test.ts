import { describe, it, expect } from 'vitest';
import type { SkillBehavior, ContextFacet, StrategyFacet, GuardrailsFacet, DependencyFacet, ObservabilityFacet, SecurityFacet, NegotiationStrategy } from '../../src/model/skill-behavior.js';

describe('SkillBehavior types', () => {
  it('should allow creating a valid skill behavior', () => {
    const skill: SkillBehavior = {
      skill: 'code-review',
      version: '1.2.0',
      context: {
        consumes: ['git_diff', 'file_tree'],
        produces: ['review_comments', 'risk_score'],
        memory: 'conversation',
      },
      strategy: {
        tools: ['read_file', 'grep'],
        approach: 'diff-first',
        steps: ['analyze_diff', 'check_patterns'],
      },
      guardrails: [
        'no_approve_without_tests',
        { max_comments: 10 },
        { timeout: '5min' },
      ],
      depends_on: [
        { skill: 'test-coverage', provides: 'test_results' },
      ],
      observability: {
        trace_level: 'detailed',
        metrics: ['tokens', 'latency', 'decisions'],
      },
      security: {
        filesystem: 'read-only',
        network: 'none',
        secrets: [],
      },
      negotiation: {
        file_conflicts: 'yield',
        priority: 2,
      },
    };

    expect(skill.skill).toBe('code-review');
    expect(skill.version).toBe('1.2.0');
    expect(skill.context.consumes).toContain('git_diff');
    expect(skill.strategy.tools).toHaveLength(2);
    expect(skill.guardrails).toHaveLength(3);
    expect(skill.depends_on).toHaveLength(1);
    expect(skill.observability.trace_level).toBe('detailed');
    expect(skill.security.filesystem).toBe('read-only');
    expect(skill.negotiation.file_conflicts).toBe('yield');
  });

  it('should allow minimal skill with only required fields', () => {
    const skill: SkillBehavior = {
      skill: 'simple-task',
      version: '0.1.0',
      context: {
        consumes: [],
        produces: [],
        memory: 'short-term',
      },
      strategy: {
        tools: [],
        approach: 'sequential',
      },
      guardrails: [],
      depends_on: [],
      observability: {
        trace_level: 'minimal',
        metrics: [],
      },
      security: {
        filesystem: 'none',
        network: 'none',
        secrets: [],
      },
      negotiation: {
        file_conflicts: 'yield',
        priority: 0,
      },
    };

    expect(skill.skill).toBe('simple-task');
  });
});
