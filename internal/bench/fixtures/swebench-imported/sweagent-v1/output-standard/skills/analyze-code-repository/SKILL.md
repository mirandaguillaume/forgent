---
name: analyze-code-repository
description: Systematically analyze repository structure to locate code relevant to the issue-based skill consuming repository, issue_description to produce code_analysis
---

# Analyze Code Repository

## Guardrails
- Timeout after 300 seconds of file analysis
- Limit file reads to 100 files per analysis
- Only analyze files with common programming extensions (.py, .js, .java, etc.)

## Context
Consumes: repository, issue_description
Produces: code_analysis
Memory: short-term

## Strategy
Approach: Systematically analyze repository structure to locate code relevant to the issue
Tools: read_file, grep, search

### Steps
1. Find and read code files relevant to the problem description
2. Analyze the repository structure and dependencies
3. Document relevant code components and their relationships

## Security
- Filesystem: read-only
- Network: none
