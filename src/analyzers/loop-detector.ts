import type { SkillBehavior } from '../model/skill-behavior.js';

export type LoopRiskType = 'self-reference' | 'no-timeout' | 'unbounded-retry';

export interface LoopRisk {
  type: LoopRiskType;
  skill: string;
  message: string;
  severity: 'warning' | 'error';
}

function hasTimeoutGuardrail(skill: SkillBehavior): boolean {
  return skill.guardrails.some((g) => {
    if (typeof g === 'string') return g.toLowerCase().includes('timeout');
    return 'timeout' in g;
  });
}

export function detectLoopRisks(skill: SkillBehavior): LoopRisk[] {
  const risks: LoopRisk[] = [];

  // Self-reference: consumes and produces the same data
  const overlap = skill.context.consumes.filter((c) =>
    skill.context.produces.includes(c),
  );
  if (overlap.length > 0) {
    risks.push({
      type: 'self-reference',
      skill: skill.skill,
      message: `Skill consumes and produces the same data: [${overlap.join(', ')}]. This can cause infinite loops.`,
      severity: 'error',
    });
  }

  // No timeout with persistent memory
  if (skill.context.memory !== 'short-term' && !hasTimeoutGuardrail(skill)) {
    risks.push({
      type: 'no-timeout',
      skill: skill.skill,
      message: `Skill uses ${skill.context.memory} memory but has no timeout guardrail. Risk of unbounded execution.`,
      severity: 'warning',
    });
  }

  return risks;
}
