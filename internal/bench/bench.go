package bench

import "time"

// ProxyResult holds metrics from the proxy reachability benchmark.
type ProxyResult struct {
	TotalSourceFiles int
	SampledFiles     int
	ReachableFiles   int
	Reachability     float64 // percentage (0-100)
	IndexEntries     int
	IndexBytes       int
}

// Task describes a navigation task for the agent benchmark.
type Task struct {
	Query         string   `yaml:"query"`
	ExpectedPaths []string `yaml:"expected_paths"`
}

// TaskResult holds the outcome of a single agent task.
type TaskResult struct {
	Query    string
	Response string
	Hit      bool
	Err      error
	Latency  time.Duration
}

// AgentResult holds aggregate metrics from the agent navigation benchmark.
type AgentResult struct {
	Tasks       int
	Hits        int
	Misses      int
	Errors      int
	HitRate     float64 // percentage (0-100)
	AvgLatency  time.Duration
	TotalCost   float64 // total USD across all tasks
	TotalTokens int     // total input tokens across all tasks
	Details     []TaskResult
}
