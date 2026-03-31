package rewrite

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/fatih/color"
)

// GenerateDiff produces a unified-style diff between original and rewritten.
// If tty is true, output is colorized (green for additions, red for removals).
func GenerateDiff(original, rewritten []byte, tty bool) string {
	origLines := strings.Split(string(normalizeLineEndings(original)), "\n")
	rewriteLines := strings.Split(string(normalizeLineEndings(rewritten)), "\n")

	// Simple line-by-line diff with context
	const contextLines = 3
	var hunks []hunk
	hunks = buildHunks(origLines, rewriteLines, contextLines)

	if len(hunks) == 0 {
		return ""
	}

	var b strings.Builder
	for _, h := range hunks {
		for _, line := range h.lines {
			if tty {
				b.WriteString(colorizeDiffLine(line))
			} else {
				b.WriteString(line)
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

type hunk struct {
	lines []string
}

func buildHunks(origLines, rewriteLines []string, ctx int) []hunk {
	maxLen := len(origLines)
	if len(rewriteLines) > maxLen {
		maxLen = len(rewriteLines)
	}

	// Find changed line indices
	var changed []int
	for i := 0; i < maxLen; i++ {
		origLine := ""
		if i < len(origLines) {
			origLine = origLines[i]
		}
		rewriteLine := ""
		if i < len(rewriteLines) {
			rewriteLine = rewriteLines[i]
		}
		if origLine != rewriteLine {
			changed = append(changed, i)
		}
	}

	if len(changed) == 0 {
		return nil
	}

	// Group changes into hunks with context
	var hunks []hunk
	var current hunk
	lastEmitted := -1

	for _, ci := range changed {
		// Context before
		start := ci - ctx
		if start < 0 {
			start = 0
		}
		if start <= lastEmitted {
			start = lastEmitted + 1
		}

		// Emit separator if gap
		if lastEmitted >= 0 && start > lastEmitted+1 {
			if len(current.lines) > 0 {
				hunks = append(hunks, current)
				current = hunk{}
			}
			current.lines = append(current.lines, fmt.Sprintf("@@ -%d +%d @@", ci+1, ci+1))
		} else if lastEmitted < 0 {
			current.lines = append(current.lines, fmt.Sprintf("@@ -%d +%d @@", ci+1, ci+1))
		}

		// Context lines before the change
		for i := start; i < ci; i++ {
			if i < len(origLines) {
				current.lines = append(current.lines, " "+origLines[i])
			}
		}

		// The change itself
		if ci < len(origLines) {
			current.lines = append(current.lines, "-"+origLines[ci])
		}
		if ci < len(rewriteLines) {
			current.lines = append(current.lines, "+"+rewriteLines[ci])
		}

		// Context after
		end := ci + ctx
		if end >= maxLen {
			end = maxLen - 1
		}
		for i := ci + 1; i <= end; i++ {
			// Only emit if not also a changed line
			isChanged := false
			for _, cj := range changed {
				if cj == i {
					isChanged = true
					break
				}
			}
			if isChanged {
				break
			}
			if i < len(origLines) {
				current.lines = append(current.lines, " "+origLines[i])
				lastEmitted = i
				continue
			}
		}

		lastEmitted = ci
		if end > lastEmitted {
			lastEmitted = end
		}
	}

	if len(current.lines) > 0 {
		hunks = append(hunks, current)
	}

	return hunks
}

func normalizeLineEndings(content []byte) []byte {
	return bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n"))
}

func colorizeDiffLine(line string) string {
	if strings.HasPrefix(line, "+") {
		return color.GreenString(line)
	}
	if strings.HasPrefix(line, "-") {
		return color.RedString(line)
	}
	if strings.HasPrefix(line, "@@") {
		return color.CyanString(line)
	}
	return line
}
