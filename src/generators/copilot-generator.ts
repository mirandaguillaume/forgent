import type { TargetGenerator } from './target-generator.js';
import type { SkillBehavior } from '../model/skill-behavior.js';
import type { AgentComposition } from '../model/agent.js';
import { generateCopilotSkillMd } from './copilot-skill-generator.js';
import { generateCopilotAgentMd } from './copilot-agent-generator.js';
import { generateCopilotInstructions } from './copilot-instructions-generator.js';
import { registerGenerator } from './target-generator.js';

export function createCopilotGenerator(): TargetGenerator {
  return {
    target: 'copilot',
    defaultOutputDir: '.github',
    generateSkill: (skill: SkillBehavior) => generateCopilotSkillMd(skill),
    generateAgent: (agent: AgentComposition, skills: SkillBehavior[], outputDir: string) =>
      generateCopilotAgentMd(agent, skills, outputDir),
    generateInstructions: (skills: SkillBehavior[], agents: AgentComposition[]) =>
      generateCopilotInstructions(skills, agents),
    getSkillPath: (name: string) => `skills/${name}/SKILL.md`,
    getAgentPath: (name: string) => `agents/${name}.agent.md`,
    getInstructionsPath: () => 'copilot-instructions.md',
  };
}

registerGenerator('copilot', createCopilotGenerator);
