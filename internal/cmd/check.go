package cmd

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/mirandaguillaume/forgent/internal/checker"
	"github.com/mirandaguillaume/forgent/internal/classifier"
	"github.com/spf13/cobra"
)

func init() {
	var providerFlag, modelFlag string
	var quick, all bool
	var targets []string
	threshold := 0.80

	checkCmd := &cobra.Command{
		Use:   "check <file>",
		Short: "Estimate instruction following compliance per harness",
		Long: `Analyzes each instruction in an agent file and estimates how reliably
it will be followed on each AI coding harness (Claude Code, Copilot, Cursor).

Uses research-backed heuristics:
  - Position effect: Liu et al. 2023 "Lost in the Middle"
  - Framing effect: IFEval benchmark (Zhou et al. 2023)
  - Harness deltas: empirical estimates (labeled as such)

Use --bench to measure actual compliance empirically.`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			runCheck(args[0], providerFlag, modelFlag, targets, quick, all, threshold)
		},
	}

	checkCmd.Flags().StringVar(&providerFlag, "provider", "", "LLM provider for classification (ollama, anthropic…)")
	checkCmd.Flags().StringVar(&modelFlag, "model", "claude-haiku-4-5-20251001", "LLM model name")
	checkCmd.Flags().BoolVar(&quick, "quick", false, "static classification only — no LLM")
	checkCmd.Flags().BoolVar(&all, "all", false, "show all instructions including high-compliance ones")
	checkCmd.Flags().Float64Var(&threshold, "threshold", 0.80, "only show instructions below this compliance score")
	checkCmd.Flags().StringSliceVar(&targets, "targets", checker.Harnesses, "harnesses to check against")

	rootCmd.AddCommand(checkCmd)
}

func runCheck(filePath, providerFlag, modelFlag string, targets []string, quick, all bool, threshold float64) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Println(color.RedString("Error: %v", err))
		os.Exit(1)
	}

	format := detectFormat(filePath, string(content))

	var cl classifier.Result
	if quick {
		cl = classifier.Classify(string(content), format)
	} else {
		provider, _, provErr := selectProvider(providerFlag, modelFlag, false)
		if provErr != nil {
			fmt.Println(color.YellowString("No LLM provider — falling back to static classification"))
			cl = classifier.Classify(string(content), format)
		} else {
			cl, _ = classifier.ClassifyLLM(string(content), format, provider)
		}
	}

	if len(cl.Items) == 0 {
		fmt.Println(color.YellowString("No instructions found."))
		return
	}

	result := checker.Check(string(content), format, targets, cl)
	printCheckResult(filePath, result, targets, all, threshold)
}

func printCheckResult(filePath string, result checker.CheckResult, targets []string, all bool, threshold float64) {
	bold := color.New(color.Bold)
	bold.Printf("Compliance check: %s\n", filePath)
	fmt.Println()

	// Per-facet breakdown
	byFacet := groupByFacet(result.Instructions)
	facetLabels := map[classifier.Facet]string{
		classifier.FacetContext:       "Context",
		classifier.FacetStrategy:      "Strategy",
		classifier.FacetGuardrails:    "Guardrails",
		classifier.FacetObservability: "Observability",
		classifier.FacetSecurity:      "Security",
	}

	for _, facet := range classifier.AllFacets {
		items := byFacet[facet]
		if len(items) == 0 {
			continue
		}
		label := facetLabels[facet]
		fmt.Printf("  %s (%d)\n", color.New(color.Bold).Sprint(label), len(items))
		shown := 0
		for _, ir := range items {
			// Skip high-compliance items without suggestions unless --all.
			if !all {
				minScore := 1.0
				for _, h := range targets {
					if s := ir.Scores[h]; s < minScore {
						minScore = s
					}
				}
				if minScore >= threshold && len(ir.Suggestions) == 0 {
					continue
				}
			}
			shown++
			text := ir.Text
			if len(text) > 70 {
				text = text[:67] + "..."
			}
			fmt.Printf("    · %s\n", text)
			for _, harness := range targets {
				score := ir.Scores[harness]
				bar := complianceBar(score)
				flag := complianceFlag(score)
				fmt.Printf("        %-12s %s %s\n", harness, bar, flag)
			}
			for _, s := range ir.Suggestions {
				fmt.Printf("        %s %s\n", color.YellowString("↑"), color.YellowString(s))
			}
		}
		if !all && shown < len(items) {
			fmt.Printf("    %s\n", color.New(color.Faint).Sprintf("… %d more instructions above threshold (use --all to show)", len(items)-shown))
		}
		fmt.Println()
	}

	// Overall summary
	fmt.Printf("  %s\n", color.New(color.Bold).Sprint("Overall compliance (estimated)"))
	for _, h := range targets {
		score := result.Overall[h]
		bar := complianceBar(score)
		flag := complianceFlag(score)
		fmt.Printf("    %-12s %s %s\n", h, bar, flag)
	}
	fmt.Println()

	// Warnings
	if len(result.Warnings) > 0 {
		fmt.Printf("  %s\n", color.YellowString("⚠ Low compliance warnings:"))
		for _, w := range result.Warnings {
			fmt.Printf("    · %s\n", w)
		}
		fmt.Println()
	}

	fmt.Println(color.New(color.Faint).Sprint("  Scores are estimates. Run 'forgent bench' to measure empirically."))
}

func complianceBar(score float64) string {
	filled := int(math.Round(score * 10))
	bar := strings.Repeat("█", filled) + strings.Repeat("░", 10-filled)
	pct := int(math.Round(score * 100))
	if pct >= 80 {
		return color.GreenString(bar)
	}
	if pct >= 60 {
		return color.YellowString(bar)
	}
	return color.RedString(bar)
}

func complianceFlag(score float64) string {
	pct := int(math.Round(score * 100))
	switch {
	case pct >= 80:
		return color.GreenString(fmt.Sprintf("%d%%", pct))
	case pct >= 60:
		return color.YellowString(fmt.Sprintf("%d%% ⚠", pct))
	default:
		return color.RedString(fmt.Sprintf("%d%% ✗", pct))
	}
}

func groupByFacet(items []checker.InstructionResult) map[classifier.Facet][]checker.InstructionResult {
	m := make(map[classifier.Facet][]checker.InstructionResult)
	for _, item := range items {
		m[item.Facet] = append(m[item.Facet], item)
	}
	return m
}
