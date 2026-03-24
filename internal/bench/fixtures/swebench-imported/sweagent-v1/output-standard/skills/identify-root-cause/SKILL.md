---
name: identify-root-cause
description: Identify the root cause of the issue through detailed code examination-based skill consuming code_analysis, issue_description to produce root_cause
---

# Identify Root Cause

## Guardrails
- Timeout after 180 seconds of root cause analysis
- Focus analysis on the most relevant code sections

## Context
Consumes: code_analysis, issue_description
Produces: root_cause
Memory: short-term

## Strategy
Approach: Identify the root cause of the issue through detailed code examination
Tools: read_file, grep

### Steps
1. Examine the analyzed code components for potential issues
2. Trace through the logic to identify the source of the problem
3. Determine the specific cause that needs to be addressed

## Security
- Filesystem: read-only
- Network: none
