# Forgent — Thesis

This document captures **what Forgent claims**, **why we built it this way**,
and **what would invalidate the bet**. It exists so a future contributor (or
future me) can rejoin the project without re-deriving the rationale from
scattered story files.

The thesis is layered: a narrow technical claim that we test empirically, and
broader architectural and product claims that we wager on. Each layer has a
clear validation path — or an honest "not testable by us alone".

---

## Layer 1 — Technical claim (testable)

**Claim:** On the project's 16-task corpus, structural enforcement at the
node level (per-step contracts) catches violations that keyword-based
guardrails miss.

**Operationalisation:** Compare two configurations of the same agent on the
same task:

- **Baseline:** keyword-string guardrails only (the 2026-04-16 sonnet baseline).
- **Treatment:** all node-level contracts active — file permissions
  (`file_access`), output contracts (REQUIRED markers), network policy,
  loop bounds, per-node violation telemetry.

The metric: `ContractComplianceRate` per (node, condition), aggregated with
bootstrap CI 95%, across `runs_per_task ≥ 3`.

**Validation:** Story `10-9-experiment-rerun.md`. Per the 2026-04-28 audit,
scope reduced to the 5-task overlap where a captured baseline exists
(`code-review`, `doc-generation`, `refactoring`, `security-audit`,
`test-generation`).

**Outcomes the experiment can produce:**

| Result | Reading |
|---|---|
| Treatment > baseline, CI excludes 0 | Layer 1 validated on the corpus |
| Treatment ≈ baseline, CI includes 0 | Keyword guardrails sufficient on this corpus; Layer 1 not validated |
| Treatment < baseline | Enforcement actively hurts; needs investigation |

**Limitations:** valid only on the corpus tested, with the model used. No
generalisation claim to "all agents" or "all tasks".

---

## Layer 2 — Architectural claim

**Claim:** When enforcement is needed, declaring it as **per-step
contracts that the runtime intercepts before execution** is strictly
preferable to:

- **Per-task guardrails** evaluated post-hoc on the final response (acts
  too late; the violation has already produced an effect).
- **Composable middleware chains** wrapped around each step (combinatorial
  complexity; behaviour depends on chain order).
- **Prompt-engineered string guardrails** as the only safety layer
  (textual proxies for semantic constraints; brittle to LLM stochasticity).

**Why per-step:** the boundary where the LLM emits a tool call IS the
point at which intervention is meaningful. Earlier (in the prompt) is
hope; later (post-hoc) is detection without prevention.

**Why declarative:** YAML contracts are auditable, comparable across
versions, and decoupled from the orchestration code. Imperative middleware
chains hide the contract in code.

**Validation:** indirect through Layer 1's results. If Theory A signal
appears (see Layer 4 — `prevented: true` events on adversarial tasks),
that proves intercept-before-execute captures real violations. Per-task
post-hoc enforcement could not capture the same events because the violation
would already have occurred. Composable middleware could in principle catch
them but without the auditability of declarative contracts.

The 2026-04-16 pivot decision (sprint-change-proposal-2026-04-07.md +
2026-04-16 party mode) committed to this architecture after Epic 5's
keyword-guardrail run produced ambiguous results. Epic 10 is the
implementation.

---

## Layer 3 — Product claim (multi-target compiler)

**Claim:** A single declarative agent specification (Forgent YAML) compiles
to multiple framework-native targets — Claude Code, GitHub Copilot, the
standalone Forgent runtime — with the same behavioural contract enforced
in each target's native primitives.

**Why a compiler, not a runtime:** the agent ecosystem is fragmented
across vendors. A library that locks users into one runtime gets stuck
behind the leader. A compiler that emits to whichever target the user
already adopted lets the contract be the unit of portability.

**Why the contract is the right portability unit:** prompts don't
port across runtimes (different system prompts, different tool conventions).
Code doesn't port (bound to a specific orchestrator). Declarative contracts
DO port — every target can implement "step X has read-only file access" in
its own way.

**Validation:** by adoption. We don't have a benchmark for "does this
contract translate well to Claude Code's primitives". That comes from
external users trying it. Internally we test by generating the targets
(`forgent build --target claude` etc.) and verifying the generated artefacts
satisfy the contract on the Forgent runtime first (Epic 10's tests provide
this floor).

**Risk to claim:** if a target backend can't implement the contract
(e.g., a runtime with no network sandbox primitive), the portability story
weakens to "best-effort across targets, full guarantees only on Forgent runtime".
This may be acceptable as a starting position.

---

## Layer 4 — Theory A vs Theory B (the durable framing)

This is the most important layer to internalise because it makes the
thesis defensible **regardless of what Layer 1 measures**.

There are two possible worlds about LLM behaviour:

| World | Description |
|---|---|
| **Theory A** | LLMs naturally drift toward violating constraints. Without structural enforcement, agents misbehave (writes to forbidden paths, exfiltrates data, etc.). With enforcement, the harness blocks the attempt. |
| **Theory B** | LLMs are aligned enough that they don't try to violate constraints in normal operation. They naturally comply. Enforcement layers fire rarely or never. |

The naive interpretation: "if Theory B is true, Forgent has no value because
nothing needed blocking." That interpretation is wrong.

**Forgent has value in BOTH worlds, because the value proposition is different
in each:**

- **In Theory A:** Forgent provides **active enforcement**. The harness
  catches and prevents real violation attempts. The agent simply cannot
  do what it would otherwise do. This is the safety claim most users
  expect.

- **In Theory B:** Forgent provides **passive attestation**. Even though
  the agent didn't try to violate, the runtime carries a *proof* that
  it could not have. The audit log shows zero `attempted` events, and
  the contract structurally prevented any. This is what compliance,
  certification, and forensic auditing need.

The crucial observation: **an agent that "behaved well" without proving
it could not have misbehaved is NOT equivalent to an agent that was
structurally constrained from the start, even if the observable outcomes
match.** The latter has a chain of causation; the former has only a
correlation.

This framing means the empirical experiment is interesting but not
existentially load-bearing. If 10-9 shows Theory B dominates on our
corpus, that's a finding — and Forgent still has value, just on
different grounds. If Theory A signal appears, that's better evidence
for a "performance" pitch but does not change the attestation argument.

---

## Layer 5 — Meta-thesis (the bet)

**Claim:** The right primitive for specifying LLM agents is the **declarative
structural contract**, not the prompt-engineered guardrail. The agent
ecosystem will move toward compiled contracts the same way the JavaScript
ecosystem moved from imperative jQuery to declarative React.

**Why this matters:** if the bet is right, Forgent's architecture
(YAML contracts → multi-target compiler) is on the trajectory. If the
ecosystem stays in prompt-engineering territory (prompt-as-code, ad-hoc
guardrails per agent), Forgent's positioning weakens regardless of
how well Layer 1's experiment validates.

**Validation:** by ecosystem adoption over time, not by any single
experiment. Comparable to the bet that types-as-tests would beat
runtime-only-tests in JavaScript — no single experiment proved it,
but the trajectory of TypeScript adoption did.

**Risk to claim:** an alternative primitive emerges that captures the
same value with less ceremony. Prime candidates:
- LLM-native multi-step protocols (e.g., Anthropic's MCP) where the
  contract is implicit in the protocol, not declared
- Self-improving runtimes that learn enforcement from experience (no
  declared contract needed; the runtime infers and enforces)
- Standardised tool sandboxes at the cloud-provider level (AWS / Anthropic
  hosting the enforcement, not the agent author declaring it)

Any of these could displace declarative-contract-as-primitive. That's the
bet.

---

## What 10-9 actually validates

Putting Layers 1–5 together, the experiment in `10-9-experiment-rerun.md`:

- **Directly tests Layer 1** on the 5-task overlap.
- **Indirectly supports Layer 2** if Theory A signal appears (proves the
  intercept-before-execute approach catches real events).
- **Supports Layer 4 framing** by producing the empirical numbers that
  classify each (node, violation_type) into Theory A / Theory B / mixed.
- **Cannot validate Layer 3 or 5** — those require external evidence
  (adoption, ecosystem trajectory).

A clear delta on Layer 1 with a coherent Theory A/B reading is sufficient
to claim Forgent has value. A null delta on Layer 1 with a Theory B reading
is *also* sufficient — just on attestation grounds rather than performance
grounds. **The only result that would meaningfully threaten the thesis is
a delta inversion** (treatment performs significantly worse than baseline)
without a clear cause.

---

## Honest limitations

1. **Single model, single corpus.** All claims are conditional on the model
   used (Sonnet for the captured baseline) and the 16-task corpus. We
   don't have evidence for behaviour on other models or other domains.
2. **API credit constraint.** Validation depends on running multiple
   conditions × multiple runs against a real LLM. Credit exhaustion has
   blocked progress repeatedly (Epic 5 retro action #1 still open).
3. **Internal coverage gap on tool-* tasks.** The 5-task overlap excludes
   the adversarial `tool-path-traversal`, `tool-accidental-destruction`,
   `tool-privilege-escalation` tasks where Theory A signal would be
   strongest. Their inclusion is deferred to T5b of story 10-9.
4. **Runtime determinism is partial.** Per the 2026-04-28 leak audit, ~6
   internal non-determinism leaks remain (concurrent ordering, map
   iteration, etc.). They affect trace reproducibility but not aggregate
   metrics. A "byte-for-byte audit log diff" claim is NOT supported today.
5. **Security claims are scoped.** Forgent is built for "developer-authored
   tasks running in trusted environment". Multi-tenant, prompt-injection-
   resistant, compliance-grade attestation are explicit non-goals at this
   stage and would each require dedicated hardening passes.

---

## Anchors

| Concept | Where it lives |
|---|---|
| Architectural pivot to runtime harness | `_bmad-output/implementation-artifacts/sprint-change-proposal-2026-04-07.md` |
| Epic 6 closure (architectural simplification) | `_bmad-output/implementation-artifacts/sprint-change-proposal-2026-04-28-epic6-closure.md` |
| Per-node contract enforcement (Layer 2 implementation) | Stories `10-1` through `10-9` |
| Theory A/B operationalisation | Story `10-8-per-node-experiment-metrics.md` |
| Determinism leak audit | `~/.claude/projects/.../memory/project_10_9_audit_2026-04-28.md` |
| Multi-target compiler positioning | `~/.claude/projects/.../memory/project_harness_positioning.md` |
| Existing baseline (5-task) | `experiment-sonnet-baseline.{text,runtime}.json` (local, not in git per `feedback_no_datasets`) |

---

## When to revisit this document

Update this document when ANY of:

- Story 10-9 lands with experiment results (record the empirical outcome
  and which layers were validated/invalidated)
- A target backend is added or removed (Layer 3 changes)
- The model substrate shifts substantially (e.g., GPT family added,
  Claude version-pinned upgrade) — Layer 1 results may no longer hold
- An alternative primitive surfaces in the ecosystem (Layer 5 risk
  materialises) — re-evaluate the bet
- A user adopts Forgent in a context the current scope doesn't cover
  (multi-tenant, compliance, etc.) — new validation work needed

Don't update for individual story implementations. The point of this
document is to outlive any single epic.
