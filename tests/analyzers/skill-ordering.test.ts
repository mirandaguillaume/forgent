import { describe, it, expect } from 'vitest';
import { checkSkillOrdering, type OrderingIssue } from '../../src/analyzers/skill-ordering.js';
import type { AgentComposition } from '../../src/model/agent.js';
import type { SkillBehavior } from '../../src/model/skill-behavior.js';

const makeAgent = (overrides: Partial<AgentComposition> = {}): AgentComposition => ({
  agent: 'test-agent',
  skills: ['code-review', 'test-runner'],
  orchestration: 'sequential',
  description: 'Reviews code changes',
  ...overrides,
});

const makeSkill = (name: string, consumes: string[] = [], produces: string[] = []): SkillBehavior => ({
  skill: name,
  version: '1.0.0',
  context: { consumes, produces, memory: 'short-term' },
  strategy: { tools: [], approach: 'sequential', steps: [] },
  guardrails: [],
  depends_on: [],
  observability: { trace_level: 'minimal', metrics: [] },
  security: { filesystem: 'none', network: 'none', secrets: [] },
  negotiation: { file_conflicts: 'yield', priority: 0 },
});

describe('Skill ordering — description-order mismatch', () => {
  it('should detect when description mentions skills in reverse order', () => {
    const agent = makeAgent({
      skills: ['code-review', 'test-runner'],
      orchestration: 'sequential',
      description: 'First use test-runner, then apply code-review',
    });
    const skillMap = new Map([
      ['code-review', makeSkill('code-review')],
      ['test-runner', makeSkill('test-runner')],
    ]);
    const issues = checkSkillOrdering(agent, skillMap);
    expect(issues.some((i) => i.type === 'description-order-mismatch')).toBe(true);
  });

  it('should NOT flag when description matches skills array order', () => {
    const agent = makeAgent({
      skills: ['test-runner', 'code-review'],
      orchestration: 'sequential',
      description: 'First run tests, then review the code',
    });
    const skillMap = new Map([
      ['test-runner', makeSkill('test-runner')],
      ['code-review', makeSkill('code-review')],
    ]);
    const issues = checkSkillOrdering(agent, skillMap);
    expect(issues.some((i) => i.type === 'description-order-mismatch')).toBe(false);
  });

  it('should NOT flag for parallel orchestration', () => {
    const agent = makeAgent({
      skills: ['code-review', 'test-runner'],
      orchestration: 'parallel',
      description: 'First running tests, then analyzing the diff',
    });
    const issues = checkSkillOrdering(agent, new Map());
    expect(issues.some((i) => i.type === 'description-order-mismatch')).toBe(false);
  });

  it('should NOT flag when description is absent', () => {
    const agent = makeAgent({
      skills: ['code-review', 'test-runner'],
      orchestration: 'sequential',
      description: undefined,
    });
    const issues = checkSkillOrdering(agent, new Map());
    expect(issues).toHaveLength(0);
  });

  it('should NOT flag when description mentions only one skill', () => {
    const agent = makeAgent({
      skills: ['code-review', 'test-runner'],
      orchestration: 'sequential',
      description: 'Focuses on reviewing code',
    });
    const skillMap = new Map([
      ['code-review', makeSkill('code-review')],
      ['test-runner', makeSkill('test-runner')],
    ]);
    const issues = checkSkillOrdering(agent, skillMap);
    expect(issues.some((i) => i.type === 'description-order-mismatch')).toBe(false);
  });

  it('should match skill names with hyphens as spaces in description', () => {
    const agent = makeAgent({
      skills: ['code-review', 'test-runner'],
      orchestration: 'sequential',
      description: 'Run the test runner first, then do code review',
    });
    const skillMap = new Map([
      ['code-review', makeSkill('code-review')],
      ['test-runner', makeSkill('test-runner')],
    ]);
    const issues = checkSkillOrdering(agent, skillMap);
    expect(issues.some((i) => i.type === 'description-order-mismatch')).toBe(true);
  });
});

describe('Skill ordering — data-flow order mismatch', () => {
  it('should detect when consumer comes before producer in skills array', () => {
    const agent = makeAgent({
      skills: ['code-review', 'test-runner'],
      orchestration: 'sequential',
    });
    const skillMap = new Map([
      ['code-review', makeSkill('code-review', ['test_results'], ['review_comments'])],
      ['test-runner', makeSkill('test-runner', [], ['test_results'])],
    ]);
    const issues = checkSkillOrdering(agent, skillMap);
    expect(issues.some((i) => i.type === 'data-flow-order-mismatch')).toBe(true);
    expect(issues.find((i) => i.type === 'data-flow-order-mismatch')!.message).toContain('test_results');
  });

  it('should NOT flag when producer comes before consumer', () => {
    const agent = makeAgent({
      skills: ['test-runner', 'code-review'],
      orchestration: 'sequential',
    });
    const skillMap = new Map([
      ['test-runner', makeSkill('test-runner', [], ['test_results'])],
      ['code-review', makeSkill('code-review', ['test_results'], ['review_comments'])],
    ]);
    const issues = checkSkillOrdering(agent, skillMap);
    expect(issues.some((i) => i.type === 'data-flow-order-mismatch')).toBe(false);
  });

  it('should NOT flag data-flow issues for parallel orchestration', () => {
    const agent = makeAgent({
      skills: ['code-review', 'test-runner'],
      orchestration: 'parallel',
    });
    const skillMap = new Map([
      ['code-review', makeSkill('code-review', ['test_results'], [])],
      ['test-runner', makeSkill('test-runner', [], ['test_results'])],
    ]);
    const issues = checkSkillOrdering(agent, skillMap);
    expect(issues.some((i) => i.type === 'data-flow-order-mismatch')).toBe(false);
  });

  it('should handle skills not found in skill map gracefully', () => {
    const agent = makeAgent({
      skills: ['nonexistent-a', 'nonexistent-b'],
      orchestration: 'sequential',
    });
    const issues = checkSkillOrdering(agent, new Map());
    expect(issues.some((i) => i.type === 'data-flow-order-mismatch')).toBe(false);
  });

  it('should detect multi-step chain ordering issue', () => {
    const agent = makeAgent({
      skills: ['a', 'b', 'c'],
      orchestration: 'sequential',
    });
    const skillMap = new Map([
      ['a', makeSkill('a', ['x'], ['y'])],
      ['b', makeSkill('b', ['y'], ['z'])],
      ['c', makeSkill('c', [], ['x'])],
    ]);
    const issues = checkSkillOrdering(agent, skillMap);
    expect(issues.some((i) => i.type === 'data-flow-order-mismatch')).toBe(true);
  });
});
