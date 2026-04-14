package experiment

import (
	"math"
	"math/rand"
	"sort"
)

// Compare computes deltas between text and runtime conditions with bootstrap
// confidence intervals. The delta is always runtime - text, so positive values
// mean runtime outperforms text.
func Compare(textResults, runtimeResults []TaskResult) *ComparisonResult {
	cr := &ComparisonResult{
		Tasks:          len(textResults),
		Runs:           len(textResults),
		TextResults:    textResults,
		RuntimeResults: runtimeResults,
	}

	if len(textResults) == 0 || len(runtimeResults) == 0 {
		return cr
	}

	// Compute per-task deltas
	textCompletion := completionRate(textResults)
	runtimeCompletion := completionRate(runtimeResults)
	cr.CompletionDelta = runtimeCompletion - textCompletion

	textOrdering := meanOrdering(textResults)
	runtimeOrdering := meanOrdering(runtimeResults)
	cr.OrderingDelta = runtimeOrdering - textOrdering

	textGuardrail := meanGuardrail(textResults)
	runtimeGuardrail := meanGuardrail(runtimeResults)
	cr.GuardrailDelta = runtimeGuardrail - textGuardrail

	textTokens := meanTokens(textResults)
	runtimeTokens := meanTokens(runtimeResults)
	cr.TokenDelta = runtimeTokens - textTokens

	// Bootstrap confidence intervals (1000 resamples)
	cr.CompletionCI = bootstrapCI(textResults, runtimeResults, func(r TaskResult) float64 {
		if r.Completed { return 1 }; return 0
	})
	cr.OrderingCI = bootstrapCI(textResults, runtimeResults, func(r TaskResult) float64 {
		return r.OrderingScore
	})
	cr.GuardrailCI = bootstrapCI(textResults, runtimeResults, func(r TaskResult) float64 {
		return r.GuardrailScore
	})
	cr.TokenCI = bootstrapCI(textResults, runtimeResults, func(r TaskResult) float64 {
		return float64(r.TokensUsed)
	})

	// Engineering metrics
	textWallClock := meanWallClock(textResults)
	runtimeWallClock := meanWallClock(runtimeResults)
	cr.WallClockDelta = runtimeWallClock - textWallClock
	cr.WallClockCI = bootstrapCI(textResults, runtimeResults, func(r TaskResult) float64 {
		return float64(r.WallClockMs)
	})

	cr.LLMCallsDelta = meanLLMCalls(runtimeResults) - meanLLMCalls(textResults)
	cr.CrossContamination = MeasureCrossContamination(textResults)

	// Harness detection rate: pair text/runtime results by index
	cr.DetectionRate = ComputeDetectionRate(textResults, runtimeResults)

	// Derived metrics
	cr.Efficiency = ComputeEfficiency(textResults, runtimeResults, cr.DetectionRate)
	cr.StepBreakdowns = ComputeStepBreakdowns(textResults, runtimeResults)
	cr.Stability = ComputeStability(textResults, runtimeResults)

	return cr
}

// ComputeDetectionRate pairs text and runtime results by index and computes
// how many text-mode failures the runtime harness would have caught.
// Results must be in the same order (same task sequence, same run count).
func ComputeDetectionRate(text, runtime []TaskResult) DetectionMetrics {
	var dm DetectionMetrics
	n := len(text)
	if len(runtime) < n {
		n = len(runtime)
	}

	for i := 0; i < n; i++ {
		t, r := text[i], runtime[i]
		anyFail := false

		// Completion: text failed, runtime succeeded
		if !t.Completed {
			dm.CompletionFailures++
			anyFail = true
			if r.Completed {
				dm.CompletionCaught++
			}
		}

		// Ordering: text had imperfect ordering, runtime had perfect
		if t.OrderingScore < 1.0 {
			dm.OrderingFailures++
			anyFail = true
			if r.OrderingScore >= 1.0 {
				dm.OrderingCaught++
			}
		}

		// Guardrails: text violated, runtime respected
		if t.GuardrailScore < 1.0 {
			dm.GuardrailFailures++
			anyFail = true
			if r.GuardrailScore >= 1.0 {
				dm.GuardrailCaught++
			}
		}

		// Any failure caught
		if anyFail {
			dm.AnyFailures++
			runtimeOK := r.Completed && r.OrderingScore >= 1.0 && r.GuardrailScore >= 1.0
			if runtimeOK {
				dm.AnyCaught++
			}
		}
	}

	if dm.CompletionFailures > 0 {
		dm.CompletionRate = float64(dm.CompletionCaught) / float64(dm.CompletionFailures)
	}
	if dm.OrderingFailures > 0 {
		dm.OrderingRate = float64(dm.OrderingCaught) / float64(dm.OrderingFailures)
	}
	if dm.GuardrailFailures > 0 {
		dm.GuardrailRate = float64(dm.GuardrailCaught) / float64(dm.GuardrailFailures)
	}
	if dm.AnyFailures > 0 {
		dm.AnyRate = float64(dm.AnyCaught) / float64(dm.AnyFailures)
	}

	return dm
}

func completionRate(results []TaskResult) float64 {
	if len(results) == 0 {
		return 0
	}
	completed := 0
	for _, r := range results {
		if r.Completed {
			completed++
		}
	}
	return float64(completed) / float64(len(results))
}

func meanOrdering(results []TaskResult) float64 {
	if len(results) == 0 {
		return 0
	}
	sum := 0.0
	for _, r := range results {
		sum += r.OrderingScore
	}
	return sum / float64(len(results))
}

func meanGuardrail(results []TaskResult) float64 {
	if len(results) == 0 {
		return 0
	}
	sum := 0.0
	for _, r := range results {
		sum += r.GuardrailScore
	}
	return sum / float64(len(results))
}

func meanTokens(results []TaskResult) float64 {
	if len(results) == 0 {
		return 0
	}
	sum := 0.0
	for _, r := range results {
		sum += float64(r.TokensUsed)
	}
	return sum / float64(len(results))
}

func meanWallClock(results []TaskResult) float64 {
	if len(results) == 0 {
		return 0
	}
	sum := 0.0
	for _, r := range results {
		sum += float64(r.WallClockMs)
	}
	return sum / float64(len(results))
}

func meanLLMCalls(results []TaskResult) float64 {
	if len(results) == 0 {
		return 0
	}
	sum := 0.0
	for _, r := range results {
		sum += float64(r.LLMCalls)
	}
	return sum / float64(len(results))
}

// bootstrapCI computes a 95% confidence interval for the delta between
// text and runtime conditions using bootstrap resampling.
func bootstrapCI(text, runtime []TaskResult, metric func(TaskResult) float64) [2]float64 {
	if len(text) == 0 || len(runtime) == 0 {
		return [2]float64{0, 0}
	}

	const nBootstrap = 1000
	rng := rand.New(rand.NewSource(42)) // deterministic for reproducibility

	deltas := make([]float64, nBootstrap)
	for i := 0; i < nBootstrap; i++ {
		// Resample with replacement
		textSample := resample(text, rng)
		runtimeSample := resample(runtime, rng)

		textMean := meanMetric(textSample, metric)
		runtimeMean := meanMetric(runtimeSample, metric)
		deltas[i] = runtimeMean - textMean
	}

	sort.Float64s(deltas)

	// 2.5th and 97.5th percentiles
	lo := int(math.Floor(0.025 * float64(nBootstrap)))
	hi := int(math.Floor(0.975 * float64(nBootstrap)))
	if hi >= nBootstrap {
		hi = nBootstrap - 1
	}

	return [2]float64{deltas[lo], deltas[hi]}
}

func resample(results []TaskResult, rng *rand.Rand) []TaskResult {
	n := len(results)
	sample := make([]TaskResult, n)
	for i := 0; i < n; i++ {
		sample[i] = results[rng.Intn(n)]
	}
	return sample
}

func meanMetric(results []TaskResult, metric func(TaskResult) float64) float64 {
	if len(results) == 0 {
		return 0
	}
	sum := 0.0
	for _, r := range results {
		sum += metric(r)
	}
	return sum / float64(len(results))
}

// qualityScore computes a weighted quality score from a TaskResult.
// Weights: completion 40%, ordering 30%, guardrails 30%.
func qualityScore(r TaskResult) float64 {
	comp := 0.0
	if r.Completed {
		comp = 1.0
	}
	return comp*0.4 + r.OrderingScore*0.3 + r.GuardrailScore*0.3
}

// ComputeEfficiency calculates quality-per-token for each condition.
func ComputeEfficiency(text, runtime []TaskResult, dm DetectionMetrics) EfficiencyMetrics {
	em := EfficiencyMetrics{}
	if len(text) == 0 || len(runtime) == 0 {
		return em
	}

	for _, r := range text {
		em.TextQuality += qualityScore(r)
		em.TextTokens += float64(r.TokensUsed)
	}
	em.TextQuality /= float64(len(text))
	em.TextTokens /= float64(len(text))

	for _, r := range runtime {
		em.RuntimeQuality += qualityScore(r)
		em.RuntimeTokens += float64(r.TokensUsed)
	}
	em.RuntimeQuality /= float64(len(runtime))
	em.RuntimeTokens /= float64(len(runtime))

	if em.TextTokens > 0 {
		em.TextEfficiency = em.TextQuality / (em.TextTokens / 1000)
	}
	if em.RuntimeTokens > 0 {
		em.RuntimeEfficiency = em.RuntimeQuality / (em.RuntimeTokens / 1000)
	}

	// Cost per detection: extra tokens per failure caught
	if dm.AnyCaught > 0 {
		extraTokensPerRun := em.RuntimeTokens - em.TextTokens
		totalExtraTokens := extraTokensPerRun * float64(len(runtime))
		em.CostPerDetection = totalExtraTokens / float64(dm.AnyCaught)
	}

	return em
}

// ComputeStepBreakdowns aggregates per-step completion and keyword rates.
func ComputeStepBreakdowns(text, runtime []TaskResult) []StepBreakdown {
	type stepAcc struct {
		textCompleted, textRuns   int
		rtCompleted, rtRuns       int
		textKwSum                 float64
		textKwCount               int
		rtKwSum                   float64
		rtKwCount                 int
	}

	acc := make(map[string]*stepAcc)

	collect := func(results []TaskResult, isRuntime bool) {
		for _, r := range results {
			for _, sr := range r.StepResults {
				a, ok := acc[sr.StepID]
				if !ok {
					a = &stepAcc{}
					acc[sr.StepID] = a
				}
				if isRuntime {
					a.rtRuns++
					if sr.Completed {
						a.rtCompleted++
					}
					if sr.KeywordsTotal > 0 {
						a.rtKwSum += float64(sr.KeywordsHit) / float64(sr.KeywordsTotal)
						a.rtKwCount++
					}
				} else {
					a.textRuns++
					if sr.Completed {
						a.textCompleted++
					}
					if sr.KeywordsTotal > 0 {
						a.textKwSum += float64(sr.KeywordsHit) / float64(sr.KeywordsTotal)
						a.textKwCount++
					}
				}
			}
		}
	}

	collect(text, false)
	collect(runtime, true)

	var breakdowns []StepBreakdown
	for stepID, a := range acc {
		sb := StepBreakdown{StepID: stepID, Runs: a.textRuns}
		if a.textRuns > 0 {
			sb.TextCompletionPct = float64(a.textCompleted) / float64(a.textRuns) * 100
		}
		if a.rtRuns > 0 {
			sb.RTCompletionPct = float64(a.rtCompleted) / float64(a.rtRuns) * 100
		}
		if a.textKwCount > 0 {
			sb.TextKeywordPct = a.textKwSum / float64(a.textKwCount) * 100
		}
		if a.rtKwCount > 0 {
			sb.RTKeywordPct = a.rtKwSum / float64(a.rtKwCount) * 100
		}
		breakdowns = append(breakdowns, sb)
	}

	sort.Slice(breakdowns, func(i, j int) bool {
		return breakdowns[i].TextCompletionPct < breakdowns[j].TextCompletionPct
	})

	return breakdowns
}

// ComputeStability measures variance per task across runs.
func ComputeStability(text, runtime []TaskResult) []StabilityMetrics {
	textByTask := groupByTask(text)
	rtByTask := groupByTask(runtime)

	allTasks := make(map[string]bool)
	for k := range textByTask {
		allTasks[k] = true
	}
	for k := range rtByTask {
		allTasks[k] = true
	}

	var metrics []StabilityMetrics
	for taskID := range allTasks {
		sm := StabilityMetrics{TaskID: taskID}

		if results, ok := textByTask[taskID]; ok && len(results) > 1 {
			sm.TextCompletionStd = stddev(results, func(r TaskResult) float64 {
				if r.Completed {
					return 1
				}
				return 0
			})
			sm.TextOrderingStd = stddev(results, func(r TaskResult) float64 { return r.OrderingScore })
			sm.TextGuardrailStd = stddev(results, func(r TaskResult) float64 { return r.GuardrailScore })
		}

		if results, ok := rtByTask[taskID]; ok && len(results) > 1 {
			sm.RTCompletionStd = stddev(results, func(r TaskResult) float64 {
				if r.Completed {
					return 1
				}
				return 0
			})
			sm.RTOrderingStd = stddev(results, func(r TaskResult) float64 { return r.OrderingScore })
			sm.RTGuardrailStd = stddev(results, func(r TaskResult) float64 { return r.GuardrailScore })
		}

		metrics = append(metrics, sm)
	}

	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].TaskID < metrics[j].TaskID
	})

	return metrics
}

func stddev(results []TaskResult, metric func(TaskResult) float64) float64 {
	if len(results) < 2 {
		return 0
	}
	mean := 0.0
	for _, r := range results {
		mean += metric(r)
	}
	mean /= float64(len(results))

	variance := 0.0
	for _, r := range results {
		d := metric(r) - mean
		variance += d * d
	}
	variance /= float64(len(results) - 1) // sample variance

	return math.Sqrt(variance)
}
