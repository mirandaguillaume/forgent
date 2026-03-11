import { describe, it, expect } from 'vitest';
import { generateCopilotInstructions } from '../../src/generators/copilot-instructions-generator.js';
import type { SkillBehavior } from '../../src/model/skill-behavior.js';
import type { AgentComposition } from '../../src/model/agent.js';

const makeSkill = (overrides: Partial<SkillBehavior> = {}): SkillBehavior => ({
  skill: 'code-review',
  version: '1.0.0',
  context: { consumes: ['git_diff'], produces: ['review_comments'], memory: 'conversation' },
  strategy: { tools: ['read_file', 'grep'], approach: 'diff-first', steps: [] },
  guardrails: ['no_approve_without_tests', { max_comments: 15 }],
  depends_on: [],
  observability: { trace_level: 'detailed', metrics: ['tokens'] },
  security: { filesystem: 'read-only', network: 'none', secrets: [] },
  negotiation: { file_conflicts: 'yield', priority: 1 },
  ...overrides,
});

const makeAgent = (overrides: Partial<AgentComposition> = {}): AgentComposition => ({
  agent: 'reviewer',
  skills: ['code-review'],
  orchestration: 'sequential',
  description: 'Reviews code',
  ...overrides,
});

describe('Copilot instructions generator', () => {
  it('should return null for empty inputs', () => {
    expect(generateCopilotInstructions([], [])).toBeNull();
  });

  it('should include project header', () => {
    const result = generateCopilotInstructions([makeSkill()], []);
    expect(result).not.toBeNull();
    expect(result!).toContain('# Project Instructions');
  });

  it('should list skills with descriptions', () => {
    const skill1 = makeSkill({ skill: 'code-review' });
    const skill2 = makeSkill({ skill: 'test-runner', strategy: { tools: [], approach: 'sequential', steps: [] } });
    const result = generateCopilotInstructions([skill1, skill2], []);
    expect(result).not.toBeNull();
    expect(result!).toContain('Code Review');
    expect(result!).toContain('Test Runner');
  });

  it('should list agents', () => {
    const result = generateCopilotInstructions([], [makeAgent()]);
    expect(result).not.toBeNull();
    expect(result!).toContain('Reviewer');
    expect(result!).toContain('Reviews code');
  });

  it('should include global guardrails from skills', () => {
    const skill = makeSkill({ guardrails: ['never_delete_production_data', { timeout: '10min' }] });
    const result = generateCopilotInstructions([skill], []);
    expect(result).not.toBeNull();
    expect(result!).toContain('## Global Guardrails');
    expect(result!).toContain('never_delete_production_data');
  });

  it('should include both skills and agents when present', () => {
    const result = generateCopilotInstructions([makeSkill()], [makeAgent()]);
    expect(result).not.toBeNull();
    expect(result!).toContain('## Available Skills');
    expect(result!).toContain('## Available Agents');
  });

  it('should skip skills section when no skills', () => {
    const result = generateCopilotInstructions([], [makeAgent()]);
    expect(result).not.toBeNull();
    expect(result!).not.toContain('## Available Skills');
  });

  it('should skip agents section when no agents', () => {
    const result = generateCopilotInstructions([makeSkill()], []);
    expect(result).not.toBeNull();
    expect(result!).not.toContain('## Available Agents');
  });
});
