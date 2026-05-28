#!/usr/bin/env bash
# Generates git_history.txt for a corpus entry.
# Env vars provided by the runner: CORPUS_OWNER, CORPUS_REPO, CORPUS_PR_NUM
set -euo pipefail

if [ -n "${CORPUS_ENTRY_DIR:-}" ] && [ -f "${CORPUS_ENTRY_DIR}/git_history.txt" ]; then
  cat "${CORPUS_ENTRY_DIR}/git_history.txt"
  exit 0
fi

if [ -z "${CORPUS_OWNER:-}" ]; then
  echo "# Git history (standalone mode — no corpus)"
  exit 0
fi

base_sha=$(gh api "repos/$CORPUS_OWNER/$CORPUS_REPO/pulls/$CORPUS_PR_NUM" --jq '.base.sha')
echo "# Git history at base commit ${base_sha:0:7}"

gh api "repos/$CORPUS_OWNER/$CORPUS_REPO/pulls/$CORPUS_PR_NUM/files" \
  --jq '.[].filename' | head -15 | while IFS= read -r path; do
  commits=$(gh api "repos/$CORPUS_OWNER/$CORPUS_REPO/commits" \
    -f "path=$path" -f "sha=$base_sha" -f "per_page=10" \
    --jq '.[] | "\(.sha[0:7]) \(.commit.committer.date[0:10]) \(.commit.author.name): \(.commit.message | split("\n")[0] | .[0:120])"' \
    2>/dev/null) || true
  if [ -n "$commits" ]; then
    printf "\n## %s\n%s\n" "$path" "$commits"
  fi
done
