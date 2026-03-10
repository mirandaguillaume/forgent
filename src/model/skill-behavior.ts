export type MemoryType = 'short-term' | 'conversation' | 'long-term';

export interface ContextFacet {
  consumes: string[];
  produces: string[];
  memory: MemoryType;
}

export interface StrategyFacet {
  tools: string[];
  approach: string;
  steps?: string[];
}

export type GuardrailRule = string | Record<string, string | number>;

export type GuardrailsFacet = GuardrailRule[];

export interface Dependency {
  skill: string;
  provides: string;
}

export type DependencyFacet = Dependency[];

export type TraceLevel = 'minimal' | 'standard' | 'detailed';

export interface ObservabilityFacet {
  trace_level: TraceLevel;
  metrics: string[];
}

export type AccessLevel = 'none' | 'read-only' | 'read-write' | 'full';
export type NetworkAccess = 'none' | 'allowlist' | 'full';

export interface SecurityFacet {
  filesystem: AccessLevel;
  network: NetworkAccess;
  secrets: string[];
  sandbox?: 'none' | 'container' | 'vm';
}

export type NegotiationStrategy = 'yield' | 'override' | 'merge';

export interface NegotiationFacet {
  file_conflicts: NegotiationStrategy;
  priority: number;
}

export interface SkillBehavior {
  skill: string;
  version: string;
  context: ContextFacet;
  strategy: StrategyFacet;
  guardrails: GuardrailsFacet;
  depends_on: DependencyFacet;
  observability: ObservabilityFacet;
  security: SecurityFacet;
  negotiation: NegotiationFacet;
}
