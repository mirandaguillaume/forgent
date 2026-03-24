---
name: unified-diff-patch-generator
description: Generate minimal unified diff patches for Python bug fixes-based skill consuming issue_analysis, python_codebase to produce unified_diff_patch
---

# Unified Diff Patch Generator

## Guardrails
- Do not change existing function or class names
- Only use standard Python libraries
- Make minimal changes focused only on the bug fix
- Preserve existing code style and conventions
- Maintain proper Python indentation
- Maximum 15 minutes per patch generation
- Changes must be in unified diff format

## Context
Consumes: issue_analysis, python_codebase
Produces: unified_diff_patch
Memory: short-term

## Output Format

### Output: Unified_diff_patch

```
--- a/path/to/file.py
+++ b/path/to/file.py
@@ -line,count +line,count @@
 context
-old line
+new line
 context
```

## Strategy
Approach: Generate minimal unified diff patches for Python bug fixes
Tools: read_file, edit_file

### Steps
1. Review the issue analysis to understand required changes
2. Make minimal code changes to fix the identified bug
3. Generate unified diff patch with proper context

## Security
- Filesystem: read-only
- Network: none
