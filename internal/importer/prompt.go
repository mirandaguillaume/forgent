package importer

import (
	"fmt"
	"strings"
)

// BuildImportPrompt constructs the LLM prompt for agent-to-skill decomposition.
func BuildImportPrompt(source Source, fm AgentFrontmatter, body string, genericTools []string) string {
	var b strings.Builder

	b.WriteString("You are converting an agent definition into Forgent skill YAML specs.\n\n")

	// Schema reference
	b.WriteString("## Forgent skill YAML schema\n\n")
	b.WriteString("Each skill MUST have these fields:\n")
	b.WriteString("- skill: string (kebab-case name)\n")
	b.WriteString("- version: string (semver)\n")
	b.WriteString("- context: { consumes: [string], produces: [string], memory: short-term|conversation|long-term }\n")
	b.WriteString("- strategy: { tools: [string], approach: string, steps: [string] }\n")
	b.WriteString("- guardrails: array of rules (strings or maps with key/value)\n")
	b.WriteString("- observability: { trace_level: minimal|standard|detailed, metrics: [string] }\n")
	b.WriteString("- security: { filesystem: none|read-only|read-write|full, network: none|allowlist|full, secrets: [string] }\n")
	b.WriteString("- negotiation: { file_conflicts: yield|override|merge, priority: int }\n\n")

	// Tool names
	b.WriteString("## Available generic tool names\n")
	b.WriteString("read_file, write_file, edit_file, grep, search, bash, web_fetch, web_search, todo, task\n\n")

	// Agent composition schema
	b.WriteString("## Agent composition schema (optional, only if input has multiple responsibilities)\n")
	b.WriteString("- agent: string (kebab-case name)\n")
	b.WriteString("- skills: [string] (skill names)\n")
	b.WriteString("- orchestration: sequential|parallel|parallel-then-merge|adaptive\n")
	b.WriteString("- description: string\n")
	b.WriteString("- consumes: [string]\n")
	b.WriteString("- produces: [string]\n\n")

	// Input
	b.WriteString("## Input to analyze\n\n")
	if fm.Name != "" {
		b.WriteString(fmt.Sprintf("Source file: %s\n", source.Name))
		b.WriteString(fmt.Sprintf("Name: %s\n", fm.Name))
		b.WriteString(fmt.Sprintf("Description: %s\n", fm.Description))
		if len(genericTools) > 0 {
			b.WriteString(fmt.Sprintf("Tools (generic): %s\n", strings.Join(genericTools, ", ")))
		}
		b.WriteString("\n")
	}
	b.WriteString("### Full agent definition\n\n")
	b.WriteString(body)
	b.WriteString("\n\n")

	// Instructions
	b.WriteString("## Instructions\n\n")
	b.WriteString("1. Analyze this agent definition and identify distinct responsibilities.\n")
	b.WriteString("2. Create one Forgent skill YAML per responsibility.\n")
	b.WriteString("   Skill decomposition rules (CRITICAL — the build will REJECT skills that violate these):\n")
	b.WriteString("   - Each skill must have exactly ONE output in 'produces'\n")
	b.WriteString("   - Each step must describe ONE action (no 'and', 'then', or '&' joining different actions)\n")
	b.WriteString("   - The 'approach' field must describe a single focused activity (no conjunctions)\n")
	b.WriteString("   - If steps span different phases (e.g., input parsing, searching, analyzing),\n")
	b.WriteString("     split into separate skills connected by data flow (one skill's 'produces' feeds the next's 'consumes')\n")
	b.WriteString("   - Maximum 5 steps per skill — if you need more, the skill is too broad\n")
	b.WriteString("   - All skill fields (approach, steps, guardrails) MUST be written in English\n")
	b.WriteString("3. If the agent has multiple responsibilities, also create an agent composition.\n")
	b.WriteString("4. If the agent has a single responsibility, create only one skill (no agent).\n")
	b.WriteString("5. Set security to the minimum required permissions.\n")
	b.WriteString("6. Add meaningful guardrails (especially timeout for long-running tasks).\n")
	b.WriteString("7. Extract output format templates: if the agent defines a structured output format\n")
	b.WriteString("   (e.g., review comment format, risk score format, report structure), extract each\n")
	b.WriteString("   as a contract keyed by the matching 'produces' name. Only extract when the agent\n")
	b.WriteString("   defines an explicit, reusable output format — not for generic outputs.\n\n")

	// Output format
	b.WriteString("## Output format\n\n")
	b.WriteString("Return ONLY valid JSON (no markdown fences, no explanation):\n")
	b.WriteString("```\n")
	b.WriteString(`{"skills": [{"yaml": "skill: name\nversion: ..."}], "agent": {"yaml": "agent: name\n..."} or null, "contracts": {"produce_name": "Output format template markdown..."} or null}`)
	b.WriteString("\n```\n")

	return b.String()
}

// BuildRetryPrompt appends validation feedback to the original prompt for a retry.
func BuildRetryPrompt(originalPrompt, originalResponse string, feedback []string) string {
	var b strings.Builder

	b.WriteString(originalPrompt)
	b.WriteString("\n\n## Previous attempt\n\n")
	b.WriteString("Your previous response:\n")
	b.WriteString(originalResponse)
	b.WriteString("\n\n## Validation feedback — Fix these issues\n\n")
	for _, f := range feedback {
		b.WriteString("- " + f + "\n")
	}
	b.WriteString("\nReturn the corrected JSON.\n")

	return b.String()
}
