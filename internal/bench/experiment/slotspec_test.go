package experiment

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSlotSpec_UnmarshalYAML_Shorthand(t *testing.T) {
	input := `produces: [bugs, style_issues]`
	var step StepDef
	require.NoError(t, yaml.Unmarshal([]byte(input), &step))
	assert.Equal(t, []SlotSpec{{Key: "bugs"}, {Key: "style_issues"}}, step.Produces)
}

func TestSlotSpec_UnmarshalYAML_FullForm(t *testing.T) {
	input := `produces:
  - key: bugs
    template: "## Bugs\n- **location**: description"`
	var step StepDef
	require.NoError(t, yaml.Unmarshal([]byte(input), &step))
	require.Len(t, step.Produces, 1)
	assert.Equal(t, "bugs", step.Produces[0].Key)
	assert.Contains(t, step.Produces[0].Template, "## Bugs")
}

func TestSlotSpec_UnmarshalYAML_Mixed(t *testing.T) {
	input := `
produces:
  - bugs
  - key: fixed_code
    template: "` + "`" + `go\n{{code}}\n` + "`" + `"
consumes: [task_input]`
	var step StepDef
	require.NoError(t, yaml.Unmarshal([]byte(input), &step))
	assert.Equal(t, "bugs", step.Produces[0].Key)
	assert.Equal(t, "fixed_code", step.Produces[1].Key)
	assert.Equal(t, "task_input", step.Consumes[0].Key)
}

func TestSlotSpec_ResolveTemplate_Inline(t *testing.T) {
	s := SlotSpec{Key: "bugs", Template: "## Bugs\n- item"}
	tmpl, err := s.ResolveTemplate("")
	require.NoError(t, err)
	assert.Equal(t, "## Bugs\n- item", tmpl)
}

func TestSlotSpec_ResolveTemplate_File(t *testing.T) {
	dir := t.TempDir()
	content := "## Style Issues\n- **rule**: description"
	path := filepath.Join(dir, "style.md")
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))

	s := SlotSpec{Key: "style", TemplateFile: "style.md"}
	tmpl, err := s.ResolveTemplate(dir)
	require.NoError(t, err)
	assert.Equal(t, content, tmpl)
}

func TestSlotSpec_ResolveTemplate_None(t *testing.T) {
	s := SlotSpec{Key: "output"}
	tmpl, err := s.ResolveTemplate("")
	require.NoError(t, err)
	assert.Equal(t, "", tmpl)
}

func TestSlotSpec_UnmarshalYAML_EmptyKey(t *testing.T) {
	input := `produces: [""]`
	var step StepDef
	err := yaml.Unmarshal([]byte(input), &step)
	assert.ErrorContains(t, err, "key must not be empty")
}

func TestSlotSpec_ResolveTemplate_PathTraversal(t *testing.T) {
	dir := t.TempDir()
	s := SlotSpec{Key: "x", TemplateFile: "../../etc/passwd"}
	_, err := s.ResolveTemplate(dir)
	assert.ErrorContains(t, err, "escapes base directory")
}

func TestSlotSpec_UnmarshalYAML_UnexpectedNodeKind(t *testing.T) {
	// Sequence-of-sequences produces an unexpected node kind for an element
	input := `produces: [[nested]]`
	var step StepDef
	err := yaml.Unmarshal([]byte(input), &step)
	assert.Error(t, err)
}

func TestDefaultProduces_Empty(t *testing.T) {
	result := defaultProduces("my-step", nil)
	assert.Equal(t, []SlotSpec{{Key: "my-step_output"}}, result)
}

func TestDefaultProduces_NonEmpty(t *testing.T) {
	slots := []SlotSpec{{Key: "bugs"}}
	result := defaultProduces("my-step", slots)
	assert.Equal(t, slots, result)
}

func TestSlotSpec_Keys(t *testing.T) {
	slots := []SlotSpec{{Key: "bugs"}, {Key: "style_issues"}}
	assert.Equal(t, []string{"bugs", "style_issues"}, SlotKeys(slots))
}

func TestTaskDefinition_BackwardCompatibility(t *testing.T) {
	yaml_input := `
id: code-review
name: Code Review
input: "some code"
steps:
  - id: identify-bugs
    prompt: Find bugs
    produces: [bugs]
    consumes: []
    expected_keywords: [bug, error]
  - id: generate-fix
    prompt: Fix bugs
    consumes: [bugs]
    produces: [fixed_code]
`
	var task TaskDefinition
	require.NoError(t, yaml.Unmarshal([]byte(yaml_input), &task))
	assert.Equal(t, "bugs", task.Steps[0].Produces[0].Key)
	assert.Equal(t, "fixed_code", task.Steps[1].Produces[0].Key)
	assert.Equal(t, "bugs", task.Steps[1].Consumes[0].Key)
}
