---
name: ts-linter
description: static-analysis-based skill consuming file_tree, source_code to produce lint_results
---

# Ts Linter

## Guardrails
- timeout: 5min
- max_file_size: 500KB

## Context
Consumes: file_tree, source_code
Produces: lint_results
Memory: short-term

## Strategy
Approach: static-analysis
Tools: bash, read_file, search

### Steps
1. check for common anti-patterns
2. produce structured lint report

## Security
- Filesystem: read-only
- Network: none
