import type { SkillBehavior } from '../model/skill-behavior.js';
import type { AgentComposition } from '../model/agent.js';
import { toTitle } from '../utils/to-title.js';

function formatGuardrail(g: string | Record<string, string | number>): string {
  if (typeof g === 'string') return `- ${g}`;
  return Object.entries(g)
    .map(([k, v]) => `- ${k}: ${v}`)
    .join('\n');
}

/**
 * Generates a copilot-instructions.md file that aggregates project-level
 * information from all skills and agents.
 *
 * Returns null when there are no skills and no agents.
 */
export function generateCopilotInstructions(
  skills: SkillBehavior[],
  agents: AgentComposition[],
): string | null {
  if (skills.length === 0 && agents.length === 0) {
    return null;
  }

  const lines: string[] = [];

  lines.push('# Project Instructions');
  lines.push('');

  // Available Skills
  if (skills.length > 0) {
    lines.push('## Available Skills');
    lines.push('');
    for (const skill of skills) {
      const desc = buildSkillDescription(skill);
      lines.push(`- **${toTitle(skill.skill)}**: ${desc}`);
    }
    lines.push('');
  }

  // Available Agents
  if (agents.length > 0) {
    lines.push('## Available Agents');
    lines.push('');
    for (const agent of agents) {
      const desc = agent.description || `${agent.orchestration} agent with ${agent.skills.length} skills`;
      lines.push(`- **${toTitle(agent.agent)}**: ${desc}`);
    }
    lines.push('');
  }

  // Global Guardrails — aggregated from all skills
  const allGuardrails = skills.flatMap((s) => s.guardrails);
  if (allGuardrails.length > 0) {
    lines.push('## Global Guardrails');
    lines.push('');
    for (const g of allGuardrails) {
      lines.push(formatGuardrail(g));
    }
    lines.push('');
  }

  return lines.join('\n');
}

function buildSkillDescription(skill: SkillBehavior): string {
  const parts: string[] = [];
  parts.push(`${skill.strategy.approach}-based skill`);
  if (skill.context.consumes.length > 0) {
    parts.push(`consuming ${skill.context.consumes.join(', ')}`);
  }
  if (skill.context.produces.length > 0) {
    parts.push(`to produce ${skill.context.produces.join(', ')}`);
  }
  return parts.join(' ');
}
