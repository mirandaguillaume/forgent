package bench

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/mirandaguillaume/forgent/internal/generator"
	"github.com/mirandaguillaume/forgent/internal/llm"
)

// SWEBenchTask represents a single SWE-bench task.
type SWEBenchTask struct {
	InstanceID       string `json:"instance_id"`
	ProblemStatement string `json:"problem_statement"`
	Repo             string `json:"repo"`
	BaseCommit       string `json:"base_commit"`
	GoldPatch        string `json:"patch"`      // gold-standard patch from SWE-bench
	HintsText        string `json:"hints_text"` // optional hints
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

	// Extract relevant files from gold patch and read them from the repo
	codeContext := buildCodeContext(task.GoldPatch, tmpDir)

	// Build prompt using agent's native template with {problem_statement} and {content}
	llmPrompt := buildSWEPrompt(prompt, task.ProblemStatement, codeContext, task.Repo)

	response, err := provider.Complete(llmPrompt)
	if err != nil {
		tr.Error = fmt.Sprintf("llm: %v", err)
		return tr
	}

	// Extract patch from response — supports unified diff, SEARCH/REPLACE, and ACR format
	patch := extractAndConvertPatch(response, tmpDir)
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

// buildCodeContext reads files referenced in the gold patch from the cloned repo.
func buildCodeContext(goldPatch, repoDir string) string {
	files := extractFilesFromPatch(goldPatch)
	if len(files) == 0 {
		return ""
	}

	var b strings.Builder
	for _, f := range files {
		content, err := os.ReadFile(filepath.Join(repoDir, f))
		if err != nil {
			continue
		}
		b.WriteString(fmt.Sprintf("### %s\n", f))
		b.Write(content)
		b.WriteString("\n\n")
	}
	return b.String()
}

// extractFilesFromPatch parses unified diff headers to find file paths.
func extractFilesFromPatch(patch string) []string {
	seen := make(map[string]bool)
	var files []string
	for _, line := range strings.Split(patch, "\n") {
		// Match "--- a/path/to/file" or "+++ b/path/to/file"
		if strings.HasPrefix(line, "--- a/") {
			f := strings.TrimPrefix(line, "--- a/")
			f = strings.TrimSpace(f)
			if f != "/dev/null" && !seen[f] {
				seen[f] = true
				files = append(files, f)
			}
		} else if strings.HasPrefix(line, "+++ b/") {
			f := strings.TrimPrefix(line, "+++ b/")
			f = strings.TrimSpace(f)
			if f != "/dev/null" && !seen[f] {
				seen[f] = true
				files = append(files, f)
			}
		}
	}
	return files
}

// buildSWEPrompt fills the agent's native template with problem statement and code context.
// If the prompt contains {problem_statement} or {content} placeholders, they are filled in.
// Otherwise, the problem statement and code context are appended.
func buildSWEPrompt(agentPrompt, problemStatement, codeContext, repo string) string {
	result := agentPrompt

	hasPS := strings.Contains(result, "{problem_statement}")
	hasContent := strings.Contains(result, "{content}")

	if hasPS {
		result = strings.Replace(result, "{problem_statement}", problemStatement, 1)
	}
	if hasContent {
		result = strings.Replace(result, "{content}", codeContext, 1)
	}

	// If the prompt had no placeholders, append the context
	if !hasPS && !hasContent {
		result += fmt.Sprintf("\n\nYou are fixing a bug in the %s repository.\n\n", repo)
		result += fmt.Sprintf("--- BEGIN ISSUE ---\n%s\n--- END ISSUE ---\n\n", problemStatement)
		if codeContext != "" {
			result += fmt.Sprintf("Below are the relevant source files:\n--- BEGIN FILE ---\n```\n%s```\n--- END FILE ---\n\n", codeContext)
		}
		result += "Generate a patch that fixes this issue. Output the patch between <patch> and </patch> tags as a unified diff."
	}

	return result
}

// extractAndConvertPatch tries multiple patch formats and converts to unified diff.
func extractAndConvertPatch(response, repoDir string) string {
	// 1. Try <patch>...</patch> tags with unified diff inside
	if patch := extractTagContent(response, "patch"); patch != "" {
		if isUnifiedDiff(patch) {
			return patch
		}
		// Might be SEARCH/REPLACE inside <patch> tags
		if ud := searchReplaceToUDiff(patch, repoDir); ud != "" {
			return ud
		}
	}

	// 2. Try AutoCodeRover format: <file>/<original>/<patched>
	if ud := acrToUDiff(response, repoDir); ud != "" {
		return ud
	}

	// 3. Try SEARCH/REPLACE blocks (Agentless / Aider format)
	if ud := searchReplaceToUDiff(response, repoDir); ud != "" {
		return ud
	}

	// 4. Try raw unified diff in the response
	if idx := strings.Index(response, "diff --git"); idx >= 0 {
		return response[idx:]
	}

	// 5. Try raw --- / +++ headers
	if idx := strings.Index(response, "--- a/"); idx >= 0 {
		return response[idx:]
	}

	return ""
}

// extractTagContent extracts content between <tag> and </tag>.
func extractTagContent(s, tag string) string {
	openTag := "<" + tag + ">"
	closeTag := "</" + tag + ">"
	start := strings.Index(s, openTag)
	end := strings.Index(s, closeTag)
	if start < 0 || end < 0 || end <= start {
		return ""
	}
	return strings.TrimSpace(s[start+len(openTag) : end])
}

// isUnifiedDiff checks if content looks like a unified diff.
func isUnifiedDiff(s string) bool {
	return strings.Contains(s, "diff --git") ||
		(strings.Contains(s, "--- a/") && strings.Contains(s, "+++ b/"))
}

// searchReplaceToUDiff converts SEARCH/REPLACE blocks to unified diff.
// Format: <<<<<<< SEARCH ... ======= ... >>>>>>> REPLACE with file path before each block.
func searchReplaceToUDiff(content, repoDir string) string {
	re := regexp.MustCompile(`(?m)^###?\s*(.+?)\s*$`)
	searchStart := "<<<<<<< SEARCH"
	divider := "======="
	replaceEnd := ">>>>>>> REPLACE"

	if !strings.Contains(content, searchStart) {
		return ""
	}

	var diffs []string
	currentFile := ""
	lines := strings.Split(content, "\n")

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])

		// Detect file path (### path/to/file or bare path ending in common extensions)
		if matches := re.FindStringSubmatch(lines[i]); len(matches) > 1 {
			candidate := strings.TrimSpace(matches[1])
			// Strip backticks or code fences
			candidate = strings.Trim(candidate, "`")
			if candidate != "" {
				currentFile = candidate
			}
			continue
		}

		// Detect bare file path (e.g. "path/to/file.py" on its own line)
		if strings.Contains(line, "/") && !strings.HasPrefix(line, "<") &&
			!strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "```") &&
			(strings.HasSuffix(line, ".py") || strings.HasSuffix(line, ".js") ||
				strings.HasSuffix(line, ".go") || strings.HasSuffix(line, ".ts") ||
				strings.HasSuffix(line, ".java") || strings.HasSuffix(line, ".rb") ||
				strings.HasSuffix(line, ".rs") || strings.HasSuffix(line, ".c") ||
				strings.HasSuffix(line, ".cpp") || strings.HasSuffix(line, ".h")) {
			currentFile = line
			continue
		}

		if line == searchStart && currentFile != "" {
			// Collect SEARCH block
			var searchLines, replaceLines []string
			i++
			for i < len(lines) && strings.TrimSpace(lines[i]) != divider {
				searchLines = append(searchLines, lines[i])
				i++
			}
			i++ // skip divider
			for i < len(lines) && strings.TrimSpace(lines[i]) != replaceEnd {
				replaceLines = append(replaceLines, lines[i])
				i++
			}

			// Build unified diff hunk
			udiff := buildUDiffHunk(currentFile, searchLines, replaceLines, repoDir)
			if udiff != "" {
				diffs = append(diffs, udiff)
			}
		}
	}

	if len(diffs) == 0 {
		return ""
	}
	return strings.Join(diffs, "\n")
}

// buildUDiffHunk creates a single unified diff hunk from search/replace content.
func buildUDiffHunk(filePath string, search, replace []string, repoDir string) string {
	// Read original file to find line number
	origContent, err := os.ReadFile(filepath.Join(repoDir, filePath))
	if err != nil {
		return ""
	}
	origLines := strings.Split(string(origContent), "\n")

	// Find the search block in the original file
	searchText := strings.Join(search, "\n")
	startLine := -1
	for i := 0; i <= len(origLines)-len(search); i++ {
		candidate := strings.Join(origLines[i:i+len(search)], "\n")
		if candidate == searchText {
			startLine = i + 1 // 1-indexed
			break
		}
	}

	if startLine < 0 {
		// Try trimmed matching as fallback
		for i := 0; i <= len(origLines)-len(search); i++ {
			match := true
			for j, s := range search {
				if strings.TrimSpace(origLines[i+j]) != strings.TrimSpace(s) {
					match = false
					break
				}
			}
			if match {
				startLine = i + 1
				break
			}
		}
	}

	if startLine < 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("--- a/%s\n", filePath))
	b.WriteString(fmt.Sprintf("+++ b/%s\n", filePath))
	b.WriteString(fmt.Sprintf("@@ -%d,%d +%d,%d @@\n", startLine, len(search), startLine, len(replace)))
	for _, l := range search {
		b.WriteString("-" + l + "\n")
	}
	for _, l := range replace {
		b.WriteString("+" + l + "\n")
	}

	return b.String()
}

// acrToUDiff converts AutoCodeRover <file>/<original>/<patched> format to unified diff.
func acrToUDiff(content, repoDir string) string {
	if !strings.Contains(content, "<original>") || !strings.Contains(content, "<patched>") {
		return ""
	}

	// Find all modifications
	fileRe := regexp.MustCompile(`<file>(.*?)</file>`)
	origRe := regexp.MustCompile(`(?s)<original>(.*?)</original>`)
	patchRe := regexp.MustCompile(`(?s)<patched>(.*?)</patched>`)

	fileMatches := fileRe.FindAllStringSubmatch(content, -1)
	origMatches := origRe.FindAllStringSubmatch(content, -1)
	patchMatches := patchRe.FindAllStringSubmatch(content, -1)

	n := len(fileMatches)
	if len(origMatches) < n {
		n = len(origMatches)
	}
	if len(patchMatches) < n {
		n = len(patchMatches)
	}
	if n == 0 {
		return ""
	}

	var diffs []string
	for i := 0; i < n; i++ {
		filePath := strings.TrimSpace(fileMatches[i][1])
		origText := strings.TrimSpace(origMatches[i][1])
		patchText := strings.TrimSpace(patchMatches[i][1])

		origLines := strings.Split(origText, "\n")
		patchLines := strings.Split(patchText, "\n")

		udiff := buildUDiffHunk(filePath, origLines, patchLines, repoDir)
		if udiff != "" {
			diffs = append(diffs, udiff)
		}
	}

	if len(diffs) == 0 {
		return ""
	}
	return strings.Join(diffs, "\n")
}

// --- Skill-chain execution (faithful to Forgent agent workflow) ---

// SWESkillStep represents a single skill in an agent's execution chain.
type SWESkillStep struct {
	Name   string // skill name (e.g. "analyze-issue")
	Prompt string // skill prompt content
}

// processSkillChainTask executes a SWE-bench task using an ordered chain of skills.
// Each skill receives the task context + accumulated outputs from previous skills.
// This faithfully reproduces how a Forgent agent runs in Claude Code.
func processSkillChainTask(task SWEBenchTask, skills []SWESkillStep, provider llm.Provider) SWETaskResult {
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

	// Build initial context from gold patch
	codeContext := buildCodeContext(task.GoldPatch, tmpDir)

	// Execute each skill in sequence, accumulating outputs
	var previousOutputs []string
	var lastResponse string

	for _, skill := range skills {
		// Build the skill prompt with task context + previous skill outputs
		var prompt strings.Builder
		prompt.WriteString(skill.Prompt)
		prompt.WriteString("\n\n--- TASK CONTEXT ---\n")
		prompt.WriteString(fmt.Sprintf("Repository: %s\n\n", task.Repo))
		prompt.WriteString(fmt.Sprintf("Issue:\n%s\n", task.ProblemStatement))
		if codeContext != "" {
			prompt.WriteString(fmt.Sprintf("\nRelevant code:\n%s\n", codeContext))
		}

		// Inject outputs from previous skills
		if len(previousOutputs) > 0 {
			prompt.WriteString("\n--- PREVIOUS SKILL OUTPUTS ---\n")
			for _, out := range previousOutputs {
				prompt.WriteString(out)
				prompt.WriteString("\n---\n")
			}
		}

		response, err := provider.Complete(prompt.String())
		if err != nil {
			tr.Error = fmt.Sprintf("llm skill %s: %v", skill.Name, err)
			return tr
		}
		previousOutputs = append(previousOutputs, fmt.Sprintf("[%s]\n%s", skill.Name, response))
		lastResponse = response
	}

	// Extract patch from the final skill's response
	patch := extractAndConvertPatch(lastResponse, tmpDir)
	if patch == "" {
		tr.Error = "no patch found in final skill response"
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

// --- SWE-bench H2H: multi-contestant comparison ---

// SWEContestant represents a single contestant in the SWE-bench H2H.
type SWEContestant struct {
	Name   string
	Prompt string
	Words  int
	// SkillChain is set for Forgent-decomposed agents (multi-skill execution).
	// When non-nil, skills are executed sequentially instead of a single prompt.
	SkillChain []SWESkillStep
}

// SWEContestantResult holds aggregate SWE-bench results for one contestant.
type SWEContestantResult struct {
	Name        string
	Words       int
	Tasks       int
	Patched     int     // produced a valid patch
	Applied     int     // patch applies cleanly
	Resolved    int     // patch passes FAIL_TO_PASS tests (via harness)
	PatchRate   float64 // % tasks that produced a patch
	ApplyRate   float64 // % tasks where patch applies
	ResolveRate float64 // % tasks resolved (real SWE-bench metric)
	Efficiency  float64 // resolve rate per 1000 words (or apply rate if no harness)
	Details     []SWETaskResult
}

// SWEH2HResult holds the full multi-contestant SWE-bench comparison.
type SWEH2HResult struct {
	Contestants []SWEContestantResult
	Tasks       int
}

// SWEH2HProgressFn is called after each contestant completes.
type SWEH2HProgressFn func(completed, total int, result SWEContestantResult)

// LoadSWEContestants reads hand-written agent .md files from a directory.
func LoadSWEContestants(dir string) ([]SWEContestant, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read swe agents dir: %w", err)
	}

	var contestants []SWEContestant
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		prompt := string(data)
		name := strings.TrimSuffix(entry.Name(), ".md")
		contestants = append(contestants, SWEContestant{
			Name:   name,
			Prompt: prompt,
			Words:  generator.CountWords(prompt),
		})
	}
	return contestants, nil
}

// LoadSWEForgentContestants builds forgent contestants from imported SWE-bench agents.
// Standard variant uses skill-chain execution (faithful to real agent workflow).
// Compact variant uses single-prompt execution (all skills inlined).
func LoadSWEForgentContestants(importedDir string) ([]SWEContestant, error) {
	var contestants []SWEContestant

	entries, err := os.ReadDir(importedDir)
	if err != nil {
		return nil, fmt.Errorf("read swe imported dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		source := entry.Name()

		// Standard variant: skill-chain execution (multi-step, faithful to agent workflow)
		stdDir := filepath.Join(importedDir, source, "output-standard")
		chain, totalWords, err := loadSkillChain(stdDir)
		if err == nil && len(chain) > 0 {
			contestants = append(contestants, SWEContestant{
				Name:       source + "/forgent-standard",
				Words:      totalWords,
				SkillChain: chain,
			})
		}

		// Compact variant: single prompt (all skills inlined in one file)
		compactDir := filepath.Join(importedDir, source, "output-compact")
		compactPrompt, err := readAllFilesInDir(compactDir)
		if err == nil && compactPrompt != "" {
			contestants = append(contestants, SWEContestant{
				Name:   source + "/forgent-compact",
				Prompt: compactPrompt,
				Words:  generator.CountWords(compactPrompt),
			})
		}
	}
	return contestants, nil
}

// loadSkillChain reads individual skill files from a build output directory
// and returns them as an ordered skill chain.
func loadSkillChain(outputDir string) ([]SWESkillStep, int, error) {
	skillsDir := filepath.Join(outputDir, "skills")
	agentsDir := filepath.Join(outputDir, "agents")

	// Read agent file to get skill order
	agentEntries, err := os.ReadDir(agentsDir)
	if err != nil {
		// Fallback: no agents dir, try reading skills directly
		return loadSkillsAlphabetically(skillsDir)
	}

	// Parse agent .md to extract skill execution order
	var skillOrder []string
	for _, ae := range agentEntries {
		if ae.IsDir() || !strings.HasSuffix(ae.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(agentsDir, ae.Name()))
		if err != nil {
			continue
		}
		// Extract skill names from "### Step N: SkillName" lines
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "### Step ") {
				// "### Step 1: Analyze Issue" → "analyze-issue"
				parts := strings.SplitN(line, ": ", 2)
				if len(parts) == 2 {
					skillOrder = append(skillOrder, strings.TrimSpace(parts[1]))
				}
			}
		}
		break // use first agent file
	}

	// Read each skill file in order
	var chain []SWESkillStep
	totalWords := 0

	for _, skillTitle := range skillOrder {
		// Convert title back to directory name (e.g. "Analyze Issue" → try various forms)
		skillPrompt, name := findAndReadSkill(skillsDir, skillTitle)
		if skillPrompt == "" {
			continue
		}
		chain = append(chain, SWESkillStep{Name: name, Prompt: skillPrompt})
		totalWords += generator.CountWords(skillPrompt)
	}

	if len(chain) == 0 {
		// Fallback: read skills alphabetically
		return loadSkillsAlphabetically(skillsDir)
	}

	return chain, totalWords, nil
}

// findAndReadSkill tries to find and read a skill file by its title.
func findAndReadSkill(skillsDir, title string) (string, string) {
	// Try exact directory match and common transformations
	candidates := []string{
		strings.ToLower(strings.ReplaceAll(title, " ", "-")),
		strings.ToLower(strings.ReplaceAll(title, " ", "_")),
		strings.ToLower(strings.ReplaceAll(title, " ", "")),
	}

	for _, candidate := range candidates {
		skillFile := filepath.Join(skillsDir, candidate, "SKILL.md")
		data, err := os.ReadFile(skillFile)
		if err == nil {
			return string(data), candidate
		}
	}

	// Try scanning all skill directories
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return "", ""
	}
	titleLower := strings.ToLower(title)
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		nameLower := strings.ToLower(strings.ReplaceAll(e.Name(), "-", " "))
		if nameLower == titleLower || strings.Contains(nameLower, titleLower) || strings.Contains(titleLower, nameLower) {
			data, err := os.ReadFile(filepath.Join(skillsDir, e.Name(), "SKILL.md"))
			if err == nil {
				return string(data), e.Name()
			}
		}
	}
	return "", ""
}

// loadSkillsAlphabetically reads all skills from a directory in alphabetical order.
func loadSkillsAlphabetically(skillsDir string) ([]SWESkillStep, int, error) {
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return nil, 0, err
	}

	var chain []SWESkillStep
	totalWords := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(skillsDir, e.Name(), "SKILL.md"))
		if err != nil {
			continue
		}
		prompt := string(data)
		chain = append(chain, SWESkillStep{Name: e.Name(), Prompt: prompt})
		totalWords += generator.CountWords(prompt)
	}
	return chain, totalWords, nil
}

// RunSWEBenchH2H loads contestants from fixture directories and evaluates them.
// It is a convenience wrapper around RunSWEBenchH2HWithContestants.
func RunSWEBenchH2H(tasksPath, agentsDir, importedDir string, provider llm.Provider, progress SWEH2HProgressFn) (*SWEH2HResult, error) {
	handWritten, err := LoadSWEContestants(agentsDir)
	if err != nil {
		return nil, fmt.Errorf("load swe agents: %w", err)
	}

	var forgent []SWEContestant
	if importedDir != "" {
		forgent, err = LoadSWEForgentContestants(importedDir)
		if err != nil {
			return nil, fmt.Errorf("load swe forgent: %w", err)
		}
	}

	return RunSWEBenchH2HWithContestants(append(handWritten, forgent...), tasksPath, provider, progress)
}

// RunSWEBenchH2HWithContestants evaluates pre-loaded contestants against SWE-bench tasks.
// Tasks for each contestant run concurrently (up to 4 at a time — lower than H2H because
// each task clones a repo).
func RunSWEBenchH2HWithContestants(contestants []SWEContestant, tasksPath string, provider llm.Provider, progress SWEH2HProgressFn) (*SWEH2HResult, error) {
	tasks, err := LoadSWEBenchTasks(tasksPath)
	if err != nil {
		return nil, err
	}

	result := &SWEH2HResult{Tasks: len(tasks)}

	for i, contestant := range contestants {
		cr := evalSWEContestant(contestant, tasks, provider)
		result.Contestants = append(result.Contestants, cr)

		if progress != nil {
			progress(i+1, len(contestants), cr)
		}
	}

	return result, nil
}

// evalSWEContestant evaluates a single contestant against all SWE-bench tasks concurrently.
func evalSWEContestant(contestant SWEContestant, tasks []SWEBenchTask, provider llm.Provider) SWEContestantResult {
	cr := SWEContestantResult{
		Name:  contestant.Name,
		Words: contestant.Words,
		Tasks: len(tasks),
	}

	type taskResult struct {
		index int
		tr    SWETaskResult
	}

	results := make(chan taskResult, len(tasks))
	sem := make(chan struct{}, 4) // limit concurrency (repo clones are heavy)

	var wg sync.WaitGroup
	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t SWEBenchTask) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			var tr SWETaskResult
			if len(contestant.SkillChain) > 0 {
				tr = processSkillChainTask(t, contestant.SkillChain, provider)
			} else {
				tr = processSWETask(t, contestant.Prompt, provider)
			}
			results <- taskResult{index: idx, tr: tr}
		}(i, task)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	ordered := make([]SWETaskResult, len(tasks))
	for r := range results {
		ordered[r.index] = r.tr
	}

	for _, tr := range ordered {
		cr.Details = append(cr.Details, tr)
		if tr.Patch != "" {
			cr.Patched++
		}
		if tr.Applied {
			cr.Applied++
		}
	}

	if cr.Tasks > 0 {
		cr.PatchRate = float64(cr.Patched) / float64(cr.Tasks) * 100
		cr.ApplyRate = float64(cr.Applied) / float64(cr.Tasks) * 100
		if contestant.Words > 0 {
			cr.Efficiency = cr.ApplyRate / (float64(contestant.Words) / 1000)
		}
	}

	return cr
}

// --- SWE-bench Harness Integration (real resolve rate via Docker) ---

// SWEHarnessConfig holds configuration for the SWE-bench harness.
type SWEHarnessConfig struct {
	PythonBin   string // path to python binary with swebench installed
	DatasetName string // e.g. "princeton-nlp/SWE-bench_Verified"
	MaxWorkers  int    // parallel Docker containers
}

// DefaultHarnessConfig returns sensible defaults for the SWE-bench harness.
func DefaultHarnessConfig() SWEHarnessConfig {
	return SWEHarnessConfig{
		PythonBin:   "/tmp/swebench-env/bin/python",
		DatasetName: "princeton-nlp/SWE-bench_Verified",
		MaxWorkers:  4,
	}
}

// swePrediction is the JSONL format expected by the SWE-bench harness.
type swePrediction struct {
	InstanceID string `json:"instance_id"`
	ModelName  string `json:"model_name_or_path"`
	ModelPatch string `json:"model_patch"`
}

// WritePredictions writes contestant results as a SWE-bench predictions JSONL file.
func WritePredictions(path string, contestantName string, details []SWETaskResult) error {
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

// RunHarnessEvaluation runs the official SWE-bench harness on a predictions file
// and returns the set of resolved instance IDs.
func RunHarnessEvaluation(cfg SWEHarnessConfig, predictionsPath, runID string) (map[string]bool, error) {
	args := []string{
		"-m", "swebench.harness.run_evaluation",
		"--dataset_name", cfg.DatasetName,
		"--predictions_path", predictionsPath,
		"--max_workers", fmt.Sprintf("%d", cfg.MaxWorkers),
		"--run_id", runID,
	}

	cmd := exec.Command(cfg.PythonBin, args...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("harness: %s (%w)", string(output), err)
	}

	// Parse the results report — the harness writes a JSON report
	return parseHarnessResults(runID, predictionsPath)
}

// parseHarnessResults reads the harness output to find resolved instances.
// SWE-bench v4 writes: {model_name}.{run_id}.json in the current working directory.
func parseHarnessResults(runID, predictionsPath string) (map[string]bool, error) {
	// Search in multiple locations: CWD, predictions dir, and logs subdirectory
	searchDirs := []string{
		".",
		filepath.Dir(predictionsPath),
	}

	var allMatches []string
	for _, dir := range searchDirs {
		for _, pattern := range []string{
			filepath.Join(dir, "*"+runID+"*.json"),
			filepath.Join(dir, "logs", "*"+runID+"*.json"),
		} {
			if m, err := filepath.Glob(pattern); err == nil {
				allMatches = append(allMatches, m...)
			}
		}
	}

	resolved := make(map[string]bool)

	for _, match := range allMatches {
		data, err := os.ReadFile(match)
		if err != nil {
			continue
		}

		var report map[string]interface{}
		if err := json.Unmarshal(data, &report); err != nil {
			continue
		}

		// SWE-bench v4 format uses "resolved_ids"
		for _, key := range []string{"resolved_ids", "resolved"} {
			if res, ok := report[key]; ok {
				if arr, ok := res.([]interface{}); ok {
					for _, v := range arr {
						if id, ok := v.(string); ok {
							resolved[id] = true
						}
					}
				}
			}
		}
	}

	return resolved, nil
}

// EvalContestantWithHarness runs patch generation + harness evaluation for a contestant.
// It first generates patches (like evalSWEContestant), then runs the official harness.
func EvalContestantWithHarness(contestant SWEContestant, tasks []SWEBenchTask, provider llm.Provider, cfg SWEHarnessConfig) SWEContestantResult {
	// Phase 1: Generate patches
	cr := evalSWEContestant(contestant, tasks, provider)

	// Phase 2: Write predictions and run harness
	tmpDir, err := os.MkdirTemp("", "swe-harness-*")
	if err != nil {
		return cr
	}
	defer os.RemoveAll(tmpDir)

	predPath := filepath.Join(tmpDir, "predictions.jsonl")
	if err := WritePredictions(predPath, contestant.Name, cr.Details); err != nil {
		return cr
	}

	runID := strings.ReplaceAll(contestant.Name, "/", "_")
	resolved, err := RunHarnessEvaluation(cfg, predPath, runID)
	if err != nil {
		// Harness failed — keep apply-rate metrics, log error
		return cr
	}

	// Phase 3: Update results with resolve data
	cr.Resolved = len(resolved)
	for i, d := range cr.Details {
		if resolved[d.InstanceID] {
			cr.Details[i].TestPassed = true
		}
	}

	if cr.Tasks > 0 {
		cr.ResolveRate = float64(cr.Resolved) / float64(cr.Tasks) * 100
		if contestant.Words > 0 {
			// Use resolve rate for efficiency when harness is available
			cr.Efficiency = cr.ResolveRate / (float64(contestant.Words) / 1000)
		}
	}

	return cr
}

// RunSWEBenchH2HWithHarness is like RunSWEBenchH2HWithContestants but uses the real harness.
func RunSWEBenchH2HWithHarness(contestants []SWEContestant, tasksPath string, provider llm.Provider, cfg SWEHarnessConfig, progress SWEH2HProgressFn) (*SWEH2HResult, error) {
	tasks, err := LoadSWEBenchTasks(tasksPath)
	if err != nil {
		return nil, err
	}

	result := &SWEH2HResult{Tasks: len(tasks)}

	for i, contestant := range contestants {
		cr := EvalContestantWithHarness(contestant, tasks, provider, cfg)
		result.Contestants = append(result.Contestants, cr)

		if progress != nil {
			progress(i+1, len(contestants), cr)
		}
	}

	return result, nil
}
