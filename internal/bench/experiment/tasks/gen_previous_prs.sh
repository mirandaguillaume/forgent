#!/usr/bin/env bash
# Generates previous_prs.txt for a corpus entry.
# Env vars provided by the runner: CORPUS_OWNER, CORPUS_REPO, CORPUS_PR_NUM
set -euo pipefail

if [ -n "${CORPUS_ENTRY_DIR:-}" ] && [ -f "${CORPUS_ENTRY_DIR}/previous_prs.txt" ]; then
  cat "${CORPUS_ENTRY_DIR}/previous_prs.txt"
  exit 0
fi

if [ -z "${CORPUS_OWNER:-}" ]; then
  echo "# (none found)"
  exit 0
fi

pr_created=$(gh api "repos/$CORPUS_OWNER/$CORPUS_REPO/pulls/$CORPUS_PR_NUM" --jq '.created_at')
files=$(gh api "repos/$CORPUS_OWNER/$CORPUS_REPO/pulls/$CORPUS_PR_NUM/files" --jq '[.[].filename]')

echo "# Previous merged PRs touching the same files:"
found=0

for page in 1 2; do
  prs=$(gh api "repos/$CORPUS_OWNER/$CORPUS_REPO/pulls" \
    -f state=closed -f sort=updated -f direction=desc -f per_page=30 -f "page=$page")

  while IFS= read -r p; do
    p_num=$(echo "$p" | jq -r '.number')
    p_title=$(echo "$p" | jq -r '.title')
    merged=$(echo "$p" | jq -r '.merged_at')

    [ "$p_num" = "$CORPUS_PR_NUM" ] && continue
    [ "$merged" = "null" ] && continue
    [[ "$merged" > "$pr_created" || "$merged" = "$pr_created" ]] && continue

    p_files=$(gh api "repos/$CORPUS_OWNER/$CORPUS_REPO/pulls/$p_num/files" \
      --jq '[.[].filename]' 2>/dev/null) || continue

    overlap=$(jq -n \
      --argjson a "$files" --argjson b "$p_files" \
      '[$a[], $b[]] | group_by(.) | map(select(length > 1) | .[0]) | join(", ")' \
      2>/dev/null) || true

    [ -z "$overlap" ] && continue

    echo "PR #$p_num: \"$p_title\" — merged ${merged:0:10} — files: $overlap"
    found=$((found + 1))
    [ $found -ge 20 ] && exit 0
  done < <(echo "$prs" | jq -c '.[]' 2>/dev/null)
done

[ $found -eq 0 ] && echo "# (none found)"
