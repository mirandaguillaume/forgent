# Sprint Change Proposal — 2026-05-13
# SWE-Bench Validation + 10-9 Redefinition

**Triggered by:** Story 10-9 (experiment-rerun) completion analysis — "Est-ce que les résultats sont probants ?"
**Scope classification:** Moderate
**Handoff:** Development team (10-12 implementation) + Product Owner (PRD FR-SWE addition)

---

## 1. Issue Summary

Story 10-9 was originally scoped to validate Epic 10's structural enforcement (FR45) via a
5-task adversarial experiment measuring Detection Rate (violations prevented / attempted).
What was actually delivered during the sprint was a different but valid experiment: the
Martian CR-Bench 50-task run comparing text / runtime / specialists conditions.

Simultaneously, a strategic question emerged: Forgent is a general-purpose agent framework,
not a code-review tool. The right scientific venue for proving framework value is not only
the internal enforcement metric (FR45) but also an established external benchmark recognized
by the ML/agent community — specifically SWE-Bench, where competing frameworks (SWE-agent,
AutoCodeRover, Agentless, OpenDevin) all report.

Two findings drive this proposal:
1. The martian specialists experiment produced statistically suggestive results (+7pp recall,
   -10% tokens) but lacks bootstrap CI and runs-per-task replication for scientific claims.
2. The FR45 enforcement experiment (T5) was never run — and carries a "Theory B risk" where
   naturally-compliant LLMs produce zero violation attempts, making FR45 unmeasurable.

---

## 2. Impact Analysis

### Epic Impact

**Epic 10 (in-progress):**
- Story 10-9 scope mismatch: martian specialists experiment is a valid deliverable but
  different from the originally planned enforcement validation.
- T5 (enforcement run) deferred post-MVP — Theory B risk + API credit cost + diminishing
  returns vs SWE-Bench for scientific credibility.
- New story 10-12 (SWE-Bench validation) added. Infrastructure is substantially ready:
  `swebench.go` (1048 lines), `--level swebench`, `--harness` flag, `--swe-size` dataset
  selection. Missing: `resolve-issue.yaml` agent spec + Docker harness smoke test.

**Future epics:** None affected. SWE-Bench is additive.

### Story Impact

| Story | Change |
|-------|--------|
| 10-9 | Redefined: martian specialists as primary deliverable, T5-enforcement → deferred |
| 10-12 (NEW) | SWE-Bench validation — resolve-issue.yaml + delta conditions proof |

### Artifact Conflicts

**PRD:** Two scientific claims now coexist (see §3). Minor PRD section addition required.

**Architecture:** No change — SWE-Bench reuses existing runtime.

**UI/UX:** N/A.

### Technical Impact

`swebench.go` comment explicitly states: "Does NOT run FAIL_TO_PASS tests — full SWE-bench
harness integration is future work." Story 10-12 completes this integration.

---

## 3. Recommended Approach — Direct Adjustment

### Rationale

- No rollback needed — martian specialists work is valid and complete.
- No MVP reduction — both FR45 and FR-SWE claims are maintained.
- SWE-Bench infrastructure is 80% done; marginal effort to close.
- Deferring T5-enforcement is low risk: FR45 is still claimed, just not yet proven at scale.
  The structural enforcement code (10-1 through 10-7) is shipped and tested.

### Scientific Claims (both maintained)

**FR45 — Enforcement claim (internal)**
> Structural enforcement (file permissions, network sandbox, output contracts, loop bounds)
> prevents violations that keyword-only guardrails miss.
> Status: Code shipped (10-1..10-7). Proof deferred to post-MVP adversarial run.

**FR-SWE — Framework value claim (external)**
> Forgent-runtime and Forgent-ranked conditions outperform Forgent-text on SWE-Bench —
> the delta between conditions, not the absolute score, proves framework value.
> Target: +5-10pp runtime vs text on SWE-Bench Lite (300 tasks).
> Note: absolute score realistic range is 6-18% depending on harness integration depth.

### On Statistical Validity

Current martian results (+7pp recall, 50 tasks, 1 run/task) are directionally strong but
not scientifically publishable without:
- Bootstrap 95% CI
- ≥3 runs/task (variance)
- Symmetric task coverage (no N/A asymmetry)

Story 10-12 should include 3 runs/task on a representative SWE-Bench sample (Lite or
Verified subset) to meet publication standard.

---

## 4. Detailed Change Proposals

### Story 10-9 — Redefinition

**OLD scope:**
- T5: run 5-task enforcement experiment (text/runtime/ranked, 3 runs)
- Measure Detection Rate, Theory A/B, ContractComplianceRate

**NEW scope:**
- T5-martian (DONE): 50-task Martian CR-Bench, specialists condition
  Results: text 22% / runtime 34% / specialists 41% recall, -10% tokens
- T5-enforcement: DEFERRED post-MVP (Theory B risk + priority shift to SWE-Bench)
- Status: → done

**Rationale:** The martian run is a valid experiment deliverable. T5-enforcement carries
high Theory B risk and lower scientific leverage than SWE-Bench for external audience.

---

### Story 10-12 (NEW) — SWE-Bench Validation

**Goal:** Prove that Forgent-runtime/ranked > Forgent-text on SWE-Bench. Establish Forgent
on an externally recognized benchmark.

**Tasks:**
- T1: Write `resolve-issue.yaml` — DAG workflow: read issue → explore repo → generate patch
  → validate (output contract: unified diff). Specialists optional (complexity-specialist,
  security-specialist callable on-demand by main resolver).
- T2: Docker harness smoke test — verify `--harness` flag end-to-end on 5 tasks.
- T3: Run SWE-Bench Lite (300 tasks) × 3 conditions (text/runtime/specialists) × 1 run.
  Checkpoint-resumable. Output: `predictions.jsonl` per condition.
- T4: Compute delta text vs runtime vs specialists. Bootstrap CI (n=300 sufficient for 95% CI).
- T5: Document results in `epic-10-swebench-results.md`. Update PRD FR-SWE section.

**Acceptance criteria:**
- AC1: `resolve-issue.yaml` passes forgent lint
- AC2: Docker harness runs end-to-end on 5 SWE-Bench tasks
- AC3: Delta runtime vs text is statistically significant (bootstrap CI non-zero)
- AC4: Results documented and committed (no raw JSON in git — policy)

**Effort estimate:** Medium (3-5 days)
**Risk:** Medium — Docker harness integration may surface gaps; Theory B applies here too
if agents naturally comply without explicit violations.

---

### PRD Addition — FR-SWE

**Section to add** (under "Scientific Claims" or equivalent):

```
FR-SWE: External Benchmark Validation
Forgent-generated agents demonstrate measurable delta between execution modes
(text < runtime < specialists) on SWE-Bench Lite, establishing framework value
independent of absolute score. Target: +5-10pp runtime vs text.
```

---

## 5. Implementation Handoff

| Role | Responsibility |
|------|---------------|
| Dev team | 10-12 T1-T4: resolve-issue.yaml + harness + run |
| Product Owner | PRD FR-SWE section addition, sprint-status.yaml update |
| SM | 10-9 closure, 10-12 backlog entry |

**Success criteria:**
- 10-9 marked done in sprint-status.yaml
- 10-12 added with status backlog
- PRD updated with FR-SWE claim
- SWE-Bench run produces statistically significant delta

**Timeline impact:** Epic 10 retrospective delayed ~1 sprint to accommodate 10-12.
Epic 10 closes after 10-12 done.
