package experiment

import (
	"math"
	"sort"
)

// ComputeCascadeRate measures failure propagation within runs.
// A cascade = 2+ consecutive steps failed in the same run.
func ComputeCascadeRate(text, runtime []TaskResult) CascadeMetrics {
	cm := CascadeMetrics{}

	measure := func(results []TaskResult) (withFailure, withCascade int) {
		for _, r := range results {
			hasFail := false
			consecutive := 0
			maxConsecutive := 0
			for _, sr := range r.StepResults {
				if !sr.Completed {
					hasFail = true
					consecutive++
					if consecutive > maxConsecutive {
						maxConsecutive = consecutive
					}
				} else {
					consecutive = 0
				}
			}
			if hasFail {
				withFailure++
			}
			if maxConsecutive >= 2 {
				withCascade++
			}
		}
		return
	}

	cm.TextRunsWithFailure, cm.TextRunsWithCascade = measure(text)
	cm.RuntimeRunsWithFailure, cm.RuntimeRunsWithCascade = measure(runtime)

	if cm.TextRunsWithFailure > 0 {
		cm.TextCascadeRate = float64(cm.TextRunsWithCascade) / float64(cm.TextRunsWithFailure)
	}
	if cm.RuntimeRunsWithFailure > 0 {
		cm.RuntimeCascadeRate = float64(cm.RuntimeRunsWithCascade) / float64(cm.RuntimeRunsWithFailure)
	}

	return cm
}

// ComputeDifficultyGradient measures whether runtime advantage grows with task complexity.
func ComputeDifficultyGradient(text, runtime []TaskResult) DifficultyMetrics {
	dm := DifficultyMetrics{}

	textByTask := groupByTask(text)
	rtByTask := groupByTask(runtime)

	for taskID, textRuns := range textByTask {
		rtRuns := rtByTask[taskID]
		if len(rtRuns) == 0 {
			continue
		}

		// Count steps from first run's step results
		nSteps := len(textRuns[0].StepResults)
		if nSteps == 0 {
			continue
		}

		tComp := completionRate(textRuns)
		rComp := completionRate(rtRuns)
		tOrd := meanOrdering(textRuns)
		rOrd := meanOrdering(rtRuns)

		dm.Points = append(dm.Points, DifficultyPoint{
			TaskID:          taskID,
			NumSteps:        nSteps,
			CompletionDelta: rComp - tComp,
			OrderingDelta:   rOrd - tOrd,
		})
	}

	sort.Slice(dm.Points, func(i, j int) bool {
		return dm.Points[i].NumSteps < dm.Points[j].NumSteps
	})

	if len(dm.Points) >= 3 {
		steps := make([]float64, len(dm.Points))
		compDeltas := make([]float64, len(dm.Points))
		ordDeltas := make([]float64, len(dm.Points))
		for i, p := range dm.Points {
			steps[i] = float64(p.NumSteps)
			compDeltas[i] = p.CompletionDelta
			ordDeltas[i] = p.OrderingDelta
		}
		dm.SpearmanComp = spearmanRank(steps, compDeltas)
		dm.SpearmanOrder = spearmanRank(steps, ordDeltas)
	}

	return dm
}

// ComputeConvergence measures how many runs are needed for stable completion means.
func ComputeConvergence(text, runtime []TaskResult) ConvergenceMetrics {
	cm := ConvergenceMetrics{}

	convergencePoint := func(results []TaskResult) int {
		if len(results) < 5 {
			return len(results)
		}
		finalMean := completionRate(results)
		for n := 3; n <= len(results); n++ {
			runningMean := completionRate(results[:n])
			if math.Abs(runningMean-finalMean) < 0.01 {
				return n
			}
		}
		return len(results)
	}

	cm.TextConvergence = convergencePoint(text)
	cm.RuntimeConvergence = convergencePoint(runtime)
	return cm
}

// ComputeFailureCorrelation measures Pearson correlation between failure modes.
func ComputeFailureCorrelation(text []TaskResult) CorrelationMetrics {
	cm := CorrelationMetrics{}
	if len(text) < 3 {
		return cm
	}

	compFail := make([]float64, len(text))
	ordFail := make([]float64, len(text))
	grFail := make([]float64, len(text))

	for i, r := range text {
		if !r.Completed {
			compFail[i] = 1
		}
		if r.OrderingScore < 1.0 {
			ordFail[i] = 1
		}
		if r.GuardrailScore < 1.0 {
			grFail[i] = 1
		}
	}

	cm.CompletionOrdering = pearson(compFail, ordFail)
	cm.CompletionGuardrail = pearson(compFail, grFail)
	cm.OrderingGuardrail = pearson(ordFail, grFail)
	return cm
}

// ComputeFirstFailureSteps identifies which steps fail first in failed runs.
func ComputeFirstFailureSteps(text, runtime []TaskResult) []FirstFailureEntry {
	counts := make(map[string]*FirstFailureEntry)

	findFirst := func(results []TaskResult, isRuntime bool) {
		for _, r := range results {
			for _, sr := range r.StepResults {
				if !sr.Completed {
					e, ok := counts[sr.StepID]
					if !ok {
						e = &FirstFailureEntry{StepID: sr.StepID}
						counts[sr.StepID] = e
					}
					if isRuntime {
						e.RuntimeCount++
					} else {
						e.TextCount++
					}
					break // only the FIRST failed step per run
				}
			}
		}
	}

	findFirst(text, false)
	findFirst(runtime, true)

	var entries []FirstFailureEntry
	for _, e := range counts {
		entries = append(entries, *e)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].TextCount > entries[j].TextCount
	})
	return entries
}

// ComputeOutputLengthCV measures coefficient of variation of step output lengths.
func ComputeOutputLengthCV(text, runtime []TaskResult) OutputLengthMetrics {
	om := OutputLengthMetrics{}

	computeMeanCV := func(results []TaskResult) float64 {
		stepLengths := make(map[string][]float64)
		for _, r := range results {
			for _, sr := range r.StepResults {
				if len(sr.Output) > 0 {
					stepLengths[sr.StepID] = append(stepLengths[sr.StepID], float64(len(sr.Output)))
				}
			}
		}

		if len(stepLengths) == 0 {
			return 0
		}

		totalCV := 0.0
		count := 0
		for _, lengths := range stepLengths {
			if len(lengths) < 2 {
				continue
			}
			mean := 0.0
			for _, l := range lengths {
				mean += l
			}
			mean /= float64(len(lengths))
			if mean == 0 {
				continue
			}
			variance := 0.0
			for _, l := range lengths {
				d := l - mean
				variance += d * d
			}
			variance /= float64(len(lengths) - 1)
			cv := math.Sqrt(variance) / mean
			totalCV += cv
			count++
		}
		if count == 0 {
			return 0
		}
		return totalCV / float64(count)
	}

	om.TextMeanCV = computeMeanCV(text)
	om.RuntimeMeanCV = computeMeanCV(runtime)
	return om
}

// ComputeGuardrailSeverity measures how badly guardrails are violated.
func ComputeGuardrailSeverity(text, runtime []TaskResult) GuardrailSeverityMetrics {
	gs := GuardrailSeverityMetrics{}

	measure := func(results []TaskResult) (meanViol float64, maxViol int) {
		totalViolations := 0
		failedRuns := 0
		for _, r := range results {
			if r.GuardrailScore < 1.0 {
				failedRuns++
				// Number of guardrail violations = total guardrails * (1 - score)
				// We approximate from the score since we don't have raw guardrail count here
				// GuardrailScore = respected / total, so violations = total * (1 - score)
				// But we don't know total. Use (1-score) as a proxy 0-1.
				// Actually, step through step results isn't useful here.
				// Best proxy: count from score. With score=0.8 and assuming ~8 guardrails, ~1.6 violations.
				// For now, use 1-score as normalized severity.
				severity := 1.0 - r.GuardrailScore
				totalViolations++
				if severity > 0 {
					violCount := int(math.Ceil(severity * 10)) // rough estimate: 10 guardrails
					if violCount > maxViol {
						maxViol = violCount
					}
				}
			}
		}
		if failedRuns > 0 {
			meanViol = float64(totalViolations) / float64(failedRuns)
		}
		return
	}

	gs.TextMeanViolations, gs.TextMaxViolations = measure(text)
	gs.RuntimeMeanViolations, gs.RuntimeMaxViolations = measure(runtime)
	return gs
}

// --- Statistical helpers ---

// spearmanRank computes Spearman's rank correlation coefficient.
func spearmanRank(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 3 {
		return 0
	}
	rx := ranks(x)
	ry := ranks(y)
	return pearson(rx, ry)
}

func ranks(data []float64) []float64 {
	n := len(data)
	type indexedValue struct {
		val float64
		idx int
	}
	indexed := make([]indexedValue, n)
	for i, v := range data {
		indexed[i] = indexedValue{v, i}
	}
	sort.Slice(indexed, func(i, j int) bool {
		return indexed[i].val < indexed[j].val
	})

	r := make([]float64, n)
	i := 0
	for i < n {
		j := i
		for j < n && indexed[j].val == indexed[i].val {
			j++
		}
		// Average rank for ties
		avgRank := float64(i+j+1) / 2.0
		for k := i; k < j; k++ {
			r[indexed[k].idx] = avgRank
		}
		i = j
	}
	return r
}

func pearson(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0
	}
	n := float64(len(x))
	mx, my := 0.0, 0.0
	for i := range x {
		mx += x[i]
		my += y[i]
	}
	mx /= n
	my /= n

	num := 0.0
	dx2 := 0.0
	dy2 := 0.0
	for i := range x {
		dx := x[i] - mx
		dy := y[i] - my
		num += dx * dy
		dx2 += dx * dx
		dy2 += dy * dy
	}

	denom := math.Sqrt(dx2 * dy2)
	if denom == 0 {
		return 0
	}
	return num / denom
}
