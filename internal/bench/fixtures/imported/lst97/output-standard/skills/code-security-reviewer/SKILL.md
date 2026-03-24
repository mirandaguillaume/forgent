---
name: code-security-reviewer
description: security-scanning-based skill consuming code-files, git-diff, project-context to produce security-findings
---

# Code Security Reviewer

## Guardrails
- Maximum review time of 15 minutes per session
- Focus only on security-related issues
- Never expose or log sensitive information found in code
- Categorize findings by CVSS severity levels

## Context
Consumes: code-files, git-diff, project-context
Produces: security-findings
Memory: short-term

## Strategy
Approach: security-scanning
Tools: read_file, grep, search, web_fetch, web_search

### Steps
1. Review code for SQL injection, XSS, and other injection vulnerabilities
2. Scan for hardcoded secrets, API keys, and passwords
3. Validate input sanitization and validation patterns
4. Check authentication and authorization implementations
5. Assess dependency security and vulnerable library usage
6. Generate security findings report with severity levels

## Security
- Filesystem: read-only
- Network: allowlist
