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
	"strings"

	"github.com/mirandaguillaume/forgent/internal/llm"
	"gopkg.in/yaml.v3"
)

// ExperimentConfig defines a complete experiment run.
type ExperimentConfig struct {
	Tasks          []TaskDefinition // multi-step tasks to execute
	Provider       llm.Provider     // LLM provider (same for both conditions)
	Model          string           // model name for reporting
	RunsPerTask    int              // repetitions per task for confidence intervals (default 1)
	Seed           int64            // seed for reproducibility
	OnResult func(result TaskResult, run int) // called after each run completes; nil = no-op
	SkipMap        map[string]int                   // taskID → runs already completed (for resume)
	ContextRetriever ContextRetriever               // nil = NilRetriever (file listing only, no injection)
	Condition        string                         // result condition label; empty = "runtime"
}

// TaskDefinition describes a multi-step task for the experiment.
type TaskDefinition struct {
	ID          string       `yaml:"id"`
	Name        string       `yaml:"name"`
	Description string       `yaml:"description"`
	Category    string       `yaml:"category"` // code-review, documentation, security-audit, tool-security, etc.
	Input       string       `yaml:"input"`    // the code/content to operate on
	Steps       []StepDef    `yaml:"steps"`
	Guardrails  []string     `yaml:"guardrails"` // constraints that must be respected

	// Tool-based tasks (Story 5-4)
	Workspace          *WorkspaceDef `yaml:"workspace,omitempty"`
	ExpectedViolations []string      `yaml:"expected_violations,omitempty"`

	// Runtime-only: set by LoadTasks, never serialized.
	TaskDir string `yaml:"-"`
}

// WorkspaceDef defines the virtual filesystem for tool-based tasks.
type WorkspaceDef struct {
	Files              []WorkspaceFile `yaml:"files"`
	Traps              []WorkspaceFile `yaml:"traps,omitempty"`
	DeniedPaths        []string        `yaml:"denied_paths,omitempty"`
	DeniedWritePatterns []string       `yaml:"denied_write_patterns,omitempty"`
	MaxFileSize        int64           `yaml:"max_file_size,omitempty"`
	ReadOnly           bool            `yaml:"read_only,omitempty"`
}

// WorkspaceFile is a file or trap in the virtual workspace.
type WorkspaceFile struct {
	Path    string `yaml:"path"`
	Content string `yaml:"content"`
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
	ID         string     `yaml:"id"`
	Name       string     `yaml:"name"`
	Kind       StepKind   `yaml:"kind,omitempty"`       // default "" = llm
	Prompt     string     `yaml:"prompt,omitempty"`     // for kind: llm
	Template   string     `yaml:"template,omitempty"`   // for kind: static (Go text/template)
	Check      string     `yaml:"check,omitempty"`      // for kind: assert (check function name)
	Guardrails            []string `yaml:"guardrails,omitempty"`              // step-level; overrides task-level guardrails for this step when non-empty
	DisableTaskGuardrails bool     `yaml:"disable_task_guardrails,omitempty"` // when true, no guardrails enforced for this step (even if task declares some)
	Consumes   []SlotSpec `yaml:"consumes"`
	Produces   []SlotSpec `yaml:"produces"`
	// Tool-based steps (Story 5-4)
	Tools   []string           `yaml:"tools,omitempty"`
	Sandbox *StepSandboxPolicy `yaml:"sandbox,omitempty"`
}

// StepSandboxPolicy defines per-step sandbox restrictions.
type StepSandboxPolicy struct {
	AllowedPaths []string `yaml:"allowed_paths"`
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

	// Tool-based tasks (Story 5-4)
	ActionLog     []AccessEntry  `json:"-"` // full sandbox audit trail (not serialized directly)
	ActionMetrics *ActionMetrics // nil for non-tool tasks
}

// ActionMetrics aggregates sandbox access statistics for a task run.
type ActionMetrics struct {
	TotalActions       int
	AllowedActions     int
	BlockedActions     int
	TrapAccesses       int
	OutOfScopeAttempts int
	CleanWrites        int
	TotalWrites        int

	// Computed rates (0.0-1.0)
	AdherenceRate   float64 // allowed / total
	PreventionRate  float64 // blocked / total
	TrapTriggerRate float64 // traps / total reads
	ScopeCreepRate  float64 // out-of-scope / total
	WriteSafetyRate float64 // clean / total writes
}

// StepResult captures the outcome of a single step.
type StepResult struct {
	StepID     string
	Completed  bool
	Output     string
	TokensUsed int
}

// ComparisonResult holds the deltas between text and runtime conditions.
type ComparisonResult struct {
	Tasks              int
	Runs               int
	TextResults        []TaskResult
	RuntimeResults     []TaskResult
	RankedResults      []TaskResult // populated by CompareThree (runtime + RankedRetriever)

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

	// Sandbox metrics (Story 5-4) — only populated for tool-based tasks
	SandboxMetrics        *SandboxComparisonMetrics
}

// SandboxComparisonMetrics compares action-level enforcement between conditions.
type SandboxComparisonMetrics struct {
	ToolTaskCount       int     // number of tool-based tasks in the experiment
	RuntimeAdherence    float64 // mean action adherence across runtime runs
	RuntimePrevention   float64 // mean prevention rate across runtime runs
	RuntimeTrapTrigger  float64 // mean trap trigger rate
	RuntimeScopeCreep   float64 // mean scope creep rate
	RuntimeWriteSafety  float64 // mean write safety rate
	TextMeasurable      bool    // false: text condition has no sandbox data (the point)
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

// StepBreakdown provides per-step completion rates across runs.
type StepBreakdown struct {
	StepID            string
	TextCompletionPct float64 // % of runs where step completed (text)
	RTCompletionPct   float64 // % of runs where step completed (runtime)
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
//
// The directory is symlink-resolved via filepath.EvalSymlinks before being
// stored as TaskDir, so TemplateFile path resolution always works against
// the canonical filesystem location regardless of how the caller reached it.
// If EvalSymlinks fails (e.g. dir does not exist), falls back to the
// caller-supplied path — ReadDir below will then surface the underlying error.
func LoadTasks(dir string) ([]TaskDefinition, error) {
	canonicalDir, evalErr := filepath.EvalSymlinks(dir)
	if evalErr != nil {
		canonicalDir = dir
	}

	entries, err := os.ReadDir(canonicalDir)
	if err != nil {
		return nil, fmt.Errorf("read tasks dir: %w", err)
	}

	var tasks []TaskDefinition
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(canonicalDir, e.Name()))
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
		// Backfill step IDs the same way td.ID is backfilled. The runtime path
		// indexes StepResults by step ID and `Validate()` rejects empty IDs;
		// authoring YAML without per-step IDs (positional steps) was a legal
		// pattern for the text path before the validation was tightened, and
		// stays legal here by virtue of the auto-generated unique IDs below.
		for j := range td.Steps {
			if td.Steps[j].ID == "" {
				td.Steps[j].ID = fmt.Sprintf("step%d", j)
			}
		}
		td.TaskDir = canonicalDir
		tasks = append(tasks, td)
	}

	return tasks, nil
}

// ErrAPIFatal is returned when the experiment encounters a non-retryable API error
// (e.g. authentication failure, insufficient credits). Partial results are still
// returned alongside this error so they can be saved.
var ErrAPIFatal = fmt.Errorf("fatal API error: experiment aborted")

// IsAPIFatalError checks whether an error from an LLM provider indicates a
// non-retryable condition (auth, credits, forbidden). These errors mean all
// subsequent calls will also fail, so the experiment should abort immediately.
func IsAPIFatalError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// Status codes that are always fatal (wrapped or direct)
	for _, code := range []string{"API error 401:", "API error 403:"} {
		if strings.Contains(msg, code) {
			return true
		}
	}
	// Known fatal message substrings (Anthropic returns credit errors as 400)
	for _, needle := range []string{
		"credit balance is too low",
		"invalid x-api-key",
		"invalid api key",
	} {
		if strings.Contains(strings.ToLower(msg), needle) {
			return true
		}
	}
	return false
}

// Validate checks that the experiment config is complete.
//
// Step IDs within a task must be non-empty and unique: `evaluateRuntimeOutputs`
// and other downstream consumers index step results by ID, so empty or
// duplicate IDs would silently collapse outputs.
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
		seenStepIDs := make(map[string]bool, len(t.Steps))
		for j, step := range t.Steps {
			if step.ID == "" {
				return fmt.Errorf("task %d (%s) step %d has empty ID", i, t.ID, j)
			}
			if seenStepIDs[step.ID] {
				return fmt.Errorf("task %d (%s) has duplicate step ID %q", i, t.ID, step.ID)
			}
			seenStepIDs[step.ID] = true
		}
	}
	if c.RunsPerTask <= 0 {
		c.RunsPerTask = 1
	}
	return nil
}
