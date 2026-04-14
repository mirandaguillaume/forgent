// Package experiment implements a controlled comparison between text-instruction
// agents and Forgent runtime (DAG) agents on identical multi-step tasks.
//
// The experiment measures: task completion rate, step ordering accuracy,
// guardrail adherence, and token usage. The only variable between conditions
// is the orchestration method — same LLM, same tools, same task definitions.
package experiment

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mirandaguillaume/forgent/internal/llm"
	"gopkg.in/yaml.v3"
)

// ExperimentConfig defines a complete experiment run.
type ExperimentConfig struct {
	Tasks       []TaskDefinition // multi-step tasks to execute
	Provider    llm.Provider     // LLM provider (same for both conditions)
	Model       string           // model name for reporting
	RunsPerTask int              // repetitions per task for confidence intervals (default 1)
	Seed        int64            // seed for reproducibility
	OnResult    func(result TaskResult, run int) // called after each run completes; nil = no-op
	SkipMap     map[string]int                   // taskID → runs already completed (for resume)
}

// TaskDefinition describes a multi-step task for the experiment.
type TaskDefinition struct {
	ID          string       `yaml:"id"`
	Name        string       `yaml:"name"`
	Description string       `yaml:"description"`
	Category    string       `yaml:"category"` // code-review, documentation, security-audit, etc.
	Input       string       `yaml:"input"`    // the code/content to operate on
	Steps       []StepDef    `yaml:"steps"`
	Guardrails  []string     `yaml:"guardrails"` // constraints that must be respected
}

// StepDef defines a single step within a multi-step task.
type StepDef struct {
	ID       string   `yaml:"id"`
	Name     string   `yaml:"name"`
	Prompt   string   `yaml:"prompt"`   // instruction for this step
	Consumes []string `yaml:"consumes"` // data keys this step needs (for DAG wiring)
	Produces []string `yaml:"produces"` // data keys this step outputs
	ExpectedKeywords []string `yaml:"expected_keywords"` // keywords to check in output
}

// TaskResult captures the outcome of executing a task under one condition.
type TaskResult struct {
	TaskID          string
	Condition       string // "text" or "runtime"
	Completed       bool   // all steps produced output
	StepResults     []StepResult
	OrderingScore   float64 // 0.0-1.0: fraction of steps in correct order
	GuardrailScore  float64 // 0.0-1.0: fraction of guardrails respected
	TokensUsed      int     // total tokens consumed (prompt + completion estimate)
	WallClockMs     int64   // wall-clock time in milliseconds
	LLMCalls        int     // number of LLM provider calls made
	Error           error
}

// StepResult captures the outcome of a single step.
type StepResult struct {
	StepID       string
	Completed    bool
	Output       string
	KeywordsHit  int // how many expected keywords appeared in output
	KeywordsTotal int
	TokensUsed   int
}

// ComparisonResult holds the deltas between text and runtime conditions.
type ComparisonResult struct {
	Tasks              int
	Runs               int
	TextResults        []TaskResult
	RuntimeResults     []TaskResult

	// Accuracy metrics (null result expected on strong models)
	CompletionDelta    float64 // runtime - text (positive = runtime better)
	OrderingDelta      float64
	GuardrailDelta     float64
	TokenDelta         float64 // runtime - text (negative = runtime more efficient)
	CompletionCI       [2]float64 // 95% confidence interval
	OrderingCI         [2]float64
	GuardrailCI        [2]float64
	TokenCI            [2]float64

	// Engineering metrics (structural advantages)
	WallClockDelta     float64    // ms: runtime - text (negative = runtime faster)
	WallClockCI        [2]float64
	LLMCallsDelta      float64    // runtime - text (positive = runtime makes more calls)
	FaultToleranceText    float64 // % of runs that succeed with error injection (text)
	FaultToleranceRuntime float64 // % of runs that succeed with error injection (runtime)
	CrossContamination    float64 // 0.0-1.0: similarity between parallel step outputs in text mode
}

// LoadTasks reads task definitions from YAML files in a directory.
func LoadTasks(dir string) ([]TaskDefinition, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read tasks dir: %w", err)
	}

	var tasks []TaskDefinition
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", e.Name(), err)
		}

		var td TaskDefinition
		if err := yaml.Unmarshal(data, &td); err != nil {
			return nil, fmt.Errorf("parse %s: %w", e.Name(), err)
		}

		if td.ID == "" {
			td.ID = e.Name()
		}
		tasks = append(tasks, td)
	}

	return tasks, nil
}

// Validate checks that the experiment config is complete.
func (c *ExperimentConfig) Validate() error {
	if len(c.Tasks) == 0 {
		return fmt.Errorf("experiment requires at least one task")
	}
	if c.Provider == nil {
		return fmt.Errorf("experiment requires an LLM provider")
	}
	for i, t := range c.Tasks {
		if len(t.Steps) == 0 {
			return fmt.Errorf("task %d (%s) has no steps", i, t.ID)
		}
	}
	if c.RunsPerTask <= 0 {
		c.RunsPerTask = 1
	}
	return nil
}
