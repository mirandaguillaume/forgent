import type { SkillBehavior } from '../model/skill-behavior.js';
import type { AgentComposition } from '../model/agent.js';

// -- Skill Score --

export interface SkillScore {
  skill: string;
  total: number;
  breakdown: {
    context: number;
    strategy: number;
    guardrails: number;
    observability: number;
    security: number;
  };
}

const SKILL_WEIGHTS = {
  context: 20,
  strategy: 25,
  guardrails: 20,
  observability: 15,
  security: 20,
};

function scoreContext(skill: SkillBehavior): number {
  let score = 0;
  const max = SKILL_WEIGHTS.context;

  // Has consumes defined (skill knows what it needs)
  if (skill.context.consumes.length > 0) score += max * 0.35;
  // Has produces defined (skill knows what it outputs)
  if (skill.context.produces.length > 0) score += max * 0.35;
  // Uses conversation or long-term memory (richer context)
  if (skill.context.memory === 'conversation') score += max * 0.15;
  if (skill.context.memory === 'long-term') score += max * 0.1;
  // At least short-term
  if (skill.context.memory === 'short-term') score += max * 0.15;

  // Bonus: both consumes and produces = well-defined I/O contract
  if (skill.context.consumes.length > 0 && skill.context.produces.length > 0) {
    score += max * 0.15;
  }

  return Math.min(max, Math.round(score));
}

function scoreStrategy(skill: SkillBehavior): number {
  let score = 0;
  const max = SKILL_WEIGHTS.strategy;

  // Has tools
  if (skill.strategy.tools.length > 0) score += max * 0.35;
  // Has approach defined
  if (skill.strategy.approach && skill.strategy.approach.length > 0) score += max * 0.25;
  // Has steps defined (actionable instructions)
  if (skill.strategy.steps && skill.strategy.steps.length > 0) {
    score += max * 0.25;
    // Bonus for detailed steps (3+)
    if (skill.strategy.steps.length >= 3) score += max * 0.15;
  }

  return Math.min(max, Math.round(score));
}

function scoreGuardrails(skill: SkillBehavior): number {
  if (skill.guardrails.length === 0) return 0;

  let score = 0;
  const max = SKILL_WEIGHTS.guardrails;

  // Has at least one guardrail
  score += max * 0.5;

  // Has timeout (critical for agent safety)
  const hasTimeout = skill.guardrails.some((g) => {
    if (typeof g === 'string') return g.includes('timeout');
    return 'timeout' in g;
  });
  if (hasTimeout) score += max * 0.3;

  // Multiple guardrails = defense in depth
  if (skill.guardrails.length >= 2) score += max * 0.2;

  return Math.min(max, Math.round(score));
}

function scoreObservability(skill: SkillBehavior): number {
  let score = 0;
  const max = SKILL_WEIGHTS.observability;

  // Has metrics
  if (skill.observability.metrics.length > 0) {
    score += max * 0.4;
    if (skill.observability.metrics.length >= 2) score += max * 0.15;
  }

  // Trace level
  switch (skill.observability.trace_level) {
    case 'detailed':
      score += max * 0.45;
      break;
    case 'standard':
      score += max * 0.3;
      break;
    case 'minimal':
      score += max * 0.1;
      break;
  }

  return Math.min(max, Math.round(score));
}

function scoreSecurity(skill: SkillBehavior): number {
  let score = 0;
  const max = SKILL_WEIGHTS.security;

  // Filesystem: more restrictive = higher score
  switch (skill.security.filesystem) {
    case 'none':
      score += max * 0.4;
      break;
    case 'read-only':
      score += max * 0.35;
      break;
    case 'read-write':
      score += max * 0.15;
      break;
    case 'full':
      score += max * 0.05;
      break;
  }

  // Network: more restrictive = higher score
  switch (skill.security.network) {
    case 'none':
      score += max * 0.35;
      break;
    case 'allowlist':
      score += max * 0.2;
      break;
    case 'full':
      score += max * 0.05;
      break;
  }

  // No secrets = better (principle of least privilege)
  if (skill.security.secrets.length === 0) {
    score += max * 0.15;
  } else {
    score += max * 0.05;
  }

  // Sandbox bonus
  if (skill.security.sandbox === 'container' || skill.security.sandbox === 'vm') {
    score += max * 0.1;
  }

  return Math.min(max, Math.round(score));
}

export function scoreSkill(skill: SkillBehavior): SkillScore {
  const breakdown = {
    context: scoreContext(skill),
    strategy: scoreStrategy(skill),
    guardrails: scoreGuardrails(skill),
    observability: scoreObservability(skill),
    security: scoreSecurity(skill),
  };

  const total = breakdown.context + breakdown.strategy + breakdown.guardrails + breakdown.observability + breakdown.security;

  return { skill: skill.skill, total, breakdown };
}

// -- Agent Score --

export interface AgentScore {
  agent: string;
  total: number;
  breakdown: {
    description: number;
    composition: number;
    dataFlow: number;
    orchestration: number;
  };
}

const AGENT_WEIGHTS = {
  description: 20,
  composition: 25,
  dataFlow: 35,
  orchestration: 20,
};

export function scoreAgent(agent: AgentComposition, resolvedSkills: SkillBehavior[]): AgentScore {
  const breakdown = {
    description: scoreDescription(agent),
    composition: scoreComposition(agent),
    dataFlow: scoreDataFlow(agent, resolvedSkills),
    orchestration: scoreOrchestration(agent, resolvedSkills),
  };

  const total = breakdown.description + breakdown.composition + breakdown.dataFlow + breakdown.orchestration;

  return { agent: agent.agent, total, breakdown };
}

function scoreDescription(agent: AgentComposition): number {
  const max = AGENT_WEIGHTS.description;
  if (!agent.description) return 0;

  let score = max * 0.6; // Has description at all

  // Longer description = more informative
  const words = agent.description.split(/\s+/).length;
  if (words >= 5) score += max * 0.2;
  if (words >= 10) score += max * 0.2;

  return Math.min(max, Math.round(score));
}

function scoreComposition(agent: AgentComposition): number {
  const max = AGENT_WEIGHTS.composition;
  let score = 0;

  // At least 2 skills = meaningful composition
  if (agent.skills.length >= 2) {
    score += max * 0.6;
  } else if (agent.skills.length === 1) {
    score += max * 0.3; // Single skill is thin wrapper
  }

  // 3+ skills = rich pipeline
  if (agent.skills.length >= 3) score += max * 0.2;

  // No duplicates
  const unique = new Set(agent.skills);
  if (unique.size === agent.skills.length) score += max * 0.2;

  return Math.min(max, Math.round(score));
}

function scoreDataFlow(agent: AgentComposition, resolvedSkills: SkillBehavior[]): number {
  const max = AGENT_WEIGHTS.dataFlow;

  // Can't evaluate without resolved skills
  if (resolvedSkills.length < 2) return Math.round(max * 0.5);

  // Only relevant for sequential — parallel doesn't have data flow concerns
  if (agent.orchestration !== 'sequential') return max;

  let score = 0;
  const orderedSkills = agent.skills
    .map((name) => resolvedSkills.find((s) => s.skill === name))
    .filter((s): s is SkillBehavior => s !== undefined);

  if (orderedSkills.length < 2) return Math.round(max * 0.5);

  // Collect all items produced by ANY skill in the pipeline
  const allProduced = new Set<string>();
  for (const skill of orderedSkills) {
    for (const item of skill.context.produces) {
      allProduced.add(item);
    }
  }

  // Check: does each inter-skill consumer come after its producer?
  // Items not produced by any skill are environment inputs — don't penalize.
  const producedBefore = new Set<string>();
  let interSkillConsumes = 0;
  let satisfiedConsumes = 0;

  for (const skill of orderedSkills) {
    for (const item of skill.context.consumes) {
      if (!allProduced.has(item)) continue; // environment input, skip
      interSkillConsumes++;
      if (producedBefore.has(item)) {
        satisfiedConsumes++;
      }
    }
    for (const item of skill.context.produces) {
      producedBefore.add(item);
    }
  }

  if (interSkillConsumes === 0) {
    // No inter-skill data flow — skills are independent, give good score
    score = max * 0.8;
  } else {
    // Ratio of satisfied inter-skill data flow
    const ratio = satisfiedConsumes / interSkillConsumes;
    score = max * ratio;
  }

  return Math.min(max, Math.round(score));
}

function scoreOrchestration(agent: AgentComposition, resolvedSkills: SkillBehavior[]): number {
  const max = AGENT_WEIGHTS.orchestration;
  let score = max * 0.5; // Base score for having an orchestration strategy

  // Sequential with proper data flow = well-designed pipeline
  if (agent.orchestration === 'sequential' && resolvedSkills.length >= 2) {
    score += max * 0.3;
  }

  // Has description matching orchestration style
  if (agent.description) {
    score += max * 0.2;
  }

  return Math.min(max, Math.round(score));
}
