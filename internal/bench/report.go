package bench

import (
	"fmt"
	"strings"
	"time"
)

// BenchReport aggregates results from all benchmark levels.
type BenchReport struct {
	Timestamp time.Time
	Agents    []string
	SWEBench  *SWEH2HResult
	Errors    map[string]string // bench name → error message
}

// GenerateReport produces a markdown report from aggregated bench results.
func GenerateReport(report BenchReport) string {
	var b strings.Builder

	b.WriteString("# Forgent Benchmark Report\n\n")
	b.WriteString(fmt.Sprintf("**Generated:** %s\n\n", report.Timestamp.Format("2006-01-02 15:04:05")))

	if len(report.Agents) > 0 {
		b.WriteString(fmt.Sprintf("**Agents:** %s\n\n", strings.Join(report.Agents, ", ")))
	}

	b.WriteString("---\n\n")

	// Summary table
	b.WriteString("## Summary\n\n")
	b.WriteString("| Benchmark | Tasks | Metric | Best Contestant | Score |\n")
	b.WriteString("|-----------|-------|--------|-----------------|-------|\n")

	if report.SWEBench != nil {
		best := bestSWEContestant(report.SWEBench)
		b.WriteString(fmt.Sprintf("| SWE-bench Verified | %d | Apply Rate | %s | %.1f%% |\n",
			report.SWEBench.Tasks, best.Name, best.ApplyRate))
	}

	// Errors
	if len(report.Errors) > 0 {
		b.WriteString("\n**Skipped benchmarks:**\n")
		for name, err := range report.Errors {
			b.WriteString(fmt.Sprintf("- %s: %s\n", name, err))
		}
	}

	b.WriteString("\n---\n\n")

	// Detailed sections
	if report.SWEBench != nil {
		writeSWEBenchSection(&b, report.SWEBench)
	}

	return b.String()
}

func writeSWEBenchSection(b *strings.Builder, result *SWEH2HResult) {
	b.WriteString("## SWE-bench Verified\n\n")
	b.WriteString(fmt.Sprintf("**Tasks:** %d\n\n", result.Tasks))
	b.WriteString("| Contestant | Patch Rate | Apply Rate | Efficiency | Words |\n")
	b.WriteString("|------------|-----------|------------|------------|-------|\n")
	for _, c := range result.Contestants {
		b.WriteString(fmt.Sprintf("| %s | %.1f%% | %.1f%% | %.1f | %d |\n",
			c.Name, c.PatchRate, c.ApplyRate, c.Efficiency, c.Words))
	}
	b.WriteString("\n")
}

// Best contestant helper — pick by primary metric.
func bestSWEContestant(r *SWEH2HResult) SWEContestantResult {
	best := r.Contestants[0]
	for _, c := range r.Contestants[1:] {
		if c.ApplyRate > best.ApplyRate {
			best = c
		}
	}
	return best
}
