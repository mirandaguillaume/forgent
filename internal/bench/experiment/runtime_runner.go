package experiment

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"
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
//
// Three step kinds are supported:
//   - llm (default): calls the LLM provider, guardrails enforced by code after each call
//   - static: executes a Go text/template against inputs, no LLM call
//   - assert: runs a guardrail check against inputs, fails if violated
func buildTaskDAG(task TaskDefinition, provider llm.Provider, capture outputCapture) (*dag.DAG, map[string][]string, error) {
	var nodes []*dag.Node
	outputKeys := make(map[string][]string) // nodeID → produces keys

	for _, step := range task.Steps {
		stepCopy := step // capture for closure
		consumes := step.Consumes
		if len(consumes) == 0 && len(nodes) == 0 {
			consumes = []string{"task_input"}
		}
		produces := step.Produces
		if len(produces) == 0 {
			produces = []string{step.ID + "_output"}
		}
		outputKeys[step.ID] = produces

		var runFn func(ctx context.Context, inputs map[string]any) (map[string]any, error)

		switch stepCopy.Kind {
		case StepKindStatic:
			// Static: execute Go template against inputs — no LLM, no tokens
			runFn = func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
				tmpl, err := template.New(stepCopy.ID).Parse(stepCopy.Template)
				if err != nil {
					return nil, fmt.Errorf("step %s: parse template: %w", stepCopy.ID, err)
				}
				data := map[string]any{"inputs": inputs}
				var buf bytes.Buffer
				if err := tmpl.Execute(&buf, data); err != nil {
					return nil, fmt.Errorf("step %s: execute template: %w", stepCopy.ID, err)
				}
				out := make(map[string]any)
				for _, p := range stepCopy.Produces {
					out[p] = buf.String()
				}
				if capture != nil {
					capture(stepCopy.ID, out)
				}
				return out, nil
			}

		case StepKindAssert:
			// Assert: verify guardrails/checks against inputs — no LLM, fail on violation
			runFn = func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
				// Aggregate all input text
				var allText strings.Builder
				for _, v := range inputs {
					if s, ok := v.(string); ok {
						allText.WriteString(s)
						allText.WriteString("\n")
					}
				}
				lower := strings.ToLower(allText.String())

				// Run the named check
				switch stepCopy.Check {
				case "guardrails_respected":
					for _, g := range task.Guardrails {
						if !checkGuardrail(lower, g) {
							return nil, fmt.Errorf("step %s: guardrail violated: %s", stepCopy.ID, g)
						}
					}
				case "all_inputs_present":
					for k, v := range inputs {
						if s, ok := v.(string); ok && strings.TrimSpace(s) == "" {
							return nil, fmt.Errorf("step %s: empty input: %s", stepCopy.ID, k)
						}
					}
				case "no_empty_steps":
					for k, v := range inputs {
						if s, ok := v.(string); ok && len(strings.TrimSpace(s)) < 10 {
							return nil, fmt.Errorf("step %s: suspiciously short input %s (%d chars)", stepCopy.ID, k, len(s))
						}
					}
				default:
					// Unknown check — pass through (lenient)
				}

				out := make(map[string]any)
				for _, p := range stepCopy.Produces {
					out[p] = allText.String()
				}
				if capture != nil {
					capture(stepCopy.ID, out)
				}
				return out, nil
			}

		default:
			// LLM step: call provider with guardrail enforcement + retry
			runFn = func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
				var inputParts []string
				for k, v := range inputs {
					if s, ok := v.(string); ok {
						inputParts = append(inputParts, fmt.Sprintf("%s:\n%s", k, s))
					}
				}

				// Build prompt with task context + guardrails (was missing before!)
				var sb strings.Builder
				sb.WriteString(fmt.Sprintf("# Task: %s\n\n", task.Name))
				sb.WriteString(fmt.Sprintf("## Step: %s\n\n%s\n\n", stepCopy.Name, stepCopy.Prompt))
				if len(task.Guardrails) > 0 {
					sb.WriteString("## Constraints (MUST respect)\n\n")
					for _, g := range task.Guardrails {
						sb.WriteString(fmt.Sprintf("- %s\n", g))
					}
					sb.WriteString("\n")
				}
				sb.WriteString("## Input\n\n")
				sb.WriteString(strings.Join(inputParts, "\n\n"))
				sb.WriteString("\n")

				// Try up to 3 times, checking guardrails after each attempt
				maxAttempts := 3
				var lastResponse string
				for attempt := 0; attempt < maxAttempts; attempt++ {
					response, err := provider.Complete(sb.String())
					if err != nil {
						return nil, fmt.Errorf("step %s attempt %d: %w", stepCopy.ID, attempt+1, err)
					}
					lastResponse = response

					// Check guardrails by code
					if len(task.Guardrails) > 0 {
						lower := strings.ToLower(response)
						allPassed := true
						for _, g := range task.Guardrails {
							if !checkGuardrail(lower, g) {
								allPassed = false
								break
							}
						}
						if allPassed {
							break // guardrails passed, use this response
						}
						if attempt < maxAttempts-1 {
							// Retry with hint
							sb.WriteString(fmt.Sprintf("\n\n[Previous attempt violated constraints. Retry %d/%d. Strictly respect ALL constraints listed above.]\n", attempt+2, maxAttempts))
						}
					} else {
						break // no guardrails to check
					}
				}

				out := make(map[string]any)
				for _, p := range stepCopy.Produces {
					out[p] = lastResponse
				}
				if capture != nil {
					capture(stepCopy.ID, out)
				}
				return out, nil
			}
		}

		node := &dag.Node{
			ID:       step.ID,
			Kind:     dag.KindTask,
			Consumes: consumes,
			Produces: produces,
			Timeout:  120 * time.Second,
			Run:      runFn,
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
