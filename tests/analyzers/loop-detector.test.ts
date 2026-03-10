import { describe, it, expect } from 'vitest';
import { detectLoopRisks, type LoopRisk } from '../../src/analyzers/loop-detector.js';
import type { SkillBehavior } from '../../src/model/skill-behavior.js';

const makeSkill = (overrides: Partial<SkillBehavior>): SkillBehavior => ({
  skill: 'test',
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

describe('Loop detector', () => {
  it('should flag skill that consumes its own output', () => {
    const skill = makeSkill({
      skill: 'self-loop',
      context: { consumes: ['data'], produces: ['data'], memory: 'short-term' },
    });
    const risks = detectLoopRisks(skill);
    expect(risks.some((r) => r.type === 'self-reference')).toBe(true);
    expect(risks.find((r) => r.type === 'self-reference')!.severity).toBe('error');
  });

  it('should flag skill with no timeout guardrail and long-term memory', () => {
    const skill = makeSkill({
      skill: 'unbounded',
      context: { consumes: [], produces: ['data'], memory: 'long-term' },
      guardrails: [],
    });
    const risks = detectLoopRisks(skill);
    expect(risks.some((r) => r.type === 'no-timeout')).toBe(true);
  });

  it('should flag skill with no timeout guardrail and conversation memory', () => {
    const skill = makeSkill({
      skill: 'unbounded-conv',
      context: { consumes: [], produces: [], memory: 'conversation' },
      guardrails: [],
    });
    const risks = detectLoopRisks(skill);
    expect(risks.some((r) => r.type === 'no-timeout')).toBe(true);
  });

  it('should not flag skill with timeout guardrail', () => {
    const skill = makeSkill({
      skill: 'bounded',
      context: { consumes: [], produces: [], memory: 'long-term' },
      guardrails: [{ timeout: '5min' }],
    });
    const risks = detectLoopRisks(skill);
    expect(risks.some((r) => r.type === 'no-timeout')).toBe(false);
  });

  it('should not flag skill with string timeout guardrail', () => {
    const skill = makeSkill({
      skill: 'bounded-str',
      context: { consumes: [], produces: [], memory: 'long-term' },
      guardrails: ['timeout: 10min'],
    });
    const risks = detectLoopRisks(skill);
    expect(risks.some((r) => r.type === 'no-timeout')).toBe(false);
  });

  it('should detect multiple overlapping consumes/produces', () => {
    const skill = makeSkill({
      skill: 'multi-loop',
      context: { consumes: ['a', 'b', 'c'], produces: ['b', 'c', 'd'], memory: 'short-term' },
    });
    const risks = detectLoopRisks(skill);
    const selfRef = risks.find((r) => r.type === 'self-reference');
    expect(selfRef).toBeDefined();
    expect(selfRef!.message).toContain('b');
    expect(selfRef!.message).toContain('c');
  });

  it('should not flag clean skill', () => {
    const skill = makeSkill({
      skill: 'clean',
      context: { consumes: ['input'], produces: ['output'], memory: 'short-term' },
    });
    const risks = detectLoopRisks(skill);
    expect(risks).toHaveLength(0);
  });
});
