package experiment

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"unicode"
)

// ContextRetriever selects and formats context to inject into a step prompt.
// Called once per LLM step before the prompt is finalized.
// Returns a markdown string injected under "## Context (pre-loaded excerpt)",
// or empty string for no injection.
type ContextRetriever interface {
	Retrieve(step StepDef, sandbox *SandboxFS, consumed []SlotSpec) (string, error)
}

// NilRetriever returns empty string always — file listing only, no pre-injection.
// This is the default (backwards-compatible) behavior from story 6-21.
type NilRetriever struct{}

func (NilRetriever) Retrieve(_ StepDef, _ *SandboxFS, _ []SlotSpec) (string, error) {
	return "", nil
}

// PrefixRetriever injects the first MaxTokens tokens of each consumed slot file.
// Silently skips slots whose sandbox files are not yet written (no error logged).
type PrefixRetriever struct {
	MaxTokens int // approximation: 1 token ≈ 4 chars (default 300)
}

func (r PrefixRetriever) Retrieve(step StepDef, sandbox *SandboxFS, consumed []SlotSpec) (string, error) {
	maxTokens := r.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 300
	}
	maxChars := maxTokens * 4

	var sb strings.Builder
	for _, slot := range consumed {
		if slot.Key == "task_input" {
			continue
		}
		data, err := sandbox.Read(step.ID, fmt.Sprintf("slots/%s.md", slot.Key))
		if err != nil {
			// Typed slots are written as .json — fall back transparently.
			data, err = sandbox.Read(step.ID, fmt.Sprintf("slots/%s.json", slot.Key))
			if err != nil {
				continue
			}
		}
		content := string(data)
		runes := []rune(content)
		if len(runes) > maxChars {
			content = string(runes[:maxChars])
		}
		fmt.Fprintf(&sb, "### %s\n\n%s\n\n", slot.Key, content)
	}
	return sb.String(), nil
}

// RankedRetriever uses BM25 scoring to select the most relevant chunks from consumed
// slot files. The step prompt is the query; each paragraph-sized chunk is a document.
// Returns top-K chunks within the MaxTokens budget.
type RankedRetriever struct {
	MaxTokens int // total char budget across all chunks (default 500 tokens)
	TopK      int // max chunks to include (default 5)
}

func (r RankedRetriever) Retrieve(step StepDef, sandbox *SandboxFS, consumed []SlotSpec) (string, error) {
	maxTokens := r.MaxTokens
	if maxTokens <= 0 {
		maxTokens = 500
	}
	topK := r.TopK
	if topK <= 0 {
		topK = 5
	}

	type chunk struct {
		slotKey string
		text    string
	}

	var chunks []chunk
	for _, slot := range consumed {
		if slot.Key == "task_input" {
			continue
		}
		data, err := sandbox.Read(step.ID, fmt.Sprintf("slots/%s.md", slot.Key))
		if err != nil {
			data, err = sandbox.Read(step.ID, fmt.Sprintf("slots/%s.json", slot.Key))
			if err != nil {
				continue
			}
		}
		for _, c := range splitChunks(string(data)) {
			chunks = append(chunks, chunk{slotKey: slot.Key, text: c})
		}
	}

	if len(chunks) == 0 {
		return "", nil
	}

	query := bm25Tokenize(step.Prompt)
	if len(query) == 0 {
		return "", nil
	}

	tokenizedChunks := make([][]string, len(chunks))
	for i, c := range chunks {
		tokenizedChunks[i] = bm25Tokenize(c.text)
	}

	scores := bm25Score(query, tokenizedChunks)

	type scored struct {
		idx   int
		score float64
	}
	ranked := make([]scored, len(chunks))
	for i, s := range scores {
		ranked[i] = scored{i, s}
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})

	var sb strings.Builder
	budgetChars := maxTokens * 4
	used := 0
	count := 0
	for _, rs := range ranked {
		if count >= topK || used >= budgetChars {
			break
		}
		c := chunks[rs.idx]
		if used+len(c.text) > budgetChars {
			break
		}
		fmt.Fprintf(&sb, "### %s (excerpt)\n\n%s\n\n", c.slotKey, c.text)
		used += len(c.text)
		count++
	}

	return sb.String(), nil
}

// splitChunks splits text on double-newlines, filtering short noise and capping chunk size.
func splitChunks(content string) []string {
	raw := strings.Split(content, "\n\n")
	var chunks []string
	for _, c := range raw {
		c = strings.TrimSpace(c)
		runes := []rune(c)
		if len(runes) < 20 {
			continue
		}
		for len(runes) > 500 {
			chunks = append(chunks, string(runes[:500]))
			runes = []rune(strings.TrimSpace(string(runes[500:])))
		}
		if len(runes) >= 20 {
			chunks = append(chunks, string(runes))
		}
	}
	return chunks
}

// bm25Tokenize lowercases and splits on non-alphanumeric characters.
func bm25Tokenize(text string) []string {
	return strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// bm25Score computes BM25 scores for each document against query terms.
// Robertson parameters: k1=1.5, b=0.75.
// IDF = ln((N - df(t) + 1) / (df(t) + 1) + 1)  (Robertson–Sparck Jones variant)
func bm25Score(query []string, docs [][]string) []float64 {
	const k1 = 1.5
	const b = 0.75
	N := len(docs)
	scores := make([]float64, N)
	if N == 0 {
		return scores
	}

	totalLen := 0
	for _, doc := range docs {
		totalLen += len(doc)
	}
	avgdl := float64(totalLen) / float64(N)
	if avgdl == 0 {
		return scores
	}

	// Document frequency: number of docs containing each term
	df := make(map[string]int, len(query)*2)
	for _, doc := range docs {
		seen := make(map[string]bool)
		for _, t := range doc {
			if !seen[t] {
				df[t]++
				seen[t] = true
			}
		}
	}

	for i, doc := range docs {
		tf := make(map[string]int, len(doc))
		for _, t := range doc {
			tf[t]++
		}
		dl := float64(len(doc))
		for _, qt := range query {
			freq := tf[qt]
			if freq == 0 {
				continue
			}
			idf := math.Log(float64(N-df[qt]+1)/float64(df[qt]+1) + 1)
			numerator := float64(freq) * (k1 + 1)
			denominator := float64(freq) + k1*(1-b+b*dl/avgdl)
			scores[i] += idf * numerator / denominator
		}
	}

	return scores
}
