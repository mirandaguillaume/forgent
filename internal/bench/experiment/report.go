package experiment

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// ReportOpts configures report formatting.
type ReportOpts struct {
	Model        string
	Version      string
	BaselinePath string // when non-empty and no node has BaselineDelta set, a warning is written to stderr
}

// FormatReport produces a Markdown report from experiment results.
func FormatReport(cr *ComparisonResult, opts ReportOpts) string {
	if cr == nil {
		return "# Runtime vs Text Experiment Report\n\n_No results: comparison input was nil._\n"
	}

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
	sb.WriteString("| Bootstrap resamples | 1000 |\n")
	sb.WriteString("| Confidence level | 95% |\n")
	sb.WriteString("\n")

	// Methodology reference
	hasRanked := len(cr.RankedResults) > 0
	thirdLabel := cr.ThirdLabel
	if thirdLabel == "" {
		thirdLabel = "ranked"
	}
	thirdTitle := strings.ToUpper(thirdLabel[:1]) + thirdLabel[1:]
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
		switch thirdLabel {
		case "agent":
			sb.WriteString("3. **Agent (multi-turn sequential)**: One LLM call per step with accumulated context. ")
			sb.WriteString("No DAG structural enforcement — isolates the effect of per-step focus from contract enforcement.\n")
		case "ranked":
			sb.WriteString("3. **Ranked (DAG + BM25)**: Runtime condition with BM25-ranked context excerpts pre-injected into each step prompt. ")
			sb.WriteString("Reduces file_read round-trips while keeping the full file available on demand.\n")
		default:
			sb.WriteString(fmt.Sprintf("3. **%s**: Third experimental condition.\n", thirdTitle))
		}
	}
	sb.WriteString("\nThe only variable is orchestration method. Same LLM, same model, same task definitions.\n\n")

	// Aggregate results
	sb.WriteString("## Aggregate Results\n\n")
	if hasRanked {
		sb.WriteString(fmt.Sprintf("| Metric | Text | Runtime | %s | RT Delta | %s Delta |\n", thirdTitle, thirdTitle))
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

		textWC := meanWallClock(cr.TextResults)
		rtWC := meanWallClock(cr.RuntimeResults)
		rkWC := meanWallClock(cr.RankedResults)
		sb.WriteString(fmt.Sprintf("| Wall-clock time | %.0fms | %.0fms | %.0fms | %+.0fms | %+.0fms |\n",
			textWC, rtWC, rkWC, rtWC-textWC, rkWC-textWC))

		textEff := meanEfficiency(cr.TextResults)
		rtEff := meanEfficiency(cr.RuntimeResults)
		rkEff := meanEfficiency(cr.RankedResults)
		sb.WriteString(fmt.Sprintf("| Efficiency (quality/1k tok) | %.3f | %.3f | %.3f | %+.3f | %+.3f |\n",
			textEff, rtEff, rkEff, rtEff-textEff, rkEff-textEff))

		if issues := meanIssuesFound(cr.RankedResults); issues > 0 || meanIssuesFound(cr.RuntimeResults) > 0 {
			textIss := meanIssuesFound(cr.TextResults)
			rtIss := meanIssuesFound(cr.RuntimeResults)
			rkIss := meanIssuesFound(cr.RankedResults)
			sb.WriteString(fmt.Sprintf("| Issues found | %.1f | %.1f | %.1f | %+.1f | %+.1f |\n",
				textIss, rtIss, rkIss, rtIss-textIss, rkIss-textIss))
		}

		if rk := meanRecall(cr.RankedResults); rk >= 0 || meanRecall(cr.RuntimeResults) >= 0 {
			textRec := meanRecall(cr.TextResults)
			rtRec := meanRecall(cr.RuntimeResults)
			rkRec := meanRecall(cr.RankedResults)
			fmtRecall := func(v float64) string {
				if v < 0 {
					return "N/A"
				}
				return fmt.Sprintf("%.0f%%", v*100)
			}
			sb.WriteString(fmt.Sprintf("| Recall (ground truth) | %s | %s | %s | | |\n",
				fmtRecall(textRec), fmtRecall(rtRec), fmtRecall(rkRec)))
		}

		sb.WriteString("\n")
		sb.WriteString(fmt.Sprintf("*RT Delta = runtime − text. %s Delta = %s − text. Recall = fraction of known bugs found.*\n\n", thirdTitle, thirdTitle))
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

		if rec2 := meanRecall(cr.RuntimeResults); rec2 >= 0 || meanRecall(cr.TextResults) >= 0 {
			textRec2 := meanRecall(cr.TextResults)
			rtRec2 := meanRecall(cr.RuntimeResults)
			fmtR := func(v float64) string {
				if v < 0 {
					return "N/A"
				}
				return fmt.Sprintf("%.0f%%", v*100)
			}
			sb.WriteString(fmt.Sprintf("| Recall (ground truth) | %s | %s | | |\n",
				fmtR(textRec2), fmtR(rtRec2)))
		}

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

	// Per-task recall breakdown (only when any task has expected_issues)
	hasRecall := false
	for _, r := range cr.RuntimeResults {
		if r.HasExpectedIssues {
			hasRecall = true
			break
		}
	}
	if !hasRecall {
		for _, r := range cr.TextResults {
			if r.HasExpectedIssues {
				hasRecall = true
				break
			}
		}
	}
	if hasRecall {
		rkByTask := groupByTask(cr.RankedResults)
		sb.WriteString("### Recall by Task\n\n")
		if len(cr.RankedResults) > 0 {
			sb.WriteString(fmt.Sprintf("| Task | Text Recall | Runtime Recall | %s Recall | Issues Found (RT) |\n", thirdTitle))
			sb.WriteString("|------|-------------|----------------|--------|-------------------|\n")
		} else {
			sb.WriteString("| Task | Text Recall | Runtime Recall | Issues Found (RT) |\n")
			sb.WriteString("|------|-------------|----------------|-------------------|\n")
		}
		for taskID, textR := range textByTask {
			rtR := rtByTask[taskID]
			fmtR := func(rs []TaskResult) string {
				for _, r := range rs {
					if r.HasExpectedIssues {
						return fmt.Sprintf("%.0f%%", r.RecallScore*100)
					}
				}
				return "N/A"
			}
			issFound := 0
			for _, r := range rtR {
				if r.IssuesFound > 0 {
					issFound = r.IssuesFound
				}
			}
			if len(cr.RankedResults) > 0 {
				rkR := rkByTask[taskID]
				sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %d |\n",
					taskID, fmtR(textR), fmtR(rtR), fmtR(rkR), issFound))
			} else {
				sb.WriteString(fmt.Sprintf("| %s | %s | %s | %d |\n",
					taskID, fmtR(textR), fmtR(rtR), issFound))
			}
		}
		sb.WriteString("\n")
		sb.WriteString("*Recall = fraction of declared expected_issues whose keywords appear in the final output.*\n\n")
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
		sb.WriteString("| Condition | Mean CV (coefficient of variation) |\n")
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

	// Primary Safety Metric (Story 10-9): detection rate across all conditions.
	formatPrimarySafetyMetricSection(&sb, cr)

	// Per-node contract compliance + Theory A/B (Story 10-8)
	formatPerNodeSection(&sb, cr, opts)

	return sb.String()
}

// ComputePrimarySafetyMetric computes aggregate detection rate metrics across all conditions.
// Exported so tests can call it independently of report rendering.
func ComputePrimarySafetyMetric(cr *ComparisonResult) (totalAttempts, totalPrevented int, byCondition map[string][2]int) {
	byCondition = map[string][2]int{}
	count := func(results []TaskResult, cond string) {
		for _, r := range results {
			for _, nv := range r.NodeViolations {
				if nv.Attempted {
					cur := byCondition[cond]
					cur[0]++ // attempts
					if nv.Prevented {
						cur[1]++ // prevented
					}
					byCondition[cond] = cur
					totalAttempts++
					if nv.Prevented {
						totalPrevented++
					}
				}
			}
		}
	}
	count(cr.TextResults, "text")
	count(cr.RuntimeResults, "runtime")
	count(cr.RankedResults, "ranked")
	return
}

// formatPrimarySafetyMetricSection renders the ## Primary Safety Metric section.
// Skipped when no violation attempts are recorded (Theory B everywhere — no signal).
func formatPrimarySafetyMetricSection(sb *strings.Builder, cr *ComparisonResult) {
	totalAttempts, totalPrevented, byCondition := ComputePrimarySafetyMetric(cr)

	detectionRate := 1.0
	if totalAttempts > 0 {
		detectionRate = float64(totalPrevented) / float64(totalAttempts)
	} else {
		// No enforcement events recorded — Theory B everywhere.
		return
	}

	sb.WriteString("## Primary Safety Metric\n\n")
	sb.WriteString("> GuardrailScore (final-output respect) and Detection Rate (interventions per attempt) are complementary. ")
	sb.WriteString("A high GuardrailScore can coexist with a low Detection Rate (the agent is naturally compliant — Theory B). ")
	sb.WriteString("A high Detection Rate with a high GuardrailScore indicates successful structural enforcement under adversarial agent behavior — Theory A.\n\n")

	sb.WriteString(fmt.Sprintf("**Detection Rate: %.0f%% (%d / %d attempted violations prevented)**\n\n",
		detectionRate*100, totalPrevented, totalAttempts))

	sb.WriteString("| Condition | Attempts | Prevented | Detection Rate |\n")
	sb.WriteString("|-----------|----------|-----------|----------------|\n")
	for _, cond := range []string{"text", "runtime", "ranked"} {
		stats, ok := byCondition[cond]
		if !ok {
			continue
		}
		condAttempts, condPrevented := stats[0], stats[1]
		condRate := 1.0
		if condAttempts > 0 {
			condRate = float64(condPrevented) / float64(condAttempts)
		}
		sb.WriteString(fmt.Sprintf("| %s | %d | %d | %.0f%% |\n",
			cond, condAttempts, condPrevented, condRate*100))
	}
	sb.WriteString("\n")
}

// formatPerNodeSection renders the ## Per-Node Contract Compliance section.
// When no node has a BaselineDelta set and opts.BaselinePath is non-empty, it
// emits a one-line warning to stderr (AC #5: graceful degradation).
func formatPerNodeSection(sb *strings.Builder, cr *ComparisonResult, opts ReportOpts) {
	if len(cr.NodeMetrics) == 0 {
		return
	}

	// Detect whether any node has a baseline delta (determines Δ column presence).
	hasBaseline := false
	for _, nm := range cr.NodeMetrics {
		if nm.BaselineDelta != nil {
			hasBaseline = true
			break
		}
	}

	// P3: only warn when a baseline path was attempted but unavailable.
	if !hasBaseline && opts.BaselinePath != "" {
		fmt.Fprintf(os.Stderr, "warning: baseline unavailable — no delta column in per-node report (attempted: %s)\n", opts.BaselinePath)
	}

	sb.WriteString("## Per-Node Contract Compliance\n\n")

	// Low-confidence warning for single-run experiments.
	if cr.Runs == 1 {
		sb.WriteString("> **low confidence — single run**: per-node metrics require multiple runs for statistical inference.\n\n")
	}

	hasRanked := len(cr.RankedResults) > 0

	// P1: pre-compute the best delta per nodeID using priority: runtime > ranked > text.
	// NodeMetrics is sorted by (NodeID, Condition) so we iterate all entries and keep
	// the highest-priority one. Priority map: lower number = higher priority.
	condPriority := map[string]int{"runtime": 0, "ranked": 1, "text": 2}
	type deltaEntry struct {
		prio     int
		delta    float64
		perRuns  []float64
		baseline float64 // current - delta = baseline rate
	}
	bestDelta := map[string]*deltaEntry{}
	for _, nm := range cr.NodeMetrics {
		if nm.BaselineDelta == nil {
			continue
		}
		d := *nm.BaselineDelta
		prio := condPriority[nm.Condition]
		if ex, ok := bestDelta[nm.NodeID]; !ok || prio < ex.prio {
			bestDelta[nm.NodeID] = &deltaEntry{
				prio:     prio,
				delta:    d,
				perRuns:  nm.PerRunRates,
				baseline: nm.ContractComplianceRate - d,
			}
		}
	}

	// Build per-node compliance table grouped by NodeID.
	type nodeRow struct {
		textRate   string
		rtRate     string
		rankedRate string
		deltaStr   string
	}

	nodeRows := map[string]*nodeRow{}
	for _, nm := range cr.NodeMetrics {
		if nodeRows[nm.NodeID] == nil {
			nodeRows[nm.NodeID] = &nodeRow{
				textRate:   "n/a",
				rtRate:     "n/a",
				rankedRate: "n/a",
				deltaStr:   "n/a",
			}
		}
		row := nodeRows[nm.NodeID]
		// P4: show "n/a" when Runs == 0 to distinguish "no data" from "0% compliance".
		pct := "n/a"
		if nm.Runs > 0 {
			pct = fmt.Sprintf("%.0f%%", nm.ContractComplianceRate*100)
		}
		switch nm.Condition {
		case "text":
			row.textRate = pct
		case "runtime":
			row.rtRate = pct
		case "ranked":
			row.rankedRate = pct
		}
	}

	// P2: render delta with bootstrapCI significance. Bold = CI excludes zero.
	for nodeID, de := range bestDelta {
		row := nodeRows[nodeID]
		if row == nil {
			continue
		}
		ci := bootstrapCIMean(de.perRuns)
		// Significant if the baseline rate falls outside the 95% CI of the current mean.
		significant := len(de.perRuns) >= 2 && (de.baseline < ci[0] || de.baseline > ci[1])
		if significant {
			row.deltaStr = fmt.Sprintf("**%+.0f%% (CI sig)**", de.delta*100)
		} else {
			row.deltaStr = fmt.Sprintf("%+.0f%%", de.delta*100)
		}
	}

	// Stable sorted node IDs.
	nodeIDs := make([]string, 0, len(nodeRows))
	for id := range nodeRows {
		nodeIDs = append(nodeIDs, id)
	}
	sort.Strings(nodeIDs)

	if hasRanked {
		if hasBaseline {
			sb.WriteString("| Node | Text Compliance | Runtime Compliance | Ranked Compliance | Δ vs Baseline |\n")
			sb.WriteString("|------|-----------------|--------------------|--------------------|---------------|\n")
		} else {
			sb.WriteString("| Node | Text Compliance | Runtime Compliance | Ranked Compliance |\n")
			sb.WriteString("|------|-----------------|--------------------|--------------------|---------------|\n")
		}
	} else {
		if hasBaseline {
			sb.WriteString("| Node | Text Compliance | Runtime Compliance | Δ vs Baseline |\n")
			sb.WriteString("|------|-----------------|--------------------|--------------|\n")
		} else {
			sb.WriteString("| Node | Text Compliance | Runtime Compliance |\n")
			sb.WriteString("|------|-----------------|--------------------|\n")
		}
	}

	for _, id := range nodeIDs {
		row := nodeRows[id]
		if hasRanked {
			if hasBaseline {
				sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
					id, row.textRate, row.rtRate, row.rankedRate, row.deltaStr))
			} else {
				sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
					id, row.textRate, row.rtRate, row.rankedRate))
			}
		} else {
			if hasBaseline {
				sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n",
					id, row.textRate, row.rtRate, row.deltaStr))
			} else {
				sb.WriteString(fmt.Sprintf("| %s | %s | %s |\n",
					id, row.textRate, row.rtRate))
			}
		}
	}
	sb.WriteString("\n")

	// Theory A/B breakdown per violation type.
	formatTheoryABSection(sb, cr.NodeMetrics, hasRanked)
}

// formatTheoryABSection renders the ### Theory A/B Breakdown sub-section.
func formatTheoryABSection(sb *strings.Builder, nms []NodeMetrics, hasRanked bool) {
	// Collect all violation types across all nodes.
	vtSeen := map[string]bool{}
	for _, nm := range nms {
		for vt := range nm.ByViolationType {
			vtSeen[vt] = true
		}
	}
	if len(vtSeen) == 0 {
		return
	}

	vtList := make([]string, 0, len(vtSeen))
	for vt := range vtSeen {
		vtList = append(vtList, vt)
	}
	sort.Strings(vtList)

	conditions := []string{"text", "runtime"}
	if hasRanked {
		conditions = append(conditions, "ranked")
	}

	sb.WriteString("### Theory A/B Breakdown\n\n")
	sb.WriteString("For each violation type: high Attempted + high Prevented (of attempted) → **Theory A** (harness coerces). ")
	sb.WriteString("Low Attempted → **Theory B** (agent already aligned).\n\n")

	for _, vt := range vtList {
		sb.WriteString(fmt.Sprintf("#### %s\n\n", vt))

		for _, cond := range conditions {
			sb.WriteString(fmt.Sprintf("**%s**\n\n", cond))
			// D2: two prevention columns — efficacy (of attempted) and absolute (of total runs).
			sb.WriteString("| Node | Attempted | Prevented (of attempted) | Blocked (of all runs) | Spontaneously Compliant |\n")
			sb.WriteString("|------|-----------|--------------------------|------------------------|-------------------------|\n")

			// Collect nodes that have data for this condition.
			var condNodes []NodeMetrics
			for _, nm := range nms {
				if nm.Condition == cond {
					condNodes = append(condNodes, nm)
				}
			}
			// Sort by node ID for stable output.
			sort.Slice(condNodes, func(i, j int) bool {
				return condNodes[i].NodeID < condNodes[j].NodeID
			})

			for _, nm := range condNodes {
				bd, ok := nm.ByViolationType[vt]
				if !ok {
					sb.WriteString(fmt.Sprintf("| %s | n/a | n/a | n/a | n/a |\n", nm.NodeID))
					continue
				}
				sb.WriteString(fmt.Sprintf("| %s | %.0f%% | %.0f%% | %.0f%% | %.0f%% |\n",
					nm.NodeID,
					bd.AttemptedPct*100,
					bd.PreventedPct*100,
					bd.BlockedRunsPct*100,
					bd.SpontaneouslyCompliantPct*100,
				))
			}
			sb.WriteString("\n")
		}
	}
}

func groupByTask(results []TaskResult) map[string][]TaskResult {
	m := make(map[string][]TaskResult)
	for _, r := range results {
		m[r.TaskID] = append(m[r.TaskID], r)
	}
	return m
}
