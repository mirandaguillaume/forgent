package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/fatih/color"
	"github.com/mirandaguillaume/forgent/internal/analyzer"
	"github.com/mirandaguillaume/forgent/internal/enricher"
	"github.com/mirandaguillaume/forgent/internal/linter"
	"github.com/mirandaguillaume/forgent/internal/scanner"
	yamlloader "github.com/mirandaguillaume/forgent/internal/yaml"
	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/mirandaguillaume/forgent/pkg/spec"
	"github.com/spf13/cobra"

	// Register generators
	_ "github.com/mirandaguillaume/forgent/internal/generator/claude"
	_ "github.com/mirandaguillaume/forgent/internal/generator/copilot"
)

const wordLimit = 500

// CodebaseIndexKey is the consumes value that triggers codebase index generation.
// The build pipeline produces the index; skills that need it declare it in consumes.
const CodebaseIndexKey = "codebase_index"

// skillConsumesIndex returns true if the skill declares codebase_index in consumes.
func skillConsumesIndex(skill model.SkillBehavior) bool {
	for _, c := range skill.Context.Consumes {
		if c == CodebaseIndexKey {
			return true
		}
	}
	return false
}

// BuildResult holds the outcome of a build operation.
type BuildResult struct {
	Success         bool
	Error           string
	Target          string
	OutputDir       string
	SkillsGenerated int
	AgentsGenerated int
	Warnings        []string
}

// GetOutputDir returns the output directory, using override if set or the generator default.
func GetOutputDir(target, override string) string {
	if override != "" {
		return override
	}
	gen, err := spec.Get(target)
	if err != nil {
		return ".claude" // fallback
	}
	return gen.DefaultOutputDir()
}

// RunBuild executes the full build pipeline: parse, lint, generate skills/agents/instructions.
func RunBuild(skillsDir, agentsDir, outputDir, target string, enrichMode scanner.EnrichMode) BuildResult {
	result := BuildResult{Target: target, OutputDir: outputDir}

	gen, err := spec.Get(target)
	if err != nil {
		result.Error = err.Error()
		return result
	}

	// 1. Parse all skills
	skillMap := make(map[string]model.SkillBehavior)
	hasLintErrors := false

	if entries, err := os.ReadDir(skillsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".skill.yaml") {
				continue
			}
			content, err := os.ReadFile(filepath.Join(skillsDir, entry.Name()))
			if err != nil {
				continue
			}
			skill, err := yamlloader.ParseSkillYAML(string(content))
			if err != nil {
				result.Error = fmt.Sprintf("Parse error in %s: %v", entry.Name(), err)
				return result
			}
			skillMap[skill.Skill] = skill
			lintResults := linter.LintSkill(skill)
			for _, lr := range lintResults {
				if lr.Severity == linter.SeverityError {
					hasLintErrors = true
				}
			}
		}
	}

	if hasLintErrors {
		result.Error = "Build failed: lint errors found. Fix errors before building."
		return result
	}

	// 2. Scan codebase if any skill consumes codebase_index or --enrich is set
	var codebaseCtx *scanner.CodebaseContext
	hasIndexConsumer := false
	for _, skill := range skillMap {
		if skillConsumesIndex(skill) {
			hasIndexConsumer = true
			break
		}
	}

	if hasIndexConsumer || enrichMode != scanner.EnrichNone {
		codebaseCtx, err = scanner.ScanCodebase(".")
		if err != nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Codebase scan failed (enrichment skipped): %v", err))
		}
	}

	// Write context files when index mode is needed
	writeContextFiles := enrichMode == scanner.EnrichIndex || (hasIndexConsumer && enrichMode != scanner.EnrichFull)
	if writeContextFiles && codebaseCtx != nil {
		contextDir := filepath.Join(outputDir, gen.ContextDir())
		if err := enricher.WriteContextFiles(codebaseCtx, contextDir); err != nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Failed to write context files: %v", err))
		}
	}

	// 3. Generate skills (enrich consumer skills or globally via --enrich)
	sg, hasSG := gen.(spec.SkillGenerator)
	if hasSG {
		for _, skill := range skillMap {
			md := sg.GenerateSkill(skill)
			if codebaseCtx != nil {
				switch {
				case enrichMode == scanner.EnrichFull:
					// --enrich=full overrides: inline into ALL skills
					md += enricher.RenderInline(codebaseCtx)
				case enrichMode == scanner.EnrichIndex:
					// --enrich=index overrides: pointer in ALL skills
					md += enricher.RenderPointer(codebaseCtx, gen.ContextDir())
				case skillConsumesIndex(skill):
					// Auto: only skills that consume codebase_index get the pointer
					md += enricher.RenderPointer(codebaseCtx, gen.ContextDir())
				}
			}
			wordCount := countWords(md)
			if wordCount > wordLimit {
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Skill %q generates %d words (limit: %d). Consider simplifying.", skill.Skill, wordCount, wordLimit))
			}

			relPath := sg.SkillPath(skill.Skill)
			fullPath := filepath.Join(outputDir, filepath.FromSlash(relPath))
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				result.Error = fmt.Sprintf("Failed to create directory for skill %q: %v", skill.Skill, err)
				return result
			}
			if err := os.WriteFile(fullPath, []byte(md), 0644); err != nil {
				result.Error = fmt.Sprintf("Failed to write skill %q: %v", skill.Skill, err)
				return result
			}
			result.SkillsGenerated++
		}
	}

	// 3. Generate agents
	var allAgents []model.AgentComposition
	ag, hasAG := gen.(spec.AgentGenerator)

	if entries, err := os.ReadDir(agentsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".agent.yaml") {
				continue
			}
			content, err := os.ReadFile(filepath.Join(agentsDir, entry.Name()))
			if err != nil {
				continue
			}
			agent, err := yamlloader.ParseAgentYAML(string(content))
			if err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Agent %s: %v", entry.Name(), err))
				continue
			}

			allAgents = append(allAgents, agent)

			if !hasAG {
				continue
			}

			var resolvedSkills []model.SkillBehavior
			for _, name := range agent.Skills {
				if s, ok := skillMap[name]; ok {
					resolvedSkills = append(resolvedSkills, s)
				}
			}
			if len(resolvedSkills) < len(agent.Skills) {
				var missing []string
				for _, name := range agent.Skills {
					if _, ok := skillMap[name]; !ok {
						missing = append(missing, name)
					}
				}
				result.Warnings = append(result.Warnings,
					fmt.Sprintf("Agent %q: unresolved skills [%s]. Tool list may be incomplete.", agent.Agent, strings.Join(missing, ", ")))
			}

			// Check ordering
			orderingIssues := analyzer.CheckSkillOrdering(agent, skillMap)
			for _, issue := range orderingIssues {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Agent %q: %s", agent.Agent, issue.Message))
			}

			md := ag.GenerateAgent(agent, resolvedSkills, outputDir)
			relPath := ag.AgentPath(agent.Agent)
			fullPath := filepath.Join(outputDir, filepath.FromSlash(relPath))
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				result.Error = fmt.Sprintf("Failed to create directory for agent %q: %v", agent.Agent, err)
				return result
			}
			if err := os.WriteFile(fullPath, []byte(md), 0644); err != nil {
				result.Error = fmt.Sprintf("Failed to write agent %q: %v", agent.Agent, err)
				return result
			}
			result.AgentsGenerated++
		}
	}

	// 4. Generate instructions (optional — only if generator implements InstructionsGenerator)
	if ig, ok := gen.(spec.InstructionsGenerator); ok {
		skills := make([]model.SkillBehavior, 0, len(skillMap))
		for _, s := range skillMap {
			skills = append(skills, s)
		}
		instructions := ig.GenerateInstructions(skills, allAgents)
		instrPath := ig.InstructionsPath()
		if instructions != "" {
			fullPath := filepath.Join(outputDir, filepath.FromSlash(instrPath))
			if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
				result.Error = fmt.Sprintf("Failed to create directory for instructions: %v", err)
				return result
			}
			if err := os.WriteFile(fullPath, []byte(instructions), 0644); err != nil {
				result.Error = fmt.Sprintf("Failed to write instructions: %v", err)
				return result
			}
		}
	}

	result.Success = true
	return result
}

func countWords(text string) int {
	return len(strings.Fields(text))
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
	var watchFlag bool

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

			result := RunBuild(skillsDir, agentsDir, outputDir, target, enrichMode)
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
	buildCmd.Flags().StringVar(&enrichFlag, "enrich", "", "enrich skills with codebase context (index|full)")
	buildCmd.Flag("enrich").NoOptDefVal = "index"

	rootCmd.AddCommand(buildCmd)
}
