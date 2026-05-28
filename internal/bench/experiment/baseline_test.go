package experiment

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoadBaseline_RoundTrip — T6.5: SaveBaseline then LoadBaseline returns equivalent data.
func TestLoadBaseline_RoundTrip(t *testing.T) {
	cr := &ComparisonResult{
		NodeMetrics: []NodeMetrics{
			{NodeID: "s1", Condition: "text", Runs: 3, ContractComplianceRate: 0.8},
			{NodeID: "s1", Condition: "runtime", Runs: 3, ContractComplianceRate: 1.0},
			{NodeID: "s2", Condition: "runtime", Runs: 3, ContractComplianceRate: 0.5},
		},
	}

	path := filepath.Join(t.TempDir(), "baseline.json")
	require.NoError(t, SaveBaseline(path, cr, "test-baseline"))

	bs, err := LoadBaseline(path)
	require.NoError(t, err)
	require.NotNil(t, bs)

	assert.Equal(t, "test-baseline", bs.Description)
	assert.NotEmpty(t, bs.CapturedAt)

	require.Contains(t, bs.NodeComplianceRates, "s1")
	assert.InDelta(t, 0.8, bs.NodeComplianceRates["s1"]["text"], 1e-9)
	assert.InDelta(t, 1.0, bs.NodeComplianceRates["s1"]["runtime"], 1e-9)
	assert.InDelta(t, 0.5, bs.NodeComplianceRates["s2"]["runtime"], 1e-9)
}

// TestLoadBaseline_MissingFileReturnsError — T6.6
func TestLoadBaseline_MissingFileReturnsError(t *testing.T) {
	bs, err := LoadBaseline("/nonexistent/path/baseline.json")
	assert.Nil(t, bs)
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err), "error should satisfy os.IsNotExist")
}

// TestLoadBaseline_MalformedJSONReturnsError — T6.7
func TestLoadBaseline_MalformedJSONReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	require.NoError(t, os.WriteFile(path, []byte("not json at all {{{"), 0644))

	bs, err := LoadBaseline(path)
	assert.Nil(t, bs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "malformed JSON")
}

// TestLoadBaseline_EmptyRatesReturnsError — empty NodeComplianceRates is invalid.
func TestLoadBaseline_EmptyRatesReturnsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"captured_at":"2026-01-01","description":"x","node_compliance_rates":{}}`), 0644))

	bs, err := LoadBaseline(path)
	assert.Nil(t, bs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "NodeComplianceRates is empty")
}

// TestApplyBaseline_SetsDelta verifies BaselineDelta is patched correctly.
func TestApplyBaseline_SetsDelta(t *testing.T) {
	bs := &BaselineSnapshot{
		NodeComplianceRates: map[string]map[string]float64{
			"s1": {"runtime": 0.6},
		},
	}
	nms := []NodeMetrics{
		{NodeID: "s1", Condition: "runtime", ContractComplianceRate: 0.9},
		{NodeID: "s2", Condition: "runtime", ContractComplianceRate: 0.8}, // not in baseline
	}
	ApplyBaseline(nms, bs)

	require.NotNil(t, nms[0].BaselineDelta)
	assert.InDelta(t, 0.3, *nms[0].BaselineDelta, 1e-9, "delta = current - baseline = 0.9 - 0.6")
	assert.Nil(t, nms[1].BaselineDelta, "node not in baseline stays nil")
}

// TestApplyBaseline_NilIsNoop — nil baseline must not panic or modify anything.
func TestApplyBaseline_NilIsNoop(t *testing.T) {
	nms := []NodeMetrics{
		{NodeID: "s1", Condition: "runtime", ContractComplianceRate: 0.9},
	}
	ApplyBaseline(nms, nil) // must not panic
	assert.Nil(t, nms[0].BaselineDelta)
}
