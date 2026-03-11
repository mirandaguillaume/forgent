import type { TargetGenerator } from './target-generator.js';
import type { SkillBehavior } from '../model/skill-behavior.js';
import type { AgentComposition } from '../model/agent.js';
import { generateSkillMd } from './skill-generator.js';
import { generateAgentMd } from './agent-generator.js';
import { registerGenerator } from './target-generator.js';

export function createClaudeGenerator(): TargetGenerator {
  return {
    target: 'claude',
    defaultOutputDir: '.claude',
    generateSkill: (skill: SkillBehavior) => generateSkillMd(skill),
    generateAgent: (agent: AgentComposition, skills: SkillBehavior[], outputDir: string) =>
      generateAgentMd(agent, skills, outputDir),
    generateInstructions: () => null,
    getSkillPath: (name: string) => `skills/${name}/SKILL.md`,
    getAgentPath: (name: string) => `agents/${name}.md`,
    getInstructionsPath: () => null,
  };
}

registerGenerator('claude', createClaudeGenerator);
