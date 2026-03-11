package model

// OrchestrationStrategy defines how skills are orchestrated within an agent.
type OrchestrationStrategy string

const (
	OrchestrationSequential       OrchestrationStrategy = "sequential"
	OrchestrationParallel         OrchestrationStrategy = "parallel"
	OrchestrationParallelThenMerge OrchestrationStrategy = "parallel-then-merge"
	OrchestrationAdaptive         OrchestrationStrategy = "adaptive"
)

// AgentComposition defines an agent as a composition of skills.
type AgentComposition struct {
	Agent         string                `yaml:"agent"`
	Skills        []string              `yaml:"skills"`
	Orchestration OrchestrationStrategy `yaml:"orchestration"`
	Description   string                `yaml:"description,omitempty"`
}
