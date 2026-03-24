---
name: code-simplification-recommendations
description: Generate specific actionable recommendations for code simplification-based skill consuming complexity_assessment, code_files to produce simplification_recommendations
---

# Code Simplification Recommendations

## Guardrails
- Timeout after 10 minutes for recommendation generation
- Focus on maximum 5 issues to avoid overwhelming output
- Always provide concrete, actionable recommendations
- Use standardized severity levels: Critical|High|Medium|Low
- Include file_path:line_number format for references

## Context
Consumes: complexity_assessment, code_files
Produces: simplification_recommendations
Memory: short-term

## Strategy
Approach: Generate specific actionable recommendations for code simplification
Tools: read_file, grep

### Steps
1. Review complexity assessment findings
2. Identify top 3-5 issues impacting developer experience
3. Create concrete simplification suggestions with examples
4. Prioritize recommendations by impact on maintainability
5. Include agent collaboration suggestions when needed

## Security
- Filesystem: read-only
- Network: none
