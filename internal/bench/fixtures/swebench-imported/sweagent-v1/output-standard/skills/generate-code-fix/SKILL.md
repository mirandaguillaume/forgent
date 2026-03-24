---
name: generate-code-fix
description: Generate minimal code changes that resolve the identified issue-based skill consuming root_cause, issue_description to produce code_patch
---

# Generate Code Fix

## Guardrails
- Only modify non-test files
- Do not change existing function or class names
- Do not suggest installing new packages
- Only use standard libraries
- Ensure changes are minimal and targeted
- Timeout after 180 seconds of fix generation

## Context
Consumes: root_cause, issue_description
Produces: code_patch
Memory: short-term

## Output Format

### Output: Code_patch

Output your fix as a unified diff patch between `<patch>` and `</patch>` tags.

## Strategy
Approach: Generate minimal code changes that resolve the identified issue
Tools: read_file, edit_file

### Steps
1. Design minimal changes to resolve the root cause
2. Consider edge cases and ensure fix handles them
3. Verify changes don't break existing function/class names
4. Generate unified diff patch for the changes

## Security
- Filesystem: read-write
- Network: none
