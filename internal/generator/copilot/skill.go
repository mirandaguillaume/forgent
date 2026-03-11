package copilot

import (
	"fmt"
	"strings"

	"github.com/mirandaguillaume/forgent/internal/generator"
	"github.com/mirandaguillaume/forgent/pkg/model"
)

// buildDescription creates a description truncated to 1024 chars for Copilot.
func buildDescription(skill model.SkillBehavior) string {
	desc := generator.BuildSkillDescription(skill)
	if len(desc) > 1024 {
		return desc[:1021] + "..."
	}
	return desc
}

// GenerateCopilotSkillMd generates a Copilot SKILL.md from a SkillBehavior.
func GenerateCopilotSkillMd(skill model.SkillBehavior) string {
	var lines []string

	// Frontmatter
	desc := buildDescription(skill)
	lines = append(lines, "---")
	lines = append(lines, "name: "+skill.Skill)
	lines = append(lines, "description: "+desc)
	lines = append(lines, "---")
	lines = append(lines, "")

	// Title
	lines = append(lines, "# "+generator.ToTitle(skill.Skill))
	lines = append(lines, "")

	// Guardrails FIRST (primacy bias)
	if len(skill.Guardrails) > 0 {
		lines = append(lines, "## Guardrails")
		for _, g := range skill.Guardrails {
			lines = append(lines, generator.FormatGuardrail(g))
		}
		lines = append(lines, "")
	}

	// Context
	lines = append(lines, "## Context")
	if len(skill.Context.Consumes) > 0 {
		lines = append(lines, "Consumes: "+strings.Join(skill.Context.Consumes, ", "))
	}
	if len(skill.Context.Produces) > 0 {
		lines = append(lines, "Produces: "+strings.Join(skill.Context.Produces, ", "))
	}
	lines = append(lines, "Memory: "+string(skill.Context.Memory))
	lines = append(lines, "")

	// Dependencies
	if len(skill.DependsOn) > 0 {
		lines = append(lines, "## Dependencies")
		for _, dep := range skill.DependsOn {
			lines = append(lines, fmt.Sprintf("- **%s** provides `%s`", dep.Skill, dep.Provides))
		}
		lines = append(lines, "")
	}

	// Strategy
	lines = append(lines, "## Strategy")
	lines = append(lines, "Approach: "+skill.Strategy.Approach)
	if len(skill.Strategy.Tools) > 0 {
		lines = append(lines, "Tools: "+strings.Join(skill.Strategy.Tools, ", "))
	}
	if len(skill.Strategy.Steps) > 0 {
		lines = append(lines, "")
		lines = append(lines, "### Steps")
		for i, step := range skill.Strategy.Steps {
			lines = append(lines, fmt.Sprintf("%d. %s", i+1, step))
		}
	}
	lines = append(lines, "")

	// Security LAST (recency bias)
	lines = append(lines, "## Security")
	lines = append(lines, "- Filesystem: "+string(skill.Security.Filesystem))
	lines = append(lines, "- Network: "+string(skill.Security.Network))
	if len(skill.Security.Secrets) > 0 {
		lines = append(lines, "- Secrets: "+strings.Join(skill.Security.Secrets, ", "))
	}
	if skill.Security.Sandbox != "" {
		lines = append(lines, "- Sandbox: "+skill.Security.Sandbox)
	}
	lines = append(lines, "")

	return strings.Join(lines, "\n")
}
