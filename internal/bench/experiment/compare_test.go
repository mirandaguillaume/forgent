package experiment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAlignResultsByTaskID_DropsTasksOnlyInOneCondition — story 10-0 Task 5.5.
// SkipMap can cause one condition to skip runs the other did not. Pairing
// by index would alias unrelated tasks; alignment must drop one-sided tasks.
func TestAlignResultsByTaskID_DropsTasksOnlyInOneCondition(t *testing.T) {
	text := []TaskResult{
		{TaskID: "a", Completed: true},
		{TaskID: "b", Completed: false},
		{TaskID: "c", Completed: true},
	}
	runtime := []TaskResult{
		{TaskID: "a", Completed: true},
		{TaskID: "c", Completed: false},
	}
	textAligned, runtimeAligned := alignResultsByTaskID(text, runtime)
	require.Len(t, textAligned, 2)
	require.Len(t, runtimeAligned, 2)
	assert.Equal(t, "a", textAligned[0].TaskID)
	assert.Equal(t, "a", runtimeAligned[0].TaskID)
	assert.Equal(t, "c", textAligned[1].TaskID)
	assert.Equal(t, "c", runtimeAligned[1].TaskID)
}

// TestAlignResultsByTaskID_PairsMultipleRunsInOrder — RunsPerTask > 1
// produces multiple TaskResult entries with the same TaskID. They must pair
// run-by-run in slice order, not collapse to first-only.
func TestAlignResultsByTaskID_PairsMultipleRunsInOrder(t *testing.T) {
	text := []TaskResult{
		{TaskID: "a", TokensUsed: 10},
		{TaskID: "a", TokensUsed: 20},
	}
	runtime := []TaskResult{
		{TaskID: "a", TokensUsed: 100},
		{TaskID: "a", TokensUsed: 200},
	}
	textAligned, runtimeAligned := alignResultsByTaskID(text, runtime)
	require.Len(t, textAligned, 2)
	require.Len(t, runtimeAligned, 2)
	assert.Equal(t, 10, textAligned[0].TokensUsed)
	assert.Equal(t, 100, runtimeAligned[0].TokensUsed)
	assert.Equal(t, 20, textAligned[1].TokensUsed)
	assert.Equal(t, 200, runtimeAligned[1].TokensUsed)
}

// TestCompare_AlignsByTaskID_NotIndex — pair-sensitive metric (DetectionRate)
// with text and runtime in different order proves alignment is doing real
// work. Aggregate metrics like CompletionDelta are invariant to pairing
// (sums are pairing-independent), so they make a weak test of the fix;
// DetectionRate pairs by index and would mis-attribute caught failures
// without correct alignment.
//
// Setup:
//   - text:    [a fail, b pass, c fail]
//   - runtime: [c pass, a pass, b fail]
//
// Aligned (by TaskID):
//   - a: text fail → runtime pass  → DETECTED
//   - b: text pass → runtime fail  → no failure to detect
//   - c: text fail → runtime pass  → DETECTED
//   → CompletionFailures = 2, CompletionCaught = 2, rate = 1.0
//
// Without alignment (positional):
//   - a (text fail) paired with c (runtime pass) → 1 caught
//   - b (text pass) paired with a (runtime pass) → no failure
//   - c (text fail) paired with b (runtime fail) → 0 caught
//   → CompletionFailures = 2, CompletionCaught = 1, rate = 0.5
//
// The aligned answer is the correct one; the un-aligned answer hides a
// real detection. Asserting rate = 1.0 fails if alignment regresses to
// positional pairing.
func TestCompare_AlignsByTaskID_NotIndex(t *testing.T) {
	text := []TaskResult{
		{TaskID: "a", Completed: false, OrderingScore: 1.0, GuardrailScore: 1.0},
		{TaskID: "b", Completed: true, OrderingScore: 1.0, GuardrailScore: 1.0},
		{TaskID: "c", Completed: false, OrderingScore: 1.0, GuardrailScore: 1.0},
	}
	runtime := []TaskResult{
		{TaskID: "c", Completed: true, OrderingScore: 1.0, GuardrailScore: 1.0},
		{TaskID: "a", Completed: true, OrderingScore: 1.0, GuardrailScore: 1.0},
		{TaskID: "b", Completed: false, OrderingScore: 1.0, GuardrailScore: 1.0},
	}
	cr := Compare(text, runtime)
	// Pin the cardinality first: a regression that returns only the first
	// matched pair would still produce CompletionRate=1.0 (1/1) and pass a
	// rate-only assertion, so cr.Tasks and CompletionFailures must be checked
	// to lock down the actual sample size.
	assert.Equal(t, 3, cr.Tasks, "all 3 tasks present in both conditions must survive alignment")
	require.Equal(t, 2, cr.DetectionRate.CompletionFailures, "two text completion failures (a, c)")
	assert.Equal(t, 2, cr.DetectionRate.CompletionCaught,
		"alignment pairs text-a<->runtime-a and text-c<->runtime-c, detecting both")
	assert.Equal(t, 1.0, cr.DetectionRate.CompletionRate,
		"all text completion failures detected when alignment is correct; positional pairing would yield 0.5")
}
