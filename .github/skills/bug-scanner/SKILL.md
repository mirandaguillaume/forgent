---
name: bug-scanner
description: diff-first-based skill consuming pr_diff to produce review_issues
---

# Bug Scanner

## Guardrails
- timeout: 5min
- no_nitpicks
- no_style_issues

## When to Use

Use for:
- always during code review

**Especially when:**
- when reviewing complex logic changes

## Context
Consumes: pr_diff
Produces: review_issues
Memory: short-term

## Strategy
Approach: diff-first
Tools: read_file

### Steps
1. read the PR diff only, no expanded context
2. scan for obvious bugs like null derefs, off-by-one, logic errors
3. ignore likely false positives
4. return issues with severity location

## Red Flags

| Excuse | Reality |
|--------|--------|
| Flagging linter/typechecker issues | These are caught by CI; duplicating them adds noise |
| Flagging intentional functionality changes | Changes related to the PR purpose are likely intentional |
| Reading beyond the diff | Shallow scan focuses on changed lines only |

## Security
- Filesystem: read-only
- Network: none
