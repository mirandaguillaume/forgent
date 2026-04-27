# Story 6.3: Budget Guards (Token Cap + Timeout)

Status: done

## Story

As a **platform operator**,
I want per-step token and time limits declared in guardrails,
so that a runaway LLM step cannot exhaust the API budget or hang indefinitely — enforced by code, not by prompting.

## Acceptance Criteria

1. A `BudgetPolicy` struct is defined in `pkg/budget/` containing `MaxTokens int` (output token cap) and `Timeout time.Duration` (wall-clock per-step guardrail timeout)
2. A `BudgetMiddleware` function returns a `dag.Middleware` that enforces both token cap and timeout on each step execution
3. **Token cap enforcement (API-side):** The middleware wraps the LLM provider to pass `max_tokens` to the API call, preventing the provider from generating more tokens than allowed at the source
4. **Token cap enforcement (post-hoc):** After the LLM response returns, the middleware estimates the response token count using the `len(response)/4` approximation (consistent with existing `internal/bench/experiment/` convention) and compares against `MaxTokens`. If exceeded, the response is truncated to approximately `MaxTokens * 4` characters, a `"__budget_warning"` key is added to the outputs map with the warning message, and the (truncated) outputs are returned — this is NOT an error, it is a graceful degradation
5. **Timeout enforcement:** The middleware wraps the entire `next(ctx, inputs)` call with `context.WithTimeout(ctx, policy.Timeout)`, separate from the existing `Node.Timeout` (which is a safety net at the DAG engine level). When the timeout fires, the step fails with `context.DeadlineExceeded`, and the DAG retry mechanism (`runWithRetry`) retries the entire chain if `MaxRetries > 0`
6. When no budget policy is configured (nil `BudgetPolicy`), the middleware is a no-op pass-through with zero overhead — no timeout context created, no token counting performed
7. When only `MaxTokens` is set (no `Timeout`), only token cap enforcement applies; when only `Timeout` is set (no `MaxTokens`), only timeout enforcement applies
8. The `GuardrailsFacet` in `pkg/model/skill.go` supports budget guardrail rules as map entries: `{max_tokens: 4000}` and `{timeout: "30s"}`, using the existing `GuardrailRule` map variant
9. A helper function `ParseBudgetPolicy(guardrails []GuardrailRule) *BudgetPolicy` extracts budget-related rules from the guardrails list and returns a `BudgetPolicy` (or nil if no budget rules found)
10. All existing tests pass without modification (`go test ./... -count=1 -race`)
11. The middleware works as both a pre-middleware (applies timeout context and injects `max_tokens` into inputs) and a post-middleware (checks response token count and truncates if needed)

## Tasks / Subtasks

- [ ] Task 1: Create budget policy package `pkg/budget/` (AC: #1, #9)
  - [ ] 1.1: Create `pkg/budget/policy.go` with `BudgetPolicy` struct:
    ```go
    // BudgetPolicy defines per-step budget constraints for LLM calls.
    type BudgetPolicy struct {
        MaxTokens int           // max output tokens; 0 = no limit
        Timeout   time.Duration // wall-clock timeout per step; 0 = no timeout
    }
    ```
  - [ ] 1.2: Create `pkg/budget/parse.go` with `ParseBudgetPolicy` function:
    ```go
    // ParseBudgetPolicy extracts budget-related guardrail rules from a guardrails list.
    // Recognizes map rules with keys "max_tokens" (int) and "timeout" (duration string).
    // Returns nil if no budget rules found.
    func ParseBudgetPolicy(guardrails []model.GuardrailRule) *BudgetPolicy
    ```
    Implementation:
    - Iterate over guardrails, looking for map-type rules (via `g.MapValue()`)
    - If a rule has key `"max_tokens"`, extract the int value (handle float64 from YAML unmarshal: `int(v.(float64))`)
    - If a rule has key `"timeout"`, extract the string value and parse with `time.ParseDuration`
    - A single map rule can contain both keys: `{max_tokens: 4000, timeout: "30s"}`
    - If neither key is found across all rules, return nil
    - If parsing fails (invalid duration string, non-numeric max_tokens), ignore the malformed rule and continue (defensive — do not block execution over a config parse error)
  - [ ] 1.3: Create `pkg/budget/estimate.go` with token estimation:
    ```go
    // EstimateTokens estimates the number of tokens in a string using the
    // char/4 approximation. This is consistent with the project convention
    // used in internal/bench/experiment/ (see text_runner.go:123, runtime_runner.go:427).
    func EstimateTokens(s string) int {
        return len(s) / 4
    }

    // TruncateToTokens truncates a string to approximately maxTokens tokens.
    // Returns the truncated string and true if truncation occurred.
    func TruncateToTokens(s string, maxTokens int) (string, bool) {
        maxChars := maxTokens * 4
        if len(s) <= maxChars {
            return s, false
        }
        return s[:maxChars], true
    }
    ```

- [ ] Task 2: Unit tests for budget policy in `pkg/budget/policy_test.go` (AC: #1, #9)
  - [ ] 2.1: `TestParseBudgetPolicy_MaxTokensOnly` — guardrails with `{max_tokens: 4000}`; returns policy with MaxTokens=4000, Timeout=0
  - [ ] 2.2: `TestParseBudgetPolicy_TimeoutOnly` — guardrails with `{timeout: "30s"}`; returns policy with MaxTokens=0, Timeout=30s
  - [ ] 2.3: `TestParseBudgetPolicy_Both` — guardrails with `{max_tokens: 4000, timeout: "30s"}`; returns policy with both set
  - [ ] 2.4: `TestParseBudgetPolicy_SeparateRules` — max_tokens in one rule, timeout in another; both values present in returned policy
  - [ ] 2.5: `TestParseBudgetPolicy_NoBudgetRules` — guardrails with only string rules and non-budget map rules; returns nil
  - [ ] 2.6: `TestParseBudgetPolicy_EmptyGuardrails` — empty slice returns nil
  - [ ] 2.7: `TestParseBudgetPolicy_NilGuardrails` — nil slice returns nil
  - [ ] 2.8: `TestParseBudgetPolicy_InvalidTimeout` — guardrails with `{timeout: "not-a-duration"}`; malformed rule ignored, returns nil (or partial policy if max_tokens also present)
  - [ ] 2.9: `TestParseBudgetPolicy_InvalidMaxTokens` — guardrails with `{max_tokens: "not-a-number"}`; malformed rule ignored
  - [ ] 2.10: `TestParseBudgetPolicy_MixedWithStringRules` — guardrails containing both string rules ("no PII in output") and budget map rules; correctly extracts budget policy while ignoring string rules
  - [ ] 2.11: `TestEstimateTokens_Basic` — "hello world" (11 chars) = 2 tokens
  - [ ] 2.12: `TestEstimateTokens_Empty` — empty string = 0 tokens
  - [ ] 2.13: `TestTruncateToTokens_NoTruncation` — short string within limit returns unchanged, false
  - [ ] 2.14: `TestTruncateToTokens_Truncates` — long string exceeding limit returns truncated string, true

- [ ] Task 3: Create `BudgetMiddleware` in `pkg/budget/middleware.go` (AC: #2, #3, #4, #5, #6, #7, #11)
  - [ ] 3.1: Create `pkg/budget/middleware.go` that imports `pkg/dag` for the `Middleware` and `MiddlewareFunc` types (from Story 6-1). Define:
    ```go
    // BudgetMiddleware returns a dag.Middleware that enforces token cap and timeout
    // constraints on LLM step execution.
    //
    // Token cap: injects "__max_tokens" into inputs (for the handler to pass to the
    // LLM provider), AND post-hoc checks the response token count. If exceeded,
    // truncates the response and adds "__budget_warning" to outputs.
    //
    // Timeout: wraps the next() call with context.WithTimeout. This is a guardrail-level
    // timeout, separate from Node.Timeout (which is a DAG-engine safety net).
    //
    // If policy is nil, returns a no-op middleware.
    func BudgetMiddleware(policy *BudgetPolicy) dag.Middleware
    ```
  - [ ] 3.2: Implementation of `BudgetMiddleware`:
    - **Nil policy:** Return pass-through: `func(ctx, inputs, next) { return next(ctx, inputs) }` — zero overhead
    - **Pre-middleware (token cap):** If `policy.MaxTokens > 0`, inject `"__max_tokens"` key into the inputs map with the int value. The handler (which wraps an LLM call) is expected to read this key and pass it to the provider's `max_tokens` parameter. The handler is NOT required to honor this — it's a hint. The post-hoc check is the enforcement mechanism.
    - **Pre-middleware (timeout):** If `policy.Timeout > 0`, wrap the context: `ctx, cancel := context.WithTimeout(ctx, policy.Timeout); defer cancel()`. Pass the timeout context to `next(ctx, inputs)`. When the timeout fires, `next` returns `context.DeadlineExceeded`, which propagates up to `runWithRetry` for potential retry.
    - **Call next:** `outputs, err := next(ctx, inputs)`
    - **Post-middleware (token check):** If `policy.MaxTokens > 0` and `err == nil`, iterate over outputs looking for string values. For each string value, call `EstimateTokens(s)`. If ANY string value exceeds `MaxTokens`, truncate it via `TruncateToTokens` and add `"__budget_warning"` to outputs: `fmt.Sprintf("output truncated: estimated %d tokens exceeds cap of %d", estimated, policy.MaxTokens)`. Replace the original string value with the truncated version. This is a warning, not an error — the middleware returns the truncated outputs successfully.
    - **Return:** Return the (possibly modified) outputs and error.
  - [ ] 3.3: Define the input key constant:
    ```go
    const (
        // InputKeyMaxTokens is the key injected into inputs to signal the handler
        // to limit LLM output tokens. Handlers that wrap LLM providers should read
        // this key and pass it to the provider's max_tokens parameter.
        InputKeyMaxTokens = "__max_tokens"

        // OutputKeyBudgetWarning is the key added to outputs when token truncation occurs.
        OutputKeyBudgetWarning = "__budget_warning"
    )
    ```

- [ ] Task 4: Unit tests for `BudgetMiddleware` in `pkg/budget/middleware_test.go` (AC: #2, #3, #4, #5, #6, #7, #11)
  - [ ] 4.1: `TestBudgetMiddleware_NilPolicy` — nil policy passes through; handler is called; outputs returned unchanged; no `__max_tokens` in inputs, no `__budget_warning` in outputs
  - [ ] 4.2: `TestBudgetMiddleware_MaxTokens_InjectsInputKey` — policy with MaxTokens=4000; verify `__max_tokens` key is present in inputs seen by the handler, with value 4000
  - [ ] 4.3: `TestBudgetMiddleware_MaxTokens_PostCheckUnderLimit` — handler returns 100-char string (25 tokens); MaxTokens=1000; no truncation, no warning in outputs
  - [ ] 4.4: `TestBudgetMiddleware_MaxTokens_PostCheckOverLimit` — handler returns 20000-char string (5000 tokens); MaxTokens=1000; string truncated to 4000 chars, `__budget_warning` key present in outputs with descriptive message
  - [ ] 4.5: `TestBudgetMiddleware_MaxTokens_MultipleStringOutputs` — handler returns 3 string outputs, one exceeds limit; only the exceeding output is truncated, others unchanged
  - [ ] 4.6: `TestBudgetMiddleware_MaxTokens_NonStringOutputsIgnored` — handler returns int and bool outputs; no truncation logic applied (only strings are checked)
  - [ ] 4.7: `TestBudgetMiddleware_Timeout_WithinLimit` — policy with Timeout=1s; handler completes in 10ms; outputs returned normally
  - [ ] 4.8: `TestBudgetMiddleware_Timeout_Exceeded` — policy with Timeout=50ms; handler sleeps 200ms; returns context.DeadlineExceeded error
  - [ ] 4.9: `TestBudgetMiddleware_Timeout_ContextCancelled` — parent context cancelled; timeout middleware propagates cancellation
  - [ ] 4.10: `TestBudgetMiddleware_BothMaxTokensAndTimeout` — policy with both set; handler completes within timeout and under token limit; verify both pre-middleware effects (inputs key + timeout context) and no post-middleware modifications
  - [ ] 4.11: `TestBudgetMiddleware_TimeoutOnly_NoTokenCheck` — policy with Timeout=1s, MaxTokens=0; handler returns long string; no truncation, no warning
  - [ ] 4.12: `TestBudgetMiddleware_MaxTokensOnly_NoTimeout` — policy with MaxTokens=1000, Timeout=0; handler sleeps 100ms; no timeout cancellation
  - [ ] 4.13: `TestBudgetMiddleware_HandlerError_NoPostCheck` — handler returns error; post-middleware token check is skipped (no truncation on error)
  - [ ] 4.14: `TestBudgetMiddleware_TruncationPreservesOtherOutputs` — handler returns string and non-string outputs; truncation only modifies the over-limit string; other outputs are preserved exactly
  - [ ] 4.15: `TestBudgetMiddleware_WarningMessageDescriptive` — verify `__budget_warning` message contains: estimated token count, cap value

- [ ] Task 5: Integration tests with DAG execution in `pkg/budget/middleware_integration_test.go` (AC: #2, #5, #10)
  - [ ] 5.1: `TestBudgetMiddleware_DAG_TimeoutTriggersRetry` — single-node DAG with BudgetMiddleware (Timeout=50ms) and MaxRetries=1; handler sleeps 200ms on first call, returns instantly on second; verify handler called twice (retry triggered), DAG succeeds
  - [ ] 5.2: `TestBudgetMiddleware_DAG_TimeoutRetryExhausted` — single-node DAG with BudgetMiddleware (Timeout=50ms) and MaxRetries=1; handler always sleeps 200ms; DAG fails with deadline error after 2 attempts
  - [ ] 5.3: `TestBudgetMiddleware_DAG_TokenTruncationFlowsDownstream` — 2-node pipeline; first node has BudgetMiddleware with MaxTokens=100 and handler returning 2000-char string; second node consumes the first node's output; verify second node sees truncated string (400 chars) and `__budget_warning` is accessible
  - [ ] 5.4: `TestBudgetMiddleware_DAG_NoBudgetPolicyUnchanged` — single-node DAG with BudgetMiddleware(nil); verify outputs identical to DAG without middleware
  - [ ] 5.5: `TestBudgetMiddleware_DAG_WithFileAccessMiddleware` — single-node DAG with both FileAccessMiddleware (from 6-2) and BudgetMiddleware chained; verify both middlewares apply correctly (budget timeout + file access check); FileAccessMiddleware is outermost (first to see inputs), BudgetMiddleware is innermost (closest to handler)
    - NOTE: This test can only be written after Story 6-2 is merged. If 6-2 is not yet available, skip this test and add a TODO comment.
  - [ ] 5.6: `TestBudgetMiddleware_DAG_ConcurrentNodes` — 2 parallel nodes (same layer) with different BudgetPolicy values; verify no race conditions (run with `-race`)

- [ ] Task 6: Verify guardrail YAML parsing for budget rules in `pkg/budget/parse_test.go` (AC: #8, #9)
  - [ ] 6.1: `TestParseBudgetPolicy_FromRealYAML` — unmarshal a skill YAML with guardrails containing budget rules:
    ```yaml
    guardrails:
      - "no PII in output"
      - max_tokens: 4000
      - timeout: "30s"
      - "respect code formatting"
    ```
    Parse the `GuardrailsFacet`, call `ParseBudgetPolicy`, verify MaxTokens=4000 and Timeout=30s
  - [ ] 6.2: `TestParseBudgetPolicy_FromCombinedMapRule` — YAML with combined map:
    ```yaml
    guardrails:
      - max_tokens: 4000
        timeout: "30s"
    ```
    Verify both values parsed from a single map rule
  - [ ] 6.3: `TestParseBudgetPolicy_FromRealYAML_NoBudget` — YAML with only string guardrails; returns nil policy

- [ ] Task 7: Verify all existing tests pass (AC: #10)
  - [ ] 7.1: Run `go test ./pkg/budget/... -count=1 -race` — all new tests pass
  - [ ] 7.2: Run `go test ./pkg/dag/... -count=1 -race` — existing DAG tests unaffected
  - [ ] 7.3: Run `go test ./pkg/model/... -count=1 -race` — existing model tests unaffected
  - [ ] 7.4: Run `go test ./... -count=1 -race` — full suite passes

## Dev Notes

### Architecture References

- **ADR-1 (architecture.md):** Doctor embedding strategy -- `BuildDAG()` in `internal/doctor/orchestrator.go` constructs the DAG. Budget middleware will be applied to doctor nodes when doctor's skill YAMLs declare budget guardrails (future story or when orchestrator reads guardrails from embedded YAML).
- **ADR-4 (architecture.md):** Doctor DAG topology -- 4 nodes across 3 layers. Budget middleware can be attached to any LLM-calling node via the `Middlewares` field (from Story 6-1).
- **Epic 6 rationale (epics.md):** "Experiment results show text beats runtime on guardrails without programmatic enforcement." Budget guards are the second enforcement mechanism (after file access control in 6-2). They enforce cost/time constraints that text instructions cannot enforce.
- **Epic 5 retro lesson #1:** "Framework != Evidence." This story builds a real enforcement mechanism. When a handler exceeds its token budget, the middleware truncates. When it exceeds its timeout, the context is cancelled. These produce measurable outcomes for Story 6-6.

### Key Design Decisions

**1. Two independent guards, not one**

Token cap and timeout are separate concerns:
- Token cap prevents cost overrun (API bills scale with tokens)
- Timeout prevents hanging (network issues, model stalling, infinite generation)

A step can exceed its token cap without timing out (fast but verbose). A step can time out without exceeding its token cap (slow but concise). The middleware handles each independently.

**2. Token cap is enforced at two levels**

- **API-side:** The middleware injects `"__max_tokens"` into the inputs map. The handler (which wraps the LLM provider) is expected to read this and pass it to the API's `max_tokens` parameter. This is the primary enforcement — it prevents the API from generating excess tokens in the first place. The existing providers (`internal/llm/anthropic.go`, `ollama.go`, `openrouter.go`) all have a `max_tokens`/`num_predict` parameter in their request structs.
- **Post-hoc:** After the response returns, the middleware estimates the response token count and truncates if over budget. This is a safety net for two cases: (a) the handler does not honor `__max_tokens`, (b) the API's own token counting differs from ours. The post-hoc check uses `len(s)/4` — the same approximation used throughout `internal/bench/experiment/` (text_runner.go:123, runtime_runner.go:427).

The two levels are complementary, not redundant. The API-side limit is efficient (stops generation early). The post-hoc check is defensive (catches edge cases).

**3. Truncation is a warning, not an error**

When the post-hoc check finds excess tokens, the middleware truncates the response and adds `"__budget_warning"` to outputs. It does NOT return an error. Rationale:
- The API already received the call — the cost is spent
- Truncated output may still be useful to downstream nodes
- Returning an error would trigger retry, which would spend MORE tokens — the opposite of budget protection
- The warning key allows downstream nodes (or quality gate middleware from Story 6-4) to detect and act on truncation

**4. Timeout is a guardrail-level timeout, separate from Node.Timeout**

The DAG engine already has `Node.Timeout` (set per-node in `pkg/dag/dag.go:27`). This story adds a guardrail-level timeout that is typically SHORTER than the node timeout:
- `Node.Timeout`: safety net at the DAG engine level. Catches catastrophic hangs. Set to generous values (e.g., 120s).
- `BudgetPolicy.Timeout`: guardrail enforcement. Enforces per-step time limits declared in skill YAML. Set to operational values (e.g., 30s).

Both use `context.WithTimeout`. The middleware timeout is applied INSIDE the middleware chain, so it nests within the node timeout. If both are set, the shorter one wins (standard Go context behavior). Example: Node.Timeout=120s, BudgetPolicy.Timeout=30s → step is cancelled after 30s.

**5. The middleware does NOT modify the Provider interface**

The `internal/llm/Provider` interface is `Complete(prompt string) (string, error)` — it does not accept `max_tokens`. This story does NOT change that interface. Instead:
- The middleware injects `"__max_tokens"` into the DAG inputs map
- The handler (the node's `Run` function) is responsible for reading `"__max_tokens"` and passing it to the provider
- Providers that support `max_tokens` already have it in their request structs (anthropic.go:46 `MaxTokens`, openrouter.go:30 `MaxTokens`, ollama.go:40 `NumPredict`)
- How the handler passes `max_tokens` to the provider is the handler's concern — it may use a context-aware provider, a wrapper, or a custom call. The middleware just signals the desired limit.

This keeps the middleware decoupled from the provider implementation. A future story could add `CompleteWithOptions(prompt string, opts Options) (string, error)` to the Provider interface if needed.

**6. Package location: `pkg/budget/` (public)**

Like `pkg/sandbox/` for file access, the budget package is public because:
- It is consumed by `pkg/dag` middleware (public API)
- Skill authors building custom runtimes may construct `BudgetPolicy` directly
- The API is small and stable: `BudgetPolicy`, `BudgetMiddleware`, `ParseBudgetPolicy`, `EstimateTokens`

**7. Guardrail YAML uses existing GuardrailRule map variant**

The `GuardrailsFacet` is `[]GuardrailRule`, where each rule can be a string or a map. Budget rules use the map variant:
```yaml
guardrails:
  - "no PII in output"           # string rule (not budget)
  - max_tokens: 4000             # map rule (budget)
  - timeout: "30s"               # map rule (budget)
```
This requires NO changes to the `GuardrailRule` type or YAML parsing — it already supports both variants. `ParseBudgetPolicy` simply filters for map rules with the `max_tokens` or `timeout` keys.

**8. Zero overhead when no budget guards configured**

The check `if policy == nil` in `BudgetMiddleware` returns an immediate pass-through. When the middleware is attached to a node with nil policy (or not attached at all), there are zero additional allocations, no timeout context creation, and no token counting. This preserves exact current behavior for existing nodes.

### Dependencies

- **Story 6-1 (middleware pipeline):** MUST be completed first. This story depends on `dag.Middleware`, `dag.MiddlewareFunc`, and the `Middlewares []Middleware` field on `Node`. If Story 6-1 is not yet merged, the middleware code (`pkg/budget/middleware.go` and its tests) cannot compile.
- **Story 6-2 (file access control):** NOT a hard dependency. Test 5.5 (combined middleware test) can be skipped if 6-2 is not merged yet. The budget middleware is independent of file access middleware — they can be composed on the same node but do not interact.
- **Upstream:** `pkg/dag` (Story 6-1 adds middleware types), `pkg/model` (existing `GuardrailRule` type)
- **Downstream stories that consume this:**
  - Story 6-4: Quality Gate Inline — may inspect `__budget_warning` to decide whether to retry
  - Story 6-6: Harness Experiment — will construct `BudgetPolicy` from task definitions and attach `BudgetMiddleware` to DAG nodes; will measure token usage and timeout enforcement rates
  - Future: Doctor DAG can declare budget guardrails in its skill YAML and get automatic enforcement

### What NOT to Build

- Do NOT modify the `Provider` interface (`internal/llm/provider.go`) — the middleware signals `max_tokens` via the inputs map, not by changing the provider API
- Do NOT modify any provider implementation (`anthropic.go`, `ollama.go`, `openrouter.go`) — the handler (not the middleware) is responsible for reading `__max_tokens` and passing it to the provider
- Do NOT modify `internal/doctor/orchestrator.go` to add budget middleware to doctor nodes — that is a future concern when doctor's skill YAMLs declare budget guardrails
- Do NOT modify `internal/bench/experiment/runtime_runner.go` — experiment integration with the new middleware comes in Story 6-6
- Do NOT implement actual tokenizer-based counting — the char/4 approximation is consistent with the existing codebase and sufficient for budget guardrails (within ~20% of actual token count for English text)
- Do NOT add cumulative budget tracking (total tokens across all steps) — this story is per-step only. Cumulative budget is a future concern
- Do NOT add cost estimation in the middleware — cost depends on model pricing, which is a higher-level concern (the bench package already has `EstimateCost` in `internal/bench/h2h.go:395`)
- Do NOT add logging/observability to the middleware — observability middleware is a separate concern (deferred)
- Do NOT modify the `SkillBehavior` struct or `GuardrailsFacet` type — the existing `[]GuardrailRule` already supports map entries with `max_tokens` and `timeout` keys

### Project Structure

New files:
- `pkg/budget/policy.go` — `BudgetPolicy` struct
- `pkg/budget/parse.go` — `ParseBudgetPolicy` function
- `pkg/budget/estimate.go` — `EstimateTokens`, `TruncateToTokens` functions
- `pkg/budget/middleware.go` — `BudgetMiddleware` function (returns `dag.Middleware`)
- `pkg/budget/policy_test.go` — parse + estimate unit tests
- `pkg/budget/middleware_test.go` — middleware unit tests
- `pkg/budget/middleware_integration_test.go` — DAG integration tests
- `pkg/budget/parse_test.go` — YAML round-trip integration tests

Modified files:
- None. This story is entirely additive — no existing files are modified.

### Existing Code Reference

**`Provider` interface (internal/llm/provider.go:4-6):**
```go
type Provider interface {
    Complete(prompt string) (string, error)
}
```

**Anthropic request struct with max_tokens (internal/llm/anthropic.go:43-47):**
```go
type anthropicRequest struct {
    Model     string             `json:"model"`
    MaxTokens int                `json:"max_tokens"`
    Messages  []anthropicMessage `json:"messages"`
}
```
Current default: `anthropicMaxTok = 8192` (line 18).

**OpenRouter request struct with max_tokens (internal/llm/openrouter.go:29-33):**
```go
type openRouterRequest struct {
    Model     string              `json:"model"`
    MaxTokens int                 `json:"max_tokens"`
    Messages  []openRouterMessage `json:"messages"`
}
```

**Ollama request struct with num_predict (internal/llm/ollama.go:38-42):**
```go
type ollamaOptions struct {
    Temperature float64 `json:"temperature"`
    NumPredict  int     `json:"num_predict"`
}
```

**Existing token estimation pattern (internal/bench/experiment/text_runner.go:123):**
```go
TokensUsed: totalTokens / 4, // rough token estimate: 4 chars/token
```

**Existing token estimation pattern (internal/bench/experiment/runtime_runner.go:427):**
```go
sr.TokensUsed = len(s) / 4
```

**Existing methodology documentation (internal/bench/experiment/METHODOLOGY.md:34):**
```
| Token usage | Total tokens consumed | Estimate from character count (4 chars/token) |
```

**GuardrailRule type (pkg/model/skill.go:89-136):**
```go
type GuardrailRule struct {
    stringVal *string
    mapVal    map[string]interface{}
}

func (g GuardrailRule) MapValue() (map[string]interface{}, bool) {
    if g.mapVal != nil {
        return g.mapVal, true
    }
    return nil, false
}

func (g GuardrailRule) HasKey(key string) bool {
    if g.mapVal == nil {
        return false
    }
    _, ok := g.mapVal[key]
    return ok
}
```

**GuardrailsFacet type (pkg/model/skill.go:167):**
```go
type GuardrailsFacet = []GuardrailRule
```

**Middleware types from Story 6-1 (to be created in pkg/dag/middleware.go):**
```go
type MiddlewareFunc func(ctx context.Context, inputs map[string]any) (map[string]any, error)
type Middleware func(ctx context.Context, inputs map[string]any, next MiddlewareFunc) (map[string]any, error)
```

**Node struct after Story 6-1 (pkg/dag/dag.go -- with Middlewares field added):**
```go
type Node struct {
    ID         string
    Kind       NodeKind
    Consumes   []string
    Produces   []string
    Run        func(ctx context.Context, inputs map[string]any) (map[string]any, error)
    MaxRetries int
    Timeout    time.Duration
    Config     *NodeConfig
    Middlewares []Middleware  // added by Story 6-1
}
```

**runWithRetry after Story 6-1 (pkg/dag/execute.go -- with middleware chain):**
```go
func runWithRetry(ctx context.Context, n *Node, inputs map[string]any) (map[string]any, error) {
    if n.Run == nil {
        return map[string]any{}, nil
    }

    handler := MiddlewareFunc(n.Run)
    if len(n.Middlewares) > 0 {
        handler = buildChain(handler, n.Middlewares)
    }

    attempt := func() (map[string]any, error) {
        runCtx := ctx
        var cancel context.CancelFunc
        if n.Timeout > 0 {
            runCtx, cancel = context.WithTimeout(ctx, n.Timeout)
            defer cancel()
        }
        return handler(runCtx, inputs)
    }

    out, err := attempt()
    for i := 0; i < n.MaxRetries && err != nil; i++ {
        if ctx.Err() != nil {
            return nil, ctx.Err()
        }
        out, err = attempt()
    }
    return out, err
}
```

**YAML example for skill with budget guardrails:**
```yaml
skill: code-analyzer
version: "1.0"
context:
  consumes: [source_code]
  produces: [analysis_report]
strategy:
  tools: [file_read]
  approach: llm
guardrails:
  - "focus on security vulnerabilities"
  - max_tokens: 4000
  - timeout: "30s"
  - "include remediation steps"
security:
  filesystem: read-only
```

### Implementation Order

Tasks MUST be executed in this order due to dependencies:

1. **Task 1** (pkg/budget/ policy + parse + estimate) — foundation; no dependencies
2. **Task 2** (unit tests for policy/parse/estimate) — depends on Task 1
3. **Task 3** (middleware) — depends on Task 1 AND Story 6-1 being merged
4. **Task 4** (middleware unit tests) — depends on Task 3
5. **Task 5** (integration tests) — depends on Task 3; test 5.5 depends on Story 6-2
6. **Task 6** (YAML round-trip tests) — depends on Task 1
7. **Task 7** (full regression) — depends on all above

If Story 6-1 is not yet merged when development starts, Tasks 1-2 and Task 6 can be implemented and tested independently. Tasks 3-5 must wait for the `dag.Middleware` type to exist.

## Testing Strategy

**Unit tests (Tasks 2, 4):** Pure function tests for policy parsing, token estimation, and middleware behavior in isolation. No DAG construction. Test the budget enforcement logic comprehensively with table-driven tests.

**Integration tests (Task 5):** Full DAG construction and execution with `BudgetMiddleware` configured on nodes. Tests verify that:
- Timeout enforcement triggers DAG retry mechanism correctly
- Token truncation flows through the pipeline to downstream nodes
- No-policy middleware is identical to no middleware (zero overhead)
- Combined middleware composition works (budget + file access)
- Concurrent nodes with different policies don't race

**YAML round-trip tests (Task 6):** Verify that budget guardrail rules survive YAML unmarshal and `ParseBudgetPolicy` extraction. Tests use real YAML strings, not hand-constructed structs.

**Regression tests (Task 7):** All existing tests must pass unchanged. Run full suite with `-race`.

**Test patterns to follow:**
- `require` for preconditions, `assert` for values (testify convention from project)
- Table-driven tests for policy parsing (many cases, compact format)
- Atomic counters for verifying handler invocation counts (retry tests)
- `time.Sleep` in handler for timeout tests — use short durations (50ms/200ms) to keep tests fast
- Section headers with ASCII art comments (project convention: `// --- Section ---`)

## References

- [Source: _bmad-output/planning-artifacts/epics.md#Story6.3] — Story acceptance criteria from epic
- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-1] — Embedded YAML + runtime DAG
- [Source: _bmad-output/implementation-artifacts/6-1-middleware-pipeline.md] — Middleware type signatures, integration point
- [Source: _bmad-output/implementation-artifacts/6-2-file-access-control.md] — Sibling story pattern, shared conventions
- [Source: internal/llm/provider.go] — Provider interface (Complete)
- [Source: internal/llm/anthropic.go] — max_tokens parameter in API request
- [Source: internal/llm/openrouter.go] — max_tokens parameter in API request
- [Source: internal/llm/ollama.go] — num_predict parameter in API request
- [Source: internal/bench/experiment/text_runner.go:123] — Token estimation: `totalTokens / 4`
- [Source: internal/bench/experiment/runtime_runner.go:427] — Token estimation: `len(s) / 4`
- [Source: internal/bench/experiment/METHODOLOGY.md:34] — Token estimation methodology
- [Source: pkg/model/skill.go] — GuardrailRule, GuardrailsFacet, SkillBehavior
- [Source: pkg/dag/dag.go] — Node struct, DAG construction
- [Source: pkg/dag/execute.go] — runWithRetry, middleware integration point
- [Source: pkg/dag/node_kinds.go] — RunFunc, NodeConfig
- [Source: _bmad-output/implementation-artifacts/epic-5-retro-2026-04-15.md] — "Framework != Evidence" lesson

## Dev Agent Record

### Agent Model Used

(to be filled by dev agent)

### Debug Log References

(to be filled by dev agent)

### Completion Notes List

(to be filled by dev agent)

### File List

(to be filled by dev agent)
