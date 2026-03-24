---
name: claudemd-auditor
description: compliance-check-based skill consuming pr_diff, claudemd_files to produce review_issues
---

# Claudemd Auditor

## Guardrails
- timeout: 5min
- require_citation

## When to Use

Use for:
- when CLAUDE.md files exist and compliance must be checked

**Don't use for:**
- when no CLAUDE.md files were found

## Context
Consumes: pr_diff, claudemd_files
Produces: review_issues
Memory: short-term

## Strategy
Approach: compliance-check
Tools: read_file, grep, search

### Steps
1. extract actionable rules from CLAUDE.md files
2. read the PR diff
3. check each change against applicable CLAUDE.md rules
4. for each violation, cite the specific CLAUDE.md rule

## Red Flags

| Excuse | Reality |
|--------|--------|
| Flagging non-applicable CLAUDE.md rules | CLAUDE.md is guidance for writing code; not all rules apply during review |
| Flagging explicitly silenced issues | Lint ignore comments and similar silencers are intentional |

## Security
- Filesystem: read-only
- Network: none
