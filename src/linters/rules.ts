import type { SkillBehavior } from '../model/skill-behavior.js';

export type LintSeverity = 'error' | 'warning' | 'info';

export interface LintResult {
  rule: string;
  severity: LintSeverity;
  message: string;
  facet: string;
}

type LintRule = (skill: SkillBehavior) => LintResult | null;

const noEmptyTools: LintRule = (skill) => {
  if (skill.strategy.tools.length === 0) {
    return {
      rule: 'no-empty-tools',
      severity: 'warning',
      message: `Skill "${skill.skill}" has no tools defined. An agent without tools has limited capability.`,
      facet: 'strategy',
    };
  }
  return null;
};

const hasGuardrails: LintRule = (skill) => {
  if (skill.guardrails.length === 0) {
    return {
      rule: 'has-guardrails',
      severity: 'warning',
      message: `Skill "${skill.skill}" has no guardrails. Consider adding limits (timeout, max_tokens, etc.).`,
      facet: 'guardrails',
    };
  }
  return null;
};

const observableOutputs: LintRule = (skill) => {
  if (skill.context.produces.length > 0 && skill.observability.metrics.length === 0) {
    return {
      rule: 'observable-outputs',
      severity: 'warning',
      message: `Skill "${skill.skill}" produces data but has no observability metrics. Add metrics to track output quality.`,
      facet: 'observability',
    };
  }
  return null;
};

const securityNeedsGuardrails: LintRule = (skill) => {
  const hasHighAccess = skill.security.filesystem === 'full' || skill.security.filesystem === 'read-write';
  if (hasHighAccess && skill.guardrails.length === 0) {
    return {
      rule: 'security-needs-guardrails',
      severity: 'error',
      message: `Skill "${skill.skill}" has ${skill.security.filesystem} filesystem access but no guardrails. This is dangerous.`,
      facet: 'security',
    };
  }
  return null;
};

const allRules: LintRule[] = [
  noEmptyTools,
  hasGuardrails,
  observableOutputs,
  securityNeedsGuardrails,
];

export function lintSkill(skill: SkillBehavior): LintResult[] {
  return allRules
    .map((rule) => rule(skill))
    .filter((result): result is LintResult => result !== null);
}
