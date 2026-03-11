package analyzer

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseTrace_ValidJSONL(t *testing.T) {
	jsonl := `{"timestamp":1,"type":"tool_call","skill":"lint","tool":"read_file","duration_ms":100,"tokens_in":50,"tokens_out":30}
{"timestamp":2,"type":"decision","skill":"lint","decision":"continue","confidence":0.95}`

	events := ParseTrace(jsonl)

	assert.Len(t, events, 2)
	assert.Equal(t, "tool_call", events[0].Type)
	assert.Equal(t, "lint", events[0].Skill)
	assert.Equal(t, "read_file", events[0].Tool)
	assert.Equal(t, 100.0, events[0].DurationMs)
	assert.Equal(t, 50, events[0].TokensIn)
	assert.Equal(t, 30, events[0].TokensOut)

	assert.Equal(t, "decision", events[1].Type)
	assert.Equal(t, "continue", events[1].Decision)
	assert.Equal(t, 0.95, events[1].Confidence)
}

func TestParseTrace_SkipsEmptyAndMalformed(t *testing.T) {
	jsonl := `{"timestamp":1,"type":"tool_call","skill":"a","tool":"b","duration_ms":10,"tokens_in":1,"tokens_out":1}

not valid json
{"timestamp":2,"type":"decision","skill":"a","decision":"done","confidence":1.0}
`

	events := ParseTrace(jsonl)

	assert.Len(t, events, 2)
	assert.Equal(t, "tool_call", events[0].Type)
	assert.Equal(t, "decision", events[1].Type)
}

func TestParseTrace_EmptyInput(t *testing.T) {
	events := ParseTrace("")
	assert.Empty(t, events)
}

func TestSummarizeTrace_Counts(t *testing.T) {
	events := []TraceEvent{
		{Timestamp: 1, Type: "tool_call", Skill: "lint", Tool: "read_file", DurationMs: 100, TokensIn: 50, TokensOut: 30},
		{Timestamp: 2, Type: "tool_call", Skill: "lint", Tool: "grep", DurationMs: 200, TokensIn: 40, TokensOut: 20},
		{Timestamp: 3, Type: "decision", Skill: "lint", Decision: "continue", Confidence: 0.9},
	}

	summary := SummarizeTrace(events)

	assert.Equal(t, 300.0, summary.TotalDurationMs)
	assert.Equal(t, 140, summary.TotalTokens) // 50+30+40+20
	assert.Equal(t, 2, summary.ToolCalls)
	assert.Equal(t, 1, summary.Decisions)
	assert.Equal(t, 1, summary.ToolFrequency["lint:read_file"])
	assert.Equal(t, 1, summary.ToolFrequency["lint:grep"])
	assert.Empty(t, summary.Warnings)
}

func TestSummarizeTrace_LoopWarning(t *testing.T) {
	var events []TraceEvent
	for i := 0; i < 6; i++ {
		events = append(events, TraceEvent{
			Timestamp:  float64(i),
			Type:       "tool_call",
			Skill:      "fetcher",
			Tool:       "http_get",
			DurationMs: 50,
			TokensIn:   10,
			TokensOut:  10,
		})
	}

	summary := SummarizeTrace(events)

	assert.Equal(t, 6, summary.ToolCalls)
	assert.Equal(t, 6, summary.ToolFrequency["fetcher:http_get"])
	assert.Len(t, summary.Warnings, 1)
	assert.True(t, strings.Contains(summary.Warnings[0], "http_get"))
	assert.True(t, strings.Contains(summary.Warnings[0], "6 times"))
	assert.True(t, strings.Contains(summary.Warnings[0], "fetcher"))
}

func TestSummarizeTrace_NoWarningBelowThreshold(t *testing.T) {
	var events []TraceEvent
	for i := 0; i < 4; i++ {
		events = append(events, TraceEvent{
			Timestamp:  float64(i),
			Type:       "tool_call",
			Skill:      "fetcher",
			Tool:       "http_get",
			DurationMs: 50,
			TokensIn:   10,
			TokensOut:  10,
		})
	}

	summary := SummarizeTrace(events)

	assert.Equal(t, 4, summary.ToolFrequency["fetcher:http_get"])
	assert.Empty(t, summary.Warnings)
}

func TestSummarizeTrace_EmptyEvents(t *testing.T) {
	summary := SummarizeTrace(nil)

	assert.Equal(t, 0.0, summary.TotalDurationMs)
	assert.Equal(t, 0, summary.TotalTokens)
	assert.Equal(t, 0, summary.ToolCalls)
	assert.Equal(t, 0, summary.Decisions)
	assert.Empty(t, summary.Warnings)
}
