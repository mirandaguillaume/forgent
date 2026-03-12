package bench

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/mirandaguillaume/forgent/internal/enricher"
	"github.com/mirandaguillaume/forgent/internal/scanner"
	"gopkg.in/yaml.v3"
)

const systemPrompt = `You are a codebase navigation assistant. You receive a compact codebase index and a task. Reply with ONLY the file path relative to the project root. No explanation, no markdown, no backticks.`

const promptTemplate = `INDEX:
%s

TASK: %s`

// RunAgent runs the agent navigation benchmark by calling the local claude CLI.
// If tasks is empty, it auto-generates tasks from the scanned index.
func RunAgent(root string, tasks []Task, model string) (*AgentResult, error) {
	ctx, err := scanner.ScanCodebase(root)
	if err != nil {
		return nil, err
	}

	if len(tasks) == 0 {
		tasks = AutoGenerateTasks(ctx)
	}

	rendered := enricher.RenderIndex(ctx)
	result := &AgentResult{Tasks: len(tasks)}
	var totalLatency time.Duration

	for _, task := range tasks {
		prompt := fmt.Sprintf(promptTemplate, rendered, task.Query)
		tr := TaskResult{Query: task.Query}

		start := time.Now()
		response, usage, err := callClaude(prompt, model)
		tr.Latency = time.Since(start)
		totalLatency += tr.Latency

		if usage != nil {
			result.TotalCost += usage.CostUSD
			result.TotalTokens += usage.TotalInputTokens
		}

		if err != nil {
			tr.Err = err
			result.Errors++
			result.Details = append(result.Details, tr)
			continue
		}

		tr.Response = strings.TrimSpace(response)
		tr.Hit = matchesExpected(tr.Response, task.ExpectedPaths)

		if tr.Hit {
			result.Hits++
		} else {
			result.Misses++
		}
		result.Details = append(result.Details, tr)
	}

	completed := result.Hits + result.Misses
	result.HitRate = safePercent(result.Hits, completed)
	if completed > 0 {
		result.AvgLatency = totalLatency / time.Duration(completed)
	}

	return result, nil
}

// matchesExpected checks if the agent's response matches any expected path.
// Supports exact match, parent-directory match, and multi-line responses
// (checks each line independently).
func matchesExpected(response string, expected []string) bool {
	// Split response into candidate paths (one per line or comma-separated).
	candidates := extractCandidates(response)

	for _, candidate := range candidates {
		for _, exp := range expected {
			if candidate == exp {
				return true
			}
			// Parent dir match: response "src/controllers" matches expected "src/controllers/auth.go"
			if strings.HasPrefix(exp, candidate+"/") {
				return true
			}
			// File in dir: response "src/controllers/auth.go" matches expected "src/controllers"
			dir := filepath.Dir(candidate)
			if dir == exp {
				return true
			}
			// Reverse prefix: expected "scripts" matches response "scripts/build.sh"
			if strings.HasPrefix(candidate, exp+"/") {
				return true
			}
		}
	}
	return false
}

// extractCandidates splits a response into individual path candidates,
// handling multi-line responses, comma separation, backticks, and markdown.
func extractCandidates(response string) []string {
	// Split by newlines and commas.
	var raw []string
	for _, line := range strings.Split(response, "\n") {
		for _, part := range strings.Split(line, ",") {
			raw = append(raw, part)
		}
	}

	var candidates []string
	for _, r := range raw {
		// Strip markdown, backticks, quotes, bullets, trailing slashes.
		clean := strings.Trim(r, " \t\r`\"'*->•")
		clean = strings.TrimSuffix(clean, "/")
		clean = strings.TrimPrefix(clean, "./")
		if clean == "" || clean == "." {
			continue
		}
		candidates = append(candidates, clean)
	}
	return candidates
}

// LoadTasks reads a YAML file containing benchmark tasks.
func LoadTasks(path string) ([]Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading tasks file: %w", err)
	}
	var tasks []Task
	if err := yaml.Unmarshal(data, &tasks); err != nil {
		return nil, fmt.Errorf("parsing tasks YAML: %w", err)
	}
	return tasks, nil
}

// AutoGenerateTasks creates simple navigation tasks from the scanned index.
// For each index entry with source files, it generates a "find this file" task.
func AutoGenerateTasks(ctx *scanner.CodebaseContext) []Task {
	var tasks []Task
	for _, entry := range ctx.Structure {
		for _, f := range entry.Files {
			// Skip directory hints (ending with /).
			if strings.HasSuffix(f, "/") {
				continue
			}
			// Skip config files.
			ext := filepath.Ext(f)
			if !scanner.SourceExts[ext] {
				continue
			}

			tasks = append(tasks, Task{
				Query:         fmt.Sprintf("Where is %s located in the project?", f),
				ExpectedPaths: []string{entry.Path + "/" + f, entry.Path},
			})

			if len(tasks) >= 20 {
				return tasks
			}
		}
	}
	return tasks
}

// claudeResponse represents the JSON output from `claude -p --output-format json`.
type claudeResponse struct {
	Result       string  `json:"result"`
	TotalCostUSD float64 `json:"total_cost_usd"`
	Usage        struct {
		InputTokens              int `json:"input_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		OutputTokens             int `json:"output_tokens"`
	} `json:"usage"`
}

// claudeUsage holds token/cost info from a single call.
type claudeUsage struct {
	TotalInputTokens int
	OutputTokens     int
	CostUSD          float64
}

// callClaude invokes the local claude CLI with a minimal system prompt
// (replacing the default ~46K token Claude Code prompt) to reduce cost.
func callClaude(prompt, model string) (string, *claudeUsage, error) {
	args := []string{
		"-p", prompt,
		"--output-format", "json",
		"--system-prompt", systemPrompt,
	}
	if model != "" {
		args = append(args, "--model", model)
	}

	cmd := exec.Command("claude", args...)
	// Allow running inside a Claude Code session by clearing the nesting guard.
	cmd.Env = filterEnv(os.Environ(), "CLAUDECODE")
	output, err := cmd.Output()
	if err != nil {
		return "", nil, fmt.Errorf("claude CLI: %w", err)
	}

	var resp claudeResponse
	if err := json.Unmarshal(output, &resp); err != nil {
		return strings.TrimSpace(string(output)), nil, nil
	}

	usage := &claudeUsage{
		TotalInputTokens: resp.Usage.InputTokens + resp.Usage.CacheCreationInputTokens + resp.Usage.CacheReadInputTokens,
		OutputTokens:     resp.Usage.OutputTokens,
		CostUSD:          resp.TotalCostUSD,
	}

	return resp.Result, usage, nil
}

// filterEnv returns env without the named variable.
func filterEnv(env []string, name string) []string {
	prefix := name + "="
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		if !strings.HasPrefix(e, prefix) {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// ClaudeAvailable checks if the claude CLI is in PATH.
func ClaudeAvailable() bool {
	_, err := exec.LookPath("claude")
	return err == nil
}
