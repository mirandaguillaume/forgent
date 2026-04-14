package experiment

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mirandaguillaume/forgent/internal/llm"
	"github.com/mirandaguillaume/forgent/pkg/dag"
)

// RunRuntimeExperiment executes all tasks as Forgent DAG agents.
// Each task is converted to a DAG where steps are nodes with consumes/produces
// contracts. The DAG engine handles ordering, parallelism, and retry.
func RunRuntimeExperiment(config ExperimentConfig) ([]TaskResult, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	total := len(config.Tasks) * config.RunsPerTask
	done := 0
	var results []TaskResult
	for i, task := range config.Tasks {
		skip := config.SkipMap[task.ID]
		for run := 0; run < config.RunsPerTask; run++ {
			done++
			if run < skip {
				fmt.Fprintf(os.Stderr, "  [runtime] task %d/%d (%s, %d steps) run %d/%d (%d/%d total) (skipped, resumed)\n",
					i+1, len(config.Tasks), task.ID, len(task.Steps), run+1, config.RunsPerTask, done, total)
				continue
			}
			fmt.Fprintf(os.Stderr, "  [runtime] task %d/%d (%s, %d steps) run %d/%d (%d/%d total)\n",
				i+1, len(config.Tasks), task.ID, len(task.Steps), run+1, config.RunsPerTask, done, total)
			result := runRuntimeTask(task, config.Provider)
			results = append(results, result)
			if config.OnResult != nil {
				config.OnResult(result, run)
			}
		}
	}
	return results, nil
}

func runRuntimeTask(task TaskDefinition, provider llm.Provider) TaskResult {
	start := time.Now()

	// Collect ALL node outputs (not just terminal) via a shared map + callback.
	allOutputs := make(map[string]any)
	var mu sync.Mutex
	var llmCalls int32

	// Wrap provider to count LLM calls
	countingProvider := &callCountingProvider{inner: provider, calls: &llmCalls}

	d, nodeOutputKeys, err := buildTaskDAG(task, countingProvider, func(nodeID string, out map[string]any) {
		mu.Lock()
		for k, v := range out {
			allOutputs[k] = v
		}
		mu.Unlock()
	})
	if err != nil {
		return TaskResult{
			TaskID:    task.ID,
			Condition: "runtime",
			Error:     fmt.Errorf("build DAG: %w", err),
		}
	}
	_ = nodeOutputKeys

	inputs := map[string]any{
		"task_input": task.Input,
	}

	_, err = d.Execute(context.Background(), dag.WithInputs(inputs))
	wallClock := time.Since(start).Milliseconds()

	if err != nil {
		return TaskResult{
			TaskID:      task.ID,
			Condition:   "runtime",
			WallClockMs: wallClock,
			LLMCalls:    int(llmCalls),
			Error:       fmt.Errorf("execute DAG: %w", err),
		}
	}

	result := evaluateRuntimeOutputs(task, allOutputs)
	result.WallClockMs = wallClock
	result.LLMCalls = int(llmCalls)
	return result
}

// callCountingProvider wraps an LLM provider to count calls.
type callCountingProvider struct {
	inner llm.Provider
	calls *int32
}

func (p *callCountingProvider) Complete(prompt string) (string, error) {
	atomic.AddInt32(p.calls, 1)
	return p.inner.Complete(prompt)
}

// outputCapture is called after each node completes with its outputs.
type outputCapture func(nodeID string, out map[string]any)

// buildTaskDAG constructs a DAG from a task definition.
// Each step becomes a node; consumes/produces contracts wire the topology.
// The capture callback receives all node outputs (not just terminal).
func buildTaskDAG(task TaskDefinition, provider llm.Provider, capture outputCapture) (*dag.DAG, map[string][]string, error) {
	var nodes []*dag.Node
	outputKeys := make(map[string][]string) // nodeID → produces keys

	for _, step := range task.Steps {
		stepCopy := step // capture for closure
		consumes := step.Consumes
		if len(consumes) == 0 && len(nodes) == 0 {
			// First step consumes the task input
			consumes = []string{"task_input"}
		}
		produces := step.Produces
		if len(produces) == 0 {
			produces = []string{step.ID + "_output"}
		}
		outputKeys[step.ID] = produces

		node := &dag.Node{
			ID:       step.ID,
			Kind:     dag.KindTask,
			Consumes: consumes,
			Produces: produces,
			Timeout:  60 * time.Second,
			Run: func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
				var inputParts []string
				for k, v := range inputs {
					if s, ok := v.(string); ok {
						inputParts = append(inputParts, fmt.Sprintf("%s:\n%s", k, s))
					}
				}

				prompt := fmt.Sprintf("## Step: %s\n\n%s\n\nInput:\n%s\n",
					stepCopy.Name, stepCopy.Prompt, strings.Join(inputParts, "\n\n"))

				response, err := provider.Complete(prompt)
				if err != nil {
					return nil, fmt.Errorf("step %s: %w", stepCopy.ID, err)
				}

				out := make(map[string]any)
				for _, p := range stepCopy.Produces {
					out[p] = response
				}

				if capture != nil {
					capture(stepCopy.ID, out)
				}
				return out, nil
			},
		}
		nodes = append(nodes, node)
	}

	d, err := dag.New(nodes...)
	return d, outputKeys, err
}

// evaluateRuntimeOutputs checks the DAG outputs against the task definition.
func evaluateRuntimeOutputs(task TaskDefinition, outputs map[string]any) TaskResult {
	result := TaskResult{
		TaskID:    task.ID,
		Condition: "runtime",
	}

	// Collect all output text
	var allOutput strings.Builder
	totalTokens := 0

	for _, step := range task.Steps {
		sr := StepResult{
			StepID:        step.ID,
			KeywordsTotal: len(step.ExpectedKeywords),
		}

		// Check each produce key for this step
		produces := step.Produces
		if len(produces) == 0 {
			produces = []string{step.ID + "_output"}
		}

		for _, key := range produces {
			if val, ok := outputs[key]; ok {
				if s, ok := val.(string); ok && len(s) > 0 {
					sr.Completed = true
					sr.Output = s
					allOutput.WriteString(s)
					allOutput.WriteString("\n")
					sr.TokensUsed = len(s) / 4
					totalTokens += sr.TokensUsed
				}
			}
		}

		// Check expected keywords
		if sr.Output != "" {
			lower := strings.ToLower(sr.Output)
			for _, kw := range step.ExpectedKeywords {
				if strings.Contains(lower, strings.ToLower(kw)) {
					sr.KeywordsHit++
				}
			}
		}

		if sr.KeywordsTotal == 0 && sr.Completed {
			sr.KeywordsHit = 1
			sr.KeywordsTotal = 1
		}

		result.StepResults = append(result.StepResults, sr)
	}

	result.TokensUsed = totalTokens

	// Completion
	completed := 0
	for _, sr := range result.StepResults {
		if sr.Completed {
			completed++
		}
	}
	result.Completed = completed == len(task.Steps)

	// Ordering: DAG enforces order, so if all completed, order is correct
	if result.Completed {
		result.OrderingScore = 1.0
	} else if len(task.Steps) > 0 {
		result.OrderingScore = float64(completed) / float64(len(task.Steps))
	}

	// Guardrails
	lowerAll := strings.ToLower(allOutput.String())
	respected := 0
	for _, g := range task.Guardrails {
		if checkGuardrail(lowerAll, g) {
			respected++
		}
	}
	if len(task.Guardrails) > 0 {
		result.GuardrailScore = float64(respected) / float64(len(task.Guardrails))
	} else {
		result.GuardrailScore = 1.0
	}

	return result
}
