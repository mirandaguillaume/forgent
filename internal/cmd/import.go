package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/mirandaguillaume/forgent/internal/importer"
	"github.com/mirandaguillaume/forgent/internal/llm"
	"github.com/spf13/cobra"
)

func init() {
	var provider, outputDir string
	var minScore int
	var yes, dryRun, force bool

	importCmd := &cobra.Command{
		Use:   "import <source>",
		Short: "Import agent definitions as Forgent skill specs",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			source := args[0]

			runImport(source, provider, outputDir, minScore, yes, dryRun, force)
		},
	}

	importCmd.Flags().StringVarP(&provider, "provider", "p", "", "LLM provider name")
	importCmd.Flags().StringVarP(&outputDir, "output", "o", ".", "output directory")
	importCmd.Flags().IntVar(&minScore, "min-score", 60, "minimum quality score")
	importCmd.Flags().BoolVar(&yes, "yes", false, "skip confirmation prompt")
	importCmd.Flags().BoolVar(&dryRun, "dry-run", false, "preview without writing files")
	importCmd.Flags().BoolVar(&force, "force", false, "overwrite existing files")

	rootCmd.AddCommand(importCmd)
}

func runImport(source, providerFlag, outputDir string, minScore int, yes, dryRun, force bool) {
	// 1. Resolve provider name: flag → env → default
	providerName := providerFlag
	if providerName == "" {
		providerName = os.Getenv("FORGENT_LLM_PROVIDER")
	}
	if providerName == "" {
		providerName = "anthropic"
	}

	// 2. Resolve API key
	envMap := map[string]string{
		"FORGENT_API_KEY":    os.Getenv("FORGENT_API_KEY"),
		"ANTHROPIC_API_KEY":  os.Getenv("ANTHROPIC_API_KEY"),
		"OPENAI_API_KEY":     os.Getenv("OPENAI_API_KEY"),
		"OPENROUTER_API_KEY": os.Getenv("OPENROUTER_API_KEY"),
	}
	apiKey, err := resolveAPIKey(providerName, envMap)
	if err != nil {
		fmt.Println(color.RedString("Error: %v", err))
		os.Exit(1)
	}

	// 3. Get provider
	llmProvider, err := llm.GetProvider(providerName, apiKey)
	if err != nil {
		fmt.Println(color.RedString("Error: %v", err))
		os.Exit(1)
	}

	// 4. Run import pipeline
	result := importer.RunImport(importer.ImportOptions{
		Source:   source,
		Provider: llmProvider,
		MinScore: minScore,
		OutputDir: outputDir,
	})

	if result.Error != "" {
		fmt.Println(color.RedString("Import failed: %s", result.Error))
		os.Exit(1)
	}

	// 5. Show preview
	importer.FormatPreview(result, os.Stdout)

	if dryRun {
		return
	}

	// 6. Prompt for confirmation unless --yes
	if !yes {
		skillCount := len(result.Skills)
		agentCount := 0
		if result.Agent != nil {
			agentCount = 1
		}
		fmt.Printf("Write %d skill(s) + %d agent(s)? [y/N] ", skillCount, agentCount)
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
		if answer != "y" && answer != "yes" {
			fmt.Println("Aborted.")
			return
		}
	}

	// 7. If --force, remove existing files before writing
	if force {
		removeExistingFiles(result, outputDir)
	}

	// 8. Write files
	written, err := importer.WriteImportResult(result, outputDir)
	if err != nil {
		fmt.Println(color.RedString("Write failed: %v", err))
		os.Exit(1)
	}

	// 9. Print written files
	for _, path := range written {
		fmt.Println(color.GreenString("  wrote %s", path))
	}
}

// resolveAPIKey resolves the API key for the given provider from an environment
// map. Priority: FORGENT_API_KEY → provider-specific key → error.
func resolveAPIKey(provider string, env map[string]string) (string, error) {
	// Check FORGENT_API_KEY first.
	if key := env["FORGENT_API_KEY"]; key != "" {
		return key, nil
	}

	// Check provider-specific key.
	var envVar string
	switch provider {
	case "anthropic":
		envVar = "ANTHROPIC_API_KEY"
	case "openai":
		envVar = "OPENAI_API_KEY"
	case "openrouter":
		envVar = "OPENROUTER_API_KEY"
	default:
		envVar = strings.ToUpper(provider) + "_API_KEY"
	}

	if key := env[envVar]; key != "" {
		return key, nil
	}

	return "", fmt.Errorf("no API key found: set FORGENT_API_KEY or %s", envVar)
}

// removeExistingFiles removes files that would conflict with WriteImportResult.
func removeExistingFiles(result importer.ImportResult, outputDir string) {
	for _, sr := range result.Skills {
		name := sr.Skill.Skill
		if name == "" {
			name = "unknown"
		}
		path := filepath.Join(outputDir, "skills", name+".skill.yaml")
		os.Remove(path)
	}
	if result.Agent != nil {
		name := result.Agent.Agent.Agent
		if name == "" {
			name = "unknown"
		}
		path := filepath.Join(outputDir, "agents", name+".agent.yaml")
		os.Remove(path)
	}
}
