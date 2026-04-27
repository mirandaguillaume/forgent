#!/usr/bin/env bash
#
# scripts/check-dead-code.sh — fail on unreachable functions across the codebase.
#
# Story 4-0 Task 5: dead-code lint rule. Carries the Epic 7 retrospective
# action #1 ("Lint for unused exported functions, prevent dead code") which
# was unaddressed in Epics 7 and 8. Epic 8 itself violated this by leaving
# `internal/doctor/rewrite/` and `runFixMode` orphaned after re-architecting
# `--fix` to deterministic scaffold.
#
# Uses `golang.org/x/tools/cmd/deadcode` (NOT staticcheck U1000) because:
#  - U1000 explicitly skips exported identifiers, treating them as part of a
#    package's public API. For an internal/-heavy CLI binary that's wrong;
#    we want to catch exported functions in internal/ that nothing reaches.
#  - `deadcode` does whole-program reachability analysis from main + tests.
#  - The `-test` flag treats functions called from tests as live, which
#    matches the AC #10 wording ("zero references outside its test file").
#
# Allowlist: a small set of functions are intentionally unreachable from
# main + tests today and must NOT cause CI to fail. Each entry is justified
# with a comment. Adding to this list should be a deliberate choice — it's
# a documented escape hatch, not a silent suppression.
#
# Usage:
#   scripts/check-dead-code.sh
#   DEADCODE_VERSION=...                # override pinned version
#
set -euo pipefail

DEADCODE_VERSION="${DEADCODE_VERSION:-v0.28.0}"
DEADCODE_BIN="${GOBIN:-${HOME}/go/bin}/deadcode"

if ! [[ -x "${DEADCODE_BIN}" ]]; then
    echo "Installing deadcode@${DEADCODE_VERSION}..." >&2
    go install -mod=mod "golang.org/x/tools/cmd/deadcode@${DEADCODE_VERSION}"
fi

# Allowlist of intentionally-unreachable functions.
# Each line is a substring matched against deadcode's output line.
# Keep this list SHORT and JUSTIFIED. Anything added here without a reason
# in this comment block should be considered a code smell.
#
# Allowed entries (2026-04-07 baseline after Story 4-0 cleanup):
#  - internal/bench/miniswe.go:58 RunMiniSWEBench
#  - internal/bench/miniswe.go:117 RunMiniSWEComparison
#      Both: unfinished mini-SWE-agent integration. Never wired into the
#      bench CLI. Kept as a partial implementation rather than deleted —
#      should be either finished or removed in Epic 5 (Benchmark & Evidence).
#      TODO(epic-5): wire into bench CLI or delete.
#  - pkg/dag/execute.go:27 WithConcurrency
#      Public API in `pkg/dag` (functional option for DAG execution).
#      Internal callers don't use it, but the package is exported for
#      external consumers. Removing would be a breaking change to the
#      pkg/dag public surface. Cannot be detected by deadcode because
#      external use is invisible to whole-program analysis.
ALLOWLIST=(
    "internal/bench/miniswe.go:58:6: unreachable func: RunMiniSWEBench"
    "internal/bench/miniswe.go:117:6: unreachable func: RunMiniSWEComparison"
    "pkg/dag/execute.go:27:6: unreachable func: WithConcurrency"
)

raw="$("${DEADCODE_BIN}" -test ./... 2>&1 || true)"

# Filter the allowlist out of the raw findings.
filtered="${raw}"
for entry in "${ALLOWLIST[@]}"; do
    # Use grep -v -xF (fixed-string, full-line match) to drop exact matching lines.
    # -x anchors the match to the entire line, preventing an allowlist entry from
    # accidentally suppressing a different finding that happens to contain the same
    # substring (e.g., a longer function name on the same file path).
    # `|| true` prevents grep returning 1 (no remaining matches) from killing the script.
    filtered="$(printf '%s\n' "${filtered}" | grep -v -xF "${entry}" || true)"
done

# Strip empty lines.
filtered="$(printf '%s\n' "${filtered}" | grep -v '^$' || true)"

if [[ -n "${filtered}" ]]; then
    echo "Dead-code lint failed (golang.org/x/tools/cmd/deadcode):" >&2
    echo "${filtered}" >&2
    echo "" >&2
    echo "Each finding above is a function unreachable from main or any test." >&2
    echo "Either delete it (preferred — see CLAUDE.md \"if certain something" >&2
    echo "is unused, you can delete it completely\") or add it to the ALLOWLIST" >&2
    echo "in this script with a comment explaining why it must stay." >&2
    exit 1
fi

echo "Dead-code lint: clean (allowlist entries: ${#ALLOWLIST[@]})"
