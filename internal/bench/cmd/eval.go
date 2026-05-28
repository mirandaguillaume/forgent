package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mirandaguillaume/forgent/internal/bench/experiment"
	"github.com/mirandaguillaume/forgent/internal/llm"
)

func init() {
	evalCmd.Flags().StringSlice("snapshots", nil, "Snapshot JSON files to evaluate (e.g. run.runtime.json,run.agent.json)")
	evalCmd.Flags().String("corpus", "", "Corpus directory containing entry subdirs with meta.json")
	evalCmd.Flags().String("provider", "", "LLM judge provider (anthropic|openrouter; auto-detected from env if unset)")
	evalCmd.Flags().String("model", "", "LLM judge model (default: claude-haiku-4-5-20251001)")
	rootCmd.AddCommand(evalCmd)
}

type evalKey struct{ condition, taskID string }

var evalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Evaluate snapshot outputs against corpus golden comments using LLM-as-judge",
	Long: `eval reads one or more snapshot JSON files produced by 'bench' and
evaluates the combined LLM output for each task against the golden comments
stored in the corpus meta.json files using an LLM judge.

Example:
  forgent-bench eval \
    --snapshots run.runtime.json,run.agent.json \
    --corpus internal/bench/experiment/corpus/martian`,
	RunE: func(cmd *cobra.Command, args []string) error {
		snapshotPaths, _ := cmd.Flags().GetStringSlice("snapshots")
		corpusDir, _ := cmd.Flags().GetString("corpus")

		if len(snapshotPaths) == 0 {
			return fmt.Errorf("--snapshots required")
		}
		if corpusDir == "" {
			return fmt.Errorf("--corpus required")
		}

		judge, err := resolveJudge(cmd)
		if err != nil {
			return fmt.Errorf("resolve judge: %w", err)
		}

		golden, err := loadCorpusMeta(corpusDir)
		if err != nil {
			return fmt.Errorf("load corpus meta: %w", err)
		}

		byKey := map[evalKey]experiment.TaskResultJSON{}
		for _, path := range snapshotPaths {
			results, err := loadSnapshot(path)
			if err != nil {
				return fmt.Errorf("load snapshot %s: %w", path, err)
			}
			for _, r := range results {
				byKey[evalKey{r.Condition, r.TaskID}] = r
			}
		}

		conditions := uniqueConditions(byKey)
		taskIDs := uniqueTaskIDs(byKey)

		header := fmt.Sprintf("%-36s", "Task")
		for _, c := range conditions {
			header += fmt.Sprintf(" | %-8s    R     P    F1", c)
		}
		fmt.Println(header)
		fmt.Println(strings.Repeat("-", len(header)+len(conditions)*26))

		type condTotals struct{ recall, prec, f1 []float64 }
		totals := map[string]*condTotals{}

		for _, taskID := range taskIDs {
			meta, hasMeta := golden[taskID]
			row := fmt.Sprintf("%-36s", taskID)
			for _, cond := range conditions {
				r, ok := byKey[evalKey{cond, taskID}]
				if !ok {
					row += fmt.Sprintf(" | %-8s  N/A   N/A  N/A", cond)
					continue
				}
				if !hasMeta || len(meta.GoldenComments) == 0 {
					row += fmt.Sprintf(" | %-8s  N/A   N/A  N/A", cond)
					continue
				}

				// CR-Bench cross-product eval when individual candidates are available (runtime).
				// Text/agent fall back to golden-vs-full-output (no candidate list to enumerate).
				detected := extractDetectedIssues(r)
				var recall, prec, f1 float64
				precStr := "N/A"
				f1Str := "N/A"

				if len(detected) > 0 {
					// Runtime: exact CR-Bench protocol — cross-product (golden × candidate).
					recallHits, precHits := crossProductMatch(judge, meta.GoldenComments, detected)
					recall = float64(recallHits) / float64(len(meta.GoldenComments))
					prec = float64(precHits) / float64(len(detected))
					precStr = fmt.Sprintf("%3.0f%%", prec*100)
					if recall+prec > 0 {
						f1 = 2 * recall * prec / (recall + prec)
						f1Str = fmt.Sprintf("%3.0f%%", f1*100)
					}
					if totals[cond] == nil {
						totals[cond] = &condTotals{}
					}
					totals[cond].recall = append(totals[cond].recall, recall)
					totals[cond].prec = append(totals[cond].prec, prec)
					totals[cond].f1 = append(totals[cond].f1, f1)
				} else {
					// Text/agent: no candidate list — compare each golden against full output.
					reviewOutput := finalStepOutput(r)
					if reviewOutput == "" {
						row += fmt.Sprintf(" | %-8s  N/A   N/A  N/A", cond)
						continue
					}
					recallHits := llmJudgeRecallText(judge, meta.GoldenComments, reviewOutput)
					recall = float64(recallHits) / float64(len(meta.GoldenComments))
					if totals[cond] == nil {
						totals[cond] = &condTotals{}
					}
					totals[cond].recall = append(totals[cond].recall, recall)
				}

				row += fmt.Sprintf(" | %-8s  %3.0f%%  %s  %s", cond, recall*100, precStr, f1Str)
			}
			fmt.Println(row)
		}

		fmt.Println(strings.Repeat("-", len(header)+len(conditions)*26))
		summary := fmt.Sprintf("%-36s", "MEAN")
		for _, cond := range conditions {
			t := totals[cond]
			if t == nil || len(t.recall) == 0 {
				summary += fmt.Sprintf(" | %-8s  N/A   N/A  N/A", cond)
				continue
			}
			meanOf := func(vs []float64) float64 {
				s := 0.0
				for _, v := range vs {
					s += v
				}
				return s / float64(len(vs))
			}
			r := meanOf(t.recall)
			pStr, f1Str := "N/A", "N/A"
			if len(t.prec) > 0 {
				p := meanOf(t.prec)
				pStr = fmt.Sprintf("%3.0f%%", p*100)
				f1Str = fmt.Sprintf("%3.0f%%", meanOf(t.f1)*100)
			}
			summary += fmt.Sprintf(" | %-8s  %3.0f%%  %s  %s  (%d tasks)", cond, r*100, pStr, f1Str, len(t.recall))
		}
		fmt.Println(summary)

		return nil
	},
}

// GoldenEntry is the meta.json content for a corpus entry.
type GoldenEntry struct {
	SourceURL      string          `json:"source_url"`
	Project        string          `json:"project"`
	Title          string          `json:"title"`
	GoldenComments []GoldenComment `json:"golden_comments"`
}

// GoldenComment is one human-verified issue from the Martian corpus.
type GoldenComment struct {
	Comment  string `json:"comment"`
	Severity string `json:"severity"`
}

// resolveJudge builds the LLM provider used for evaluation.
// Defaults to Haiku — cheap, fast, sufficient for binary hit/miss judgement.
func resolveJudge(cmd *cobra.Command) (llm.Provider, error) {
	providerName, _ := cmd.Flags().GetString("provider")
	if providerName == "" {
		if resolveEnvKey("ANTHROPIC_API_KEY", "ANTHROPIC_FORGENT_API_KEY") != "" {
			providerName = "anthropic"
		} else if resolveEnvKey("OPENROUTER_API_KEY", "OPENROUTER_FORGENT_API_KEY") != "" {
			providerName = "openrouter"
		} else {
			return nil, fmt.Errorf("no LLM API key found — set ANTHROPIC_API_KEY or OPENROUTER_API_KEY")
		}
	}

	var apiKey string
	switch providerName {
	case "anthropic":
		apiKey = resolveEnvKey("ANTHROPIC_API_KEY", "ANTHROPIC_FORGENT_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY required for provider %q", providerName)
		}
	case "openrouter":
		apiKey = resolveEnvKey("OPENROUTER_API_KEY", "OPENROUTER_FORGENT_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("OPENROUTER_API_KEY required for provider %q", providerName)
		}
	default:
		return nil, fmt.Errorf("unknown provider %q (use 'anthropic' or 'openrouter')", providerName)
	}

	model, _ := cmd.Flags().GetString("model")
	if model == "" {
		model = "claude-sonnet-4-6"
	}
	return llm.GetProviderWithModel(providerName, apiKey, model)
}

// extractDetectedIssues returns the list of issues from the collect-filtered step (runtime).
// Returns nil when no structured issue list is available (text/agent conditions).
func extractDetectedIssues(r experiment.TaskResultJSON) []string {
	for _, step := range r.StepResults {
		if step.StepID != "collect-filtered" {
			continue
		}
		raw := strings.TrimSpace(step.Output)
		if raw == "" {
			return nil
		}
		var items []json.RawMessage
		if err := json.Unmarshal([]byte(raw), &items); err != nil {
			return nil
		}
		var issues []string
		for _, item := range items {
			// item is either a plain string or {"issue":"...","score":N}
			var s string
			if err := json.Unmarshal(item, &s); err == nil {
				issues = append(issues, s)
				continue
			}
			var obj map[string]any
			if err := json.Unmarshal(item, &obj); err == nil {
				if iss, ok := obj["issue"].(string); ok {
					issues = append(issues, iss)
				}
			}
		}
		return issues
	}
	return nil
}

// judgeOnePair is the exact CR-Bench judge prompt: one golden vs one candidate.
// Returns true when the judge considers them the same issue with confidence >= 0.5.
func judgeOnePair(judge llm.Provider, golden GoldenComment, candidate string) bool {
	prompt := fmt.Sprintf(`You are evaluating AI code review tools.
Determine if the candidate issue matches the golden (expected) comment.

Golden Comment (the issue we're looking for):
%s

Candidate Issue (from the tool's review):
%s

Instructions:
- Determine if the candidate identifies the SAME underlying issue as the golden comment
- Accept semantic matches - different wording is fine if it's the same problem
- Focus on whether they point to the same bug, concern, or code issue

Respond with ONLY a JSON object:
{"reasoning": "brief explanation", "match": true/false, "confidence": 0.0-1.0}`,
		golden.Comment, candidate)

	if sp, ok := judge.(llm.StructuredProvider); ok {
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"reasoning":  map[string]any{"type": "string"},
				"match":      map[string]any{"type": "boolean"},
				"confidence": map[string]any{"type": "number"},
			},
			"required":             []string{"reasoning", "match", "confidence"},
			"additionalProperties": false,
		}
		result, err := sp.CompleteStructured(context.Background(), prompt, schema)
		if err == nil {
			match, _ := result["match"].(bool)
			confidence, _ := result["confidence"].(float64)
			return match && confidence >= 0.5
		}
	}

	resp, err := judge.Complete(prompt)
	if err != nil {
		return false
	}
	resp = strings.ToLower(resp)
	return strings.Contains(resp, `"match":true`) || strings.Contains(resp, `"match": true`)
}

// crossProductMatch runs the CR-Bench cross-product evaluation:
// each golden comment vs each candidate issue (one LLM call per pair).
// Returns (recallHits, precisionHits):
//   - recallHits: number of golden comments matched by at least one candidate
//   - precisionHits: number of candidates that matched at least one golden
func crossProductMatch(judge llm.Provider, goldens []GoldenComment, candidates []string) (recallHits, precisionHits int) {
	goldenMatched := make([]bool, len(goldens))
	candidateMatched := make([]bool, len(candidates))

	for gi, golden := range goldens {
		for ci, candidate := range candidates {
			if judgeOnePair(judge, golden, candidate) {
				goldenMatched[gi] = true
				candidateMatched[ci] = true
			}
		}
	}
	for _, m := range goldenMatched {
		if m {
			recallHits++
		}
	}
	for _, m := range candidateMatched {
		if m {
			precisionHits++
		}
	}
	return
}

// llmJudgeRecallText counts golden comments found in a free-form text output.
// Used for text/agent conditions where individual candidates cannot be extracted.
func llmJudgeRecallText(judge llm.Provider, comments []GoldenComment, output string) int {
	const maxOutput = 4000
	excerpt := output
	if idx := strings.LastIndex(excerpt, "### Code review"); idx != -1 {
		excerpt = excerpt[idx:]
	} else if idx := strings.LastIndex(excerpt, "REVIEW_DECISION"); idx != -1 {
		start := idx - maxOutput
		if start < 0 {
			start = 0
		}
		excerpt = excerpt[start:]
	}
	if len(excerpt) > maxOutput {
		excerpt = excerpt[len(excerpt)-maxOutput:]
	}

	found := 0
	for _, gc := range comments {
		if judgeOnePair(judge, gc, excerpt) {
			found++
		}
	}
	return found
}

// finalStepOutput returns the most relevant output for judge evaluation.
// For multi-step pipelines the last completed step is the final review comment.
// Falls back to CombinedOutput when no step results are available (text mode).
func finalStepOutput(r experiment.TaskResultJSON) string {
	for i := len(r.StepResults) - 1; i >= 0; i-- {
		if r.StepResults[i].Completed && r.StepResults[i].Output != "" {
			return r.StepResults[i].Output
		}
	}
	return r.CombinedOutput
}

func loadCorpusMeta(corpusDir string) (map[string]GoldenEntry, error) {
	entries, err := os.ReadDir(corpusDir)
	if err != nil {
		return nil, err
	}
	result := map[string]GoldenEntry{}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		metaPath := filepath.Join(corpusDir, e.Name(), "meta.json")
		data, err := os.ReadFile(metaPath)
		if err != nil {
			continue
		}
		var meta GoldenEntry
		if err := json.Unmarshal(data, &meta); err != nil {
			continue
		}
		result[e.Name()] = meta
	}
	return result, nil
}

func loadSnapshot(path string) ([]experiment.TaskResultJSON, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var results []experiment.TaskResultJSON
	if err := json.Unmarshal(data, &results); err != nil {
		return nil, err
	}
	return results, nil
}

func uniqueConditions(byKey map[evalKey]experiment.TaskResultJSON) []string {
	seen := map[string]bool{}
	for k := range byKey {
		seen[k.condition] = true
	}
	var out []string
	for c := range seen {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

func uniqueTaskIDs(byKey map[evalKey]experiment.TaskResultJSON) []string {
	seen := map[string]bool{}
	for k := range byKey {
		seen[k.taskID] = true
	}
	var out []string
	for id := range seen {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
