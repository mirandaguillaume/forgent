#!/usr/bin/env bash
# Extract code comments from pr_diff.patch (added lines only).
# Output: JSON array of "[COMMENT] <text>" strings.
# In corpus mode CORPUS_ENTRY_DIR points to the pre-computed entry dir.
set -euo pipefail

DIFF_FILE="pr_diff.patch"
[ -n "${CORPUS_ENTRY_DIR:-}" ] && DIFF_FILE="${CORPUS_ENTRY_DIR}/pr_diff.patch"

if [ ! -f "$DIFF_FILE" ]; then
  echo '[]'
  exit 0
fi

{ grep '^+' "$DIFF_FILE" | grep -v '^+++' | sed 's/^+//' | \
  grep -E '^\s*(//|/\*|\*|#|<!--|.*TODO:|.*FIXME:|.*HACK:|.*NOTE:|.*XXX:)' || true; } | \
  sed 's/^\s*//' | sort -u | \
  jq -Rn '[inputs | "[COMMENT] " + .]'
