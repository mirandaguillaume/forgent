---
name: minimal-fix-generation
description: Generate minimal code fixes based on bug analysis to resolve issues without breaking existing functionality-based skill consuming bug_location_analysis, codebase_files to produce unified_diff_patch
---

# Minimal Fix Generation

## Guardrails
- Only modify files necessary to fix the reported bug
- Preserve all existing program functionality
- Do not refactor unrelated code
- Do not modify test files or write new tests
- Use proper Python indentation and syntax
- Make minimal changes to resolve the issue
- Import necessary libraries only if needed for the fix
- timeout: 300

## Context
Consumes: bug_location_analysis, codebase_files
Produces: unified_diff_patch
Memory: short-term

## Output Format

### Output: Unified_diff_patch

Generate a unified diff patch that fixes the issue. The patch should:
- Only modify the files necessary to fix the bug
- Preserve existing program functionality
- Use proper Python indentation
- Not modify test files or write tests

Output the patch between `<patch>` and `</patch>` tags as a unified diff format:

```
--- a/path/to/file.py
+++ b/path/to/file.py
@@ -line,count +line,count @@
 context line
-removed line
+added line
 context line
```

## Strategy
Approach: Generate minimal code fixes based on bug analysis to resolve issues without breaking existing functionality
Tools: read_file, edit_file

### Steps
1. Review bug location analysis and root cause findings
2. Generate minimal code changes that fix the identified issue
3. Ensure changes preserve existing program functionality
4. Output unified diff patch in proper format

## Security
- Filesystem: read-write
- Network: none
