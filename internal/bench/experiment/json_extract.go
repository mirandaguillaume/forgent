package experiment

import (
	"encoding/json"
	"fmt"
	"strings"
)

// extractJSON extracts and parses a JSON object from an LLM response.
// It handles three common wrapping patterns produced by LLMs:
//  1. Bare JSON object:    {"key": "value"}
//  2. Code-fence wrapped:  ```json\n{"key": "value"}\n```
//  3. Preamble text:       "Here is the result:\n{...}"
//
// Returns an error with "json extraction:" prefix on all failure paths
// so classifyValidationReason can map it to a stable telemetry prefix.
func extractJSON(s string) (map[string]any, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("json extraction: empty input")
	}

	// 1. Code-fence extraction: strip ```[json]\n ... ``` wrapper.
	if idx := strings.Index(s, "```"); idx >= 0 {
		fence := s[idx+3:]
		// Skip optional language tag (e.g. "json\n")
		if nl := strings.Index(fence, "\n"); nl >= 0 {
			fence = fence[nl+1:]
		}
		if end := strings.Index(fence, "```"); end >= 0 {
			candidate := strings.TrimSpace(fence[:end])
			var m map[string]any
			if err := json.Unmarshal([]byte(candidate), &m); err == nil {
				return m, nil
			}
		}
	}

	// 2. Bare JSON unmarshal.
	var m map[string]any
	if err := json.Unmarshal([]byte(s), &m); err == nil {
		return m, nil
	}

	// 3. Brace scan: try each '{' in order until one yields a valid JSON object.
	//    Depth-aware matching respects string literals so embedded braces don't count.
	//    Trying all candidates handles preamble text like "Note: {...} actual: {...}".
	for searchFrom := 0; ; {
		rel := strings.Index(s[searchFrom:], "{")
		if rel < 0 {
			break
		}
		start := searchFrom + rel
		depth := 0
		inStr := false
		escaped := false
		end := -1
		for i := start; i < len(s); i++ {
			ch := s[i]
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' && inStr {
				escaped = true
				continue
			}
			if ch == '"' {
				inStr = !inStr
				continue
			}
			if inStr {
				continue
			}
			switch ch {
			case '{':
				depth++
			case '}':
				depth--
				if depth == 0 {
					end = i
				}
			}
			if end >= 0 {
				break
			}
		}
		if end >= 0 {
			if err := json.Unmarshal([]byte(s[start:end+1]), &m); err == nil {
				return m, nil
			}
		}
		searchFrom = start + 1
	}
	return nil, fmt.Errorf("json extraction: no valid JSON object found")
}
