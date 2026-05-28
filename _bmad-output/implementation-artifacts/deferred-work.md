# Deferred Work

## Deferred from: code review of 10-10-typed-node-output-schema (2026-04-29)

- `getStructuredProvider` only unwraps `callCountingProvider` — any deeper wrapper (future telemetry, budget) hides `StructuredProvider` capability, silently downgrades to text extraction [internal/bench/experiment/runtime_runner.go:160]
- `callCountingProvider` call count not incremented for `CompleteStructured` calls — cost telemetry under-counts typed-produce steps when constrained decoding is used [internal/bench/experiment/runtime_runner.go:162]
- `ToJSONSchema` lacks `additionalProperties: false` — required by many strict-mode providers (OpenAI, Outlines); without it providers may reject schema or allow unconstrained extra fields [pkg/schema/compile.go:34]
- Retrievers (`PrefixRetriever`, `RankedRetriever`) only read `.md` slot files — typed `.json` slots silently skipped, context injection disabled for typed slot lineages [internal/bench/experiment/retriever.go]
- `StructuredProvider` failure is fatal with no text-extraction fallback — transient API error causes hard step failure rather than graceful degradation; deliberate design choice, may reconsider when real providers are wired [internal/bench/experiment/runtime_runner.go:871]
- No real provider implements `StructuredProvider` — structured path exercised by test mocks only; implementation for Anthropic/Ollama tracked as future work [internal/llm/]
- Lint rule for `schema` (inter-node) vs `template_file` (human-facing) not implemented in `forgent lint` — AC10 defers this to lint epic
- `ValidateTyped` iterates schema map in Go map order (non-deterministic) — per-field violation reason in telemetry may vary across runs; validation pass/fail is deterministic [pkg/qualitygate/typed.go:13]
- `consumesRetryMiddleware` passes empty inputs to typed producers on recovery — re-produced typed value may diverge from original; pre-existing design, documented in code comment [internal/bench/experiment/runtime_runner.go:95]
- `extractJSON` brace scan uses first `{` in string — fails when preamble text contains `{...}` before actual JSON; rare in practice [internal/bench/experiment/json_extract.go:47]
- Retry prefix not passed to `CompleteStructured` call — DAG `runWithRetry` re-invokes whole runFn; no failure context (which field, which constraint) reaches the structured provider on retry [internal/bench/experiment/runtime_runner.go:870]
- `extractJSON` has no input size limit — large LLM responses scanned char-by-char; DoS protection belongs at budget layer not extraction layer [internal/bench/experiment/json_extract.go:17]

## Deferred from: code review of 10-8-per-node-experiment-metrics (2026-04-29)

- Theory A/B table emits `n/a` rows for every node in a condition even when that node never observed a given violation type — produces sparse, cluttered tables; cosmetic UX issue [internal/bench/experiment/report.go:538-549]
- `collect(ranked, "ranked")` iterates a nil slice without an explicit nil check — safe no-op in Go but lacks defensive comment; brittle if surrounding code is refactored [internal/bench/experiment/compare.go:81]
- `formatPerNodeSection` writes warning to `os.Stderr` from inside a string-builder formatter — side-effect makes the function untestable without capturing stderr; design smell [internal/bench/experiment/report.go:379-384]
- `SaveBaseline` stores only ISO date in `CapturedAt`, losing sub-day granularity — two baselines captured on the same day are indistinguishable by timestamp [internal/bench/experiment/baseline.go:55]

## Deferred from: code review of 10-12-swebench-validation (2026-05-13)

- `swebench.go` bufio.Scanner 64KB default buffer exceeded by large JSONL lines — task list silently truncated mid-file [internal/bench/swebench.go:55-66]
- `swebench.go` `--depth 1` clone misses `base_commit` when not HEAD of default branch — all tasks produce wrong diff base silently [internal/bench/swebench.go:113]
- `swebench.go` search/replace edge cases — empty search block always matches at line 0; empty replace block produces malformed `+N,0` hunk; both fail `git apply` silently [internal/bench/swebench.go:368-400]
- `swebench.go` `parseHarnessResults` returns empty map with nil error when no JSON file matches glob — all tasks marked unresolved with no indicator harness output was missing [internal/bench/swebench.go:914-983]
- `swebench.go` `extractTagContent` returns garbage substring when closeTag precedes openTag in output [internal/bench/swebench.go:269-278]
- AC2 default conditions = text,runtime only — user must pass `--conditions text,runtime,specialists` for 3-condition run; no smoke-test guard enforces this [bench.go CLI]
- AC3 `--swe-size lite` maps to `swebench_lite_30.jsonl` (30 tasks, not 300) — blocked on API budget; bootstrap CI n=30 is too small for 95% CI claim [internal/bench/cmd/bench.go]
- `extractUnifiedDiffFromOutput` brittle for multi-file diffs with interleaved LLM prose — `isDiffLine` heuristic cannot distinguish diff lines from commentary sharing the same character prefix [internal/bench/swebench_experiment.go:319-338]

## Deferred from: code review of 10-11-markdown-ast-output-validation (2026-04-29)

- `extractOutputSection` only walks top-level document children — headings inside blockquotes or HTML blocks are invisible; real-world LLM outputs rarely use this pattern but it is a known gap [pkg/qualitygate/ast.go:81]
- `classifyValidationReason` uses fragile string matching on human-readable error messages instead of typed sentinel errors; pin tests in `output_contract_test.go` currently guard the wording but refactoring away from `errors.As` is blocked by the flat error model in `validateASTStructure` [internal/bench/experiment/runtime_runner.go:126]
- Budget middleware truncation and qualitygate inline check see inconsistent data: the inline `ValidateProduces` call inside `runFn` runs on `lastResponse` (pre-truncation), the middleware post-check runs on budget-truncated output; a structurally valid but oversized LLM response will loop until retries are exhausted with no actionable error — cross-story concern (budget + qualitygate interaction)



## Triage 2026-04-27 (Epic 9 retro action #3)

**Status:** Full triage done. See `deferred-work-triage-2026-04-27.md` for the per-item table.

| Bucket | Count | Meaning |
|---|---|---|
| **A** | 33 | Epic 10 prerequisite — fix before/during stories 10-3..10-9 |
| **B** | 60 | Post-MVP — real but not urgent |
| **C** | 16 | Won't fix — close (intentional trade-off, resolved, or accepted risk) |

**Top 5 Bucket A items (Epic 10 critical path):**
1. TOCTOU lock gap in `SandboxFS.Read` (story 6-21) — blocks 10-3 (file-level permissions)
2. `TaskDir` unset in 15+ test constructions (10-2) — falsifies 10-4 (output contract validation) metrics
3. `provErr` swallows + `executeLayerBased` single-error drain (4-2) — blocks 10-6 (violation telemetry)
4. Tool-call loop `toolPrompt` reset + unbounded growth (6-21) — blocks 10-7 (loop bounds)
5. AC4/AC5 16-task experiment rerun (6-21) — IS story 10-9, fuse not a separate fix

**Items already addressed in Epic 9 retro action #1 (swallow & continue audit, 2026-04-27):**
- `_ = sandbox.Write(...)` ×3 in `runtime_runner.go` → log to stderr
- `_ = cpWriter.Append(...)` ×3 in `bench.go` → log to stderr
- `_ = os.WriteFile(...)` ×3 (flush snapshots) in `bench.go` → log to stderr
- `_ = os.WriteFile(go.mod/go.sum)` in `forgent.go` generator → log to stderr
- `_ = json.Unmarshal(report)` in `appendH2HReport` → log on corrupt + rebuild

**Quick wins applied 2026-04-27 (Epic 9 retro post-triage):**
- ✅ TOCTOU `SandboxFS.Read` (6-21) — single exclusive `Lock` for entire policy check + read in `internal/bench/experiment/sandbox.go`
- ✅ `executeLayerBased` + `executeEventDriven` only drained first error (4-2) — drain all errors and return `errors.Join(errs...)` in `pkg/dag/execute.go`
- ✅ `provErr` silently swallows LLM unavailability (4-2) — always log to stderr (warn) outside debug mode in `internal/cmd/doctor.go`
- ✅ `FormatReport` nil panic (5-2) — added nil guard returning placeholder report in `internal/bench/experiment/report.go`
- ✅ Bootstrap CI degenerate with n=1 (5-2) — return `[0,0]` when either condition has fewer than 2 results in `internal/bench/experiment/compare.go`
- ✅ Spinner `started` field data race (6-0) — `atomic.Bool` in `internal/doctor/spinner.go`
- ✅ Panic wrapper swallows stack trace (4-3) — `runtime/debug.Stack()` captured in `NodeError` in `internal/doctor/diagnostic.go`
- ✅ `gate.go` `computeMetrics` Severity (6-0) — verified already fixed; effectiveSeverity pattern present (`sev := f.Severity; if sev == "" { sev = f.Confidence }`)
- ✅ `secrets.go` `inCodeFence` skip (6-0) — verified already fixed; `if inCodeFence { continue }` present

## Deferred from: second-pass code review of 10-0-epic9-debt-cleanup (2026-04-27)

- **`ResolveTemplate` rejects valid bytes when `os.Lstat` errors with ENOENT after successful `os.ReadFile`** — rare TOCTOU between read and stat (file watcher rotation, FS hiccup). Hard-fail is the safer trade-off than fail-open; document the rationale.
- **`alignResultsByTaskID` positional fallback aliases the caller's underlying slice** — no current Compute* mutates, but contract unenforced. Footgun for future contributors. Document or copy on fallback.
- **`Compare([], runtimeNonEmpty)` loses diagnostic signal** — returns `cr.Tasks=0` hiding the runtime activity. No panic, just a UX gap when SkipMap fully drains one condition.
- **`evaluateRuntimeOutputs` scores empty-string output for never-executed steps** — when upstream errors, downstream steps' `resultByStepID["s2"] = ""`, and negative guardrails (`!destructive`) are falsely credited as respected. Pre-existing pattern in both old positional and new map code; merits dedicated cleanup story.

## Deferred from: code review of 10-0-epic9-debt-cleanup (2026-04-27)

- **`buildSandboxFromWorkspace` produces-key collision with `slots/task_input.md`** — a step that produces a slot named `task_input` would overwrite the auto-injected input at first execution. Extremely unlikely in practice (no realistic naming convention would clash); revisit when slot keys become a formalised reserved namespace.
- **`effectiveGuardrails` defensive-copy allocation cost in retry loop** — copies on every call; current architecture has no mutator that needs the safety. Revisit if profiling shows allocation hotspot.
- **`taskContextContent` step-name fallback to `step.ID`** — exercised by no test. Coverage gap, not a behaviour gap.
- **`slotspec.ResolveTemplate` lexical `strings.HasPrefix(rel, "..")`** — false-rejects `..backup.md`-style names; false-accepts impossible. Cosmetic.
- **`cr.Tasks` / `cr.Runs` semantic drift after alignment** — pre-existing labeling confusion in `report.go`; alignment makes the divergence visible but does not introduce it. Fix labels (Tasks → "paired tasks") in a report polish pass.
- **`alignResultsByTaskID` silently drops extra runs when `RunsPerTask > 1` with asymmetric SkipMap** — doc-comment is accurate; a stderr summary of dropped runs would aid debug-mode users.
- **`LoadTasks` discards `EvalSymlinks` error silently on fallback** — pre-existing transparency gap; one-line stderr warning.

## Closed Bucket C — 2026-04-27 (won't fix)

Sixteen items reviewed and accepted as-is, with the rationale below. They remain in their original sections for traceability but are no longer tracked as actionable.

| Source story | Item | Why closed |
|---|---|---|
| 1.1 | No `io.LimitReader` on API responses | Theoretical OOM; trusted endpoints; no exploitation path |
| 1.1 | `ParseFindings` nil/nil ambiguity | Resolved by Story 1.5 schema validation |
| 1.1 | No retry on 429 rate limit | Pre-existing pattern; user can retry |
| 4-0 | Cache key migration: no explicit invalidation | TTL (7 days) handles drift; documented |
| 4-0 | 32-bit prefix collision in `Purge` | Negligible at realistic cache sizes; documented trade-off |
| 4-0 | Concurrent `Purge`+`Put` logical race | Architectural; only matters with concurrent analysis |
| 4-2 | FR50 gap: `BuildDAG` hardcoded | Tracked TODO in code; deliberate Epic-scoped deferral |
| 5-2 | Unresolved `consumes` silently ignored | Pre-existing `pkg/dag` semantics; intentional |
| 5-3 | Forgent `Validate()` ignores `original` | Identity validation trivially correct |
| 6-19 | Empty `consumes[]` on first node skips task_input | Pre-existing DAG guard logic; by design |
| 6-19 | `ResolveTemplate` never called in runtime path | Resolved by 6-20 / 10-2 |
| 6-19 | Unknown YAML fields silently ignored | `gopkg.in/yaml.v3` library limitation |
| 6-21 | Guardrails injected inline alongside file listing | Intentional; payload small; visibility required |
| 6-21 | Consuming step pre-write attribution in audit | Cosmetic; audit metadata only |
| 6-21 | `funcProvider` missing compile-time interface guard | Style consistency only |
| 10-1 | Negative guardrail (`!prefix`) not tested in step-level tests | Pre-existing logic; not regressed by this story |

## Epic backlog: node-level behavioral contracts (2026-04-27)

**Vision:** Behaviors (guardrails, tool permissions, I/O contracts, loop bounds) are node-level, not agent or task-level. A `StepDef` carries its complete behavioral contract. The task YAML becomes pure topology. The runtime enforces everything at each node boundary.

**Why:** Current experiment baseline exposed the gap — guardrails are task-level, metrics aggregate across nodes with different behavioral profiles. The experiment can't validate the core thesis ("structural enforcement makes agents better") until enforcement is per-node. Node-level contracts also enable reusable nodes that carry their behavioral contract across tasks.

**Stories to open:**

- Move `guardrails []string` from `TaskDef` → `StepDef` (today it's shared across all steps)
- Node-level tool permissions — each step declares its allowed tools, runtime enforces
- Produces/consumes schema validation at slot boundary — validate content when node writes to `slots/*.md`
- Per-node violation telemetry — log attempted vs. prevented violations per node boundary
- Loop bounds at step level — DAG enforces iteration contract, not LLM decision

**Experiment impact:** Once implemented, rerun the 16-task experiment. Delta vs. current baseline (keyword guardrails, task-level) = empirical proof of the thesis. Current experiment run (2026-04-27) is the formal pre-structural-enforcement baseline.

**Relationship to guardrails structurels backlog entry (above):** Output contract, tool call, and network guardrails are the *content* of node-level contracts. This epic is the *infrastructure* to host them.

## Deferred from: code review of story-1.1 (2026-03-30)

- **No `io.LimitReader` on API responses.** Theoretical OOM risk on malicious response. Pre-existing pattern from OpenRouter adapter — fix both together when hardening LLM providers.
- **`ParseFindings` doesn't distinguish "zero findings" from "wrong YAML schema".** Returns `nil, nil` for both. Will be addressed in Story 1.5 (analyzer skill) with proper schema validation.
- **No retry on 429 rate limit.** Pre-existing pattern. Address when building production Ollama adapter (Story 1.2) or runtime provider.
- **Prompt injection via agent file content.** Inherent to LLM usage, not spike-specific. Architectural concern for doctor's security model — address in Growth phase.

## Deferred from: Epic 7 code reviews (2026-03-31)

### Story 7.0 — Quick Wins
- Hash truncation to 64-bit (CacheKey) — birthday collision risk at ~4B entries. Acceptable for local CLI.
- `Purge()` does O(n) full scan of cache files — optimize with naming convention later.
- No test for empty findings as valid cache hit — integration test.
- `reasoning` field intentionally discarded by parse.go — document when reasoning is displayed.
- Timestamp set by caller not inside Put() — minor robustness.
- No integration test for --no-cache flag.

### Story 7.1 — Static Check Foundation
- `DefaultSeverity()` method on interface but `Finding` struct has no Severity field — add when scoring model matures.
- Global registry not concurrency-safe (`sync.Mutex`) — add if concurrent usage needed.
- Tilde fences (`~~~`) not handled by vague check — rare edge case.
- No animated spinner (static "Analyzing with LLM..." message) — cosmetic.

### Story 7.2 — Full Static Suite
- Generic password/secret/token patterns false-positive on documentation text — needs context-aware filtering.
- Unclosed code fence silently disables all remaining checks — add warning.
- O(n²) duplicates with no cap — add early exit at 50 findings.
- Numbered list `3.5 GHz` false positive in duplicates — tighten regex.
- `truncate` slices by byte not rune — corrupts UTF-8 on non-ASCII.
- OpenAI `sk-` regex matches non-API prefixed strings — needs tighter pattern.

### Story 7.3 — Pipeline + Quality Gate
- Secret detection string matching duplicated in gate.go and render.go — extract constant/helper.
- Float equality in gate conditions — use epsilon for non-integer metrics.
- Nil analysis panics in RunPipeline — add nil guard.
- Empty gate with no conditions always passes — add warning.

### Story 7.4 — Cross-File + Drift
- npm builtins falsely validate `npm test` without scripts entry.
- `.PHONY` substring match gives false positives for target substrings.
- No staleness_test.go — add git-mock tests.
- File I/O on every line — no caching of Makefile/package.json reads.
- Windows path separators not handled by filePathRe.
- `findProjectRoot` / `isGitRepo` depend on process-global cwd.

### Story 7.5 — Agent Smells
- LLM-generated detection uses AND (both signals required) — consider OR for either signal alone.
- "should" as permission overcounts (very common word).
- Empty sections inflate checkbox agent average — filter zero-content sections.
- Integer division truncation in checkbox avg — use float.

## Deferred from: code review of 4-0-epic7-debt-cleanup (2026-04-07)

- **Empty gate `Pass:true` with warning** — callers (CI mode) may not surface `Warnings` to stderr before exit-0; revisit in Epic 4 rendering layer when doctor runtime is built.
- **`internal/doctor/validate/` orphaned** — sole caller `runFixMode` deleted in this story; package not imported → not flagged by `deadcode`; clean up when validate/ is superseded or reused in Epic 4.
- **Cache key migration: no explicit invalidation on upgrade** — old-format keys become orphans; TTL (7 days) handles it; documented in comments.
- **`truncate` maxLen≤3 returns no ellipsis** — undocumented behavior; no current callers pass such small values; document or add guard if `truncate` is ever made public.
- **`countWholeWord` case-sensitivity contract** — callers lower-case content by convention only; not enforced in signature; add a `strings.ToLower(word)` inside or a lint check if new callers are added.
- **`ComputeStructural` `passed` can go negative if `totalSections=0`** — no current callers pass 0; add `max(0, ...)` guard when stabilizing the metrics API.
- **32-bit prefix collision space in `Purge`** — 8 hex chars = 2^32 birthday space; negligible at realistic cache sizes but document the trade-off.
- **`countWholeWord` inflates count on all-punctuation `word` input** — no current callers pass such input; add guard if the helper becomes more widely used.
- **`DEADCODE_BIN` path assumes GOBIN matches install target** — works in standard setup; edge case when GOBIN is explicitly set to a non-standard location.
- **Concurrent `Purge`+`Put` logical race** — file-based cache has no mutex; newly-written entry can be immediately deleted by concurrent Purge; architectural limitation for Epic 5 if concurrent analysis is added.
- **`vague.go` still uses `strings.Contains`** — AC #2 narrowly targeted `strings.Count`; broader substring-false-positive cleanup across static checks for a future sprint.
- **Gate regression test assertion tightening** — `LessOrEqual(len(report.AllFindings), 4)` doesn't assert truncation actually happened; NIT for a future cleanup pass.

## Deferred from: code review of 4-1-doctor-skill-and-agent-yaml (2026-04-10)

- **Sequential vs parallelism in doctor.agent.yaml** — Flat `orchestration: sequential` doesn't express Layer 1 parallelism (analyzer ‖ context-enricher). Deferred because Story 4.2 DAG auto-wiring resolves this from consumes/produces contracts regardless of YAML ordering.
- **Hardcoded bench file counts fragile** — `token_test.go` uses exact file count assertions (22, 4) that break on every skill/agent addition. Consider dynamic count or lower-bound assertion.
- **Non-recursive scanners miss `skills/doctor/`** — `parseSkills`, `LintDirectory`, `RunScore` use `os.ReadDir` (non-recursive). Doctor subdirectory skills are invisible to default `forgent lint`/`forgent score`/build. Story 4.2 should address the build pipeline; lint/score need a `--recursive` flag or subdirectory walk.
- **Relative paths in tests are brittle** — `../../skills/doctor` and `../../agents` assume fixed package depth. Pre-existing pattern across all bench/cmd tests.
- **No agent-level vs skill-chain contract validation** — `ValidateAgent()` doesn't verify that agent `consumes`/`produces` matches the I/O of its first/last skill.
- **Custom guardrail keys not validated** — Keys like `max_file_size`, `max_tokens` parse silently via the generic `GuardrailRule` map type but are never validated or enforced.
- **`scoreDataFlow` doesn't flag orphan outputs** — Produced types consumed by no downstream skill are not penalized. Could mask unnecessary skills in pipeline.

## Deferred from: code review of 4-2-embedded-yaml-and-runtime-dag (2026-04-10)

- **FR50 gap: `BuildDAG()` uses hardcoded nodes, not parsed embedded YAML** — `SkillSpecs()` and `AgentSpec()` exist and are tested but not called by production code. `TODO(FR50)` added to `BuildDAG()`. Deferred: implement YAML-to-node translation in a future story to fully satisfy FR50.
- **`provErr` silently swallows LLM unavailability** — non-debug runs get no feedback that LLM analysis was skipped; pre-existing behavior pattern carried into DAG path.
- **`SkillSpecs()` no memoization** — reads embedded FS on every call; currently only consumed by tests so no hot-path impact; add `sync.Once` if production callers are added.
- **Double parse in `analyzeFile`** — caller and DAG both call `DetectFormat`+`Parse` on the same content; minor allocation waste; refactor when DAG inputs are richer.
- **`recommendation-builder` terminal-status fragility** — `RunDAG` assumes it is the terminal node; adding any downstream node would silently break output extraction; add an assertion or named-output extraction.
- **`BuildDAG` rebuilt per file in directory mode** — stateless; could be constructed once and reused; minor allocation overhead for large directory scans.
- **Analysis warnings not surfaced on cache-hit path** — `result.Analysis.Warnings` only visible on DAG execution path; cache-hit callers silently skip them.
- **Equivalence test uses category/confidence bag comparison only** — with a deterministic mock, a field-level `assert.Equal` on sorted findings would be stronger and safe to add.
- **CI uses `-race` not `-short` as specified in AC#6** — test still runs in CI; only the spec's documented command is wrong; update AC#6 comment.
- **`executeLayerBased` drains only first error from same-layer failures** — pre-existing DAG engine behavior; second error from a parallel-layer failure is dropped.

## Deferred from: code review of 4-3-runtime-error-handling (2026-04-10)

- **Concurrent `fmt.Fprintf` to shared `os.Stderr`** — parallel Layer 1 nodes can interleave debug output; cosmetic only, `os.File.Write` is atomic for short lines on Linux.
- **No backoff between retry attempts** — constrained by "Do NOT modify pkg/dag" and no `RetryDelay` in Node struct; tight retry loop may not help with rate-limited LLM providers.
- **Panic wrapper swallows stack trace** — `recover()` loses location info; add `runtime/debug.Stack()` for production panic diagnostics.
- **LLM node detection via magic string `"provider"`** — no shared constant across packages; defer to shared constants work.
- **Hardcoded retry counts (1 and 3)** — inline literals in `BuildDAG`; consider named constants for configurability.
- **Debug wrapper only sees final retry error** — per-attempt errors only available programmatically via `NodeError.AllErrors`, not in debug output.
- **Misleading `Attempts` count on context cancellation** — `Attempts` reflects completed attempts, not configured max; technically correct but misleading.

## Deferred from: code review of 5-3-format-fidelity-test-suite (2026-04-15)

- **`RoundTrip` hardcodes 9 filenames** — opaque failure if testdata files are renamed; consider dynamic selection or `require.FileExists` guard.
- **`formatDetectPath` not extensible** — new parsers with path-based detection need a manual case; contributor docs should mention this.
- **CONTRIBUTING.md example instantiates `claudeParser{}` directly** — tight coupling; suggest shared helper or registry lookup instead.
- **Forgent `Validate()` ignores `original` parameter** — pre-existing asymmetry in forgent.go:54; identity validation is trivially correct; fix when Validate contract is strengthened.

## Deferred from: code review of 5-2-runtime-vs-text-experiment (2026-04-15)

- **`FormatReport` nil panic** — no nil guard on `*ComparisonResult` input; add `if cr == nil { return "No results" }`.
- **`SkipMap` + `Compare` index misalignment** — `Compare` pairs by slice index; SkipMap skips runs, breaking alignment; caller must reconstruct full result set before comparing.
- **Bootstrap CI degenerate with n=1** — single result produces point-estimate CI `[x, x]`; add warning or skip CI when n < 2.
- **Unresolved `consumes` silently ignored** — DAG auto-wiring skips edges for types with no producer; pre-existing limitation of `pkg/dag`.
- **`evaluateTextResponse` relies on external `len(Steps) > 0` invariant** — guarded by `Validate()` upstream but not defensively checked.

## Deferred from: code review of 6-0-dx-polish-and-edge-cases (2026-04-15)

- **`gate.go` `computeMetrics` uses Confidence but not Severity** — HIGH: static findings with `Severity: "high"` and empty `Confidence` are not counted as high findings in the quality gate. Must use `effectiveSeverity` pattern.
- **`secrets.go` `inCodeFence` tracked but never used to skip lines** — HIGH: secret patterns match inside code fence blocks (example configs in docs). Must add `if inCodeFence { continue }`.
- **`smells.go` Checkbox message uses `len(sections)` not `nonEmptySections`** — MED: misleading count in finding message.
- **Spinner `started` field data race** — no synchronization between Start() write and Stop() read; use `atomic.Bool`.
- **Config context `"set "` substring matches inside "reset", "preset"** — false suppression of real secrets near words containing "set".
- **`TestSecrets_SkipsConfigContext` doesn't exercise `isConfigContext`** — test cases match placeholder filter, not config context filter.

## Deferred from: Epic 7 retrospective action items (2026-03-31)

1. **[HIGH] Lint for unused exported functions** — prevent dead code from shipping. Consider `staticcheck` or custom grep in CI.
2. **[HIGH] Create `countWholeWord()` helper** — replace all `strings.Count` substring matching with word-boundary matching across static checks.
3. **[MEDIUM] Test with realistic agent file inputs** — add real scraped agent files to smells testdata, not just synthetic strings.
4. **[MEDIUM] Escape all output for target format** — audit CI annotations, JSON, and plain text output for injection risks.
5. **[MEDIUM] Add test verifying gate evaluates before PostProcess truncation** — regression test for the ordering fix.
6. **[LOW] Carried debt from Epic 3** — size guard on LLM prompts, corpus test for --fix, .bak file cleanup.

## Deferred from: code review of 6-20-qualitygate-format-validation (2026-04-17)

- **First-slot-wins silently ignores subsequent slot templates** — `buildTaskDAG` breaks on first non-empty template per produces/consumes; multi-slot enforcement requires `QualityGatePolicy` to hold `[]string` per direction; future story.
- **`template_file` broken in runtime path** — `slot.ResolveTemplate("")` with empty baseDir resolves relative to CWD, bypasses traversal guard; requires TaskDefinition to carry source path; pre-existing from 6-19 deferred.
- **`"- "` marker trivially satisfied** — dash-space matches any prose with hyphen-space; spec defines markers as list item prefixes; tightening the check requires a new spec.
- **Budget middleware 60s overrides node-level 120s** — pre-existing harness behavior; shorter timeout silently wins; add `BudgetPolicy.Timeout` override support when hardening providers.
- **`MaxRetries = 2` set unconditionally including pass-through nodes** — template-less LLM steps now get 2 DAG retries; retries on network errors are reasonable; revisit if retries cause excessive LLM cost.
- **No warning when template has no extractable markers** — passthrough is silent; add observability log line when marker extraction yields nil.
- **ConsumesTemplate failure is invariant across retries** — upstream bad output committed to DAG store; retries exhaust budget without helping; architectural limitation.
- **YAML single-quoted scalar `\n` authoring trap** — `\n` in single-quoted scalars is literal; current YAMLs use correct double-quotes; add contributor docs note.

## Deferred from: code review of 6-19-slotspec-template-contracts (2026-04-17)

- **Template + TemplateFile both set silently drops Template** — TemplateFile wins, precedence undocumented. Document in story 6-20 when QualityGate wires SlotSpec.
- **ResolveTemplate never called in runtime path** — Template/TemplateFile fields are dead code until story 6-20 wires QualityGate to use them. By design (story split).
- **Empty consumes[] on first node skips task_input injection** — pre-existing DAG guard logic, not introduced by this story.
- **LoadTasks doesn't bind TaskDefinition to source file** — ambiguous baseDir for TemplateFile resolution in multi-directory loading. Story 6-20 caller must pass correct baseDir.
- **Unknown YAML fields silently ignored** — gopkg.in/yaml.v3 library behavior; typo in `template_file` key is undetectable without a custom decoder.

## Deferred from: code review of 6-21-file-based-slot-passing (2026-04-17)

- **Tool-call loop resets toolPrompt to base on each guardrail retry** — accumulated tool-read context lost if guardrails fail; now exposed to all LLM steps (not just tool-declared steps). Address when hardening guardrail retry logic.
- **toolPrompt grows unboundedly across iterations** — no length cap on tool-result accumulation in the inner tool loop. Address when adding LLM context budget awareness.
- **TOCTOU lock gap in SandboxFS.Read** — RLock released before Lock acquired; concurrent write can race between policy check and read. Fix with a single exclusive lock in Read.
- **task_input not written to slots/task_input.md** — if LLM guesses file_read("slots/task_input.md") it gets not-found; task_context.md covers the content but the naming gap is surprising. Consider adding a symlink-like alias or an explicit note in the prompt.
- **Guardrails still injected inline alongside file listing** — small payload; intentional (guardrails must be immediately visible); revisit if guardrail lists become large.
- **lastResponse empty/malformed when tool loop exhausts max iterations** — LLM keeps emitting tool calls for all 5 iterations; the malformed output is returned as step result. Add a final "synthesize your answer" instruction when maxToolIterations is reached.
- **Consuming step's pre-write of upstream slot file attributed to consuming step ID** — audit log misleadingly shows the consuming step as writer of the upstream slot; cosmetic only.
- **seen["slots/**"] asymmetry in buildSandboxFromWorkspace** — seen["task_context.md"] has a correct guard but seen["slots/**"] is never populated; "slots/**" is always appended unconditionally. Normalize both checks.
- **funcProvider in experiment_test.go missing compile-time interface guard** — add `var _ llm.Provider = (*funcProvider)(nil)` for consistency with other test providers.
- **AC4/AC5 not verified** — guardrail gap and token reduction improvements require running the 16-task forgent-bench experiment to confirm.

## Backlog: guardrails structurels (2026-04-27)

Remplacer les guardrails keyword par trois catégories déterministes :

- **Output contract** — valider la structure de la réponse (JSON schema, champs obligatoires) via le `produces` contract. Lié à story 6-6 (`output-schema-and-produces-contract`).
- **Tool call** — intercepter les appels d'outils dans le tool-call loop : `!file_delete`, `!bash(rm -rf)`, etc. Soit la tool n'a pas été appelée, soit le guardrail échoue. Zéro sémantique.
- **Network** — sandbox réseau : bloquer les appels HTTP sortants non autorisés. Complémentaire au VirtualFS sandbox existant (story 5-4).

Motivation : les guardrails keyword (`!LGTM`, `!safe`) sont des proxies sémantiques fragiles qui déclenchent des retries LLM inutiles. Ces trois catégories sont **observables et déterministes** — pas de retry possible sur un fait constaté.

## Deferred from: code review of 6-22-context-retriever (2026-04-17)

- **`RankedRetriever` breaks early without trying smaller chunks** — when no chunk fits within `MaxTokens` budget, returns empty result without retrying with smaller chunk sizing. Optimization opportunity.
- **`CompareThree` missing ranked deltas; ranked table missing 95% CI** — the 3-condition report omits delta computation for the ranked column and confidence intervals; enhancement beyond story 6-22 scope.

## Deferred from: code review of 10-1-node-level-guardrails (2026-04-27)

- **`taskContextContent` doesn't include step-level guardrails** — `task_context.md` in the sandbox lists only task-level guardrails; step-level ones are injected in the step prompt but absent from the audit file. Out of scope for 10-1.
- **`effectiveGuardrails` returns reference not copy** — alias risk if slices are mutated in future concurrent contexts. Pre-existing pattern; safe for current single-threaded-per-step model.
- **Three ad-hoc mock provider types in test package** — `mockExpProvider`, `mockGuardrailProvider`, `callCountingMock` with no shared abstraction. Test organization debt.
- **`effectiveGuardrails` undocumented return-reference semantics** — should note it returns a slice header, not a defensive copy.
- **3-retry exhaustion case not tested for step-level guardrails** — `maxAttempts := 3` pre-existing logic; test only covers 2-attempt path.
- **Negative guardrail (`!prefix`) semantics not tested in new step-level tests** — `checkGuardrail` negation logic is pre-existing; not regressed by this story.

## Deferred from: code review of 10-2-fix-template-file-runtime (2026-04-27)

- **Error message wording: "escapes base directory" ≠ "escapes task directory" (AC2)** — `slotspec.go:61` says "escapes base directory"; AC2 requires "escapes task directory". Pre-existing wording, not caused by this change. Update if error messages become user-facing or tested.
- **StepResults ordering assumes sequential DAG execution** — `evaluateRuntimeOutputs` correlates `StepResults[i]` with `task.Steps[i]` by position; this assumption breaks under future parallel DAG execution. Deferred until parallelism is introduced.
- **TOCTOU symlink race in ResolveTemplate** — path validated via `filepath.Rel`, then read via `os.ReadFile`; symlink could be swapped between check and read. Pre-existing in `slotspec.go`.
- **Concurrent StepDef.Sandbox mutation in per-step closure** — `stepCopy.Sandbox = &StepSandboxPolicy{...}` mutates inside closure; races with concurrent DAG node execution. Pre-existing.
- **TaskDir not symlink-resolved in LoadTasks** — `td.TaskDir = dir` stores `dir` without `filepath.EvalSymlinks`; symlink-accessed task dirs may resolve TemplateFile paths to unintended locations. Low-risk edge case.
- **`effectiveGuardrails` returns direct slice reference** — same as deferred in 10-1; alias risk if callers mutate; safe for current sequential model.
- **TaskDir unset in 15+ programmatic TaskDefinition constructions across tests** — latent risk: any test that adds a `TemplateFile` slot to a directly-constructed `TaskDefinition` will silently skip the quality gate. Not an active bug.

## Deferred from: code review of 10-4-output-contract-validation (2026-04-28)

- **Multi-key recovery test coverage** — partial recovery with multiple consumes slots is untested in `internal/bench/experiment/output_contract_test.go`. Source: Blind Hunter B20.
- **Partial recovery state corruption when one of multiple producers fails mid-recovery** in `consumesRetryMiddleware` — sandbox left with cleared upstream files but no replacements. Multi-consumes scenario; current 10-4 tasks all use single-consumes. Source: Edge Case Hunter E5. Pointer: `internal/bench/experiment/runtime_runner.go:consumesRetryMiddleware`.
- **Multiple-produces-with-different-templates: only first non-empty template enforced** — pre-existing behavior in `buildTaskDAG` template-resolution loop, not introduced by 10-4 (same in old middleware path). Should validate every produces slot's template against its slot text. Source: Edge Case Hunter E14.
- **Sentinel-error refactor for `isConsumesValidationError`** — replace string-match on qualitygate error wording with `errors.Is/As` and a typed error. Larger qualitygate refactor; out of 10-4 scope. Source: Blind Hunter B2 + Edge Case Hunter E1. Pointer: `internal/bench/experiment/runtime_runner.go:isConsumesValidationError` + `pkg/qualitygate/gate.go`.
- **`sandbox.Delete` error swallowed in `consumesRetryMiddleware` slot-clear path** — bounded impact in practice (producer rewrite covers stale slot), but the swallow + double-record pattern (manual `sandbox.record` followed by `Delete` which records again) is a smell worth cleaning. Source: Edge Case Hunter E2. Pointer: `internal/bench/experiment/runtime_runner.go:consumesRetryMiddleware`.

## Deferred from: code review of 6-18-harness-experiment (2026-04-28)

- **`consumesRetryMiddleware`: delete slot before confirming `producerOut[key]`** — slot is cleared from sandbox via `sandbox.Delete` before verifying the producer's output contains the expected key. If the producer succeeds but returns a different key name, the slot is cleared but never refilled — cascades into missing `file_read` target. Conflicts semantically with 10-6 (slot_clear audit semantics), so deferred post-10-6. Source: Edge Case Hunter. Pointer: `internal/bench/experiment/runtime_runner.go:91-95`.
- **`len(consumes) > 0` guard too broad in middleware wiring** — `consumesRetryMiddleware` is attached even when `consumes = [{Key: "task_input"}]` only (no upstream producer), causing a misleading "no upstream producer for recovery" error on consumes-gate failure. Deferred: potential line conflict with 10-5 middleware wiring section. Source: Edge Case Hunter. Pointer: `internal/bench/experiment/runtime_runner.go:685`.

## Backlog — Query/Command node split (2026-04-29)

- **Produces slots = seuls writes autorisés par défaut** — si un step ne déclare pas `FileAccess` explicitement, dériver les allowed write targets depuis `step.Produces` (i.e. `slots/{key}.md` uniquement). Rend les tâches existantes plus sûres sans changer les YAML. La séparation formelle Query/Command est laissée de côté — ambiguë en pratique, aucune enforcement nouvelle par rapport aux mécanismes existants (`FileAccess` + `NetworkPolicy` + `AllowedPaths`).

## Deferred from: code review of 10-5-http-proxy-network-sandbox (2026-04-28)

- **`os.Setenv("HTTP_PROXY")` process-global Phase 1 limitation** — Documented in code. Safe for sequential per-task execution. Phase 2 switches to per-request `http.Transport.Proxy` when parallel DAG layers are enabled (story 10-7+). Source: Blind Hunter + Acceptance Auditor. Pointer: `internal/bench/experiment/runtime_runner.go`.
- **Network audit reason may inflate file `ScopeCreepRate` metric** — `ComputeActionMetrics` may match `"domain not in allowed_domains"` reason strings against file scope creep heuristics. Needs verification that `Action: "network"` entries are correctly excluded from file-specific metrics. Source: Edge Case Hunter (C12). Pointer: `internal/bench/experiment/sandbox.go:ComputeActionMetrics`.
- **`context.Background()` hardcoded in `runRuntimeTask`** — `d.Execute(context.Background(), ...)` discards any outer cancellation signal (SIGINT, HTTP deadline). Structural fix: thread the caller's context through. Deferred post-MVP. Source: Edge Case Hunter. Pointer: `internal/bench/experiment/runtime_runner.go:271`.
- **Goroutine leak: `provider.Complete` has no `ctx` parameter** — budget timeout via `context.WithTimeout` cannot abort an in-flight HTTP call when `Complete(prompt string)` doesn't accept a context. Structural fix: add `ctx context.Context` to `llm.Provider.Complete` and all implementations. Deferred post-MVP (interface-breaking change). Source: Edge Case Hunter. Pointer: `pkg/budget/middleware.go` + `llm.Provider` interface.

## Deferred from: code review of 10-6-violation-telemetry (2026-04-29)

- **Unknown step IDs sort to position 0 in `aggregateViolations`** — The "dag" stepID used in `sandbox.Write("dag", "task_context.md", ...)` is not present in `task.Steps`; `stepIndex["dag"]` returns Go's zero value (0), silently placing that violation first — same position as the real first step. Pre-existing design in runtime_runner.go; requires renaming the "dag" stepID to a sentinel (e.g., "__harness__") or filtering harness-level writes from the violation log. Pointer: `internal/bench/experiment/telemetry.go:aggregateViolations`, `internal/bench/experiment/runtime_runner.go`.
- **AttemptNumber=0 violations excluded from CCR penalty** — `computeContractComplianceRate` checks `v.AttemptNumber == 1`; violations with `AttemptNumber=0` (recorded outside the retry loop, before `SetCurrentAttempt` is called) are not penalized. Unreachable in current paths (`SetCurrentAttempt(stepID, 1)` is always called before any enforcement fires). Latent bug for future callers that use `SandboxFS` outside the retry loop. Pointer: `internal/bench/experiment/telemetry.go:computeContractComplianceRate`.
- **DAG-level retry resets AttemptNumber** — If `pkg/dag` MaxRetries > 0, `runRuntimeTask` can be re-invoked; each re-invocation starts `attempt=0` and calls `SetCurrentAttempt(stepID, 1)`, making second-DAG-invocation violations indistinguishable from first-invocation attempt-1 violations. Bench tasks don't use DAG-level node retries currently. Pointer: `internal/bench/experiment/runtime_runner.go`, `pkg/dag/execute.go:runWithRetry`.
- **Sandboxless tasks report CCR=1.0, indistinguishable from measured compliance** — When `sandbox == nil`, `runRuntimeTask` returns `ContractComplianceRate: 1.0` (default). Consumers can't tell if 1.0 means "measured zero violations" or "no sandbox = no measurement". Requires nullable float or a `ContractComplianceMeasured bool` field. Pointer: `internal/bench/experiment/runtime_runner.go:runRuntimeTask`.

## Deferred from: code review of 10-7-loop-bounds-exit-criteria (2026-04-29)

- **Legacy loop off-by-one: MaxIterations+1 total LLM calls** — In the legacy path (Loop=nil), the first `provider.Complete` is called before the tool-call loop, giving `MaxIterations+1` total calls instead of `MaxIterations`. Pre-existing behavior, not introduced by 10-7. Pointer: `internal/bench/experiment/runtime_runner.go` legacy loop path.
- **No upper bound on `max_iterations` value** — `validateLoopPolicy` accepts `max_iterations: 10000` with no exit criterion. A misconfigured task can run thousands of billable LLM calls. Requires a product decision on the max sane cap (e.g., 100). Pointer: `internal/bench/experiment/design.go:validateLoopPolicy`.
- **Multiple `loop_exceeded` audit records across DAG retries** — With `node.MaxRetries=2`, a step that exhausts its loop 3 times emits 3 `loop_exceeded` records with distinct attempt numbers. Inflates `NodeViolations` volume. Design decision: should loop exhaustion be deduplicated per step per task run? Pointer: `internal/bench/experiment/runtime_runner.go:717`, `pkg/dag/execute.go:runWithRetry`.
- **Synthesize call returns `ErrLoopExhausted` if LLM includes `<tool .../>` XML in prose (no exit criterion)** — When `hasExitCriterion=false` and the synthesize response contains any `<tool .../>` XML (e.g., an example in a prose explanation), `len(synthCalls) > 0` triggers `ErrLoopExhausted` even if the prose answer is otherwise valid. Theoretical edge case; "Final Answer Required" prompt should prevent this in practice. Pointer: `internal/bench/experiment/runtime_runner.go:715`.
- **`exitToolArgs` synthetic key not DAG-wired to downstream steps** — The key `<stepID>_<toolName>_args` is added to `out` at runtime but not declared in `step.Produces`; DAG auto-wiring ignores it. Downstream steps can only access it via `allOutputs` (global), not via DAG `consumes` declaration. Requires design decision on whether synthetic keys should be registered. Pointer: `internal/bench/experiment/runtime_runner.go:820`.
- **Accumulator reset between DAG-level retries** — `acc` is declared inside `runFn` body; when `pkg/dag` retries the node, a fresh accumulator is created and tool-result history from the prior invocation is lost. AC6 scopes accumulation to iteration-level (within a single `runFn` invocation); surviving DAG retries requires moving `acc` to the closure capture scope. Design decision. Pointer: `internal/bench/experiment/runtime_runner.go:624`.
