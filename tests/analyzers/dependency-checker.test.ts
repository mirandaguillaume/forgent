import { describe, it, expect } from 'vitest';
import { checkDependencies, type DependencyIssue } from '../../src/analyzers/dependency-checker.js';
import type { SkillBehavior } from '../../src/model/skill-behavior.js';

const makeSkill = (name: string, deps: Array<{ skill: string; provides: string }> = [], consumes: string[] = [], produces: string[] = []): SkillBehavior => ({
  skill: name,
  version: '1.0.0',
  context: { consumes, produces, memory: 'short-term' },
  strategy: { tools: [], approach: 'sequential' },
  guardrails: [],
  depends_on: deps,
  observability: { trace_level: 'minimal', metrics: [] },
  security: { filesystem: 'none', network: 'none', secrets: [] },
  negotiation: { file_conflicts: 'yield', priority: 0 },
});

describe('Dependency checker', () => {
  it('should detect circular dependencies', () => {
    const skills = [
      makeSkill('a', [{ skill: 'b', provides: 'x' }]),
      makeSkill('b', [{ skill: 'a', provides: 'y' }]),
    ];
    const issues = checkDependencies(skills);
    expect(issues.some((i) => i.type === 'circular')).toBe(true);
  });

  it('should detect missing dependencies', () => {
    const skills = [
      makeSkill('a', [{ skill: 'nonexistent', provides: 'x' }]),
    ];
    const issues = checkDependencies(skills);
    expect(issues.some((i) => i.type === 'missing')).toBe(true);
  });

  it('should detect unmet context requirements', () => {
    const skills = [
      makeSkill('a', [{ skill: 'b', provides: 'test_results' }], ['test_results']),
      makeSkill('b', [], [], ['other_data']),
    ];
    const issues = checkDependencies(skills);
    expect(issues.some((i) => i.type === 'unmet-context')).toBe(true);
  });

  it('should pass for valid dependency graph', () => {
    const skills = [
      makeSkill('test-runner', [], [], ['test_results']),
      makeSkill('reviewer', [{ skill: 'test-runner', provides: 'test_results' }], ['test_results']),
    ];
    const issues = checkDependencies(skills);
    expect(issues).toHaveLength(0);
  });

  it('should detect 3-node circular dependency', () => {
    const skills = [
      makeSkill('a', [{ skill: 'b', provides: 'x' }]),
      makeSkill('b', [{ skill: 'c', provides: 'y' }]),
      makeSkill('c', [{ skill: 'a', provides: 'z' }]),
    ];
    const issues = checkDependencies(skills);
    expect(issues.some((i) => i.type === 'circular')).toBe(true);
  });

  it('should handle skills with no dependencies', () => {
    const skills = [
      makeSkill('standalone-a'),
      makeSkill('standalone-b'),
    ];
    const issues = checkDependencies(skills);
    expect(issues).toHaveLength(0);
  });

  it('should include skill name in issue details', () => {
    const skills = [
      makeSkill('a', [{ skill: 'missing', provides: 'x' }]),
    ];
    const issues = checkDependencies(skills);
    expect(issues[0].skill).toBe('a');
    expect(issues[0].message).toContain('missing');
  });
});
