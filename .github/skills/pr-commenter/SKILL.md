---
name: pr-commenter
description: output-format-based skill consuming scored_issues, pr_url to produce review_comment
---

# Pr Commenter

## Guardrails
- timeout: 2min
- require_full_sha
- max_issues: 20

## When to Use

Use for:
- as final step of code review pipeline

**Don't use for:**
- for local-only review without GitHub integration

## Context
Consumes: scored_issues, pr_url
Produces: review_comment
Memory: short-term

## Strategy
Approach: output-format
Tools: bash

### Steps
1. re-verify PR eligibility to confirm not closed or merged since analysis
2. format issues with file links using full SHA
3. post comment via gh pr comment
4. if no issues, post no issues found message

## Red Flags

| Excuse | Reality |
|--------|--------|
| Using short SHA or HEAD references in links | GitHub markdown renders links at comment time; short refs may be ambiguous |

## Security
- Filesystem: none
- Network: allowlist
- Secrets: GITHUB_TOKEN
