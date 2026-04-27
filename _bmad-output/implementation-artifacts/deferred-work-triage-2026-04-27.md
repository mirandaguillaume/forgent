# Forgent — Triage Deferred Work (Epic 10 prep)

Date: 2026-04-27
Source: `_bmad-output/implementation-artifacts/deferred-work.md`
Buckets:
- **A** — Epic 10 prerequisite (block/parallel with stories 10-3..10-9)
- **B** — Post-MVP (real but not urgent)
- **C** — Won't fix (close)

## Triage Table

| Bucket | Section source | Item (resume) | Justification |
|--------|----------------|---------------|---------------|
| A | 6-21 | TOCTOU lock gap in SandboxFS.Read (RLock->Lock race) | Sandbox FS — 10-3 base |
| A | 6-21 | seen["slots/**"] asymmetry in buildSandboxFromWorkspace | Sandbox policy correctness |
| A | 6-21 | task_input not written to slots/task_input.md | Sandbox file mapping |
| A | 6-21 | Tool-call loop resets toolPrompt on guardrail retry | Guardrail enforcement state |
| A | 6-21 | toolPrompt grows unboundedly across iterations | Loop bounds 10-7 |
| A | 6-21 | lastResponse empty/malformed on max iterations | Loop bounds 10-7 |
| A | 6-21 | AC4/AC5 not verified (16-task experiment rerun) | Direct 10-9 dependency |
| A | 10-2 | TOCTOU symlink race in ResolveTemplate | FS sandbox 10-3 |
| A | 10-2 | Concurrent StepDef.Sandbox mutation in closure | Per-step sandbox 10-3 |
| A | 10-2 | TaskDir not symlink-resolved in LoadTasks | FS sandbox 10-3 |
| A | 10-2 | TaskDir unset in 15+ test constructions (silent skip) | Falsifies 10-4 metrics |
| A | 10-2 | StepResults ordering assumes sequential DAG | Breaks 10-6 telemetry |
| A | 10-1 | taskContextContent excludes step-level guardrails | 10-1 follow-up, audit gap |
| A | 10-1 | effectiveGuardrails returns reference not copy | Concurrent 10-3 risk |
| A | 10-1 | 3-retry exhaustion not tested for step-level guardrails | 10-6 metrics integrity |
| A | 6-22 retro | Backlog: output contract / tool call / network guardrails | Defines 10-3/10-4/10-5 |
| A | 6-20 | template_file broken in runtime (CWD bypasses traversal) | FS sandbox correctness |
| A | 6-20 | Budget middleware 60s overrides node-level 120s | Per-node contract violation |
| A | 6-20 | ConsumesTemplate failure invariant across retries | Wastes 10-6 retry budget |
| A | 5-2 | FormatReport nil panic (no nil guard) | 10-9 rerun stability |
| A | 5-2 | SkipMap + Compare index misalignment | 10-9 baseline alignment |
| A | 5-2 | Bootstrap CI degenerate with n=1 | 10-9 stats sanity |
| A | 6-0 | gate.go computeMetrics ignores Severity | Falsifies gate metrics |
| A | 6-0 | secrets.go inCodeFence tracked but not used | HIGH bug, contract integrity |
| A | 6-0 | Spinner started field data race | Concurrency hygiene |
| A | 4-2 | provErr silently swallows LLM unavailability | Fakes 10-6 telemetry |
| A | 4-2 | recommendation-builder terminal-status fragility | Output contract 10-4 |
| A | 4-2 | executeLayerBased drains only first error | 10-6 lost violations |
| A | 4-3 | No backoff between retry attempts | 10-7 loop bounds |
| A | 4-3 | Panic wrapper swallows stack trace | 10-6 diagnostics |
| A | 4-1 | Custom guardrail keys parsed silently, never validated | 10-3..10-5 silent failure |
| A | 4-1 | No agent-level vs skill-chain contract validation | Output contract 10-4 |
| A | retro 7 | Lint for unused exported functions | Hygiene before Epic 10 |
| A | spike 1.1 | Prompt injection via agent file content | Security model for 10-5 |
| B | 7.0 | Purge() O(n) full scan | Perf only |
| B | 7.0 | No test for empty findings as valid cache hit | Coverage |
| B | 7.0 | reasoning field discarded by parse.go | Doc only |
| B | 7.0 | Timestamp set by caller not Put() | Robustness nit |
| B | 7.0 | No integration test for --no-cache | Coverage |
| B | 7.1 | DefaultSeverity() vs Finding.Severity field | Future model |
| B | 7.1 | Global registry not concurrency-safe | Add when needed |
| B | 7.2 | Generic password regex doc false-positive | Tuning |
| B | 7.2 | Unclosed code fence silently disables checks | Add warning |
| B | 7.2 | O(n^2) duplicates with no cap | Perf |
| B | 7.2 | Numbered list 3.5 GHz false positive | Regex tuning |
| B | 7.2 | truncate slices by byte not rune | UTF-8 edge |
| B | 7.2 | OpenAI sk- regex over-broad | Tuning |
| B | 7.3 | Secret string match duplicated gate.go/render.go | DRY |
| B | 7.3 | Float equality in gate conditions | Epsilon |
| B | 7.3 | Nil analysis panics in RunPipeline | Defensive |
| B | 7.3 | Empty gate with no conditions always passes | UX warn |
| B | 7.4 | npm builtins falsely validate npm test | Tuning |
| B | 7.4 | .PHONY substring match false positive | Tuning |
| B | 7.4 | No staleness_test.go | Coverage |
| B | 7.4 | File I/O on every line — no cache | Perf |
| B | 7.4 | findProjectRoot/isGitRepo depend on cwd | Refactor |
| B | 7.5 | LLM-detection AND vs OR | Tuning |
| B | 7.5 | "should" overcounts | Tuning |
| B | 7.5 | Empty sections inflate checkbox avg | Tuning |
| B | 7.5 | Integer division in checkbox avg | Float |
| B | 4-0 | Empty gate Pass:true with warning | Rendering layer |
| B | 4-0 | internal/doctor/validate orphaned | Cleanup |
| B | 4-0 | truncate maxLen<=3 ellipsis | Doc/guard |
| B | 4-0 | countWholeWord case-sensitivity | Lint check |
| B | 4-0 | ComputeStructural negative passed if total=0 | Guard |
| B | 4-0 | countWholeWord punctuation input | Guard |
| B | 4-0 | DEADCODE_BIN GOBIN edge | Edge env |
| B | 4-0 | vague.go strings.Contains broader cleanup | Future sprint |
| B | 4-0 | Gate regression assertion tightening | NIT |
| B | 4-1 | Hardcoded bench file counts fragile | Test brittle |
| B | 4-1 | Non-recursive scanners miss skills/doctor/ | --recursive flag |
| B | 4-1 | Relative paths in tests are brittle | Pre-existing |
| B | 4-1 | scoreDataFlow doesn't flag orphan outputs | Score nit |
| B | 4-1 | Sequential vs parallelism in doctor.agent.yaml | Resolved by 4.2 |
| B | 4-2 | SkillSpecs() no memoization | Perf |
| B | 4-2 | Double parse in analyzeFile | Perf |
| B | 4-2 | BuildDAG rebuilt per file | Perf |
| B | 4-2 | Analysis warnings not surfaced on cache hit | UX |
| B | 4-2 | Equivalence test bag comparison | Test strength |
| B | 4-2 | CI uses -race not -short | AC comment |
| B | 4-3 | LLM detection via magic string "provider" | Constants |
| B | 4-3 | Hardcoded retry counts (1 and 3) | Constants |
| B | 4-3 | Debug wrapper sees only final retry error | Debug detail |
| B | 4-3 | Concurrent fmt.Fprintf to os.Stderr | Cosmetic |
| B | 4-3 | Misleading Attempts on context cancel | Doc |
| B | 5-3 | RoundTrip hardcodes 9 filenames | Test brittle |
| B | 5-3 | formatDetectPath not extensible | Docs |
| B | 5-3 | CONTRIBUTING claudeParser{} direct | Style |
| B | 5-2 | evaluateTextResponse external invariant | Defensive |
| B | 6-0 | smells.go Checkbox uses len(sections) | Cosmetic msg |
| B | 6-0 | Config context "set " inside reset/preset | Tuning |
| B | 6-0 | TestSecrets_SkipsConfigContext miss-target | Test rewrite |
| B | retro 7 | countWholeWord helper rollout | Hygiene |
| B | retro 7 | Realistic agent file inputs in tests | Coverage |
| B | retro 7 | Escape output for target format | Audit |
| B | retro 7 | Test gate-before-truncate ordering | Regression |
| B | retro 7 | Carried Epic 3 debt (size guard, .bak) | LOW priority |
| B | 6-20 | First-slot-wins ignores subsequent templates | Multi-slot future |
| B | 6-20 | "- " marker trivially satisfied | Spec change |
| B | 6-20 | No warning when template has no markers | Observability |
| B | 6-20 | YAML single-quoted \n trap | Docs |
| B | 6-19 | Template + TemplateFile both set | Doc precedence |
| B | 6-19 | LoadTasks doesn't bind source file | Story 6-20 caller |
| B | 6-22 | RankedRetriever no smaller-chunk retry | Optimization |
| B | 6-22 | CompareThree missing ranked deltas / CI | Enhancement |
| B | 10-1 | 3 ad-hoc mock provider types in tests | Test debt |
| B | 10-1 | effectiveGuardrails undocumented semantics | Doc |
| B | 10-2 | Error wording escapes base vs task dir | Wording |
| C | 1.1 | No io.LimitReader on API responses | Theoretical OOM, no path |
| C | 1.1 | ParseFindings nil,nil ambiguity | Already addressed by 1.5 |
| C | 1.1 | No retry on 429 | Pre-existing pattern, deferred |
| C | 4-0 | Cache key migration: no explicit invalidation | TTL handles it, documented |
| C | 4-0 | 32-bit prefix collision in Purge | Explicit assumed trade-off |
| C | 4-0 | Concurrent Purge+Put logical race | Architectural, no path |
| C | 6-19 | Empty consumes[] on first node skips task_input | Pre-existing DAG, by design |
| C | 6-19 | ResolveTemplate never called in runtime | Resolved by 6-20 / 10-2 |
| C | 6-19 | Unknown YAML fields silently ignored | yaml.v3 library limitation |
| C | 6-21 | Guardrails injected inline alongside file listing | Intentional, payload small |
| C | 6-21 | Consuming step pre-write attribution | Cosmetic, audit only |
| C | 6-21 | funcProvider missing compile-time guard | Style consistency |
| C | 5-2 | Unresolved consumes silently ignored | Pre-existing pkg/dag, accepted |
| C | 5-3 | Forgent Validate() ignores original | Trivially correct, pre-existing |
| C | 4-2 | FR50 gap BuildDAG hardcoded | Tracked TODO, Epic-scoped |
| C | 10-1 | Negative guardrail (!prefix) not tested | Pre-existing, not regressed |

## Counts

- **Bucket A** (Epic 10 prerequisite): 33
- **Bucket B** (Post-MVP): 60
- **Bucket C** (Won't fix): 16

## Top 5 Bucket A — Action Recommendations

1. **TOCTOU lock gap in SandboxFS.Read (6-21)** — Replace `RLock`/`Lock` dance with single exclusive `Lock` in `Read`. Required before story 10-3 (file-level permissions) ships, otherwise concurrent test workers fabricate false "no violation" results.
2. **TaskDir unset in 15+ test constructions (10-2)** — Add a constructor `NewTaskDefinition(dir, ...)` and migrate all programmatic test builders. Without this, story 10-4 quality-gate metrics will silently pass on tests that should fail.
3. **`provErr` swallows LLM unavailability + `executeLayerBased` drains only first error (4-2)** — Wrap provider errors with explicit telemetry events; surface all parallel errors. Blocks story 10-6 — currently impossible to count violations correctly.
4. **Tool-call loop resets toolPrompt on guardrail retry + grows unboundedly (6-21)** — Define a per-iteration prompt accumulator with a hard cap. Required before 10-7 (loop bounds & exit criteria).
5. **AC4/AC5 not verified — 16-task experiment rerun (6-21)** — This IS story 10-9. Treat as the validation story for the whole Epic 10, not just a 6-21 follow-up.

## Top 5 Bucket B — Action Recommendations

1. **Non-recursive scanners miss `skills/doctor/` (4-1)** — Add `--recursive` flag to lint/score after Epic 10 ships.
2. **truncate slices by byte not rune (7.2)** — UTF-8 corruption on display path; schedule next polish sprint.
3. **`countWholeWord` helper rollout (retro 7 + 4-0)** — Replace remaining `strings.Count`/`Contains` substring matching across static checks.
4. **Hardcoded bench file counts fragile (4-1)** — Convert to lower-bound assertions; trivial.
5. **Float equality + nil analysis guards in pipeline (7.3)** — Combine with rendering polish; not load-bearing for Epic 10.

## Top 5 Bucket C — Action Recommendations (close)

1. **32-bit prefix collision in Purge** — Explicit trade-off documented; close.
2. **No `io.LimitReader` on API responses** — Theoretical, no exploitation path; close as accepted risk.
3. **ParseFindings nil/nil ambiguity** — Resolved by Story 1.5; close.
4. **Concurrent Purge+Put logical race** — Architectural; only matters if concurrent analysis added; close until then.
5. **Unresolved `consumes` silently ignored (5-2)** — Pre-existing `pkg/dag` semantics, intentional and documented; close.
