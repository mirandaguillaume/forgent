package experiment

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sampleStepResults() []StepResult {
	return []StepResult{
		{StepID: "step1", Completed: true, Output: "output1", KeywordsHit: 2, KeywordsTotal: 3, TokensUsed: 50},
		{StepID: "step2", Completed: true, Output: "output2", KeywordsHit: 1, KeywordsTotal: 1, TokensUsed: 30},
	}
}

// TestTaskResultJSON_RoundTrip verifies that a TaskResult with all fields
// survives a JSON round-trip unchanged.
func TestTaskResultJSON_RoundTrip(t *testing.T) {
	original := TaskResult{
		TaskID:         "task-1",
		Condition:      "text",
		Completed:      true,
		StepResults:    sampleStepResults(),
		OrderingScore:  0.85,
		GuardrailScore: 0.90,
		TokensUsed:     1234,
		WallClockMs:    567,
		LLMCalls:       3,
		Error:          nil,
	}

	j := ToResultJSON(original)
	back := j.ToTaskResult()

	assert.Equal(t, original.TaskID, back.TaskID)
	assert.Equal(t, original.Condition, back.Condition)
	assert.Equal(t, original.Completed, back.Completed)
	assert.Equal(t, original.StepResults, back.StepResults)
	assert.Equal(t, original.OrderingScore, back.OrderingScore)
	assert.Equal(t, original.GuardrailScore, back.GuardrailScore)
	assert.Equal(t, original.TokensUsed, back.TokensUsed)
	assert.Equal(t, original.WallClockMs, back.WallClockMs)
	assert.Equal(t, original.LLMCalls, back.LLMCalls)
	assert.Nil(t, back.Error)
}

// TestTaskResultJSON_WithError verifies that the error string survives the round-trip.
func TestTaskResultJSON_WithError(t *testing.T) {
	original := TaskResult{
		TaskID:    "task-err",
		Condition: "runtime",
		Error:     assert.AnError,
	}

	j := ToResultJSON(original)
	assert.Equal(t, assert.AnError.Error(), j.Error)

	back := j.ToTaskResult()
	require.NotNil(t, back.Error)
	assert.Equal(t, assert.AnError.Error(), back.Error.Error())
}

// TestCheckpointWriter_AppendAndLoad writes 4 entries and reads them back.
func TestCheckpointWriter_AppendAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "checkpoint.jsonl")

	w, err := NewCheckpointWriter(path)
	require.NoError(t, err)

	entries := []CheckpointEntry{
		{TaskID: "t1", Condition: "text", Run: 1, Result: ToResultJSON(TaskResult{TaskID: "t1", Condition: "text", Completed: true, TokensUsed: 100})},
		{TaskID: "t1", Condition: "runtime", Run: 1, Result: ToResultJSON(TaskResult{TaskID: "t1", Condition: "runtime", Completed: true, TokensUsed: 80})},
		{TaskID: "t2", Condition: "text", Run: 1, Result: ToResultJSON(TaskResult{TaskID: "t2", Condition: "text", Completed: false, TokensUsed: 200})},
		{TaskID: "t2", Condition: "runtime", Run: 1, Result: ToResultJSON(TaskResult{TaskID: "t2", Condition: "runtime", Completed: true, TokensUsed: 160})},
	}

	for _, e := range entries {
		require.NoError(t, w.Append(e))
	}
	require.NoError(t, w.Close())

	loaded, err := LoadCheckpoint(path)
	require.NoError(t, err)
	require.Len(t, loaded, 4)

	assert.Equal(t, "t1", loaded[0].TaskID)
	assert.Equal(t, "text", loaded[0].Condition)
	assert.Equal(t, 1, loaded[0].Run)
	assert.Equal(t, true, loaded[0].Result.Completed)
	assert.Equal(t, 100, loaded[0].Result.TokensUsed)

	assert.Equal(t, "t2", loaded[2].TaskID)
	assert.Equal(t, "text", loaded[2].Condition)
	assert.Equal(t, false, loaded[2].Result.Completed)
}

// TestBuildSkipMap verifies run-count grouping by condition.
func TestBuildSkipMap(t *testing.T) {
	entries := []CheckpointEntry{
		{TaskID: "t1", Condition: "text", Run: 1},
		{TaskID: "t1", Condition: "text", Run: 2},
		{TaskID: "t2", Condition: "text", Run: 1},
		{TaskID: "t1", Condition: "runtime", Run: 1},
	}

	textMap := BuildSkipMap(entries, "text")
	assert.Equal(t, 2, textMap["t1"], "t1 should have 2 text runs")
	assert.Equal(t, 1, textMap["t2"], "t2 should have 1 text run")
	assert.Equal(t, 0, textMap["t3"], "t3 should be absent (zero value)")

	runtimeMap := BuildSkipMap(entries, "runtime")
	assert.Equal(t, 1, runtimeMap["t1"], "t1 should have 1 runtime run")
	assert.Equal(t, 0, runtimeMap["t2"], "t2 should be absent")
}

// TestExtractResults verifies filtering by condition.
func TestExtractResults(t *testing.T) {
	entries := []CheckpointEntry{
		{TaskID: "t1", Condition: "text", Result: ToResultJSON(TaskResult{TaskID: "t1", Condition: "text", Completed: true})},
		{TaskID: "t2", Condition: "text", Result: ToResultJSON(TaskResult{TaskID: "t2", Condition: "text", Completed: false})},
		{TaskID: "t1", Condition: "runtime", Result: ToResultJSON(TaskResult{TaskID: "t1", Condition: "runtime", Completed: true})},
	}

	textResults := ExtractResults(entries, "text")
	require.Len(t, textResults, 2)
	assert.Equal(t, "t1", textResults[0].TaskID)
	assert.Equal(t, "text", textResults[0].Condition)
	assert.Equal(t, "t2", textResults[1].TaskID)

	runtimeResults := ExtractResults(entries, "runtime")
	require.Len(t, runtimeResults, 1)
	assert.Equal(t, "t1", runtimeResults[0].TaskID)
	assert.Equal(t, "runtime", runtimeResults[0].Condition)
}

// TestLoadCheckpoint_FileNotFound verifies that a missing file returns an error.
func TestLoadCheckpoint_FileNotFound(t *testing.T) {
	_, err := LoadCheckpoint(filepath.Join(t.TempDir(), "nonexistent.jsonl"))
	require.Error(t, err)
}

// TestTaskResultJSON_ErrorSerializesToString verifies that the error field
// in TaskResultJSON marshals to a readable string, not an empty object "{}".
// This was the root cause of invisible errors in experiment-results-v2.
func TestTaskResultJSON_ErrorSerializesToString(t *testing.T) {
	creditErr := fmt.Errorf(`API error 400: {"error":{"message":"Your credit balance is too low"}}`)
	result := TaskResult{
		TaskID:    "test",
		Condition: "text",
		Error:     creditErr,
	}

	j := ToResultJSON(result)
	data, err := json.Marshal(j)
	require.NoError(t, err)

	jsonStr := string(data)
	assert.Contains(t, jsonStr, "credit balance is too low", "error message should be readable in JSON")
	assert.NotContains(t, jsonStr, `"error":{}`, "error should NOT serialize as empty object")

	// Verify it also unmarshals back correctly
	var decoded TaskResultJSON
	require.NoError(t, json.Unmarshal(data, &decoded))
	assert.Contains(t, decoded.Error, "credit balance is too low")
}

// TestTaskResult_DirectMarshalLosesError demonstrates the bug that was fixed:
// directly marshaling TaskResult (with error interface) produces "{}".
func TestTaskResult_DirectMarshalLosesError(t *testing.T) {
	result := TaskResult{
		TaskID: "test",
		Error:  fmt.Errorf("some error"),
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	// This is the bug: error interface marshals as {}
	assert.Contains(t, string(data), `"Error":{}`,
		"raw TaskResult.Error marshals as {} — this is why we use TaskResultJSON")
}
