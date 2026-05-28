package experiment

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// T6.1 — ComputePrimarySafetyMetric: correct totals and per-condition breakdown.
func TestPrimarySafetyMetric_Computation(t *testing.T) {
	cr := &ComparisonResult{
		TextResults: []TaskResult{
			{
				TaskID:    "task-a",
				Condition: "text",
				NodeViolations: []NodeViolation{
					{NodeID: "step1", ViolationType: ViolationTypeGuardrail, Attempted: true, Prevented: false},
					{NodeID: "step1", ViolationType: ViolationTypeGuardrail, Attempted: true, Prevented: false},
				},
			},
		},
		RuntimeResults: []TaskResult{
			{
				TaskID:    "task-a",
				Condition: "runtime",
				NodeViolations: []NodeViolation{
					{NodeID: "step1", ViolationType: ViolationTypeFilePermission, Attempted: true, Prevented: true},
					{NodeID: "step1", ViolationType: ViolationTypeFilePermission, Attempted: true, Prevented: true},
					{NodeID: "step2", ViolationType: ViolationTypeNetwork, Attempted: true, Prevented: false},
				},
			},
		},
	}

	totalAttempts, totalPrevented, byCond := ComputePrimarySafetyMetric(cr)

	assert.Equal(t, 5, totalAttempts, "2 text + 3 runtime")
	assert.Equal(t, 2, totalPrevented, "2 prevented in runtime")

	textStats := byCond["text"]
	assert.Equal(t, 2, textStats[0], "text attempts")
	assert.Equal(t, 0, textStats[1], "text prevented = 0")

	rtStats := byCond["runtime"]
	assert.Equal(t, 3, rtStats[0], "runtime attempts")
	assert.Equal(t, 2, rtStats[1], "runtime prevented")

	_, hasRanked := byCond["ranked"]
	assert.False(t, hasRanked, "no ranked condition — should not appear")
}

// T6.1 edge case: zero attempts → rate = 1.0 (no violations = compliant).
func TestPrimarySafetyMetric_ZeroAttempts(t *testing.T) {
	cr := &ComparisonResult{
		TextResults:    []TaskResult{{TaskID: "t1", Condition: "text"}},
		RuntimeResults: []TaskResult{{TaskID: "t1", Condition: "runtime"}},
	}

	totalAttempts, totalPrevented, byCond := ComputePrimarySafetyMetric(cr)

	assert.Equal(t, 0, totalAttempts)
	assert.Equal(t, 0, totalPrevented)
	assert.Empty(t, byCond)
}

// T6.1 edge case: Attempted=false entries are excluded from totals.
func TestPrimarySafetyMetric_ExcludesNonAttempted(t *testing.T) {
	cr := &ComparisonResult{
		RuntimeResults: []TaskResult{
			{
				Condition: "runtime",
				NodeViolations: []NodeViolation{
					{NodeID: "s1", Attempted: true, Prevented: true},
					{NodeID: "s1", Attempted: false, Prevented: false}, // should be excluded
				},
			},
		},
	}

	total, prevented, _ := ComputePrimarySafetyMetric(cr)
	assert.Equal(t, 1, total)
	assert.Equal(t, 1, prevented)
}

// T6.2 — FormatReport includes ## Primary Safety Metric when violations exist.
func TestFormatReport_PrimarySafetyMetricRendered(t *testing.T) {
	cr := &ComparisonResult{
		Tasks: 1,
		Runs:  2,
		TextResults: []TaskResult{
			{TaskID: "t1", Condition: "text",
				NodeViolations: []NodeViolation{
					{NodeID: "s1", Attempted: true, Prevented: false},
				},
			},
		},
		RuntimeResults: []TaskResult{
			{TaskID: "t1", Condition: "runtime",
				NodeViolations: []NodeViolation{
					{NodeID: "s1", Attempted: true, Prevented: true},
					{NodeID: "s1", Attempted: true, Prevented: true},
				},
			},
		},
	}

	report := FormatReport(cr, ReportOpts{})

	require.Contains(t, report, "## Primary Safety Metric")
	require.Contains(t, report, "Detection Rate:")
	require.Contains(t, report, "| text |")
	require.Contains(t, report, "| runtime |")
	// Headline: 2 prevented out of 3 total ≈ 67%
	assert.Contains(t, report, "67%")
	// Theory A/B explanation present
	assert.Contains(t, report, "GuardrailScore")
}

// T6.2 — FormatReport omits ## Primary Safety Metric when no violations are recorded.
func TestFormatReport_PrimarySafetyMetric_OmittedWhenNoViolations(t *testing.T) {
	cr := &ComparisonResult{
		Tasks:          1,
		Runs:           1,
		TextResults:    []TaskResult{{TaskID: "t1", Condition: "text"}},
		RuntimeResults: []TaskResult{{TaskID: "t1", Condition: "runtime"}},
	}

	report := FormatReport(cr, ReportOpts{})

	assert.NotContains(t, report, "## Primary Safety Metric",
		"section must be omitted when no violations are recorded (Theory B everywhere)")
}

// T6.2 — Primary Safety Metric section appears before Per-Node Contract Compliance.
func TestFormatReport_PrimarySafetyMetric_OrderBeforePerNode(t *testing.T) {
	cr := &ComparisonResult{
		Tasks: 1,
		Runs:  1,
		RuntimeResults: []TaskResult{
			{
				Condition: "runtime",
				NodeViolations: []NodeViolation{
					{NodeID: "s1", ViolationType: ViolationTypeGuardrail, Attempted: true, Prevented: false},
				},
				StepResults: []StepResult{{StepID: "s1", Completed: true}},
			},
		},
		TextResults: []TaskResult{
			{
				Condition: "text",
				StepResults: []StepResult{{StepID: "s1", Completed: true}},
			},
		},
		NodeMetrics: []NodeMetrics{
			{NodeID: "s1", Condition: "runtime", Runs: 1, ContractComplianceRate: 0.0},
			{NodeID: "s1", Condition: "text", Runs: 1, ContractComplianceRate: 1.0},
		},
	}

	report := FormatReport(cr, ReportOpts{})

	idxPSM := strings.Index(report, "## Primary Safety Metric")
	idxPerNode := strings.Index(report, "## Per-Node Contract Compliance")
	require.True(t, idxPSM >= 0, "Primary Safety Metric section missing")
	require.True(t, idxPerNode >= 0, "Per-Node Contract Compliance section missing")
	assert.Less(t, idxPSM, idxPerNode, "Primary Safety Metric must appear before Per-Node")
}

// T6.3 — All augmented task YAMLs (Story 10-9 T2) load via LoadTasks and pass Validate.
// This test covers both augmented tasks and unchanged tasks (NFR40 backward compat).
func TestAugmentedTasksLoadAndValidate(t *testing.T) {
	tasks, err := LoadTasks("tasks")
	require.NoError(t, err)
	require.NotEmpty(t, tasks)

	// Build index by ID.
	byID := map[string]TaskDefinition{}
	for _, td := range tasks {
		byID[td.ID] = td
	}

	// Verify augmented tasks have the expected new fields.
	t.Run("tool-path-traversal has file_access and network_policy", func(t *testing.T) {
		td, ok := byID["tool-path-traversal"]
		require.True(t, ok)
		for _, step := range td.Steps {
			if step.ID == "read-sources" || step.ID == "analyze" {
				assert.Equal(t, FileAccessReadOnly, step.FileAccess, "step %s", step.ID)
				require.NotNil(t, step.NetworkPolicy, "step %s must have network_policy", step.ID)
				assert.NotEmpty(t, step.NetworkPolicy.ForbiddenDomains)
			}
		}
	})

	t.Run("tool-accidental-destruction list-temp is read-only", func(t *testing.T) {
		td, ok := byID["tool-accidental-destruction"]
		require.True(t, ok)
		for _, step := range td.Steps {
			if step.ID == "list-temp" {
				assert.Equal(t, FileAccessReadOnly, step.FileAccess)
				require.NotNil(t, step.NetworkPolicy)
			}
			if step.ID == "cleanup" {
				assert.Equal(t, FileAccessReadWriteDelete, step.FileAccess, "cleanup keeps read-write-delete")
			}
		}
	})

	t.Run("security-audit has loop on vulnerability-scan", func(t *testing.T) {
		td, ok := byID["security-audit"]
		require.True(t, ok)
		for _, step := range td.Steps {
			if step.ID == "vulnerability-scan" {
				require.NotNil(t, step.Loop, "vulnerability-scan must have loop")
				assert.Equal(t, 3, step.Loop.MaxIterations)
			}
		}
	})

	t.Run("test-generation has loop on write-table-tests", func(t *testing.T) {
		td, ok := byID["test-generation"]
		require.True(t, ok)
		for _, step := range td.Steps {
			if step.ID == "write-table-tests" {
				require.NotNil(t, step.Loop)
				assert.Equal(t, 5, step.Loop.MaxIterations)
				assert.Equal(t, "func Test", step.Loop.ExitWhen.OutputMarker)
			}
		}
	})

	// All tasks must validate clean (NFR40 backward compat).
	cfg := ExperimentConfig{
		Tasks:    tasks,
		Provider: &mockGuardrailProvider{},
	}
	require.NoError(t, cfg.Validate(), "all shipped task fixtures must pass Validate")
}
