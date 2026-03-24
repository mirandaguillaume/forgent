---
name: code-comment-auditor
description: compliance-check-based skill consuming pr_diff, source_code to produce review_issues
---

# Code Comment Auditor

## Guardrails
- timeout: 5min

## When to Use

Use for:
- when modified files contain meaningful code comments

## Context
Consumes: pr_diff, source_code
Produces: review_issues
Memory: short-term

## Strategy
Approach: compliance-check
Tools: read_file, grep

### Steps
1. read code comments in modified files like TODOs, warnings, invariants
2. check if PR changes comply with comment guidance
3. flag violations where changes contradict documented invariants

## Red Flags

| Excuse | Reality |
|--------|--------|
| Treating all comments as requirements | Some comments are notes, not constraints; focus on invariants and warnings |

## Security
- Filesystem: read-only
- Network: none
