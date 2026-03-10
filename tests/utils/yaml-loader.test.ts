import { describe, it, expect } from 'vitest';
import { parseSkillYaml, parseAgentYaml } from '../../src/utils/yaml-loader.js';

const validSkillYaml = `
skill: code-review
version: "1.0.0"
context:
  consumes: [git_diff]
  produces: [comments]
  memory: conversation
strategy:
  tools: [read_file]
  approach: diff-first
guardrails: []
depends_on: []
observability:
  trace_level: minimal
  metrics: []
security:
  filesystem: read-only
  network: none
  secrets: []
negotiation:
  file_conflicts: yield
  priority: 1
`;

describe('YAML loader', () => {
  it('should parse a valid skill YAML string', () => {
    const result = parseSkillYaml(validSkillYaml);
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.skill).toBe('code-review');
      expect(result.data.version).toBe('1.0.0');
      expect(result.data.context.memory).toBe('conversation');
    }
  });

  it('should return error for invalid YAML syntax', () => {
    const result = parseSkillYaml('skill: [invalid: yaml: :::');
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error).toBeDefined();
      expect(result.error).toContain('YAML syntax error');
    }
  });

  it('should return validation errors for schema violations', () => {
    const result = parseSkillYaml('skill: incomplete\nversion: "1.0.0"');
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.validationErrors).toBeDefined();
      expect(result.validationErrors!.length).toBeGreaterThan(0);
    }
  });

  it('should reject skill with invalid enum values', () => {
    const yaml = `
skill: bad
version: "1.0.0"
context:
  consumes: []
  produces: []
  memory: invalid-memory
strategy:
  tools: []
  approach: seq
guardrails: []
depends_on: []
observability:
  trace_level: minimal
  metrics: []
security:
  filesystem: read-only
  network: none
  secrets: []
negotiation:
  file_conflicts: yield
  priority: 1
`;
    const result = parseSkillYaml(yaml);
    expect(result.success).toBe(false);
  });

  it('should parse a valid agent YAML string', () => {
    const yaml = `
agent: reviewer
skills: [code-review, security-audit]
orchestration: parallel-then-merge
description: "Reviews code"
`;
    const result = parseAgentYaml(yaml);
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.data.agent).toBe('reviewer');
      expect(result.data.skills).toHaveLength(2);
    }
  });

  it('should reject agent with invalid orchestration', () => {
    const yaml = `
agent: bad
skills: [x]
orchestration: nonexistent
`;
    const result = parseAgentYaml(yaml);
    expect(result.success).toBe(false);
  });

  it('should handle empty YAML content', () => {
    const result = parseSkillYaml('');
    expect(result.success).toBe(false);
  });
});
