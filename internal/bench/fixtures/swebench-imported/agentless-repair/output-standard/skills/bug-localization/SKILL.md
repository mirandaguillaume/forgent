---
name: bug-localization
description: Analyze GitHub issues to localize bugs by identifying affected components-based skill consuming github_issue, codebase_files to produce bug_location_analysis
---

# Bug Localization

## Guardrails
- Focus only on identifying the bug location and root cause
- Do not make any code changes during analysis
- timeout: 300

## Context
Consumes: github_issue, codebase_files
Produces: bug_location_analysis
Memory: short-term

## Strategy
Approach: Analyze GitHub issues to localize bugs by identifying affected components
Tools: read_file, grep, search

### Steps
1. Read and parse the GitHub issue description to identify affected components
2. Search codebase to locate relevant files and functions/classes
3. Narrow down to the specific code paths involved in the reported behavior

## Security
- Filesystem: read-only
- Network: none
