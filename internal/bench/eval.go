package bench

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/mirandaguillaume/forgent/internal/llm"
	"gopkg.in/yaml.v3"
)

// EvalTask represents a code review evaluation task.
type EvalTask struct {
	ID               string   `yaml:"id"`
	Input            string   `yaml:"input"`
	ExpectedCriteria []string `yaml:"expected_criteria"`
}

// EvalResult holds the outcome of a single evaluation task.
type EvalResult struct {
	TaskID          string
	ComposedScore   int
	MonolithicScore int
	ComposedWins    bool
}

// AggregateEvalResult holds the aggregate outcome of all evaluation tasks.
type AggregateEvalResult struct {
	Tasks        int
	ComposedWins int
	Ties         int
	WinRate      float64
	Results      []EvalResult
}

// LoadEvalTasks reads evaluation tasks from a YAML file.
func LoadEvalTasks(path string) ([]EvalTask, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var tasks []EvalTask
	if err := yaml.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("parse eval tasks: %w", err)
	}
	return tasks, nil
}

// RunEval runs the LLM-as-judge evaluation comparing composed vs monolithic prompts.
func RunEval(tasks []EvalTask, composedPrompt, monolithicPrompt string, provider llm.Provider) (*AggregateEvalResult, error) {
	agg := &AggregateEvalResult{Tasks: len(tasks)}

	for _, task := range tasks {
		// Get composed review
		composedReview, err := provider.Complete(fmt.Sprintf(
			"%s\n\nReview the following code:\n```\n%s\n```",
			composedPrompt, task.Input))
		if err != nil {
			return nil, fmt.Errorf("composed review for %s: %w", task.ID, err)
		}

		// Get monolithic review
		monolithicReview, err := provider.Complete(fmt.Sprintf(
			"%s\n\nReview the following code:\n```\n%s\n```",
			monolithicPrompt, task.Input))
		if err != nil {
			return nil, fmt.Errorf("monolithic review for %s: %w", task.ID, err)
		}

		// Judge both reviews
		judgeResult, err := judge(provider, task, composedReview, monolithicReview)
		if err != nil {
			return nil, fmt.Errorf("judge for %s: %w", task.ID, err)
		}

		result := EvalResult{
			TaskID:          task.ID,
			ComposedScore:   judgeResult.ComposedScore,
			MonolithicScore: judgeResult.MonolithicScore,
			ComposedWins:    judgeResult.ComposedScore > judgeResult.MonolithicScore,
		}

		if result.ComposedWins {
			agg.ComposedWins++
		} else if judgeResult.ComposedScore == judgeResult.MonolithicScore {
			agg.Ties++
		}

		agg.Results = append(agg.Results, result)
	}

	if agg.Tasks > 0 {
		agg.WinRate = float64(agg.ComposedWins) / float64(agg.Tasks) * 100
	}

	return agg, nil
}

type judgeResponse struct {
	ComposedScore   int    `json:"composed_score"`
	MonolithicScore int    `json:"monolithic_score"`
	Reasoning       string `json:"reasoning"`
}

func judge(provider llm.Provider, task EvalTask, composedReview, monolithicReview string) (*judgeResponse, error) {
	prompt := fmt.Sprintf(`You are a code review quality judge. Score two reviews of the same code.

Code being reviewed:
%s

Expected criteria the review should cover:
%s

Review A (Composed Agent):
%s

Review B (Monolithic Agent):
%s

Score each review from 0-100 based on:
- Coverage of expected criteria
- Actionability of suggestions
- Accuracy of findings
- Clarity of explanation

Respond with ONLY valid JSON (no markdown):
{"composed_score": <0-100>, "monolithic_score": <0-100>, "reasoning": "<brief explanation>"}`,
		task.Input,
		strings.Join(task.ExpectedCriteria, ", "),
		composedReview,
		monolithicReview,
	)

	response, err := provider.Complete(prompt)
	if err != nil {
		return nil, err
	}

	return parseJudgeResponse(response)
}

// parseJudgeResponse extracts JSON from a judge response, handling markdown wrapping.
func parseJudgeResponse(response string) (*judgeResponse, error) {
	response = strings.TrimSpace(response)
	if idx := strings.Index(response, "{"); idx >= 0 {
		if end := strings.LastIndex(response, "}"); end >= idx {
			response = response[idx : end+1]
		}
	}

	var result judgeResponse
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("parse judge response: %w (raw: %s)", err, response)
	}

	return &result, nil
}
