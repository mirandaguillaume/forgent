---
name: issue-confidence-scorer
description: scoring-based skill consuming review_issues, claudemd_files to produce scored_issues
---

# Issue Confidence Scorer

## Guardrails
- timeout: 2min
- min_score_threshold: 80

## When to Use

Use for:
- after collecting review issues from multiple analyzers

**Don't use for:**
- as standalone without preceding analysis skills

## Context
Consumes: review_issues, claudemd_files
Produces: scored_issues
Memory: short-term

## Strategy
Approach: scoring
Tools: read_file

### Steps
1. for each issue, evaluate confidence on 0-100 scale
2. for CLAUDE.md issues, double-check rule actually exists
3. apply scoring rubric (0 false positive, 25 uncertain, 50 nitpick, 75 likely real, 100 certain)
4. filter out issues scoring below 80

## Security
- Filesystem: read-only
- Network: none
