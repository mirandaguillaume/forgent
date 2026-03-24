---
name: type-checker
description: static-analysis-based skill consuming file_tree, source_code to produce type_errors
---

# Type Checker

## Guardrails
- timeout: 5min
- max_file_size: 500KB

## Context
Consumes: file_tree, source_code
Produces: type_errors
Memory: short-term

## Strategy
Approach: static-analysis
Tools: bash, read_file

### Steps
1. run TypeScript compiler in noEmit mode
2. collect type errors with file locations

## Security
- Filesystem: read-only
- Network: none
