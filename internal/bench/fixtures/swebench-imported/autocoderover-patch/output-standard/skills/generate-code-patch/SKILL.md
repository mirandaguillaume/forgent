---
name: generate-code-patch
description: minimal code changes to resolve identified issue-based skill consuming bug_analysis, code_context to produce unified_diff_patch
---

# Generate Code Patch

## Guardrails
- timeout: 600s
- minimal_changes_only: true
- preserve_existing_functionality: true
- no_test_file_modifications: true
- proper_python_indentation: required

## Context
Consumes: bug_analysis, code_context
Produces: unified_diff_patch
Memory: short-term

## Output Format

### Output: Unified_diff_patch

```
--- a/full/path/to/file.py
+++ b/full/path/to/file.py
@@ -start,count +start,count @@
 context line
-line to remove
+line to add
 context line
```

The patch should be enclosed in `<patch>` and `</patch>` tags and follow unified diff format with proper Python indentation.

## Strategy
Approach: minimal code changes to resolve identified issue
Tools: read_file, edit_file, grep

### Steps
1. identify exact locations requiring changes
2. implement minimal fixes preserving functionality
3. add necessary imports if required
4. generate unified diff format

## Security
- Filesystem: read-only
- Network: none
