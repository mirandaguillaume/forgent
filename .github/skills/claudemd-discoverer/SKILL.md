---
name: claudemd-discoverer
description: file-discovery-based skill consuming pr_diff, file_tree to produce claudemd_files
---

# Claudemd Discoverer

## Guardrails
- timeout: 1min
- max_depth: 5

## When to Use

Use for:
- when code review needs project-specific rules

**Don't use for:**
- when reviewing without CLAUDE.md compliance

## Context
Consumes: pr_diff, file_tree
Produces: claudemd_files
Memory: short-term

## Strategy
Approach: file-discovery
Tools: bash, read_file, glob

### Steps
1. find root CLAUDE.md if it exists
2. identify directories modified by the PR diff
3. find CLAUDE.md files in those directories
4. return list of file paths with contents

## Security
- Filesystem: read-only
- Network: none
