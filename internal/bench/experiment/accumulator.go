package experiment

import (
	"fmt"
	"os"
)

// promptAccumulator is an append-only string accumulator with a hard byte cap
// and oldest-first eviction for tool-result entries.
//
// Design contract:
//   - base is set once via AppendBase and is NEVER evicted — it contains the
//     step instructions that must always be present in the LLM context.
//   - entries are tool-result blocks appended chronologically; when adding a new
//     entry would exceed cap, the oldest entries are evicted first until it fits.
//   - If a single new entry is too large to fit alongside base (even after all
//     prior entries are evicted), the entry is truncated from the front so that
//     base + truncated-entry ≤ cap. The beginning of the entry is dropped because
//     the end of a tool result (typically the summary/conclusion) is more valuable
//     than its start (typically the raw dump header).
//   - String() concatenates base + entries in declaration order (oldest first).
type promptAccumulator struct {
	base    string
	entries []string
	cap     int
}

// newPromptAccumulator creates an accumulator with the given byte cap.
func newPromptAccumulator(capBytes int) *promptAccumulator {
	return &promptAccumulator{cap: capBytes}
}

// AppendBase sets the base prompt. Should be called once before any tool results.
func (a *promptAccumulator) AppendBase(s string) {
	a.base = s
}

// AppendToolResults appends a tool-result block, evicting oldest entries if
// necessary to stay within the byte cap.
func (a *promptAccumulator) AppendToolResults(s string) {
	// Evict oldest entries until s fits (or entries is empty).
	for len(a.entries) > 0 && a.totalLen()+len(s) > a.cap {
		a.entries = a.entries[1:]
	}

	// If s still doesn't fit alongside base alone, truncate s from the front.
	if a.totalLen()+len(s) > a.cap {
		budget := a.cap - len(a.base)
		if budget <= 0 {
			// Base alone exceeds cap — silently drop the entry.
			fmt.Fprintf(os.Stderr, "warn: promptAccumulator: base (%d B) exceeds cap (%d B); tool result dropped\n", len(a.base), a.cap)
			return
		}
		if len(s) > budget {
			// Warn and truncate from the front.
			fmt.Fprintf(os.Stderr, "warn: promptAccumulator: tool result (%d B) truncated to %d B to fit cap\n", len(s), budget)
			s = s[len(s)-budget:]
		}
	}

	a.entries = append(a.entries, s)
}

// String returns the concatenation of base and all surviving entries.
func (a *promptAccumulator) String() string {
	if len(a.entries) == 0 {
		return a.base
	}
	total := len(a.base)
	for _, e := range a.entries {
		total += len(e)
	}
	out := make([]byte, 0, total)
	out = append(out, a.base...)
	for _, e := range a.entries {
		out = append(out, e...)
	}
	return string(out)
}

// totalLen returns the current byte length of base + all entries.
func (a *promptAccumulator) totalLen() int {
	n := len(a.base)
	for _, e := range a.entries {
		n += len(e)
	}
	return n
}
