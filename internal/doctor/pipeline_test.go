package doctor

import (
	"testing"

	"github.com/mirandaguillaume/forgent/internal/doctor/llmutil"
	"github.com/mirandaguillaume/forgent/internal/doctor/parser"
	"github.com/stretchr/testify/assert"
)

func TestRunPipeline_QuickMode(t *testing.T) {
	analysis := &parser.AgentAnalysis{
		Format:     "claude",
		RawContent: []byte("## Rules\nBe nice"),
		Sections:   []parser.Section{{Header: "Rules", Level: 2}},
	}

	report := RunPipeline(analysis, nil, PipelineOpts{Mode: "quick", SectionCount: 15})

	assert.Equal(t, "claude", report.Format)
	assert.True(t, len(report.StaticFindings) > 0, "quick mode should produce static findings")
	assert.Empty(t, report.LLMFindings)
	assert.True(t, report.StructuralScore.Total > 0)
}

func TestRunPipeline_WithLLMFindings(t *testing.T) {
	analysis := &parser.AgentAnalysis{
		Format:     "claude",
		RawContent: []byte("## Security\nRestricted.\n## Rules\nBe nice"),
		Sections: []parser.Section{
			{Header: "Security", Level: 2},
			{Header: "Rules", Level: 2},
		},
	}

	llmFindings := []llmutil.Finding{
		{Category: "specificity", Issue: "Vague instruction", Confidence: "moderate"},
	}

	report := RunPipeline(analysis, llmFindings, PipelineOpts{Mode: "default", SectionCount: 15})

	assert.True(t, len(report.AllFindings) > 0)
	// Should contain both static and LLM findings
	hasLLM := false
	for _, f := range report.AllFindings {
		if f.Issue == "Vague instruction" {
			hasLLM = true
		}
	}
	assert.True(t, hasLLM, "merged findings should include LLM findings")
}

func TestRunPipeline_GatePass(t *testing.T) {
	analysis := &parser.AgentAnalysis{
		Format: "claude",
		RawContent: []byte("## Identity\nYou are a reviewer.\n## Security\nRestricted.\n## Guardrails\nTimeout 5min.\n## Testing\nRun go test.\n## Examples\nHere.\n## Error Handling\nRetry.\n## Build\ngo build.\n## Architecture\nLayered.\n## Output\nJSON.\n## Decision\nEscalate.\n## Trigger\nOn PR.\n## Dependencies\nInput/output.\n## Memory\nSession.\n## Goals\nQuality.\n## Constraints\nNever delete."),
		Sections: []parser.Section{
			{Header: "Identity", Level: 2}, {Header: "Security", Level: 2},
			{Header: "Guardrails", Level: 2}, {Header: "Testing", Level: 2},
			{Header: "Examples", Level: 2}, {Header: "Error Handling", Level: 2},
			{Header: "Build", Level: 2}, {Header: "Architecture", Level: 2},
			{Header: "Output", Level: 2}, {Header: "Decision", Level: 2},
			{Header: "Trigger", Level: 2}, {Header: "Dependencies", Level: 2},
			{Header: "Memory", Level: 2}, {Header: "Goals", Level: 2},
			{Header: "Constraints", Level: 2},
		},
	}

	report := RunPipeline(analysis, nil, PipelineOpts{Mode: "quick", SectionCount: 15})
	assert.True(t, report.GateResult.Pass, "all sections present, no secrets → gate should pass")
}

func TestRunPipeline_DefaultGate(t *testing.T) {
	report := RunPipeline(
		&parser.AgentAnalysis{Format: "claude", RawContent: []byte("x")},
		nil,
		PipelineOpts{Mode: "quick"},
	)
	// Default gate should be applied
	assert.NotNil(t, report.GateResult)
}
