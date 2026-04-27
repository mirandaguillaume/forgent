// Package qualitygate provides a DAG middleware that validates node outputs
// against format templates declared in SlotSpec contracts. When required
// structural markers are missing, the middleware returns an error that triggers
// the DAG's built-in retry mechanism (runWithRetry).
package qualitygate

import (
	"context"
	"fmt"
	"strings"

	"github.com/mirandaguillaume/forgent/pkg/dag"
)

// QualityGatePolicy defines the format contracts for a step's inputs and outputs.
// Templates come from SlotSpec.ResolveTemplate() on consumes/produces slots.
// Empty template = no constraint (pass-through).
type QualityGatePolicy struct {
	// ProducesTemplate: format template validated against node output after execution.
	ProducesTemplate string
	// ConsumesTemplate: format template validated against node input before execution.
	ConsumesTemplate string
}

// QualityGateMiddleware returns a dag.Middleware that enforces structural format
// contracts declared in the policy templates.
//
// Validation is structural (not schema parsing): markers are extracted from the
// template (headings, bold list items) and checked for presence in the target
// text (case-insensitive substring match).
//
// If policy is nil or both templates produce no markers, returns a no-op pass-through.
func QualityGateMiddleware(policy *QualityGatePolicy) dag.Middleware {
	if policy == nil {
		return passthrough
	}

	producesMarkers := extractMarkers(policy.ProducesTemplate)
	consumesMarkers := extractMarkers(policy.ConsumesTemplate)

	if len(producesMarkers) == 0 && len(consumesMarkers) == 0 {
		return passthrough
	}

	return func(ctx context.Context, inputs map[string]any, next dag.MiddlewareFunc) (map[string]any, error) {
		// Pre-check: validate inputs against consumes template markers.
		if len(consumesMarkers) > 0 {
			combined := strings.ToLower(collectStrings(inputs))
			for _, marker := range consumesMarkers {
				if !checkMarker(combined, marker) {
					return nil, fmt.Errorf("quality gate: input missing expected structure %q", marker)
				}
			}
		}

		out, err := next(ctx, inputs)
		if err != nil {
			return nil, err
		}

		// Post-check: validate output against produces template markers.
		if len(producesMarkers) > 0 {
			combined := strings.ToLower(collectStrings(out))
			for _, marker := range producesMarkers {
				if !checkMarker(combined, marker) {
					return nil, fmt.Errorf("quality gate: output missing expected structure %q", marker)
				}
			}
		}

		return out, nil
	}
}

// passthrough is a no-op middleware used when no constraints are active.
var passthrough dag.Middleware = func(ctx context.Context, inputs map[string]any, next dag.MiddlewareFunc) (map[string]any, error) {
	return next(ctx, inputs)
}

// extractMarkers scans a format template and returns structural markers:
//   - Markdown headings (## Title → "## title")
//   - Bold list items    (- **field**: ... → "- **")
//   - Plain list items   (- item → "- ")
//
// Markers are lowercased and deduplicated. Returns nil for empty templates.
func extractMarkers(tmpl string) []string {
	if tmpl == "" {
		return nil
	}

	var markers []string
	seen := make(map[string]bool)

	add := func(s string) {
		if !seen[s] {
			seen[s] = true
			markers = append(markers, s)
		}
	}

	hasBoldList := false
	hasPlainList := false

	for _, line := range strings.Split(tmpl, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "## ") || strings.HasPrefix(trimmed, "### "):
			add(strings.ToLower(trimmed))
		case strings.HasPrefix(trimmed, "- **") || strings.HasPrefix(trimmed, "* **"):
			if !hasBoldList {
				hasBoldList = true
				add("- **")
			}
		case strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* "):
			if !hasBoldList && !hasPlainList {
				hasPlainList = true
				add("- ")
			}
		}
	}

	if len(markers) == 0 {
		return nil
	}
	return markers
}

// checkMarker reports whether the combined text satisfies the marker.
// The bold-list marker "- **" also matches "* **" since both are valid
// Markdown bullet styles that an LLM may choose regardless of the template style.
func checkMarker(combined, marker string) bool {
	if strings.Contains(combined, marker) {
		return true
	}
	if marker == "- **" {
		return strings.Contains(combined, "* **")
	}
	return false
}

// collectStrings concatenates all string values from the output map,
// separated by spaces. Non-string values are silently skipped.
func collectStrings(outputs map[string]any) string {
	var b strings.Builder
	for _, v := range outputs {
		if s, ok := v.(string); ok {
			if b.Len() > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(s)
		}
	}
	return b.String()
}
