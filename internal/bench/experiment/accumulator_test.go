package experiment

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// T7.1: base + entries within cap → String returns all of them.
func TestPromptAccumulator_PreservesContextAcrossIterations(t *testing.T) {
	acc := newPromptAccumulator(1024)
	acc.AppendBase("BASE")
	acc.AppendToolResults(" R1")
	acc.AppendToolResults(" R2")
	acc.AppendToolResults(" R3")

	got := acc.String()
	assert.Equal(t, "BASE R1 R2 R3", got)
}

// T7.2: entries that exceed cap are evicted oldest-first; base is preserved.
func TestPromptAccumulator_HardCapTrimsOldestFirst(t *testing.T) {
	// cap = 20, base = "BASE" (4 B). Entries take the remaining 16 B.
	acc := newPromptAccumulator(20)
	acc.AppendBase("BASE")

	acc.AppendToolResults(" AAAA") // total 9 B — fits
	acc.AppendToolResults(" BBBB") // total 14 B — fits
	acc.AppendToolResults(" CCCC") // total 19 B — fits
	// Adding " DDDD" (5 B) → total 24 B > 20 → evict " AAAA" → 19 B → fits
	acc.AppendToolResults(" DDDD")

	got := acc.String()
	assert.NotContains(t, got, "AAAA", "oldest entry should be evicted")
	assert.Contains(t, got, "BBBB")
	assert.Contains(t, got, "CCCC")
	assert.Contains(t, got, "DDDD")
	assert.True(t, strings.HasPrefix(got, "BASE"), "base must be preserved")
}

// T7.3: base wins when a single entry + base exceeds cap; entry is truncated.
func TestPromptAccumulator_BasePreservedEvenWhenSingleEntryFitsAlone(t *testing.T) {
	// cap = 10. base = "BASE" (4 B). budget = 6 B.
	// One entry "LONGENTRY" (9 B) fits alone but not with base.
	acc := newPromptAccumulator(10)
	acc.AppendBase("BASE")
	acc.AppendToolResults("LONGENTRY") // 9 B alone OK, but 4+9=13 > 10 → truncate to 6 B

	got := acc.String()
	assert.True(t, strings.HasPrefix(got, "BASE"), "base must be preserved")
	assert.LessOrEqual(t, len(got), 10, "result must be within cap")
	// The entry is truncated from the front: "LONGENTRY"[9-6:] = "ENTRY"... but wait
	// budget=6, len("LONGENTRY")=9, so s[9-6:] = s[3:] = "GENTRY"
	assert.Contains(t, got, "GENTRY", "truncated entry (end) should survive")
}

// T7.5: base alone exceeds cap → tool result silently dropped, String returns base only.
func TestPromptAccumulator_BaseExceedsCap_ToolResultDropped(t *testing.T) {
	// cap = 4, base = "BASE" (4 B). budget = 0 → any tool result is dropped.
	acc := newPromptAccumulator(4)
	acc.AppendBase("BASE")
	acc.AppendToolResults("TOOL") // budget = cap - len(base) = 0 → silent drop

	got := acc.String()
	assert.Equal(t, "BASE", got, "base is preserved; tool result is dropped when budget=0")
	assert.NotContains(t, got, "TOOL")
}

// T7.4: guardrail retry does NOT reset the accumulator; tool results from
// attempt 1 are still present in the prompt sent during attempt 2.
func TestStepLoop_GuardrailRetry_DoesNotResetAccumulator(t *testing.T) {
	capturedPrompts := make([]string, 0)
	callN := 0
	p := &funcProvider{fn: func(prompt string) (string, error) {
		callN++
		capturedPrompts = append(capturedPrompts, prompt)
		switch callN {
		case 1:
			// First LLM call: emit a tool request so tool results accumulate.
			return `<tool name="file_read" path="task_context.md"/>`, nil
		case 2:
			// After tool execution in the inner loop, violate guardrail.
			return "BANNED output", nil
		case 3:
			// Guardrail retry: this prompt must contain the tool result from call 1.
			return "clean output without the banned word", nil
		default:
			return "clean output", nil
		}
	}}

	task := taskWithStep(StepDef{
		ID:         "s1",
		Tools:      []string{"file_read"},
		Guardrails: []string{"!BANNED"},
		Produces: []SlotSpec{{Key: "out"}},
		Loop:     &LoopPolicy{MaxIterations: 3},
	})
	// buildTaskDAG always writes task_context.md from taskContextContent(task),
	// which embeds task.Input in the ## Original Input section. Set it here so
	// the marker survives regardless of any workspace overwrite order.
	task.Input = "context content XYZ"

	result := runRuntimeTask(task, p, nil, nil, "runtime")
	assert.NoError(t, result.Error)

	// At guardrail retry (callN=3), the prompt must still include the tool
	// result from the file_read executed during attempt 1.
	if len(capturedPrompts) >= 3 {
		assert.Contains(t, capturedPrompts[2], "context content XYZ",
			"guardrail retry prompt should carry tool results from prior attempt")
	}
}
