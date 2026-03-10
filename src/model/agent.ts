import type { SkillBehavior } from './skill-behavior.js';

export type OrchestrationStrategy =
  | 'sequential'
  | 'parallel'
  | 'parallel-then-merge'
  | 'adaptive';

export interface AgentComposition {
  agent: string;
  skills: string[];
  orchestration: OrchestrationStrategy;
  description?: string;
}

export interface ResolvedAgent {
  agent: string;
  skills: SkillBehavior[];
  orchestration: OrchestrationStrategy;
}
