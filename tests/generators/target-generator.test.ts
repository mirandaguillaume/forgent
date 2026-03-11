import { describe, it, expect } from 'vitest';
import { getGenerator, registerGenerator, getAvailableTargets, type TargetGenerator } from '../../src/generators/target-generator.js';

describe('TargetGenerator registry', () => {
  it('should throw for unknown target', () => {
    expect(() => getGenerator('unknown' as any)).toThrow('Unknown build target');
  });

  it('should return registered generator', () => {
    // Note: claude will be registered by claude-generator.ts in Task 2
    // For now, test the registry mechanism with a mock
    const mock: TargetGenerator = {
      target: 'claude',
      defaultOutputDir: '.claude',
      generateSkill: () => '',
      generateAgent: () => '',
      generateInstructions: () => null,
      getSkillPath: (n) => `skills/${n}/SKILL.md`,
      getAgentPath: (n) => `agents/${n}.md`,
      getInstructionsPath: () => null,
    };
    registerGenerator('claude', () => mock);
    const gen = getGenerator('claude');
    expect(gen.target).toBe('claude');
    expect(gen.defaultOutputDir).toBe('.claude');
  });

  it('should list available targets', () => {
    const targets = getAvailableTargets();
    expect(targets).toContain('claude');
  });
});
