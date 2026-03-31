package doctor

import (
	"fmt"
	"strings"

	"github.com/mirandaguillaume/forgent/internal/doctor/llmutil"
)

// GateCondition defines a single quality gate condition.
type GateCondition struct {
	Metric   string  `yaml:"metric" json:"metric"`
	Operator string  `yaml:"operator" json:"operator"`
	Threshold float64 `yaml:"threshold" json:"threshold"`
	Blocking bool    `yaml:"blocking" json:"blocking"`
}

// QualityGate holds conditions that must be satisfied for the gate to pass.
type QualityGate struct {
	Conditions []GateCondition
}

// GateResult holds the outcome of a quality gate evaluation.
type GateResult struct {
	Pass     bool     `json:"pass"`
	Failures []string `json:"failures,omitempty"`
	Warnings []string `json:"warnings,omitempty"`
}

// DefaultGate returns the built-in quality gate conditions.
func DefaultGate() *QualityGate {
	return &QualityGate{
		Conditions: []GateCondition{
			{Metric: "secrets_found", Operator: "==", Threshold: 0, Blocking: true},
			{Metric: "structural_pct", Operator: ">=", Threshold: 50, Blocking: true},
			{Metric: "high_findings", Operator: "<=", Threshold: 5, Blocking: false},
		},
	}
}

// Evaluate checks all conditions against the report metrics.
// Gate passes iff ALL blocking conditions pass.
func (g *QualityGate) Evaluate(structural StructuralResult, findings []llmutil.Finding) GateResult {
	metrics := computeMetrics(structural, findings)
	result := GateResult{Pass: true}

	for _, cond := range g.Conditions {
		val, ok := metrics[cond.Metric]
		if !ok {
			msg := fmt.Sprintf("unknown metric %q — condition skipped", cond.Metric)
			result.Warnings = append(result.Warnings, msg)
			continue
		}

		passed := evalCondition(val, cond.Operator, cond.Threshold)
		if !passed {
			msg := fmt.Sprintf("%s %s %.0f (actual: %.0f)", cond.Metric, cond.Operator, cond.Threshold, val)
			if cond.Blocking {
				result.Pass = false
				result.Failures = append(result.Failures, msg)
			} else {
				result.Warnings = append(result.Warnings, msg)
			}
		}
	}

	return result
}

func computeMetrics(structural StructuralResult, findings []llmutil.Finding) map[string]float64 {
	secretsFound := 0
	highFindings := 0
	for _, f := range findings {
		if f.Category == "security" && strings.Contains(f.Issue, "detected") {
			secretsFound++
		}
		if strings.ToLower(f.Confidence) == "high" {
			highFindings++
		}
	}

	pct := float64(0)
	if structural.Total > 0 {
		pct = float64(structural.Passed) * 100 / float64(structural.Total)
	}

	return map[string]float64{
		"secrets_found":  float64(secretsFound),
		"structural_pct": pct,
		"high_findings":  float64(highFindings),
		"total_findings": float64(len(findings)),
	}
}

func evalCondition(value float64, operator string, threshold float64) bool {
	switch operator {
	case "==":
		return value == threshold
	case "!=":
		return value != threshold
	case ">=":
		return value >= threshold
	case "<=":
		return value <= threshold
	case ">":
		return value > threshold
	case "<":
		return value < threshold
	default:
		return false // unknown operator — fail-closed for safety
	}
}
