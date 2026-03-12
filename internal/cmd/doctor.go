package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/mirandaguillaume/forgent/internal/analyzer"
	"github.com/mirandaguillaume/forgent/internal/linter"
	yamlloader "github.com/mirandaguillaume/forgent/internal/yaml"
	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/spf13/cobra"
)

// DoctorReport holds the results of a full diagnostic run.
type DoctorReport struct {
	Skills           []model.SkillBehavior
	ParseErrors      []ParseError
	LintIssues       map[string][]linter.LintResult
	DependencyIssues []analyzer.DependencyIssue
	LoopRisks        map[string][]analyzer.LoopRisk
	OrderingIssues   []analyzer.OrderingIssue
	Score            int
}

// ParseError represents a skill file that failed to parse.
type ParseError struct {
	File  string
	Error string
}

// RunDoctor performs a full diagnostic on all skills and agents.
func RunDoctor(skillsDir string, agentsDir string) DoctorReport {
	report := DoctorReport{
		LintIssues: make(map[string][]linter.LintResult),
		LoopRisks:  make(map[string][]analyzer.LoopRisk),
	}

	// Parse all skills
	entries, _ := os.ReadDir(skillsDir)

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
			report.ParseErrors = append(report.ParseErrors, ParseError{
				File:  entry.Name(),
				Error: err.Error(),
			})
			continue
		}
		report.Skills = append(report.Skills, skill)
	}

	// Lint each skill
	for _, skill := range report.Skills {
		issues := linter.LintSkill(skill)
		if len(issues) > 0 {
			report.LintIssues[skill.Skill] = issues
		}
	}

	// Check dependencies across all skills
	report.DependencyIssues = analyzer.CheckDependencies(report.Skills)

	// Detect loop risks per skill
	checker := &analyzer.DefaultGuardrailChecker{}
	for _, skill := range report.Skills {
		risks := analyzer.DetectLoopRisks(skill, checker)
		if len(risks) > 0 {
			report.LoopRisks[skill.Skill] = risks
		}
	}

	// Check skill ordering in agents
	if agentsDir != "" {
		agentEntries, err := os.ReadDir(agentsDir)
		if err == nil {
			skillMap := make(map[string]model.SkillBehavior)
			for _, s := range report.Skills {
				skillMap[s.Skill] = s
			}
			for _, entry := range agentEntries {
				if !strings.HasSuffix(entry.Name(), ".agent.yaml") {
					continue
				}
				content, err := os.ReadFile(filepath.Join(agentsDir, entry.Name()))
				if err != nil {
					continue
				}
				agent, err := yamlloader.ParseAgentYAML(string(content))
				if err != nil {
					continue
				}
				report.OrderingIssues = append(report.OrderingIssues,
					analyzer.CheckSkillOrdering(agent, skillMap)...)
			}
		}
	}

	// Calculate health score
	totalIssues := len(report.ParseErrors) + len(report.DependencyIssues) + len(report.OrderingIssues)
	for _, issues := range report.LintIssues {
		for _, i := range issues {
			if i.Severity == linter.SeverityError {
				totalIssues++
			}
		}
	}
	for _, risks := range report.LoopRisks {
		for _, r := range risks {
			if r.Severity == "error" {
				totalIssues++
			}
		}
	}
	fileCount := len(report.Skills) + len(report.ParseErrors)
	maxScore := fileCount * 10
	if maxScore < 100 {
		maxScore = 100
	}
	report.Score = max(0, 100-(totalIssues*100/maxScore))

	return report
}

// PrintDoctorReport prints the doctor report to stdout with colored output.
func PrintDoctorReport(report DoctorReport) {
	bold := color.New(color.Bold)
	fmt.Println()
	bold.Println("=== Forgent Doctor Report ===")
	fmt.Println()
	fmt.Printf("Skills found: %d\n", len(report.Skills))

	if len(report.ParseErrors) > 0 {
		fmt.Println(color.RedString("\nParse Errors (%d):", len(report.ParseErrors)))
		for _, err := range report.ParseErrors {
			fmt.Printf("  %s %s: %s\n", color.RedString("x"), err.File, err.Error)
		}
	}

	if len(report.LintIssues) > 0 {
		fmt.Println(color.YellowString("\nLint Issues:"))
		for skill, issues := range report.LintIssues {
			for _, issue := range issues {
				icon := color.YellowString("!")
				if issue.Severity == linter.SeverityError {
					icon = color.RedString("x")
				}
				fmt.Printf("  %s %s: %s\n", icon, skill, issue.Message)
			}
		}
	}

	if len(report.DependencyIssues) > 0 {
		fmt.Println(color.RedString("\nDependency Issues (%d):", len(report.DependencyIssues)))
		for _, issue := range report.DependencyIssues {
			fmt.Printf("  %s %s: %s\n", color.RedString("x"), issue.Skill, issue.Message)
		}
	}

	if len(report.LoopRisks) > 0 {
		fmt.Println(color.YellowString("\nLoop Risks:"))
		for skill, risks := range report.LoopRisks {
			for _, risk := range risks {
				icon := color.YellowString("!")
				if risk.Severity == "error" {
					icon = color.RedString("x")
				}
				fmt.Printf("  %s %s: %s\n", icon, skill, risk.Message)
			}
		}
	}

	if len(report.OrderingIssues) > 0 {
		fmt.Println(color.YellowString("\nSkill Ordering Issues (%d):", len(report.OrderingIssues)))
		for _, issue := range report.OrderingIssues {
			fmt.Printf("  %s %s: %s\n", color.YellowString("!"), issue.Agent, issue.Message)
		}
	}

	scoreColor := color.GreenString
	if report.Score < 80 {
		scoreColor = color.YellowString
	}
	if report.Score < 50 {
		scoreColor = color.RedString
	}
	fmt.Printf("\nHealth Score: %s\n\n", scoreColor("%d/100", report.Score))
}

func init() {
	var agentsDir string
	doctorCmd := &cobra.Command{
		Use:   "doctor [path]",
		Short: "Run full diagnostic on all skills and agents",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			path := "skills"
			if len(args) > 0 {
				path = args[0]
			}
			report := RunDoctor(path, agentsDir)
			PrintDoctorReport(report)
			if report.Score < 50 {
				os.Exit(1)
			}
		},
	}
	doctorCmd.Flags().StringVarP(&agentsDir, "agents", "a", "agents", "agents directory")
	rootCmd.AddCommand(doctorCmd)
}
