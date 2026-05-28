package experiment

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mirandaguillaume/forgent/pkg/schema"
	"gopkg.in/yaml.v3"
)

// SlotSpec defines a named data slot in a step's consumes/produces contract.
// It carries an optional format template or typed schema:
//
//   - template / template_file: Markdown structural validation for human-facing outputs.
//   - schema: Typed field contract for inter-node structured outputs (AC7, NFR40).
//     When Schema is non-empty the runtime uses ValidateTyped + constrained decoding
//     instead of the Markdown path; the slot file is written as slots/<key>.json.
//   - list_schema: Declares that this slot's value is a JSON array ([]any). Each element
//     must satisfy the FieldSpec. Mutually exclusive with schema. Used by kind: map and
//     kind: filter steps as their input type; also used by aggregation steps that need
//     to produce a list for downstream map/filter consumption.
//     The LLM is instructed to output a JSON array; runtime extracts via extractJSONArray.
//
// YAML supports two forms:
//
//	shorthand: produces: [bugs]             → SlotSpec{Key: "bugs"}
//	full form: produces: [{key: bugs, template: "## Bugs\n..."}]
type SlotSpec struct {
	Key          string             `yaml:"key"`
	Template     string             `yaml:"template,omitempty"`      // inline format template
	TemplateFile string             `yaml:"template_file,omitempty"` // path relative to task file dir
	Schema       schema.FieldSchema `yaml:"schema,omitempty"`        // typed inter-node schema (object)
	ListSchema   *schema.FieldSpec  `yaml:"list_schema,omitempty"`   // typed inter-node schema (array)
}

// UnmarshalYAML supports both shorthand (string) and full-form (map) YAML.
func (s *SlotSpec) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		if value.Value == "" {
			return fmt.Errorf("slotspec: key must not be empty")
		}
		s.Key = value.Value
		return nil
	case yaml.MappingNode:
		// slotAlias avoids infinite recursion; schema.FieldSchema (map type)
		// is decoded natively by the yaml library.
		type slotAlias SlotSpec
		var alias slotAlias
		if err := value.Decode(&alias); err != nil {
			return fmt.Errorf("slotspec: %w", err)
		}
		*s = SlotSpec(alias)
		if s.Key == "" {
			return fmt.Errorf("slotspec: key must not be empty")
		}
		return nil
	default:
		return fmt.Errorf("slotspec: unexpected YAML node kind %v", value.Kind)
	}
}

// ResolveTemplate returns the resolved template string.
//   - If TemplateFile is set: reads the file relative to baseDir.
//     Empty baseDir is rejected to prevent the traversal check being bypassed
//     and the path being resolved against the process CWD (defense in depth
//     on top of the runtime_runner.go pre-check).
//     Symlinks in both baseDir and the resolved path are evaluated before
//     the relative-path check, so a symlink pointing outside baseDir is
//     detected even when the link target is created after task load
//     (TOCTOU resistance).
//   - Else: returns inline Template (may be empty = no constraint)
func (s SlotSpec) ResolveTemplate(baseDir string) (string, error) {
	if s.TemplateFile == "" {
		return s.Template, nil
	}
	if baseDir == "" {
		return "", fmt.Errorf("slotspec %q: template_file %q requires a non-empty baseDir (set TaskDefinition.TaskDir or load via LoadTasks)", s.Key, s.TemplateFile)
	}

	path := filepath.Join(baseDir, s.TemplateFile)

	// Lexical check first — catches "../etc/passwd" before touching the filesystem.
	if rel, err := filepath.Rel(baseDir, path); err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("slotspec %q: template_file %q escapes base directory", s.Key, s.TemplateFile)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("slotspec %q: read template file %q: %w", s.Key, path, err)
	}

	// Post-read symlink check. Fail-closed when the path IS a symlink: any
	// failure to resolve it canonically must reject the read, otherwise an
	// attacker who can cause EvalSymlinks to fail (deleted target after read,
	// permission denied on intermediate component, etc.) would silently bypass
	// the traversal defense. When the path is NOT a symlink, no resolution is
	// necessary — the lexical check earlier has already proved it stays inside
	// baseDir.
	fi, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("slotspec %q: post-read stat of %q failed: %w", s.Key, path, err)
	}
	if fi.Mode()&os.ModeSymlink != 0 {
		realPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			return "", fmt.Errorf("slotspec %q: post-read symlink resolution of %q failed: %w", s.Key, path, err)
		}
		realBase, err := filepath.EvalSymlinks(baseDir)
		if err != nil {
			return "", fmt.Errorf("slotspec %q: baseDir symlink resolution of %q failed: %w", s.Key, baseDir, err)
		}
		if rel, relErr := filepath.Rel(realBase, realPath); relErr != nil || strings.HasPrefix(rel, "..") {
			return "", fmt.Errorf("slotspec %q: template_file %q escapes base directory via symlink", s.Key, s.TemplateFile)
		}
	}
	return string(data), nil
}

// SlotKeys extracts the key names from a slice of SlotSpec.
// Used when callers need the []string form for DAG node wiring.
func SlotKeys(slots []SlotSpec) []string {
	keys := make([]string, len(slots))
	for i, s := range slots {
		keys[i] = s.Key
	}
	return keys
}

// defaultProduces returns the step's produces slots, falling back to a
// synthetic key derived from the step ID when the slice is empty.
// Both buildTaskDAG and evaluateRuntimeOutputs must use this to stay in sync.
func defaultProduces(stepID string, produces []SlotSpec) []SlotSpec {
	if len(produces) > 0 {
		return produces
	}
	return []SlotSpec{{Key: stepID + "_output"}}
}
