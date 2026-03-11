package generator

import (
	"fmt"
	"strings"

	"github.com/mirandaguillaume/forgent/pkg/model"
)

// ToTitle converts "my-skill-name" to "My Skill Name".
func ToTitle(slug string) string {
	words := strings.Split(slug, "-")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}

// CountWords counts whitespace-separated words in text.
func CountWords(text string) int {
	return len(strings.Fields(text))
}

// FormatGuardrail formats a guardrail rule as markdown list item(s).
func FormatGuardrail(g model.GuardrailRule) string {
	if s, ok := g.StringValue(); ok {
		return "- " + s
	}
	if m, ok := g.MapValue(); ok {
		var lines []string
		for k, v := range m {
			lines = append(lines, fmt.Sprintf("- %s: %v", k, v))
		}
		return strings.Join(lines, "\n")
	}
	return ""
}

// BuildSkillDescription creates a description from skill facets.
func BuildSkillDescription(skill model.SkillBehavior) string {
	parts := []string{skill.Strategy.Approach + "-based skill"}
	if len(skill.Context.Consumes) > 0 {
		parts = append(parts, "consuming "+strings.Join(skill.Context.Consumes, ", "))
	}
	if len(skill.Context.Produces) > 0 {
		parts = append(parts, "to produce "+strings.Join(skill.Context.Produces, ", "))
	}
	return strings.Join(parts, " ")
}
