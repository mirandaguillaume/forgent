---
name: code-quality-analyzer
description: comprehensive code quality assessment-based skill consuming code-changes, coding-standards to produce quality-metrics
---

# Code Quality Analyzer

## Guardrails
- timeout: 20 minutes for quality analysis
- max_cyclomatic_complexity: 10
- minimum_coverage_threshold: 80%

## Context
Consumes: code-changes, coding-standards
Produces: quality-metrics
Memory: short-term

## Strategy
Approach: comprehensive code quality assessment
Tools: read_file, grep, search, bash

### Steps
1. evaluate logic correctness
2. analyze naming conventions
3. check code organization
4. measure function complexity
5. detect code duplication
6. assess readability
7. validate best practices compliance

## Security
- Filesystem: read-only
- Network: none
