package doctor

import (
	"github.com/mirandaguillaume/forgent/internal/doctor/llmutil"
	"github.com/mirandaguillaume/forgent/internal/doctor/parser"
	"github.com/mirandaguillaume/forgent/internal/doctor/static"
)

// PipelineOpts configures the doctor pipeline.
type PipelineOpts struct {
	Mode         string // "default", "quick", "thorough", "security"
	MaxFindings  int
	Gate         *QualityGate
	SectionCount int // number of section categories for structural score
}

// Report holds the complete result of a doctor pipeline run.
type Report struct {
	FilePath        string
	Format          string
	StructuralScore StructuralResult
	Indicators      []static.Indicator
	StaticFindings  []llmutil.Finding
	LLMFindings     []llmutil.Finding
	AllFindings     []llmutil.Finding // merged + post-processed
	GateResult      GateResult
}

// RunPipeline executes the full doctor pipeline: static → merge → post-process → gate.
// LLM findings are passed in separately (caller handles LLM call + cache).
func RunPipeline(analysis *parser.AgentAnalysis, llmFindings []llmutil.Finding, opts PipelineOpts) *Report {
	if opts.Gate == nil {
		opts.Gate = DefaultGate()
	}
	if opts.SectionCount <= 0 {
		opts.SectionCount = 15
	}

	report := &Report{
		Format: analysis.Format,
	}

	// Phase 1: Static checks
	report.StaticFindings = static.RunChecks(analysis, opts.Mode)
	report.StructuralScore = ComputeStructural(report.StaticFindings, opts.SectionCount)

	// Phase 1b: Indicators
	report.Indicators = static.CollectIndicators(analysis, opts.Mode)

	// Phase 2: Merge static + LLM
	report.LLMFindings = llmFindings
	var merged []llmutil.Finding
	merged = append(merged, report.StaticFindings...)
	merged = append(merged, report.LLMFindings...)

	// Phase 3: Quality gate (evaluate BEFORE truncation)
	report.GateResult = opts.Gate.Evaluate(report.StructuralScore, merged)

	// Phase 4: Post-process (dedup, sort, limit for display)
	report.AllFindings = PostProcess(merged, opts.MaxFindings)

	return report
}
