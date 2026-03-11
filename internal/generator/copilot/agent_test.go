package copilot_test

import (
	"strings"
	"testing"

	"github.com/mirandaguillaume/forgent/internal/generator/copilot"
	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/stretchr/testify/assert"
)

func testAgent() model.AgentComposition {
	return model.AgentComposition{
		Agent:         "code-reviewer",
		Description:   "Reviews code for quality and security issues",
		Skills:        []string{"code-review", "security-scan"},
		Orchestration: model.OrchestrationSequential,
	}
}

func testResolvedSkills() []model.SkillBehavior {
	return []model.SkillBehavior{
		{
			Skill: "code-review",
			Context: model.ContextFacet{
				Consumes: []string{"source-code"},
				Produces: []string{"review-report"},
				Memory:   model.MemoryConversation,
			},
			Strategy: model.StrategyFacet{
				Approach: "analytical",
				Tools:    []string{"read", "grep"},
			},
			Security: model.SecurityFacet{
				Filesystem: model.AccessReadOnly,
				Network:    model.NetworkNone,
			},
		},
		{
			Skill: "security-scan",
			Context: model.ContextFacet{
				Consumes: []string{"source-code"},
				Produces: []string{"security-report"},
				Memory:   model.MemoryShortTerm,
			},
			Strategy: model.StrategyFacet{
				Approach: "scanning",
				Tools:    []string{"bash", "grep"},
			},
			Security: model.SecurityFacet{
				Filesystem: model.AccessReadWrite,
				Network:    model.NetworkFull,
			},
		},
	}
}

func TestGenerateCopilotAgentMd_Frontmatter(t *testing.T) {
	md := copilot.GenerateCopilotAgentMd(testAgent(), testResolvedSkills(), ".github")
	assert.Contains(t, md, "---\nname: code-reviewer\n")
	assert.Contains(t, md, "description: Reviews code for quality and security issues")
}

func TestGenerateCopilotAgentMd_ToolsAsYAMLArray(t *testing.T) {
	md := copilot.GenerateCopilotAgentMd(testAgent(), testResolvedSkills(), ".github")
	// Tools should be formatted as YAML array: tools: ["read", "search", ...]
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(line, "tools: ") {
			assert.True(t, strings.HasPrefix(line, "tools: ["), "tools should be YAML array format")
			assert.True(t, strings.HasSuffix(line, "]"), "tools should end with ]")
			// Should contain quoted lowercase tool names
			assert.Contains(t, line, `"read"`)
			assert.Contains(t, line, `"search"`)
			break
		}
	}
}

func TestGenerateCopilotAgentMd_ToolsAreLowercase(t *testing.T) {
	md := copilot.GenerateCopilotAgentMd(testAgent(), testResolvedSkills(), ".github")
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(line, "tools: ") {
			// Should NOT contain uppercase Claude-style tool names
			assert.NotContains(t, line, "Read")
			assert.NotContains(t, line, "Bash")
			assert.NotContains(t, line, "Grep")
			assert.NotContains(t, line, "WebFetch")
			break
		}
	}
}

func TestGenerateCopilotAgentMd_ReadAlwaysPresent(t *testing.T) {
	md := copilot.GenerateCopilotAgentMd(testAgent(), testResolvedSkills(), ".github")
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(line, "tools: ") {
			assert.Contains(t, line, `"read"`)
			break
		}
	}
}

func TestGenerateCopilotAgentMd_SequentialOrchestration(t *testing.T) {
	md := copilot.GenerateCopilotAgentMd(testAgent(), testResolvedSkills(), ".github")
	assert.Contains(t, md, "Execute 2 skills in order")
}

func TestGenerateCopilotAgentMd_ParallelOrchestration(t *testing.T) {
	agent := testAgent()
	agent.Orchestration = model.OrchestrationParallel
	md := copilot.GenerateCopilotAgentMd(agent, testResolvedSkills(), ".github")
	assert.Contains(t, md, "Execute 2 skills concurrently")
}

func TestGenerateCopilotAgentMd_AdaptiveOrchestration(t *testing.T) {
	agent := testAgent()
	agent.Orchestration = model.OrchestrationAdaptive
	md := copilot.GenerateCopilotAgentMd(agent, testResolvedSkills(), ".github")
	assert.Contains(t, md, "Choose execution order dynamically for 2 skills")
}

func TestGenerateCopilotAgentMd_SkillReferences(t *testing.T) {
	md := copilot.GenerateCopilotAgentMd(testAgent(), testResolvedSkills(), ".github")
	assert.Contains(t, md, "### Step 1: Code Review")
	assert.Contains(t, md, "Read `.github/skills/code-review/SKILL.md` and follow its instructions.")
	assert.Contains(t, md, "### Step 2: Security Scan")
	assert.Contains(t, md, "Read `.github/skills/security-scan/SKILL.md` and follow its instructions.")
}

func TestGenerateCopilotAgentMd_SkillContextInfo(t *testing.T) {
	md := copilot.GenerateCopilotAgentMd(testAgent(), testResolvedSkills(), ".github")
	assert.Contains(t, md, "Consumes: source-code")
	assert.Contains(t, md, "Produces: review-report")
	assert.Contains(t, md, "Produces: security-report")
}

func TestGenerateCopilotAgentMd_OutputSection(t *testing.T) {
	md := copilot.GenerateCopilotAgentMd(testAgent(), testResolvedSkills(), ".github")
	assert.Contains(t, md, "## Output")
	assert.Contains(t, md, "review-report")
	assert.Contains(t, md, "security-report")
}

func TestGenerateCopilotAgentMd_NoDescription(t *testing.T) {
	agent := testAgent()
	agent.Description = ""
	md := copilot.GenerateCopilotAgentMd(agent, testResolvedSkills(), ".github")
	assert.NotContains(t, md, "description:")
}

func TestGenerateCopilotAgentMd_NoSkills(t *testing.T) {
	agent := testAgent()
	md := copilot.GenerateCopilotAgentMd(agent, nil, ".github")
	assert.NotContains(t, md, "tools:")
	assert.NotContains(t, md, "## Output")
}

func TestResolveCopilotAgentTools(t *testing.T) {
	tools := copilot.ResolveCopilotAgentTools(testResolvedSkills())
	// Should contain tools from both skills, merged and ordered
	assert.Contains(t, tools, "search")
	assert.Contains(t, tools, "read")
	assert.Contains(t, tools, "execute")
}
