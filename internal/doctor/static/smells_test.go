package static

import (
	"strings"
	"testing"

	"github.com/mirandaguillaume/forgent/internal/doctor/parser"
	"github.com/stretchr/testify/assert"
)

func TestSmells_GodAgent_Detected(t *testing.T) {
	// >3000 words + >8 sections
	bigContent := strings.Repeat("word ", 3100)
	sections := make([]parser.Section, 10)
	for i := range sections {
		sections[i] = parser.Section{Header: "Section", Level: 2, Content: "content"}
	}

	analysis := &parser.AgentAnalysis{
		RawContent: []byte(bigContent),
		Sections:   sections,
	}

	f := detectGodAgent(analysis)
	assert.NotNil(t, f)
	assert.Contains(t, f.Issue, "God agent")
}

func TestSmells_GodAgent_Normal(t *testing.T) {
	analysis := &parser.AgentAnalysis{
		RawContent: []byte("Short file with a few words."),
		Sections:   []parser.Section{{Header: "Rules", Level: 2}},
	}

	f := detectGodAgent(analysis)
	assert.Nil(t, f)
}

func TestSmells_LLMGenerated_Detected(t *testing.T) {
	content := `## Rules
Additionally, you should follow these rules.
Furthermore, always validate input.
Moreover, use proper error handling.
In addition, log all operations.
1. First step
2. Second step
3. Third step
4. Fourth step
5. Fifth step
6. Sixth step`

	analysis := &parser.AgentAnalysis{RawContent: []byte(content)}
	f := detectLLMGenerated(analysis)
	assert.NotNil(t, f)
	assert.Contains(t, f.Issue, "LLM-generated")
}

func TestSmells_LLMGenerated_HumanStyle(t *testing.T) {
	content := "## Rules\nAlways run tests.\nNever skip linting.\nUse gofmt."
	analysis := &parser.AgentAnalysis{RawContent: []byte(content)}
	f := detectLLMGenerated(analysis)
	assert.Nil(t, f, "human-style text should not trigger LLM detection")
}

func TestSmells_CheckboxAgent_Detected(t *testing.T) {
	// 9 sections, each with <20 words
	sections := make([]parser.Section, 9)
	for i := range sections {
		sections[i] = parser.Section{Header: "Section", Level: 2, Content: "Short."}
	}

	analysis := &parser.AgentAnalysis{
		RawContent: []byte("content"),
		Sections:   sections,
	}

	f := detectCheckboxAgent(analysis)
	assert.NotNil(t, f)
	assert.Contains(t, f.Issue, "Checkbox agent")
}

func TestSmells_CheckboxAgent_Substantial(t *testing.T) {
	sections := make([]parser.Section, 9)
	for i := range sections {
		sections[i] = parser.Section{Header: "Section", Level: 2, Content: strings.Repeat("word ", 30)}
	}

	analysis := &parser.AgentAnalysis{
		RawContent: []byte("content"),
		Sections:   sections,
	}

	f := detectCheckboxAgent(analysis)
	assert.Nil(t, f, "substantial sections should not trigger checkbox smell")
}

func TestSmells_OverConstrained_Detected(t *testing.T) {
	// >10 prohibitions, <3 permissions
	prohibitions := strings.Repeat("Never do X. Don't do Y. Must not do Z. Avoid A. ", 4)
	content := "## Rules\n" + prohibitions

	analysis := &parser.AgentAnalysis{RawContent: []byte(content)}
	f := detectOverConstrained(analysis)
	assert.NotNil(t, f)
	assert.Contains(t, f.Issue, "Over-constrained")
}

func TestSmells_OverConstrained_Balanced(t *testing.T) {
	content := "## Rules\nNever expose secrets.\nDon't skip tests.\nYou should always run linting.\nYou can use any testing framework.\nYou are allowed to modify test files.\nYou are recommended to use Go conventions."

	analysis := &parser.AgentAnalysis{RawContent: []byte(content)}
	f := detectOverConstrained(analysis)
	assert.Nil(t, f, "balanced file should not trigger over-constrained smell")
}

func TestSmells_CopyPaste_Detected(t *testing.T) {
	// Content that matches most Claude default template phrases
	content := "You are Claude, an AI assistant made by Anthropic. You are helpful harmless and honest. Respond to the human in a helpful and informative way. Follow instructions carefully. Think step by step. Provide accurate information. If you are unsure, let the user know."
	analysis := &parser.AgentAnalysis{RawContent: []byte(content)}
	f := detectCopyPaste(analysis)
	assert.NotNil(t, f, "should detect copy-paste from default Claude template")
	assert.Contains(t, f.Issue, "Copy-paste")
}

func TestSmells_CopyPaste_Customized(t *testing.T) {
	content := "## Identity\nYou are a Go code reviewer for the Forgent project.\n## Rules\nAlways run go test before suggesting changes.\nUse gofmt conventions."
	analysis := &parser.AgentAnalysis{RawContent: []byte(content)}
	f := detectCopyPaste(analysis)
	assert.Nil(t, f, "customized file should not trigger copy-paste")
}

func TestSmells_NilAnalysis(t *testing.T) {
	check := &smellsCheck{}
	assert.Nil(t, check.Run(nil))
}

func TestSmells_EmptyContent(t *testing.T) {
	check := &smellsCheck{}
	assert.Nil(t, check.Run(&parser.AgentAnalysis{RawContent: []byte{}}))
}

func TestSmells_AllTaggedThorough(t *testing.T) {
	check := &smellsCheck{}
	assert.Equal(t, []string{"thorough"}, check.Tags())
}
