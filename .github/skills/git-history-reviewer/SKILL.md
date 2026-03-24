---
name: git-history-reviewer
description: history-first-based skill consuming pr_diff, git_blame to produce review_issues
---

# Git History Reviewer

## Guardrails
- timeout: 5min
- max_history_depth: 20

## When to Use

Use for:
- when modified code has meaningful git history

**Don't use for:**
- for brand new files with no history

## Context
Consumes: pr_diff, git_blame
Produces: review_issues
Memory: short-term

## Strategy
Approach: history-first
Tools: bash, read_file

### Steps
1. read git blame for modified files
2. analyze commit history of modified code sections
3. identify bugs in light of historical context
4. flag regressions or patterns that previously caused issues

## Red Flags

| Excuse | Reality |
|--------|--------|
| Flagging pre-existing issues | Only flag issues introduced or worsened by this PR |

## Security
- Filesystem: read-only
- Network: none
