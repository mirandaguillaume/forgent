import { describe, it, expect } from 'vitest';
import { createClaudeGenerator } from '../../src/generators/claude-generator.js';
import type { SkillBehavior } from '../../src/model/skill-behavior.js';

const makeSkill = (overrides: Partial<SkillBehavior> = {}): SkillBehavior => ({
  skill: 'code-review',
  version: '1.0.0',
  context: { consumes: ['git_diff'], produces: ['review_comments'], memory: 'conversation' },
  strategy: { tools: ['read_file', 'grep'], approach: 'diff-first', steps: [] },
  guardrails: [{ timeout: '5min' }],
  depends_on: [],
  observability: { trace_level: 'detailed', metrics: ['tokens'] },
  security: { filesystem: 'read-only', network: 'none', secrets: [] },
  negotiation: { file_conflicts: 'yield', priority: 1 },
  ...overrides,
});

describe('ClaudeGenerator', () => {
  const gen = createClaudeGenerator();

  it('should have correct target and defaultOutputDir', () => {
    expect(gen.target).toBe('claude');
    expect(gen.defaultOutputDir).toBe('.claude');
  });

  it('should generate skill markdown with frontmatter', () => {
    const md = gen.generateSkill(makeSkill());
    expect(md).toContain('name: code-review');
    expect(md).toContain('## Guardrails');
  });

  it('should return correct skill path', () => {
    expect(gen.getSkillPath('code-review')).toBe('skills/code-review/SKILL.md');
  });

  it('should return correct agent path', () => {
    expect(gen.getAgentPath('reviewer')).toBe('agents/reviewer.md');
  });

  it('should return null for instructions', () => {
    expect(gen.generateInstructions([], [])).toBeNull();
    expect(gen.getInstructionsPath()).toBeNull();
  });
});
