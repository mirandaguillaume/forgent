package bench

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultClaudeRunConfig(t *testing.T) {
	cfg := DefaultClaudeRunConfig()
	assert.NotEmpty(t, cfg.ClaudeBin)
	assert.NotEmpty(t, cfg.Model)
	assert.Greater(t, cfg.MaxBudgetUSD, 0.0)
	assert.Greater(t, cfg.Workers, 0)
}

func TestCheckClaudeInstalled(t *testing.T) {
	// Should succeed with the real claude CLI
	cfg := DefaultClaudeRunConfig()
	err := CheckClaudeInstalled(cfg.ClaudeBin)
	if err != nil {
		t.Skip("claude CLI not available")
	}
	assert.NoError(t, err)
}

func TestCheckClaudeInstalled_Missing(t *testing.T) {
	err := CheckClaudeInstalled("/nonexistent/claude")
	assert.Error(t, err)
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "hello", truncate("hello", 10))
	assert.Equal(t, "hel...", truncate("hello world", 3))
	assert.Equal(t, "", truncate("", 5))
}

func TestExtractPatchFiles(t *testing.T) {
	patch := `diff --git a/foo/bar.py b/foo/bar.py
--- a/foo/bar.py
+++ b/foo/bar.py
@@ -1 +1 @@
-old
+new
diff --git a/baz/qux.py b/baz/qux.py
--- a/baz/qux.py
+++ b/baz/qux.py
@@ -1 +1 @@
-old
+new`

	files := extractPatchFiles(patch)
	assert.Equal(t, []string{"baz/qux.py", "foo/bar.py"}, files)
}

func TestExtractPatchFiles_Empty(t *testing.T) {
	assert.Empty(t, extractPatchFiles(""))
	assert.Empty(t, extractPatchFiles("no diff here"))
}

func TestComparePatchToGold_ExactMatch(t *testing.T) {
	patch := "diff --git a/foo.py b/foo.py\n-old\n+new\n"
	overlap, filesMatch, exact := comparePatchToGold(patch, patch)
	assert.Equal(t, 1.0, overlap)
	assert.True(t, filesMatch)
	assert.True(t, exact)
}

func TestComparePatchToGold_SameFiles_DifferentContent(t *testing.T) {
	gold := "diff --git a/foo.py b/foo.py\n-old\n+correct\n"
	gen := "diff --git a/foo.py b/foo.py\n-old\n+wrong\n"
	overlap, filesMatch, exact := comparePatchToGold(gen, gold)
	assert.Equal(t, 1.0, overlap)
	assert.True(t, filesMatch)
	assert.False(t, exact)
}

func TestComparePatchToGold_PartialOverlap(t *testing.T) {
	gold := "diff --git a/a.py b/a.py\n-1\n+2\ndiff --git a/b.py b/b.py\n-3\n+4\n"
	gen := "diff --git a/a.py b/a.py\n-1\n+2\ndiff --git a/c.py b/c.py\n-5\n+6\n"
	overlap, filesMatch, exact := comparePatchToGold(gen, gold)
	assert.Equal(t, 0.5, overlap) // 1 of 2 gold files
	assert.False(t, filesMatch)
	assert.False(t, exact)
}

func TestComparePatchToGold_NoOverlap(t *testing.T) {
	gold := "diff --git a/a.py b/a.py\n-1\n+2\n"
	gen := "diff --git a/z.py b/z.py\n-1\n+2\n"
	overlap, filesMatch, exact := comparePatchToGold(gen, gold)
	assert.Equal(t, 0.0, overlap)
	assert.False(t, filesMatch)
	assert.False(t, exact)
}

func TestComparePatchToGold_EmptyGold(t *testing.T) {
	overlap, filesMatch, exact := comparePatchToGold("diff --git a/a.py b/a.py\n", "")
	assert.Equal(t, 0.0, overlap)
	assert.False(t, filesMatch)
	assert.False(t, exact)
}

func TestNormalizePatch(t *testing.T) {
	assert.Equal(t, "-old\n+new", normalizePatch("-old  \n+new\n  \n"))
	assert.Equal(t, "", normalizePatch("  \n  \n"))
}
