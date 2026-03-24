package bench

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGenerateReport_AllBenches(t *testing.T) {
	report := BenchReport{
		Timestamp: time.Date(2026, 3, 15, 14, 0, 0, 0, time.UTC),
		Agents:    []string{"agentless-repair", "sweagent-default"},
		SWEBench: &SWEH2HResult{
			Tasks: 30,
			Contestants: []SWEContestantResult{
				{Name: "agentless-repair", Words: 500, PatchRate: 80, ApplyRate: 40, Efficiency: 0.8},
				{Name: "agentless-repair/forgent-standard", Words: 600, PatchRate: 85, ApplyRate: 45, Efficiency: 0.75},
			},
		},
	}

	md := GenerateReport(report)

	// Check header
	assert.Contains(t, md, "# Forgent Benchmark Report")
	assert.Contains(t, md, "2026-03-15")
	assert.Contains(t, md, "agentless-repair, sweagent-default")

	// Check summary table
	assert.Contains(t, md, "| SWE-bench Verified |")

	// Check section headers
	assert.Contains(t, md, "## SWE-bench Verified")

	// Check it has contestant data
	assert.Contains(t, md, "agentless-repair/forgent-standard")

	t.Logf("Report (%d chars):\n%s", len(md), md)
}

func TestGenerateReport_PartialResults(t *testing.T) {
	report := BenchReport{
		Timestamp: time.Date(2026, 3, 15, 14, 0, 0, 0, time.UTC),
		SWEBench: &SWEH2HResult{
			Tasks: 5,
			Contestants: []SWEContestantResult{
				{Name: "test", Words: 100, ApplyRate: 20},
			},
		},
		Errors: map[string]string{
			"swebench-live": "claude CLI not found",
		},
	}

	md := GenerateReport(report)
	assert.Contains(t, md, "## SWE-bench Verified")
	assert.Contains(t, md, "Skipped benchmarks")
	assert.Contains(t, md, "claude CLI not found")

	// Count sections — should only have SWE-bench
	assert.Equal(t, 1, strings.Count(md, "## SWE-bench"))
}

func TestGenerateReport_Empty(t *testing.T) {
	report := BenchReport{
		Timestamp: time.Now(),
		Errors: map[string]string{
			"all": "no API key",
		},
	}

	md := GenerateReport(report)
	assert.Contains(t, md, "# Forgent Benchmark Report")
	assert.Contains(t, md, "no API key")
}
