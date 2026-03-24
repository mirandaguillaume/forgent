---
name: code-review-reporter
description: comprehensive review report generation-based skill consuming security-findings, quality-metrics, review-context to produce code-review-report
---

# Code Review Reporter

## Guardrails
- timeout: 10 minutes for report generation
- require_findings_input: true

## Context
Consumes: security-findings, quality-metrics, review-context
Produces: code-review-report
Memory: conversation

## Strategy
Approach: comprehensive review report generation
Tools: write_file

### Steps
1. aggregate security and quality findings
2. prioritize issues by criticality
3. generate actionable recommendations
4. format comprehensive review report

## Security
- Filesystem: read-write
- Network: none
