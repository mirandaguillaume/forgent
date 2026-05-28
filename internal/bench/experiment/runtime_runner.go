package experiment

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/mirandaguillaume/forgent/internal/llm"
	"github.com/mirandaguillaume/forgent/pkg/budget"
	"github.com/mirandaguillaume/forgent/pkg/dag"
	"github.com/mirandaguillaume/forgent/pkg/qualitygate"
	pkgschema "github.com/mirandaguillaume/forgent/pkg/schema"
	pkgsandbox "github.com/mirandaguillaume/forgent/pkg/sandbox"
)

// stepRunFn is the runFn signature for an in-flight DAG node — kept as a
// type alias so the consumes-retry wrapper signatures stay readable.
type stepRunFn = func(ctx context.Context, inputs map[string]any) (map[string]any, error)

// isConsumesValidationError reports whether err is a quality-gate consumes
// pre-check failure. QualityGateMiddleware wraps consumes errors as
// "consumes: <err>" so they are distinguishable from produces errors, which
// share the same "quality gate: heading/section" AST prefixes.
func isConsumesValidationError(err error) bool {
	if err == nil {
		return false
	}
	return strings.HasPrefix(err.Error(), "consumes:")
}

// consumesRetryMiddleware returns a dag.Middleware that recovers from
// consumes-template pre-check failures by clearing the upstream slot files
// from the sandbox and re-invoking the upstream producer's runFn. The
// downstream node is then retried ONCE with the refreshed inputs.
//
// Order matters: this middleware MUST sit OUTSIDE qualitygate so it can
// observe and react to qualitygate's pre-check error (qualitygate returns
// error from its pre-check WITHOUT calling next(), which is exactly what
// gives us a chance to recover before the chain unwinds).
//
// Why this lives here and not in pkg/dag: layer-based DAG execution does not
// re-run upstream nodes — `runWithRetry` only retries the failing node. AC #5
// of story 10-4 demands fresh upstream values, so recovery is implemented at
// the runtime-runner layer where producer runFn closures are addressable.
//
// Producer runFn invocation passes EMPTY inputs intentionally: LLM-step
// prompts derive from step metadata + sandbox `file_read`, not from the
// consumer's inputs map (the producer's inner runFn filters task_input
// when writing slot files). Passing the consumer's stale inputs would cause
// the producer to write them as upstream slot files — the opposite of "fresh".
func consumesRetryMiddleware(
	producers map[string]stepRunFn,
	sandbox *SandboxFS,
	step StepDef,
	consumes []SlotSpec,
) dag.Middleware {
	return func(ctx context.Context, inputs map[string]any, next dag.MiddlewareFunc) (map[string]any, error) {
		out, err := next(ctx, inputs)
		if !isConsumesValidationError(err) {
			return out, err
		}

		refreshedAny := false
		refreshedInputs := make(map[string]any, len(inputs))
		for k, v := range inputs {
			refreshedInputs[k] = v
		}

		for _, slot := range consumes {
			key := slot.Key
			if key == "task_input" {
				continue
			}
			producer, ok := producers[key]
			if !ok {
				continue
			}
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if sandbox != nil {
				// Clear both .md and .json variants — the producer may have written either.
				sandbox.record(step.ID, "slot_clear", fmt.Sprintf("slots/%s.md", key), true,
					"consumes-template failure: clearing upstream for re-execution")
				_ = sandbox.Delete(step.ID, fmt.Sprintf("slots/%s.md", key))
				_ = sandbox.Delete(step.ID, fmt.Sprintf("slots/%s.json", key))
			}
			producerOut, perr := producer(ctx, map[string]any{})
			if perr != nil {
				return nil, fmt.Errorf("step %s: upstream producer for slot %q failed during recovery: %w", step.ID, key, perr)
			}
			if newVal, ok := producerOut[key]; ok {
				refreshedInputs[key] = newVal
				refreshedAny = true
			}
		}

		if !refreshedAny {
			return nil, fmt.Errorf("consumes-retry: no upstream producer for recovery: %w", err)
		}
		return next(ctx, refreshedInputs)
	}
}

// classifyValidationReason returns a stable, parseable reason string for the
// sandbox audit trail derived from a quality-gate validation error. Story 10-6
// telemetry tallies these by prefix, so the wording must stay stable:
//
//	"heading missing: <heading>"              AST heading-absent failures
//	"section structural violation: <heading>" AST empty/no-list failures
//	"json extraction failed"                  LLM output had no parseable JSON
//	"typed schema violation: <field>"         typed field missing/wrong type/enum
//
// Falls back to the raw error string if no prefix matches (defensive — keeps the
// audit trail informative if a future error wording slips through).
func classifyValidationReason(vErr error) string {
	if vErr == nil {
		return ""
	}
	msg := vErr.Error()

	if i := strings.Index(msg, "quality gate: heading "); i >= 0 {
		rest := msg[i+len("quality gate: heading "):]
		if name, ok := extractFirstQuoted(rest); ok {
			return "heading missing: " + name
		}
		return "heading missing"
	}
	if i := strings.Index(msg, "quality gate: section "); i >= 0 {
		rest := msg[i+len("quality gate: section "):]
		if name, ok := extractFirstQuoted(rest); ok {
			return "section structural violation: " + name
		}
		return "section structural violation"
	}
	if strings.HasPrefix(msg, "json extraction:") {
		return "json extraction failed"
	}
	if i := strings.Index(msg, "typed schema: field "); i >= 0 {
		rest := msg[i+len("typed schema: field "):]
		if name, ok := extractFirstQuoted(rest); ok {
			return "typed schema violation: " + name
		}
		return "typed schema violation"
	}
	return msg
}

// providerUnwrapper is implemented by provider wrappers that layer behaviour
// on top of an inner provider. getStructuredProvider walks the chain so that
// any depth of wrapping is transparent to constrained-decoding dispatch.
type providerUnwrapper interface {
	Unwrap() llm.Provider
}

// getStructuredProvider walks the provider wrapper chain and returns the outermost
// llm.StructuredProvider that can actually deliver constrained decoding.
//
// For wrappers (callCountingProvider, limitedCallProvider): returned only when the
// inner provider also implements StructuredProvider, so the forwarding call succeeds
// and accounting (call counts, token sums) is preserved at the wrapper level.
// For leaf providers (Anthropic, OpenRouter): returned directly when they implement SP.
// Returns nil when no provider in the chain supports constrained decoding — the
// runtime falls back to text extraction.
func getStructuredProvider(p llm.Provider) llm.StructuredProvider {
	for p != nil {
		u, isWrapper := p.(providerUnwrapper)
		if !isWrapper {
			// Leaf provider — return it if it implements StructuredProvider.
			sp, _ := p.(llm.StructuredProvider)
			return sp
		}
		// Wrapper provider — return it only when its inner also supports SP,
		// guaranteeing the forwarding call won't fail.
		if sp, ok := p.(llm.StructuredProvider); ok {
			if _, innerOK := u.Unwrap().(llm.StructuredProvider); innerOK {
				return sp
			}
		}
		// Inner doesn't support SP — descend further into the chain.
		p = u.Unwrap()
	}
	return nil
}

// getAgenticProvider walks the provider wrapper chain and returns the first
// llm.AgenticProvider found. Returns nil when no provider in the chain supports
// multi-turn tool-use (CompleteWithTools).
func getAgenticProvider(p llm.Provider) llm.AgenticProvider {
	for p != nil {
		if ap, ok := p.(llm.AgenticProvider); ok {
			return ap
		}
		u, ok := p.(providerUnwrapper)
		if !ok {
			return nil
		}
		p = u.Unwrap()
	}
	return nil
}

// extractFirstQuoted returns the content of the first double-quoted token in s.
func extractFirstQuoted(s string) (string, bool) {
	l := strings.Index(s, "\"")
	if l < 0 {
		return "", false
	}
	r := strings.Index(s[l+1:], "\"")
	if r < 0 {
		return "", false
	}
	return s[l+1 : l+1+r], true
}

// effectiveFileAccess returns the file-access permission level for a given
// step. Empty StepDef.FileAccess maps to FileAccessDefault (= read-write) so
// callers can switch on the returned value exhaustively without having to
// nil-check or empty-string-check each call site. NFR40-safe: tasks
// authored before story 10-3 transparently get the read-write default.
func effectiveFileAccess(step StepDef) FileAccess {
	if step.FileAccess == "" {
		return FileAccessDefault
	}
	return step.FileAccess
}

// effectiveGuardrails returns the guardrails to enforce for a given step.
// Step-level guardrails take precedence when declared; otherwise the task-level
// guardrails are used, preserving full backward compatibility (NFR40).
//
// Returns a defensive copy so callers may sort, append, or otherwise mutate
// the result without affecting the underlying StepDef or TaskDefinition.
// Required because effectiveGuardrails is called from per-step closures that
// run concurrently under DAG execution; aliasing the source slice would risk
// races if any caller decided to filter or extend it.
func effectiveGuardrails(step StepDef, task TaskDefinition) []string {
	if step.DisableTaskGuardrails {
		return nil
	}
	src := step.Guardrails
	if len(src) == 0 {
		src = task.Guardrails
	}
	if len(src) == 0 {
		return nil
	}
	out := make([]string, len(src))
	copy(out, src)
	return out
}

// effectiveLoopPolicy returns the loop iteration contract for a step.
// When step.Loop is nil (the default), returns a LoopPolicy with
// MaxIterations = DefaultMaxToolIterations and no exit criteria — identical
// to the implicit behaviour before story 10-7 (NFR40).
// Returns by value: no aliasing concern since LoopPolicy is a plain struct.
func effectiveLoopPolicy(step StepDef) LoopPolicy {
	if step.Loop != nil {
		return *step.Loop
	}
	return LoopPolicy{MaxIterations: DefaultMaxToolIterations}
}

// exitCriteriaMet reports whether a response satisfies the declared ExitCriteria.
// The two criteria are OR-combined: either one firing is sufficient for early exit.
//
//   - OutputMarker: case-sensitive substring match in response.
//   - ToolCalled: any call in calls has Name == ec.ToolCalled.
//
// Returns false when ec is the zero value (no criteria declared).
func exitCriteriaMet(response string, ec ExitCriteria, calls []ToolCall) bool {
	if ec.OutputMarker != "" && strings.Contains(response, ec.OutputMarker) {
		return true
	}
	if ec.ToolCalled != "" {
		for _, c := range calls {
			if c.Name == ec.ToolCalled {
				return true
			}
		}
	}
	return false
}

// RunRuntimeExperiment executes all tasks as Forgent DAG agents.
// Each task is converted to a DAG where steps are nodes with consumes/produces
// contracts. The DAG engine handles ordering, parallelism, and retry.
func RunRuntimeExperiment(config ExperimentConfig) ([]TaskResult, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	total := len(config.Tasks) * config.RunsPerTask
	done := 0
	label := config.Condition
	if label == "" {
		label = "runtime"
	}
	var results []TaskResult
	for i, task := range config.Tasks {
		skip := config.SkipMap[task.ID]
		for run := 0; run < config.RunsPerTask; run++ {
			done++
			if run < skip {
				fmt.Fprintf(os.Stderr, "  [%s] task %d/%d (%s, %d steps) run %d/%d (%d/%d total) (skipped, resumed)\n",
					label, i+1, len(config.Tasks), task.ID, len(task.Steps), run+1, config.RunsPerTask, done, total)
				continue
			}
			fmt.Fprintf(os.Stderr, "  [%s] task %d/%d (%s, %d steps) run %d/%d (%d/%d total)\n",
				label, i+1, len(config.Tasks), task.ID, len(task.Steps), run+1, config.RunsPerTask, done, total)
			retriever := config.ContextRetriever
			if retriever == nil {
				retriever = NilRetriever{}
			}
			condition := label
			result := runRuntimeTask(task, config.Provider, config.StepModelOverride, retriever, condition)
			results = append(results, result)
			if config.OnResult != nil {
				config.OnResult(result, run)
			}
			if result.Error != nil {
				if IsAPIFatalError(result.Error) {
					fmt.Fprintf(os.Stderr, "  ✗ fatal API error, aborting experiment: %v\n", result.Error)
					return results, fmt.Errorf("%w: %v", ErrAPIFatal, result.Error)
				}
				fmt.Fprintf(os.Stderr, "  ✗ task %s failed: %v\n", task.ID, result.Error)
			}
		}
	}
	return results, nil
}

func runRuntimeTask(task TaskDefinition, provider llm.Provider, stepModelOverride func(string) (llm.Provider, error), retriever ContextRetriever, condition string) TaskResult {
	start := time.Now()

	// Collect ALL node outputs (not just terminal) via a shared map + callback.
	allOutputs := make(map[string]any)
	var mu sync.Mutex
	var llmCalls int32
	var llmTokensTotal int64 // includes tokens from blocked/failed calls

	// Wrap provider to count LLM calls and accumulate total tokens (including
	// calls that resulted in blocked actions — the API still charges for those).
	countingProvider := &callCountingProvider{inner: provider, calls: &llmCalls, tokens: &llmTokensTotal}

	// Build sandbox from workspace (if tool-based task)
	var sandbox *SandboxFS
	if task.Workspace != nil {
		sandbox = buildSandboxFromWorkspace(task)
	} else {
		sandbox = NewSandboxFS(nil, nil, SandboxPolicy{})
	}

	// capture collects outputs from LLM runFns (called inside each runFn).
	// WithNodeOutput below supplements this with assembled outputs from
	// kind:map and kind:filter nodes, whose per-item runFns don't call capture.
	capture := func(nodeID string, out map[string]any) {
		mu.Lock()
		for k, v := range out {
			allOutputs[k] = v
		}
		mu.Unlock()
	}

	d, nodeOutputKeys, err := buildTaskDAG(task, countingProvider, stepModelOverride, sandbox, retriever, capture)
	if err != nil {
		return TaskResult{
			TaskID:                 task.ID,
			Condition:              condition,
			Error:                  fmt.Errorf("build DAG: %w", err),
			ContractComplianceRate: 1.0, // P2: no steps ran, defensive 1.0 (not 0.0)
		}
	}
	_ = nodeOutputKeys

	inputs := map[string]any{
		"task_input": task.Input,
	}

	_, err = d.Execute(context.Background(),
		dag.WithInputs(inputs),
		dag.WithNodeOutput(func(nodeID string, out map[string]any) {
			mu.Lock()
			for k, v := range out {
				// Only overwrite if not already set by a runFn capture call —
				// the runFn capture is more precise (per-item) for LLM nodes.
				if _, exists := allOutputs[k]; !exists {
					allOutputs[k] = v
				}
			}
			mu.Unlock()
		}),
	)
	wallClock := time.Since(start).Milliseconds()

	if err != nil {
		// Surface audit trail and violation telemetry on failure too — story 10-6
		// needs them to attribute violations even when the task did not complete.
		failResult := TaskResult{
			TaskID:      task.ID,
			Condition:   condition,
			WallClockMs: wallClock,
			LLMCalls:    int(llmCalls),
			Error:       fmt.Errorf("execute DAG: %w", err),
		}
		if sandbox != nil {
			failResult.ActionLog = sandbox.Log()
			failResult.ActionMetrics = ComputeActionMetrics(sandbox.Log())
			failResult.NodeViolations = aggregateViolations(task, sandbox)
		}
		failResult.ContractComplianceRate = computeContractComplianceRate(len(task.Steps), failResult.NodeViolations)
		return failResult
	}

	result := evaluateRuntimeOutputs(task, allOutputs, sandbox)
	result.Condition = condition
	result.WallClockMs = wallClock

	// Attach sandbox metrics and violation telemetry (Story 10-6).
	if sandbox != nil {
		result.ActionLog = sandbox.Log()
		result.ActionMetrics = ComputeActionMetrics(sandbox.Log())
		result.NodeViolations = aggregateViolations(task, sandbox)
	}
	result.ContractComplianceRate = computeContractComplianceRate(len(task.Steps), result.NodeViolations)
	result.LLMCalls = int(llmCalls)
	// Use the provider-level token accumulator as the authoritative total —
	// it includes tokens from retries and blocked actions that evaluateRuntimeOutputs
	// misses (it only counts output lengths from completed steps).
	if total := int(atomic.LoadInt64(&llmTokensTotal)); total > result.TokensUsed {
		result.TokensUsed = total
	}
	return result
}

// callCountingProvider wraps an LLM provider to count calls and accumulate
// token consumption (including calls that resulted in blocked actions).
// Uses real API token counts when the provider implements TokenAwareProvider;
// falls back to (len(prompt)+len(response))/4 for providers that don't.
type callCountingProvider struct {
	inner  llm.Provider
	calls  *int32
	tokens *int64 // accumulated across ALL calls, including failed/blocked ones
}

func (p *callCountingProvider) Complete(prompt string) (string, error) {
	atomic.AddInt32(p.calls, 1)
	if tap, ok := p.inner.(llm.TokenAwareProvider); ok {
		resp, usage, err := tap.CompleteWithUsage(prompt)
		if p.tokens != nil {
			atomic.AddInt64(p.tokens, int64(usage.InputTokens+usage.OutputTokens))
		}
		return resp, err
	}
	resp, err := p.inner.Complete(prompt)
	if p.tokens != nil {
		toks := int64((len(prompt) + len(resp)) / 4)
		atomic.AddInt64(p.tokens, toks)
	}
	return resp, err
}

func (p *callCountingProvider) CompleteStructured(ctx context.Context, prompt string, jsonSchema map[string]any) (map[string]any, error) {
	atomic.AddInt32(p.calls, 1)
	sp, ok := p.inner.(llm.StructuredProvider)
	if !ok {
		return nil, fmt.Errorf("inner provider does not implement StructuredProvider")
	}
	result, err := sp.CompleteStructured(ctx, prompt, jsonSchema)
	if p.tokens != nil && err == nil {
		outBytes, _ := json.Marshal(result)
		atomic.AddInt64(p.tokens, int64((len(prompt)+len(outBytes))/4))
	}
	return result, err
}

func (p *callCountingProvider) Unwrap() llm.Provider { return p.inner }

func (p *callCountingProvider) CompleteWithTools(ctx context.Context, prompt string, tools []llm.Tool, opts ...llm.AgenticOptions) (string, llm.AgenticStats, error) {
	ap, ok := p.inner.(llm.AgenticProvider)
	if !ok {
		return "", llm.AgenticStats{}, fmt.Errorf("inner provider does not implement AgenticProvider")
	}
	// Count each API round-trip (one per iteration of the agentic loop).
	countOpt := llm.AgenticOptions{
		OnIteration: func(i int) error {
			atomic.AddInt32(p.calls, 1)
			return nil
		},
	}
	if len(opts) > 0 && opts[0].OnIteration != nil {
		outer := opts[0].OnIteration
		countOpt.OnIteration = func(i int) error {
			atomic.AddInt32(p.calls, 1)
			return outer(i)
		}
	}
	return ap.CompleteWithTools(ctx, prompt, tools, countOpt)
}

// limitedCallProvider wraps a provider and enforces a per-step max_llm_calls
// contract. When the limit is exceeded, Complete returns an error so the step
// fails cleanly and the violation is recorded by the caller.
type limitedCallProvider struct {
	inner    llm.Provider
	calls    *int32
	maxCalls int
	stepID   string
}

func (p *limitedCallProvider) Complete(prompt string) (string, error) {
	n := atomic.AddInt32(p.calls, 1)
	if int(n) > p.maxCalls {
		return "", fmt.Errorf("max_llm_calls exceeded on step %q: limit=%d (ViolationTypeCallBudgetExceeded)", p.stepID, p.maxCalls)
	}
	return p.inner.Complete(prompt)
}

func (p *limitedCallProvider) CompleteStructured(ctx context.Context, prompt string, jsonSchema map[string]any) (map[string]any, error) {
	n := atomic.AddInt32(p.calls, 1)
	if int(n) > p.maxCalls {
		return nil, fmt.Errorf("max_llm_calls exceeded on step %q: limit=%d (ViolationTypeCallBudgetExceeded)", p.stepID, p.maxCalls)
	}
	sp, ok := p.inner.(llm.StructuredProvider)
	if !ok {
		return nil, fmt.Errorf("inner provider does not implement StructuredProvider")
	}
	return sp.CompleteStructured(ctx, prompt, jsonSchema)
}

func (p *limitedCallProvider) Unwrap() llm.Provider { return p.inner }

// extractJSONArray finds and parses the first JSON array in s.
// Used for list_schema produces slots where the LLM is instructed to output []any.
// Strips markdown code fences (```json ... ```) before parsing to handle LLMs
// that ignore the "no markdown fences" instruction.
func extractJSONArray(s string) ([]any, error) {
	s = stripMarkdownFences(s)
	start := strings.Index(s, "[")
	end := strings.LastIndex(s, "]")
	if start < 0 || end <= start {
		return nil, fmt.Errorf("json extraction: no JSON array found in response")
	}
	var list []any
	if err := json.Unmarshal([]byte(s[start:end+1]), &list); err != nil {
		return nil, fmt.Errorf("json extraction: %w", err)
	}
	return list, nil
}

// stripMarkdownFences removes opening/closing markdown code fences from s.
func stripMarkdownFences(s string) string {
	s = strings.TrimSpace(s)
	for _, fence := range []string{"```json", "```"} {
		if strings.HasPrefix(s, fence) {
			s = strings.TrimPrefix(s, fence)
			s = strings.TrimSuffix(strings.TrimSpace(s), "```")
			s = strings.TrimSpace(s)
			break
		}
	}
	return s
}

// dagNodeKind maps StepKind to the corresponding dag.NodeKind.
// StepKindLLM / StepKindStatic / StepKindAssert / StepKindConcat all map to KindTask.
func dagNodeKind(k StepKind) dag.NodeKind {
	switch k {
	case StepKindMap:
		return dag.KindMap
	case StepKindFilter:
		return dag.KindFilter
	default:
		return dag.KindTask
	}
}

// listFormatInstruction returns a prompt suffix that instructs the LLM to output
// a JSON array of items matching the declared list_schema element type.
// toolBased: true when the step has file_read tools — the instruction must
// explicitly say "after reading files" so the LLM doesn't skip tool calls.
// Without this qualifier, "Output ONLY" is interpreted as "output immediately"
// and the LLM bypasses file_read entirely, returning a hallucinated array.
func listFormatInstruction(elemType string, toolBased bool) string {
	if toolBased {
		return fmt.Sprintf(
			"\nAfter reading all required files using file_read, output ONLY a valid JSON array where each element is a %s.\n"+
				"Example: [\"item one\", \"item two\"]\n"+
				"Do not wrap in markdown code fences. Do not include any text outside the array.",
			elemType,
		)
	}
	return fmt.Sprintf(
		"\nOutput ONLY a valid JSON array where each element is a %s.\n"+
			"Example: [\"item one\", \"item two\"]\n"+
			"Do not wrap in markdown code fences. Do not include any text outside the array.",
		elemType,
	)
}

// evalPredicate evaluates a deterministic predicate against a single item value.
// Supported syntax: "score >= N", "score > N", "score <= N", "score < N", "score == N"
// When item is a typed map[string]any (from a schema-producing map step), reads the
// "score" field directly. Falls back to extractScoreFromText for plain-string items.
// Returns an error if the expression is unrecognized or the item contains no score.
func evalPredicate(predicate string, item any) (bool, error) {
	pred := strings.ReplaceAll(predicate, " ", "")

	ops := []string{">=", "<=", ">", "<", "=="}
	for _, op := range ops {
		rest, found := strings.CutPrefix(pred, "score"+op)
		if !found {
			continue
		}
		threshold, err := strconv.Atoi(strings.TrimSpace(rest))
		if err != nil {
			return false, fmt.Errorf("predicate: invalid threshold in %q: %w", predicate, err)
		}
		score, err := extractScore(item)
		if err != nil {
			return false, fmt.Errorf("predicate: %w", err)
		}
		switch op {
		case ">=":
			return score >= threshold, nil
		case "<=":
			return score <= threshold, nil
		case ">":
			return score > threshold, nil
		case "<":
			return score < threshold, nil
		case "==":
			return score == threshold, nil
		}
	}
	return false, fmt.Errorf("predicate: unsupported expression %q (supported: score >= N)", predicate)
}

// extractScore reads a score from a typed map or plain text item.
// Typed maps (from schema-producing kind:map steps) carry an integer "score" field
// directly — no regex needed. Plain strings fall back to extractScoreFromText.
func extractScore(item any) (int, error) {
	if m, ok := item.(map[string]any); ok {
		if v, ok := m["score"]; ok {
			switch n := v.(type) {
			case int:
				return n, nil
			case float64:
				return int(n), nil
			case json.Number:
				i, err := n.Int64()
				return int(i), err
			}
		}
	}
	return extractScoreFromText(fmt.Sprintf("%v", item))
}

// extractScoreFromText finds the last "SCORE: N" in s and returns N.
func extractScoreFromText(s string) (int, error) {
	const marker = "SCORE:"
	idx := strings.LastIndex(strings.ToUpper(s), marker)
	if idx < 0 {
		return 0, fmt.Errorf("no SCORE: N found in item text")
	}
	rest := strings.TrimSpace(s[idx+len(marker):])
	// read digits
	end := 0
	for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
		end++
	}
	if end == 0 {
		return 0, fmt.Errorf("SCORE: found but no number follows in item text")
	}
	return strconv.Atoi(rest[:end])
}

// outputCapture is called after each node completes with its outputs.
type outputCapture func(nodeID string, out map[string]any)

// buildTaskDAG constructs a DAG from a task definition.
// Each step becomes a node; consumes/produces contracts wire the topology.
// The capture callback receives all node outputs (not just terminal).
//
// Four step behaviors are supported:
//   - llm (default): calls the LLM provider, guardrails enforced by code after each call
//   - llm with tools: LLM + tool-call loop through sandbox (Story 5-4)
//   - static: executes a Go text/template against inputs, no LLM call
//   - assert: runs a guardrail check against inputs, fails if violated
func buildTaskDAG(task TaskDefinition, provider llm.Provider, stepModelOverride func(string) (llm.Provider, error), sandbox *SandboxFS, retriever ContextRetriever, capture outputCapture) (*dag.DAG, map[string][]string, error) {
	var nodes []*dag.Node
	outputKeys := make(map[string][]string) // nodeID → produces keys

	// producerByKey holds each step's UNWRAPPED runFn keyed by every slot key
	// it produces. Used by wrapWithConsumesRetry to refresh upstream slots when
	// a consumer's consumes-template pre-check fails (story 10-4 AC #5).
	// Populated as we walk task.Steps in declaration order, so a consumer can
	// only address producers declared earlier — the same constraint the DAG
	// auto-wiring enforces, just made explicit for the recovery path.
	producerByKey := make(map[string]stepRunFn)

	// Write task_context.md only for non-workspace tasks: when a task has workspace
	// files, those files ARE the context and injecting task_context.md causes every
	// step to read it (burning tokens) even when the prompt doesn't request it.
	hasWorkspaceFiles := task.Workspace != nil && len(task.Workspace.Files) > 0
	if sandbox != nil && !hasWorkspaceFiles {
		if err := sandbox.Write("dag", "task_context.md", taskContextContent(task)); err != nil {
			fmt.Fprintf(os.Stderr, "warn: sandbox write task_context.md for %s: %v\n", task.ID, err)
		}
	}

	// Build tool_node execution closures — keyed by node ID.
	// Tool nodes are NOT added to the DAG; they're called on-demand by steps
	// that declare agent_tools. Each closure: reads consumed slots from sandbox,
	// builds a prompt from the node's definition + caller context, calls LLM.
	toolNodeFns := make(map[string]func(ctx context.Context, input map[string]any) (string, error))
	for _, node := range task.ToolNodes {
		nodeCopy := node
		nodeProvider := llm.Provider(provider)
		if nodeCopy.Model != "" && stepModelOverride != nil {
			if p, err := stepModelOverride(nodeCopy.Model); err == nil {
				nodeProvider = p
			} else {
				fmt.Fprintf(os.Stderr, "warn: tool_node %s model override %q failed: %v — using global provider\n", nodeCopy.ID, nodeCopy.Model, err)
			}
		}
		toolSandbox := sandbox
		toolNodeFns[nodeCopy.ID] = func(ctx context.Context, input map[string]any) (string, error) {
			var buf strings.Builder
			buf.WriteString(fmt.Sprintf("# Expert Node: %s\n\n%s\n\n", nodeCopy.Name, nodeCopy.Prompt))
			// Read consumed slots from the shared sandbox (written by upstream DAG nodes).
			if toolSandbox != nil {
				for _, slot := range nodeCopy.Consumes {
					if data, err := toolSandbox.Read(nodeCopy.ID, fmt.Sprintf("slots/%s.json", slot.Key)); err == nil {
						buf.WriteString(fmt.Sprintf("## %s\n```json\n%s\n```\n\n", slot.Key, string(data)))
					} else if data, err := toolSandbox.Read(nodeCopy.ID, fmt.Sprintf("slots/%s.md", slot.Key)); err == nil {
						buf.WriteString(fmt.Sprintf("## %s\n\n%s\n\n", slot.Key, string(data)))
					}
				}
			}
			// Inject the specific context provided by the calling LLM step.
			if ctxStr, ok := input["context"].(string); ok && ctxStr != "" {
				buf.WriteString(fmt.Sprintf("## Context from caller\n\n%s\n\n", ctxStr))
			}
			response, err := nodeProvider.Complete(buf.String())
			if err != nil {
				return "", fmt.Errorf("tool_node %s: %w", nodeCopy.ID, err)
			}
			return response, nil
		}
	}

	for _, step := range task.Steps {
		stepCopy := step // capture for closure
		consumes := step.Consumes
		if len(consumes) == 0 && len(nodes) == 0 {
			consumes = []SlotSpec{{Key: "task_input"}}
		}
		produces := defaultProduces(step.ID, step.Produces)
		outputKeys[step.ID] = SlotKeys(produces)

		// Resolve produces/consumes templates ONCE up-front so both the inline
		// validation (inside runFn, before slot writes) and the middleware
		// (defense-in-depth) see identical contracts. Story 10-4 / AC #1.
		var producesTemplate, consumesTemplate string
		if stepCopy.Kind == StepKindLLM {
			for _, slot := range stepCopy.Produces {
				if slot.TemplateFile != "" && task.TaskDir == "" {
					return nil, nil, fmt.Errorf("step %s: template_file %q requires TaskDir (load task via LoadTasks)", stepCopy.ID, slot.TemplateFile)
				}
				tmpl, err := slot.ResolveTemplate(task.TaskDir)
				if err != nil {
					return nil, nil, fmt.Errorf("step %s produces: %w", stepCopy.ID, err)
				}
				if tmpl != "" {
					producesTemplate = tmpl
					break
				}
			}
			for _, slot := range stepCopy.Consumes {
				if slot.TemplateFile != "" && task.TaskDir == "" {
					return nil, nil, fmt.Errorf("step %s: template_file %q requires TaskDir (load task via LoadTasks)", stepCopy.ID, slot.TemplateFile)
				}
				tmpl, err := slot.ResolveTemplate(task.TaskDir)
				if err != nil {
					return nil, nil, fmt.Errorf("step %s consumes: %w", stepCopy.ID, err)
				}
				if tmpl != "" {
					consumesTemplate = tmpl
					break
				}
			}
		}

		// Collect typed produces slots (those with a Schema declaration).
		// These take the JSON extraction path instead of the Markdown path.
		var typedProducesSlots []SlotSpec
		for _, p := range stepCopy.Produces {
			if len(p.Schema) > 0 {
				typedProducesSlots = append(typedProducesSlots, p)
			}
		}

		// For non-StructuredProvider fallback, build the format instruction once
		// (outside the runFn closure so it is computed at DAG-build time, not per-call).
		var typedFormatInstruction string
		if len(typedProducesSlots) > 0 {
			if getStructuredProvider(provider) == nil {
				typedFormatInstruction = pkgschema.ToPromptInstruction(typedProducesSlots[0].Schema)
			}
		}

		// Collect list_schema produces slots ([]any output path).
		// Mutually exclusive with typedProducesSlots (validated by Validate).
		var listProducesSlot *SlotSpec
		for i := range stepCopy.Produces {
			if stepCopy.Produces[i].ListSchema != nil {
				listProducesSlot = &stepCopy.Produces[i]
				break
			}
		}
		var listItemFormatInstruction string
		if listProducesSlot != nil {
			// Omit the text-format instruction only when the fast path will handle
			// formatting (list_schema + no tools + StructuredProvider available).
			// Steps with tools always use the text loop and need the instruction.
			listFastPath := len(stepCopy.Tools) == 0 && getStructuredProvider(provider) != nil
			if !listFastPath {
				toolBased := len(stepCopy.Tools) > 0
				listItemFormatInstruction = listFormatInstruction(listProducesSlot.ListSchema.Type, toolBased)
			}
		}

		// Resolve per-step provider: apply model override first, then call limit.
		stepProvider := llm.Provider(provider)
		if stepCopy.Model != "" && stepModelOverride != nil {
			if overrideProvider, err := stepModelOverride(stepCopy.Model); err == nil {
				stepProvider = overrideProvider
			} else {
				fmt.Fprintf(os.Stderr, "warn: step %s model override %q failed: %v — using global provider\n", stepCopy.ID, stepCopy.Model, err)
			}
		}
		var stepCallCount int32
		if stepCopy.MaxLLMCalls > 0 {
			stepProvider = &limitedCallProvider{
				inner:    stepProvider,
				calls:    &stepCallCount,
				maxCalls: stepCopy.MaxLLMCalls,
				stepID:   stepCopy.ID,
			}
		}

		var runFn func(ctx context.Context, inputs map[string]any) (map[string]any, error)

		switch stepCopy.Kind {
		case StepKindStatic:
			// Static: execute Go template against inputs — no LLM, no tokens
			runFn = func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
				tmpl, err := template.New(stepCopy.ID).Parse(stepCopy.Template)
				if err != nil {
					return nil, fmt.Errorf("step %s: parse template: %w", stepCopy.ID, err)
				}
				data := map[string]any{"inputs": inputs}
				var buf bytes.Buffer
				if err := tmpl.Execute(&buf, data); err != nil {
					return nil, fmt.Errorf("step %s: execute template: %w", stepCopy.ID, err)
				}
				out := make(map[string]any)
				for _, p := range stepCopy.Produces {
					out[p.Key] = buf.String()
				}
				if capture != nil {
					capture(stepCopy.ID, out)
				}
				return out, nil
			}

		case StepKindAssert:
			// Assert: verify guardrails/checks against inputs — no LLM, fail on violation
			runFn = func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
				// Aggregate all input text
				var allText strings.Builder
				for _, v := range inputs {
					if s, ok := v.(string); ok {
						allText.WriteString(s)
						allText.WriteString("\n")
					}
				}
				lower := strings.ToLower(allText.String())

				// Run the named check
				switch stepCopy.Check {
				case "guardrails_respected":
					for _, g := range effectiveGuardrails(stepCopy, task) {
						if !checkGuardrail(lower, g) {
							return nil, fmt.Errorf("step %s: guardrail violated: %s", stepCopy.ID, g)
						}
					}
				case "all_inputs_present":
					for k, v := range inputs {
						if s, ok := v.(string); ok && strings.TrimSpace(s) == "" {
							return nil, fmt.Errorf("step %s: empty input: %s", stepCopy.ID, k)
						}
					}
				case "no_empty_steps":
					for k, v := range inputs {
						if s, ok := v.(string); ok && len(strings.TrimSpace(s)) < 10 {
							return nil, fmt.Errorf("step %s: suspiciously short input %s (%d chars)", stepCopy.ID, k, len(s))
						}
					}
				default:
					// Unknown check — pass through (lenient)
				}

				out := make(map[string]any)
				for _, p := range stepCopy.Produces {
					out[p.Key] = allText.String()
				}
				if capture != nil {
					capture(stepCopy.ID, out)
				}
				return out, nil
			}

		case StepKindMap:
			// Map: LLM called once per item in the consumed []any slot (parallel).
			// runFn receives a single item as input (DAG runMap injects it under the
			// consumed slot key). Returns {produces[0]: result_string|map} per item.
			// DAG runMap collects all results into a []any output slot.
			// When produces[0] declares a Schema, uses CompleteStructured so each
			// item is a guaranteed typed map — filter predicates read it directly.
			stepSandboxM := sandbox
			mapTypedSchema := map[string]any(nil)
			if len(produces) > 0 && len(produces[0].Schema) > 0 {
				mapTypedSchema = pkgschema.ToJSONSchema(produces[0].Schema)
			}
			runFn = func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
				// Build prompt: task header + step instructions + single-item context.
				var base strings.Builder
				base.WriteString(fmt.Sprintf("# Task: %s\n\n## Step: %s\n\n%s\n\n", task.Name, stepCopy.Name, stepCopy.Prompt))
				// Inject the item value directly into the prompt context.
				for k, v := range inputs {
					switch sv := v.(type) {
					case string:
						base.WriteString(fmt.Sprintf("## Input item (%s)\n\n%s\n\n", k, sv))
					default:
						if b, err := json.Marshal(v); err == nil {
							base.WriteString(fmt.Sprintf("## Input item (%s)\n\n%s\n\n", k, string(b)))
						}
					}
				}
				if stepSandboxM != nil {
					stepSandboxM.SetCurrentAttempt(stepCopy.ID, 1)
				}
				out := make(map[string]any)
				if mapTypedSchema != nil {
					if sp := getStructuredProvider(stepProvider); sp != nil {
						parsed, err := sp.CompleteStructured(ctx, base.String(), mapTypedSchema)
						if err != nil {
							return nil, fmt.Errorf("step %s map item: %w", stepCopy.ID, err)
						}
						if len(produces) > 0 {
							out[produces[0].Key] = parsed
						}
						return out, nil
					}
				}
				response, err := stepProvider.Complete(base.String())
				if err != nil {
					return nil, fmt.Errorf("step %s map item: %w", stepCopy.ID, err)
				}
				if len(produces) > 0 {
					out[produces[0].Key] = response
				}
				return out, nil
			}

		case StepKindFilter:
			// Filter: evaluates each item deterministically (Predicate field) or via LLM.
			// Deterministic path (preferred): Predicate is a simple expression like "score >= 80".
			// LLM path (fallback): prompt instructs the LLM to respond "KEEP" or "DROP".
			// Both paths return {"__keep": bool} consumed by the DAG runFilter engine.
			if stepCopy.Predicate != "" {
				pred := stepCopy.Predicate
				runFn = func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
					// Find the single item value — the DAG injects it under the consumed slot key.
					var item any
					for _, v := range inputs {
						item = v
						break
					}
					keep, err := evalPredicate(pred, item)
					if err != nil {
						// Malformed score (e.g. "SCORE: N/A") → drop the item, don't crash the DAG.
						return map[string]any{"__keep": false}, nil
					}
					return map[string]any{"__keep": keep}, nil
				}
			} else {
				stepSandboxF := sandbox
				runFn = func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
					var base strings.Builder
					base.WriteString(fmt.Sprintf("# Task: %s\n\n## Step: %s\n\n%s\n\n", task.Name, stepCopy.Name, stepCopy.Prompt))
					for k, v := range inputs {
						if s, ok := v.(string); ok {
							base.WriteString(fmt.Sprintf("## Item (%s)\n\n%s\n\n", k, s))
						}
					}
					base.WriteString("Respond with 'KEEP' if this item should be included, or 'DROP' if it should be excluded.\n")
					if stepSandboxF != nil {
						stepSandboxF.SetCurrentAttempt(stepCopy.ID, 1)
					}
					response, err := stepProvider.Complete(base.String())
					if err != nil {
						return nil, fmt.Errorf("step %s filter item: %w", stepCopy.ID, err)
					}
					keep := strings.Contains(strings.ToUpper(strings.TrimSpace(response)), "KEEP")
					return map[string]any{"__keep": keep}, nil
				}
			}

		case StepKindConcat:
			// Concat: merge multiple []any input slots into a single []any output — no LLM call.
			// Each consumed slot must be []any (from list_schema) or a JSON array string.
			// When Deduplicate is true, items are deduped by their string representation.
			stepSandboxC := sandbox
			dedup := stepCopy.Deduplicate
			runFn = func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
				var combined []any
				for _, slot := range consumes {
					v, ok := inputs[slot.Key]
					if !ok {
						continue
					}
					switch sv := v.(type) {
					case []any:
						combined = append(combined, sv...)
					case string:
						if list, err := extractJSONArray(sv); err == nil {
							combined = append(combined, list...)
						}
					}
				}
				if dedup {
					seen := make(map[string]bool, len(combined))
					deduped := make([]any, 0, len(combined))
					for _, item := range combined {
						k := fmt.Sprintf("%v", item)
						if !seen[k] {
							seen[k] = true
							deduped = append(deduped, item)
						}
					}
					combined = deduped
				}
				if combined == nil {
					combined = []any{}
				}
				produceKey := produces[0].Key
				if stepSandboxC != nil {
					if b, merr := json.Marshal(combined); merr == nil {
						if werr := stepSandboxC.Write(stepCopy.ID, fmt.Sprintf("slots/%s.json", produceKey), b); werr != nil {
							fmt.Fprintf(os.Stderr, "warn: concat write slot %s: %v\n", produceKey, werr)
						}
						stepSandboxC.record(stepCopy.ID, "slot_write", fmt.Sprintf("slots/%s.json", produceKey), true, "concat")
					}
				}
				out := map[string]any{produceKey: combined}
				if capture != nil {
					capture(stepCopy.ID, out)
				}
				return out, nil
			}

		case StepKindShell:
			// Shell: run a bash command — fully deterministic, no LLM call.
			// stdout → sandbox workspace file (step.Writes) + produces[0] string slot.
			// task.Env provides corpus variables (CORPUS_OWNER, CORPUS_REPO, etc.).
			shellCmd := stepCopy.Shell
			writesFile := stepCopy.Writes
			shellSandbox := sandbox
			shellStepID := stepCopy.ID
			taskEnv := task.Env
			taskDir := task.TaskDir
			shellProduces := produces
			runFn = func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
				cmd := exec.CommandContext(ctx, "bash", "-c", shellCmd)
				cmd.Env = os.Environ()
				for k, v := range taskEnv {
					cmd.Env = append(cmd.Env, k+"="+v)
				}
				if taskDir != "" {
					cmd.Dir = taskDir
				}
				out, err := cmd.Output()
				if err != nil {
					var exitErr *exec.ExitError
					if errors.As(err, &exitErr) && len(exitErr.Stderr) > 0 {
						return nil, fmt.Errorf("step %s shell: %w\nstderr: %s", shellStepID, err, string(exitErr.Stderr))
					}
					return nil, fmt.Errorf("step %s shell: %w", shellStepID, err)
				}
				content := string(out)
				if writesFile != "" && shellSandbox != nil {
					if werr := shellSandbox.Write(shellStepID, writesFile, out); werr != nil {
						fmt.Fprintf(os.Stderr, "warn: shell write %s: %v\n", writesFile, werr)
					}
				}
				result := map[string]any{}
				if len(shellProduces) > 0 {
					slot := shellProduces[0]
					if slot.ListSchema != nil {
						// Shell step declares a list output — parse stdout as JSON array.
						// If stdout is not valid JSON, split into one item per non-empty line.
						var list []any
						if err := json.Unmarshal(out, &list); err == nil {
							result[slot.Key] = list
						} else {
							var lines []any
							for _, line := range strings.Split(strings.TrimSpace(content), "\n") {
								if line != "" {
									lines = append(lines, line)
								}
							}
							result[slot.Key] = lines
						}
					} else {
						result[slot.Key] = content
					}
				}
				if capture != nil {
					capture(shellStepID, result)
				}
				return result, nil
			}

		case StepKindLoop:
			// Loop (Reflector): body LLM generates a draft; router LLM decides CONTINUE or exit.
			// Router feedback is injected into the next body prompt. No DAG back-edges —
			// the entire loop runs inside this single runFn closure.
			maxIter := stepCopy.MaxIterations
			if maxIter <= 0 {
				maxIter = 4
			}
			bodyDef := stepCopy.Body
			routerDef := stepCopy.LoopRouter
			loopStepID := stepCopy.ID
			loopProduces := produces

			bodyProvider := llm.Provider(stepProvider)
			if bodyDef != nil && bodyDef.Model != "" && stepModelOverride != nil {
				if p, err := stepModelOverride(bodyDef.Model); err == nil {
					bodyProvider = p
				} else {
					fmt.Fprintf(os.Stderr, "warn: loop step %s body model %q: %v\n", loopStepID, bodyDef.Model, err)
				}
			}
			routerProvider := llm.Provider(stepProvider)
			if routerDef != nil && routerDef.Model != "" && stepModelOverride != nil {
				if p, err := stepModelOverride(routerDef.Model); err == nil {
					routerProvider = p
				} else {
					fmt.Fprintf(os.Stderr, "warn: loop step %s router model %q: %v\n", loopStepID, routerDef.Model, err)
				}
			}

			runFn = func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
				if bodyDef == nil {
					return map[string]any{}, nil
				}
				exitWhen := ""
				if routerDef != nil {
					exitWhen = routerDef.ExitWhen
				}
				var lastBodyResp string
				var routerFeedback string

				for i := 0; i < maxIter; i++ {
					var buf strings.Builder
					buf.WriteString(fmt.Sprintf("# Task: %s\n\n## Step: %s\n\n%s\n\n", task.Name, stepCopy.Name, bodyDef.Prompt))
					for k, v := range inputs {
						if s, ok := v.(string); ok {
							buf.WriteString(fmt.Sprintf("## %s\n\n%s\n\n", k, s))
						}
					}
					if lastBodyResp != "" {
						buf.WriteString(fmt.Sprintf("## Previous draft (iteration %d)\n\n%s\n\n", i, lastBodyResp))
					}
					if routerFeedback != "" {
						buf.WriteString(fmt.Sprintf("## Router feedback\n\n%s\n\n", routerFeedback))
					}

					bodyResp, err := bodyProvider.Complete(buf.String())
					if err != nil {
						return nil, fmt.Errorf("step %s body iteration %d: %w", loopStepID, i+1, err)
					}
					lastBodyResp = bodyResp

					if routerDef == nil {
						continue
					}
					var rbuf strings.Builder
					rbuf.WriteString(fmt.Sprintf("## Body output (iteration %d)\n\n%s\n\n", i+1, bodyResp))
					rbuf.WriteString(fmt.Sprintf("## Router instructions\n\n%s\n\n", routerDef.Prompt))
					routerResp, err := routerProvider.Complete(rbuf.String())
					if err != nil {
						return nil, fmt.Errorf("step %s router iteration %d: %w", loopStepID, i+1, err)
					}
					routerFeedback = routerResp
					if exitWhen != "" && strings.Contains(strings.ToUpper(routerResp), strings.ToUpper(exitWhen)) {
						break
					}
				}

				out := make(map[string]any)
				for _, slot := range loopProduces {
					out[slot.Key] = lastBodyResp
				}
				if capture != nil {
					capture(loopStepID, out)
				}
				return out, nil
			}

		default:
			// LLM step: call provider with guardrail enforcement + retry
			// All LLM steps use file-based slot passing via the sandbox:
			// consumed slots are written to slots/{key}.md before the LLM call,
			// and the prompt lists file paths instead of embedding content inline.
			// file_read is always available so the LLM can pull context on demand.
			stepSandbox := sandbox // capture sandbox reference for closure
			stepConsumes := consumes
			stepNetworkPolicy := stepCopy.NetworkPolicy // capture for closure
			runFn = func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
				// Network proxy lifecycle (Story 10-5): start a policy-enforcing HTTP proxy
				// for this step if a NetworkPolicy is declared. Only HTTP_PROXY is set —
				// not HTTPS_PROXY — because the LLM inference API (Anthropic, Ollama) uses
				// HTTPS and the experiment runner runs text+runtime in parallel goroutines.
				// Setting HTTPS_PROXY process-globally would race with the text goroutine's
				// Anthropic calls and block them through the enforcement proxy.
				// LLM-controlled tool calls in the experiment use file_read (local FS, no
				// HTTP), so HTTP_PROXY suffices for the current task corpus. Phase 2 will
				// switch to per-request http.Transport.Proxy to lift this constraint.
				if stepNetworkPolicy != nil {
					auditFn := pkgsandbox.AuditFunc(func(host string, allowed bool, reason string) {
						if stepSandbox != nil {
							stepSandbox.record(stepCopy.ID, "network", host, allowed, reason)
						}
					})
					proxy, err := pkgsandbox.NewProxyServer(stepNetworkPolicy, auditFn)
					if err != nil {
						return nil, fmt.Errorf("step %s: start network proxy: %w", stepCopy.ID, err)
					}
					oldHTTP, hadHTTP := os.LookupEnv("HTTP_PROXY")
					os.Setenv("HTTP_PROXY", proxy.URL()) //nolint:errcheck
					defer func() {
						proxy.Close() //nolint:errcheck
						if hadHTTP {
							os.Setenv("HTTP_PROXY", oldHTTP) //nolint:errcheck
						} else {
							os.Unsetenv("HTTP_PROXY") //nolint:errcheck
						}
					}()
				}

				// Write consumed slot values to sandbox files before calling the LLM.
				// Typed inputs (map[string]any from a schema-producing step) are written
				// as slots/<key>.json; free-text inputs are written as slots/<key>.md.
				if stepSandbox != nil {
					for k, v := range inputs {
						if k == "task_input" {
							continue
						}
						switch val := v.(type) {
						case string:
							if err := stepSandbox.Write(stepCopy.ID, fmt.Sprintf("slots/%s.md", k), []byte(val)); err != nil {
								fmt.Fprintf(os.Stderr, "warn: sandbox write input slot %s for step %s: %v\n", k, stepCopy.ID, err)
							}
						case map[string]any:
							if b, err := json.Marshal(val); err == nil {
								if werr := stepSandbox.Write(stepCopy.ID, fmt.Sprintf("slots/%s.json", k), b); werr != nil {
									fmt.Fprintf(os.Stderr, "warn: sandbox write typed input slot %s for step %s: %v\n", k, stepCopy.ID, werr)
								}
							}
						case []any:
							if val == nil {
								val = []any{}
							}
							if b, err := json.Marshal(val); err == nil {
								if werr := stepSandbox.Write(stepCopy.ID, fmt.Sprintf("slots/%s.json", k), b); werr != nil {
									fmt.Fprintf(os.Stderr, "warn: sandbox write list input slot %s for step %s: %v\n", k, stepCopy.ID, werr)
								}
							}
						}
					}
				}

				// Build base prompt: minimum required = step instructions + file paths.
				var base strings.Builder
				base.WriteString(fmt.Sprintf("# Task: %s\n\n", task.Name))
				base.WriteString(fmt.Sprintf("## Step: %s\n\n%s\n\n", stepCopy.Name, stepCopy.Prompt))

				// TextResolver owns the allowed-tool set, prompt description, and dispatch.
				// Built once here; shared across guardrail retry attempts.
				resolver := NewTextResolver(stepCopy.Tools, stepSandbox, stepCopy)
				base.WriteString(resolver.Describe())

				stepGuardrails := effectiveGuardrails(stepCopy, task)
				if len(stepGuardrails) > 0 {
					base.WriteString("## Constraints (MUST respect)\n\n")
					for _, g := range stepGuardrails {
						base.WriteString(fmt.Sprintf("- %s\n", g))
					}
					base.WriteString("\n")
				}

				// Context injection: pre-loaded excerpts from consumed slots (if retriever set).
				// The retriever can read sandbox files because slot writes happened above.
				if retriever != nil {
					if excerpt, _ := retriever.Retrieve(stepCopy, stepSandbox, stepConsumes); excerpt != "" {
						base.WriteString("## Context (pre-loaded excerpt)\n\n")
						base.WriteString(excerpt)
					}
				}

				// File listing replaces inline input injection — LLM reads on demand.
				// Typed/list inputs are available as .json; free-text inputs as .md.
				// Exception: []any slots are also injected inline so steps that don't
				// call file_read (e.g. post-comment on small Haiku) still see the data.
				base.WriteString("## Available files (use file_read to access)\n")
				if !hasWorkspaceFiles {
					base.WriteString("- task_context.md — full task context and original input\n")
				} else if task.Workspace != nil {
					for _, wf := range task.Workspace.Files {
						base.WriteString(fmt.Sprintf("- %s\n", wf.Path))
					}
				}
				var inlineSlots []SlotSpec
				for _, slot := range stepConsumes {
					if slot.Key != "task_input" {
						switch inputs[slot.Key].(type) {
						case []any:
							// Inject inline below; still list as file for reference.
							base.WriteString(fmt.Sprintf("- slots/%s.json — structured output from: %s\n", slot.Key, slot.Key))
							inlineSlots = append(inlineSlots, slot)
						case map[string]any:
							base.WriteString(fmt.Sprintf("- slots/%s.json — structured output from: %s\n", slot.Key, slot.Key))
						default:
							base.WriteString(fmt.Sprintf("- slots/%s.md — output from: %s\n", slot.Key, slot.Key))
						}
					}
				}
				base.WriteString("\n")
				// Inline-inject []any slots so the LLM has the data without calling file_read.
				if len(inlineSlots) > 0 {
					base.WriteString("## Pre-loaded inputs\n\n")
					for _, slot := range inlineSlots {
						val := inputs[slot.Key].([]any)
						if b, err := json.Marshal(val); err == nil {
							base.WriteString(fmt.Sprintf("### %s\n```json\n%s\n```\n\n", slot.Key, string(b)))
						}
					}
				}

				// Append format instruction for typed or list produces.
				// Placed at the END of the base prompt so it is the LLM's last constraint.
				baseStr := base.String()
				if typedFormatInstruction != "" {
					baseStr += "\n\n" + typedFormatInstruction + "\n"
				} else if listItemFormatInstruction != "" {
					baseStr += listItemFormatInstruction + "\n"
				}
				basePrompt := baseStr

				// Accumulator preserves tool-result context across guardrail retries.
				// Base = static step prompt (without retry prefix so it stays clean).
				acc := newPromptAccumulator(MaxToolPromptBytes)
				acc.AppendBase(basePrompt)

				// ── AgentTools path (agent-as-node) ──────────────────────────────
				// When the step declares agent_tools, build llm.Tool instances from
				// the task's tool_nodes and call CompleteWithTools. Each tool.Run
				// executes the referenced tool_node inline (reads sandbox slots,
				// calls LLM with node's own prompt + caller context). This path
				// bypasses the text-based tool loop and guardrail retry — the
				// agentic loop is managed by the AgenticProvider implementation.
				if len(stepCopy.AgentTools) > 0 {
					if ap := getAgenticProvider(stepProvider); ap != nil {
						agentTools := make([]llm.Tool, 0, len(stepCopy.Tools)+len(stepCopy.AgentTools))

						// Expose primitive tools (file_read, etc.) as real llm.Tool instances
						// so the LLM can read sandbox files via the Anthropic tool_use API.
						// Without this, file_read is only available in the text-loop path.
						for _, toolName := range stepCopy.Tools {
							if toolName != "file_read" {
								continue
							}
							sbRef := stepSandbox
							stepIDRef := stepCopy.ID
							agentTools = append(agentTools, llm.Tool{
								Name:        "file_read",
								Description: "Read a file from the workspace (pr_diff.patch, pr_description.md, git_history.txt, slots/*.md, slots/*.json)",
								InputSchema: map[string]any{
									"type": "object",
									"properties": map[string]any{
										"path": map[string]any{
											"type":        "string",
											"description": "File path relative to workspace root",
										},
									},
									"required": []string{"path"},
								},
								Run: func(ctx context.Context, input map[string]any) (string, error) {
									path, _ := input["path"].(string)
									if path == "" {
										return "error: path is required", nil
									}
									if sbRef == nil {
										return "error: no sandbox available", nil
									}
									data, err := sbRef.Read(stepIDRef, path)
									if err != nil {
										return fmt.Sprintf("error reading %s: %v", path, err), nil
									}
									return string(data), nil
								},
							})
						}
						for _, toolID := range stepCopy.AgentTools {
							fn, ok := toolNodeFns[toolID]
							if !ok {
								return nil, fmt.Errorf("step %s: agent_tool %q not found in tool_nodes", stepCopy.ID, toolID)
							}
							// Find the tool_node definition for name/description.
							var nodeDef StepDef
							for _, n := range task.ToolNodes {
								if n.ID == toolID {
									nodeDef = n
									break
								}
							}
							fnCopy := fn
							nodeID := toolID
							agentTools = append(agentTools, llm.Tool{
								Name:        nodeID,
								Description: nodeDef.Name,
								InputSchema: map[string]any{
									"type": "object",
									"properties": map[string]any{
										"context": map[string]any{
											"type":        "string",
											"description": "The specific code snippet or concern to investigate",
										},
									},
									"required": []string{"context"},
								},
								Run: fnCopy,
							})
						}
						finalResponse, _, agentErr := ap.CompleteWithTools(ctx, acc.String(), agentTools)
						if agentErr != nil {
							return nil, fmt.Errorf("step %s agent_tools: %w", stepCopy.ID, agentErr)
						}
						out := make(map[string]any)
						for _, p := range produces {
							out[p.Key] = finalResponse
						}
						// Handle list_schema produces: extract JSON array from response.
						if listProducesSlot != nil {
							if list, err := extractJSONArray(finalResponse); err == nil {
								out[listProducesSlot.Key] = list
							} else {
								out[listProducesSlot.Key] = []any{finalResponse}
							}
						}
						if stepSandbox != nil {
							for k, v := range out {
								if s, ok := v.(string); ok {
									_ = stepSandbox.Write(stepCopy.ID, fmt.Sprintf("slots/%s.md", k), []byte(s))
								}
							}
						}
						if capture != nil {
							capture(stepCopy.ID, out)
						}
						return out, nil
					}
					// No AgenticProvider available — fall through to text loop.
					fmt.Fprintf(os.Stderr, "warn: step %s declares agent_tools but provider does not implement AgenticProvider — falling back to text loop\n", stepCopy.ID)
				}

				// Loop policy for this step (constant across attempts).
				loopPolicy := effectiveLoopPolicy(stepCopy)
				hasExitCriterion := loopPolicy.ExitWhen.OutputMarker != "" || loopPolicy.ExitWhen.ToolCalled != ""

				// exitToolArgs captures the args of the named exit tool (T4.3).
				// Added to out under "<stepID>_<toolName>_args" after the loop.
				var exitToolArgs map[string]string

				// ── StructuredProvider fast path (Story 10-10 P1) ──────────────────
				// For typed-only steps with a StructuredProvider, skip the text loop
				// entirely. Constrained decoding eliminates format non-determinism and
				// the schema is the enforcement mechanism — guardrail keyword checks
				// on JSON-formatted output are never actionable.
				if len(typedProducesSlots) > 0 {
					if sp := getStructuredProvider(provider); sp != nil {
						jsonSchema := pkgschema.ToJSONSchema(typedProducesSlots[0].Schema)
						parsed, spErr := sp.CompleteStructured(ctx, acc.String(), jsonSchema)
						if spErr != nil {
							return nil, fmt.Errorf("step %s: structured complete: %w", stepCopy.ID, spErr)
						}
						for _, p := range typedProducesSlots {
							if vErr := qualitygate.ValidateTyped(parsed, p.Schema); vErr != nil {
								reason := classifyValidationReason(vErr)
								if stepSandbox != nil {
									stepSandbox.record(stepCopy.ID, "slot_write", fmt.Sprintf("slots/%s.json", p.Key), false, reason)
								}
								return nil, vErr
							}
							if stepSandbox != nil {
								b, mErr := json.Marshal(parsed)
								if mErr != nil {
									return nil, fmt.Errorf("step %s: marshal typed slot %s: %w", stepCopy.ID, p.Key, mErr)
								}
								if err := stepSandbox.Write(stepCopy.ID, fmt.Sprintf("slots/%s.json", p.Key), b); err != nil {
									return nil, fmt.Errorf("step %s: write typed slot %s: %w", stepCopy.ID, p.Key, err)
								}
								stepSandbox.record(stepCopy.ID, "slot_write", fmt.Sprintf("slots/%s.json", p.Key), true, "typed schema valid")
							}
						}
						out := make(map[string]any, len(typedProducesSlots))
						for _, p := range typedProducesSlots {
							out[p.Key] = parsed
						}
						if capture != nil {
							capture(stepCopy.ID, out)
						}
						return out, nil
					}
				}

				// ── list_schema StructuredProvider fast path ─────────────────────
				// Symmetric to the typed fast path above: when the step produces a
				// list_schema slot and a StructuredProvider is available, use
				// constrained decoding instead of the text loop + fragile extraction.
				// The wrapper object {"items": [...]} is required because Anthropic's
				// tool-use constrained decoding requires an object root schema.
				if listProducesSlot != nil && len(stepCopy.Tools) == 0 {
					if sp := getStructuredProvider(provider); sp != nil {
						wrapperSchema := map[string]any{
							"type": "object",
							"properties": map[string]any{
								"items": map[string]any{
									"type":  "array",
									"items": map[string]any{"type": listProducesSlot.ListSchema.Type},
								},
							},
							"required": []string{"items"},
						}
						parsed, spErr := sp.CompleteStructured(ctx, acc.String(), wrapperSchema)
						if spErr != nil {
							return nil, fmt.Errorf("step %s: structured complete (list): %w", stepCopy.ID, spErr)
						}
						rawItems, ok := parsed["items"].([]any)
						if !ok {
							return nil, fmt.Errorf("step %s: structured response missing 'items' array", stepCopy.ID)
						}
						if stepSandbox != nil {
							b, _ := json.Marshal(rawItems)
							if werr := stepSandbox.Write(stepCopy.ID, fmt.Sprintf("slots/%s.json", listProducesSlot.Key), b); werr != nil {
								return nil, fmt.Errorf("step %s: write list slot %s: %w", stepCopy.ID, listProducesSlot.Key, werr)
							}
							stepSandbox.record(stepCopy.ID, "slot_write", fmt.Sprintf("slots/%s.json", listProducesSlot.Key), true, "list schema valid")
						}
						out := map[string]any{listProducesSlot.Key: rawItems}
						if capture != nil {
							capture(stepCopy.ID, out)
						}
						return out, nil
					}
				}

				// Try up to 3 times, checking guardrails after each attempt.
				maxAttempts := 3
				var lastResponse string
				for attempt := 0; attempt < maxAttempts; attempt++ {
					// Story 10-6: stamp attempt for violation telemetry.
					if stepSandbox != nil {
						stepSandbox.SetCurrentAttempt(stepCopy.ID, attempt+1)
					}

					retryPrefix := ""
					if attempt > 0 {
						retryPrefix = fmt.Sprintf("\n\n[Previous attempt violated constraints. Retry %d/%d. Strictly respect ALL constraints listed above.]\n", attempt+1, maxAttempts)
					}

					var response string
					var iterErr error

					if stepCopy.Loop != nil {
						// ── Declarative loop ─────────────────────────────────────────────
						// First provider call is iteration 0 of the loop.
						// Total calls per guardrail attempt = MaxIterations (loop) + 1 (synthesize if needed).
						// Note: stepSandbox is always non-nil (NewSandboxFS fallback in runRuntimeTask).
						loopComplete := false

						for iter := 0; iter < loopPolicy.MaxIterations; iter++ {
							response, iterErr = stepProvider.Complete(acc.String() + retryPrefix)
							if iterErr != nil {
								if stepSandbox != nil && stepCopy.MaxLLMCalls > 0 && strings.Contains(iterErr.Error(), "ViolationTypeCallBudgetExceeded") {
									stepSandbox.record(stepCopy.ID, "call_budget_exceeded", fmt.Sprintf("max_llm_calls=%d exceeded", stepCopy.MaxLLMCalls), false, iterErr.Error())
								}
								return nil, fmt.Errorf("step %s attempt %d iter %d: %w", stepCopy.ID, attempt+1, iter+1, iterErr)
							}

							calls := ParseToolCalls(response)

							if exitCriteriaMet(response, loopPolicy.ExitWhen, calls) {
								if loopPolicy.ExitWhen.ToolCalled != "" {
									for _, c := range calls {
										if c.Name == loopPolicy.ExitWhen.ToolCalled {
											exitToolArgs = c.Args
											break
										}
									}
								}
								loopComplete = true
								break
							}

							// No exit criterion + no tool calls → LLM gave a plain-text answer.
							if !hasExitCriterion && len(calls) == 0 {
								loopComplete = true
								break
							}

							if len(calls) > 0 {
								results := resolver.Resolve(calls)
								acc.AppendToolResults("\n\n" + FormatToolResults(results) +
									"\nContinue your analysis based on the tool results above. If you need more tools, use them. Otherwise, provide your final answer.\n")
							}
						}

						if !loopComplete {
							// MaxIterations exhausted: one final synthesize-your-answer call.
							acc.AppendToolResults("\n\n## Final Answer Required\nThe loop has exhausted. Produce your final answer now.")
							response, iterErr = stepProvider.Complete(acc.String() + retryPrefix)
							if iterErr != nil {
								return nil, fmt.Errorf("step %s synthesize attempt %d: %w", stepCopy.ID, attempt+1, iterErr)
							}
							synthCalls := ParseToolCalls(response)
							met := exitCriteriaMet(response, loopPolicy.ExitWhen, synthCalls)
							// Malformed: criteria unmet, OR no criterion but still emitting tool calls.
							if !met && (hasExitCriterion || len(synthCalls) > 0) {
								if stepSandbox != nil {
									stepSandbox.record(stepCopy.ID, "loop_exceeded", "", false, "max_iterations exceeded")
								}
								return nil, ErrLoopExhausted
							}
						}

					} else {
						// ── Legacy implicit loop ──────────────────────────────────────────
						// Initial provider call first; tool loop follows on the response.
						response, iterErr = stepProvider.Complete(acc.String() + retryPrefix)
						if iterErr != nil {
							if stepSandbox != nil && stepCopy.MaxLLMCalls > 0 && strings.Contains(iterErr.Error(), "ViolationTypeCallBudgetExceeded") {
								stepSandbox.record(stepCopy.ID, "call_budget_exceeded", fmt.Sprintf("max_llm_calls=%d exceeded", stepCopy.MaxLLMCalls), false, iterErr.Error())
							}
							return nil, fmt.Errorf("step %s attempt %d: %w", stepCopy.ID, attempt+1, iterErr)
						}

						if stepSandbox != nil {
							for iter := 0; iter < loopPolicy.MaxIterations; iter++ {
								calls := ParseToolCalls(response)
								if len(calls) == 0 {
									break
								}
								acc.AppendToolResults("\n\n" + FormatToolResults(resolver.Resolve(calls)) +
									"\nContinue your analysis based on the tool results above. If you need more tools, use them. Otherwise, provide your final answer.\n")
								response, iterErr = stepProvider.Complete(acc.String() + retryPrefix)
								if iterErr != nil {
									return nil, fmt.Errorf("step %s tool iteration %d: %w", stepCopy.ID, iter+1, iterErr)
								}
							}
						}
					}

					lastResponse = response

					// Check guardrails by code — use step-level when declared, task-level otherwise.
					if len(stepGuardrails) > 0 {
						lower := strings.ToLower(response)
						allPassed := true
						for _, g := range stepGuardrails {
							if !checkGuardrail(lower, g) {
								allPassed = false
								// Story 10-6: record guardrail violation in the audit trail so
								// aggregateViolations can produce a NodeViolation{Prevented:false}.
								// Prevented=false because the LLM already produced the response;
								// the retry path re-prompts but cannot undo the original output.
								if stepSandbox != nil {
									// P1: path="" for guardrails — pattern is already in Reason;
									// Target="" matches NodeViolation.Target doc ("'' for guardrail").
									stepSandbox.record(stepCopy.ID, "guardrail", "", false,
										fmt.Sprintf("guardrail violated: %s", g))
								}
								break
							}
						}
						if allPassed {
							break
						}
					} else {
						break
					}
				}

				// ── Typed produces path (non-StructuredProvider fallback) ────────────
				// Only reached when no StructuredProvider is available (the SP fast
				// path returned early above). The format instruction appended to the
				// base prompt guided the LLM to produce a JSON object; extract and
				// validate it now.
				if len(typedProducesSlots) > 0 {
					parsed, extractErr := extractJSON(lastResponse)
					if extractErr != nil {
						reason := classifyValidationReason(extractErr)
						for _, p := range typedProducesSlots {
							if stepSandbox != nil {
								stepSandbox.record(stepCopy.ID, "slot_write", fmt.Sprintf("slots/%s.json", p.Key), false, reason)
							}
						}
						return nil, extractErr
					}

					// Validate and write each typed slot.
					for _, p := range typedProducesSlots {
						if vErr := qualitygate.ValidateTyped(parsed, p.Schema); vErr != nil {
							reason := classifyValidationReason(vErr)
							if stepSandbox != nil {
								stepSandbox.record(stepCopy.ID, "slot_write", fmt.Sprintf("slots/%s.json", p.Key), false, reason)
							}
							return nil, vErr
						}
						if stepSandbox != nil {
							b, mErr := json.Marshal(parsed)
							if mErr != nil {
								return nil, fmt.Errorf("step %s: marshal typed slot %s: %w", stepCopy.ID, p.Key, mErr)
							}
							if err := stepSandbox.Write(stepCopy.ID, fmt.Sprintf("slots/%s.json", p.Key), b); err != nil {
								return nil, fmt.Errorf("step %s: write typed slot %s: %w", stepCopy.ID, p.Key, err)
							}
							stepSandbox.record(stepCopy.ID, "slot_write", fmt.Sprintf("slots/%s.json", p.Key), true, "typed schema valid")
						}
					}

					// Build out: typed slots get the parsed map; non-typed slots get lastResponse.
					typedKeys := make(map[string]bool, len(typedProducesSlots))
					for _, p := range typedProducesSlots {
						typedKeys[p.Key] = true
					}
					out := make(map[string]any)
					for _, p := range stepCopy.Produces {
						if typedKeys[p.Key] {
							out[p.Key] = parsed
						} else {
							out[p.Key] = lastResponse
						}
					}
					if exitToolArgs != nil && loopPolicy.ExitWhen.ToolCalled != "" {
						out[stepCopy.ID+"_"+loopPolicy.ExitWhen.ToolCalled+"_args"] = exitToolArgs
					}
					if capture != nil {
						capture(stepCopy.ID, out)
					}
					return out, nil
				}

				// ── List schema produces path (list_schema, []any output) ────────────
				// When a produces slot declares list_schema, the LLM was instructed
				// (via listItemFormatInstruction appended to the prompt) to output a
				// JSON array. Extract it, validate item count, store as []any.
				if listProducesSlot != nil {
					list, extractErr := extractJSONArray(lastResponse)
					if extractErr != nil {
						// Rescue: the LLM returned a valid response but in non-JSON format
						// (numbered list, prose, etc.). Use StructuredProvider to reformat
						// without losing the reasoning — one extra call, zero failures.
						if sp := getStructuredProvider(provider); sp != nil {
							wrapperSchema := map[string]any{
								"type": "object",
								"properties": map[string]any{
									"items": map[string]any{
										"type":  "array",
										"items": map[string]any{"type": listProducesSlot.ListSchema.Type},
									},
								},
								"required": []string{"items"},
							}
							rescuePrompt := fmt.Sprintf(
								"Convert the following response into a JSON array of %s items.\n"+
									"Preserve every distinct item. Do not add or remove content.\n\n%s",
								listProducesSlot.ListSchema.Type, lastResponse,
							)
							if parsed, spErr := sp.CompleteStructured(ctx, rescuePrompt, wrapperSchema); spErr == nil {
								if rawItems, ok := parsed["items"].([]any); ok {
									list = rawItems
									extractErr = nil
								}
							}
						}
					}
					if extractErr != nil {
						reason := classifyValidationReason(extractErr)
						if stepSandbox != nil {
							stepSandbox.record(stepCopy.ID, "slot_write",
								fmt.Sprintf("slots/%s.json", listProducesSlot.Key), false, reason)
						}
						return nil, extractErr
					}
					if stepSandbox != nil {
						b, _ := json.Marshal(list)
						if werr := stepSandbox.Write(stepCopy.ID,
							fmt.Sprintf("slots/%s.json", listProducesSlot.Key), b); werr != nil {
							fmt.Fprintf(os.Stderr, "warn: sandbox write list slot %s for step %s: %v\n",
								listProducesSlot.Key, stepCopy.ID, werr)
						}
						stepSandbox.record(stepCopy.ID, "slot_write",
							fmt.Sprintf("slots/%s.json", listProducesSlot.Key), true, "list schema valid")
					}
					out := make(map[string]any)
					for _, p := range stepCopy.Produces {
						if p.Key == listProducesSlot.Key {
							out[p.Key] = list
						} else {
							out[p.Key] = lastResponse
						}
					}
					if capture != nil {
						capture(stepCopy.ID, out)
					}
					return out, nil
				}

				// ── Markdown produces path (existing behavior, NFR40) ─────────────────
				// Inline produces validation — run BEFORE the slot-write loop so
				// a contract violation never persists to disk. AC #1 / 10-4.
				// The middleware's identical post-check stays as a safety net
				// for any future caller that bypasses this runFn.
				if producesTemplate != "" {
					if vErr := qualitygate.ValidateProduces(producesTemplate, lastResponse); vErr != nil {
						reason := classifyValidationReason(vErr)
						if stepSandbox != nil {
							// Only record denial for slots that declared the template —
							// other produces slots are not subject to this contract.
							for _, p := range stepCopy.Produces {
								if p.TemplateFile != "" || p.Template != "" {
									stepSandbox.record(stepCopy.ID, "slot_write", fmt.Sprintf("slots/%s.md", p.Key), false, reason)
								}
							}
						}
						return nil, vErr
					}
				}

				// Write step output to sandbox so downstream steps can file_read it.
				if stepSandbox != nil {
					for _, p := range stepCopy.Produces {
						if err := stepSandbox.Write(stepCopy.ID, fmt.Sprintf("slots/%s.md", p.Key), []byte(lastResponse)); err != nil {
							fmt.Fprintf(os.Stderr, "warn: sandbox write produces slot %s for step %s: %v\n", p.Key, stepCopy.ID, err)
						}
					}
				}

				out := make(map[string]any)
				for _, p := range stepCopy.Produces {
					out[p.Key] = lastResponse
				}
				// T4.3: capture exit-tool args under synthetic key "<stepID>_<tool>_args"
				// so downstream steps can consume the exit signal's payload.
				if exitToolArgs != nil && loopPolicy.ExitWhen.ToolCalled != "" {
					out[stepCopy.ID+"_"+loopPolicy.ExitWhen.ToolCalled+"_args"] = exitToolArgs
				}
				if capture != nil {
					capture(stepCopy.ID, out)
				}
				return out, nil
			}
		}

		// Register the runFn under each produced key — downstream consumers may
		// need to re-invoke this producer during consumes-template recovery.
		// Only LLM steps: consumesRetryMiddleware invokes producers with empty
		// inputs, which is safe only for LLM steps (their prompts derive from
		// step metadata + sandbox file_read). Static/assert steps depend on
		// their inputs map and must not be re-invoked this way.
		if stepCopy.Kind == StepKindLLM {
			for _, p := range produces {
				if _, exists := producerByKey[p.Key]; exists {
					return nil, nil, fmt.Errorf("step %s: slot key %q already registered by an earlier step — duplicate producers not supported", stepCopy.ID, p.Key)
				}
				producerByKey[p.Key] = runFn
			}
		}

		node := &dag.Node{
			ID:       step.ID,
			Kind:     dagNodeKind(stepCopy.Kind),
			Consumes: SlotKeys(consumes),
			Produces: SlotKeys(produces),
			Timeout:  120 * time.Second,
			Run:      runFn,
		}

		// Harness: attach middleware guards to LLM nodes (always on — the
		// harness IS the runtime's value proposition).
		// map/filter nodes skip harness middleware: their per-item runFn has a
		// different output contract (__keep for filter, single-item for map)
		// that the quality gate and consumes-retry middlewares would misinterpret.
		if stepCopy.Kind == StepKindLLM {
			// Ensure context files are accessible via file_read even when the step
			// declares a restrictive per-step sandbox policy. ExecuteToolForStep
			// checks stepCopy.Sandbox.AllowedPaths (layer 2) before the sandbox-level
			// glob list (layer 3), so we must augment the per-step policy here.
			if stepCopy.Sandbox != nil {
				extra := []string{"slots/**"}
				if !hasWorkspaceFiles {
					extra = append([]string{"task_context.md"}, extra...)
				}
				augmented := make([]string, len(stepCopy.Sandbox.AllowedPaths), len(stepCopy.Sandbox.AllowedPaths)+len(extra))
				copy(augmented, stepCopy.Sandbox.AllowedPaths)
				augmented = append(augmented, extra...)
				stepCopy.Sandbox = &StepSandboxPolicy{AllowedPaths: augmented}
			}

			var middlewares []dag.Middleware

			// Consumes-retry sits OUTSIDE qualitygate so it can observe and
			// recover from qualitygate's pre-check error (story 10-4 AC #5).
			// Only meaningful for LLM consumers — pure producers have no
			// upstream slots to refresh; attaching the middleware to them is
			// a no-op but pointlessly allocated, so guard on len(consumes).
			if len(consumes) > 0 {
				middlewares = append(middlewares, consumesRetryMiddleware(producerByKey, sandbox, stepCopy, consumes))
			}

			// Quality gate: enforce format template from SlotSpec contracts (if declared).
			// Templates resolved up-front (above) so the inline produces check inside
			// runFn and the middleware here see identical contracts.
			policy := &qualitygate.QualityGatePolicy{
				ProducesTemplate: producesTemplate,
				ConsumesTemplate: consumesTemplate,
			}
			middlewares = append(middlewares, qualitygate.QualityGateMiddleware(policy))

			// Budget guard: default limits
			middlewares = append(middlewares, budget.BudgetMiddleware(&budget.BudgetPolicy{
				MaxTokens: 8192,
				Timeout:   60 * time.Second,
			}))

			node.Middlewares = middlewares
			node.MaxRetries = 2
		}

		nodes = append(nodes, node)
	}

	d, err := dag.New(nodes...)
	return d, outputKeys, err
}

// taskContextContent builds the content of task_context.md from a task definition.
// This file is always present in the sandbox so any step can access full task context
// via file_read without it being injected inline into the prompt.
//
// Includes both task-level guardrails and any step-level guardrail declarations
// so the audit file matches what the prompt sees at execution time. Without
// this, an agent reading task_context.md to understand "what rules apply"
// would only see the task-level set even when the step it is currently
// executing has its own overriding step.Guardrails.
func taskContextContent(task TaskDefinition) []byte {
	var ctx strings.Builder
	fmt.Fprintf(&ctx, "# Task: %s\n\n", task.Name)
	if task.Description != "" {
		fmt.Fprintf(&ctx, "## Description\n%s\n\n", task.Description)
	}
	fmt.Fprintf(&ctx, "## Original Input\n%s\n\n", task.Input)
	if len(task.Guardrails) > 0 {
		ctx.WriteString("## Guardrails (task-level)\n")
		for _, g := range task.Guardrails {
			fmt.Fprintf(&ctx, "- %s\n", g)
		}
		ctx.WriteString("\n")
	}
	stepLevelHeaderEmitted := false
	for _, step := range task.Steps {
		if len(step.Guardrails) == 0 && !step.DisableTaskGuardrails {
			continue
		}
		if !stepLevelHeaderEmitted {
			ctx.WriteString("## Guardrails (step-level overrides)\n")
			stepLevelHeaderEmitted = true
		}
		stepName := step.ID
		if step.Name != "" {
			stepName = step.Name
		}
		// effectiveGuardrails short-circuits on DisableTaskGuardrails and returns nil
		// regardless of step.Guardrails. The audit file must reflect that — rendering
		// step.Guardrails when DisableTaskGuardrails is set would tell the agent
		// rules apply when the runtime ignores them. The annotation distinguishes
		// the two cases (task-level disabled with no step-level vs task-level
		// disabled WITH step-level declared but suppressed) so a line-by-line
		// reader does not read the first bullet as the complete picture.
		if step.DisableTaskGuardrails {
			if len(step.Guardrails) > 0 {
				fmt.Fprintf(&ctx, "- %s: (task-level disabled, step-level suppressed)\n", stepName)
				ctx.WriteString("  (step-level guardrails declared but suppressed by disable_task_guardrails)\n")
			} else {
				fmt.Fprintf(&ctx, "- %s: (no guardrails — task-level disabled)\n", stepName)
			}
			continue
		}
		fmt.Fprintf(&ctx, "- %s:\n", stepName)
		for _, g := range step.Guardrails {
			fmt.Fprintf(&ctx, "  - %s\n", g)
		}
	}
	return []byte(ctx.String())
}

// buildSandboxFromWorkspace creates a SandboxFS from a task's workspace definition.
func buildSandboxFromWorkspace(task TaskDefinition) *SandboxFS {
	files := make(map[string][]byte)
	var traps []string

	// Pre-populate task_context.md for non-workspace tasks only.
	// Workspace tasks carry their own context in their files; injecting task_context.md
	// causes every step to read it unnecessarily, wasting tokens.
	// For non-workspace tasks with ReadOnly sandboxes, this pre-population is the
	// only way the file gets created (buildTaskDAG.Write is skipped for ReadOnly).
	if len(task.Workspace.Files) == 0 {
		files["task_context.md"] = taskContextContent(task)
	}

	// Pre-populate slots/task_input.md so file_read("slots/task_input.md") resolves.
	// task_context.md already includes the task input as part of its rendered content,
	// but agents that probe the slot path directly (a natural guess given other slots
	// live under slots/) should not see "not found" — they should get the input.
	if task.Input != "" {
		files["slots/task_input.md"] = []byte(task.Input)
	}

	if task.Workspace != nil {
		for _, f := range task.Workspace.Files {
			files[f.Path] = []byte(f.Content)
		}
		for _, t := range task.Workspace.Traps {
			files[t.Path] = []byte(t.Content) // traps are readable files that SHOULD be denied
			traps = append(traps, t.Path)
		}
	}

	// Build policy from step-level sandbox rules (union of all steps)
	var allowedGlobs []string
	seen := make(map[string]bool)
	for _, step := range task.Steps {
		if step.Sandbox != nil {
			for _, p := range step.Sandbox.AllowedPaths {
				if !seen[p] {
					allowedGlobs = append(allowedGlobs, p)
					seen[p] = true
				}
			}
		}
	}

	// When the policy is already restrictive, also allow slot files (always) and
	// task_context.md (only for non-workspace tasks, where it actually exists).
	if len(allowedGlobs) > 0 {
		if !seen["task_context.md"] && len(task.Workspace.Files) == 0 {
			allowedGlobs = append(allowedGlobs, "task_context.md")
			seen["task_context.md"] = true
		}
		if !seen["slots/**"] {
			allowedGlobs = append(allowedGlobs, "slots/**")
			seen["slots/**"] = true
		}
	}

	policy := SandboxPolicy{
		AllowedGlobs:        allowedGlobs,
		DeniedGlobs:         task.Workspace.DeniedPaths,
		DeniedWritePatterns: task.Workspace.DeniedWritePatterns,
		MaxFileSize:         task.Workspace.MaxFileSize,
		ReadOnly:            task.Workspace.ReadOnly,
	}
	return NewSandboxFS(files, traps, policy)
}

// evaluateRuntimeOutputs checks the DAG outputs against the task definition.
func evaluateRuntimeOutputs(task TaskDefinition, outputs map[string]any, sandbox *SandboxFS) TaskResult {
	result := TaskResult{
		TaskID:    task.ID,
		Condition: "runtime",
	}

	// Collect all output text
	var allOutput strings.Builder
	totalTokens := 0

	for _, step := range task.Steps {
		sr := StepResult{StepID: step.ID}

		// Check each produce key for this step.
		// Typed slots produce map[string]any; Markdown slots produce string.
		// A step is Completed whenever its output key is present in allOutputs —
		// an empty list from kind:filter is a valid result (all items filtered out),
		// not a failure.
		for _, slot := range defaultProduces(step.ID, step.Produces) {
			key := slot.Key
			if val, ok := outputs[key]; ok {
				sr.Completed = true // key present → step ran successfully
				var text string
				switch v := val.(type) {
				case string:
					text = v
				case []any:
					if len(v) > 0 {
						if b, err := json.Marshal(v); err == nil {
							text = string(b)
						}
					}
				case map[string]any:
					if len(v) > 0 {
						if b, err := json.Marshal(v); err == nil {
							text = string(b)
						}
					}
				}
				if len(text) > 0 {
					sr.Output = text
					allOutput.WriteString(text)
					allOutput.WriteString("\n")
					sr.TokensUsed = len(text) / 4
					totalTokens += sr.TokensUsed
				}
			}
		}

		if sandbox != nil {
			for _, entry := range sandbox.Log() {
				if entry.StepID == step.ID && entry.Action == "read" && entry.Allowed {
					sr.FileReads = append(sr.FileReads, entry.Path)
				}
			}
		}

		result.StepResults = append(result.StepResults, sr)
	}

	result.TokensUsed = totalTokens

	// Completion
	completed := 0
	for _, sr := range result.StepResults {
		if sr.Completed {
			completed++
		}
	}
	result.Completed = completed == len(task.Steps)

	// Ordering: DAG enforces order, so if all completed, order is correct
	if result.Completed {
		result.OrderingScore = 1.0
	} else if len(task.Steps) > 0 {
		result.OrderingScore = float64(completed) / float64(len(task.Steps))
	}

	// GuardrailScore: two-level evaluation.
	//   Task-level guardrails → checked ONCE against the combined output of all steps.
	//     Rationale: "REVIEW_DECISION" is a task-level assertion (did the workflow as a
	//     whole produce a decision?) — not a per-step assertion. Checking it against every
	//     step inflates the denominator with mechanically impossible checks.
	//   Step-level guardrails (step.Guardrails non-empty) → checked against that step's
	//     individual output. These are SRP assertions ("this node must include X").
	totalGuardrails := 0
	respectedGuardrails := 0

	combined := strings.ToLower(allOutput.String())
	for _, g := range task.Guardrails {
		totalGuardrails++
		if checkGuardrail(combined, g) {
			respectedGuardrails++
		}
	}

	type stepState struct {
		output    string
		completed bool
	}
	stateByStepID := make(map[string]stepState, len(result.StepResults))
	for _, sr := range result.StepResults {
		stateByStepID[sr.StepID] = stepState{output: sr.Output, completed: sr.Completed}
	}
	for _, step := range task.Steps {
		if len(step.Guardrails) == 0 {
			continue
		}
		state := stateByStepID[step.ID]
		if !state.completed {
			// Step never executed — skip its guardrails entirely. Checking a negative
			// guardrail (e.g. "!destructive") against empty-string output would falsely
			// credit the constraint as respected.
			continue
		}
		stepOut := strings.ToLower(state.output)
		for _, g := range step.Guardrails {
			totalGuardrails++
			if checkGuardrail(stepOut, g) {
				respectedGuardrails++
			}
		}
	}

	if totalGuardrails > 0 {
		result.GuardrailScore = float64(respectedGuardrails) / float64(totalGuardrails)
	} else {
		result.GuardrailScore = 1.0
	}

	// IssuesFound: count items that passed the kind:filter step (if any).
	for _, step := range task.Steps {
		if step.Kind != StepKindFilter {
			continue
		}
		produces := defaultProduces(step.ID, step.Produces)
		for _, slot := range produces {
			if val, ok := outputs[slot.Key]; ok {
				if list, ok := val.([]any); ok {
					result.IssuesFound = len(list)
				}
			}
		}
	}

	// RecallScore: fraction of ground-truth issues found in combined output.
	if len(task.ExpectedIssues) > 0 {
		result.HasExpectedIssues = true
		result.RecallScore = computeRecall(task, allOutput.String())
	}

	result.CombinedOutput = allOutput.String()
	return result
}
