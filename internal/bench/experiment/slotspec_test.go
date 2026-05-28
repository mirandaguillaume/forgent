package experiment

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mirandaguillaume/forgent/pkg/schema"
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

// TestResolveTemplate_EmptyBaseDir_ReturnsError — story 10-0 Task 1.3.
// Empty baseDir + non-empty TemplateFile must fail explicitly so callers cannot
// silently fall through to reading the TemplateFile path from the process CWD.
func TestResolveTemplate_EmptyBaseDir_ReturnsError(t *testing.T) {
	s := SlotSpec{Key: "verdict", TemplateFile: "verdict-schema.md"}
	_, err := s.ResolveTemplate("")
	require.Error(t, err, "ResolveTemplate must reject empty baseDir when TemplateFile is set")
	assert.Contains(t, err.Error(), "non-empty baseDir")
}

// TestResolveTemplate_EmptyBaseDir_NoTemplateFileIsOk — inline templates are
// unaffected by the empty-baseDir rule.
func TestResolveTemplate_EmptyBaseDir_NoTemplateFileIsOk(t *testing.T) {
	s := SlotSpec{Key: "verdict", Template: "## Verdict"}
	out, err := s.ResolveTemplate("")
	require.NoError(t, err)
	assert.Equal(t, "## Verdict", out)
}

// TestResolveTemplate_RejectsSymlinkOutsideBase — story 10-0 Task 1.3.
// A symlink inside baseDir that resolves to a file outside baseDir must be
// rejected even though the lexical filepath.Rel check passes (the symlink
// path itself is inside baseDir; only EvalSymlinks reveals the target).
func TestResolveTemplate_RejectsSymlinkOutsideBase(t *testing.T) {
	baseDir := t.TempDir()
	outsideDir := t.TempDir()

	// Create a sensitive file outside baseDir.
	secret := filepath.Join(outsideDir, "secret.md")
	require.NoError(t, os.WriteFile(secret, []byte("secret content"), 0644))

	// Create a symlink inside baseDir pointing to the outside file.
	link := filepath.Join(baseDir, "alias.md")
	if err := os.Symlink(secret, link); err != nil {
		t.Skipf("symlinks not supported on this platform: %v", err)
	}

	s := SlotSpec{Key: "out", TemplateFile: "alias.md"}
	_, err := s.ResolveTemplate(baseDir)
	require.Error(t, err, "symlinked TemplateFile pointing outside baseDir must be rejected")
	assert.Contains(t, err.Error(), "escapes base directory")
}

// TestResolveTemplate_AcceptsSymlinkInsideBase — symlinks that stay inside
// baseDir are allowed (e.g. shared schema files in a sub-dir).
func TestResolveTemplate_AcceptsSymlinkInsideBase(t *testing.T) {
	baseDir := t.TempDir()

	target := filepath.Join(baseDir, "schemas", "verdict.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(target), 0755))
	require.NoError(t, os.WriteFile(target, []byte("## Verdict"), 0644))

	link := filepath.Join(baseDir, "verdict.md")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks not supported on this platform: %v", err)
	}

	s := SlotSpec{Key: "verdict", TemplateFile: "verdict.md"}
	out, err := s.ResolveTemplate(baseDir)
	require.NoError(t, err)
	assert.Equal(t, "## Verdict", out)
}

// ─── T1.3: FieldSchema YAML round-trip ───────────────────────────────────────

func TestSlotSpec_Schema_UnmarshalYAML(t *testing.T) {
	input := `
produces:
  - key: verdict
    schema:
      ruling:
        type: string
        enum: [approved, rejected]
      rationale:
        type: string
        min_length: 20
        description: "Explanation of the ruling"
`
	var step StepDef
	require.NoError(t, yaml.Unmarshal([]byte(input), &step))
	require.Len(t, step.Produces, 1)
	s := step.Produces[0]
	assert.Equal(t, "verdict", s.Key)
	assert.Equal(t, schema.FieldSchema{
		"ruling": {Type: "string", Enum: []string{"approved", "rejected"}},
		"rationale": {Type: "string", MinLength: 20, Description: "Explanation of the ruling"},
	}, s.Schema)
}

func TestExperimentConfig_Validate_UnknownFieldType_ReturnsError(t *testing.T) {
	cfg := ExperimentConfig{
		Tasks: []TaskDefinition{{
			ID: "bad",
			Steps: []StepDef{{
				ID: "step1",
				Produces: []SlotSpec{{
					Key: "verdict",
					Schema: schema.FieldSchema{
						"ruling": {Type: "object"}, // invalid
					},
				}},
			}},
		}},
		Provider:    &mockGuardrailProvider{},
		RunsPerTask: 1,
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported type")
	assert.Contains(t, err.Error(), "object")
}

func TestExperimentConfig_Validate_ValidFieldTypes_NoError(t *testing.T) {
	cfg := ExperimentConfig{
		Tasks: []TaskDefinition{{
			ID: "ok",
			Steps: []StepDef{{
				ID: "step1",
				Produces: []SlotSpec{{
					Key: "verdict",
					Schema: schema.FieldSchema{
						"ruling":    {Type: "string"},
						"score":     {Type: "number"},
						"count":     {Type: "integer"},
						"compliant": {Type: "boolean"},
					},
				}},
			}},
		}},
		Provider:    &mockGuardrailProvider{},
		RunsPerTask: 1,
	}
	assert.NoError(t, cfg.Validate())
}

func TestExperimentConfig_Validate_MultipleTypedProducesSlots_ReturnsError(t *testing.T) {
	cfg := ExperimentConfig{
		Tasks: []TaskDefinition{{
			ID: "bad",
			Steps: []StepDef{{
				ID: "step1",
				Produces: []SlotSpec{
					{Key: "verdict", Schema: schema.FieldSchema{"ruling": {Type: "string"}}},
					{Key: "score", Schema: schema.FieldSchema{"value": {Type: "number"}}},
				},
			}},
		}},
		Provider:    &mockGuardrailProvider{},
		RunsPerTask: 1,
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "at most 1 is supported")
}

func TestExperimentConfig_Validate_MinLengthOnNumber_ReturnsError(t *testing.T) {
	cfg := ExperimentConfig{
		Tasks: []TaskDefinition{{
			ID: "bad",
			Steps: []StepDef{{
				ID: "step1",
				Produces: []SlotSpec{{
					Key: "out",
					Schema: schema.FieldSchema{
						"score": {Type: "number", MinLength: 5},
					},
				}},
			}},
		}},
		Provider:    &mockGuardrailProvider{},
		RunsPerTask: 1,
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "min_length is only valid for type \"string\"")
}

func TestExperimentConfig_Validate_MixedTypedAndMarkdownProduces_ReturnsError(t *testing.T) {
	cfg := ExperimentConfig{
		Tasks: []TaskDefinition{{
			ID: "bad",
			Steps: []StepDef{{
				ID: "step1",
				Produces: []SlotSpec{
					{Key: "typed", Schema: schema.FieldSchema{"ruling": {Type: "string"}}},
					{Key: "doc", Template: "## Report"},
				},
			}},
		}},
		Provider:    &mockGuardrailProvider{},
		RunsPerTask: 1,
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mixes typed produces")
}

func TestExperimentConfig_Validate_TypedProducesWithExitWhen_ReturnsError(t *testing.T) {
	cfg := ExperimentConfig{
		Tasks: []TaskDefinition{{
			ID: "bad",
			Steps: []StepDef{{
				ID: "step1",
				Loop: &LoopPolicy{MaxIterations: 3, ExitWhen: ExitCriteria{OutputMarker: "DONE"}},
				Produces: []SlotSpec{{
					Key: "verdict",
					Schema: schema.FieldSchema{"ruling": {Type: "string"}},
				}},
			}},
		}},
		Provider:    &mockGuardrailProvider{},
		RunsPerTask: 1,
	}
	err := cfg.Validate()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "incompatible with loop exit criteria")
}
