---
name: analyze-bug-root-cause
description: systematic code analysis to identify root cause-based skill consuming issue_description, code_context to produce bug_analysis
---

# Analyze Bug Root Cause

## Guardrails
- timeout: 300s
- must preserve existing functionality
- focus only on bug analysis, not test modifications

## Context
Consumes: issue_description, code_context
Produces: bug_analysis
Memory: short-term

## Strategy
Approach: systematic code analysis to identify root cause
Tools: read_file, grep, search

### Steps
1. analyze issue description for symptoms
2. examine provided code context
3. trace through code flow to identify bug location
4. determine minimal scope of changes needed

## Security
- Filesystem: read-only
- Network: none
