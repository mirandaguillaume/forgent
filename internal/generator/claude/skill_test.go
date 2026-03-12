package claude_test

import (
	"strings"
	"testing"

	"github.com/mirandaguillaume/forgent/internal/generator/claude"
	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func makeGuardrailString(s string) model.GuardrailRule {
	var g model.GuardrailRule
	node := &yaml.Node{Kind: yaml.ScalarNode, Value: s}
	_ = g.UnmarshalYAML(node)
	return g
}

func testSkill() model.SkillBehavior {
	return model.SkillBehavior{
		Skill:   "code-review",
		Version: "1.0",
		Context: model.ContextFacet{
			Consumes: []string{"source-code", "diff"},
			Produces: []string{"review-report"},
			Memory:   model.MemoryConversation,
		},
		Strategy: model.StrategyFacet{
			Approach: "analytical",
			Tools:    []string{"read", "grep"},
			Steps:    []string{"Read the code", "Analyze patterns", "Write report"},
		},
		Guardrails: []model.GuardrailRule{
			makeGuardrailString("Never modify source files directly"),
			makeGuardrailString("Always explain reasoning"),
		},
		DependsOn: []model.Dependency{
			{Skill: "file-reader", Provides: "source-code"},
		},
		Security: model.SecurityFacet{
			Filesystem: model.AccessReadOnly,
			Network:    model.NetworkNone,
			Secrets:    []string{"GITHUB_TOKEN"},
			Sandbox:    "docker",
		},
	}
}

func TestGenerateSkillMd_Frontmatter(t *testing.T) {
	md := claude.GenerateSkillMd(testSkill())
	assert.Contains(t, md, "---\nname: code-review\n")
	assert.Contains(t, md, "description: analytical-based skill")
}

func TestGenerateSkillMd_Title(t *testing.T) {
	md := claude.GenerateSkillMd(testSkill())
	assert.Contains(t, md, "# Code Review")
}

func TestGenerateSkillMd_GuardrailsBeforeContext(t *testing.T) {
	md := claude.GenerateSkillMd(testSkill())
	guardrailIdx := strings.Index(md, "## Guardrails")
	contextIdx := strings.Index(md, "## Context")
	assert.Greater(t, contextIdx, guardrailIdx, "Guardrails should appear before Context")
}

func TestGenerateSkillMd_SecurityLast(t *testing.T) {
	md := claude.GenerateSkillMd(testSkill())
	securityIdx := strings.Index(md, "## Security")
	strategyIdx := strings.Index(md, "## Strategy")
	assert.Greater(t, securityIdx, strategyIdx, "Security should appear after Strategy")
}

func TestGenerateSkillMd_ContextSection(t *testing.T) {
	md := claude.GenerateSkillMd(testSkill())
	assert.Contains(t, md, "Consumes: source-code, diff")
	assert.Contains(t, md, "Produces: review-report")
	assert.Contains(t, md, "Memory: conversation")
}

func TestGenerateSkillMd_StrategySection(t *testing.T) {
	md := claude.GenerateSkillMd(testSkill())
	assert.Contains(t, md, "Approach: analytical")
	assert.Contains(t, md, "Tools: read, grep")
}

func TestGenerateSkillMd_StepsNumbered(t *testing.T) {
	md := claude.GenerateSkillMd(testSkill())
	assert.Contains(t, md, "1. Read the code")
	assert.Contains(t, md, "2. Analyze patterns")
	assert.Contains(t, md, "3. Write report")
}

func TestGenerateSkillMd_Dependencies(t *testing.T) {
	md := claude.GenerateSkillMd(testSkill())
	assert.Contains(t, md, "## Dependencies")
	assert.Contains(t, md, "**file-reader** provides `source-code`")
}

func TestGenerateSkillMd_Security(t *testing.T) {
	md := claude.GenerateSkillMd(testSkill())
	assert.Contains(t, md, "- Filesystem: read-only")
	assert.Contains(t, md, "- Network: none")
	assert.Contains(t, md, "- Secrets: GITHUB_TOKEN")
	assert.Contains(t, md, "- Sandbox: docker")
}

func TestGenerateSkillMd_NoGuardrails(t *testing.T) {
	skill := testSkill()
	skill.Guardrails = nil
	md := claude.GenerateSkillMd(skill)
	assert.NotContains(t, md, "## Guardrails")
}

func TestGenerateSkillMd_NoDependencies(t *testing.T) {
	skill := testSkill()
	skill.DependsOn = nil
	md := claude.GenerateSkillMd(skill)
	assert.NotContains(t, md, "## Dependencies")
}

func TestGenerateSkillMd_WhenToUse(t *testing.T) {
	skill := testSkill()
	skill.WhenToUse = model.WhenToUseFacet{
		Triggers:   []string{"Test failures", "Bug reports"},
		DontUse:    []string{"Simple typos"},
		Especially: []string{"Under pressure"},
	}
	md := claude.GenerateSkillMd(skill)
	assert.Contains(t, md, "## When to Use")
	assert.Contains(t, md, "- Test failures")
	assert.Contains(t, md, "- Simple typos")
	assert.Contains(t, md, "- Under pressure")
}

func TestGenerateSkillMd_WhenToUseEmpty(t *testing.T) {
	skill := testSkill()
	md := claude.GenerateSkillMd(skill)
	assert.NotContains(t, md, "## When to Use")
}

func TestGenerateSkillMd_WhenToUseBeforeContext(t *testing.T) {
	skill := testSkill()
	skill.WhenToUse = model.WhenToUseFacet{Triggers: []string{"always"}}
	md := claude.GenerateSkillMd(skill)
	wtuIdx := strings.Index(md, "## When to Use")
	ctxIdx := strings.Index(md, "## Context")
	assert.Greater(t, ctxIdx, wtuIdx, "When to Use should appear before Context")
}

func TestGenerateSkillMd_Examples(t *testing.T) {
	skill := testSkill()
	skill.Examples = []model.CodeExample{
		{Label: "Good: verify", Code: "go test ./...", Lang: "bash"},
	}
	md := claude.GenerateSkillMd(skill)
	assert.Contains(t, md, "## Examples")
	assert.Contains(t, md, "**Good: verify**")
	assert.Contains(t, md, "```bash")
	assert.Contains(t, md, "go test ./...")
}

func TestGenerateSkillMd_ExamplesEmpty(t *testing.T) {
	skill := testSkill()
	md := claude.GenerateSkillMd(skill)
	assert.NotContains(t, md, "## Examples")
}

func TestGenerateSkillMd_AntiPatterns(t *testing.T) {
	skill := testSkill()
	skill.AntiPatterns = []model.AntiPattern{
		{Excuse: "Quick fix", Reality: "Do it right"},
	}
	md := claude.GenerateSkillMd(skill)
	assert.Contains(t, md, "## Red Flags")
	assert.Contains(t, md, "| Quick fix | Do it right |")
}

func TestGenerateSkillMd_AntiPatternsEmpty(t *testing.T) {
	skill := testSkill()
	md := claude.GenerateSkillMd(skill)
	assert.NotContains(t, md, "## Red Flags")
}

func TestGenerateSkillMd_ExamplesBeforeSecurity(t *testing.T) {
	skill := testSkill()
	skill.Examples = []model.CodeExample{{Label: "test", Code: "echo hi"}}
	skill.AntiPatterns = []model.AntiPattern{{Excuse: "a", Reality: "b"}}
	md := claude.GenerateSkillMd(skill)
	exIdx := strings.Index(md, "## Examples")
	apIdx := strings.Index(md, "## Red Flags")
	secIdx := strings.Index(md, "## Security")
	assert.Greater(t, apIdx, exIdx, "Red Flags should appear after Examples")
	assert.Greater(t, secIdx, apIdx, "Security should appear after Red Flags")
}
