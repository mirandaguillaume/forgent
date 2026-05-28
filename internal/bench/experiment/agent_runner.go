package experiment

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mirandaguillaume/forgent/internal/llm"
)

// RunAgentExperiment executes all tasks as multi-turn sequential agents.
//
// Each step gets its own LLM call (unlike text: 1 call total) with accumulated
// context from all prior steps visible in the prompt. No structural enforcement
// (unlike runtime: DAG contracts). Deterministic steps (kind: filter with
// predicate, kind: static) are applied inline without an LLM call.
//
// This condition isolates the structural benefit of the runtime harness from
// the benefit of simply having multiple focused LLM calls.
func RunAgentExperiment(config ExperimentConfig) ([]TaskResult, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	total := len(config.Tasks) * config.RunsPerTask
	done := 0
	label := config.Condition
	if label == "" {
		label = "agent"
	}
	var results []TaskResult
	for i, task := range config.Tasks {
		skip := config.SkipMap[task.ID]
		for run := 0; run < config.RunsPerTask; run++ {
			done++
			if run < skip {
				fmt.Fprintf(os.Stderr, "  [%s] task %d/%d (%s, %d steps) run %d/%d (%d/%d total) (skipped, resumed)\n",
					label, i+1, len(config.Tasks), task.ID, len(task.Steps), run+1, config.RunsPerTask, done, total)
				continue
			}
			fmt.Fprintf(os.Stderr, "  [%s] task %d/%d (%s, %d steps) run %d/%d (%d/%d total)\n",
				label, i+1, len(config.Tasks), task.ID, len(task.Steps), run+1, config.RunsPerTask, done, total)
			result := runAgentTask(task, config.Provider, label)
			results = append(results, result)
			if config.OnResult != nil {
				config.OnResult(result, run)
			}
			if IsAPIFatalError(result.Error) {
				fmt.Fprintf(os.Stderr, "  ✗ fatal API error, aborting experiment: %v\n", result.Error)
				return results, fmt.Errorf("%w: %v", ErrAPIFatal, result.Error)
			}
		}
	}
	return results, nil
}

func runAgentTask(task TaskDefinition, provider llm.Provider, condition string) TaskResult {
	start := time.Now()

	// ctxBuf accumulates all prior step outputs — each new step receives the
	// full history, giving the agent growing context across turns.
	var ctxBuf strings.Builder
	ctxBuf.WriteString(fmt.Sprintf("# Task: %s\n\n%s\n\n", task.Name, task.Description))
	if task.Input != "" {
		ctxBuf.WriteString(fmt.Sprintf("## Input\n\n%s\n\n", task.Input))
	}
	if task.Workspace != nil {
		ctxBuf.WriteString("## Available Files\n\n")
		for _, f := range task.Workspace.Files {
			ctxBuf.WriteString(fmt.Sprintf("### %s\n```\n%s\n```\n\n", f.Path, f.Content))
		}
	}

	allOutputs := make(map[string]any)
	var stepResults []StepResult
	var totalTokens int
	var llmCalls int
	var firstErr error

	for _, step := range task.Steps {
		switch {
		case step.Kind == StepKindFilter && step.Predicate != "":
			// Deterministic filter: apply predicate to whatever the previous step collected.
			kept := agentApplyPredicate(step, allOutputs)
			allOutputs[agentProducesKey(step)] = kept
			ctxBuf.WriteString(fmt.Sprintf("## %s Result\n\n%s\n\n", step.Name, kept))
			stepResults = append(stepResults, StepResult{StepID: step.ID, Completed: true})

		case step.Kind == StepKindStatic:
			// No LLM needed — template execution would go here; emit placeholder.
			allOutputs[agentProducesKey(step)] = "[static step]"
			stepResults = append(stepResults, StepResult{StepID: step.ID, Completed: true})

		default:
			// LLM call: accumulated context + this step's instructions.
			instruction := stepTextBody(step)
			prompt := ctxBuf.String() + fmt.Sprintf("## Now execute: %s\n\n%s\n", step.Name, instruction)
			llmCalls++
			totalTokens += len(prompt) / 4
			response, err := provider.Complete(prompt)
			totalTokens += len(response) / 4
			if err != nil {
				if firstErr == nil {
					firstErr = err
				}
				stepResults = append(stepResults, StepResult{StepID: step.ID, Completed: false})
				if IsAPIFatalError(err) {
					break
				}
				continue
			}
			allOutputs[agentProducesKey(step)] = response
			ctxBuf.WriteString(fmt.Sprintf("## %s Result\n\n%s\n\n", step.Name, response))
			stepResults = append(stepResults, StepResult{StepID: step.ID, Completed: true})
		}
	}

	wallClock := time.Since(start).Milliseconds()

	if firstErr != nil && IsAPIFatalError(firstErr) {
		return TaskResult{
			TaskID:    task.ID,
			Condition: condition,
			Error:     fmt.Errorf("execute agent: %w", firstErr),
		}
	}

	// Evaluate using the same output-based criteria as the runtime condition.
	result := evaluateAgentOutputs(task, stepResults, allOutputs, totalTokens)
	result.Condition = condition
	result.WallClockMs = wallClock
	result.LLMCalls = llmCalls
	if firstErr != nil {
		result.Error = firstErr
	}
	return result
}

// agentProducesKey returns the first produces key for a step, or the step ID
// as fallback. Used to store each step's output in allOutputs.
func agentProducesKey(step StepDef) string {
	if len(step.Produces) > 0 {
		return step.Produces[0].Key
	}
	return step.ID
}

// agentApplyPredicate applies the step's Predicate to the list accumulated so
// far. It looks for the consumed slot value in allOutputs; if it's a []any
// (from a list_schema step) it evaluates per-item, otherwise treats the string
// output of the previous step as a single item.
func agentApplyPredicate(step StepDef, allOutputs map[string]any) string {
	var items []string

	// Find the consumed slot value.
	for _, c := range step.Consumes {
		if v, ok := allOutputs[c.Key]; ok {
			switch val := v.(type) {
			case []any:
				for _, item := range val {
					items = append(items, fmt.Sprintf("%v", item))
				}
			case string:
				// Split on newlines to treat each line as an item.
				for _, line := range strings.Split(val, "\n") {
					if strings.TrimSpace(line) != "" {
						items = append(items, line)
					}
				}
			}
			break
		}
	}

	var kept []string
	for _, item := range items {
		ok, err := evalPredicate(step.Predicate, item)
		if err != nil {
			// If predicate fails (e.g. no SCORE: in item), skip silently.
			continue
		}
		if ok {
			kept = append(kept, item)
		}
	}
	if len(kept) == 0 {
		return "(no items passed the filter)"
	}
	return strings.Join(kept, "\n")
}

// evaluateAgentOutputs scores an agent run using the same guardrail and
// ordering logic as the text runner, but step completion is taken from the
// per-step results collected during execution (not from name-presence in a
// single response).
func evaluateAgentOutputs(task TaskDefinition, stepResults []StepResult, allOutputs map[string]any, totalTokens int) TaskResult {
	result := TaskResult{
		TaskID:      task.ID,
		StepResults: stepResults,
		TokensUsed:  totalTokens,
	}

	// Combine all step outputs into one blob for guardrail evaluation.
	var combined strings.Builder
	for _, v := range allOutputs {
		combined.WriteString(fmt.Sprintf("%v\n", v))
	}
	lower := strings.ToLower(combined.String())

	// Completion: all steps must have succeeded.
	completedSteps := 0
	for _, sr := range stepResults {
		if sr.Completed {
			completedSteps++
		}
	}
	result.Completed = completedSteps == len(task.Steps)

	// Ordering: agent always executes steps in declaration order — score is 1.0
	// if all steps completed, proportional otherwise.
	if len(task.Steps) > 0 {
		result.OrderingScore = float64(completedSteps) / float64(len(task.Steps))
	}

	// Task-level guardrails: checked against combined output (same semantics as runtime).
	guardrailsRespected := 0
	for _, g := range task.Guardrails {
		if checkGuardrail(lower, g) {
			guardrailsRespected++
		}
	}
	totalGuardrails := len(task.Guardrails)
	// Step-level guardrails: checked per-step.
	for _, step := range task.Steps {
		if len(step.Guardrails) == 0 {
			continue
		}
		stepOut := strings.ToLower(fmt.Sprintf("%v", allOutputs[agentProducesKey(step)]))
		for _, g := range step.Guardrails {
			totalGuardrails++
			if checkGuardrail(stepOut, g) {
				guardrailsRespected++
			}
		}
	}
	if totalGuardrails > 0 {
		result.GuardrailScore = float64(guardrailsRespected) / float64(totalGuardrails)
	} else {
		result.GuardrailScore = 1.0
	}

	// IssuesFound: count items that passed the kind:filter step (if any).
	for _, step := range task.Steps {
		if step.Kind != StepKindFilter {
			continue
		}
		key := agentProducesKey(step)
		if v, ok := allOutputs[key]; ok {
			text := fmt.Sprintf("%v", v)
			count := 0
			for _, line := range strings.Split(text, "\n") {
				if strings.TrimSpace(line) != "" {
					count++
				}
			}
			result.IssuesFound = count
		}
	}

	// RecallScore: fraction of ground-truth issues found in combined output.
	if len(task.ExpectedIssues) > 0 {
		result.HasExpectedIssues = true
		result.RecallScore = computeRecall(task, combined.String())
	}

	result.CombinedOutput = combined.String()
	return result
}

