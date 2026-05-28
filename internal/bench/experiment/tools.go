package experiment

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// ToolName is the closed set of tool kinds the experiment runtime knows how
// to dispatch. Idiomatic Go enum (typed string + constants), parallel to
// StepKind and FileAccess. ToolCall.Name carries the LLM-emitted string,
// lowercased by ParseToolCalls so the dispatch switch operates on a
// case-normalised input.
//
// **Divergence from `pkg/sandbox/middleware.go`'s `toolAction` map:** that
// map (used by the not-yet-wired sandbox middleware) accepts alias names
// (`read_file`, `write_file`, `delete_file`, `list_directory`, `list_dir`).
// The experiment runtime currently uses ONLY the `file_*` form, both at the
// LLM prompt level (ParseToolCalls + tool prompts) and at this dispatcher.
// If a future story wires the sandbox middleware as the enforcement seam,
// the alias map must be reconciled with this enum: either extend ToolName
// + checkFileAccess + ExecuteTool's switches to recognise aliases, or drop
// the aliases from `pkg/sandbox`. Until then the divergence is benign —
// no caller bridges the two — but flagged here as a known contract gap.
type ToolName string

const (
	ToolFileRead    ToolName = "file_read"
	ToolFileWrite   ToolName = "file_write"
	ToolFileDelete  ToolName = "file_delete"
	ToolCtx7Lookup  ToolName = "ctx7_lookup"
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
//
// Tool names are lowercased at parse time so the dispatcher (ExecuteTool /
// checkFileAccess) can switch on the closed ToolName enum without an
// LLM-induced case-sensitivity bypass. An LLM emitting `<tool name="FILE_WRITE"/>`
// would otherwise fall through every switch's `default:` arm and silently
// skip both enforcement and dispatch — a real determinism leak between LLM
// stochasticity and runtime decisions. Normalising here makes the runtime's
// response a pure function of the SEMANTIC tool call, not its surface casing.
func ParseToolCalls(response string) []ToolCall {
	matches := toolCallRe.FindAllStringSubmatch(response, -1)
	var calls []ToolCall
	for _, m := range matches {
		attrs := attrRe.FindAllStringSubmatch(m[1], -1)
		args := make(map[string]string)
		for _, a := range attrs {
			args[a[1]] = a[2]
		}
		name := strings.ToLower(args["name"])
		delete(args, "name")
		if name != "" {
			calls = append(calls, ToolCall{Name: name, Args: args})
		}
	}
	return calls
}

// ExecuteToolWithPolicy checks per-step allowed paths before dispatching to the sandbox.
// If the step has a sandbox policy, the tool call's path must match at least one allowed path.
//
// Deprecated: prefer ExecuteToolForStep when the caller has access to the StepDef
// so the per-step file_access permission (story 10-3) is also enforced.
//
// **Behavioural change vs pre-10-3:** this shim now applies `FileAccessDefault`
// (= read-write) at layer 1, which BLOCKS file_delete calls that previously
// passed through unchecked. Legacy callers that emit `file_delete` must
// migrate to `ExecuteToolForStep` with a `StepDef` carrying
// `FileAccess: FileAccessReadWriteDelete`. The tightening is intentional —
// the new default matches the security posture this story documents — but
// is a breaking change for anyone embedding the package programmatically
// who was relying on the old "no permission check" semantics.
func ExecuteToolWithPolicy(sandbox *SandboxFS, stepID string, call ToolCall, stepPolicy *StepSandboxPolicy) ToolResult {
	return executeToolWithPolicies(sandbox, stepID, call, stepPolicy, FileAccessDefault)
}

// ExecuteToolForStep is the preferred entry point. Enforces the layers in
// the documented precedence:
//
//  1. Per-step file_access (read-only / read-write / read-write-delete) —
//     story 10-3. Blocks file_write when the step is read-only; blocks
//     file_delete unless the step is read-write-delete.
//  2. Per-step Sandbox.AllowedPaths globs — story 5-4. Blocks any path not
//     matching at least one allowed pattern.
//  3. Workspace-level SandboxPolicy (ReadOnly, MaxFileSize, DeniedGlobs,
//     DeniedWritePatterns) — pre-existing. Enforced inside SandboxFS methods.
//
// Most-restrictive-wins: any of the three layers can deny independently.
// Permission denials at layer 1 record an audit entry with the StepID,
// the attempted action, and a stable Reason string ("step is read-only" or
// "step does not have delete permission"). The stable wording is part of
// the contract — log parsers (story 10-6 telemetry) depend on it.
func ExecuteToolForStep(sandbox *SandboxFS, step StepDef, call ToolCall) ToolResult {
	return executeToolWithPolicies(sandbox, step.ID, call, step.Sandbox, effectiveFileAccess(step))
}

func executeToolWithPolicies(sandbox *SandboxFS, stepID string, call ToolCall, stepPolicy *StepSandboxPolicy, fileAccess FileAccess) ToolResult {
	toolName := ToolName(call.Name)
	verb := toolVerb(toolName)

	// Layer 1: per-step file_access permission (story 10-3).
	if reason, ok := checkFileAccess(toolName, fileAccess); !ok {
		path := call.Args["path"]
		sandbox.record(stepID, verb, path, false, reason)
		return ToolResult{Name: call.Name, Blocked: true, Reason: reason}
	}

	// Layer 2: per-step AllowedPaths globs (story 5-4).
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
				sandbox.record(stepID, verb, path, false, reason)
				return ToolResult{Name: call.Name, Blocked: true, Reason: reason}
			}
		}
	}

	// Layer 3: workspace SandboxPolicy enforced inside SandboxFS methods.
	return ExecuteTool(sandbox, stepID, call)
}

// toolVerb maps a ToolName to the audit-trail verb string used by SandboxFS.
// Layer-1 (this file) and layer-2 (this file) audit entries must use the same
// verb vocabulary as layer-3 (sandbox.go's recordLocked: "read"/"write"/
// "delete") so ComputeActionMetrics and downstream telemetry (story 10-6)
// see ONE consistent vocabulary across all denial paths. Without this
// translation, layer-1 entries land as "file_write" and silently bypass the
// `e.Action == "write"` filter in ComputeActionMetrics.
//
// Unknown tool names pass through verbatim so audit consumers can still
// distinguish "wrong tool name" from "permission denied" cases.
func toolVerb(name ToolName) string {
	switch name {
	case ToolFileRead:
		return "read"
	case ToolFileWrite:
		return "write"
	case ToolFileDelete:
		return "delete"
	default:
		return string(name)
	}
}

// checkFileAccess reports whether the tool call is permitted by the
// step's file_access level. Returns ("", true) when allowed, or
// (denial-reason, false) when blocked.
//
// Reason wording is stable contract — see the doc comment on
// ExecuteToolForStep.
//
// Both inputs are typed (ToolName, FileAccess) so the switch statements
// inside operate on the closed enum sets without further casts; the cast
// from ToolCall.Name (raw string) happens at the dispatcher boundary.
func checkFileAccess(toolName ToolName, fileAccess FileAccess) (string, bool) {
	switch toolName {
	case ToolFileRead:
		// Read is permitted at every level (FileAccessReadOnly, ReadWrite,
		// ReadWriteDelete). Always allowed at this layer.
		return "", true
	case ToolFileWrite:
		switch fileAccess {
		case FileAccessReadOnly:
			return "permission denied: step is read-only", false
		case FileAccessReadWrite, FileAccessReadWriteDelete:
			return "", true
		default:
			// Empty already mapped to FileAccessDefault by effectiveFileAccess;
			// any other value should have been rejected by Validate. Defensive
			// fall-through: deny to avoid silent escalation.
			return fmt.Sprintf("permission denied: unknown file_access level %q", fileAccess), false
		}
	case ToolFileDelete:
		switch fileAccess {
		case FileAccessReadOnly:
			return "permission denied: step is read-only", false
		case FileAccessReadWrite:
			return "permission denied: step does not have delete permission", false
		case FileAccessReadWriteDelete:
			return "", true
		default:
			return fmt.Sprintf("permission denied: unknown file_access level %q", fileAccess), false
		}
	default:
		// Non-file tools (custom user tools) bypass this layer.
		return "", true
	}
}

// ExecuteTool dispatches a tool call through the sandbox and returns the result.
func ExecuteTool(sandbox *SandboxFS, stepID string, call ToolCall) ToolResult {
	switch ToolName(call.Name) {
	case ToolFileRead:
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

	case ToolFileWrite:
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

	case ToolFileDelete:
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

	case ToolCtx7Lookup:
		return executeCtx7Lookup(call)

	default:
		return ToolResult{Name: call.Name, Output: fmt.Sprintf("Unknown tool: %s", call.Name)}
	}
}

// executeCtx7Lookup resolves a library via ctx7 and fetches relevant docs.
// Two-step: (1) npx ctx7@latest library <name> "<query>" to get ID,
// (2) npx ctx7@latest docs <id> "<query>" to fetch content.
// Each step has a 30s timeout; total cap is 60s.
func executeCtx7Lookup(call ToolCall) ToolResult {
	library := call.Args["library"]
	query := call.Args["query"]
	if library == "" || query == "" {
		return ToolResult{Name: call.Name, Output: "Error: ctx7_lookup requires 'library' and 'query' arguments"}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	libOut, err := exec.CommandContext(ctx, "npx", "ctx7@latest", "library", library, query).Output()
	if err != nil {
		return ToolResult{Name: call.Name, Output: fmt.Sprintf("ctx7 library lookup failed for %q: %v", library, err)}
	}

	libraryID := parseCtx7LibraryID(string(libOut))
	if libraryID == "" {
		return ToolResult{Name: call.Name, Output: fmt.Sprintf("ctx7: no library found matching %q", library)}
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel2()

	docsOut, err := exec.CommandContext(ctx2, "npx", "ctx7@latest", "docs", libraryID, query).Output()
	if err != nil {
		return ToolResult{Name: call.Name, Output: fmt.Sprintf("ctx7 docs fetch failed for %s: %v", libraryID, err)}
	}

	docs := string(docsOut)
	const maxDocs = 3000
	if len(docs) > maxDocs {
		docs = docs[:maxDocs] + "\n...(truncated)"
	}
	return ToolResult{Name: call.Name, Output: fmt.Sprintf("Library: %s\n\n%s", libraryID, docs)}
}

// parseCtx7LibraryID scans ctx7 library output for the first /org/project ID.
func parseCtx7LibraryID(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		for _, field := range strings.Fields(line) {
			if strings.HasPrefix(field, "/") && strings.Count(field, "/") >= 2 {
				return field
			}
		}
	}
	return ""
}

// TextResolver encapsulates the demand→resolution mapping for the text-loop tool path.
// Built once at step init from the YAML tools declaration; the LLM sees only the
// lightweight Describe() output in its prompt — no JSON schemas injected.
type TextResolver struct {
	names   []string // ordered, for stable Describe() output
	allowed map[string]bool
	sandbox *SandboxFS
	step    StepDef
}

// NewTextResolver builds a resolver from the step's declared tool names.
// file_read is always included; names are deduplicated.
func NewTextResolver(names []string, sandbox *SandboxFS, step StepDef) *TextResolver {
	seen := map[string]bool{"file_read": true}
	order := []string{"file_read"}
	for _, n := range names {
		if !seen[n] {
			seen[n] = true
			order = append(order, n)
		}
	}
	return &TextResolver{names: order, allowed: seen, sandbox: sandbox, step: step}
}

// Describe generates the ## Available Tools prompt section.
// One line per tool with usage hint — no JSON schema, minimal token cost.
func (r *TextResolver) Describe() string {
	var sb strings.Builder
	sb.WriteString("## Available Tools\n\n")
	sb.WriteString("Use XML format: `<tool name=\"tool_name\" arg1=\"value1\"/>`\n\n")
	for _, name := range r.names {
		sb.WriteString(fmt.Sprintf("- %s\n", toolUsageHint(ToolName(name))))
	}
	sb.WriteString("\n")
	return sb.String()
}

// Resolve dispatches a batch of tool calls, enforcing the allowed set.
// Unknown or forbidden tools return a blocked ToolResult.
func (r *TextResolver) Resolve(calls []ToolCall) []ToolResult {
	results := make([]ToolResult, 0, len(calls))
	for _, call := range calls {
		if !r.allowed[call.Name] {
			results = append(results, ToolResult{
				Name:    call.Name,
				Blocked: true,
				Reason:  fmt.Sprintf("tool %q not available in this step (allowed: %v)", call.Name, r.names),
			})
			continue
		}
		results = append(results, ExecuteToolForStep(r.sandbox, r.step, call))
	}
	return results
}

// toolUsageHint returns a one-line usage hint shown to the LLM in the prompt.
func toolUsageHint(name ToolName) string {
	switch name {
	case ToolFileRead:
		return `file_read path="<path>" — Read a file from the workspace`
	case ToolFileWrite:
		return `file_write path="<path>" content="<content>" — Write a file`
	case ToolFileDelete:
		return `file_delete path="<path>" — Delete a file`
	case ToolCtx7Lookup:
		return `ctx7_lookup library="<name>" query="<question>" — Look up library docs`
	default:
		return string(name)
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
