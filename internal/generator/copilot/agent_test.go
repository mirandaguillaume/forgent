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
				Effort:   model.EffortMedium,
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
				Effort:   model.EffortLight,
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


func TestGenerateCopilotAgentMd_SequentialOrchestration(t *testing.T) {
	md := copilot.GenerateCopilotAgentMd(testAgent(), testResolvedSkills(), ".github")
	assert.Contains(t, md, "Execute 2 skills sequentially as independent subagents")
}

func TestGenerateCopilotAgentMd_ParallelOrchestration(t *testing.T) {
	agent := testAgent()
	agent.Orchestration = model.OrchestrationParallel
	md := copilot.GenerateCopilotAgentMd(agent, testResolvedSkills(), ".github")
	assert.Contains(t, md, "Launch 2 skills as parallel subagents")
}

func TestGenerateCopilotAgentMd_AdaptiveOrchestration(t *testing.T) {
	agent := testAgent()
	agent.Orchestration = model.OrchestrationAdaptive
	md := copilot.GenerateCopilotAgentMd(agent, testResolvedSkills(), ".github")
	assert.Contains(t, md, "Dispatch 2 skills as subagents, choosing execution order dynamically")
}

func TestGenerateCopilotAgentMd_SkillContextInfo(t *testing.T) {
	md := copilot.GenerateCopilotAgentMd(testAgent(), testResolvedSkills(), ".github")
	assert.Contains(t, md, "- In: source-code")
	assert.Contains(t, md, "- Out: review-report")
	assert.Contains(t, md, "- Out: security-report")
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

func TestGenerateCopilotAgentMd_SubagentFormat(t *testing.T) {
	md := copilot.GenerateCopilotAgentMd(testAgent(), testResolvedSkills(), ".github")
	assert.Contains(t, md, "Launch a subagent")
	assert.Contains(t, md, "Model: sonnet")
	assert.Contains(t, md, "Model: haiku")
}

func TestGenerateCopilotAgentMd_SkillPath(t *testing.T) {
	md := copilot.GenerateCopilotAgentMd(testAgent(), testResolvedSkills(), ".github")
	assert.Contains(t, md, "Skill: `.github/skills/code-review/SKILL.md`")
}

func TestGenerateCopilotAgentMd_OrchestratorTool(t *testing.T) {
	md := copilot.GenerateCopilotAgentMd(testAgent(), testResolvedSkills(), ".github")
	assert.Contains(t, md, `tools: ["task"]`)
}

// --- Compact format tests ---

func TestGenerateCompactCopilotAgentMd_Frontmatter(t *testing.T) {
	md := copilot.GenerateCompactCopilotAgentMd(testAgent(), testResolvedSkills())
	assert.Contains(t, md, "---\nname: code-reviewer\n")
	assert.Contains(t, md, "description: Reviews code for quality and security issues")
	assert.Contains(t, md, "tools: [")
}

func TestGenerateCompactCopilotAgentMd_NoStepHeaders(t *testing.T) {
	md := copilot.GenerateCompactCopilotAgentMd(testAgent(), testResolvedSkills())
	assert.NotContains(t, md, "### Step")
	assert.NotContains(t, md, "Read `.github/skills/")
	assert.NotContains(t, md, "follow its instructions")
}

func TestGenerateCompactCopilotAgentMd_InlinedSkills(t *testing.T) {
	md := copilot.GenerateCompactCopilotAgentMd(testAgent(), testResolvedSkills())
	assert.Contains(t, md, "**code-review**")
	assert.Contains(t, md, "**security-scan**")
	assert.Contains(t, md, "FS: read-only")
	assert.Contains(t, md, "FS: read-write")
}

func TestGenerateCompactCopilotAgentMd_FewerWords(t *testing.T) {
	standard := copilot.GenerateCopilotAgentMd(testAgent(), testResolvedSkills(), ".github")
	compact := copilot.GenerateCompactCopilotAgentMd(testAgent(), testResolvedSkills())
	stdWords := len(strings.Fields(standard))
	cmpWords := len(strings.Fields(compact))
	assert.Less(t, cmpWords, stdWords, "compact should have fewer words")
}

func TestGenerateCompactCopilotAgentMd_OutputSection(t *testing.T) {
	md := copilot.GenerateCompactCopilotAgentMd(testAgent(), testResolvedSkills())
	assert.Contains(t, md, "## Output")
	assert.Contains(t, md, "review-report")
	assert.Contains(t, md, "security-report")
}
