---
name: tdd-runner
description: test-first-based skill consuming file_tree, source_code to produce test_results, coverage_report
---

# Tdd Runner

## Guardrails
- timeout: 10min
- max_retries: 2
- fail_fast_on_syntax_error

## Context
Consumes: file_tree, source_code
Produces: test_results, coverage_report
Memory: short-term

## Strategy
Approach: test-first
Tools: bash, read_file, grep

### Steps
1. detect test framework from package.json
2. run test suite with coverage
3. parse test output for failures
4. summarize results with pass/fail counts

## Security
- Filesystem: full
- Network: none
