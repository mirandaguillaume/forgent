package checker

import (
	"strings"

	"github.com/mirandaguillaume/forgent/internal/classifier"
)

// Harnesses supported by the compliance estimator.
var Harnesses = []string{"claude-code", "copilot", "cursor"}

// InstructionResult holds compliance estimates for one instruction.
type InstructionResult struct {
	Text         string
	Facet        classifier.Facet
	Section      string
	Position     float64            // 0.0 = top of file, 1.0 = bottom
	IsNegative   bool               // starts with never/don't/avoid/must not
	IsVerifiable bool               // can be checked statically/deterministically
	Scores       map[string]float64 // harness → 0.0-1.0
	Suggestions  []string
}

// CheckResult is the output of the check command.
type CheckResult struct {
	Instructions []InstructionResult
	Overall      map[string]float64 // harness → mean compliance
	Warnings     []string
}

// Check classifies and estimates compliance for each instruction in content.
// format is the detected source format (claude, copilot, etc.).
// targets is the list of harnesses to estimate against (defaults to Harnesses).
func Check(content, format string, targets []string, classification classifier.Result) CheckResult {
	if len(targets) == 0 {
		targets = Harnesses
	}

	items := classification.Items
	if len(items) == 0 {
		return CheckResult{}
	}

	results := make([]InstructionResult, 0, len(items))
	lines := strings.Split(content, "\n")
	total := len(lines)

	for _, item := range items {
		pos := estimatePosition(item.Text, lines, total)
		isNeg := isNegativeInstruction(item.Text)
		isVerif := isVerifiableInstruction(item.Text, item.Facet)

		ir := InstructionResult{
			Text:         item.Text,
			Facet:        item.Facet,
			Section:      item.Section,
			Position:     pos,
			IsNegative:   isNeg,
			IsVerifiable: isVerif,
			Scores:       make(map[string]float64, len(targets)),
		}

		for _, harness := range targets {
			ir.Scores[harness] = EstimateCompliance(ir, harness)
		}

		ir.Suggestions = buildSuggestions(ir, targets)
		results = append(results, ir)
	}

	overall := computeOverall(results, targets)
	warnings := buildWarnings(results, targets)

	return CheckResult{
		Instructions: results,
		Overall:      overall,
		Warnings:     warnings,
	}
}

func estimatePosition(text string, lines []string, total int) float64 {
	if total == 0 {
		return 0.5
	}
	for i, line := range lines {
		if strings.Contains(line, text[:min(len(text), 30)]) {
			return float64(i) / float64(total)
		}
	}
	return 0.5
}

func computeOverall(results []InstructionResult, targets []string) map[string]float64 {
	sums := make(map[string]float64, len(targets))
	counts := make(map[string]int, len(targets))
	for _, r := range results {
		for h, s := range r.Scores {
			sums[h] += s
			counts[h]++
		}
	}
	overall := make(map[string]float64, len(targets))
	for h, sum := range sums {
		if counts[h] > 0 {
			overall[h] = sum / float64(counts[h])
		}
	}
	return overall
}

func buildWarnings(results []InstructionResult, targets []string) []string {
	var warnings []string
	for _, r := range results {
		for _, h := range targets {
			if score, ok := r.Scores[h]; ok && score < 0.55 {
				text := r.Text
				if len(text) > 60 {
					text = text[:57] + "..."
				}
				warnings = append(warnings, h+": low compliance on \""+text+"\"")
			}
		}
	}
	return warnings
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
