package experiment

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mirandaguillaume/forgent/internal/llm"
)

// RunTextExperiment executes all tasks as single monolithic text-instruction prompts.
// Each task is converted to a complete prompt with all steps, constraints, and
// expected format — the LLM must follow the instructions in sequence within
// a single response.
func RunTextExperiment(config ExperimentConfig) ([]TaskResult, error) {
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
				fmt.Fprintf(os.Stderr, "  [text] task %d/%d (%s) run %d/%d (%d/%d total) (skipped, resumed)\n",
					i+1, len(config.Tasks), task.ID, run+1, config.RunsPerTask, done, total)
				continue
			}
			fmt.Fprintf(os.Stderr, "  [text] task %d/%d (%s) run %d/%d (%d/%d total)\n",
				i+1, len(config.Tasks), task.ID, run+1, config.RunsPerTask, done, total)
			result := runTextTask(task, config.Provider)
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

func runTextTask(task TaskDefinition, provider llm.Provider) TaskResult {
	start := time.Now()
	prompt := buildTextPrompt(task)

	response, err := provider.Complete(prompt)
	wallClock := time.Since(start).Milliseconds()

	if err != nil {
		return TaskResult{
			TaskID:      task.ID,
			Condition:   "text",
			WallClockMs: wallClock,
			LLMCalls:    1,
			Error:       err,
		}
	}

	result := evaluateTextResponse(task, response, len(prompt)+len(response))
	result.WallClockMs = wallClock
	result.LLMCalls = 1 // text mode always makes exactly 1 call
	result.CombinedOutput = response
	return result
}

// buildTextPrompt creates a single monolithic prompt containing all steps,
// constraints, and expected format. This is the "text instruction" condition.
func buildTextPrompt(task TaskDefinition) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# Task: %s\n\n", task.Name))
	sb.WriteString(fmt.Sprintf("%s\n\n", task.Description))

	if task.Input != "" {
		sb.WriteString("## Input\n\n")
		sb.WriteString(task.Input)
		sb.WriteString("\n\n")
	}

	// For tool-based tasks, include workspace file listing in the prompt
	// (text condition gets no tool access — just the file contents as context)
	if task.Workspace != nil {
		sb.WriteString("## Available Files\n\n")
		for _, f := range task.Workspace.Files {
			sb.WriteString(fmt.Sprintf("### %s\n```\n%s\n```\n\n", f.Path, f.Content))
		}
	}

	sb.WriteString("## Steps (execute in order)\n\n")
	for i, step := range task.Steps {
		sb.WriteString(fmt.Sprintf("### Step %d: %s\n", i+1, step.Name))
		body := stepTextBody(step)
		sb.WriteString(body)
		sb.WriteString("\n\n")
	}

	if len(task.Guardrails) > 0 {
		sb.WriteString("## Constraints\n\n")
		for _, g := range task.Guardrails {
			sb.WriteString(fmt.Sprintf("- %s\n", g))
		}
		sb.WriteString("\n")
	}

	sb.WriteString("## Output Format\n\n")
	sb.WriteString("Respond with clearly labeled sections for each step, using the step names as headers.\n")

	return sb.String()
}

// stepTextBody returns the instruction body for a step in the text prompt.
// For steps with no Prompt (e.g. kind: filter with predicate, kind: static),
// synthesizes a brief description so the LLM knows what to do.
func stepTextBody(step StepDef) string {
	if step.Prompt != "" {
		return step.Prompt
	}
	switch step.Kind {
	case StepKindFilter:
		if step.Predicate != "" {
			return fmt.Sprintf("Filter the items from the previous step. Keep only items that satisfy: %s\nList the kept items.", step.Predicate)
		}
	case StepKindMap:
		return "Process each item from the previous step individually. List all results."
	case StepKindStatic:
		if step.Template != "" {
			return "Apply the template to the inputs and output the result."
		}
	}
	return fmt.Sprintf("Execute step: %s", step.Name)
}

// evaluateTextResponse checks the LLM response against the task definition.
func evaluateTextResponse(task TaskDefinition, response string, totalTokens int) TaskResult {
	lower := strings.ToLower(response)

	result := TaskResult{
		TaskID:     task.ID,
		Condition:  "text",
		TokensUsed: totalTokens / 4, // rough token estimate: 4 chars/token
	}

	// Evaluate each step
	lastStepIdx := -1
	inOrder := 0
	for _, step := range task.Steps {
		sr := evaluateStep(step, lower)
		sr.TokensUsed = result.TokensUsed / len(task.Steps) // distribute evenly
		result.StepResults = append(result.StepResults, sr)

		if sr.Completed {
			stepIdx := strings.Index(lower, strings.ToLower(step.Name))
			if stepIdx > lastStepIdx {
				inOrder++
			}
			if stepIdx >= 0 {
				lastStepIdx = stepIdx
			}
		}
	}

	// Completion: all steps produced meaningful output
	completedSteps := 0
	for _, sr := range result.StepResults {
		if sr.Completed {
			completedSteps++
		}
	}
	result.Completed = completedSteps == len(task.Steps)

	// Ordering: fraction of steps that appeared in correct order
	if len(task.Steps) > 0 {
		result.OrderingScore = float64(inOrder) / float64(len(task.Steps))
	}

	// Guardrails: check each constraint
	guardrailsRespected := 0
	for _, g := range task.Guardrails {
		if checkGuardrail(lower, g) {
			guardrailsRespected++
		}
	}
	if len(task.Guardrails) > 0 {
		result.GuardrailScore = float64(guardrailsRespected) / float64(len(task.Guardrails))
	} else {
		result.GuardrailScore = 1.0 // no guardrails = perfect score
	}

	// RecallScore: fraction of ground-truth issues found in combined response.
	if len(task.ExpectedIssues) > 0 {
		result.HasExpectedIssues = true
		result.RecallScore = computeRecall(task, response)
	}

	return result
}

func evaluateStep(step StepDef, lowerResponse string) StepResult {
	sr := StepResult{StepID: step.ID}
	// Normalize unicode dashes (em dash U+2014, en dash U+2013) to ASCII hyphen
	// before matching so LLM responses that substitute "-" still score correctly.
	normName := normalizeDashes(strings.ToLower(step.Name))
	normResp := normalizeDashes(lowerResponse)
	if strings.Contains(normResp, normName) || strings.Contains(normResp, step.ID) {
		sr.Completed = true
		return sr
	}
	// Fallback: check the prefix of the step name (before qualifiers like " - ",
	// " (" etc.) so LLMs that abbreviate long names still score correctly.
	if prefix := stepNamePrefix(normName); prefix != normName && strings.Contains(normResp, prefix) {
		sr.Completed = true
	}
	return sr
}

// stepNamePrefix returns the significant root of a step name, stripping
// trailing qualifiers introduced by " - " or " (" (e.g. description clauses,
// parenthetical notes). Used as a fallback match in evaluateStep when the
// LLM omits the qualifier but writes the root name correctly.
func stepNamePrefix(name string) string {
	for _, sep := range []string{" - ", " ("} {
		if i := strings.Index(name, sep); i > 0 {
			return name[:i]
		}
	}
	return name
}

// normalizeDashes replaces em dash and en dash with ASCII hyphen-minus.
func normalizeDashes(s string) string {
	s = strings.ReplaceAll(s, "—", "-") // em dash —
	s = strings.ReplaceAll(s, "–", "-") // en dash –
	return s
}

// computeRecall checks how many of the task's expected_issues are present in
// combinedOutput (case-insensitive). Returns 0.0 when no expected issues are
// declared (not 1.0 — callers must check task.ExpectedIssues != nil).
func computeRecall(task TaskDefinition, combinedOutput string) float64 {
	if len(task.ExpectedIssues) == 0 {
		return 0.0
	}
	lower := strings.ToLower(combinedOutput)
	found := 0
	for _, issue := range task.ExpectedIssues {
		for _, kw := range issue.MatchAny {
			if strings.Contains(lower, strings.ToLower(kw)) {
				found++
				break
			}
		}
	}
	return float64(found) / float64(len(task.ExpectedIssues))
}

// checkGuardrail verifies a guardrail constraint is respected.
// Guardrails prefixed with "!" are negative constraints (must NOT appear).
func checkGuardrail(lowerResponse, guardrail string) bool {
	if strings.HasPrefix(guardrail, "!") {
		// Negative constraint: must NOT contain this
		forbidden := strings.ToLower(strings.TrimPrefix(guardrail, "!"))
		return !strings.Contains(lowerResponse, forbidden)
	}
	// Positive constraint: must contain this
	return strings.Contains(lowerResponse, strings.ToLower(guardrail))
}
