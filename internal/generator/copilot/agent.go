package copilot

import (
	"fmt"
	"strings"

	"github.com/mirandaguillaume/forgent/internal/generator"
	"github.com/mirandaguillaume/forgent/pkg/model"
)

// ResolveCopilotAgentTools collects and merges Copilot tools from all skills.
func ResolveCopilotAgentTools(skills []model.SkillBehavior) []string {
	var allLists [][]string
	for _, skill := range skills {
		allLists = append(allLists, MapToolsToCopilot(skill.Strategy.Tools))
		allLists = append(allLists, InferCopilotToolsFromSecurity(
			string(skill.Security.Filesystem),
			string(skill.Security.Network),
		))
	}
	return MergeCopilotToolLists(allLists...)
}

// GenerateCopilotAgentMd generates a Copilot .agent.md from an AgentComposition.
func GenerateCopilotAgentMd(agent model.AgentComposition, resolvedSkills []model.SkillBehavior, outputDir string) string {
	var lines []string

	// Frontmatter
	lines = append(lines, "---")
	lines = append(lines, "name: "+agent.Agent)
	if agent.Description != "" {
		lines = append(lines, "description: "+agent.Description)
	}

	if len(resolvedSkills) > 0 {
		tools := ResolveCopilotAgentTools(resolvedSkills)
		hasRead := false
		for _, t := range tools {
			if t == "read" {
				hasRead = true
				break
			}
		}
		if !hasRead {
			tools = append([]string{"read"}, tools...)
		}
		// Format as YAML array: tools: ["read", "search"]
		quoted := make([]string, len(tools))
		for i, t := range tools {
			quoted[i] = fmt.Sprintf("%q", t)
		}
		lines = append(lines, fmt.Sprintf("tools: [%s]", strings.Join(quoted, ", ")))
	}

	lines = append(lines, "---")
	lines = append(lines, "")

	// Body
	lines = append(lines, fmt.Sprintf("You are %s. %s", generator.ToTitle(agent.Agent), agent.Description))
	lines = append(lines, "")

	// Orchestration
	lines = append(lines, "## Execution")
	n := len(agent.Skills)
	switch agent.Orchestration {
	case model.OrchestrationSequential:
		lines = append(lines, fmt.Sprintf(
			"Execute %d skills in order. Read each skill file, follow its instructions, then pass the output to the next skill.", n))
	case model.OrchestrationParallel:
		lines = append(lines, fmt.Sprintf(
			"Execute %d skills concurrently. Read each skill file and follow its instructions independently.", n))
	case model.OrchestrationParallelThenMerge:
		lines = append(lines, fmt.Sprintf(
			"Execute %d skills concurrently, then merge their outputs. Read each skill file and follow its instructions.", n))
	case model.OrchestrationAdaptive:
		lines = append(lines, fmt.Sprintf(
			"Choose execution order dynamically for %d skills. Read each skill file and follow its instructions based on intermediate results.", n))
	}
	lines = append(lines, "")

	// Skill references
	for i, skillName := range agent.Skills {
		skillPath := fmt.Sprintf("%s/skills/%s/SKILL.md", outputDir, skillName)
		lines = append(lines, fmt.Sprintf("### Step %d: %s", i+1, generator.ToTitle(skillName)))
		lines = append(lines, fmt.Sprintf("Read `%s` and follow its instructions.", skillPath))

		for _, s := range resolvedSkills {
			if s.Skill == skillName {
				var parts []string
				if len(s.Context.Consumes) > 0 {
					parts = append(parts, "Consumes: "+strings.Join(s.Context.Consumes, ", "))
				}
				if len(s.Context.Produces) > 0 {
					parts = append(parts, "Produces: "+strings.Join(s.Context.Produces, ", "))
				}
				if len(parts) > 0 {
					lines = append(lines, strings.Join(parts, " → "))
				}
				break
			}
		}
		lines = append(lines, "")
	}

	// Output
	if len(resolvedSkills) > 0 {
		seen := map[string]bool{}
		var unique []string
		for _, s := range resolvedSkills {
			for _, p := range s.Context.Produces {
				if !seen[p] {
					unique = append(unique, p)
					seen[p] = true
				}
			}
		}
		if len(unique) > 0 {
			lines = append(lines, "## Output")
			lines = append(lines, fmt.Sprintf("Produce a structured report containing: %s.", strings.Join(unique, ", ")))
			lines = append(lines, "")
		}
	}

	return strings.Join(lines, "\n")
}
