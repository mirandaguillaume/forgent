---
name: coverage-reporter
description: test-first-based skill consuming file_tree, source_code to produce coverage_report
---

# Coverage Reporter

## Guardrails
- timeout: 10min

## Context
Consumes: file_tree, source_code
Produces: coverage_report
Memory: short-term

## Strategy
Approach: test-first
Tools: bash, read_file

### Steps
1. detect test framework from package.json
2. run test suite with coverage flag
3. parse coverage output into structured report

## Security
- Filesystem: full
- Network: none
