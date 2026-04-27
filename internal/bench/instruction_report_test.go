package bench

import (
	"strings"
	"testing"

	"github.com/mirandaguillaume/forgent/internal/doctor/llmutil"
	"github.com/stretchr/testify/assert"
)

func TestFormatInstructionReport_AllSections(t *testing.T) {
	result := &InstructionBenchResult{
		TotalFiles:    10,
		TotalFindings: 25,
		ByCategory: map[string]*CategoryResult{
			"tool-integration":    {Count: 10, AcceptanceRate: 80.0, FalsePositives: 2},
			"context-management":  {Count: 8, AcceptanceRate: 62.5},
			"guardrails":          {Count: 7, AcceptanceRate: 100.0},
		},
		ConfidenceDistribution: map[string]int{
			"high":     10,
			"moderate": 10,
			"low":      5,
		},
		OverallScore: 90.0,
		FileResults: []FileResult{
			{Path: "a.md", Format: "claude", Findings: make([]llmutil.Finding, 3)},
			{Path: "b.md", Format: "copilot", Findings: make([]llmutil.Finding, 2)},
		},
	}

	report := FormatInstructionReport(result, InstructionReportOpts{
		Model:       "llama3.1:8b",
		Version:     "2026.03.2",
		GoldenFiles: 20,
	})

	// Required sections per AC #4
	assert.Contains(t, report, "# InstructionBench Report")
	assert.Contains(t, report, "## Metadata")
	assert.Contains(t, report, "## Methodology")
	assert.Contains(t, report, "## Corpus Statistics")
	assert.Contains(t, report, "## Per-Category Results")
	assert.Contains(t, report, "## Confidence Distribution")
	assert.Contains(t, report, "## Overall Quality Score")

	// Metadata fields
	assert.Contains(t, report, "llama3.1:8b")
	assert.Contains(t, report, "10 files")
	assert.Contains(t, report, "2026.03.2")
	assert.Contains(t, report, "20 annotated files")

	// Category data
	assert.Contains(t, report, "tool-integration")
	assert.Contains(t, report, "context-management")
	assert.Contains(t, report, "guardrails")

	// Confidence data
	assert.Contains(t, report, "high")
	assert.Contains(t, report, "moderate")
	assert.Contains(t, report, "low")

	// Overall score
	assert.Contains(t, report, "90%")
}

func TestFormatInstructionReport_EmptyResult(t *testing.T) {
	result := &InstructionBenchResult{
		ByCategory:             make(map[string]*CategoryResult),
		ConfidenceDistribution: make(map[string]int),
	}

	report := FormatInstructionReport(result, InstructionReportOpts{})
	assert.Contains(t, report, "# InstructionBench Report")
	assert.Contains(t, report, "0 files")
	assert.Contains(t, report, "0%")
}

func TestFormatInstructionReport_IsMarkdown(t *testing.T) {
	result := &InstructionBenchResult{
		TotalFiles:    5,
		TotalFindings: 10,
		ByCategory: map[string]*CategoryResult{
			"test": {Count: 10, AcceptanceRate: 50.0},
		},
		ConfidenceDistribution: map[string]int{"moderate": 10},
		OverallScore:           100.0,
	}

	report := FormatInstructionReport(result, InstructionReportOpts{Model: "test"})

	// Must be valid Markdown (headers, tables)
	lines := strings.Split(report, "\n")
	headerCount := 0
	tableCount := 0
	for _, line := range lines {
		if strings.HasPrefix(line, "#") {
			headerCount++
		}
		if strings.HasPrefix(line, "|") {
			tableCount++
		}
	}
	assert.GreaterOrEqual(t, headerCount, 5, "report must have at least 5 Markdown headers")
	assert.GreaterOrEqual(t, tableCount, 6, "report must have Markdown tables")
}
