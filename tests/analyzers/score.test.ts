import { describe, it, expect } from 'vitest';
import { scoreSkill, scoreAgent, type SkillScore, type AgentScore } from '../../src/analyzers/score.js';
import type { SkillBehavior } from '../../src/model/skill-behavior.js';
import type { AgentComposition } from '../../src/model/agent.js';

const makeSkill = (overrides: Partial<SkillBehavior> = {}): SkillBehavior => ({
  skill: 'test-skill',
  version: '1.0.0',
  context: { consumes: ['input'], produces: ['output'], memory: 'conversation' },
  strategy: { tools: ['read_file', 'grep'], approach: 'diff-first', steps: ['step1', 'step2'] },
  guardrails: [{ timeout: '5min' }, 'max_retries: 3'],
  depends_on: [],
  observability: { trace_level: 'detailed', metrics: ['tokens', 'latency'] },
  security: { filesystem: 'read-only', network: 'none', secrets: [] },
  negotiation: { file_conflicts: 'yield', priority: 1 },
  ...overrides,
});

const makeAgent = (overrides: Partial<AgentComposition> = {}): AgentComposition => ({
  agent: 'test-agent',
  skills: ['skill-a', 'skill-b'],
  orchestration: 'sequential',
  description: 'A well-described agent',
  ...overrides,
});

describe('scoreSkill', () => {
  it('should give high score to a well-defined skill', () => {
    const result = scoreSkill(makeSkill());
    expect(result.total).toBeGreaterThanOrEqual(80);
  });

  it('should penalize missing tools', () => {
    const full = scoreSkill(makeSkill());
    const noTools = scoreSkill(makeSkill({ strategy: { tools: [], approach: 'seq', steps: ['s1'] } }));
    expect(noTools.total).toBeLessThan(full.total);
    expect(noTools.breakdown.strategy).toBeLessThan(full.breakdown.strategy);
  });

  it('should penalize missing guardrails', () => {
    const full = scoreSkill(makeSkill());
    const noGuardrails = scoreSkill(makeSkill({ guardrails: [] }));
    expect(noGuardrails.total).toBeLessThan(full.total);
    expect(noGuardrails.breakdown.guardrails).toBe(0);
  });

  it('should penalize missing steps', () => {
    const withSteps = scoreSkill(makeSkill());
    const noSteps = scoreSkill(makeSkill({ strategy: { tools: ['read_file'], approach: 'seq', steps: [] } }));
    expect(noSteps.breakdown.strategy).toBeLessThan(withSteps.breakdown.strategy);
  });

  it('should penalize missing observability', () => {
    const full = scoreSkill(makeSkill());
    const noObs = scoreSkill(makeSkill({ observability: { trace_level: 'minimal', metrics: [] } }));
    expect(noObs.breakdown.observability).toBeLessThan(full.breakdown.observability);
  });

  it('should reward restrictive security', () => {
    const readOnly = scoreSkill(makeSkill({ security: { filesystem: 'read-only', network: 'none', secrets: [] } }));
    const fullAccess = scoreSkill(makeSkill({ security: { filesystem: 'full', network: 'full', secrets: ['API_KEY'] } }));
    expect(readOnly.breakdown.security).toBeGreaterThan(fullAccess.breakdown.security);
  });

  it('should penalize empty context (no consumes or produces)', () => {
    const full = scoreSkill(makeSkill());
    const emptyCtx = scoreSkill(makeSkill({ context: { consumes: [], produces: [], memory: 'short-term' } }));
    expect(emptyCtx.breakdown.context).toBeLessThan(full.breakdown.context);
  });

  it('should return breakdown by facet', () => {
    const result = scoreSkill(makeSkill());
    expect(result.breakdown).toHaveProperty('context');
    expect(result.breakdown).toHaveProperty('strategy');
    expect(result.breakdown).toHaveProperty('guardrails');
    expect(result.breakdown).toHaveProperty('observability');
    expect(result.breakdown).toHaveProperty('security');
  });

  it('should return total between 0 and 100', () => {
    const result = scoreSkill(makeSkill());
    expect(result.total).toBeGreaterThanOrEqual(0);
    expect(result.total).toBeLessThanOrEqual(100);
  });

  it('should give minimum score to worst-case skill', () => {
    const worst = scoreSkill(makeSkill({
      context: { consumes: [], produces: [], memory: 'short-term' },
      strategy: { tools: [], approach: '', steps: [] },
      guardrails: [],
      observability: { trace_level: 'minimal', metrics: [] },
      security: { filesystem: 'full', network: 'full', secrets: ['SECRET'] },
    }));
    expect(worst.total).toBeLessThan(30);
  });
});

describe('scoreAgent', () => {
  it('should give high score to a well-defined agent', () => {
    const skills = [
      makeSkill({ skill: 'skill-a', context: { consumes: [], produces: ['data'], memory: 'short-term' } }),
      makeSkill({ skill: 'skill-b', context: { consumes: ['data'], produces: ['result'], memory: 'short-term' } }),
    ];
    const result = scoreAgent(makeAgent(), skills);
    expect(result.total).toBeGreaterThanOrEqual(80);
  });

  it('should penalize missing description', () => {
    const withDesc = scoreAgent(makeAgent(), []);
    const noDesc = scoreAgent(makeAgent({ description: undefined }), []);
    expect(noDesc.total).toBeLessThan(withDesc.total);
  });

  it('should penalize broken data-flow in sequential mode', () => {
    const goodFlow = [
      makeSkill({ skill: 'skill-a', context: { consumes: [], produces: ['data'], memory: 'short-term' } }),
      makeSkill({ skill: 'skill-b', context: { consumes: ['data'], produces: ['result'], memory: 'short-term' } }),
    ];
    const brokenFlow = [
      makeSkill({ skill: 'skill-a', context: { consumes: ['data'], produces: ['result'], memory: 'short-term' } }),
      makeSkill({ skill: 'skill-b', context: { consumes: [], produces: ['data'], memory: 'short-term' } }),
    ];
    const good = scoreAgent(makeAgent(), goodFlow);
    const broken = scoreAgent(makeAgent(), brokenFlow);
    expect(broken.breakdown.dataFlow).toBeLessThan(good.breakdown.dataFlow);
  });

  it('should not penalize environment inputs (consumes not produced by any skill)', () => {
    const skills = [
      makeSkill({ skill: 'skill-a', context: { consumes: ['file_tree', 'source_code'], produces: ['lint_results'], memory: 'short-term' } }),
      makeSkill({ skill: 'skill-b', context: { consumes: ['file_tree', 'lint_results'], produces: ['report'], memory: 'short-term' } }),
    ];
    const result = scoreAgent(makeAgent(), skills);
    // lint_results is the only inter-skill dependency and it's correctly ordered
    expect(result.breakdown.dataFlow).toBe(35);
  });

  it('should penalize single-skill agents', () => {
    const multi = scoreAgent(makeAgent({ skills: ['a', 'b'] }), []);
    const single = scoreAgent(makeAgent({ skills: ['a'] }), []);
    expect(single.breakdown.composition).toBeLessThan(multi.breakdown.composition);
  });

  it('should return total between 0 and 100', () => {
    const result = scoreAgent(makeAgent(), []);
    expect(result.total).toBeGreaterThanOrEqual(0);
    expect(result.total).toBeLessThanOrEqual(100);
  });
});
