---
name: ts-linter
description: static-analysis-based skill consuming file_tree, source_code to produce lint_results, type_errors
---

# Ts Linter

## Guardrails
- timeout: 5min
- max_file_size: 500KB

## Context
Consumes: file_tree, source_code
Produces: lint_results, type_errors
Memory: short-term

## Strategy
Approach: static-analysis
Tools: bash, read_file, search

### Steps
1. run TypeScript compiler in noEmit mode
2. collect type errors with file locations
3. check for common anti-patterns
4. produce structured lint report

## Security
- Filesystem: read-only
- Network: none
