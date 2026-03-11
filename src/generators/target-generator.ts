import type { SkillBehavior } from '../model/skill-behavior.js';
import type { AgentComposition } from '../model/agent.js';

export type BuildTarget = 'claude' | 'copilot';

export interface TargetGenerator {
  target: BuildTarget;
  defaultOutputDir: string;
  generateSkill(skill: SkillBehavior): string;
  generateAgent(agent: AgentComposition, skills: SkillBehavior[], outputDir: string): string;
  generateInstructions(skills: SkillBehavior[], agents: AgentComposition[]): string | null;
  getSkillPath(name: string): string;
  getAgentPath(name: string): string;
  getInstructionsPath(): string | null;
}

type GeneratorFactory = () => TargetGenerator;

const registry = new Map<BuildTarget, GeneratorFactory>();

export function registerGenerator(target: BuildTarget, factory: GeneratorFactory): void {
  registry.set(target, factory);
}

export function getGenerator(target: BuildTarget): TargetGenerator {
  const factory = registry.get(target);
  if (!factory) {
    throw new Error(`Unknown build target: "${target}". Available targets: ${getAvailableTargets().join(', ') || 'none'}`);
  }
  return factory();
}

export function getAvailableTargets(): BuildTarget[] {
  return [...registry.keys()];
}
