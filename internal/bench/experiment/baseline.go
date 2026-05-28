package experiment

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// BaselineSnapshot holds a point-in-time capture of per-node compliance rates,
// used for delta computation in the per-node report section (Story 10-8).
//
// Workflow:
//
//	# First run (no structural enforcement):
//	forgent-bench bench tasks/ --runs 3 --save-baseline baseline.json
//
//	# Second run (with enforcement):
//	forgent-bench bench tasks/ --runs 3 --baseline baseline.json
//
// The snapshot is a local artifact and must NOT be committed to version control
// (per the project's no-datasets-in-git rule).
//
// Precedence for the baseline path: CLI flag --baseline > BENCH_BASELINE env var > none.
type BaselineSnapshot struct {
	CapturedAt          string                        `json:"captured_at"`           // ISO date (YYYY-MM-DD)
	Description         string                        `json:"description"`           // e.g. "keyword-guardrail baseline"
	NodeComplianceRates map[string]map[string]float64 `json:"node_compliance_rates"` // node_id → condition → ContractComplianceRate
}

// LoadBaseline reads and validates a baseline snapshot from path.
//
// Returns (nil, err) for missing files, unreadable paths, malformed JSON, or
// an empty NodeComplianceRates map. The caller is responsible for graceful
// degradation (AC #5): do not fail the experiment on a missing baseline.
func LoadBaseline(path string) (*BaselineSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var bs BaselineSnapshot
	if err := json.Unmarshal(data, &bs); err != nil {
		return nil, fmt.Errorf("baseline %s: malformed JSON: %w", path, err)
	}
	if len(bs.NodeComplianceRates) == 0 {
		return nil, fmt.Errorf("baseline %s: NodeComplianceRates is empty", path)
	}
	return &bs, nil
}

// SaveBaseline captures the current ComparisonResult.NodeMetrics into a
// BaselineSnapshot at path. Existing files are overwritten.
func SaveBaseline(path string, cr *ComparisonResult, description string) error {
	rates := make(map[string]map[string]float64, len(cr.NodeMetrics))
	for _, nm := range cr.NodeMetrics {
		if rates[nm.NodeID] == nil {
			rates[nm.NodeID] = make(map[string]float64)
		}
		rates[nm.NodeID][nm.Condition] = nm.ContractComplianceRate
	}
	bs := BaselineSnapshot{
		CapturedAt:          time.Now().Format("2006-01-02"),
		Description:         description,
		NodeComplianceRates: rates,
	}
	data, err := json.MarshalIndent(bs, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal baseline: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// WriteBaselineSnapshot writes a pre-built BaselineSnapshot directly to path.
// Used by convert-baseline to persist a snapshot derived from legacy JSON files
// rather than from a live ComparisonResult.
func WriteBaselineSnapshot(path string, bs BaselineSnapshot) error {
	data, err := json.MarshalIndent(bs, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal baseline snapshot: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// ApplyBaseline computes baseline deltas and patches NodeMetrics.BaselineDelta
// for each (node, condition) pair found in the snapshot. If bs is nil, this is
// a no-op (supports AC #5 graceful degradation).
func ApplyBaseline(nms []NodeMetrics, bs *BaselineSnapshot) {
	if bs == nil {
		return
	}
	for i := range nms {
		nm := &nms[i]
		condRates, ok := bs.NodeComplianceRates[nm.NodeID]
		if !ok {
			continue
		}
		baseRate, ok := condRates[nm.Condition]
		if !ok {
			continue
		}
		delta := nm.ContractComplianceRate - baseRate
		nm.BaselineDelta = &delta
	}
}
