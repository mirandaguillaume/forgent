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
