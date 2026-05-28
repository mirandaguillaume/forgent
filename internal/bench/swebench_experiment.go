package bench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mirandaguillaume/forgent/internal/bench/experiment"
	"github.com/mirandaguillaume/forgent/internal/llm"
)

// SWEBenchExperimentConfig configures a SWE-bench experiment run.
type SWEBenchExperimentConfig struct {
	// Tasks to evaluate (loaded from JSONL via loadSWETasks).
	Tasks []SWEBenchTask

	// Template is the base TaskDefinition (e.g. resolve-issue.yaml).
	// Each task gets a copy with Input set to the problem statement.
	Template experiment.TaskDefinition

	// Provider is the LLM provider for text and runtime conditions.
	Provider llm.Provider

	// Conditions to run: "text" or "runtime". Default (empty): both.
	Conditions []string

	// CheckpointPath enables resume. Completed (taskID, condition) pairs are
	// skipped; new results are appended.
	CheckpointPath string

	// HarnessConfig is non-nil when the official SWE-bench harness should run
	// after patch generation. Nil = format check only.
	HarnessConfig *SWEHarnessConfig

	// OnProgress is called after each (task, condition) pair completes.
	OnProgress func(task, condition string, result SWETaskResult)

	// StepModelOverride overrides the LLM per step model ID.
	StepModelOverride func(model string) (llm.Provider, error)
}

// SWEBenchExperimentResult holds aggregate results per condition.
type SWEBenchExperimentResult struct {
	// Results maps condition → per-task results (in cfg.Tasks order).
	Results map[string][]SWETaskResult
	// Tasks is the total number of tasks evaluated.
	Tasks int
	// Warnings collects non-fatal issues encountered during the run.
	Warnings []string
}

// PatchRate returns the fraction of tasks that produced any patch text.
func (r *SWEBenchExperimentResult) PatchRate(condition string) float64 {
	results, ok := r.Results[condition]
	if !ok || r.Tasks == 0 {
		return 0
	}
	var n int
	for _, tr := range results {
		if tr.Patch != "" {
			n++
		}
	}
	return float64(n) / float64(r.Tasks) * 100
}

// ApplyRate returns the fraction of tasks where the patch passed the format
// check (patch != ""). A real apply check requires the target repo.
func (r *SWEBenchExperimentResult) ApplyRate(condition string) float64 {
	results, ok := r.Results[condition]
	if !ok || r.Tasks == 0 {
		return 0
	}
	var n int
	for _, tr := range results {
		if tr.Applied {
			n++
		}
	}
	return float64(n) / float64(r.Tasks) * 100
}

// ResolveRate returns the fraction of tasks resolved via official harness.
func (r *SWEBenchExperimentResult) ResolveRate(condition string) float64 {
	results, ok := r.Results[condition]
	if !ok || r.Tasks == 0 {
		return 0
	}
	var n int
	for _, tr := range results {
		if tr.TestPassed {
			n++
		}
	}
	return float64(n) / float64(r.Tasks) * 100
}

// RunSWEBenchExperiment runs the SWE-bench experiment for all configured conditions.
//
// For each task × condition: builds a TaskDefinition from the template,
// runs the appropriate experiment runner, and extracts the unified diff
// from TaskResult.CombinedOutput.
//
// When HarnessConfig is non-nil, the official SWE-bench harness evaluates
// each condition's predictions after generation completes.
func RunSWEBenchExperiment(cfg SWEBenchExperimentConfig) (*SWEBenchExperimentResult, error) {
	conditions := cfg.Conditions
	if len(conditions) == 0 {
		conditions = []string{"text", "runtime"}
	}
	for _, c := range conditions {
		if c != "text" && c != "runtime" {
			return nil, fmt.Errorf("unsupported condition %q (valid: text, runtime)", c)
		}
	}

	// Load checkpoint for resume.
	var existingEntries []experiment.CheckpointEntry
	if cfg.CheckpointPath != "" {
		if entries, err := experiment.LoadCheckpoint(cfg.CheckpointPath); err == nil {
			existingEntries = entries
		}
	}

	// Open checkpoint writer (append mode).
	var cpWriter *experiment.CheckpointWriter
	if cfg.CheckpointPath != "" {
		var err error
		cpWriter, err = experiment.NewCheckpointWriter(cfg.CheckpointPath)
		if err != nil {
			return nil, fmt.Errorf("open checkpoint: %w", err)
		}
		defer cpWriter.Close()
	}

	// Build skip maps per condition: instanceID → runs already completed.
	skipMaps := make(map[string]map[string]int, len(conditions))
	for _, cond := range conditions {
		skipMaps[cond] = experiment.BuildSkipMap(existingEntries, cond)
	}

	// Restore prior results from checkpoint.
	priorResults := make(map[string]map[string]SWETaskResult, len(conditions))
	for _, cond := range conditions {
		priorResults[cond] = make(map[string]SWETaskResult)
	}
	for _, entry := range existingEntries {
		if _, ok := priorResults[entry.Condition]; !ok {
			continue // condition from old checkpoint not requested in this run
		}
		tr := entry.Result.ToTaskResult()
		patch := extractUnifiedDiffFromOutput(tr.CombinedOutput)
		priorResults[entry.Condition][entry.TaskID] = SWETaskResult{
			InstanceID: entry.TaskID,
			Patch:      patch,
			Applied:    patch != "",
		}
	}

	out := &SWEBenchExperimentResult{
		Results: make(map[string][]SWETaskResult, len(conditions)),
		Tasks:   len(cfg.Tasks),
	}
	for _, cond := range conditions {
		out.Results[cond] = make([]SWETaskResult, 0, len(cfg.Tasks))
	}

	var mu sync.Mutex

	for _, sweTask := range cfg.Tasks {
		taskDef := buildSWETaskDef(cfg.Template, sweTask)

		for _, cond := range conditions {
			instanceID := sweTask.InstanceID

			// Skip tasks already in checkpoint.
			if skipMaps[cond][instanceID] > 0 {
				mu.Lock()
				if prior, ok := priorResults[cond][instanceID]; ok {
					out.Results[cond] = append(out.Results[cond], prior)
				}
				mu.Unlock()
				continue
			}

			expCfg := experiment.ExperimentConfig{
				Tasks:             []experiment.TaskDefinition{taskDef},
				Provider:          cfg.Provider,
				RunsPerTask:       1,
				Condition:         cond,
				StepModelOverride: cfg.StepModelOverride,
				OnResult: func(r experiment.TaskResult, run int) {
					if cpWriter == nil {
						return
					}
					mu.Lock()
					defer mu.Unlock()
					_ = cpWriter.Append(experiment.CheckpointEntry{
						TaskID:    r.TaskID,
						Condition: cond,
						Run:       run,
						Result:    experiment.ToResultJSON(r),
					})
				},
			}

			var taskResults []experiment.TaskResult
			var runErr error
			switch cond {
			case "text":
				taskResults, runErr = experiment.RunTextExperiment(expCfg)
			case "runtime":
				taskResults, runErr = experiment.RunRuntimeExperiment(expCfg)
			}

			sweResult := SWETaskResult{InstanceID: instanceID}
			if runErr != nil {
				sweResult.Error = runErr.Error()
			} else if len(taskResults) > 0 {
				patch := extractUnifiedDiffFromOutput(taskResults[0].CombinedOutput)
				sweResult.Patch = patch
				// Applied is a format-only check: patch != "" means the LLM produced
				// a unified diff header. A real apply check requires the target repo.
				sweResult.Applied = patch != ""
			}

			mu.Lock()
			out.Results[cond] = append(out.Results[cond], sweResult)
			mu.Unlock()

			if cfg.OnProgress != nil {
				cfg.OnProgress(instanceID, cond, sweResult)
			}
		}
	}

	// Optional: run official harness per condition.
	if cfg.HarnessConfig != nil {
		for _, cond := range conditions {
			details := out.Results[cond]
			resolved, err := sweHarnessEval(details, cond, *cfg.HarnessConfig)
			if err != nil {
				out.Warnings = append(out.Warnings,
					fmt.Sprintf("harness eval for condition %q failed: %v — resolve rates will show 0%%", cond, err))
				continue
			}
			for i, d := range details {
				if resolved[d.InstanceID] {
					out.Results[cond][i].TestPassed = true
				}
			}
		}
	}

	return out, nil
}

// WriteSWEPredictions writes SWETaskResults as a SWE-bench predictions JSONL file.
func WriteSWEPredictions(path, conditionName string, results []SWETaskResult) error {
	return WritePredictions(path, conditionName, results)
}

// LoadSWETasks reads SWE-bench tasks from a JSONL file.
// Uses json.Decoder to avoid bufio.Scanner's 64KB line limit.
func LoadSWETasks(path string) ([]SWEBenchTask, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var tasks []SWEBenchTask
	dec := json.NewDecoder(f)
	for dec.More() {
		var task SWEBenchTask
		if err := dec.Decode(&task); err != nil {
			return nil, fmt.Errorf("decode task: %w", err)
		}
		if task.InstanceID != "" {
			tasks = append(tasks, task)
		}
	}
	return tasks, nil
}

// sweHarnessEval runs the official SWE-bench harness for one condition.
func sweHarnessEval(details []SWETaskResult, conditionName string, cfg SWEHarnessConfig) (map[string]bool, error) {
	tmpDir, err := os.MkdirTemp("", "swe-harness-"+conditionName+"-*")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(tmpDir)

	predPath := filepath.Join(tmpDir, "predictions.jsonl")
	if err := WritePredictions(predPath, conditionName, details); err != nil {
		return nil, fmt.Errorf("write predictions: %w", err)
	}

	return RunHarnessEvaluation(cfg, predPath, "exp-"+conditionName)
}

// buildSWETaskDef creates a TaskDefinition from the template, injecting the
// SWE-bench task's problem statement into the Input field.
func buildSWETaskDef(tmpl experiment.TaskDefinition, sweTask SWEBenchTask) experiment.TaskDefinition {
	td := tmpl
	td.ID = sweTask.InstanceID
	td.Input = fmt.Sprintf("# SWE-bench Task: %s\n\n## Repository\n\n%s\n\n## Problem Statement\n\n%s%s",
		sweTask.InstanceID,
		sweTask.Repo,
		sweTask.ProblemStatement,
		buildSWEHints(sweTask.HintsText),
	)
	return td
}

func buildSWEHints(hints string) string {
	if hints == "" {
		return ""
	}
	return "\n\n## Hints\n\n" + hints
}

// buildSWECodeSummary returns a minimal code context string for a SWE-bench task.
// Gold patch file list is intentionally NOT included to avoid benchmark contamination:
// the agent must discover affected files from the problem statement alone.
func buildSWECodeSummary(task SWEBenchTask) string {
	var sb strings.Builder
	sb.WriteString("Repository: " + task.Repo + "\n")
	if task.HintsText != "" {
		sb.WriteString("\nHints:\n" + task.HintsText + "\n")
	}
	return sb.String()
}

// extractUnifiedDiffFromOutput extracts the first unified diff block from an
// LLM output string using hunk-aware parsing.
//
// The parser recognises three diff-header prefixes ("--- a/", "+++ b/", "@@ ")
// and hunk content lines ("+", "-", " " prefixes). It only accepts content lines
// after a "@@" header has been seen, so LLM commentary that starts with "+"/"-"
// before the first hunk is excluded. The extraction stops at the first line that
// cannot be classified as part of the diff.
func extractUnifiedDiffFromOutput(output string) string {
	lines := strings.Split(output, "\n")

	// Find the first "--- a/" header.
	startIdx := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "--- a/") || strings.HasPrefix(line, "--- /dev/null") {
			startIdx = i
			break
		}
	}
	if startIdx < 0 {
		return ""
	}

	var out []string
	inHunk := false
	lastGoodIdx := -1 // last index in out[] that is a real diff line (non-blank)

LINES:
	for i := startIdx; i < len(lines); i++ {
		line := lines[i]
		switch {
		case strings.HasPrefix(line, "--- a/"), strings.HasPrefix(line, "--- /dev/null"):
			out = append(out, line)
			lastGoodIdx = len(out) - 1
			inHunk = false

		case strings.HasPrefix(line, "+++ b/"), strings.HasPrefix(line, "+++ /dev/null"):
			out = append(out, line)
			lastGoodIdx = len(out) - 1

		case strings.HasPrefix(line, "@@ "):
			out = append(out, line)
			lastGoodIdx = len(out) - 1
			inHunk = true

		case inHunk && len(line) > 0 && (line[0] == ' ' || line[0] == '+' || line[0] == '-'):
			out = append(out, line)
			lastGoodIdx = len(out) - 1

		case inHunk && line == "":
			// Blank lines between files in a multi-file diff are valid.
			// Don't advance lastGoodIdx so trailing blanks are trimmed.
			out = append(out, line)

		default:
			// Non-diff content — stop extraction.
			break LINES
		}
	}

	if lastGoodIdx < 0 {
		return ""
	}
	return strings.TrimRight(strings.Join(out[:lastGoodIdx+1], "\n"), " \t")
}
