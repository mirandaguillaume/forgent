package bench

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// InstructionReportOpts configures report formatting.
type InstructionReportOpts struct {
	Model       string
	CorpusDir   string
	Version     string // registry version
	GoldenFiles int    // number of golden-annotated files
}

// FormatInstructionReport produces a Markdown report from InstructionBench results.
func FormatInstructionReport(result *InstructionBenchResult, opts InstructionReportOpts) string {
	var sb strings.Builder

	sb.WriteString("# InstructionBench Report\n\n")

	// Metadata
	sb.WriteString("## Metadata\n\n")
	sb.WriteString(fmt.Sprintf("| Field | Value |\n"))
	sb.WriteString(fmt.Sprintf("|-------|-------|\n"))
	sb.WriteString(fmt.Sprintf("| Date | %s |\n", time.Now().Format("2006-01-02")))
	if opts.Model != "" {
		sb.WriteString(fmt.Sprintf("| Model | %s |\n", opts.Model))
	}
	sb.WriteString(fmt.Sprintf("| Corpus size | %d files |\n", result.TotalFiles))
	if opts.Version != "" {
		sb.WriteString(fmt.Sprintf("| Registry version | %s |\n", opts.Version))
	}
	if opts.GoldenFiles > 0 {
		sb.WriteString(fmt.Sprintf("| Golden set | %d annotated files |\n", opts.GoldenFiles))
	}
	sb.WriteString("\n")

	// Methodology
	sb.WriteString("## Methodology\n\n")
	sb.WriteString("InstructionBench measures the quality of `forgent doctor`'s recommendations ")
	sb.WriteString("by running the analyzer against a corpus of real agent files scraped from public GitHub repos. ")
	sb.WriteString("Each file is parsed, analyzed with the configured LLM provider, and findings are collected. ")
	sb.WriteString("Metrics are computed per category: finding count, acceptance rate (% of findings with confidence >= moderate), ")
	sb.WriteString("and false positive rate (when a golden set is available).\n\n")

	// Corpus statistics
	sb.WriteString("## Corpus Statistics\n\n")
	formatCounts := make(map[string]int)
	errorCount := 0
	for _, fr := range result.FileResults {
		if fr.Error != nil {
			errorCount++
		} else {
			formatCounts[fr.Format]++
		}
	}
	sb.WriteString(fmt.Sprintf("| Format | Files |\n"))
	sb.WriteString(fmt.Sprintf("|--------|-------|\n"))
	formats := sortedMapKeys(formatCounts)
	for _, f := range formats {
		sb.WriteString(fmt.Sprintf("| %s | %d |\n", f, formatCounts[f]))
	}
	if errorCount > 0 {
		sb.WriteString(fmt.Sprintf("| (errors) | %d |\n", errorCount))
	}
	sb.WriteString(fmt.Sprintf("| **Total** | **%d** |\n", result.TotalFiles))
	sb.WriteString("\n")

	// Per-category results
	sb.WriteString("## Per-Category Results\n\n")
	sb.WriteString("| Category | Findings | Acceptance Rate | FP Rate |\n")
	sb.WriteString("|----------|----------|-----------------|----------|\n")
	categories := sortedMapKeys(result.ByCategory)
	for _, cat := range categories {
		cr := result.ByCategory[cat]
		fpStr := "N/A"
		if cr.FalsePositives > 0 || opts.GoldenFiles > 0 {
			fpStr = fmt.Sprintf("%.0f%% (%d)", cr.FalsePositiveRate, cr.FalsePositives)
		}
		sb.WriteString(fmt.Sprintf("| %s | %d | %.0f%% | %s |\n", cat, cr.Count, cr.AcceptanceRate, fpStr))
	}
	sb.WriteString(fmt.Sprintf("| **Total** | **%d** | | |\n", result.TotalFindings))
	sb.WriteString("\n")

	// Confidence distribution
	sb.WriteString("## Confidence Distribution\n\n")
	sb.WriteString("| Confidence | Count | Percentage |\n")
	sb.WriteString("|------------|-------|------------|\n")
	confOrder := []string{"high", "moderate", "low", "unknown"}
	for _, conf := range confOrder {
		count, ok := result.ConfidenceDistribution[conf]
		if !ok {
			continue
		}
		pct := 0.0
		if result.TotalFindings > 0 {
			pct = float64(count) / float64(result.TotalFindings) * 100
		}
		sb.WriteString(fmt.Sprintf("| %s | %d | %.0f%% |\n", conf, count, pct))
	}
	sb.WriteString("\n")

	// Overall quality score
	sb.WriteString("## Overall Quality Score\n\n")
	sb.WriteString(fmt.Sprintf("**%.0f%%** of files received at least one recommendation.\n\n", result.OverallScore))
	sb.WriteString(fmt.Sprintf("Average findings per file: %.1f\n\n", avgFindings(result)))

	return sb.String()
}

func avgFindings(result *InstructionBenchResult) float64 {
	if result.TotalFiles == 0 {
		return 0
	}
	return float64(result.TotalFindings) / float64(result.TotalFiles)
}

func sortedMapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
