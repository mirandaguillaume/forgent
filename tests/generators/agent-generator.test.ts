import { describe, it, expect } from 'vitest';
import { generateAgentMd } from '../../src/generators/agent-generator.js';
import type { AgentComposition } from '../../src/model/agent.js';

const makeAgent = (overrides: Partial<AgentComposition> = {}): AgentComposition => ({
  agent: 'reviewer',
  skills: ['code-review', 'test-runner', 'security-audit'],
  orchestration: 'parallel-then-merge',
  description: 'Reviews code with testing and security analysis',
  ...overrides,
});

describe('Agent generator', () => {
  it('should generate valid frontmatter', () => {
    const md = generateAgentMd(makeAgent());
    expect(md).toMatch(/^---\nname: reviewer\n/);
    expect(md).toContain('model: inherit');
    expect(md).toContain('---');
  });

  it('should include description in frontmatter', () => {
    const md = generateAgentMd(makeAgent());
    expect(md).toContain('description: Reviews code with testing and security analysis');
  });

  it('should describe the agent role', () => {
    const md = generateAgentMd(makeAgent());
    expect(md).toContain('reviewer');
    expect(md).toContain('parallel-then-merge');
  });

  it('should list all skills', () => {
    const md = generateAgentMd(makeAgent());
    expect(md).toContain('code-review');
    expect(md).toContain('test-runner');
    expect(md).toContain('security-audit');
  });

  it('should handle agent without description', () => {
    const agent = makeAgent({ description: undefined });
    const md = generateAgentMd(agent);
    expect(md).toContain('name: reviewer');
    expect(md).not.toContain('description: undefined');
  });

  it('should handle single skill agent', () => {
    const agent = makeAgent({ skills: ['code-review'], orchestration: 'sequential' });
    const md = generateAgentMd(agent);
    expect(md).toContain('code-review');
    expect(md).toContain('sequential');
  });

  it('should generate concise output', () => {
    const md = generateAgentMd(makeAgent());
    const words = md.split(/\s+/).filter((w) => w.length > 0).length;
    expect(words).toBeLessThan(200);
  });
});
