---
name: complexity-assessment
description: complexity-analysis-based skill consuming code_files, project_requirements to produce complexity_assessment
---

# Complexity Assessment

## Guardrails
- Timeout after 8 minutes to prevent excessive analysis
- Consider project scale (MVP vs enterprise) in assessment
- Use standardized complexity levels: Low|Medium|High

## Context
Consumes: code_files, project_requirements
Produces: complexity_assessment
Memory: short-term

## Strategy
Approach: complexity-analysis
Tools: read_file, grep, search

### Steps
1. Scan codebase for complexity indicators and patterns
2. Evaluate complexity against actual project requirements
3. Generate complexity score with justification
4. Identify over-engineering patterns and unnecessary abstractions

## Security
- Filesystem: read-only
- Network: none
