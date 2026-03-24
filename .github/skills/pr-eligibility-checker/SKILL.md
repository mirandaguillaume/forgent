---
name: pr-eligibility-checker
description: gate-check-based skill consuming pr_url to produce eligibility_status
---

# Pr Eligibility Checker

## Guardrails
- timeout: 1min
- fail_closed

## When to Use

Use for:
- before any automated code review
- when a PR URL is provided for review

**Don't use for:**
- for manual reviews without a PR
- for local-only code review

## Context
Consumes: pr_url
Produces: eligibility_status
Memory: short-term

## Strategy
Approach: gate-check
Tools: bash

### Steps
1. fetch PR metadata via gh
2. check if PR is closed, draft, automated, or already reviewed
3. return eligibility status with reason

## Red Flags

| Excuse | Reality |
|--------|--------|
| Skipping eligibility check | Reviewing closed/draft/automated PRs wastes tokens and creates noise |

## Security
- Filesystem: none
- Network: allowlist
- Secrets: GITHUB_TOKEN
