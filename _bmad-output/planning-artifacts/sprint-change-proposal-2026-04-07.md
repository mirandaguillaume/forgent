---
date: 2026-04-07
author: Guillaume (facilitated by Bob — Scrum Master, with John and Mary)
trigger_type: Opportunistic resequencing — vision intact, schedule slipped
scope_classification: Moderate (backlog reorganization, no PRD or epic-scope changes)
status: Approved
related_artifacts:
  - _bmad-output/planning-artifacts/prd.md (unchanged — north star)
  - _bmad-output/planning-artifacts/epics.md (unchanged — scope intact)
  - _bmad-output/implementation-artifacts/sprint-status.yaml (updated)
  - _bmad-output/implementation-artifacts/sprint-review-2026-03-30.md (referenced)
---

# Sprint Change Proposal — 2026-04-07

## Section 1: Issue Summary

### Problem Statement

Between Sprint 2 (completed 2026-03-30) and today (2026-04-07), the team executed **Epic 7 (Static Analysis Pipeline)** and **Epic 8 (Scaffold + Index Architecture)** instead of the originally-planned **Epic 4 (Doctor as Forgent Runtime Agent — KEYSTONE)** and **Epic 5 (Benchmark & Evidence)**.

The vision remains intact — Epic 7 and Epic 8 ship genuine value and are defensible against the PRD. The **schedule** has slipped: Epic 4 and Epic 5, both of which are *load-bearing for the PRD's central claim*, are 1-2 sprints late. This proposal records the deviation, the decision behind it, and the corrective resequence.

### How it was discovered

A vision-drift check was run on 2026-04-07 via `bmad-party-mode` with John (Product Manager), Bob (Scrum Master), and Mary (Business Analyst). The check compared the PRD's Success Criteria, the original Sprint Timeline (`sprint-status.yaml` revised 2026-03-31), and the actual git/sprint history. The deviation was visible in three independent signals.

### Evidence

| Signal | Source | Reading |
|---|---|---|
| Stories `4-1`, `4-2`, `4-3`, `5-1`, `5-2`, `5-3` are `ready-for-dev` and untouched for 7 days | `sprint-status.yaml` lines 84-95 | Original Sprint 3-4 work was created but never picked up |
| Epic 4 explicitly labeled **(KEYSTONE)** in pre-correction sprint review | `sprint-review-2026-03-30.md` line 121, `epics.md` line 211 | Team's most recent canonical sprint review identified Epic 4 as the load-bearing epic, *and skipped it anyway* |
| Epic List in `epics.md` contains only Epic 1-6 | `epics.md` lines 195-219 | Epic 7 (added "from Technical Research") and Epic 8 (added "from bench learnings") were appended *after* the original PRD scope was set — they are "bonus epics", not part of the original scoped delivery |
| `last_story_created: 7-5-agent-smells` in sprint status | `sprint-status.yaml` line 49 | Sprint focus moved fully into Epic 7/8 territory; the original Sprint 3-4 stories were never started |

### Categorization

- **NOT** failed approach
- **NOT** new requirement from stakeholder
- **NOT** misunderstanding of original requirements
- **NOT** strategic pivot
- **IS** opportunistic resequencing — the team discovered Epic 7 static checks were higher-leverage than originally scored while in the code, and Epic 8 emerged from bench infrastructure work that was prerequisite-shaped. Both substitutions had real engineering rationale at the time. The cost is the schedule slip, not the strategy.

## Section 2: Impact Analysis

### Epic Impact

| Epic | Status | Impact of correction |
|---|---|---|
| 1, 2, 3 | done | None — already shipped |
| **4 (KEYSTONE)** | backlog → **next** | Resequenced forward to Sprint 5 (was Sprint 3). All 3 stories `ready-for-dev`, no rework needed |
| **5** | backlog → **after Epic 4** | Resequenced forward to Sprint 6 (was Sprint 4). All 3 stories `ready-for-dev`, no rework needed. **High attention** on fear-of-result deferral pattern (see §3) |
| **6** | backlog → after Epic 5 | Resequenced to Sprint 7 (was Sprint 5). No content change |
| **7** | done | Acknowledged as opportunistic pull-forward. Retro already done (`epic-7-retro-2026-03-31.md`) |
| **8** | in-progress (8.1, 8.2, 8.3 done) | Acknowledged as bench-learnings emergent epic. Retro pending — should be completed *before* Epic 4 starts |

**Net epic impact:** Zero scope changes. Three sequence changes (4, 5, 6 each shift forward by 2 sprint slots).

### Story Impact

- No stories added, removed, modified, or rewritten
- All Epic 4-6 stories remain `ready-for-dev` exactly as drafted
- `sprint-status.yaml` `last_story_created` field updated from `7-5-agent-smells` to `8-3-complete-templates` (housekeeping fix)

### Artifact Conflicts

| Artifact | Conflict? | Action |
|---|---|---|
| **PRD** (`prd.md`) | NO | **Unchanged.** Held as north star per Guillaume's decision. Epic 7/8 work is PRD-defensible (research-backed quality checks fall under "12 research-backed checks" framing; scaffolding aligns with "AGENTS.md surface standardization" mention) but not load-bearing for the central claim. |
| **Epics** (`epics.md`) | Mild | The Epic List header (lines 195-219) contains only Epic 1-6, but Epic 7 and Epic 8 have full sections later in the file (lines 753+, 907+). This is an inconsistency — but resolving it requires deciding whether Epic 7/8 are *officially scoped* or *bonus work*. Deferred to a separate epics-catalog cleanup task. |
| **Architecture** (`architecture.md`) | Yes (deferred) | Doesn't reference Epic 7/8 packages (`internal/static/`, `internal/scaffold/`, etc.) because the doc was written before these epics existed. Architecture retrofit is a documentation task, out of scope for this correction. Flagged as follow-up. |
| **UX** | N/A | CLI project, no UX spec |
| **Sprint Status** (`sprint-status.yaml`) | Yes | Update timeline header + `last_updated` + `last_story_created`. See edit proposals. |

### Technical Impact

- **Code:** Zero. No rollback, no rewrite, no new code from this correction.
- **Tests:** Zero changes from this correction. (Epic 4 will require test work when it starts, but that's per-story.)
- **CI/CD:** Zero changes.
- **Documentation:** Sprint plan documentation only (`sprint-status.yaml` header).

## Section 3: Recommended Approach

### Selected: **Option 1 — Direct Adjustment (Resequence Forward)**

**Action:** Update `sprint-status.yaml` to reflect what actually happened in Sprints 3-4 and to set Epic 4 as the next priority. Write this proposal as the formal record. Hold PRD and epics.md unchanged.

### Rationale

- **Effort:** LOW. Two file edits + one new artifact. No code, no rollback, no rework.
- **Risk:** LOW. Reverting this correction is trivial (revert two lines). Nothing destructive.
- **Vision impact:** Positive. Recovers schedule alignment without rewriting strategy.
- **Team morale:** Neutral-to-positive. Acknowledges Epic 7/8 as real shipped value rather than retroactively framing them as mistakes. Frames the recovery as "snap back to plan", not "you screwed up".

### Alternatives considered and rejected

| Option | Rejected because |
|---|---|
| **Option 2 — Rollback Epic 7/8** | Destroys real value. Epic 7's static analysis pipeline measurably improves Doctor's recommendations. Epic 8's scaffold + index architecture is prerequisite-shaped for InstructionBench (it gives Epic 5 the corpus index it will need). Rolling back would set the project back without restoring vision alignment any faster than just resequencing forward. |
| **Option 3 — Update PRD/MVP Scope** | Would amount to *retrofitting the strategy to match what we built*, which is the exact pattern Mary warned against in the party-mode discussion. The PRD's central claim ("structural enforcement gives +20-33%, text instructions +4% at best") is *falsifiable* — that's a feature, not a bug. Rewriting the PRD now would protect us from ever testing it. |
| **Option 4 — Status quo, no formal correction** | Process debt. Without a paper trail, future-team has no record of *why* the sprint plan deviated. Six months from now, this would look like undocumented chaos instead of a deliberate (if costly) opportunistic pull-forward. |

### Critical attention point — Epic 5 deferral risk

Mary's diagnosis from the party-mode discussion: **Epic 5 has structural properties that make it the highest-deferral-risk epic in the entire backlog.** It is the *one epic that could prove the runtime claim wrong*. When Epic 4 finishes and Epic 5 begins, the team will encounter genuine-sounding reasons to delay (corpus needs more curation, methodology needs another iteration, runtime needs one more feature for fairness). Each will be legitimate. Each will also be a tractable adjacency that delays the load-bearing measurement.

**Mitigation built into this proposal:**
1. Epic 5 is sequenced *immediately* after Epic 4 — no slack epics in between to hide behind
2. The "fear-of-result deferral pattern" is documented in this artifact and in Claude memory (`memory/project_vision_drift_2026-04-07.md`) so future sessions can surface it
3. **Epic 5 should ship a v0 result** with rough corpus and rough methodology, *then* iterate. The PRD itself says: *"Ship the smallest thing that validates the assumption."* Hold John to his own words.

If Epic 5 is deferred *again* after this correction — even with legitimate-sounding reasons — that would be a **second** correction-of-course event and should escalate to a more serious vision review.

## Section 4: Detailed Change Proposals

### Change 4.1: `sprint-status.yaml` — Timeline header

(See Edit Proposal #1 in main conversation. Records actual Sprint 3-4 execution and revised forward sequence.)

### Change 4.2: `sprint-status.yaml` — `last_updated` and `last_story_created`

(See Edit Proposal #2 in main conversation. Housekeeping fix.)

### Change 4.3: `sprint-change-proposal-2026-04-07.md`

This file. The formal correction-of-course artifact.

### Changes explicitly NOT made (recorded for future reference)

- **PRD unchanged** — held as north star
- **Epics.md Epic List header unchanged** — Epic 7/8 inclusion is a separate hygiene decision
- **Architecture.md unchanged** — retrofit is a separate documentation task
- **No code changes** — Epic 7/8 work stays as shipped; no rollback

## Section 5: Implementation Handoff

### Scope classification: **Moderate**

- More than minor (it's not just an in-place code edit)
- Less than major (no PRD rewrite, no architectural redesign, no scope change)
- Requires: backlog reorganization in `sprint-status.yaml` + formal artifact creation + handoff to next-epic owner

### Handoff plan

| Recipient | Responsibility | Trigger |
|---|---|---|
| **Bob (Scrum Master)** | Apply Edit Proposals #1 and #2 to `sprint-status.yaml`. Verify Epic 8 retrospective completion (or mark optional in status file). | Immediately upon Guillaume's approval |
| **Bob (Scrum Master)** | After Epic 8 wraps cleanly, transition `4-1-doctor-skill-and-agent-yaml` from `ready-for-dev` to `in-progress`. | Once Epic 8 retro is closed |
| **Amelia (Developer)** or **Barry (Quick Flow Solo Dev)** | Execute Story 4.1 via `bmad-dev-story` skill. | Once Bob transitions story status |
| **Future Claude session** | Watch for fear-of-result deferral pattern when Epic 5 begins. Surface explicitly if any "the corpus needs more curation" / "methodology needs iteration" pattern appears. Hold the v0 budget. | When Epic 5 stories enter `in-progress` |

### Success criteria for the correction itself

This correction is **successful** if:

1. ✅ `sprint-status.yaml` accurately reflects Sprint 3-4 actuals (Epic 7 and Epic 8) and shows Epic 4 next
2. ✅ This proposal artifact exists and is discoverable
3. ⏳ Epic 4 Story 4.1 enters `in-progress` within 1 working day of Epic 8 retro closure
4. ⏳ Epic 4 ships within ~2 sprints of starting (no further deferral)
5. ⏳ Epic 5 v0 ships within ~1-2 sprints of Epic 4 closure (the load-bearing test of the PRD's central claim)

Items 3-5 are the *real* test. Items 1-2 are necessary but not sufficient.

### What this proposal does NOT do

- Does not update the PRD or revise vision (Guillaume's explicit decision)
- Does not retrofit `architecture.md` for Epic 7/8 (separate task)
- Does not formally adopt Epic 7/8 into the canonical Epic List in `epics.md` (separate hygiene task — would require deciding scope vs bonus framing)
- Does not assign blame. Epic 7 and 8 are real, valuable work. The error is sequence, not substance.

---

## Approval

- [x] **Approved** by Guillaume on 2026-04-07
- Implementation handoff: Bob (Scrum Master) — sprint-status.yaml edits applied 2026-04-07
- Next action: Verify Epic 8 retrospective state, then transition `4-1-doctor-skill-and-agent-yaml` from `ready-for-dev` to `in-progress`
