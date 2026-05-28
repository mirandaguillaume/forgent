# Sprint Change Proposal — 2026-04-28 — Epic 6 Backlog Closure

**Trigger:** Epic 9 retrospective action #4 ("Decision Epic 6 backlog vs Epic 10").
**Decision date:** 2026-04-28
**Decision author:** Guillaume + Bob (Scrum Master)
**Predecessor pivot:** sprint-change-proposal-2026-04-07.md (which defined the
2026-04-16 "Runtime Harness" pivot of Epic 6).

## Context

The 2026-04-16 party mode decision pivoted Epic 6 from "Ecosystem & DX" to
"Runtime Harness — Enforcement by Construction". Stories 6-5 through 6-17 were
defined under the new harness scope.

In Sprint 9 (2026-04-27), Epic 10 ("Node-Level Behavioral Contracts") was created
with stories 10-3 through 10-9 covering the same problem space at a more
architecturally coherent level (per-step contracts vs. middleware composition).
The Epic 9 retrospective flagged the unresolved overlap as action #4.

After authoring all Epic 10 stories on 2026-04-27/28, the overlap is now
explicit and decidable.

## Decision

**Close Epic 6 backlog (stories 6-5 through 6-17) without implementation.** Six
stories are superseded by Epic 10 work; seven stories are orthogonal concerns
deferred to post-MVP.

Epic 6's `done` stories (6-1 through 6-4, 6-18, 6-19, 6-20, 6-21, 6-22, 6-0
hotfix) and `review` stories (6-18) remain as-is. Epic 6 transitions to
`done` once 6-18 closes.

## Item-by-item disposition

### Replaced by Epic 10 (close as superseded)

| Story | Replaced by | Why |
|---|---|---|
| **6-5 pii-output-filter** | 10-4 output-contract-validation | PII filtering becomes a specialisation of output contracts: `<!-- REQUIRED: NO_PII -->` markers + validation against PII patterns at the slot-write boundary. Same enforcement seam, broader applicability. |
| **6-6 output-schema-and-produces-contract** | 10-4 output-contract-validation | Direct duplicate. Same scope, same outcome. 10-4's REQUIRED markers and `template_file` resolution (with the symlink/TOCTOU fixes from 10-0) are the canonical implementation. |
| **6-7 regex-and-length-enforcement** | 10-1 node-level-guardrails | Step-level guardrails already support `!pattern` semantics for regex-style negation. Length enforcement is a trivial extension of the same pattern matcher (no new infra). |
| **6-10 code-syntax-validation** | 10-4 output-contract-validation | "Output must be valid Go" is an output contract. The validator can be plugged into 10-4's REQUIRED-marker pipeline as a custom check. |
| **6-11 diff-applicability** | 10-4 output-contract-validation | Same: diff structure validation is a specialised output contract. |
| **6-17 audit-trail** | 10-3 + 10-4 + 10-5 + 10-6 (implicit) | Already delivered. `sandbox.AccessEntry` (10-3 onward) plus `NodeViolations` and `ContractComplianceRate` (10-6) form the audit substrate. No additional story needed. |

### Deferred to post-MVP

| Story | Reason for deferral |
|---|---|
| **6-8 idempotency-detection** | Different concern: replay safety / side-effect determinism. Needs its own design. Out of MVP scope; revisit when production replay scenarios surface. |
| **6-9 rate-limiter** | Provider-level concern (LLM API throttling), not node-level. Belongs in `internal/llm/` or a dedicated `pkg/budget` extension, not in the runtime contracts layer. Defer until production traffic requires it. |
| **6-12 build-gate** | CI/CD level concern. `forgent build` + `go build` already exist; any "gate" is a workflow integration on top, not a runtime feature. Defer to a future "DX & CI Integration" pass. |
| **6-13 lint-gate** | Same as 6-12. `forgent lint` exists; gate is workflow integration. |
| **6-14 test-gate** | Same. `go test` is the substrate; gate is workflow. |
| **6-15 format-gate** | Same. `gofmt`/`goimports` is the substrate; gate is workflow. |
| **6-16 circuit-breaker** | Provider-level resilience pattern. Orthogonal to runtime contracts. Pairs naturally with rate-limiter (6-9) in a future provider-resilience story. |

## Impact

### sprint-status.yaml

- 6-5 through 6-17 → `closed` (new status, distinguishes from `done` which implies "implemented")
- Add a comment block at the top of the Epic 6 section pointing to this decision file
- `epic-6` epic status remains `in-progress` until 6-18 (still in review) closes

### Epic 9 retrospective action items

- Action #4 ("Decision Epic 6 backlog vs Epic 10") → **resolved 2026-04-28**
- Update the Epic 9 retro file's action-items table accordingly when the closure
  commit lands.

### Future epic considerations

The seven deferred items (6-8, 6-9, 6-12 through 6-17) form a coherent post-MVP
cluster around "Provider Resilience + CI Integration". If demand surfaces post-
MVP, they can be regrouped into a future Epic 11 ("Provider & CI/CD Concerns").
That epic is NOT created today — only the option is preserved.

### No code changes

This decision is documentation-only. No source files are touched. No tests change.
No commits to `internal/`, `pkg/`, or `cmd/`.

## Validation

- Each "replaced by" claim was verified against Epic 10 stories 10-1 through 10-9
  authored on 2026-04-27/28; the relevant story's scope absorbs the Epic 6 item
  per the rationale above.
- Each "deferred" claim is justified by the orthogonality of the concern to
  the node-level contract model.
- No story replacement creates a scope ambiguity for the Epic 10 stories — the
  Epic 6 item's intent is fully covered by the Epic 10 enforcement layer.

## Memory update

Add a memory entry summarising this decision so future sessions do not re-litigate:

`project_epic6_closure_2026-04-28.md` — Epic 6 backlog closed. 6 items replaced by
Epic 10, 7 deferred post-MVP. Action #4 of Epic 9 retro resolved.
