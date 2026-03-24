package bench

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/mirandaguillaume/forgent/internal/generator"
	"github.com/mirandaguillaume/forgent/internal/llm"
)

// H2HJudgeResult holds the judge's evaluation of a single response.
type H2HJudgeResult struct {
	CriteriaHits    []bool       `json:"criteria_hits"`
	WeightedScore   float64      `json:"weighted_score"` // severity-weighted hit rate (0-100)
	Reasoning       string       `json:"reasoning"`
	Severity        SeverityHits `json:"severity"`
	JudgeInputWords  int         // words in judge prompt
	JudgeOutputWords int         // words in judge response
}

type h2hJudgeResponse struct {
	CriteriaHits []bool `json:"criteria_hits"`
	Reasoning    string `json:"reasoning"`
}

// SeverityWeight returns the numeric weight for a severity level.
func SeverityWeight(severity string) float64 {
	switch strings.ToLower(severity) {
	case "critical":
		return 4
	case "high":
		return 3
	case "medium":
		return 2
	default:
		return 1 // "Low" or unknown
	}
}

// JudgeH2HResponse evaluates a single agent response against task criteria.
// It returns severity-weighted scoring based on criteria hits.
func JudgeH2HResponse(provider llm.Provider, task H2HTask, response string) (*H2HJudgeResult, error) {
	criteriaList := ""
	for i, c := range task.Criteria {
		criteriaList += fmt.Sprintf("%d. %s\n", i+1, c)
	}

	prompt := fmt.Sprintf(`You are an expert code review evaluator. A code review agent was given this code to review:

%s

The agent produced this review:
%s

Evaluate the review against these criteria (true if addressed, false if not):
%s
Respond with ONLY valid JSON (no markdown):
{"criteria_hits": [true/false for each criterion in order], "reasoning": "<brief explanation>"}`,
		task.Code, response, criteriaList)

	judgeInputWords := generator.CountWords(prompt)

	raw, err := provider.Complete(prompt)
	if err != nil {
		return nil, fmt.Errorf("judge call: %w", err)
	}
	judgeOutputWords := generator.CountWords(raw)

	parsed, err := parseH2HJudgeResponse(raw, len(task.Criteria))
	if err != nil {
		return nil, err
	}

	// Calculate severity-weighted score and per-severity hits
	weightedHits := 0.0
	totalWeight := 0.0
	var sevHits SeverityHits
	for i, hit := range parsed.CriteriaHits {
		sev := "Low"
		if i < len(task.Severities) {
			sev = task.Severities[i]
		}
		w := SeverityWeight(sev)
		totalWeight += w

		switch strings.ToLower(sev) {
		case "critical":
			sevHits.CriticalTotal++
			if hit {
				sevHits.CriticalHits++
			}
		case "high":
			sevHits.HighTotal++
			if hit {
				sevHits.HighHits++
			}
		case "medium":
			sevHits.MediumTotal++
			if hit {
				sevHits.MediumHits++
			}
		default:
			sevHits.LowTotal++
			if hit {
				sevHits.LowHits++
			}
		}

		if hit {
			weightedHits += w
		}
	}
	score := 0.0
	if totalWeight > 0 {
		score = weightedHits / totalWeight * 100
	}

	return &H2HJudgeResult{
		CriteriaHits:    parsed.CriteriaHits,
		WeightedScore:   score,
		Reasoning:       parsed.Reasoning,
		Severity:        sevHits,
		JudgeInputWords:  judgeInputWords,
		JudgeOutputWords: judgeOutputWords,
	}, nil
}

func parseH2HJudgeResponse(response string, expectedCriteria int) (*h2hJudgeResponse, error) {
	response = strings.TrimSpace(response)
	if idx := strings.Index(response, "{"); idx >= 0 {
		if end := strings.LastIndex(response, "}"); end >= idx {
			response = response[idx : end+1]
		}
	}

	// Use flexible map parsing to handle field name variations from the LLM.
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(response), &raw); err != nil {
		return nil, fmt.Errorf("parse h2h judge response: %w (raw: %.200s)", err, response)
	}

	var result h2hJudgeResponse

	// Extract criteria_hits — try common field names
	for _, key := range []string{"criteria_hits", "criteriaHits", "criteria"} {
		if v, ok := raw[key]; ok {
			if err := json.Unmarshal(v, &result.CriteriaHits); err == nil && len(result.CriteriaHits) > 0 {
				break
			}
		}
	}

	// Extract reasoning — try common field names
	for _, key := range []string{"reasoning", "explanation", "rationale", "summary"} {
		if v, ok := raw[key]; ok {
			if err := json.Unmarshal(v, &result.Reasoning); err == nil && result.Reasoning != "" {
				break
			}
		}
	}

	// Pad or truncate criteria_hits to match expected count
	for len(result.CriteriaHits) < expectedCriteria {
		result.CriteriaHits = append(result.CriteriaHits, false)
	}
	if len(result.CriteriaHits) > expectedCriteria {
		result.CriteriaHits = result.CriteriaHits[:expectedCriteria]
	}

	return &result, nil
}
