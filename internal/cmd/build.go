package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/mirandaguillaume/forgent/internal/builder"
	"github.com/mirandaguillaume/forgent/internal/scanner"
	"github.com/mirandaguillaume/forgent/pkg/spec"
	"github.com/spf13/cobra"
)

// BuildResult is an alias for builder.BuildResult for backward compatibility.
type BuildResult = builder.BuildResult

// RunBuild delegates to builder.RunBuild.
func RunBuild(skillsDir, agentsDir, outputDir, target string, enrichMode scanner.EnrichMode) BuildResult {
	return builder.RunBuild(skillsDir, agentsDir, outputDir, target, enrichMode)
}

// RunBuildWithOptions delegates to builder.RunBuildWithOptions.
func RunBuildWithOptions(skillsDir, agentsDir, outputDir, target string, enrichMode scanner.EnrichMode, compact bool) BuildResult {
	return builder.RunBuildWithOptions(skillsDir, agentsDir, outputDir, target, enrichMode, compact)
}

// GetOutputDir delegates to builder.GetOutputDir.
func GetOutputDir(target, override string) string {
	return builder.GetOutputDir(target, override)
}

// PrintBuildResult prints the build result to stdout with colored output.
func PrintBuildResult(result BuildResult) {
	if !result.Success {
		fmt.Println(color.RedString("Build failed: %s", result.Error))
		return
	}

	fmt.Println(color.GreenString("Build complete (target: %s):", result.Target))
	fmt.Printf("  Output: %s\n", result.OutputDir)
	fmt.Printf("  Skills generated: %d\n", result.SkillsGenerated)
	fmt.Printf("  Agents generated: %d\n", result.AgentsGenerated)

	if len(result.Warnings) > 0 {
		fmt.Println(color.YellowString("\nWarnings:"))
		for _, w := range result.Warnings {
			fmt.Printf("  %s %s\n", color.YellowString("!"), w)
		}
	}
}

func init() {
	var target, skillsDir, agentsDir, outputDirFlag, enrichFlag string
	var watchFlag, compactFlag bool

	buildCmd := &cobra.Command{
		Use:   "build",
		Short: "Generate skills and agents for a target framework",
		Run: func(cmd *cobra.Command, args []string) {
			available := spec.Available()
			found := false
			for _, a := range available {
				if a == target {
					found = true
					break
				}
			}
			if !found {
				fmt.Println(color.RedString("Unknown target %q. Available: %s", target, strings.Join(available, ", ")))
				os.Exit(1)
			}

			enrichMode := scanner.EnrichMode(enrichFlag)

			outputDir := GetOutputDir(target, outputDirFlag)

			if watchFlag {
				controller := CreateWatcher(WatchOptions{
					SkillsDir:  skillsDir,
					AgentsDir:  agentsDir,
					OutputDir:  outputDir,
					Target:     target,
					EnrichMode: enrichMode,
				})
				defer controller.Stop()
				sigCh := make(chan os.Signal, 1)
				signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
				<-sigCh
				return
			}

			result := RunBuildWithOptions(skillsDir, agentsDir, outputDir, target, enrichMode, compactFlag)
			PrintBuildResult(result)
			if !result.Success {
				os.Exit(1)
			}
		},
	}

	buildCmd.Flags().StringVarP(&target, "target", "t", "claude", "target framework")
	buildCmd.Flags().StringVarP(&skillsDir, "skills", "s", "skills", "skills directory")
	buildCmd.Flags().StringVarP(&agentsDir, "agents", "a", "agents", "agents directory")
	buildCmd.Flags().StringVarP(&outputDirFlag, "output", "o", "", "output directory")
	buildCmd.Flags().BoolVarP(&watchFlag, "watch", "w", false, "watch for changes")
	buildCmd.Flags().BoolVar(&compactFlag, "compact", false, "inline skills into agent file for lower token overhead")
	buildCmd.Flags().StringVar(&enrichFlag, "enrich", "", "enrich skills with codebase context (index|full)")
	buildCmd.Flag("enrich").NoOptDefVal = "index"

	rootCmd.AddCommand(buildCmd)
}
