package experiment

import (
	"fmt"
	"regexp"
	"strings"
)

// ToolCall represents a parsed tool invocation from an LLM response.
type ToolCall struct {
	Name string
	Args map[string]string
}

// ToolResult represents the outcome of executing a tool call.
type ToolResult struct {
	Name    string
	Output  string
	Blocked bool
	Reason  string
}

// toolCallRe matches XML-style tool calls: <tool name="file_read" path="src/main.go"/>
// (?s) enables dot-matches-newline for multiline tool calls.
var toolCallRe = regexp.MustCompile(`(?s)<tool\s+(.*?)\s*/>`)
var attrRe = regexp.MustCompile(`(\w+)="([^"]*)"`)

// ParseToolCalls extracts structured tool calls from LLM free-text response.
// Format: <tool name="file_read" path="src/main.go"/>
func ParseToolCalls(response string) []ToolCall {
	matches := toolCallRe.FindAllStringSubmatch(response, -1)
	var calls []ToolCall
	for _, m := range matches {
		attrs := attrRe.FindAllStringSubmatch(m[1], -1)
		args := make(map[string]string)
		for _, a := range attrs {
			args[a[1]] = a[2]
		}
		name := args["name"]
		delete(args, "name")
		if name != "" {
			calls = append(calls, ToolCall{Name: name, Args: args})
		}
	}
	return calls
}

// ExecuteToolWithPolicy checks per-step allowed paths before dispatching to the sandbox.
// If the step has a sandbox policy, the tool call's path must match at least one allowed path.
func ExecuteToolWithPolicy(sandbox *SandboxFS, stepID string, call ToolCall, stepPolicy *StepSandboxPolicy) ToolResult {
	if stepPolicy != nil && len(stepPolicy.AllowedPaths) > 0 {
		path := call.Args["path"]
		if path != "" {
			allowed := false
			for _, pattern := range stepPolicy.AllowedPaths {
				if globMatch(pattern, path) {
					allowed = true
					break
				}
			}
			if !allowed {
				reason := fmt.Sprintf("step policy: path %q not in allowed_paths %v", path, stepPolicy.AllowedPaths)
				sandbox.record(stepID, call.Name, path, false, reason)
				return ToolResult{Name: call.Name, Blocked: true, Reason: reason}
			}
		}
	}
	return ExecuteTool(sandbox, stepID, call)
}

// ExecuteTool dispatches a tool call through the sandbox and returns the result.
func ExecuteTool(sandbox *SandboxFS, stepID string, call ToolCall) ToolResult {
	switch call.Name {
	case "file_read":
		path := call.Args["path"]
		if path == "" {
			return ToolResult{Name: call.Name, Output: "Error: missing required argument 'path'"}
		}
		content, err := sandbox.Read(stepID, path)
		if err != nil {
			if strings.Contains(err.Error(), "denied") {
				return ToolResult{Name: call.Name, Blocked: true, Reason: err.Error()}
			}
			return ToolResult{Name: call.Name, Output: fmt.Sprintf("Error: %s", err.Error())}
		}
		return ToolResult{Name: call.Name, Output: string(content)}

	case "file_write":
		path := call.Args["path"]
		if path == "" {
			return ToolResult{Name: call.Name, Output: "Error: missing required argument 'path'"}
		}
		content := call.Args["content"]
		err := sandbox.Write(stepID, path, []byte(content))
		if err != nil {
			if strings.Contains(err.Error(), "denied") {
				return ToolResult{Name: call.Name, Blocked: true, Reason: err.Error()}
			}
			return ToolResult{Name: call.Name, Output: fmt.Sprintf("Error: %s", err.Error())}
		}
		return ToolResult{Name: call.Name, Output: fmt.Sprintf("Written %d bytes to %s", len(content), path)}

	case "file_delete":
		path := call.Args["path"]
		if path == "" {
			return ToolResult{Name: call.Name, Output: "Error: missing required argument 'path'"}
		}
		err := sandbox.Delete(stepID, path)
		if err != nil {
			if strings.Contains(err.Error(), "denied") {
				return ToolResult{Name: call.Name, Blocked: true, Reason: err.Error()}
			}
			return ToolResult{Name: call.Name, Output: fmt.Sprintf("Error: %s", err.Error())}
		}
		return ToolResult{Name: call.Name, Output: fmt.Sprintf("Deleted %s", path)}

	default:
		return ToolResult{Name: call.Name, Output: fmt.Sprintf("Unknown tool: %s", call.Name)}
	}
}

// FormatToolResults formats tool results as text for LLM re-prompting.
func FormatToolResults(results []ToolResult) string {
	var sb strings.Builder
	sb.WriteString("## Tool Results\n\n")
	for _, r := range results {
		if r.Blocked {
			sb.WriteString(fmt.Sprintf("**%s**: BLOCKED — %s\n\n", r.Name, r.Reason))
		} else {
			sb.WriteString(fmt.Sprintf("**%s**:\n%s\n\n", r.Name, r.Output))
		}
	}
	return sb.String()
}
