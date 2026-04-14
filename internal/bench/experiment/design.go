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

// StepKind discriminates how a step is executed in the runtime.
type StepKind string

const (
	StepKindLLM    StepKind = ""       // default: LLM call
	StepKindStatic StepKind = "static" // deterministic template
	StepKindAssert StepKind = "assert" // verification check
)

// StepDef defines a single step within a multi-step task.
type StepDef struct {
	ID       string   `yaml:"id"`
	Name     string   `yaml:"name"`
	Kind     StepKind `yaml:"kind,omitempty"`     // default "" = llm
	Prompt   string   `yaml:"prompt,omitempty"`   // for kind: llm
	Template string   `yaml:"template,omitempty"` // for kind: static (Go text/template)
	Check    string   `yaml:"check,omitempty"`    // for kind: assert (check function name)
	Consumes []string `yaml:"consumes"`
	Produces []string `yaml:"produces"`
	ExpectedKeywords []string `yaml:"expected_keywords"`
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

	// Harness detection metrics
	// "How many text failures would the runtime harness have caught?"
	DetectionRate         DetectionMetrics

	// Derived metrics
	Efficiency            EfficiencyMetrics
	StepBreakdowns        []StepBreakdown
	Stability             []StabilityMetrics
	CascadeRate           CascadeMetrics
	DifficultyGradient    DifficultyMetrics
	Convergence           ConvergenceMetrics
	FailureCorrelation    CorrelationMetrics
	FirstFailureSteps     []FirstFailureEntry
	OutputLengthCV        OutputLengthMetrics
	GuardrailSeverity     GuardrailSeverityMetrics
}

// DetectionMetrics measures how many soft-harness (text) failures are caught
// by the hard-harness (runtime). This is the core harness metric:
// not "does the harness improve results?" but "does the harness catch failures?"
type DetectionMetrics struct {
	// Completion: runs where text failed to complete but runtime succeeded
	CompletionFailures int     // total text completion failures
	CompletionCaught   int     // of those, how many runtime completed successfully
	CompletionRate     float64 // caught / failures (0 if no failures)

	// Ordering: runs where text had imperfect ordering but runtime had perfect
	OrderingFailures   int
	OrderingCaught     int
	OrderingRate       float64

	// Guardrails: runs where text violated guardrails but runtime respected them
	GuardrailFailures  int
	GuardrailCaught    int
	GuardrailRate      float64

	// Combined: runs with ANY text failure that runtime would have caught
	AnyFailures        int
	AnyCaught          int
	AnyRate            float64
}

// EfficiencyMetrics measures quality-per-token for each condition.
type EfficiencyMetrics struct {
	TextQuality       float64 // weighted quality score (0-1)
	RuntimeQuality    float64
	TextTokens        float64 // mean tokens per run
	RuntimeTokens     float64
	TextEfficiency    float64 // quality / (tokens/1000)
	RuntimeEfficiency float64
	CostPerDetection  float64 // extra tokens spent per failure detected
}

// StepBreakdown provides per-step completion and keyword rates across runs.
type StepBreakdown struct {
	StepID            string
	TextCompletionPct float64 // % of runs where step completed (text)
	RTCompletionPct   float64 // % of runs where step completed (runtime)
	TextKeywordPct    float64 // mean(keywords_hit/keywords_total) (text)
	RTKeywordPct      float64 // mean(keywords_hit/keywords_total) (runtime)
	Runs              int     // number of runs sampled
}

// StabilityMetrics measures variance across runs for each condition × task.
type StabilityMetrics struct {
	TaskID             string
	TextCompletionStd  float64 // std dev of completion (0/1) across runs
	RTCompletionStd    float64
	TextOrderingStd    float64
	RTOrderingStd      float64
	TextGuardrailStd   float64
	RTGuardrailStd     float64
}

// CascadeMetrics measures failure propagation: when a step fails, do subsequent steps also fail?
type CascadeMetrics struct {
	TextRunsWithFailure    int
	TextRunsWithCascade    int     // runs where 2+ consecutive steps failed
	TextCascadeRate        float64 // cascade / failure
	RuntimeRunsWithFailure int
	RuntimeRunsWithCascade int
	RuntimeCascadeRate     float64
}

// DifficultyMetrics measures whether the runtime advantage grows with task complexity.
type DifficultyMetrics struct {
	Points        []DifficultyPoint
	SpearmanComp  float64 // Spearman rank correlation: n_steps vs completion delta
	SpearmanOrder float64 // Spearman rank correlation: n_steps vs ordering delta
}

// DifficultyPoint is one task's complexity vs delta.
type DifficultyPoint struct {
	TaskID          string
	NumSteps        int
	CompletionDelta float64
	OrderingDelta   float64
}

// ConvergenceMetrics measures how many runs are needed for stable means.
type ConvergenceMetrics struct {
	TextConvergence    int // runs needed for text completion mean to stabilize (±1%)
	RuntimeConvergence int
}

// CorrelationMetrics measures whether failure modes are correlated across runs.
type CorrelationMetrics struct {
	CompletionOrdering  float64 // Pearson r between completion and ordering failure indicators
	CompletionGuardrail float64
	OrderingGuardrail   float64
}

// FirstFailureEntry tracks which step fails first in failed runs.
type FirstFailureEntry struct {
	StepID       string
	TextCount    int
	RuntimeCount int
}

// OutputLengthMetrics measures output length consistency via coefficient of variation.
type OutputLengthMetrics struct {
	TextMeanCV    float64 // mean CV across steps (text)
	RuntimeMeanCV float64
}

// GuardrailSeverityMetrics measures how badly guardrails are violated.
type GuardrailSeverityMetrics struct {
	TextMeanViolations    float64 // mean violations per failed run
	RuntimeMeanViolations float64
	TextMaxViolations     int
	RuntimeMaxViolations  int
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
