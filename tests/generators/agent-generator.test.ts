import { describe, it, expect } from 'vitest';
import { generateAgentMd, resolveAgentTools } from '../../src/generators/agent-generator.js';
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

describe('Agent generator — skill references', () => {
  it('should reference skill files instead of embedding', () => {
    const skills = [makeSkill()];
    const md = generateAgentMd(makeAgent({ skills: ['code-review'] }), skills);
    expect(md).toContain('.claude/skills/code-review/SKILL.md');
    expect(md).toContain('Read `.claude/skills/code-review/SKILL.md`');
  });

  it('should include data-flow hints (consumes/produces)', () => {
    const skills = [makeSkill()];
    const md = generateAgentMd(makeAgent({ skills: ['code-review'] }), skills);
    expect(md).toContain('Consumes: git_diff');
    expect(md).toContain('Produces: review_comments');
  });

  it('should number steps for sequential orchestration', () => {
    const skill1 = makeSkill({ skill: 'code-review' });
    const skill2 = makeSkill({ skill: 'test-runner', context: { consumes: [], produces: ['test_results'], memory: 'short-term' } });
    const md = generateAgentMd(makeAgent(), [skill1, skill2]);
    expect(md).toContain('Step 1: Code Review');
    expect(md).toContain('Step 2: Test Runner');
  });

  it('should use custom outputDir for skill paths', () => {
    const skills = [makeSkill()];
    const md = generateAgentMd(makeAgent({ skills: ['code-review'] }), skills, 'build/output');
    expect(md).toContain('build/output/skills/code-review/SKILL.md');
  });

  it('should always include Read in tools for skill file access', () => {
    const skill = makeSkill({ strategy: { tools: ['bash'], approach: 'seq', steps: [] }, security: { filesystem: 'full', network: 'none', secrets: [] } });
    const md = generateAgentMd(makeAgent({ skills: ['code-review'] }), [skill]);
    expect(md).toMatch(/tools:.*Read/);
  });

  it('should list all produced outputs in Output section', () => {
    const skill1 = makeSkill({ skill: 'code-review', context: { consumes: [], produces: ['comments'], memory: 'short-term' } });
    const skill2 = makeSkill({ skill: 'test-runner', context: { consumes: [], produces: ['results'], memory: 'short-term' } });
    const md = generateAgentMd(makeAgent(), [skill1, skill2]);
    expect(md).toContain('comments');
    expect(md).toContain('results');
  });
});

describe('Agent generator — frontmatter', () => {
  it('should generate valid frontmatter with tools', () => {
    const skills = [makeSkill()];
    const md = generateAgentMd(makeAgent({ skills: ['code-review'] }), skills);
    expect(md).toMatch(/^---\nname: reviewer\n/);
    expect(md).toContain('tools:');
    expect(md).not.toContain('model: inherit');
  });

  it('should include description', () => {
    const md = generateAgentMd(makeAgent());
    expect(md).toContain('description: Reviews code with testing');
  });

  it('should not include tools when no skills resolved', () => {
    const md = generateAgentMd(makeAgent());
    expect(md).not.toMatch(/^tools:/m);
  });

  it('should handle agent without description', () => {
    const md = generateAgentMd(makeAgent({ description: undefined }));
    expect(md).toContain('name: reviewer');
    expect(md).not.toContain('description: undefined');
  });
});

describe('Agent generator — without resolved skills', () => {
  it('should still reference skill files by name', () => {
    const md = generateAgentMd(makeAgent());
    expect(md).toContain('.claude/skills/code-review/SKILL.md');
    expect(md).toContain('.claude/skills/test-runner/SKILL.md');
  });

  it('should not include consumes/produces hints without resolution', () => {
    const md = generateAgentMd(makeAgent());
    expect(md).not.toContain('Consumes:');
    expect(md).not.toContain('Produces:');
  });
});

describe('Agent generator — orchestration modes', () => {
  it('should describe sequential mode', () => {
    const md = generateAgentMd(makeAgent({ orchestration: 'sequential' }));
    expect(md).toContain('in order');
  });

  it('should describe parallel mode', () => {
    const md = generateAgentMd(makeAgent({ orchestration: 'parallel' }));
    expect(md).toContain('concurrently');
  });

  it('should describe parallel-then-merge mode', () => {
    const md = generateAgentMd(makeAgent({ orchestration: 'parallel-then-merge' }));
    expect(md).toContain('merge');
  });

  it('should describe adaptive mode', () => {
    const md = generateAgentMd(makeAgent({ orchestration: 'adaptive' }));
    expect(md).toContain('dynamically');
  });
});

describe('resolveAgentTools', () => {
  it('should map Forgent tools to Claude Code tools', () => {
    const skill = makeSkill({ strategy: { tools: ['read_file', 'grep', 'search'], approach: 'seq', steps: [] } });
    const tools = resolveAgentTools([skill]);
    expect(tools).toContain('Read');
    expect(tools).toContain('Grep');
    expect(tools).toContain('Glob');
  });

  it('should infer tools from security facet', () => {
    const skill = makeSkill({
      strategy: { tools: [], approach: 'seq', steps: [] },
      security: { filesystem: 'full', network: 'full', secrets: [] },
    });
    const tools = resolveAgentTools([skill]);
    expect(tools).toContain('Write');
    expect(tools).toContain('Bash');
    expect(tools).toContain('WebFetch');
  });

  it('should deduplicate tools', () => {
    const skill1 = makeSkill({ strategy: { tools: ['read_file', 'grep'], approach: 'seq', steps: [] } });
    const skill2 = makeSkill({ strategy: { tools: ['read_file', 'bash'], approach: 'seq', steps: [] } });
    const tools = resolveAgentTools([skill1, skill2]);
    expect(tools.filter((t) => t === 'Read').length).toBe(1);
  });
});
