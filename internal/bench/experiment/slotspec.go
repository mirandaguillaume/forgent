package experiment

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SlotSpec defines a named data slot in a step's consumes/produces contract.
// It carries an optional format template that the QualityGate uses to validate
// that outputs conform to the declared structure.
//
// YAML supports two forms:
//
//	shorthand: produces: [bugs]             → SlotSpec{Key: "bugs"}
//	full form: produces: [{key: bugs, template: "## Bugs\n..."}]
type SlotSpec struct {
	Key          string `yaml:"key"`
	Template     string `yaml:"template,omitempty"`      // inline format template
	TemplateFile string `yaml:"template_file,omitempty"` // path relative to task file dir
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
//   - If TemplateFile is set: reads the file relative to baseDir
//   - Else: returns inline Template (may be empty = no constraint)
func (s SlotSpec) ResolveTemplate(baseDir string) (string, error) {
	if s.TemplateFile != "" {
		path := filepath.Join(baseDir, s.TemplateFile)
		// Prevent path traversal: resolved path must stay within baseDir.
		if baseDir != "" {
			rel, err := filepath.Rel(baseDir, path)
			if err != nil || strings.HasPrefix(rel, "..") {
				return "", fmt.Errorf("slotspec %q: template_file %q escapes base directory", s.Key, s.TemplateFile)
			}
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return "", fmt.Errorf("slotspec %q: read template file %q: %w", s.Key, path, err)
		}
		return string(data), nil
	}
	return s.Template, nil
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
