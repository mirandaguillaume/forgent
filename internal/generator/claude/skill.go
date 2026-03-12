package claude

import (
	"fmt"
	"strings"

	"github.com/mirandaguillaume/forgent/internal/generator"
	"github.com/mirandaguillaume/forgent/pkg/model"
)

// GenerateSkillMd generates a Claude Code SKILL.md from a SkillBehavior.
func GenerateSkillMd(skill model.SkillBehavior) string {
	var lines []string

	// Frontmatter
	desc := generator.BuildSkillDescription(skill)
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

	// When to Use (after guardrails, before context)
	if wtu := generator.FormatWhenToUse(skill.WhenToUse); wtu != "" {
		lines = append(lines, wtu)
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

	// Examples (after strategy)
	if exs := generator.FormatExamples(skill.Examples); exs != "" {
		lines = append(lines, exs)
	}

	// Anti-patterns / Red Flags (before security)
	if aps := generator.FormatAntiPatterns(skill.AntiPatterns); aps != "" {
		lines = append(lines, aps)
	}

	// Security LAST (recency bias)
	lines = append(lines, "## Security")
	lines = append(lines, "- Filesystem: "+string(skill.Security.Filesystem))
	lines = append(lines, "- Network: "+string(skill.Security.Network))
	if len(skill.Security.Secrets) > 0 {
		lines = append(lines, "- Secrets: "+strings.Join(skill.Security.Secrets, ", "))
	}
	if skill.Security.Sandbox != "" {
		lines = append(lines, "- Sandbox: "+string(skill.Security.Sandbox))
	}
	lines = append(lines, "")

	return strings.Join(lines, "\n")
}
