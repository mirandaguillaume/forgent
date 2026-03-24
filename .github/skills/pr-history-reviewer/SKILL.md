---
name: pr-history-reviewer
description: history-first-based skill consuming pr_diff, pr_history to produce review_issues
---

# Pr History Reviewer

## Guardrails
- timeout: 5min
- max_prs: 10

## When to Use

Use for:
- when files have been reviewed in previous PRs

**Don't use for:**
- for repositories with no PR history

## Context
Consumes: pr_diff, pr_history
Produces: review_issues
Memory: short-term

## Strategy
Approach: history-first
Tools: bash

### Steps
1. find previous PRs that touched the same files
2. read comments on those PRs
3. check if any feedback applies to the current PR
4. return applicable issues with links to original comments

## Security
- Filesystem: none
- Network: allowlist
- Secrets: GITHUB_TOKEN
