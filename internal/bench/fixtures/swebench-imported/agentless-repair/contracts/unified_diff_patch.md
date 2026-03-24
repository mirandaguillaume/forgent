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
