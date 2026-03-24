---
name: tdd-runner
description: test-first-based skill consuming file_tree, source_code to produce test_results
---

# Tdd Runner

## Guardrails
- timeout: 10min
- max_retries: 2
- fail_fast_on_syntax_error

## Steps
1. detect test framework from package.json
2. run test suite
3. parse test output for failures
4. summarize results with pass/fail counts

