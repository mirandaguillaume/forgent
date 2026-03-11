import { describe, it, expect } from 'vitest';
import { generateCopilotAgentMd, resolveCopilotAgentTools } from '../../src/generators/copilot-agent-generator.js';
import type { AgentComposition } from '../../src/model/agent.js';
import type { SkillBehavior } from '../../src/model/skill-behavior.js';

const makeAgent = (overrides: Partial<AgentComposition> = {}): AgentComposition => ({
  agent: 'reviewer',
  skills: ['code-review', 'test-runner'],
  orchestration: 'sequential',
  description: 'Reviews code with testing',
  ...overrides,
});

const makeSkill = (overrides: Partial<SkillBehavior> = {}): SkillBehavior => ({
  skill: 'code-review',
  version: '1.0.0',
  context: { consumes: ['git_diff'], produces: ['review_comments'], memory: 'conversation' },
  strategy: { tools: ['read_file', 'grep'], approach: 'diff-first', steps: ['analyze diff', 'write review'] },
  guardrails: [{ timeout: '5min' }],
  depends_on: [],
  observability: { trace_level: 'detailed', metrics: ['tokens'] },
  security: { filesystem: 'read-only', network: 'none', secrets: [] },
  negotiation: { file_conflicts: 'yield', priority: 1 },
  ...overrides,
});

describe('Copilot agent generator — .agent.md format', () => {
  it('should generate valid frontmatter with name', () => {
    const skills = [makeSkill()];
    const md = generateCopilotAgentMd(makeAgent({ skills: ['code-review'] }), skills);
    expect(md).toMatch(/^---\nname: reviewer\n/);
    expect(md).toContain('description: Reviews code with testing');
  });

  it('should use Copilot tool aliases (NOT Claude names)', () => {
    const skills = [makeSkill()];
    const md = generateCopilotAgentMd(makeAgent({ skills: ['code-review'] }), skills);
    // Should contain lowercase copilot aliases
    expect(md).toContain('tools:');
    expect(md).toMatch(/tools: \[.*"read".*\]/);
    expect(md).toMatch(/tools: \[.*"search".*\]/);
    // Should NOT contain Claude tool names
    expect(md).not.toMatch(/tools:.*Read[",\]]/);
    expect(md).not.toMatch(/tools:.*Grep/);
    expect(md).not.toMatch(/tools:.*Glob/);
  });

  it('should format tools as YAML array', () => {
    const skills = [makeSkill()];
    const md = generateCopilotAgentMd(makeAgent({ skills: ['code-review'] }), skills);
    const toolsMatch = md.match(/tools: \[(.+)\]/);
    expect(toolsMatch).not.toBeNull();
    // Should be quoted strings in array format
    expect(toolsMatch![1]).toMatch(/"[a-z]+"/);
  });

  it('should reference skill paths under .github/', () => {
    const skills = [makeSkill()];
    const md = generateCopilotAgentMd(makeAgent({ skills: ['code-review'] }), skills);
    expect(md).toContain('.github/skills/code-review/SKILL.md');
  });

  it('should use custom outputDir for skill paths', () => {
    const skills = [makeSkill()];
    const md = generateCopilotAgentMd(makeAgent({ skills: ['code-review'] }), skills, 'build/output');
    expect(md).toContain('build/output/skills/code-review/SKILL.md');
  });

  it('should include execution strategy', () => {
    const md = generateCopilotAgentMd(makeAgent({ orchestration: 'sequential' }), []);
    expect(md).toContain('## Execution');
    expect(md).toContain('in order');
  });

  it('should describe parallel mode', () => {
    const md = generateCopilotAgentMd(makeAgent({ orchestration: 'parallel' }), []);
    expect(md).toContain('concurrently');
  });

  it('should describe parallel-then-merge mode', () => {
    const md = generateCopilotAgentMd(makeAgent({ orchestration: 'parallel-then-merge' }), []);
    expect(md).toContain('merge');
  });

  it('should describe adaptive mode', () => {
    const md = generateCopilotAgentMd(makeAgent({ orchestration: 'adaptive' }), []);
    expect(md).toContain('dynamically');
  });

  it('should number steps for sequential orchestration', () => {
    const skill1 = makeSkill({ skill: 'code-review' });
    const skill2 = makeSkill({ skill: 'test-runner', context: { consumes: [], produces: ['test_results'], memory: 'short-term' } });
    const md = generateCopilotAgentMd(makeAgent(), [skill1, skill2]);
    expect(md).toContain('Step 1: Code Review');
    expect(md).toContain('Step 2: Test Runner');
  });

  it('should handle agent without description', () => {
    const md = generateCopilotAgentMd(makeAgent({ description: undefined }), []);
    expect(md).toContain('name: reviewer');
    expect(md).not.toContain('description: undefined');
  });

  it('should not include tools when no skills resolved', () => {
    const md = generateCopilotAgentMd(makeAgent(), []);
    expect(md).not.toMatch(/^tools:/m);
  });
});

describe('resolveCopilotAgentTools', () => {
  it('should map Forgent tools to Copilot aliases', () => {
    const skill = makeSkill({ strategy: { tools: ['read_file', 'grep', 'bash'], approach: 'seq', steps: [] } });
    const tools = resolveCopilotAgentTools([skill]);
    expect(tools).toContain('read');
    expect(tools).toContain('search');
    expect(tools).toContain('execute');
  });

  it('should infer tools from security facet', () => {
    const skill = makeSkill({
      strategy: { tools: [], approach: 'seq', steps: [] },
      security: { filesystem: 'full', network: 'full', secrets: [] },
    });
    const tools = resolveCopilotAgentTools([skill]);
    expect(tools).toContain('edit');
    expect(tools).toContain('execute');
    expect(tools).toContain('web');
  });

  it('should deduplicate tools', () => {
    const skill1 = makeSkill({ strategy: { tools: ['read_file', 'grep'], approach: 'seq', steps: [] } });
    const skill2 = makeSkill({ strategy: { tools: ['read_file', 'bash'], approach: 'seq', steps: [] } });
    const tools = resolveCopilotAgentTools([skill1, skill2]);
    expect(tools.filter((t) => t === 'read').length).toBe(1);
  });
});
