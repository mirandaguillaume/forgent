package experiment

import (
	"fmt"
	"strings"
	"time"
)

// ReportOpts configures report formatting.
type ReportOpts struct {
	Model   string
	Version string
}

// FormatReport produces a Markdown report from experiment results.
func FormatReport(cr *ComparisonResult, opts ReportOpts) string {
	var sb strings.Builder

	sb.WriteString("# Runtime vs Text Experiment Report\n\n")

	// Metadata
	sb.WriteString("## Experiment Parameters\n\n")
	sb.WriteString("| Parameter | Value |\n")
	sb.WriteString("|-----------|-------|\n")
	sb.WriteString(fmt.Sprintf("| Date | %s |\n", time.Now().Format("2006-01-02")))
	if opts.Model != "" {
		sb.WriteString(fmt.Sprintf("| Model | %s |\n", opts.Model))
	}
	sb.WriteString(fmt.Sprintf("| Tasks | %d |\n", cr.Tasks))
	sb.WriteString(fmt.Sprintf("| Total runs | %d |\n", cr.Runs))
	sb.WriteString(fmt.Sprintf("| Bootstrap resamples | 1000 |\n"))
	sb.WriteString(fmt.Sprintf("| Confidence level | 95%% |\n"))
	sb.WriteString("\n")

	// Methodology reference
	hasRanked := len(cr.RankedResults) > 0
	sb.WriteString("## Methodology\n\n")
	if hasRanked {
		sb.WriteString("Each task is executed under three conditions:\n\n")
	} else {
		sb.WriteString("Each task is executed under two conditions:\n\n")
	}
	sb.WriteString("1. **Text instruction**: All steps given as a single monolithic prompt. ")
	sb.WriteString("The LLM must follow the step sequence, respect guardrails, and produce all outputs in one response.\n")
	sb.WriteString("2. **Runtime (DAG)**: Steps are encoded as DAG nodes with consumes/produces contracts. ")
	sb.WriteString("The Forgent engine enforces ordering, parallelizes independent steps, and provides per-step isolation.\n")
	if hasRanked {
		sb.WriteString("3. **Ranked (DAG + BM25)**: Runtime condition with BM25-ranked context excerpts pre-injected into each step prompt. ")
		sb.WriteString("Reduces file_read round-trips while keeping the full file available on demand.\n")
	}
	sb.WriteString("\nThe only variable is orchestration method. Same LLM, same model, same task definitions.\n\n")

	// Aggregate results
	sb.WriteString("## Aggregate Results\n\n")
	if hasRanked {
		sb.WriteString("| Metric | Text | Runtime | Ranked | RT Delta | Ranked Delta |\n")
		sb.WriteString("|--------|------|---------|--------|----------|-------------|\n")

		textComp := completionRate(cr.TextResults)
		rtComp := completionRate(cr.RuntimeResults)
		rkComp := completionRate(cr.RankedResults)
		sb.WriteString(fmt.Sprintf("| Completion rate | %.0f%% | %.0f%% | %.0f%% | %+.0f%% | %+.0f%% |\n",
			textComp*100, rtComp*100, rkComp*100,
			cr.CompletionDelta*100, (rkComp-textComp)*100))

		textOrd := meanOrdering(cr.TextResults)
		rtOrd := meanOrdering(cr.RuntimeResults)
		rkOrd := meanOrdering(cr.RankedResults)
		sb.WriteString(fmt.Sprintf("| Step ordering | %.0f%% | %.0f%% | %.0f%% | %+.0f%% | %+.0f%% |\n",
			textOrd*100, rtOrd*100, rkOrd*100,
			cr.OrderingDelta*100, (rkOrd-textOrd)*100))

		textGR := meanGuardrail(cr.TextResults)
		rtGR := meanGuardrail(cr.RuntimeResults)
		rkGR := meanGuardrail(cr.RankedResults)
		sb.WriteString(fmt.Sprintf("| Guardrail adherence | %.0f%% | %.0f%% | %.0f%% | %+.0f%% | %+.0f%% |\n",
			textGR*100, rtGR*100, rkGR*100,
			cr.GuardrailDelta*100, (rkGR-textGR)*100))

		textTok := meanTokens(cr.TextResults)
		rtTok := meanTokens(cr.RuntimeResults)
		rkTok := meanTokens(cr.RankedResults)
		sb.WriteString(fmt.Sprintf("| Token usage | %.0f | %.0f | %.0f | %+.0f | %+.0f |\n",
			textTok, rtTok, rkTok,
			cr.TokenDelta, rkTok-textTok))

		textCalls := meanLLMCalls(cr.TextResults)
		rtCalls := meanLLMCalls(cr.RuntimeResults)
		rkCalls := meanLLMCalls(cr.RankedResults)
		sb.WriteString(fmt.Sprintf("| LLM calls | %.0f | %.0f | %.0f | %+.0f | %+.0f |\n",
			textCalls, rtCalls, rkCalls,
			cr.LLMCallsDelta, rkCalls-textCalls))

		sb.WriteString("\n")
		sb.WriteString("*RT Delta = runtime − text. Ranked Delta = ranked − text. Negative token delta = more efficient.*\n\n")
	} else {
		sb.WriteString("| Metric | Text | Runtime | Delta | 95% CI |\n")
		sb.WriteString("|--------|------|---------|-------|--------|\n")

		textComp := completionRate(cr.TextResults)
		rtComp := completionRate(cr.RuntimeResults)
		sb.WriteString(fmt.Sprintf("| Completion rate | %.0f%% | %.0f%% | %+.0f%% | [%+.1f%%, %+.1f%%] |\n",
			textComp*100, rtComp*100, cr.CompletionDelta*100, cr.CompletionCI[0]*100, cr.CompletionCI[1]*100))

		textOrd := meanOrdering(cr.TextResults)
		rtOrd := meanOrdering(cr.RuntimeResults)
		sb.WriteString(fmt.Sprintf("| Step ordering | %.0f%% | %.0f%% | %+.0f%% | [%+.1f%%, %+.1f%%] |\n",
			textOrd*100, rtOrd*100, cr.OrderingDelta*100, cr.OrderingCI[0]*100, cr.OrderingCI[1]*100))

		textGR := meanGuardrail(cr.TextResults)
		rtGR := meanGuardrail(cr.RuntimeResults)
		sb.WriteString(fmt.Sprintf("| Guardrail adherence | %.0f%% | %.0f%% | %+.0f%% | [%+.1f%%, %+.1f%%] |\n",
			textGR*100, rtGR*100, cr.GuardrailDelta*100, cr.GuardrailCI[0]*100, cr.GuardrailCI[1]*100))

		textTok := meanTokens(cr.TextResults)
		rtTok := meanTokens(cr.RuntimeResults)
		sb.WriteString(fmt.Sprintf("| Token usage | %.0f | %.0f | %+.0f | [%+.0f, %+.0f] |\n",
			textTok, rtTok, cr.TokenDelta, cr.TokenCI[0], cr.TokenCI[1]))

		sb.WriteString("\n")
		sb.WriteString("*Positive delta = runtime outperforms text. Negative token delta = runtime is more efficient.*\n\n")
	}

	// Engineering properties
	sb.WriteString("## Engineering Properties\n\n")
	sb.WriteString("These metrics measure structural advantages that text instructions *cannot* provide by construction.\n\n")
	sb.WriteString("| Property | Text | Runtime | Delta | 95% CI |\n")
	sb.WriteString("|----------|------|---------|-------|--------|\n")

	textWC := meanWallClock(cr.TextResults)
	rtWC := meanWallClock(cr.RuntimeResults)
	sb.WriteString(fmt.Sprintf("| Wall-clock time | %.0fms | %.0fms | %+.0fms | [%+.0f, %+.0f] |\n",
		textWC, rtWC, cr.WallClockDelta, cr.WallClockCI[0], cr.WallClockCI[1]))

	textCalls := meanLLMCalls(cr.TextResults)
	rtCalls := meanLLMCalls(cr.RuntimeResults)
	sb.WriteString(fmt.Sprintf("| LLM calls | %.0f | %.0f | %+.0f | |\n",
		textCalls, rtCalls, cr.LLMCallsDelta))

	sb.WriteString(fmt.Sprintf("| Cross-contamination | %.0f%% | 0%% (isolated) | | |\n",
		cr.CrossContamination*100))

	if cr.FaultToleranceText > 0 || cr.FaultToleranceRuntime > 0 {
		sb.WriteString(fmt.Sprintf("| Fault tolerance (20%% error rate) | %.0f%% | %.0f%% | %+.0f%% | |\n",
			cr.FaultToleranceText*100, cr.FaultToleranceRuntime*100,
			(cr.FaultToleranceRuntime-cr.FaultToleranceText)*100))
	}

	sb.WriteString("\n")
	sb.WriteString("*Cross-contamination: Jaccard similarity between outputs of steps that should be independent.*\n")
	sb.WriteString("*Fault tolerance: % of complete runs that succeed when 20% of LLM calls fail randomly.*\n\n")

	// Harness detection rate
	dm := cr.DetectionRate
	if dm.AnyFailures > 0 {
		sb.WriteString("## Harness Detection Rate\n\n")
		sb.WriteString("How many soft-harness (text) failures would the hard-harness (runtime) have caught?\n\n")
		sb.WriteString("| Failure Type | Text Failures | Caught by Runtime | Detection Rate |\n")
		sb.WriteString("|-------------|---------------|-------------------|----------------|\n")
		sb.WriteString(fmt.Sprintf("| Completion | %d | %d | %.0f%% |\n",
			dm.CompletionFailures, dm.CompletionCaught, dm.CompletionRate*100))
		sb.WriteString(fmt.Sprintf("| Ordering | %d | %d | %.0f%% |\n",
			dm.OrderingFailures, dm.OrderingCaught, dm.OrderingRate*100))
		sb.WriteString(fmt.Sprintf("| Guardrails | %d | %d | %.0f%% |\n",
			dm.GuardrailFailures, dm.GuardrailCaught, dm.GuardrailRate*100))
		sb.WriteString(fmt.Sprintf("| **Any failure** | **%d** | **%d** | **%.0f%%** |\n",
			dm.AnyFailures, dm.AnyCaught, dm.AnyRate*100))
		sb.WriteString("\n")
		sb.WriteString("*Detection rate: of all runs where the text condition failed, what % would the runtime harness have caught (i.e., runtime succeeded where text failed).*\n\n")
	}

	// Per-task breakdown
	sb.WriteString("## Per-Task Results\n\n")
	sb.WriteString("| Task | Text Complete | Runtime Complete | Text Order | Runtime Order |\n")
	sb.WriteString("|------|--------------|------------------|------------|---------------|\n")

	textByTask := groupByTask(cr.TextResults)
	rtByTask := groupByTask(cr.RuntimeResults)

	for taskID, textR := range textByTask {
		rtR := rtByTask[taskID]
		tComp := "No"
		if len(textR) > 0 && textR[0].Completed {
			tComp = "Yes"
		}
		rComp := "No"
		if len(rtR) > 0 && rtR[0].Completed {
			rComp = "Yes"
		}
		tOrd := 0.0
		if len(textR) > 0 {
			tOrd = textR[0].OrderingScore
		}
		rOrd := 0.0
		if len(rtR) > 0 {
			rOrd = rtR[0].OrderingScore
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %.0f%% | %.0f%% |\n",
			taskID, tComp, rComp, tOrd*100, rOrd*100))
	}

	sb.WriteString("\n")

	// Efficiency metrics
	em := cr.Efficiency
	if em.TextTokens > 0 {
		sb.WriteString("## Cost-Efficiency\n\n")
		sb.WriteString("| Metric | Text | Runtime |\n")
		sb.WriteString("|--------|------|--------|\n")
		sb.WriteString(fmt.Sprintf("| Quality score (weighted) | %.1f%% | %.1f%% |\n", em.TextQuality*100, em.RuntimeQuality*100))
		sb.WriteString(fmt.Sprintf("| Mean tokens/run | %.0f | %.0f |\n", em.TextTokens, em.RuntimeTokens))
		sb.WriteString(fmt.Sprintf("| Efficiency (quality per 1k tokens) | %.3f | %.3f |\n", em.TextEfficiency, em.RuntimeEfficiency))
		if em.CostPerDetection > 0 {
			sb.WriteString(fmt.Sprintf("| Cost per detection | — | %.0f extra tokens |\n", em.CostPerDetection))
		}
		sb.WriteString("\n")
		sb.WriteString("*Quality = 40% completion + 30% ordering + 30% guardrails. Efficiency = quality / (tokens/1000).*\n\n")
	}

	// Step breakdown
	if len(cr.StepBreakdowns) > 0 {
		sb.WriteString("## Per-Step Breakdown\n\n")
		sb.WriteString("| Step | Text Complete | Runtime Complete |\n")
		sb.WriteString("|------|-------------|------------------|\n")
		for _, s := range cr.StepBreakdowns {
			sb.WriteString(fmt.Sprintf("| %s | %.0f%% | %.0f%% |\n",
				s.StepID, s.TextCompletionPct, s.RTCompletionPct))
		}
		sb.WriteString("\n")
		sb.WriteString("*Sorted by text completion rate (lowest first = most fragile steps).*\n\n")
	}

	// Stability metrics
	if len(cr.Stability) > 0 {
		sb.WriteString("## Stability (Standard Deviation across runs)\n\n")
		sb.WriteString("| Task | Text σ(completion) | RT σ(completion) | Text σ(ordering) | RT σ(ordering) | Text σ(guardrail) | RT σ(guardrail) |\n")
		sb.WriteString("|------|-------------------|-----------------|-----------------|---------------|------------------|----------------|\n")
		for _, s := range cr.Stability {
			sb.WriteString(fmt.Sprintf("| %s | %.3f | %.3f | %.3f | %.3f | %.3f | %.3f |\n",
				s.TaskID, s.TextCompletionStd, s.RTCompletionStd, s.TextOrderingStd, s.RTOrderingStd, s.TextGuardrailStd, s.RTGuardrailStd))
		}
		sb.WriteString("\n")
		sb.WriteString("*Lower σ = more stable. Runtime σ(ordering) should be 0.000 (guaranteed by topological sort).*\n\n")
	}

	// Cascade rate
	cas := cr.CascadeRate
	if cas.TextRunsWithFailure > 0 || cas.RuntimeRunsWithFailure > 0 {
		sb.WriteString("## Failure Cascade Rate\n\n")
		sb.WriteString("| Condition | Runs with failure | Runs with cascade (2+ consecutive) | Cascade rate |\n")
		sb.WriteString("|-----------|------------------|------------------------------------|--------------|\n")
		sb.WriteString(fmt.Sprintf("| Text | %d | %d | %.0f%% |\n",
			cas.TextRunsWithFailure, cas.TextRunsWithCascade, cas.TextCascadeRate*100))
		sb.WriteString(fmt.Sprintf("| Runtime | %d | %d | %.0f%% |\n",
			cas.RuntimeRunsWithFailure, cas.RuntimeRunsWithCascade, cas.RuntimeCascadeRate*100))
		sb.WriteString("\n*Cascade = 2+ consecutive steps failed in the same run. Runtime isolates steps, preventing propagation.*\n\n")
	}

	// Difficulty gradient
	dg := cr.DifficultyGradient
	if len(dg.Points) > 0 {
		sb.WriteString("## Difficulty Gradient\n\n")
		sb.WriteString("Does the runtime advantage grow with task complexity?\n\n")
		sb.WriteString("| Task | Steps | Completion Δ | Ordering Δ |\n")
		sb.WriteString("|------|-------|-------------|------------|\n")
		for _, p := range dg.Points {
			sb.WriteString(fmt.Sprintf("| %s | %d | %+.0f%% | %+.0f%% |\n",
				p.TaskID, p.NumSteps, p.CompletionDelta*100, p.OrderingDelta*100))
		}
		sb.WriteString(fmt.Sprintf("\nSpearman ρ (steps vs completion Δ): **%.3f**\n", dg.SpearmanComp))
		sb.WriteString(fmt.Sprintf("Spearman ρ (steps vs ordering Δ): **%.3f**\n\n", dg.SpearmanOrder))
		sb.WriteString("*Positive ρ = runtime advantage increases with complexity.*\n\n")
	}

	// Convergence
	conv := cr.Convergence
	if conv.TextConvergence > 0 {
		sb.WriteString("## Convergence Speed\n\n")
		sb.WriteString(fmt.Sprintf("| Condition | Runs to converge (±1%%) |\n"))
		sb.WriteString("|-----------|------------------------|\n")
		sb.WriteString(fmt.Sprintf("| Text | %d |\n", conv.TextConvergence))
		sb.WriteString(fmt.Sprintf("| Runtime | %d |\n\n", conv.RuntimeConvergence))
		sb.WriteString("*Fewer runs = more predictable behavior.*\n\n")
	}

	// Failure correlation
	fc := cr.FailureCorrelation
	if fc.CompletionOrdering != 0 || fc.CompletionGuardrail != 0 || fc.OrderingGuardrail != 0 {
		sb.WriteString("## Failure Mode Correlation (text condition)\n\n")
		sb.WriteString("| Pair | Pearson r |\n")
		sb.WriteString("|------|-----------|\n")
		sb.WriteString(fmt.Sprintf("| Completion ↔ Ordering | %.3f |\n", fc.CompletionOrdering))
		sb.WriteString(fmt.Sprintf("| Completion ↔ Guardrails | %.3f |\n", fc.CompletionGuardrail))
		sb.WriteString(fmt.Sprintf("| Ordering ↔ Guardrails | %.3f |\n\n", fc.OrderingGuardrail))
		sb.WriteString("*High r = failure modes are correlated (same root cause). Low r = independent failure modes (harness must protect each separately).*\n\n")
	}

	// First failure step
	if len(cr.FirstFailureSteps) > 0 {
		sb.WriteString("## First Failure Step\n\n")
		sb.WriteString("Which step fails first in failed runs?\n\n")
		sb.WriteString("| Step | Text (first fail count) | Runtime (first fail count) |\n")
		sb.WriteString("|------|------------------------|---------------------------|\n")
		for _, e := range cr.FirstFailureSteps {
			sb.WriteString(fmt.Sprintf("| %s | %d | %d |\n", e.StepID, e.TextCount, e.RuntimeCount))
		}
		sb.WriteString("\n")
	}

	// Output length CV
	ol := cr.OutputLengthCV
	if ol.TextMeanCV > 0 || ol.RuntimeMeanCV > 0 {
		sb.WriteString("## Output Length Consistency\n\n")
		sb.WriteString(fmt.Sprintf("| Condition | Mean CV (coefficient of variation) |\n"))
		sb.WriteString("|-----------|------------------------------------|\n")
		sb.WriteString(fmt.Sprintf("| Text | %.3f |\n", ol.TextMeanCV))
		sb.WriteString(fmt.Sprintf("| Runtime | %.3f |\n\n", ol.RuntimeMeanCV))
		sb.WriteString("*Lower CV = more consistent output lengths. Runtime isolates step context.*\n\n")
	}

	// Guardrail severity
	gsev := cr.GuardrailSeverity
	if gsev.TextMeanViolations > 0 || gsev.RuntimeMeanViolations > 0 {
		sb.WriteString("## Guardrail Violation Severity\n\n")
		sb.WriteString("| Condition | Mean violations/failed run | Max violations |\n")
		sb.WriteString("|-----------|--------------------------|----------------|\n")
		sb.WriteString(fmt.Sprintf("| Text | %.1f | %d |\n", gsev.TextMeanViolations, gsev.TextMaxViolations))
		sb.WriteString(fmt.Sprintf("| Runtime | %.1f | %d |\n\n", gsev.RuntimeMeanViolations, gsev.RuntimeMaxViolations))
	}

	// Sandbox metrics (Story 5-4)
	if cr.SandboxMetrics != nil {
		sm := cr.SandboxMetrics
		sb.WriteString("## Sandbox Enforcement Metrics\n\n")
		sb.WriteString(fmt.Sprintf("Tool-based tasks: %d\n\n", sm.ToolTaskCount))
		sb.WriteString("| Metric | Text | Runtime |\n")
		sb.WriteString("|--------|------|--------|\n")
		sb.WriteString(fmt.Sprintf("| Action adherence | N/A (no sandbox) | %.0f%% |\n", sm.RuntimeAdherence*100))
		sb.WriteString(fmt.Sprintf("| Prevention rate | N/A | %.0f%% |\n", sm.RuntimePrevention*100))
		sb.WriteString(fmt.Sprintf("| Trap trigger rate | N/A | %.0f%% |\n", sm.RuntimeTrapTrigger*100))
		sb.WriteString(fmt.Sprintf("| Scope creep rate | N/A | %.0f%% |\n", sm.RuntimeScopeCreep*100))
		sb.WriteString(fmt.Sprintf("| Write safety | N/A | %.0f%% |\n\n", sm.RuntimeWriteSafety*100))
		sb.WriteString("*Text condition has no sandbox — enforcement is unmeasurable. This is the point: text instructions cannot prevent destructive actions.*\n\n")
	}

	return sb.String()
}

func groupByTask(results []TaskResult) map[string][]TaskResult {
	m := make(map[string][]TaskResult)
	for _, r := range results {
		m[r.TaskID] = append(m[r.TaskID], r)
	}
	return m
}
