package bench

import (
	"fmt"
	"strings"

	"github.com/mirandaguillaume/forgent/internal/generator"
	"github.com/mirandaguillaume/forgent/internal/llm"
	"github.com/mirandaguillaume/forgent/pkg/model"
)

// PassStrategy generates the review prompt for a given pass number.
type PassStrategy interface {
	// ReviewPrompt returns the prompt for pass N (1-indexed).
	// history contains all previous (pass, response) pairs.
	ReviewPrompt(pass int, basePrompt string, code string, history []PassRecord) string
}

// GenericPassStrategy handles baselines and non-staged contestants.
// Pass 1: normal review. Pass 2+: self-critique with accumulated history.
type GenericPassStrategy struct{}

func (g *GenericPassStrategy) ReviewPrompt(pass int, basePrompt string, code string, history []PassRecord) string {
	if pass == 1 {
		return fmt.Sprintf("%s\n\nReview the following code:\n```\n%s\n```", basePrompt, code)
	}

	var historyBlock strings.Builder
	for _, h := range history {
		fmt.Fprintf(&historyBlock, "\n--- Review Pass %d ---\n%s\n", h.Pass, h.Response)
	}

	return fmt.Sprintf(`%s

Review the following code:
`+"```"+`
%s
`+"```"+`

Your previous review passes:
%s
Pass %d instructions: Critically re-examine your review. Look for:
1. Issues you missed entirely
2. False positives to retract
3. Severity assessments that need adjustment
4. Vague findings that need concrete evidence

Produce an improved, consolidated review incorporating all valid findings.`,
		basePrompt, code, historyBlock.String(), pass)
}

// StagedPassStrategy maps passes to stage-level guidance from agent YAML.
// Pass 1 is standard review. Pass 2+ uses stage definitions to focus each pass.
type StagedPassStrategy struct {
	Stages     []model.Stage
	SkillSteps map[string][]string // skill name → strategy.steps
}

func (s *StagedPassStrategy) ReviewPrompt(pass int, basePrompt string, code string, history []PassRecord) string {
	if pass == 1 {
		return fmt.Sprintf("%s\n\nReview the following code:\n```\n%s\n```", basePrompt, code)
	}

	var historyBlock strings.Builder
	for _, h := range history {
		fmt.Fprintf(&historyBlock, "\n--- Review Pass %d ---\n%s\n", h.Pass, h.Response)
	}

	// Map pass to stage (pass 2 = stage index 1, etc.)
	stageIdx := pass - 1
	if stageIdx >= len(s.Stages) {
		stageIdx = len(s.Stages) - 1
	}
	stage := s.Stages[stageIdx]

	// Build stage-specific guidance from skill steps
	var guidance strings.Builder
	fmt.Fprintf(&guidance, "Stage: %s (strategy: %s)\n", stage.Name, stage.Strategy)
	guidance.WriteString("Focus areas for this pass:\n")
	for _, skillName := range stage.Skills {
		if steps, ok := s.SkillSteps[skillName]; ok && len(steps) > 0 {
			fmt.Fprintf(&guidance, "  %s:\n", skillName)
			for _, step := range steps {
				fmt.Fprintf(&guidance, "    - %s\n", step)
			}
		}
	}

	return fmt.Sprintf(`%s

Review the following code:
`+"```"+`
%s
`+"```"+`

Your previous review passes:
%s
Pass %d — %s
%s
Produce an improved, consolidated review incorporating all valid findings from previous passes plus this pass's focus.`,
		basePrompt, code, historyBlock.String(), pass, stage.Name, guidance.String())
}

// ResolvePassStrategy determines which strategy a contestant should use.
func ResolvePassStrategy(c H2HContestant) PassStrategy {
	if len(c.Stages) > 0 && c.SkillSteps != nil {
		return &StagedPassStrategy{
			Stages:     c.Stages,
			SkillSteps: c.SkillSteps,
		}
	}
	return &GenericPassStrategy{}
}

// evalTaskMultiPass runs N review passes then judges the final output.
// For passes=1, delegates to the existing evalTask for backward compatibility.
func evalTaskMultiPass(contestant H2HContestant, task H2HTask, provider llm.Provider, passes int) H2HTaskResult {
	if passes <= 1 {
		return evalTask(contestant, task, provider)
	}

	strategy := ResolvePassStrategy(contestant)

	// Resolve code/diff once (same as BuildReviewPrompt)
	code := task.Code
	if code == "" && task.URL != "" {
		diff, err := FetchPRDiff(task.URL)
		if err != nil {
			code = fmt.Sprintf("[Failed to fetch diff: %v]\nPR: %s — %s", err, task.URL, task.Title)
		} else {
			code = diff
		}
	}

	var history []PassRecord
	totalInputWords := 0
	totalOutputWords := 0
	llmCalls := 0

	for pass := 1; pass <= passes; pass++ {
		prompt := strategy.ReviewPrompt(pass, contestant.Prompt, code, history)
		inputWords := generator.CountWords(prompt)

		response, err := provider.Complete(prompt)
		llmCalls++
		if err != nil {
			return H2HTaskResult{
				ContestantName: contestant.Name,
				TaskID:         task.ID,
				Reasoning:      fmt.Sprintf("error on pass %d: %v", pass, err),
				InputWords:     totalInputWords + inputWords,
				OutputWords:    totalOutputWords,
				LLMCalls:       llmCalls,
				Passes:         pass,
				PassDetails:    history,
			}
		}

		outputWords := generator.CountWords(response)
		totalInputWords += inputWords
		totalOutputWords += outputWords

		history = append(history, PassRecord{
			Pass:        pass,
			Response:    response,
			InputWords:  inputWords,
			OutputWords: outputWords,
		})
	}

	// Judge evaluates only the final review
	finalResponse := history[len(history)-1].Response
	judgeResult, err := JudgeH2HResponse(provider, task, finalResponse)
	if err != nil {
		return H2HTaskResult{
			ContestantName: contestant.Name,
			TaskID:         task.ID,
			Reasoning:      fmt.Sprintf("judge error: %v", err),
			InputWords:     totalInputWords,
			OutputWords:    totalOutputWords,
			LLMCalls:       llmCalls,
			Passes:         passes,
			PassDetails:    history,
		}
	}

	return H2HTaskResult{
		ContestantName: contestant.Name,
		TaskID:         task.ID,
		Score:          judgeResult.WeightedScore,
		Reasoning:      judgeResult.Reasoning,
		Severity:       judgeResult.Severity,
		InputWords:     totalInputWords + judgeResult.JudgeInputWords,
		OutputWords:    totalOutputWords + judgeResult.JudgeOutputWords,
		LLMCalls:       llmCalls + 1,
		Passes:         passes,
		PassDetails:    history,
	}
}
