package experiment

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T3.1 — LoadTasks sets TaskDir on each loaded task.
func TestLoadTasks_SetsTaskDir(t *testing.T) {
	dir := t.TempDir()
	taskYAML := `id: test-task
name: Test
input: some code
steps:
  - id: step1
    prompt: do it
    produces: [output]
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "task.yaml"), []byte(taskYAML), 0644))

	tasks, err := LoadTasks(dir)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	// LoadTasks canonicalizes via EvalSymlinks (story 10-0 Task 1.2). On macOS
	// t.TempDir() returns /var/... which is a symlink to /private/var/...,
	// so compare via EvalSymlinks rather than the raw input.
	expectedDir, err := filepath.EvalSymlinks(dir)
	require.NoError(t, err)
	assert.Equal(t, expectedDir, tasks[0].TaskDir, "TaskDir should be the canonical (symlink-resolved) directory containing the task YAML")
}

// T3.2 — template_file resolves relative to task dir, not process CWD.
// This is the core integration test for the bug fix.
func TestRuntime_TemplateFile_ResolvesFromTaskDir(t *testing.T) {
	taskDir := t.TempDir()

	// Write a schema file in the task dir
	schemaContent := "## Verdict\n- **decision**: approved or rejected"
	require.NoError(t, os.WriteFile(filepath.Join(taskDir, "verdict-schema.md"), []byte(schemaContent), 0644))

	// Provider returns output that satisfies the template marker "## Verdict"
	provider := &mockGuardrailProvider{
		responses: []string{"## Verdict\n- **decision**: approved"},
	}

	task := TaskDefinition{
		ID:      "template-file-test",
		Name:    "Template File Test",
		Input:   "review this",
		TaskDir: taskDir, // set explicitly (simulating what LoadTasks does)
		Steps: []StepDef{
			{
				ID:     "review",
				Name:   "Review",
				Prompt: "Review and provide verdict",
				Produces: []SlotSpec{
					{Key: "verdict", TemplateFile: "verdict-schema.md"},
				},
			},
		},
	}

	config := ExperimentConfig{
		Tasks:       []TaskDefinition{task},
		Provider:    provider,
		RunsPerTask: 1,
	}

	results, err := RunRuntimeExperiment(config)
	require.NoError(t, err)
	require.Len(t, results, 1)

	// Task should complete — quality gate activated with template from task dir,
	// and provider output satisfies the "## Verdict" marker.
	assert.True(t, results[0].Completed, "task should complete when template_file resolves correctly")
	assert.Equal(t, 1, provider.callIdx, "no retry needed — output satisfies quality gate")
}

// T3.3 — template_file with empty TaskDir now returns an error (D1 decision: enforce invariant).
// buildTaskDAG returns an error rather than silently bypassing the quality gate.
func TestRuntime_TemplateFile_EmptyTaskDir_ReturnsError(t *testing.T) {
	provider := &mockGuardrailProvider{
		responses: []string{"some output"},
	}

	task := TaskDefinition{
		ID:    "no-taskdir-test",
		Name:  "No TaskDir",
		Input: "review this",
		// TaskDir intentionally empty — should now surface as a DAG build error
		Steps: []StepDef{
			{
				ID:     "review",
				Name:   "Review",
				Prompt: "Review",
				Produces: []SlotSpec{
					{Key: "verdict", TemplateFile: "verdict-schema.md"},
				},
			},
		},
	}

	config := ExperimentConfig{
		Tasks:       []TaskDefinition{task},
		Provider:    provider,
		RunsPerTask: 1,
	}

	results, err := RunRuntimeExperiment(config)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Error(t, results[0].Error, "empty TaskDir + template_file should return a DAG build error")
	assert.False(t, results[0].Completed, "task should not complete when DAG build fails")
	assert.Contains(t, results[0].Error.Error(), "requires TaskDir")
}

// T3.5 — path traversal via template_file returns an error (AC2).
func TestRuntime_TemplateFile_PathTraversal_ReturnsError(t *testing.T) {
	taskDir := t.TempDir()
	provider := &mockGuardrailProvider{
		responses: []string{"output"},
	}

	task := TaskDefinition{
		ID:      "traversal-test",
		Name:    "Traversal Test",
		Input:   "test",
		TaskDir: taskDir,
		Steps: []StepDef{
			{
				ID:     "step1",
				Name:   "Step 1",
				Prompt: "do it",
				Produces: []SlotSpec{{Key: "out", TemplateFile: "../../etc/passwd"}},
			},
		},
	}

	config := ExperimentConfig{
		Tasks:       []TaskDefinition{task},
		Provider:    provider,
		RunsPerTask: 1,
	}

	results, err := RunRuntimeExperiment(config)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Error(t, results[0].Error, "path traversal should surface as a task error")
	assert.Contains(t, results[0].Error.Error(), "escapes base directory")
}

// T3.4 — inline template (not template_file) is unaffected by this fix.
func TestRuntime_InlineTemplate_Unaffected(t *testing.T) {
	provider := &mockGuardrailProvider{
		responses: []string{"## Verdict\n- **decision**: approved"},
	}

	task := TaskDefinition{
		ID:    "inline-template-test",
		Name:  "Inline Template",
		Input: "review this",
		Steps: []StepDef{
			{
				ID:     "review",
				Name:   "Review",
				Prompt: "Review",
				Produces: []SlotSpec{
					{Key: "verdict", Template: "## Verdict\n- **decision**: description"},
				},
			},
		},
	}

	config := ExperimentConfig{
		Tasks:       []TaskDefinition{task},
		Provider:    provider,
		RunsPerTask: 1,
	}

	results, err := RunRuntimeExperiment(config)
	require.NoError(t, err)
	require.Len(t, results, 1)
	assert.True(t, results[0].Completed, "inline template should work regardless of TaskDir")
}

// TestLoadTasks_ResolvesSymlinkedDir — story 10-0 Task 1.2.
// LoadTasks must store the canonical (symlink-resolved) directory in TaskDir
// so TemplateFile path resolution always works against the real location even
// when callers reach the directory through a symlink.
func TestLoadTasks_ResolvesSymlinkedDir(t *testing.T) {
	realDir := t.TempDir()

	// Write a task in the real directory.
	taskYAML := `id: linked-task
name: Linked
input: code
steps:
  - id: step1
    prompt: do it
    produces: [output]
`
	require.NoError(t, os.WriteFile(filepath.Join(realDir, "task.yaml"), []byte(taskYAML), 0644))

	// Create a symlink that points to the real dir, then load via the symlink.
	linkParent := t.TempDir()
	link := filepath.Join(linkParent, "linked-tasks")
	if err := os.Symlink(realDir, link); err != nil {
		t.Skipf("symlinks not supported on this platform: %v", err)
	}

	tasks, err := LoadTasks(link)
	require.NoError(t, err)
	require.Len(t, tasks, 1)

	// EvalSymlinks may add or strip /private prefix on macOS — compare via EvalSymlinks itself.
	expectedDir, err := filepath.EvalSymlinks(realDir)
	require.NoError(t, err)
	assert.Equal(t, expectedDir, tasks[0].TaskDir,
		"TaskDir must be the canonical (symlink-resolved) directory")
}
