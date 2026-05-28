package experiment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveIssueTaskLoads verifies that resolve-issue.yaml loads without error
// and satisfies all structural requirements for the SWE-bench experiment.
func TestResolveIssueTaskLoads(t *testing.T) {
	tasks, err := LoadTasks("tasks")
	require.NoError(t, err, "LoadTasks must not error on the tasks directory")

	var task *TaskDefinition
	for i := range tasks {
		if tasks[i].ID == "resolve-issue" {
			task = &tasks[i]
			break
		}
	}
	require.NotNil(t, task, "resolve-issue.yaml must be present in tasks directory")

	// AC1 structural requirements.
	assert.Equal(t, "resolve-issue", task.ID)
	assert.NotEmpty(t, task.Name)

	// Must have at least 3 steps (read-issue, generate-patch, validate-patch minimum).
	assert.GreaterOrEqual(t, len(task.Steps), 3, "must have at least 3 steps")

	// Required step IDs.
	stepIDs := make(map[string]bool, len(task.Steps))
	for _, s := range task.Steps {
		stepIDs[s.ID] = true
	}
	assert.True(t, stepIDs["read-issue"], "must have read-issue step")
	assert.True(t, stepIDs["generate-patch"], "must have generate-patch step")
	assert.True(t, stepIDs["validate-patch"], "must have validate-patch step")

	// Final step must produce a patch slot.
	finalStep := task.Steps[len(task.Steps)-1]
	assert.Equal(t, "validate-patch", finalStep.ID, "validate-patch must be the final step")
	require.NotEmpty(t, finalStep.Produces, "validate-patch must produce at least one slot")

	// Task-level guardrails must include unified diff markers.
	guardrailSet := make(map[string]bool, len(task.Guardrails))
	for _, g := range task.Guardrails {
		guardrailSet[g] = true
	}
	assert.True(t, guardrailSet["--- a/"], `guardrails must contain "--- a/"`)
	assert.True(t, guardrailSet["+++ b/"], `guardrails must contain "+++ b/"`)

	// Specialist tool_nodes must be defined (complexity + security).
	toolNodeIDs := make(map[string]bool, len(task.ToolNodes))
	for _, n := range task.ToolNodes {
		toolNodeIDs[n.ID] = true
	}
	assert.True(t, toolNodeIDs["complexity-specialist"], "must have complexity-specialist tool_node")
	assert.True(t, toolNodeIDs["security-specialist"], "must have security-specialist tool_node")

	// generate-patch must reference agent_tools.
	var generatePatch *StepDef
	for i := range task.Steps {
		if task.Steps[i].ID == "generate-patch" {
			generatePatch = &task.Steps[i]
			break
		}
	}
	require.NotNil(t, generatePatch)
	assert.NotEmpty(t, generatePatch.AgentTools, "generate-patch must declare agent_tools")

	// All steps must use file_access: read-only (no writes during patch generation).
	for _, step := range task.Steps {
		if step.Kind != "" {
			continue // non-LLM steps may not declare file_access
		}
		assert.Equal(t, FileAccessReadOnly, step.FileAccess,
			"step %q must be read-only", step.ID)
	}
}
