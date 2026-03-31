package rewrite

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateDiff_NoChanges(t *testing.T) {
	content := []byte("line1\nline2\nline3\n")
	diff := GenerateDiff(content, content, false)
	assert.Empty(t, diff)
}

func TestGenerateDiff_SimpleChange(t *testing.T) {
	orig := []byte("line1\nline2\nline3\nline4\nline5\n")
	rewritten := []byte("line1\nline2 changed\nline3\nline4\nline5\n")

	diff := GenerateDiff(orig, rewritten, false)
	assert.Contains(t, diff, "-line2")
	assert.Contains(t, diff, "+line2 changed")
	assert.Contains(t, diff, "@@")
}

func TestGenerateDiff_Addition(t *testing.T) {
	orig := []byte("line1\nline2\n")
	rewritten := []byte("line1\nline2\nline3 new\n")

	diff := GenerateDiff(orig, rewritten, false)
	assert.Contains(t, diff, "+line3 new")
}

func TestGenerateDiff_Removal(t *testing.T) {
	orig := []byte("line1\nline2\nline3\n")
	rewritten := []byte("line1\nline2\n")

	diff := GenerateDiff(orig, rewritten, false)
	assert.Contains(t, diff, "-line3")
}

func TestGenerateDiff_CRLF(t *testing.T) {
	orig := []byte("line1\r\nline2\r\n")
	rewritten := []byte("line1\nline2\n")

	// After normalization, these should be identical
	diff := GenerateDiff(orig, rewritten, false)
	assert.Empty(t, diff)
}

func TestGenerateDiff_TTYColorized(t *testing.T) {
	orig := []byte("line1\nline2\nline3\n")
	rewritten := []byte("line1\nline2 changed\nline3\n")

	diff := GenerateDiff(orig, rewritten, true)
	// Colorized output should contain ANSI codes
	assert.NotEmpty(t, diff)
}
