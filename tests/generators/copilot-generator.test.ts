import { describe, it, expect } from 'vitest';
import { createCopilotGenerator } from '../../src/generators/copilot-generator.js';
import type { SkillBehavior } from '../../src/model/skill-behavior.js';
import type { AgentComposition } from '../../src/model/agent.js';

const makeSkill = (overrides: Partial<SkillBehavior> = {}): SkillBehavior => ({
  skill: 'code-review',
  version: '1.0.0',
  context: { consumes: ['git_diff'], produces: ['review_comments'], memory: 'conversation' },
  strategy: { tools: ['read_file', 'grep'], approach: 'diff-first', steps: [] },
  guardrails: ['no_approve_without_tests'],
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

describe('CopilotGenerator', () => {
  it('should have target set to copilot', () => {
    const gen = createCopilotGenerator();
    expect(gen.target).toBe('copilot');
  });

  it('should have defaultOutputDir set to .github', () => {
    const gen = createCopilotGenerator();
    expect(gen.defaultOutputDir).toBe('.github');
  });

  it('should generate skill markdown', () => {
    const gen = createCopilotGenerator();
    const md = gen.generateSkill(makeSkill());
    expect(md).toContain('# Code Review');
    expect(md).toContain('---');
  });

  it('should getSkillPath return skills/<name>/SKILL.md', () => {
    const gen = createCopilotGenerator();
    expect(gen.getSkillPath('code-review')).toBe('skills/code-review/SKILL.md');
  });

  it('should getAgentPath return agents/<name>.agent.md', () => {
    const gen = createCopilotGenerator();
    expect(gen.getAgentPath('reviewer')).toBe('agents/reviewer.agent.md');
  });

  it('should getInstructionsPath return copilot-instructions.md', () => {
    const gen = createCopilotGenerator();
    expect(gen.getInstructionsPath()).toBe('copilot-instructions.md');
  });

  it('should generateInstructions work with skills', () => {
    const gen = createCopilotGenerator();
    const result = gen.generateInstructions([makeSkill()], []);
    expect(result).not.toBeNull();
    expect(result!).toContain('# Project Instructions');
    expect(result!).toContain('Code Review');
  });

  it('should generateInstructions return null for empty inputs', () => {
    const gen = createCopilotGenerator();
    expect(gen.generateInstructions([], [])).toBeNull();
  });

  it('should generateAgent produce markdown', () => {
    const gen = createCopilotGenerator();
    const md = gen.generateAgent(makeAgent(), [makeSkill()], '.github');
    expect(md).toContain('name: reviewer');
    expect(md).toContain('.github/skills/code-review/SKILL.md');
  });
});
