package bench

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// ClaudeRunConfig holds configuration for Claude Code benchmark runs.
type ClaudeRunConfig struct {
	ClaudeBin    string  // path to claude CLI
	Model        string  // e.g. "sonnet", "opus", "haiku"
	MaxBudgetUSD float64 // cost limit per task
	Workers      int     // parallel instances
}

// DefaultClaudeRunConfig returns sensible defaults.
func DefaultClaudeRunConfig() ClaudeRunConfig {
	claudeBin, _ := exec.LookPath("claude")
	if claudeBin == "" {
		claudeBin = "claude"
	}
	return ClaudeRunConfig{
		ClaudeBin:    claudeBin,
		Model:        "sonnet",
		MaxBudgetUSD: 2.0,
		Workers:      2,
	}
}

// ClaudeRunResult holds the result of a single Claude Code task execution.
type ClaudeRunResult struct {
	InstanceID  string
	Patch       string  // git diff output
	Applied     bool    // patch is non-empty (agent made changes)
	Error       string
	FileOverlap float64 // fraction of gold files touched (0.0-1.0)
	ExactMatch  bool    // generated patch matches gold patch exactly
	FilesMatch  bool    // same set of files modified as gold
}

// ClaudeRunContestantResult holds aggregate results for one prompt variant.
type ClaudeRunContestantResult struct {
	Name        string
	Words       int
	Tasks       int
	Patched     int     // produced a non-empty diff
	PatchRate   float64 // % tasks with changes
	FilesMatch  int     // tasks where exact same files were modified
	MatchRate   float64 // % tasks with matching files
	AvgOverlap  float64 // average file overlap with gold patch
	ExactMatch  int     // tasks where patch matches gold exactly
	Details     []ClaudeRunResult
}

// RunClaudeBenchTask runs a single SWE-bench task using Claude Code.
// It clones the repo at the exact base commit, runs claude -p with the given
// system prompt and task, then captures git diff as the patch.
func RunClaudeBenchTask(task SWEBenchTask, systemPrompt string, cfg ClaudeRunConfig) ClaudeRunResult {
	cr := ClaudeRunResult{InstanceID: task.InstanceID}

	// Create temp dir and shallow-clone repo at the exact base commit
	tmpDir, err := os.MkdirTemp("", "claude-bench-*")
	if err != nil {
		cr.Error = fmt.Sprintf("create tmpdir: %v", err)
		return cr
	}
	defer os.RemoveAll(tmpDir)

	gitEnv := append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	repoURL := fmt.Sprintf("https://github.com/%s.git", task.Repo)

	// git init + fetch specific commit (fast, only fetches one commit)
	for _, step := range []struct {
		name string
		args []string
	}{
		{"init", []string{"init", tmpDir}},
		{"remote", []string{"-C", tmpDir, "remote", "add", "origin", repoURL}},
		{"fetch", []string{"-C", tmpDir, "fetch", "--depth", "1", "origin", task.BaseCommit}},
		{"checkout", []string{"-C", tmpDir, "checkout", "FETCH_HEAD"}},
	} {
		cmd := exec.Command("git", step.args...)
		cmd.Env = gitEnv
		if out, err := cmd.CombinedOutput(); err != nil {
			cr.Error = fmt.Sprintf("git %s: %s (%v)", step.name, truncate(string(out), 200), err)
			return cr
		}
	}

	// Build the task prompt
	taskPrompt := fmt.Sprintf("Fix this GitHub issue in the %s repository.\n\n%s\n\nMake the minimal code changes needed to resolve the issue. Do not modify tests.", task.Repo, task.ProblemStatement)

	// Run Claude Code
	args := []string{
		"-p", taskPrompt,
		"--model", cfg.Model,
		"--max-budget-usd", fmt.Sprintf("%.2f", cfg.MaxBudgetUSD),
		"--allowedTools", "Bash,Read,Write,Edit,Grep,Glob",
		"--dangerously-skip-permissions",
	}
	if systemPrompt != "" {
		args = append(args, "--append-system-prompt", systemPrompt)
	}

	claudeCmd := exec.Command(cfg.ClaudeBin, args...)
	claudeCmd.Dir = tmpDir
	claudeCmd.Env = cleanClaudeEnv()

	if out, err := claudeCmd.CombinedOutput(); err != nil {
		cr.Error = fmt.Sprintf("claude: %s (%v)", truncate(string(out), 500), err)
		return cr
	}

	// Stage all changes (including new files) and capture the full diff
	addCmd := exec.Command("git", "-C", tmpDir, "add", "-A")
	addCmd.Env = gitEnv
	if _, err := addCmd.CombinedOutput(); err != nil {
		cr.Error = fmt.Sprintf("git add: %v", err)
		return cr
	}

	diffCmd := exec.Command("git", "-C", tmpDir, "diff", "--cached")
	diffCmd.Env = gitEnv
	diffOutput, err := diffCmd.Output()
	if err != nil {
		cr.Error = fmt.Sprintf("git diff: %v", err)
		return cr
	}

	patch := string(diffOutput)
	if strings.TrimSpace(patch) == "" {
		cr.Error = "no changes produced"
		return cr
	}

	cr.Patch = patch
	cr.Applied = true

	// Validate against gold patch
	if task.GoldPatch != "" {
		cr.FileOverlap, cr.FilesMatch, cr.ExactMatch = comparePatchToGold(patch, task.GoldPatch)
	}

	return cr
}

// EvalClaudeContestant evaluates a single prompt against all SWE-bench tasks.
func EvalClaudeContestant(name string, systemPrompt string, words int, tasks []SWEBenchTask, cfg ClaudeRunConfig) ClaudeRunContestantResult {
	cr := ClaudeRunContestantResult{
		Name:  name,
		Words: words,
		Tasks: len(tasks),
	}

	type indexedResult struct {
		index int
		r     ClaudeRunResult
	}

	results := make(chan indexedResult, len(tasks))
	sem := make(chan struct{}, cfg.Workers)

	var wg sync.WaitGroup
	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t SWEBenchTask) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			r := RunClaudeBenchTask(t, systemPrompt, cfg)
			results <- indexedResult{index: idx, r: r}
		}(i, task)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	ordered := make([]ClaudeRunResult, len(tasks))
	for r := range results {
		ordered[r.index] = r.r
	}

	var totalOverlap float64
	var overlapCount int
	for _, r := range ordered {
		cr.Details = append(cr.Details, r)
		if r.Applied {
			cr.Patched++
			if r.FilesMatch {
				cr.FilesMatch++
			}
			if r.ExactMatch {
				cr.ExactMatch++
			}
			totalOverlap += r.FileOverlap
			overlapCount++
		}
	}

	if cr.Tasks > 0 {
		cr.PatchRate = float64(cr.Patched) / float64(cr.Tasks) * 100
		cr.MatchRate = float64(cr.FilesMatch) / float64(cr.Tasks) * 100
	}
	if overlapCount > 0 {
		cr.AvgOverlap = totalOverlap / float64(overlapCount) * 100
	}

	return cr
}

// ClaudeContestant defines a named prompt variant for comparison.
type ClaudeContestant struct {
	Name   string
	Prompt string
	Words  int
}

// RunClaudeComparison evaluates multiple prompt contestants on the same tasks.
func RunClaudeComparison(contestants []ClaudeContestant, tasks []SWEBenchTask, cfg ClaudeRunConfig, progress func(string, ClaudeRunContestantResult)) ([]ClaudeRunContestantResult, error) {
	var results []ClaudeRunContestantResult

	for _, c := range contestants {
		r := EvalClaudeContestant(c.Name, c.Prompt, c.Words, tasks, cfg)
		results = append(results, r)
		if progress != nil {
			progress(c.Name, r)
		}
	}

	return results, nil
}

// WriteClaudePredictions writes Claude Code results as SWE-bench predictions JSONL.
func WriteClaudePredictions(path string, contestantName string, details []ClaudeRunResult) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	for _, d := range details {
		if d.Patch == "" {
			continue
		}
		if err := enc.Encode(swePrediction{
			InstanceID: d.InstanceID,
			ModelName:  contestantName,
			ModelPatch: d.Patch,
		}); err != nil {
			return err
		}
	}
	return nil
}

// CheckClaudeInstalled verifies the claude CLI is available.
func CheckClaudeInstalled(claudeBin string) error {
	cmd := exec.Command(claudeBin, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("claude CLI not found at %s: %s (%w)", claudeBin, string(output), err)
	}
	return nil
}

// --- Patch comparison helpers ---

// diffFileRe matches "diff --git a/path b/path" or "+++ b/path" lines in unified diffs.
var diffFileRe = regexp.MustCompile(`(?m)^diff --git a/(.+?) b/(.+?)$`)

// extractPatchFiles returns the sorted list of files modified in a unified diff.
func extractPatchFiles(patch string) []string {
	seen := make(map[string]bool)
	for _, m := range diffFileRe.FindAllStringSubmatch(patch, -1) {
		seen[m[2]] = true
	}
	files := make([]string, 0, len(seen))
	for f := range seen {
		files = append(files, f)
	}
	sort.Strings(files)
	return files
}

// comparePatchToGold compares a generated patch against the gold-standard patch.
// Returns: fileOverlap (0.0-1.0), filesMatch (exact file set), exactMatch (identical patch).
func comparePatchToGold(generated, gold string) (fileOverlap float64, filesMatch bool, exactMatch bool) {
	goldFiles := extractPatchFiles(gold)
	genFiles := extractPatchFiles(generated)

	if len(goldFiles) == 0 {
		return 0, false, false
	}

	// Exact patch match (normalize whitespace)
	exactMatch = normalizePatch(generated) == normalizePatch(gold)

	// File overlap: what fraction of gold files did the agent touch?
	goldSet := make(map[string]bool, len(goldFiles))
	for _, f := range goldFiles {
		goldSet[f] = true
	}
	genSet := make(map[string]bool, len(genFiles))
	for _, f := range genFiles {
		genSet[f] = true
	}

	var overlap int
	for f := range goldSet {
		if genSet[f] {
			overlap++
		}
	}
	fileOverlap = float64(overlap) / float64(len(goldFiles))

	// Files match: exact same set of files
	if len(goldFiles) == len(genFiles) && overlap == len(goldFiles) {
		filesMatch = true
	}

	return
}

// normalizePatch strips trailing whitespace and normalizes line endings for comparison.
func normalizePatch(patch string) string {
	lines := strings.Split(patch, "\n")
	var normalized []string
	for _, line := range lines {
		normalized = append(normalized, strings.TrimRight(line, " \t\r"))
	}
	return strings.TrimSpace(strings.Join(normalized, "\n"))
}

// cleanClaudeEnv returns os.Environ() with nesting-detection vars removed,
// so that a child Claude Code process doesn't think it's inside a parent session.
func cleanClaudeEnv() []string {
	skip := map[string]bool{
		"CLAUDECODE":             true,
		"CLAUDE_CODE_ENTRYPOINT": true,
	}
	var env []string
	for _, e := range os.Environ() {
		key := e
		if idx := strings.IndexByte(e, '='); idx >= 0 {
			key = e[:idx]
		}
		if skip[key] {
			continue
		}
		env = append(env, e)
	}
	return env
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
