// Package experiment implements a controlled comparison between text-instruction
// agents and Forgent runtime (DAG) agents on identical multi-step tasks.
//
// The experiment measures: task completion rate, step ordering accuracy,
// guardrail adherence, token usage, and (from Epic 10) per-node contract
// compliance with structural enforcement (file access, network policy, output
// contracts, loop bounds).
//
// Primary evidence for FR45 (node-level behavioral contracts improve agent
// behavior) is documented in:
//   _bmad-output/implementation-artifacts/epic-10-experiment-results-*.md
//
// To reproduce: see the "Reproduction Command" section in that file.
// The only variable between conditions is the orchestration method —
// same LLM, same model, same task definitions.
package experiment

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mirandaguillaume/forgent/internal/llm"
	"github.com/mirandaguillaume/forgent/pkg/sandbox"
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
	// StepModelOverride returns a Provider configured for the given model.
	// Called when a StepDef declares a model override. nil = no override support.
	StepModelOverride func(model string) (llm.Provider, error)
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
	// ToolNodes are full node definitions (with prompt, model, consumes, etc.) that
	// are NOT scheduled by the DAG. They are callable on-demand by steps that declare
	// agent_tools. Each tool_node is executed inline (not concurrently with the DAG)
	// when the parent step emits a tool_use call naming the node's ID.
	ToolNodes []StepDef `yaml:"tool_nodes,omitempty"`

	// Tool-based tasks (Story 5-4)
	Workspace          *WorkspaceDef `yaml:"workspace,omitempty"`
	ExpectedViolations []string      `yaml:"expected_violations,omitempty"`

	// Ground truth for content quality measurement.
	// Each entry is a known issue in the task's input; the evaluator checks
	// whether the final review output contains at least one keyword from match_any.
	// Recall = found / total. Zero entries → RecallScore not set (0.0 sentinel).
	ExpectedIssues []ExpectedIssue `yaml:"expected_issues,omitempty"`

	// Runtime-only: set by LoadTasks/LoadCorpus, never serialized.
	TaskDir string            `yaml:"-"`
	Env     map[string]string `yaml:"-"` // extra env vars injected into kind: shell steps
}

// ExpectedIssue is a known ground-truth defect used to measure reviewer recall.
// An issue is considered "found" if at least one string in MatchAny appears
// (case-insensitive) anywhere in the combined final output of the workflow.
type ExpectedIssue struct {
	ID          string   `yaml:"id"`
	Description string   `yaml:"description"`
	MatchAny    []string `yaml:"match_any"` // at least one must appear in final output
}

// WorkspaceDef defines the virtual filesystem for tool-based tasks.
type WorkspaceDef struct {
	Files               []WorkspaceFile `yaml:"files"`
	Traps               []WorkspaceFile `yaml:"traps,omitempty"`
	DeniedPaths         []string        `yaml:"denied_paths,omitempty"`
	DeniedWritePatterns []string        `yaml:"denied_write_patterns,omitempty"`
	MaxFileSize         int64           `yaml:"max_file_size,omitempty"`
	ReadOnly            bool            `yaml:"read_only,omitempty"`
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
	// StepKindMap maps an LLM call over each item in a []any input slot (parallel).
	// The DAG engine calls the step's runFn once per item concurrently; results are
	// collected into a new []any output slot. Requires the consumed slot to carry a
	// []any value (produced by a step with list_schema or another kind: map step).
	StepKindMap StepKind = "map"
	// StepKindFilter filters a []any input slot.
	// When Predicate is set, the item is evaluated deterministically (no LLM call).
	// The runFn returns {"__keep": bool}; the DAG engine retains only items where
	// __keep == true. Requires a []any input slot (same as kind: map).
	StepKindFilter StepKind = "filter"
	// StepKindConcat merges multiple []any input slots into a single []any output slot.
	// Deterministic — no LLM call. Each consumed slot value must be []any (from a
	// list_schema produces step) or a JSON array string. When Deduplicate is true,
	// duplicate items (by fmt.Sprintf representation) are removed from the merged list.
	// Requires exactly one produces slot; produces[0].Key receives the merged []any.
	StepKindConcat StepKind = "concat"
	// StepKindShell runs a bash command. No LLM call. stdout is written to the
	// sandbox workspace file named by the Writes field (if set) and also placed
	// into produces[0] as a string slot for downstream consumption.
	// The command inherits os.Environ() plus any task-level Env vars (e.g.
	// CORPUS_OWNER, CORPUS_REPO, CORPUS_PR_NUM set by LoadCorpus).
	// cmd.Dir is set to TaskDef.TaskDir so relative script paths resolve correctly.
	StepKindShell StepKind = "shell"
	// StepKindLoop implements the Reflector pattern: a body LLM answers a question,
	// a router LLM evaluates the answer and decides CONTINUE or exit (exit_when).
	// The loop runs up to MaxIterations times. Router feedback is injected into the
	// next body prompt. The last body response is mapped to the step's produces slots.
	// Body and router may each declare a model override (independent of the step-level
	// Model field). No DAG back-edges — the loop is entirely internal to the runFn.
	StepKindLoop StepKind = "loop"
)

// LoopBodyDef defines the body LLM of a kind: loop step (Reflector pattern).
// The body is called once per iteration; its output accumulates across iterations.
type LoopBodyDef struct {
	Model  string `yaml:"model,omitempty"` // model override for the body LLM
	Prompt string `yaml:"prompt"`
}

// LoopRouterDef defines the router LLM of a kind: loop step (Reflector pattern).
// The router evaluates each body output and decides whether to continue or exit.
// ExitWhen is a case-insensitive substring checked against the router response.
type LoopRouterDef struct {
	Model    string `yaml:"model,omitempty"` // model override for the router LLM (typically cheaper)
	Prompt   string `yaml:"prompt"`
	ExitWhen string `yaml:"exit_when"` // e.g. "DONE" — loop exits when router response contains this
}

// FileAccess is the per-step file-access permission level.
// Idiomatic Go enum pattern (type-aliased string + typed constants), parallel
// to StepKind above.
//
// Three values:
//
//   - FileAccessReadOnly: only file_read tool calls are permitted; file_write
//     and file_delete are blocked at the tool dispatcher with a stable
//     denial reason.
//   - FileAccessReadWrite: file_read and file_write are permitted; file_delete
//     is blocked. This is the default when the field is empty (preserves
//     NFR40 backward compatibility for tasks authored before story 10-3).
//   - FileAccessReadWriteDelete: all three operations are permitted, subject
//     to the existing path glob policy and workspace-level ReadOnly.
//
// Empty value (the YAML field was omitted) is also valid; effectiveFileAccess
// maps it to FileAccessDefault.
type FileAccess string

const (
	FileAccessReadOnly        FileAccess = "read-only"
	FileAccessReadWrite       FileAccess = "read-write"
	FileAccessReadWriteDelete FileAccess = "read-write-delete"
	FileAccessDefault         FileAccess = FileAccessReadWrite
)

// validFileAccessValues is the closed set used by Validate to reject typos.
// Empty string is also valid (treated as default downstream).
var validFileAccessValues = map[FileAccess]bool{
	"":                        true, // empty = default
	FileAccessReadOnly:        true,
	FileAccessReadWrite:       true,
	FileAccessReadWriteDelete: true,
}

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
	// AgentTools lists tool_node IDs callable by this step via CompleteWithTools.
	// Each referenced ID must appear in TaskDefinition.ToolNodes.
	// The runtime builds an llm.Tool for each, whose Run executes the node inline.
	AgentTools []string `yaml:"agent_tools,omitempty"`
	Sandbox *StepSandboxPolicy `yaml:"sandbox,omitempty"`
	// File-level permissions (Story 10-3). One of FileAccessReadOnly,
	// FileAccessReadWrite, FileAccessReadWriteDelete. Empty = FileAccessDefault
	// (= FileAccessReadWrite). The check is enforced at the tool dispatcher
	// (ExecuteToolForStep, layer 1) before the existing per-step AllowedPaths
	// glob check (layer 2) and the workspace-level SandboxPolicy (layer 3).
	// Most-restrictive-wins composition: any layer can deny independently;
	// only the highest-precedence deny is observable in the audit log because
	// layer 1 short-circuits before layer 2 evaluates.
	FileAccess FileAccess `yaml:"file_access,omitempty"`
	// NetworkPolicy declares domain-level network access rules for this step (Story 10-5).
	// nil means no proxy is started; the step has unrestricted network access (NFR40 compat).
	// Non-nil starts an HTTP proxy on an ephemeral port. HTTP_PROXY/HTTPS_PROXY are set to
	// the proxy URL for the duration of the step runFn, then restored.
	NetworkPolicy *sandbox.NetworkPolicy `yaml:"network_policy,omitempty"`
	// Loop declares the iteration contract for the step's tool-call loop (Story 10-7).
	// nil → implicit default: DefaultMaxToolIterations iterations, no early exit (NFR40).
	// Non-nil → declarative: the declared MaxIterations replaces the default; ExitWhen
	// criteria (OR-combined) allow early exit before MaxIterations is exhausted.
	Loop *LoopPolicy `yaml:"loop,omitempty"`
	// MaxLLMCalls enforces the Single Responsibility principle at the LLM call level.
	// 0 = no limit (default). Set to 1 for analysis nodes that must produce their
	// output in a single inference — no sub-loops, no re-prompting, no internal
	// reasoning chains that generate additional API calls. If the step exceeds this
	// budget, the harness records a ViolationTypeCallBudgetExceeded event and the
	// step fails (Prevented=true). Parallel execution of independent nodes is
	// guaranteed structurally by the DAG engine — this field only governs the
	// internal call budget of each individual node.
	MaxLLMCalls int `yaml:"max_llm_calls,omitempty"`
	// Model overrides the global provider model for this step only.
	// When set, the runtime creates a new provider instance with this model.
	// Requires ExperimentConfig.StepModelOverride to be configured.
	Model string `yaml:"model,omitempty"`
	// Shell is the bash command (or script path relative to TaskDir) for kind: shell steps.
	Shell string `yaml:"shell,omitempty"`
	// Writes is the workspace filename that receives stdout for kind: shell steps.
	// The file is written into the sandbox so downstream steps can file_read it.
	// Optional: if empty, stdout is only available as a string slot via produces[0].
	Writes string `yaml:"writes,omitempty"`
	// Predicate is a deterministic expression for kind: filter steps.
	// When set, the filter does NOT call the LLM — the item is evaluated inline.
	// Syntax: "score >= N" (extracts last "SCORE: N" from item text and compares).
	// Mutually exclusive with using kind: filter as an LLM-scored filter.
	Predicate string `yaml:"predicate,omitempty"`
	// Deduplicate removes duplicate items from the merged output for kind: concat steps.
	// Items are compared by their fmt.Sprintf("%v", item) representation.
	Deduplicate bool `yaml:"deduplicate,omitempty"`
	// For kind: loop steps (Reflector pattern).
	// MaxIterations caps the loop (default 4 when unset). Body is the generator LLM;
	// LoopRouter is the exit-condition checker (typically a cheaper model).
	MaxIterations int            `yaml:"max_iterations,omitempty"`
	Body          *LoopBodyDef   `yaml:"body,omitempty"`
	LoopRouter    *LoopRouterDef `yaml:"router,omitempty"`
	// ParallelGroupID is set during YAML loading when this step belongs to a
	// parallel: block. Steps sharing the same ParallelGroupID must execute
	// concurrently — the DAG engine enforces this structurally via auto-wiring
	// of independent nodes (same consumes, disjoint produces). This field is
	// metadata only: it enables harness-side validation that the declared group
	// is actually independent.
	ParallelGroupID string `yaml:"-"`
}

// StepSandboxPolicy defines per-step sandbox restrictions.
type StepSandboxPolicy struct {
	AllowedPaths []string `yaml:"allowed_paths"`
}

// ExitCriteria declares when a loop step should exit early (OR-combined).
// At least one field must be non-empty if an early-exit is desired; an absent
// ExitCriteria means the loop runs until max_iterations (pure safety backstop).
type ExitCriteria struct {
	OutputMarker string `yaml:"output_marker,omitempty"` // case-sensitive substring in LLM response
	ToolCalled   string `yaml:"tool_called,omitempty"`   // tool name emitted by LLM triggers exit

	// unknownKeys captures YAML keys that don't map to any declared field.
	// Populated by UnmarshalYAML; checked by validateLoopPolicy.
	unknownKeys []string
}

// UnmarshalYAML captures unknown fields in ExitCriteria for strict validation.
func (ec *ExitCriteria) UnmarshalYAML(value *yaml.Node) error {
	type Alias ExitCriteria
	var alias Alias
	if err := value.Decode(&alias); err != nil {
		return err
	}
	*ec = ExitCriteria(alias)
	known := map[string]bool{"output_marker": true, "tool_called": true}
	for i := 0; i+1 < len(value.Content); i += 2 {
		if key := value.Content[i].Value; !known[key] {
			ec.unknownKeys = append(ec.unknownKeys, key)
		}
	}
	return nil
}

// LoopPolicy declares the iteration contract for a step's tool-call loop.
// When nil (the default), the implicit safety backstop (DefaultMaxToolIterations)
// applies with no early exit — preserving NFR40 backward compatibility.
type LoopPolicy struct {
	MaxIterations int         `yaml:"max_iterations,omitempty"` // must be > 0 when Loop is non-nil
	ExitWhen      ExitCriteria `yaml:"exit_when,omitempty"`

	unknownKeys []string
}

// UnmarshalYAML captures unknown fields in LoopPolicy for strict validation.
func (lp *LoopPolicy) UnmarshalYAML(value *yaml.Node) error {
	type Alias LoopPolicy
	var alias Alias
	if err := value.Decode(&alias); err != nil {
		return err
	}
	*lp = LoopPolicy(alias)
	known := map[string]bool{"max_iterations": true, "exit_when": true}
	for i := 0; i+1 < len(value.Content); i += 2 {
		if key := value.Content[i].Value; !known[key] {
			lp.unknownKeys = append(lp.unknownKeys, key)
		}
	}
	return nil
}

// DefaultMaxToolIterations is the implicit tool-call loop backstop used when
// a step has no explicit LoopPolicy. Exported for test assertions.
const DefaultMaxToolIterations = 5

// MaxToolPromptBytes is the hard byte cap for the prompt accumulator.
// At ~4 bytes/token, 512 KiB ≈ 128K tokens — practical ceiling for real PR diffs.
// Exported so tests can compare against it (not intended as a mutable var).
const MaxToolPromptBytes = 512 * 1024

// ViolationType* constants define the closed set of violation kinds recorded
// by the enforcement channels. Each maps to a canonical Action value in
// AccessEntry:
//
//	ViolationTypeGuardrail        → Action: "guardrail"      (post-hoc, not prevented)
//	ViolationTypeFilePermission   → Action: "read"/"write"/"delete"
//	ViolationTypeOutputContract   → Action: "slot_write"     (blocked before persist)
//	ViolationTypeNetwork          → Action: "network"        (blocked at proxy)
//	ViolationTypeLoopBoundExceeded → Action: "loop_exceeded" (post-hoc, not prevented)
const (
	ViolationTypeGuardrail         = "guardrail"
	ViolationTypeFilePermission    = "file_permission"
	ViolationTypeOutputContract    = "output_contract"
	ViolationTypeNetwork           = "network"
	ViolationTypeLoopBoundExceeded = "loop_bound_exceeded"
	ViolationTypeCallBudgetExceeded = "call_budget_exceeded" // step exceeded max_llm_calls (SRP enforcement)
)

// NodeViolation records a single enforcement event at a node boundary.
// Prevented=true means the action was blocked before it took effect (file,
// network, output-contract); Prevented=false means it was observed after the
// LLM responded (guardrail).
type NodeViolation struct {
	NodeID        string // step ID
	ViolationType string // one of ViolationType* constants
	Attempted     bool   // always true when recorded
	Prevented     bool   // true = structural block; false = post-hoc detection
	AttemptNumber int    // 1-indexed; 0 = recorded outside the retry loop
	Reason        string // stable wording from the enforcer
	Target        string // offending resource: file path, hostname, or guardrail text
}

// TaskResult captures the outcome of executing a task under one condition.
//
// GuardrailScore and ContractComplianceRate measure compliance from two
// orthogonal angles:
//   - GuardrailScore: fraction of guardrails respected in the FINAL output (post-retry).
//   - ContractComplianceRate: fraction of steps with zero violations on FIRST attempt.
//
// High GuardrailScore + low ContractComplianceRate = enforcement worked but agent needed
// coercion (Theory A). High both = natural compliance (Theory B). Low GuardrailScore =
// enforcement insufficient.
type TaskResult struct {
	TaskID          string
	Condition       string // "text" or "runtime"
	Completed       bool   // all steps produced output
	StepResults     []StepResult
	OrderingScore   float64 // 0.0-1.0: fraction of steps in correct order
	GuardrailScore  float64 // 0.0-1.0: fraction of guardrails respected (final output)
	TokensUsed      int     // total tokens consumed (prompt + completion estimate)
	WallClockMs     int64   // wall-clock time in milliseconds
	LLMCalls        int     // number of LLM provider calls made
	IssuesFound        int     // count of items that passed the kind:filter step (0 when no filter step)
	RecallScore        float64 // fraction of expected_issues found in final output (0 when no expected_issues)
	HasExpectedIssues  bool    // true when the task declared expected_issues (distinguishes 0% recall from N/A)
	CombinedOutput     string  // concatenated text output of all steps (for external eval)
	Error              error

	// Tool-based tasks (Story 5-4)
	ActionLog     []AccessEntry  `json:"-"` // full sandbox audit trail (not serialized directly)
	ActionMetrics *ActionMetrics // nil for non-tool tasks

	// Violation telemetry (Story 10-6)
	// NodeViolations lists every enforcement event in (step-index, attempt-number)
	// order. Empty slice means all steps were compliant on every attempt.
	NodeViolations        []NodeViolation `json:"node_violations,omitempty"`
	ContractComplianceRate float64        `json:"contract_compliance_rate"`
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
	FileReads  []string // files read by this step via sandbox file_read
}

// ComparisonResult holds the deltas between text and runtime conditions.
type ComparisonResult struct {
	Tasks              int
	Runs               int
	TextResults        []TaskResult
	RuntimeResults     []TaskResult
	RankedResults      []TaskResult // populated by CompareThree; holds whatever third condition was run (ranked, agent, …)
	ThirdLabel         string       // human-readable name for RankedResults (e.g. "agent", "ranked") — set by CompareThree

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

	// Per-node contract compliance + Theory A/B breakdown (Story 10-8).
	// Sorted ascending by (NodeID, Condition). Empty slice (not nil) when no
	// violation telemetry is available. Populated by ComputeNodeMetrics.
	NodeMetrics []NodeMetrics
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

// NodeMetrics captures per-node contract compliance and Theory A/B breakdown
// aggregated across all runs for a single (node, condition) pair.
//
// ContractComplianceRate is the mean of TaskResult.ContractComplianceRate values
// from runs that exercised this node (not recomputed — preserves consistency
// with the per-task view from story 10-6).
//
// PerRunRates stores the individual ContractComplianceRate from each run, in
// arrival order. Used by bootstrapCIMean to compute a 95% CI for the baseline
// delta significance check (AC4 of story 10-8). Preserved after aggregation so
// the renderer does not need the raw TaskResult slice.
//
// ByViolationType maps ViolationType* constants to their per-run breakdown.
// An empty map means the node had no enforcement events in any run (all runs
// were spontaneously compliant for every violation type).
//
// BaselineDelta is nil when no baseline snapshot has been loaded. When set
// (via ApplyBaseline), it is current ContractComplianceRate − baseline rate for
// this (node, condition). Positive = structural enforcement improved compliance.
type NodeMetrics struct {
	NodeID                 string
	Condition              string                            // "text" | "runtime" | "ranked"
	Runs                   int                               // runs that exercised this node
	ContractComplianceRate float64                           // 0.0–1.0 mean across runs
	PerRunRates            []float64                         // per-run ContractComplianceRate; for bootstrapCI
	ByViolationType        map[string]ViolationTypeBreakdown // key: ViolationType* constants
	BaselineDelta          *float64                          // current − baseline ContractComplianceRate; nil = no baseline
}

// ViolationTypeBreakdown holds Theory A/B fractions for a (node, condition,
// violation_type) triple. All percentage fields are fractions in [0.0, 1.0].
//
// Reading the breakdown:
//   - AttemptedPct high + PreventedPct high  → Theory A: enforcement does real work.
//   - AttemptedPct low (Spontaneous high)    → Theory B: agent is already aligned.
//   - AttemptedPct high + PreventedPct low   → enforcement gap (detection-only channel).
//
// Two complementary prevention metrics are provided:
//   - PreventedPct: of the runs that attempted, what fraction was blocked.
//     Measures harness efficacy when called upon (denominator = attempted).
//   - BlockedRunsPct: of all runs, what fraction experienced a structural block.
//     Measures absolute incidence of enforcement events (denominator = total runs).
//
// AttemptedPct + SpontaneouslyCompliantPct == 1.0 by construction.
type ViolationTypeBreakdown struct {
	AttemptedPct              float64 // fraction of runs with ≥1 attempt of this type
	PreventedPct              float64 // fraction of attempted runs where ≥1 was blocked
	BlockedRunsPct            float64 // fraction of total runs where ≥1 was structurally blocked
	SpontaneouslyCompliantPct float64 // fraction of runs with 0 violations (1 − AttemptedPct)
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

// LoadCorpus loads a workflow template from a single task YAML file and
// instantiates it for each entry (subdirectory) in corpusDir. Each entry
// directory provides the workspace files; the step topology comes from the
// template. The task ID is set to the entry directory name.
//
// meta.json (optional) in each entry may contain:
//
//	{"source_url": "...", "title": "...", "golden_comments": [...]}
//
// The template's own Workspace is replaced by the corpus entry's files.
func LoadCorpus(taskFile, corpusDir string) ([]TaskDefinition, error) {
	data, err := os.ReadFile(taskFile)
	if err != nil {
		return nil, fmt.Errorf("read task template %s: %w", taskFile, err)
	}
	var base TaskDefinition
	if err := yaml.Unmarshal(data, &base); err != nil {
		return nil, fmt.Errorf("parse task template %s: %w", taskFile, err)
	}
	if base.ID == "" {
		base.ID = strings.TrimSuffix(filepath.Base(taskFile), ".yaml")
	}
	for j := range base.Steps {
		if base.Steps[j].ID == "" {
			base.Steps[j].ID = fmt.Sprintf("step%d", j)
		}
	}

	canonicalCorpus, evalErr := filepath.EvalSymlinks(corpusDir)
	if evalErr != nil {
		canonicalCorpus = corpusDir
	}
	entries, err := os.ReadDir(canonicalCorpus)
	if err != nil {
		return nil, fmt.Errorf("read corpus dir: %w", err)
	}

	ghURLRe := regexp.MustCompile(`github\.com/([^/]+)/([^/]+)/pull/(\d+)`)

	var tasks []TaskDefinition
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") {
			continue
		}
		entryDir, _ := filepath.Abs(filepath.Join(canonicalCorpus, e.Name()))
		task := base
		task.ID = e.Name()
		task.Name = e.Name()
		task.TaskDir = filepath.Dir(taskFile)
		task.Workspace = &WorkspaceDef{}

		// Extract corpus variables from meta.json and expose them to kind: shell steps.
		metaData, _ := os.ReadFile(filepath.Join(entryDir, "meta.json"))
		sourceURL := jsonStringField(metaData, "source_url")
		// Corpus values override YAML-declared env (corpus is authoritative).
		if task.Env == nil {
			task.Env = make(map[string]string)
		}
		task.Env = map[string]string{"CORPUS_SOURCE_URL": sourceURL, "CORPUS_ENTRY_DIR": entryDir}
		if m := ghURLRe.FindStringSubmatch(sourceURL); m != nil {
			task.Env["CORPUS_OWNER"] = m[1]
			task.Env["CORPUS_REPO"] = m[2]
			task.Env["CORPUS_PR_NUM"] = m[3]
		}

		files, err := os.ReadDir(entryDir)
		if err != nil {
			return nil, fmt.Errorf("read corpus entry %s: %w", e.Name(), err)
		}
		for _, f := range files {
			if f.IsDir() || f.Name() == "meta.json" {
				continue
			}
			content, err := os.ReadFile(filepath.Join(entryDir, f.Name()))
			if err != nil {
				return nil, fmt.Errorf("read corpus file %s/%s: %w", e.Name(), f.Name(), err)
			}
			task.Workspace.Files = append(task.Workspace.Files, WorkspaceFile{
				Path:    f.Name(),
				Content: string(content),
			})
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// jsonStringField extracts a top-level JSON string field without a full JSON parse.
func jsonStringField(data []byte, field string) string {
	needle := `"` + field + `":`
	idx := strings.Index(string(data), needle)
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(string(data)[idx+len(needle):])
	if len(rest) == 0 || rest[0] != '"' {
		return ""
	}
	end := strings.Index(rest[1:], `"`)
	if end < 0 {
		return ""
	}
	return rest[1 : end+1]
}

// UnmarshalYAML implements yaml.Unmarshaler to support parallel: blocks in the
// steps list. Each item in steps is either a regular StepDef ({id: ...}) or a
// parallel group ({parallel: [...]}) whose members are flattened into
// TaskDefinition.Steps with a shared ParallelGroupID.
//
// YAML topology example:
//
//	steps:
//	  - id: discovery
//	    ...
//	  - parallel:
//	    - id: security-analysis
//	      consumes: [discovery_summary]
//	      produces: [security_analysis]
//	    - id: correctness-analysis
//	      consumes: [discovery_summary]
//	      produces: [correctness_analysis]
//	  - id: dedup-issues
//	    consumes: [security_analysis, correctness_analysis]
//	    ...
//
// The DAG engine auto-wires parallel group members as concurrent goroutines
// because they share the same consumes and have disjoint produces — this is
// purely structural, not a runtime configuration.
func (td *TaskDefinition) UnmarshalYAML(value *yaml.Node) error {
	// Decode all non-steps fields; unknown keys (including "steps") are silently
	// ignored by gopkg.in/yaml.v3 when decoding into a plain struct.
	type rawTask struct {
		ID                 string          `yaml:"id"`
		Name               string          `yaml:"name"`
		Description        string          `yaml:"description"`
		Category           string          `yaml:"category"`
		Input              string          `yaml:"input"`
		Guardrails         []string        `yaml:"guardrails"`
		Workspace          *WorkspaceDef   `yaml:"workspace,omitempty"`
		ExpectedViolations []string        `yaml:"expected_violations,omitempty"`
		ExpectedIssues     []ExpectedIssue `yaml:"expected_issues,omitempty"`
		ToolNodes          []StepDef       `yaml:"tool_nodes,omitempty"`
	}
	var raw rawTask
	if err := value.Decode(&raw); err != nil {
		return err
	}
	td.ID = raw.ID
	td.Name = raw.Name
	td.Description = raw.Description
	td.Category = raw.Category
	td.Input = raw.Input
	td.Guardrails = raw.Guardrails
	td.Workspace = raw.Workspace
	td.ExpectedViolations = raw.ExpectedViolations
	td.ExpectedIssues = raw.ExpectedIssues
	td.ToolNodes = raw.ToolNodes

	if value.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(value.Content); i += 2 {
		if value.Content[i].Value != "steps" {
			continue
		}
		stepsNode := value.Content[i+1]
		if stepsNode.Kind != yaml.SequenceNode {
			break
		}
		groupCounter := 0
		for _, itemNode := range stepsNode.Content {
			if itemNode.Kind != yaml.MappingNode {
				continue
			}
			// Detect parallel group: item has a "parallel" key at its root.
			isParallel := false
			for j := 0; j+1 < len(itemNode.Content); j += 2 {
				if itemNode.Content[j].Value == "parallel" {
					isParallel = true
					break
				}
			}
			if isParallel {
				var group struct {
					Steps []StepDef `yaml:"parallel"`
				}
				if err := itemNode.Decode(&group); err != nil {
					return fmt.Errorf("parallel group %d: %w", groupCounter, err)
				}
				gid := fmt.Sprintf("parallel_%d", groupCounter)
				groupCounter++
				for k := range group.Steps {
					group.Steps[k].ParallelGroupID = gid
					td.Steps = append(td.Steps, group.Steps[k])
				}
			} else {
				var step StepDef
				if err := itemNode.Decode(&step); err != nil {
					return err
				}
				td.Steps = append(td.Steps, step)
			}
		}
		break
	}
	return nil
}

// ErrLoopExhausted is returned when a step's tool-call loop exhausts its
// MaxIterations without an exit criterion firing AND the final synthesize-your-
// answer call still does not produce a valid output. Exported so callers can
// identify it via errors.Is and story 10-6 telemetry can classify it as
// ViolationTypeLoopBoundExceeded.
var ErrLoopExhausted = errors.New("max_iterations exceeded")

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

// validateLoopPolicy checks that a LoopPolicy is well-formed.
// It reports unknown YAML fields (detected at unmarshal time by the custom
// UnmarshalYAML methods) and rejects non-positive MaxIterations.
func validateLoopPolicy(lp *LoopPolicy, taskID, stepID string) error {
	if len(lp.unknownKeys) > 0 {
		return fmt.Errorf("task %s step %q: unknown loop field(s) %v (valid: max_iterations, exit_when)",
			taskID, stepID, lp.unknownKeys)
	}
	if len(lp.ExitWhen.unknownKeys) > 0 {
		return fmt.Errorf("task %s step %q: unknown loop.exit_when field(s) %v (valid: exit_when.output_marker, exit_when.tool_called)",
			taskID, stepID, lp.ExitWhen.unknownKeys)
	}
	if lp.MaxIterations <= 0 {
		return fmt.Errorf("task %s step %q: loop.max_iterations must be a positive integer (got %d)",
			taskID, stepID, lp.MaxIterations)
	}
	return nil
}

// validateNetworkPolicy checks that network policy patterns are well-formed.
// Rejected: empty patterns, path characters ('/' or '?'), port suffixes (':'),
// wildcards anywhere other than the leading '*.' prefix, and bare '*'.
func validateNetworkPolicy(p *sandbox.NetworkPolicy, taskID, stepID string) error {
	check := func(field, pat string) error {
		if pat == "" {
			return fmt.Errorf("task %s step %q: network_policy %s contains empty pattern", taskID, stepID, field)
		}
		if strings.Contains(pat, "/") {
			return fmt.Errorf("task %s step %q: network_policy %s pattern %q must be a hostname (no '/')", taskID, stepID, field, pat)
		}
		if strings.Contains(pat, "?") {
			return fmt.Errorf("task %s step %q: network_policy %s pattern %q must be a hostname (no '?')", taskID, stepID, field, pat)
		}
		if strings.Contains(pat, ":") {
			return fmt.Errorf("task %s step %q: network_policy %s pattern %q must be a bare hostname without port (no ':')", taskID, stepID, field, pat)
		}
		// Wildcard rules: '*' is only valid as a leading '*.' prefix (e.g. '*.example.com').
		// Bare '*', double wildcards '*.*.example.com', or mid-pattern '*' are rejected.
		if strings.Contains(pat, "*") {
			if !strings.HasPrefix(pat, "*.") || strings.Count(pat, "*") > 1 {
				return fmt.Errorf("task %s step %q: network_policy %s pattern %q: '*' is only valid as a leading '*.' wildcard (e.g. '*.example.com')", taskID, stepID, field, pat)
			}
		}
		return nil
	}
	for _, pat := range p.AllowedDomains {
		if err := check("allowed_domains", pat); err != nil {
			return err
		}
	}
	for _, pat := range p.ForbiddenDomains {
		if err := check("forbidden_domains", pat); err != nil {
			return err
		}
	}
	return nil
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
			if !validFileAccessValues[step.FileAccess] {
				return fmt.Errorf("task %d (%s) step %q has invalid file_access %q (must be one of: %q, %q, %q, or empty for default)",
					i, t.ID, step.ID, step.FileAccess,
					FileAccessReadOnly, FileAccessReadWrite, FileAccessReadWriteDelete)
			}
			if step.NetworkPolicy != nil {
				if err := validateNetworkPolicy(step.NetworkPolicy, t.ID, step.ID); err != nil {
					return err
				}
			}
			if step.Loop != nil {
				if err := validateLoopPolicy(step.Loop, t.ID, step.ID); err != nil {
					return err
				}
			}
			var typedProducesCount int
			var listProducesCount int
			var hasMarkdownProduces bool
			for _, slot := range step.Produces {
				if len(slot.Schema) > 0 {
					typedProducesCount++
				}
				if slot.ListSchema != nil {
					listProducesCount++
					switch slot.ListSchema.Type {
					case "string", "number", "boolean":
					default:
						return fmt.Errorf("task %d (%s) step %q slot %q: list_schema type %q unsupported (valid: string, number, boolean)",
							i, t.ID, step.ID, slot.Key, slot.ListSchema.Type)
					}
					if len(slot.Schema) > 0 {
						return fmt.Errorf("task %d (%s) step %q slot %q: list_schema and schema are mutually exclusive",
							i, t.ID, step.ID, slot.Key)
					}
				}
				if slot.Template != "" || slot.TemplateFile != "" {
					hasMarkdownProduces = true
				}
				for fieldName, spec := range slot.Schema {
					switch spec.Type {
					case "string", "number", "integer", "boolean":
					default:
						return fmt.Errorf("task %d (%s) step %q field %q: unsupported type %q (valid: string, number, integer, boolean)",
							i, t.ID, step.ID, fieldName, spec.Type)
					}
					if spec.MinLength > 0 && spec.Type != "string" {
						return fmt.Errorf("task %d (%s) step %q field %q: min_length is only valid for type \"string\" (got %q)",
							i, t.ID, step.ID, fieldName, spec.Type)
					}
				}
			}
			if typedProducesCount > 1 {
				return fmt.Errorf("task %d (%s) step %q has %d typed produces slots; at most 1 is supported (use a single schema-bearing slot)",
					i, t.ID, step.ID, typedProducesCount)
			}
			if listProducesCount > 1 {
				return fmt.Errorf("task %d (%s) step %q has %d list_schema produces slots; at most 1 is supported",
					i, t.ID, step.ID, listProducesCount)
			}
			if typedProducesCount > 0 && listProducesCount > 0 {
				return fmt.Errorf("task %d (%s) step %q mixes typed produces (schema) and list produces (list_schema); use one or the other",
					i, t.ID, step.ID)
			}
			if typedProducesCount > 0 && hasMarkdownProduces {
				return fmt.Errorf("task %d (%s) step %q mixes typed produces (schema) and Markdown produces (template/template_file); use one or the other",
					i, t.ID, step.ID)
			}
			if typedProducesCount > 0 && step.Loop != nil {
				if step.Loop.ExitWhen.OutputMarker != "" || step.Loop.ExitWhen.ToolCalled != "" {
					return fmt.Errorf("task %d (%s) step %q: typed produces is incompatible with loop exit criteria (ExitWhen); the exit criterion fires before typed-produces extraction is reached",
						i, t.ID, step.ID)
				}
			}
		}

		// Validate parallel groups: steps in the same group must be independent
		// (no step in the group may consume a slot produced by another step in the
		// same group). Violated when the YAML author declares a parallel: block
		// whose members have hidden data dependencies — the DAG would schedule them
		// concurrently but one would race to read a slot the other hasn't written yet.
		groups := make(map[string][]StepDef)
		for _, step := range t.Steps {
			if step.ParallelGroupID != "" {
				groups[step.ParallelGroupID] = append(groups[step.ParallelGroupID], step)
			}
		}
		for gid, groupSteps := range groups {
			groupProduces := make(map[string]string) // slot key → step ID that produces it
			for _, step := range groupSteps {
				for _, p := range defaultProduces(step.ID, step.Produces) {
					groupProduces[p.Key] = step.ID
				}
			}
			for _, step := range groupSteps {
				for _, c := range step.Consumes {
					if producer, ok := groupProduces[c.Key]; ok && producer != step.ID {
						return fmt.Errorf("task %d (%s) parallel group %q: step %q consumes slot %q produced by step %q in the same group (steps are not independent)",
							i, t.ID, gid, step.ID, c.Key, producer)
					}
				}
			}
		}
	}
	if c.RunsPerTask <= 0 {
		c.RunsPerTask = 1
	}
	return nil
}
