package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/mirandaguillaume/forgent/internal/analyzer"
	"github.com/spf13/cobra"
)

// TraceFile reads and analyzes a JSONL trace file, printing the summary.
func TraceFile(tracePath string) {
	content, err := os.ReadFile(tracePath)
	if err != nil {
		fmt.Println(color.RedString("Trace file not found: %s", tracePath))
		os.Exit(1)
	}

	events := analyzer.ParseTrace(string(content))
	summary := analyzer.SummarizeTrace(events)

	bold := color.New(color.Bold)
	fmt.Println()
	bold.Println("=== Forgent Trace Summary ===")
	fmt.Println()
	fmt.Printf("Total duration: %.0fms\n", summary.TotalDurationMs)
	fmt.Printf("Total tokens: %d\n", summary.TotalTokens)
	fmt.Printf("Tool calls: %d\n", summary.ToolCalls)
	fmt.Printf("Decisions: %d\n", summary.Decisions)

	if len(summary.ToolFrequency) > 0 {
		bold.Println("\nTool usage:")
		for tool, count := range summary.ToolFrequency {
			fmt.Printf("  %s: %dx\n", tool, count)
		}
	}

	if len(summary.Warnings) > 0 {
		fmt.Println(color.YellowString("\nWarnings:"))
		for _, w := range summary.Warnings {
			fmt.Printf("  %s %s\n", color.YellowString("!"), w)
		}
	}
	fmt.Println()
}

func init() {
	traceCmd := &cobra.Command{
		Use:   "trace <file>",
		Short: "Analyze a JSONL trace file",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			TraceFile(args[0])
		},
	}
	rootCmd.AddCommand(traceCmd)
}
