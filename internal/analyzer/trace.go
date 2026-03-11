package analyzer

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TraceEvent represents a single event in a JSONL trace file.
type TraceEvent struct {
	Timestamp  float64 `json:"timestamp"`
	Type       string  `json:"type"`
	Skill      string  `json:"skill"`
	Tool       string  `json:"tool,omitempty"`
	DurationMs float64 `json:"duration_ms,omitempty"`
	TokensIn   int     `json:"tokens_in,omitempty"`
	TokensOut  int     `json:"tokens_out,omitempty"`
	Decision   string  `json:"decision,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
}

// TraceSummary aggregates statistics from trace events.
type TraceSummary struct {
	TotalDurationMs float64
	TotalTokens     int
	ToolCalls       int
	Decisions       int
	ToolFrequency   map[string]int
	Warnings        []string
}

const loopThreshold = 5

// ParseTrace parses a JSONL string into a slice of TraceEvents.
// Empty and malformed lines are skipped.
func ParseTrace(jsonl string) []TraceEvent {
	var events []TraceEvent
	for _, line := range strings.Split(jsonl, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var event TraceEvent
		if err := json.Unmarshal([]byte(line), &event); err == nil {
			events = append(events, event)
		}
	}
	return events
}

// SummarizeTrace aggregates trace events into a summary with totals,
// frequency counts, and loop warnings.
func SummarizeTrace(events []TraceEvent) TraceSummary {
	summary := TraceSummary{
		ToolFrequency: make(map[string]int),
	}

	type freqInfo struct {
		count int
		skill string
	}
	freq := make(map[string]freqInfo)

	for _, event := range events {
		if event.Type == "tool_call" {
			summary.TotalDurationMs += event.DurationMs
			summary.TotalTokens += event.TokensIn + event.TokensOut
			summary.ToolCalls++

			key := event.Skill + ":" + event.Tool
			f := freq[key]
			f.count++
			f.skill = event.Skill
			freq[key] = f
		} else if event.Type == "decision" {
			summary.Decisions++
		}
	}

	for key, f := range freq {
		summary.ToolFrequency[key] = f.count
		if f.count >= loopThreshold {
			tool := strings.SplitN(key, ":", 2)[1]
			summary.Warnings = append(summary.Warnings,
				fmt.Sprintf("Tool %q called %d times by %q — possible loop", tool, f.count, f.skill))
		}
	}

	return summary
}
