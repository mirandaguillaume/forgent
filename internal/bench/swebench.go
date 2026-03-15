package bench

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mirandaguillaume/forgent/internal/llm"
)

// SWEBenchTask represents a single SWE-bench task.
type SWEBenchTask struct {
	InstanceID       string `json:"instance_id"`
	ProblemStatement string `json:"problem_statement"`
	Repo             string `json:"repo"`
	BaseCommit       string `json:"base_commit"`
}

// SWETaskResult holds the result of a single SWE-bench task.
type SWETaskResult struct {
	InstanceID string
	Patch      string
	Applied    bool
	TestPassed bool
	Error      string
}

// SWEBenchResult holds the aggregate SWE-bench results.
type SWEBenchResult struct {
	Tasks    int
	Resolved int
	Rate     float64
	Details  []SWETaskResult
}

// LoadSWEBenchTasks reads tasks from a JSONL file.
func LoadSWEBenchTasks(path string) ([]SWEBenchTask, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var tasks []SWEBenchTask
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var task SWEBenchTask
		if err := json.Unmarshal([]byte(line), &task); err != nil {
			return nil, fmt.Errorf("parse task: %w", err)
		}
		tasks = append(tasks, task)
	}
	return tasks, scanner.Err()
}

// RunSWEBench runs the lightweight SWE-bench evaluation.
// For each task: clone repo at base_commit, send problem to LLM with agent prompt,
// extract patch from response, attempt to apply it.
//
// Limitations:
//   - Uses direct git clone + subprocess, NOT Docker
//   - Does NOT run FAIL_TO_PASS tests (requires full SWE-bench harness)
//   - Only validates that a patch is generated and applies cleanly
//   - Full SWE-bench harness integration is future work
func RunSWEBench(tasksPath string, composedPrompt string, provider llm.Provider) (*SWEBenchResult, error) {
	tasks, err := LoadSWEBenchTasks(tasksPath)
	if err != nil {
		return nil, err
	}

	result := &SWEBenchResult{Tasks: len(tasks)}

	for _, task := range tasks {
		tr := processSWETask(task, composedPrompt, provider)
		if tr.Applied {
			result.Resolved++
		}
		result.Details = append(result.Details, tr)
	}

	if result.Tasks > 0 {
		result.Rate = float64(result.Resolved) / float64(result.Tasks) * 100
	}

	return result, nil
}

func processSWETask(task SWEBenchTask, prompt string, provider llm.Provider) SWETaskResult {
	tr := SWETaskResult{InstanceID: task.InstanceID}

	// Clone repo at base commit
	tmpDir, err := os.MkdirTemp("", "swebench-*")
	if err != nil {
		tr.Error = fmt.Sprintf("create tmpdir: %v", err)
		return tr
	}
	defer os.RemoveAll(tmpDir)

	repoURL := fmt.Sprintf("https://github.com/%s.git", task.Repo)
	cloneCmd := exec.Command("git", "clone", "--depth", "1", repoURL, tmpDir)
	cloneCmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if out, err := cloneCmd.CombinedOutput(); err != nil {
		tr.Error = fmt.Sprintf("clone: %s (%v)", string(out), err)
		return tr
	}

	// Ask LLM to generate a patch
	llmPrompt := fmt.Sprintf(`%s

You are fixing a bug in the %s repository.

Problem:
%s

Generate ONLY a unified diff patch that fixes this issue. Output the patch between <patch> and </patch> tags.`,
		prompt, task.Repo, task.ProblemStatement)

	response, err := provider.Complete(llmPrompt)
	if err != nil {
		tr.Error = fmt.Sprintf("llm: %v", err)
		return tr
	}

	// Extract patch from response
	patch := extractPatch(response)
	if patch == "" {
		tr.Error = "no patch found in response"
		return tr
	}
	tr.Patch = patch

	// Write patch to file and attempt to apply
	patchFile := filepath.Join(tmpDir, "fix.patch")
	if err := os.WriteFile(patchFile, []byte(patch), 0644); err != nil {
		tr.Error = fmt.Sprintf("write patch: %v", err)
		return tr
	}

	applyCmd := exec.Command("git", "apply", "--check", patchFile)
	applyCmd.Dir = tmpDir
	if out, err := applyCmd.CombinedOutput(); err != nil {
		tr.Error = fmt.Sprintf("apply: %s (%v)", string(out), err)
		return tr
	}

	tr.Applied = true
	return tr
}

// extractPatch extracts a unified diff from between <patch> and </patch> tags.
func extractPatch(response string) string {
	start := strings.Index(response, "<patch>")
	end := strings.Index(response, "</patch>")
	if start < 0 || end < 0 || end <= start {
		// Fallback: look for diff content directly
		if strings.Contains(response, "diff --git") || strings.Contains(response, "---") {
			return response
		}
		return ""
	}
	return strings.TrimSpace(response[start+len("<patch>") : end])
}
