---
name: pr-summarizer
description: diff-first-based skill consuming pr_url to produce pr_summary
---

# Pr Summarizer

## Guardrails
- timeout: 2min
- max_summary_length: 500

## When to Use

Use for:
- when a human-readable PR summary is needed before deeper analysis

## Context
Consumes: pr_url
Produces: pr_summary
Memory: short-term

## Strategy
Approach: diff-first
Tools: bash

### Steps
1. fetch PR diff via gh
2. summarize the nature of changes
3. return structured summary with files changed, purpose, risk areas

## Security
- Filesystem: none
- Network: allowlist
- Secrets: GITHUB_TOKEN
