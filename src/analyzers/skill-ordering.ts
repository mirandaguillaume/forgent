import type { AgentComposition } from '../model/agent.js';
import type { SkillBehavior } from '../model/skill-behavior.js';

export type OrderingIssueType = 'description-order-mismatch' | 'data-flow-order-mismatch';

export interface OrderingIssue {
  type: OrderingIssueType;
  agent: string;
  message: string;
  severity: 'warning';
}

export function checkSkillOrdering(
  agent: AgentComposition,
  skillMap: Map<string, SkillBehavior>,
): OrderingIssue[] {
  if (agent.orchestration !== 'sequential') return [];

  const issues: OrderingIssue[] = [];
  issues.push(...checkDescriptionOrder(agent));
  issues.push(...checkDataFlowOrder(agent, skillMap));
  return issues;
}

/**
 * Checks if the agent's description mentions skills in a different order
 * than the skills[] array. Matches skill names with hyphens replaced by spaces too.
 */
function checkDescriptionOrder(agent: AgentComposition): OrderingIssue[] {
  if (!agent.description) return [];

  const desc = agent.description.toLowerCase();

  // Find position of each skill name in the description
  const positions: { skill: string; index: number; arrayPos: number }[] = [];

  for (let i = 0; i < agent.skills.length; i++) {
    const skillName = agent.skills[i];
    const variants = [skillName.toLowerCase(), skillName.toLowerCase().replace(/-/g, ' ')];

    let earliest = -1;
    for (const variant of variants) {
      const pos = desc.indexOf(variant);
      if (pos !== -1 && (earliest === -1 || pos < earliest)) {
        earliest = pos;
      }
    }

    if (earliest !== -1) {
      positions.push({ skill: skillName, index: earliest, arrayPos: i });
    }
  }

  // Need at least 2 skills mentioned to detect mismatch
  if (positions.length < 2) return [];

  // Sort by position in description
  const descOrder = [...positions].sort((a, b) => a.index - b.index);

  // Check if description order matches array order
  const isOrdered = descOrder.every((item, idx) => {
    if (idx === 0) return true;
    return item.arrayPos > descOrder[idx - 1].arrayPos;
  });

  if (!isOrdered) {
    const descNames = descOrder.map((p) => p.skill).join(' → ');
    const arrayNames = positions.sort((a, b) => a.arrayPos - b.arrayPos).map((p) => p.skill).join(' → ');
    return [{
      type: 'description-order-mismatch',
      agent: agent.agent,
      message: `Skill ordering mismatch: description suggests [${descNames}] but skills array is [${arrayNames}]`,
      severity: 'warning',
    }];
  }

  return [];
}

/**
 * Checks if a skill consumes data that is produced by a skill appearing later
 * in the array. In sequential mode, later skills run after earlier ones,
 * so a consumer must come after its producer.
 */
function checkDataFlowOrder(agent: AgentComposition, skillMap: Map<string, SkillBehavior>): OrderingIssue[] {
  const issues: OrderingIssue[] = [];

  // Build produces map: dataItem → arrayIndex of the skill that produces it
  const producesMap = new Map<string, { skillName: string; arrayIndex: number }>();

  for (let i = 0; i < agent.skills.length; i++) {
    const skill = skillMap.get(agent.skills[i]);
    if (!skill) continue;

    for (const item of skill.context.produces) {
      producesMap.set(item, { skillName: agent.skills[i], arrayIndex: i });
    }
  }

  // For each skill, check if it consumes something produced by a later skill
  for (let i = 0; i < agent.skills.length; i++) {
    const skill = skillMap.get(agent.skills[i]);
    if (!skill) continue;

    for (const item of skill.context.consumes) {
      const producer = producesMap.get(item);
      if (producer && producer.arrayIndex > i) {
        issues.push({
          type: 'data-flow-order-mismatch',
          agent: agent.agent,
          message: `"${agent.skills[i]}" consumes "${item}" but it is produced by "${producer.skillName}" which runs later (index ${producer.arrayIndex} > ${i})`,
          severity: 'warning',
        });
      }
    }
  }

  return issues;
}
