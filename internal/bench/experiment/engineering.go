package experiment

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/mirandaguillaume/forgent/internal/llm"
)

// FaultyProvider wraps an LLM provider to inject transient failures at a given rate.
// Used to measure fault tolerance: the runtime retries individual steps while
// the text condition loses everything on failure.
type FaultyProvider struct {
	Inner    llm.Provider
	FailRate float64 // 0.0-1.0: probability of failure per call
	rng      *rand.Rand
}

// NewFaultyProvider creates a provider that fails failRate% of calls.
func NewFaultyProvider(inner llm.Provider, failRate float64, seed int64) *FaultyProvider {
	return &FaultyProvider{
		Inner:    inner,
		FailRate: failRate,
		rng:      rand.New(rand.NewSource(seed)),
	}
}

func (p *FaultyProvider) Complete(prompt string) (string, error) {
	if p.rng.Float64() < p.FailRate {
		return "", fmt.Errorf("transient LLM error (injected)")
	}
	return p.Inner.Complete(prompt)
}

// MeasureFaultTolerance runs the experiment N times with error injection and
// returns the success rate for each condition.
func MeasureFaultTolerance(config ExperimentConfig, failRate float64, runs int) (textRate, runtimeRate float64) {
	textSuccess := 0
	runtimeSuccess := 0

	for i := 0; i < runs; i++ {
		seed := config.Seed + int64(i)
		faultyConfig := ExperimentConfig{
			Tasks:       config.Tasks,
			Provider:    NewFaultyProvider(config.Provider, failRate, seed),
			Model:       config.Model,
			RunsPerTask: 1,
			Seed:        seed,
		}

		textResults, err := RunTextExperiment(faultyConfig)
		if err == nil {
			allCompleted := true
			for _, r := range textResults {
				if r.Error != nil {
					allCompleted = false
					break
				}
			}
			if allCompleted {
				textSuccess++
			}
		}

		// Runtime gets retry via DAG wrappers — use a fresh faulty provider
		faultyConfig.Provider = NewFaultyProvider(config.Provider, failRate, seed+1000)
		runtimeResults, err := RunRuntimeExperiment(faultyConfig)
		if err == nil {
			allCompleted := true
			for _, r := range runtimeResults {
				if r.Error != nil {
					allCompleted = false
					break
				}
			}
			if allCompleted {
				runtimeSuccess++
			}
		}
	}

	if runs > 0 {
		textRate = float64(textSuccess) / float64(runs)
		runtimeRate = float64(runtimeSuccess) / float64(runs)
	}
	return
}

// MeasureCrossContamination computes how similar the outputs of "independent"
// steps are in text mode. In DAG mode, parallel steps receive only their
// declared inputs — they can't see each other. In text mode, all steps are
// in the same response, so later steps may reference earlier ones.
//
// Returns Jaccard similarity (0.0 = fully independent, 1.0 = identical).
func MeasureCrossContamination(textResults []TaskResult) float64 {
	var similarities []float64

	for _, tr := range textResults {
		if len(tr.StepResults) < 2 {
			continue
		}

		// Compare consecutive step outputs for shared unique terms
		for i := 0; i < len(tr.StepResults)-1; i++ {
			a := tr.StepResults[i].Output
			b := tr.StepResults[i+1].Output
			if a == "" || b == "" {
				continue
			}
			sim := jaccardSimilarity(tokenize(a), tokenize(b))
			similarities = append(similarities, sim)
		}
	}

	if len(similarities) == 0 {
		return 0
	}

	sum := 0.0
	for _, s := range similarities {
		sum += s
	}
	return sum / float64(len(similarities))
}

func tokenize(text string) map[string]bool {
	tokens := make(map[string]bool)
	for _, word := range strings.Fields(strings.ToLower(text)) {
		// Skip very short words and common stops
		if len(word) > 3 {
			tokens[word] = true
		}
	}
	return tokens
}

func jaccardSimilarity(a, b map[string]bool) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}

	intersection := 0
	for k := range a {
		if b[k] {
			intersection++
		}
	}

	union := len(a)
	for k := range b {
		if !a[k] {
			union++
		}
	}

	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}
