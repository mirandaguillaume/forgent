---
name: github-issue-analyzer
description: Analyze GitHub issues to understand bugs in Python code-based skill consuming github_issue, python_codebase to produce issue_analysis
---

# Github Issue Analyzer

## Guardrails
- Focus only on understanding the problem, not implementing fixes
- Only analyze standard Python code
- Maximum 15 minutes per issue analysis

## Context
Consumes: github_issue, python_codebase
Produces: issue_analysis
Memory: short-term

## Strategy
Approach: Analyze GitHub issues to understand bugs in Python code
Tools: read_file, grep, search

### Steps
1. Read and understand the GitHub issue to identify the bug
2. Analyze relevant code files to understand current behavior
3. Determine what changes are needed to fix the issue

## Security
- Filesystem: read-only
- Network: none
