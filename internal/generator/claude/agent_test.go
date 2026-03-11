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

func TestGenerateAgentMd_Frontmatter(t *testing.T) {
	md := claude.GenerateAgentMd(testAgent(), testResolvedSkills(), ".claude")
	assert.Contains(t, md, "---\nname: code-reviewer\n")
	assert.Contains(t, md, "description: Reviews code for quality and security issues")
}

func TestGenerateAgentMd_ToolsResolvedAndMerged(t *testing.T) {
	md := claude.GenerateAgentMd(testAgent(), testResolvedSkills(), ".claude")
	// Should have tools line with merged tools from both skills
	assert.Contains(t, md, "tools: ")
	// Read should always be present
	assert.Contains(t, md, "Read")
	// Bash from security-scan
	assert.Contains(t, md, "Bash")
	// WebFetch/WebSearch from network full
	assert.Contains(t, md, "WebFetch")
	assert.Contains(t, md, "WebSearch")
}

func TestGenerateAgentMd_ReadAlwaysFirst(t *testing.T) {
	md := claude.GenerateAgentMd(testAgent(), testResolvedSkills(), ".claude")
	// Find the tools line
	for _, line := range strings.Split(md, "\n") {
		if strings.HasPrefix(line, "tools: ") {
			tools := strings.TrimPrefix(line, "tools: ")
			parts := strings.Split(tools, ", ")
			// Read should be in the list (in canonical order it comes after Glob and Grep)
			found := false
			for _, p := range parts {
				if p == "Read" {
					found = true
					break
				}
			}
			assert.True(t, found, "Read should be in tool list")
			break
		}
	}
}

func TestGenerateAgentMd_SequentialOrchestration(t *testing.T) {
	md := claude.GenerateAgentMd(testAgent(), testResolvedSkills(), ".claude")
	assert.Contains(t, md, "Execute 2 skills in order")
}

func TestGenerateAgentMd_ParallelOrchestration(t *testing.T) {
	agent := testAgent()
	agent.Orchestration = model.OrchestrationParallel
	md := claude.GenerateAgentMd(agent, testResolvedSkills(), ".claude")
	assert.Contains(t, md, "Execute 2 skills concurrently")
}

func TestGenerateAgentMd_AdaptiveOrchestration(t *testing.T) {
	agent := testAgent()
	agent.Orchestration = model.OrchestrationAdaptive
	md := claude.GenerateAgentMd(agent, testResolvedSkills(), ".claude")
	assert.Contains(t, md, "Choose execution order dynamically for 2 skills")
}

func TestGenerateAgentMd_SkillReferences(t *testing.T) {
	md := claude.GenerateAgentMd(testAgent(), testResolvedSkills(), ".claude")
	assert.Contains(t, md, "### Step 1: Code Review")
	assert.Contains(t, md, "Read `.claude/skills/code-review/SKILL.md` and follow its instructions.")
	assert.Contains(t, md, "### Step 2: Security Scan")
	assert.Contains(t, md, "Read `.claude/skills/security-scan/SKILL.md` and follow its instructions.")
}

func TestGenerateAgentMd_SkillContextInfo(t *testing.T) {
	md := claude.GenerateAgentMd(testAgent(), testResolvedSkills(), ".claude")
	assert.Contains(t, md, "Consumes: source-code")
	assert.Contains(t, md, "Produces: review-report")
	assert.Contains(t, md, "Produces: security-report")
}

func TestGenerateAgentMd_OutputSection(t *testing.T) {
	md := claude.GenerateAgentMd(testAgent(), testResolvedSkills(), ".claude")
	assert.Contains(t, md, "## Output")
	assert.Contains(t, md, "review-report")
	assert.Contains(t, md, "security-report")
}

func TestGenerateAgentMd_NoDescription(t *testing.T) {
	agent := testAgent()
	agent.Description = ""
	md := claude.GenerateAgentMd(agent, testResolvedSkills(), ".claude")
	assert.NotContains(t, md, "description:")
}

func TestGenerateAgentMd_NoSkills(t *testing.T) {
	agent := testAgent()
	md := claude.GenerateAgentMd(agent, nil, ".claude")
	assert.NotContains(t, md, "tools:")
	assert.NotContains(t, md, "## Output")
}

func TestResolveAgentTools(t *testing.T) {
	tools := claude.ResolveAgentTools(testResolvedSkills())
	// Should contain tools from both skills, merged and ordered
	assert.Contains(t, tools, "Grep")
	assert.Contains(t, tools, "Read")
	assert.Contains(t, tools, "Bash")
}
