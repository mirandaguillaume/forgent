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
			header += fmt.Sprintf(" | %-8s recall  issues", c)
		}
		fmt.Println(header)
		fmt.Println(strings.Repeat("-", len(header)+len(conditions)*24))

		totals := map[string][]float64{}

		for _, taskID := range taskIDs {
			meta, hasMeta := golden[taskID]
			row := fmt.Sprintf("%-36s", taskID)
			for _, cond := range conditions {
				r, ok := byKey[evalKey{cond, taskID}]
				if !ok {
					row += fmt.Sprintf(" | %-8s  N/A     N/A", cond)
					continue
				}
				var recall float64
				var issuesNote string
				reviewOutput := finalStepOutput(r)
				if hasMeta && len(meta.GoldenComments) > 0 && reviewOutput != "" {
					found := llmJudgeRecall(judge, meta.GoldenComments, reviewOutput)
					recall = float64(found) / float64(len(meta.GoldenComments))
					issuesNote = fmt.Sprintf("%d/%d", found, len(meta.GoldenComments))
					totals[cond] = append(totals[cond], recall)
					row += fmt.Sprintf(" | %-8s  %3.0f%%  %s", cond, recall*100, issuesNote)
				} else {
					issuesNote = "N/A"
					row += fmt.Sprintf(" | %-8s  N/A     %s", cond, issuesNote)
				}
			}
			fmt.Println(row)
		}

		fmt.Println(strings.Repeat("-", len(header)+len(conditions)*24))
		summary := fmt.Sprintf("%-36s", "MEAN RECALL")
		for _, cond := range conditions {
			vals := totals[cond]
			if len(vals) == 0 {
				summary += fmt.Sprintf(" | %-8s  N/A", cond)
				continue
			}
			sum := 0.0
			for _, v := range vals {
				sum += v
			}
			summary += fmt.Sprintf(" | %-8s  %3.0f%%  (%d tasks)", cond, sum/float64(len(vals))*100, len(vals))
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
		model = "claude-haiku-4-5-20251001"
	}
	return llm.GetProviderWithModel(providerName, apiKey, model)
}

// llmJudgeRecall counts how many golden comments are addressed in the output.
func llmJudgeRecall(judge llm.Provider, comments []GoldenComment, output string) int {
	found := 0
	for _, gc := range comments {
		if llmJudgeHit(judge, gc, output) {
			found++
		}
	}
	return found
}

// llmJudgeHit uses an LLM to determine if the output addresses the golden comment.
// Inspired by CR-Bench's CR-Evaluator — semantic matching regardless of wording.
func llmJudgeHit(judge llm.Provider, comment GoldenComment, output string) bool {
	const maxOutput = 4000
	excerpt := output
	if len(excerpt) > maxOutput {
		excerpt = excerpt[:maxOutput]
	}

	prompt := fmt.Sprintf(`You are a code review evaluator.

GOLDEN ISSUE (severity: %s):
%s

REVIEW OUTPUT:
%s

Does the review output identify or address the issue described in the GOLDEN ISSUE?
A hit means the review mentions the same underlying bug or problem, even using different words.
Respond with JSON only: {"hit": true} or {"hit": false}`,
		comment.Severity, comment.Comment, excerpt)

	if sp, ok := judge.(llm.StructuredProvider); ok {
		schema := map[string]any{
			"type": "object",
			"properties": map[string]any{
				"hit": map[string]any{"type": "boolean"},
			},
			"required": []string{"hit"},
		}
		result, err := sp.CompleteStructured(context.Background(), prompt, schema)
		if err == nil {
			if hit, ok := result["hit"].(bool); ok {
				return hit
			}
		}
	}

	resp, err := judge.Complete(prompt)
	if err != nil {
		return false
	}
	resp = strings.ToLower(resp)
	return strings.Contains(resp, `"hit":true`) || strings.Contains(resp, `"hit": true`)
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
