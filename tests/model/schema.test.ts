import { describe, it, expect } from 'vitest';
import { validateSkill, validateAgent } from '../../src/model/schema.js';

describe('Schema validation', () => {
  it('should validate a correct skill', () => {
    const result = validateSkill({
      skill: 'code-review',
      version: '1.0.0',
      context: { consumes: ['git_diff'], produces: ['comments'], memory: 'conversation' },
      strategy: { tools: ['read_file'], approach: 'diff-first' },
      guardrails: [],
      depends_on: [],
      observability: { trace_level: 'minimal', metrics: [] },
      security: { filesystem: 'read-only', network: 'none', secrets: [] },
      negotiation: { file_conflicts: 'yield', priority: 1 },
    });
    expect(result.valid).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it('should reject a skill missing required fields', () => {
    const result = validateSkill({ skill: 'incomplete' });
    expect(result.valid).toBe(false);
    expect(result.errors.length).toBeGreaterThan(0);
  });

  it('should reject invalid memory type', () => {
    const result = validateSkill({
      skill: 'bad-memory',
      version: '1.0.0',
      context: { consumes: [], produces: [], memory: 'invalid-type' },
      strategy: { tools: [], approach: 'seq' },
      guardrails: [],
      depends_on: [],
      observability: { trace_level: 'minimal', metrics: [] },
      security: { filesystem: 'read-only', network: 'none', secrets: [] },
      negotiation: { file_conflicts: 'yield', priority: 1 },
    });
    expect(result.valid).toBe(false);
  });

  it('should validate a correct agent composition', () => {
    const result = validateAgent({
      agent: 'reviewer',
      skills: ['code-review', 'security-audit'],
      orchestration: 'parallel-then-merge',
    });
    expect(result.valid).toBe(true);
  });

  it('should reject agent with invalid orchestration', () => {
    const result = validateAgent({
      agent: 'bad',
      skills: [],
      orchestration: 'nonexistent',
    });
    expect(result.valid).toBe(false);
  });
});
