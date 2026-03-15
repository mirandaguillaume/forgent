package bench

import (
	"strings"

	"github.com/mirandaguillaume/forgent/internal/llm"
)

// ConsistencyResult holds the outcome of a pass@k consistency benchmark.
type ConsistencyResult struct {
	Task            string
	Runs            int
	UniqueResponses int
	ConsistencyRate float64 // 1.0 = all identical, 0.0 = all different
	AvgResponseLen  int
}

// RunConsistency sends the same prompt+task k times and measures response consistency.
func RunConsistency(task EvalTask, prompt string, provider llm.Provider, k int) (*ConsistencyResult, error) {
	if k < 2 {
		k = 2
	}

	fullPrompt := prompt + "\n\nReview the following code:\n```\n" + task.Input + "\n```"

	var responses []string
	totalLen := 0
	for i := 0; i < k; i++ {
		resp, err := provider.Complete(fullPrompt)
		if err != nil {
			return nil, err
		}
		responses = append(responses, resp)
		totalLen += len(resp)
	}

	// Count unique responses after normalization
	unique := countUnique(responses)

	rate := 1.0
	if k > 1 {
		rate = 1.0 - float64(unique-1)/float64(k-1)
	}

	return &ConsistencyResult{
		Task:            task.ID,
		Runs:            k,
		UniqueResponses: unique,
		ConsistencyRate: rate,
		AvgResponseLen:  totalLen / k,
	}, nil
}

// countUnique normalizes responses and counts distinct ones.
func countUnique(responses []string) int {
	seen := make(map[string]bool)
	for _, r := range responses {
		normalized := normalizeResponse(r)
		seen[normalized] = true
	}
	return len(seen)
}

// normalizeResponse strips whitespace and lowercases for comparison.
func normalizeResponse(s string) string {
	s = strings.ToLower(s)
	s = strings.Join(strings.Fields(s), " ")
	return s
}
