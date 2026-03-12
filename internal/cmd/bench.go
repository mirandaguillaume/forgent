package cmd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/mirandaguillaume/forgent/internal/bench"
	"github.com/spf13/cobra"
)

func init() {
	benchCmd := &cobra.Command{
		Use:   "bench <repo-path>",
		Short: "Benchmark index quality for agent navigation",
		Long:  "Evaluates how well the generated codebase index helps an AI agent navigate the project.",
		Args:  cobra.ExactArgs(1),
		RunE:  runBench,
	}
	benchCmd.Flags().String("level", "proxy", "Benchmark level: proxy or agent")
	benchCmd.Flags().String("tasks", "", "YAML file with navigation tasks (agent level only)")
	benchCmd.Flags().Int("samples", 100, "Number of files to sample (proxy level only)")
	benchCmd.Flags().String("model", "haiku", "Claude model for agent bench")
	rootCmd.AddCommand(benchCmd)
}

func runBench(cmd *cobra.Command, args []string) error {
	repoPath, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}

	level, _ := cmd.Flags().GetString("level")

	switch level {
	case "proxy":
		return runProxyBench(cmd, repoPath)
	case "agent":
		return runAgentBench(cmd, repoPath)
	default:
		return fmt.Errorf("unknown level %q (use 'proxy' or 'agent')", level)
	}
}

func runProxyBench(cmd *cobra.Command, repoPath string) error {
	samples, _ := cmd.Flags().GetInt("samples")

	bold := color.New(color.Bold)
	bold.Fprintf(cmd.OutOrStdout(), "Proxy Benchmark: %s\n", repoPath)

	result, err := bench.RunProxy(repoPath, samples, time.Now().UnixNano())
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "  Source files:   %d\n", result.TotalSourceFiles)
	fmt.Fprintf(cmd.OutOrStdout(), "  Sampled:        %d\n", result.SampledFiles)

	reachColor := color.New(color.FgGreen)
	if result.Reachability < 80 {
		reachColor = color.New(color.FgYellow)
	}
	if result.Reachability < 60 {
		reachColor = color.New(color.FgRed)
	}
	reachColor.Fprintf(cmd.OutOrStdout(), "  Reachable:      %d/%d (%.1f%%)\n",
		result.ReachableFiles, result.SampledFiles, result.Reachability)

	fmt.Fprintf(cmd.OutOrStdout(), "  Index entries:  %d\n", result.IndexEntries)
	fmt.Fprintf(cmd.OutOrStdout(), "  Index size:     %d bytes\n", result.IndexBytes)

	return nil
}

func runAgentBench(cmd *cobra.Command, repoPath string) error {
	if !bench.ClaudeAvailable() {
		return fmt.Errorf("claude CLI not found in PATH (required for agent benchmark)")
	}

	model, _ := cmd.Flags().GetString("model")
	tasksFile, _ := cmd.Flags().GetString("tasks")

	bold := color.New(color.Bold)
	bold.Fprintf(cmd.OutOrStdout(), "Agent Benchmark: %s\n", repoPath)

	var tasks []bench.Task
	if tasksFile != "" {
		var err error
		tasks, err = bench.LoadTasks(tasksFile)
		if err != nil {
			return err
		}
	}
	// If no tasks file, RunAgent will auto-generate from the index.

	result, err := bench.RunAgent(repoPath, tasks, model)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "  Tasks:          %d\n", result.Tasks)

	hitColor := color.New(color.FgGreen)
	if result.HitRate < 70 {
		hitColor = color.New(color.FgYellow)
	}
	if result.HitRate < 50 {
		hitColor = color.New(color.FgRed)
	}
	hitColor.Fprintf(cmd.OutOrStdout(), "  Hits:           %d (%.1f%%)\n", result.Hits, result.HitRate)

	if result.Misses > 0 {
		color.New(color.FgYellow).Fprintf(cmd.OutOrStdout(), "  Misses:         %d\n", result.Misses)
	}
	if result.Errors > 0 {
		color.New(color.FgRed).Fprintf(cmd.OutOrStdout(), "  Errors:         %d\n", result.Errors)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "  Avg latency:    %s\n", result.AvgLatency.Round(time.Millisecond))

	if result.TotalTokens > 0 {
		avgTokens := result.TotalTokens / result.Tasks
		fmt.Fprintf(cmd.OutOrStdout(), "  Avg tokens/task: %d\n", avgTokens)
		fmt.Fprintf(cmd.OutOrStdout(), "  Total cost:     $%.4f\n", result.TotalCost)
		fmt.Fprintf(cmd.OutOrStdout(), "  Cost/task:      $%.4f\n", result.TotalCost/float64(result.Tasks))
	}

	// Show details for misses.
	for _, d := range result.Details {
		if !d.Hit && d.Err == nil {
			fmt.Fprintf(cmd.OutOrStdout(), "  MISS: %q → %q\n", d.Query, d.Response)
		}
	}

	return nil
}
