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
