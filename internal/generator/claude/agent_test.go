package claude_test

import (
	"strings"
	"testing"

	"github.com/mirandaguillaume/forgent/internal/generator/claude"
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

func TestGenerateAgentMd_Frontmatter(t *testing.T) {
	md := claude.GenerateAgentMd(testAgent(), testResolvedSkills(), ".claude", nil, "")
	assert.Contains(t, md, "---\nname: code-reviewer\n")
	assert.Contains(t, md, "description: Reviews code for quality and security issues")
}

func TestGenerateAgentMd_SequentialOrchestration(t *testing.T) {
	md := claude.GenerateAgentMd(testAgent(), testResolvedSkills(), ".claude", nil, "")
	assert.Contains(t, md, "sequentially as independent subagents")
}

func TestGenerateAgentMd_ParallelOrchestration(t *testing.T) {
	agent := testAgent()
	agent.Orchestration = model.OrchestrationParallel
	md := claude.GenerateAgentMd(agent, testResolvedSkills(), ".claude", nil, "")
	assert.Contains(t, md, "parallel subagents")
}

func TestGenerateAgentMd_AdaptiveOrchestration(t *testing.T) {
	agent := testAgent()
	agent.Orchestration = model.OrchestrationAdaptive
	md := claude.GenerateAgentMd(agent, testResolvedSkills(), ".claude", nil, "")
	assert.Contains(t, md, "subagents, choosing execution order")
}

func TestGenerateAgentMd_SkillReferences(t *testing.T) {
	md := claude.GenerateAgentMd(testAgent(), testResolvedSkills(), ".claude", nil, "")
	assert.Contains(t, md, "### Step 1: Code Review")
	assert.Contains(t, md, "### Step 2: Security Scan")
}

func TestGenerateAgentMd_SkillContextInfo(t *testing.T) {
	md := claude.GenerateAgentMd(testAgent(), testResolvedSkills(), ".claude", nil, "")
	assert.Contains(t, md, "- In: source-code")
	assert.Contains(t, md, "- Out: review-report")
	assert.Contains(t, md, "- Out: security-report")
}

func TestGenerateAgentMd_OutputSection(t *testing.T) {
	md := claude.GenerateAgentMd(testAgent(), testResolvedSkills(), ".claude", nil, "")
	assert.Contains(t, md, "## Output")
	assert.Contains(t, md, "review-report")
	assert.Contains(t, md, "security-report")
}

func TestGenerateAgentMd_NoDescription(t *testing.T) {
	agent := testAgent()
	agent.Description = ""
	md := claude.GenerateAgentMd(agent, testResolvedSkills(), ".claude", nil, "")
	assert.NotContains(t, md, "description:")
}

func TestGenerateAgentMd_NoSkills(t *testing.T) {
	agent := testAgent()
	md := claude.GenerateAgentMd(agent, nil, ".claude", nil, "")
	assert.NotContains(t, md, "tools:")
	assert.NotContains(t, md, "## Output")
}

func TestGenerateAgentMd_OrchestratorToolIsTask(t *testing.T) {
	md := claude.GenerateAgentMd(testAgent(), testResolvedSkills(), ".claude", nil, "")
	assert.Contains(t, md, "tools: Task")
	assert.NotContains(t, md, "Bash")
	assert.NotContains(t, md, "Grep")
}

func TestGenerateAgentMd_SubagentModel(t *testing.T) {
	md := claude.GenerateAgentMd(testAgent(), testResolvedSkills(), ".claude", nil, "")
	assert.Contains(t, md, "Model: sonnet")
	assert.Contains(t, md, "Model: haiku")
}

func TestGenerateAgentMd_SubagentSkillPath(t *testing.T) {
	md := claude.GenerateAgentMd(testAgent(), testResolvedSkills(), ".claude", nil, "")
	assert.Contains(t, md, "Skill: `.claude/skills/code-review/SKILL.md`")
	assert.Contains(t, md, "Skill: `.claude/skills/security-scan/SKILL.md`")
}

func TestGenerateAgentMd_LaunchSubagent(t *testing.T) {
	md := claude.GenerateAgentMd(testAgent(), testResolvedSkills(), ".claude", nil, "")
	assert.Contains(t, md, "Launch a subagent")
}

func TestResolveAgentTools(t *testing.T) {
	tools := claude.ResolveAgentTools(testResolvedSkills())
	// Should contain tools from both skills, merged and ordered
	assert.Contains(t, tools, "Grep")
	assert.Contains(t, tools, "Read")
	assert.Contains(t, tools, "Bash")
}

// --- Compact format tests ---

func TestGenerateCompactAgentMd_Frontmatter(t *testing.T) {
	md := claude.GenerateCompactAgentMd(testAgent(), testResolvedSkills())
	assert.Contains(t, md, "---\nname: code-reviewer\n")
	assert.Contains(t, md, "description: Reviews code for quality and security issues")
	assert.Contains(t, md, "tools: ")
}

func TestGenerateCompactAgentMd_NoStepHeaders(t *testing.T) {
	md := claude.GenerateCompactAgentMd(testAgent(), testResolvedSkills())
	assert.NotContains(t, md, "### Step")
	assert.NotContains(t, md, "Read `.claude/skills/")
	assert.NotContains(t, md, "follow its instructions")
}

func TestGenerateCompactAgentMd_InlinedSkills(t *testing.T) {
	md := claude.GenerateCompactAgentMd(testAgent(), testResolvedSkills())
	assert.Contains(t, md, "**code-review**")
	assert.Contains(t, md, "**security-scan**")
	assert.Contains(t, md, "FS: read-only")
	assert.Contains(t, md, "FS: read-write")
}

func TestGenerateCompactAgentMd_FewerWords(t *testing.T) {
	standard := claude.GenerateAgentMd(testAgent(), testResolvedSkills(), ".claude", nil, "")
	compact := claude.GenerateCompactAgentMd(testAgent(), testResolvedSkills())
	stdWords := len(strings.Fields(standard))
	cmpWords := len(strings.Fields(compact))
	assert.Less(t, cmpWords, stdWords, "compact should have fewer words")
}

func TestGenerateCompactAgentMd_OutputSection(t *testing.T) {
	md := claude.GenerateCompactAgentMd(testAgent(), testResolvedSkills())
	assert.Contains(t, md, "## Output")
	assert.Contains(t, md, "review-report")
	assert.Contains(t, md, "security-report")
}

func TestGenerateCompactAgentMd_Orchestration(t *testing.T) {
	md := claude.GenerateCompactAgentMd(testAgent(), testResolvedSkills())
	assert.Contains(t, md, "Execute 2 skills in order.")
}

func TestGenerateCompactAgentMd_NoSkills(t *testing.T) {
	agent := testAgent()
	md := claude.GenerateCompactAgentMd(agent, nil)
	assert.NotContains(t, md, "tools:")
	assert.NotContains(t, md, "## Output")
}
