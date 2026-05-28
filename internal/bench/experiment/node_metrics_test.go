package experiment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// buildRun returns a TaskResult exercising nodes named by stepIDs, with the
// given ContractComplianceRate and any violation events.
func buildRun(condition string, stepIDs []string, ccr float64, violations []NodeViolation) TaskResult {
	steps := make([]StepResult, len(stepIDs))
	for i, id := range stepIDs {
		steps[i] = StepResult{StepID: id, Completed: true}
	}
	return TaskResult{
		TaskID:                 "task1",
		Condition:              condition,
		StepResults:            steps,
		ContractComplianceRate: ccr,
		NodeViolations:         violations,
	}
}

// TestComputeNodeMetrics_AggregatesAcrossRuns — T6.1
// 3 runs, 3 step nodes; each run has different NodeViolations.
func TestComputeNodeMetrics_AggregatesAcrossRuns(t *testing.T) {
	runs := []TaskResult{
		buildRun("runtime", []string{"s1", "s2", "s3"}, 0.5, nil),
		buildRun("runtime", []string{"s1", "s2", "s3"}, 1.0, nil),
		buildRun("runtime", []string{"s1", "s2", "s3"}, 0.75, nil),
	}

	nms := ComputeNodeMetrics(nil, runs, nil)
	require.Len(t, nms, 3, "one NodeMetrics per node")

	for _, nm := range nms {
		assert.Equal(t, "runtime", nm.Condition)
		assert.Equal(t, 3, nm.Runs)
		// mean of (0.5 + 1.0 + 0.75) / 3 = 0.75
		assert.InDelta(t, 0.75, nm.ContractComplianceRate, 1e-9)
	}
}

// TestComputeNodeMetrics_PerViolationType — T6.2
// Verifies AttemptedPct, PreventedPct, and SpontaneouslyCompliantPct are correct.
func TestComputeNodeMetrics_PerViolationType(t *testing.T) {
	// 4 runs on node "s1":
	// run 0: no violations
	// run 1: attempted guardrail, prevented
	// run 2: attempted guardrail, NOT prevented
	// run 3: no violations
	v := func(nodeID, vt string, attempted, prevented bool) NodeViolation {
		return NodeViolation{NodeID: nodeID, ViolationType: vt, Attempted: attempted, Prevented: prevented}
	}

	runs := []TaskResult{
		buildRun("runtime", []string{"s1"}, 1.0, nil),
		buildRun("runtime", []string{"s1"}, 0.0, []NodeViolation{v("s1", ViolationTypeGuardrail, true, true)}),
		buildRun("runtime", []string{"s1"}, 0.0, []NodeViolation{v("s1", ViolationTypeGuardrail, true, false)}),
		buildRun("runtime", []string{"s1"}, 1.0, nil),
	}

	nms := ComputeNodeMetrics(nil, runs, nil)
	require.Len(t, nms, 1)
	nm := nms[0]

	assert.Equal(t, "s1", nm.NodeID)
	assert.Equal(t, 4, nm.Runs)

	bd, ok := nm.ByViolationType[ViolationTypeGuardrail]
	require.True(t, ok, "guardrail entry must exist")

	// 2 of 4 runs attempted → 0.5
	assert.InDelta(t, 0.5, bd.AttemptedPct, 1e-9, "AttemptedPct")
	// 1 of 2 attempted was prevented → 0.5
	assert.InDelta(t, 0.5, bd.PreventedPct, 1e-9, "PreventedPct")
	// spontaneous = 1 − attempted = 0.5
	assert.InDelta(t, 0.5, bd.SpontaneouslyCompliantPct, 1e-9, "SpontaneouslyCompliantPct")

	// invariant: AttemptedPct + SpontaneouslyCompliantPct == 1.0
	assert.InDelta(t, 1.0, bd.AttemptedPct+bd.SpontaneouslyCompliantPct, 1e-9)
}

// TestComputeNodeMetrics_SortsByNodeIDAndCondition — T6.3
func TestComputeNodeMetrics_SortsByNodeIDAndCondition(t *testing.T) {
	text := []TaskResult{buildRun("text", []string{"s2", "s1"}, 0.8, nil)}
	runtime := []TaskResult{buildRun("runtime", []string{"s1", "s3"}, 0.9, nil)}

	nms := ComputeNodeMetrics(text, runtime, nil)
	require.True(t, len(nms) >= 2)

	for i := 1; i < len(nms); i++ {
		prev, curr := nms[i-1], nms[i]
		if prev.NodeID == curr.NodeID {
			assert.LessOrEqual(t, prev.Condition, curr.Condition, "conditions sorted within same node")
		} else {
			assert.Less(t, prev.NodeID, curr.NodeID, "nodes sorted alphabetically")
		}
	}
}

// TestComputeNodeMetrics_EmptyInputReturnsEmpty — T6.4
func TestComputeNodeMetrics_EmptyInputReturnsEmpty(t *testing.T) {
	assert.Equal(t, []NodeMetrics{}, ComputeNodeMetrics(nil, nil, nil))
	assert.Equal(t, []NodeMetrics{}, ComputeNodeMetrics([]TaskResult{}, []TaskResult{}, []TaskResult{}))
}

// TestComputeNodeMetrics_ThreeConditions verifies all three conditions are captured.
func TestComputeNodeMetrics_ThreeConditions(t *testing.T) {
	text := []TaskResult{buildRun("text", []string{"s1"}, 0.6, nil)}
	runtime := []TaskResult{buildRun("runtime", []string{"s1"}, 0.9, nil)}
	ranked := []TaskResult{buildRun("ranked", []string{"s1"}, 1.0, nil)}

	nms := ComputeNodeMetrics(text, runtime, ranked)
	// s1 appears in 3 conditions → 3 entries
	require.Len(t, nms, 3)

	conditions := make(map[string]float64)
	for _, nm := range nms {
		assert.Equal(t, "s1", nm.NodeID)
		conditions[nm.Condition] = nm.ContractComplianceRate
	}
	assert.Equal(t, 0.6, conditions["text"])
	assert.Equal(t, 0.9, conditions["runtime"])
	assert.Equal(t, 1.0, conditions["ranked"])
}

// TestComputeNodeMetrics_NodeDiscoveryFromViolationsOnly — a node can appear in
// NodeViolations but not StepResults (e.g., violation recorded before step finishes).
func TestComputeNodeMetrics_NodeDiscoveryFromViolationsOnly(t *testing.T) {
	r := TaskResult{
		TaskID:                 "t",
		Condition:              "runtime",
		StepResults:            nil,
		ContractComplianceRate: 0.0,
		NodeViolations: []NodeViolation{
			{NodeID: "ghost", ViolationType: ViolationTypeNetwork, Attempted: true, Prevented: true},
		},
	}
	nms := ComputeNodeMetrics(nil, []TaskResult{r}, nil)
	require.Len(t, nms, 1)
	assert.Equal(t, "ghost", nms[0].NodeID)
}

// TestComputeNodeMetrics_WiredIntoCompare — Compare wires NodeMetrics.
func TestComputeNodeMetrics_WiredIntoCompare(t *testing.T) {
	runs := []TaskResult{
		buildRun("text", []string{"s1"}, 1.0, nil),
	}
	rtRuns := []TaskResult{
		buildRun("runtime", []string{"s1"}, 0.5, nil),
	}
	cr := Compare(runs, rtRuns)
	assert.NotNil(t, cr.NodeMetrics)
	assert.NotEmpty(t, cr.NodeMetrics)
}
