package doctor

import (
	"testing"

	"github.com/mirandaguillaume/forgent/internal/doctor/llmutil"
	"github.com/stretchr/testify/assert"
)

func TestGate_AllPass(t *testing.T) {
	gate := DefaultGate()
	structural := StructuralResult{Passed: 14, Total: 16}
	findings := []llmutil.Finding{
		{Category: "ordering", Issue: "Bad ordering", Confidence: "moderate"},
	}

	result := gate.Evaluate(structural, findings)
	assert.True(t, result.Pass)
	assert.Empty(t, result.Failures)
}

func TestGate_SecretsFail(t *testing.T) {
	gate := DefaultGate()
	structural := StructuralResult{Passed: 14, Total: 16}
	findings := []llmutil.Finding{
		{Category: "security", Issue: "Possible OpenAI API key detected on line 5", Confidence: "high"},
	}

	result := gate.Evaluate(structural, findings)
	assert.False(t, result.Pass, "secrets should fail gate")
	assert.Len(t, result.Failures, 1)
	assert.Contains(t, result.Failures[0], "secrets_found")
}

func TestGate_LowStructuralFail(t *testing.T) {
	gate := DefaultGate()
	structural := StructuralResult{Passed: 3, Total: 16} // 18.75%
	findings := []llmutil.Finding{}

	result := gate.Evaluate(structural, findings)
	assert.False(t, result.Pass, "low structural score should fail gate")
	assert.Len(t, result.Failures, 1)
	assert.Contains(t, result.Failures[0], "structural_pct")
}

func TestGate_HighFindingsWarning(t *testing.T) {
	gate := DefaultGate()
	structural := StructuralResult{Passed: 14, Total: 16}
	findings := make([]llmutil.Finding, 0, 6)
	for i := 0; i < 6; i++ {
		findings = append(findings, llmutil.Finding{Category: "specificity", Issue: "Vague", Confidence: "high"})
	}

	result := gate.Evaluate(structural, findings)
	assert.True(t, result.Pass, "high_findings is non-blocking — gate should pass")
	assert.Len(t, result.Warnings, 1)
	assert.Contains(t, result.Warnings[0], "high_findings")
}

func TestGate_CustomConditions(t *testing.T) {
	gate := &QualityGate{
		Conditions: []GateCondition{
			{Metric: "total_findings", Operator: "<=", Threshold: 3, Blocking: true},
		},
	}
	structural := StructuralResult{Passed: 10, Total: 16}
	findings := make([]llmutil.Finding, 5)

	result := gate.Evaluate(structural, findings)
	assert.False(t, result.Pass)
	assert.Contains(t, result.Failures[0], "total_findings")
}
