---
name: code-quality-assessment
description: Static analysis for code maintainability using complexity metrics-based skill consuming source_code, test_files to produce quality_report
---

# Code Quality Assessment

## Guardrails
- timeout: 240s
- max_complexity_threshold: 15
- min_test_coverage: 80

## Context
Consumes: source_code, test_files
Produces: quality_report
Memory: short-term

## Strategy
Approach: Static analysis for code maintainability using complexity metrics
Tools: read_file, grep, search, bash

### Steps
1. Analyze code complexity and maintainability
2. Check coding standards compliance
3. Detect code smells and anti-patterns
4. Review test coverage and quality
5. Assess technical debt
6. Validate documentation completeness

## Security
- Filesystem: read-only
- Network: none
