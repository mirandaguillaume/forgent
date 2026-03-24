---
name: code-review-reporter
description: report-generation-based skill consuming security-findings, quality-assessment, project-context to produce code-review-report
---

# Code Review Reporter

## Guardrails
- Maximum report generation time of 5 minutes
- Use specified terminal-optimized output format
- Maintain mentoring tone throughout report
- Provide concrete code examples for all suggestions

## Context
Consumes: security-findings, quality-assessment, project-context
Produces: code-review-report
Memory: short-term

## Strategy
Approach: report-generation
Tools: task

### Steps
1. Compile security and quality findings into unified report
2. Prioritize issues by severity and impact
3. Format findings with specific code examples and fixes
4. Structure report with critical issues, warnings, and suggestions
5. Ensure terminal-friendly output format
6. Provide educational context for each recommendation

## Security
- Filesystem: none
- Network: none
