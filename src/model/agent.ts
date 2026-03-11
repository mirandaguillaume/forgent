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
