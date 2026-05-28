package experiment

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// ─── Unit tests on checkFileAccess + ExecuteToolForStep ───────────────────────

// TestStepFileAccess_ReadOnly_AllowsRead — story 10-3 T3.2.
// AC1: read operations are always permitted at the file_access layer.
func TestStepFileAccess_ReadOnly_AllowsRead(t *testing.T) {
	files := map[string][]byte{"src/main.go": []byte("package main")}
	sb := NewSandboxFS(files, nil, SandboxPolicy{})

	step := StepDef{ID: "review", FileAccess: FileAccessReadOnly}
	res := ExecuteToolForStep(sb, step, ToolCall{Name: "file_read", Args: map[string]string{"path": "src/main.go"}})

	require.False(t, res.Blocked, "file_read on a read-only step must not be blocked: reason=%q", res.Reason)
	assert.Equal(t, "package main", res.Output)
}

// TestStepFileAccess_ReadOnly_BlocksWriteAndDelete — story 10-3 T3.1 (AC1).
// Two attempts (write then delete) on a read-only step. Both must be blocked
// with the stable AC1 wording, and both must appear in the sandbox audit trail
// with Allowed: false.
func TestStepFileAccess_ReadOnly_BlocksWriteAndDelete(t *testing.T) {
	files := map[string][]byte{"src/main.go": []byte("package main")}
	sb := NewSandboxFS(files, nil, SandboxPolicy{})
	step := StepDef{ID: "review", FileAccess: FileAccessReadOnly}

	writeRes := ExecuteToolForStep(sb, step, ToolCall{Name: "file_write", Args: map[string]string{"path": "src/main.go", "content": "package x"}})
	require.True(t, writeRes.Blocked, "file_write must be blocked on read-only step")
	assert.Contains(t, writeRes.Reason, "step is read-only", "AC1 stable wording")

	deleteRes := ExecuteToolForStep(sb, step, ToolCall{Name: "file_delete", Args: map[string]string{"path": "src/main.go"}})
	require.True(t, deleteRes.Blocked, "file_delete must be blocked on read-only step")
	assert.Contains(t, deleteRes.Reason, "step is read-only", "AC1 stable wording — same message for write and delete on read-only")

	// Audit trail check (AC5): both denials recorded with Allowed: false.
	// Verb vocabulary: layer-1 uses "write"/"delete" (not "file_write"/"file_delete"),
	// matching layer-3's vocabulary so ComputeActionMetrics tallies correctly.
	log := sb.Log()
	denied := 0
	verbs := make(map[string]int)
	for _, e := range log {
		if !e.Allowed && e.StepID == "review" {
			denied++
			verbs[e.Action]++
			assert.Contains(t, e.Reason, "step is read-only", "audit trail Reason matches the denial reason")
		}
	}
	assert.Equal(t, 2, denied, "audit trail must record both denied attempts (write + delete)")
	assert.Equal(t, 1, verbs["write"], "exactly one denied entry with verb 'write'")
	assert.Equal(t, 1, verbs["delete"], "exactly one denied entry with verb 'delete'")
}

// TestStepFileAccess_ReadWrite_BlocksDelete — story 10-3 T3.3 (AC2).
// Read and write succeed; delete is blocked with the AC2 stable wording.
func TestStepFileAccess_ReadWrite_BlocksDelete(t *testing.T) {
	files := map[string][]byte{"src/main.go": []byte("package main")}
	sb := NewSandboxFS(files, nil, SandboxPolicy{})
	step := StepDef{ID: "edit", FileAccess: FileAccessReadWrite}

	readRes := ExecuteToolForStep(sb, step, ToolCall{Name: "file_read", Args: map[string]string{"path": "src/main.go"}})
	require.False(t, readRes.Blocked, "read must succeed on read-write step")

	writeRes := ExecuteToolForStep(sb, step, ToolCall{Name: "file_write", Args: map[string]string{"path": "src/main.go", "content": "package x"}})
	require.False(t, writeRes.Blocked, "write must succeed on read-write step: reason=%q", writeRes.Reason)

	deleteRes := ExecuteToolForStep(sb, step, ToolCall{Name: "file_delete", Args: map[string]string{"path": "src/main.go"}})
	require.True(t, deleteRes.Blocked, "delete must be blocked on read-write step")
	assert.Contains(t, deleteRes.Reason, "does not have delete permission", "AC2 stable wording")
}

// TestStepFileAccess_ReadWriteDelete_AllowsAll — story 10-3 T3.4 (AC3).
// All three tool calls succeed at the file_access layer.
func TestStepFileAccess_ReadWriteDelete_AllowsAll(t *testing.T) {
	files := map[string][]byte{"src/main.go": []byte("package main")}
	sb := NewSandboxFS(files, nil, SandboxPolicy{})
	step := StepDef{ID: "cleanup", FileAccess: FileAccessReadWriteDelete}

	readRes := ExecuteToolForStep(sb, step, ToolCall{Name: "file_read", Args: map[string]string{"path": "src/main.go"}})
	require.False(t, readRes.Blocked)

	writeRes := ExecuteToolForStep(sb, step, ToolCall{Name: "file_write", Args: map[string]string{"path": "src/new.go", "content": "package x"}})
	require.False(t, writeRes.Blocked, "write must succeed: reason=%q", writeRes.Reason)

	deleteRes := ExecuteToolForStep(sb, step, ToolCall{Name: "file_delete", Args: map[string]string{"path": "src/main.go"}})
	require.False(t, deleteRes.Blocked, "delete must succeed on read-write-delete step: reason=%q", deleteRes.Reason)
}

// TestStepFileAccess_DefaultIsReadWrite — story 10-3 T3.5 (AC4).
// A step with no FileAccess set behaves identically to FileAccessReadWrite.
// Pin the default explicitly so a future contributor cannot silently widen it
// to ReadWriteDelete (which would weaken existing tasks' security posture).
func TestStepFileAccess_DefaultIsReadWrite(t *testing.T) {
	files := map[string][]byte{"src/main.go": []byte("package main")}
	sb := NewSandboxFS(files, nil, SandboxPolicy{})
	step := StepDef{ID: "default-access"} // no FileAccess set

	// effectiveFileAccess sanity check.
	assert.Equal(t, FileAccessReadWrite, effectiveFileAccess(step), "empty FileAccess must map to read-write default")

	writeRes := ExecuteToolForStep(sb, step, ToolCall{Name: "file_write", Args: map[string]string{"path": "src/main.go", "content": "package x"}})
	assert.False(t, writeRes.Blocked, "default access must permit file_write")

	deleteRes := ExecuteToolForStep(sb, step, ToolCall{Name: "file_delete", Args: map[string]string{"path": "src/main.go"}})
	assert.True(t, deleteRes.Blocked, "default access must NOT permit file_delete")
	assert.Contains(t, deleteRes.Reason, "does not have delete permission")
}

// TestStepFileAccess_AuditTrailRecordsAttemptAndDenial — story 10-3 T3.8 (AC5).
// Verify the audit trail captures both attempted action and the denial reason
// in the form expected by the future telemetry aggregator (story 10-6):
//   - StepID set
//   - Action matches the tool's "verb" (the tool name)
//   - Path captures the offending file
//   - Allowed = false
//   - Reason matches the AC1/AC2 wording
func TestStepFileAccess_AuditTrailRecordsAttemptAndDenial(t *testing.T) {
	sb := NewSandboxFS(map[string][]byte{"x": []byte("y")}, nil, SandboxPolicy{})
	step := StepDef{ID: "telemetry-test", FileAccess: FileAccessReadOnly}

	_ = ExecuteToolForStep(sb, step, ToolCall{Name: "file_write", Args: map[string]string{"path": "x", "content": "z"}})

	log := sb.Log()
	require.Len(t, log, 1, "exactly one audit entry expected")
	entry := log[0]
	assert.Equal(t, "telemetry-test", entry.StepID)
	// Layer-1 audit entries use the SAME verb vocabulary as layer-3
	// (sandbox.go's recordLocked) — "write" not "file_write" — so
	// ComputeActionMetrics and downstream telemetry see one consistent
	// vocabulary. Pinned by the toolVerb helper in tools.go.
	assert.Equal(t, "write", entry.Action)
	assert.Equal(t, "x", entry.Path)
	assert.False(t, entry.Allowed)
	assert.Contains(t, entry.Reason, "step is read-only")
}

// TestStepFileAccess_PrecedenceFileAccessBeforeAllowedPaths — story 10-3 T2.1.
// When BOTH file_access (layer 1) and AllowedPaths (layer 2) would deny, layer 1
// fires first. This locks in the documented precedence so a future refactor
// that swaps the order is detected.
func TestStepFileAccess_PrecedenceFileAccessBeforeAllowedPaths(t *testing.T) {
	sb := NewSandboxFS(map[string][]byte{"src/main.go": []byte("package main")}, nil, SandboxPolicy{})
	step := StepDef{
		ID:         "deny-test",
		FileAccess: FileAccessReadOnly,
		Sandbox:    &StepSandboxPolicy{AllowedPaths: []string{"src/**"}},
	}

	// file_write to dist/x — both layers would deny (read-only AND not in allowed_paths).
	res := ExecuteToolForStep(sb, step, ToolCall{Name: "file_write", Args: map[string]string{"path": "dist/x", "content": "y"}})
	require.True(t, res.Blocked)
	assert.Contains(t, res.Reason, "step is read-only", "file_access layer should fire first, not the path-glob layer")
}

// ─── Validate tests ───────────────────────────────────────────────────────────

// TestValidate_RejectsUnknownFileAccess — story 10-3 T3.6 (AC6).
func TestValidate_RejectsUnknownFileAccess(t *testing.T) {
	cfg := ExperimentConfig{
		Tasks: []TaskDefinition{
			{
				ID:    "t1",
				Steps: []StepDef{{ID: "s1", FileAccess: "read-modify"}}, // typo
			},
		},
		Provider: &mockGuardrailProvider{},
	}
	err := cfg.Validate()
	require.Error(t, err, "Validate must reject unknown file_access values")
	assert.Contains(t, err.Error(), "read-modify", "error must reference the offending value")
	assert.Contains(t, err.Error(), "read-only", "error must list valid values")
	assert.Contains(t, err.Error(), "read-write", "error must list valid values")
	assert.Contains(t, err.Error(), "read-write-delete", "error must list valid values")
}

// TestValidate_AcceptsEmptyFileAccessAsDefault — story 10-3 T1.4.
// Empty string is valid (treated as default downstream). Pin so a future
// stricter validator that rejects empty does not silently break NFR40.
func TestValidate_AcceptsEmptyFileAccessAsDefault(t *testing.T) {
	cfg := ExperimentConfig{
		Tasks: []TaskDefinition{
			{
				ID:    "t1",
				Steps: []StepDef{{ID: "s1" /* FileAccess unset */}},
			},
		},
		Provider: &mockGuardrailProvider{},
	}
	err := cfg.Validate()
	assert.NoError(t, err, "empty FileAccess must be valid (treated as default)")
}

// TestValidate_AcceptsAllValidFileAccessValues — story 10-3 T1.4.
func TestValidate_AcceptsAllValidFileAccessValues(t *testing.T) {
	for _, val := range []FileAccess{FileAccessReadOnly, FileAccessReadWrite, FileAccessReadWriteDelete} {
		t.Run(string(val), func(t *testing.T) {
			cfg := ExperimentConfig{
				Tasks: []TaskDefinition{
					{
						ID:    "t1",
						Steps: []StepDef{{ID: "s1", FileAccess: val}},
					},
				},
				Provider: &mockGuardrailProvider{},
			}
			err := cfg.Validate()
			assert.NoError(t, err, "valid FileAccess value %q must pass Validate", val)
		})
	}
}

// ─── YAML round-trip ──────────────────────────────────────────────────────────

// TestStepDef_FileAccess_YAMLRoundTrip — story 10-3 T3.9.
// file_access parses correctly; omit-empty respected on marshal so existing
// fixture YAMLs that don't set the field stay unchanged.
func TestStepDef_FileAccess_YAMLRoundTrip(t *testing.T) {
	raw := `
id: review
name: Review
prompt: Review the code
file_access: read-only
`
	var step StepDef
	require.NoError(t, yaml.Unmarshal([]byte(raw), &step))
	assert.Equal(t, FileAccessReadOnly, step.FileAccess)

	out, err := yaml.Marshal(step)
	require.NoError(t, err)
	assert.Contains(t, string(out), "file_access: read-only")

	// Empty FileAccess must be omitted on marshal (NFR40 — old YAMLs unchanged).
	step2 := StepDef{ID: "x", Name: "X", Prompt: "do stuff"}
	out2, err := yaml.Marshal(step2)
	require.NoError(t, err)
	assert.NotContains(t, string(out2), "file_access", "empty file_access must be omitted via omitempty")
}

// TestStepFileAccess_WorkspaceReadOnlyOverridesStepLevel — story 10-3 T5.2.
// Workspace-level ReadOnly is a more restrictive layer than per-step
// FileAccess. A step that declares FileAccessReadWriteDelete on a ReadOnly
// workspace must STILL be denied at write — the most-restrictive-wins
// composition is the documented contract (precedence layer 3 in the
// ExecuteToolForStep doc comment).
func TestStepFileAccess_WorkspaceReadOnlyOverridesStepLevel(t *testing.T) {
	sb := NewSandboxFS(map[string][]byte{"x": []byte("y")}, nil, SandboxPolicy{ReadOnly: true})
	step := StepDef{ID: "ws-readonly", FileAccess: FileAccessReadWriteDelete}

	res := ExecuteToolForStep(sb, step, ToolCall{Name: "file_write", Args: map[string]string{"path": "x", "content": "z"}})
	require.True(t, res.Blocked, "workspace ReadOnly must override step FileAccess: read-write-delete")
	assert.Contains(t, res.Reason, "read-only sandbox", "denial comes from the workspace layer (SandboxFS.Write), not the step layer")
}

// TestStepFileAccess_NoRetryOnPermissionDenial — story 10-3 T3.7.
// Permission denial via FileAccess at the tool dispatcher must NOT trigger
// the OUTER guardrail-retry path. The retry path fires only when the FINAL
// response violates a guardrail.
//
// Distinguishing "no retry because guardrail was satisfied" from "no retry
// because permission denial doesn't trigger retry" requires checking the
// AUDIT-TRAIL count, not just the provider callCount: a hypothetical bug
// that retried each outer attempt would re-issue the same denied tool call,
// producing N denials (one per attempt) rather than 1.
func TestStepFileAccess_NoRetryOnPermissionDenial(t *testing.T) {
	callCount := 0
	provider := &callCountingMock{
		fn: func(prompt string) (string, error) {
			callCount++
			if callCount == 1 {
				return `<tool name="file_write" path="x" content="bad"/>`, nil
			}
			return "Final answer: APPROVED", nil
		},
	}

	task := TaskDefinition{
		ID:    "no-retry-on-denial",
		Name:  "No retry on denial",
		Input: "test",
		Steps: []StepDef{
			{
				ID:         "review",
				Name:       "Review",
				Prompt:     "Review",
				Guardrails: []string{"APPROVED"},
				FileAccess: FileAccessReadOnly,
				Tools:      []string{"file_write"},
				Produces:   []SlotSpec{{Key: "verdict"}},
			},
		},
		Workspace: &WorkspaceDef{Files: []WorkspaceFile{{Path: "x", Content: "y"}}},
	}

	results, err := RunRuntimeExperiment(ExperimentConfig{
		Tasks:       []TaskDefinition{task},
		Provider:    provider,
		RunsPerTask: 1,
	})
	require.NoError(t, err)
	require.Len(t, results, 1)

	// callCount == 2: (1) initial tool emission, (2) post-tool-result final
	// answer satisfying the guardrail. Necessary check, not sufficient on its
	// own (V1 of this test stopped here, which Blind Hunter flagged as
	// "passes for the wrong reason" — see Review Findings).
	assert.Equal(t, 2, callCount, "tool-loop pair: 1 tool emission + 1 final answer")
	assert.True(t, results[0].Completed, "step completes despite the blocked tool call")
	assert.Equal(t, 1.0, results[0].GuardrailScore, "guardrail satisfied — score 1.0")

	// Audit-trail count is the actual distinguisher. With correct semantics:
	// exactly 1 denial (the single tool call blocked once). With a hypothetical
	// "retry-on-denial" bug: 2-3 denials (each outer attempt re-issues the
	// same tool call, producing one denial per attempt). This assertion fails
	// under the bug — V1's callCount-only check did not.
	denials := 0
	for _, e := range results[0].ActionLog {
		if !e.Allowed && e.StepID == "review" {
			denials++
		}
	}
	assert.Equal(t, 1, denials,
		"exactly one permission denial recorded — outer guardrail-retry must NOT re-trigger on permission denial")
}

// ─── Backward-compatibility check on the deprecated entry point ───────────────

// TestExecuteToolWithPolicy_DefaultsToReadWrite — story 10-3 backward compat.
// The legacy ExecuteToolWithPolicy entry point (no file_access parameter) must
// behave as if file_access were FileAccessDefault (= read-write). Pinned so a
// future cleanup that removes the shim is forced to also migrate every caller.
func TestExecuteToolWithPolicy_DefaultsToReadWrite(t *testing.T) {
	sb := NewSandboxFS(map[string][]byte{"x": []byte("y")}, nil, SandboxPolicy{})

	// Write must succeed (default = read-write).
	writeRes := ExecuteToolWithPolicy(sb, "legacy", ToolCall{Name: "file_write", Args: map[string]string{"path": "x", "content": "z"}}, nil)
	assert.False(t, writeRes.Blocked)

	// Delete must be blocked (default = read-write, not read-write-delete).
	deleteRes := ExecuteToolWithPolicy(sb, "legacy", ToolCall{Name: "file_delete", Args: map[string]string{"path": "x"}}, nil)
	require.True(t, deleteRes.Blocked, "legacy entry point inherits the read-write default; delete must be denied")
	assert.True(t, strings.Contains(deleteRes.Reason, "does not have delete permission"))
}

// ─── End-to-end fixture validation ────────────────────────────────────────────

// TestLoadAllTasks_ValidateClean — story 10-3 P5 (review).
// Loads every YAML under internal/bench/experiment/tasks/ and runs Validate
// on the resulting config. Catches NFR40 regressions (e.g. the
// tool-accidental-destruction fixture's file_delete step needing
// file_access: read-write-delete) at CI time rather than at experiment-run
// time. The provider is a no-op mock — Validate does not call it.
func TestLoadAllTasks_ValidateClean(t *testing.T) {
	tasks, err := LoadTasks("tasks")
	require.NoError(t, err, "LoadTasks must succeed for the shipped fixture corpus")
	require.NotEmpty(t, tasks, "expected at least one task fixture under tasks/")

	cfg := ExperimentConfig{
		Tasks:    tasks,
		Provider: &mockGuardrailProvider{},
	}
	require.NoError(t, cfg.Validate(), "every shipped task fixture must pass Validate without modification (NFR40)")
}

// TestCheckFileAccess_UnknownLevelDefaultDeny — story 10-3 review P10.
// The inner switch's default branch — reached via direct API misuse that
// bypasses Validate — must DENY rather than silently allow. Pinned so a
// future refactor that changes default to "allow" would fail this test.
func TestCheckFileAccess_UnknownLevelDefaultDeny(t *testing.T) {
	// file_write with an unknown FileAccess value
	reason, ok := checkFileAccess(ToolFileWrite, FileAccess("read-modify"))
	require.False(t, ok, "unknown FileAccess level must default-deny")
	assert.Contains(t, reason, "unknown file_access level")
	assert.Contains(t, reason, "read-modify")

	// file_delete with the same garbage value — same default-deny
	reason, ok = checkFileAccess(ToolFileDelete, FileAccess("garbage"))
	require.False(t, ok)
	assert.Contains(t, reason, "unknown file_access level")
}

// TestParseToolCalls_LowercasesName — story 10-3 review P3.
// LLM-emitted tool names are normalised to lowercase at parse time so the
// dispatcher's enum switch is not bypassed by case variations. Closes the
// determinism leak between LLM stochasticity and runtime decisions.
func TestParseToolCalls_LowercasesName(t *testing.T) {
	resp := `<tool name="FILE_WRITE" path="x" content="y"/>` + "\n" + `<tool name="File_Read" path="z"/>`
	calls := ParseToolCalls(resp)
	require.Len(t, calls, 2)
	assert.Equal(t, "file_write", calls[0].Name, "uppercase tool name must be lowercased")
	assert.Equal(t, "file_read", calls[1].Name, "mixed-case tool name must be lowercased")
}

// TestStepFileAccess_CaseInsensitiveDispatch — story 10-3 review P3 integration.
// An LLM emitting `<tool name="FILE_WRITE"/>` on a read-only step must hit
// the layer-1 deny (not slip past as an "unknown tool") because ParseToolCalls
// normalises before dispatch. End-to-end check.
func TestStepFileAccess_CaseInsensitiveDispatch(t *testing.T) {
	sb := NewSandboxFS(map[string][]byte{"x": []byte("y")}, nil, SandboxPolicy{})
	step := StepDef{ID: "case-test", FileAccess: FileAccessReadOnly}

	// Simulate post-parse: the call name is already lowercase by contract.
	// (The runtime calls ParseToolCalls, then dispatches via ExecuteToolForStep.)
	calls := ParseToolCalls(`<tool name="FILE_WRITE" path="x" content="z"/>`)
	require.Len(t, calls, 1)

	res := ExecuteToolForStep(sb, step, calls[0])
	require.True(t, res.Blocked, "uppercase FILE_WRITE on read-only step must be blocked after parse-time lowercasing")
	assert.Contains(t, res.Reason, "step is read-only")
}
