---
name: pr-reviewer
description: diff-first-based skill consuming git_diff, test_results, lint_results to produce review_comments, risk_score
---

# Pr Reviewer

## Guardrails
- max_comments: 15
- timeout: 5min
- no_approve_without_tests
- require_risk_score

## Context
Consumes: git_diff, test_results, lint_results
Produces: review_comments, risk_score
Memory: conversation

## Dependencies
- **tdd-runner** provides `test_results`
- **ts-linter** provides `lint_results`

## Strategy
Approach: diff-first
Tools: read_file, grep, search

### Steps
1. read the git diff
2. cross-reference with test results
3. cross-reference with lint results
4. identify risky changes and anti-patterns
5. write actionable review comments
6. assign risk score (0-10)

## Security
- Filesystem: read-only
- Network: none
