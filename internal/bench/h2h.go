package bench

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mirandaguillaume/forgent/internal/generator"
	"github.com/mirandaguillaume/forgent/internal/llm"
	yamlloader "github.com/mirandaguillaume/forgent/internal/yaml"
	"github.com/mirandaguillaume/forgent/pkg/analysis"
	"github.com/mirandaguillaume/forgent/pkg/model"
)

// H2HTask represents a single head-to-head evaluation task.
type H2HTask struct {
	ID         string
	Category   string
	Title      string
	Code       string   // inline code (legacy synthetic tasks)
	URL        string   // PR URL (Martian tasks)
	Criteria   []string
	Severities []string // parallel to Criteria — "Critical", "High", "Medium", "Low"
}

// H2HContestant represents a single contestant in the head-to-head benchmark.
type H2HContestant struct {
	Name       string
	Prompt     string
	Words      int
	Stages     []model.Stage          // nil for non-staged agents
	SkillSteps map[string][]string    // skill name → strategy.steps (nil for non-staged)
	BinaryPath string                 // if set, invoke binary instead of calling provider directly
}

// SeverityHits tracks hits/total per severity level.
type SeverityHits struct {
	CriticalHits, CriticalTotal int
	HighHits, HighTotal         int
	MediumHits, MediumTotal     int
	LowHits, LowTotal           int
}

// Rate returns the hit rate for a severity level as a percentage (0-100).
func (s SeverityHits) Rate(level string) float64 {
	var hits, total int
	switch level {
	case "Critical":
		hits, total = s.CriticalHits, s.CriticalTotal
	case "High":
		hits, total = s.HighHits, s.HighTotal
	case "Medium":
		hits, total = s.MediumHits, s.MediumTotal
	case "Low":
		hits, total = s.LowHits, s.LowTotal
	}
	if total == 0 {
		return 0
	}
	return float64(hits) / float64(total) * 100
}

// PassRecord holds one pass's input/output for history tracking.
type PassRecord struct {
	Pass        int
	Response    string
	InputWords  int
	OutputWords int
}

// H2HTaskResult holds the result for one (contestant, task) pair.
type H2HTaskResult struct {
	ContestantName string
	TaskID         string
	Score          float64 // severity-weighted hit rate (0-100)
	Reasoning      string
	Severity       SeverityHits
	InputWords     int          // total input words (review + judge prompts)
	OutputWords    int          // total output words (review + judge responses)
	LLMCalls       int          // number of provider.Complete calls
	Passes         int          // number of review passes executed
	PassDetails    []PassRecord // per-pass details (populated when passes > 1)
}

// H2HContestantResult holds aggregate results for one contestant.
type H2HContestantResult struct {
	Name       string
	Words      int
	AvgScore   float64 // mean severity-weighted score across tasks
	EstCost    float64 // estimated API cost in dollars
	CostScore  float64 // AvgScore / EstCost — quality per dollar
	LLMCalls   int     // total provider.Complete calls across all tasks
	Tasks      int
	Details    []H2HTaskResult
	Severity   SeverityHits // aggregate across all tasks
}

// H2HResult holds the full benchmark result.
type H2HResult struct {
	Contestants []H2HContestantResult
}

// LoadContestants reads hand-written agent .md files from a fixtures directory.
func LoadContestants(fixturesDir string) ([]H2HContestant, error) {
	entries, err := os.ReadDir(fixturesDir)
	if err != nil {
		return nil, fmt.Errorf("read fixtures dir: %w", err)
	}

	var contestants []H2HContestant
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(fixturesDir, entry.Name()))
		if err != nil {
			continue
		}
		prompt := string(data)
		name := strings.TrimSuffix(entry.Name(), ".md")
		contestants = append(contestants, H2HContestant{
			Name:   name,
			Prompt: prompt,
			Words:  generator.CountWords(prompt),
		})
	}

	return contestants, nil
}

// LoadForgentBuiltContestants loads Forgent-built agents and skills from the
// .claude/ build output directory as H2H contestants.
// It loads all agents from .claude/agents/*.md and the pr-reviewer skill.
func LoadForgentBuiltContestants(repoPath string) ([]H2HContestant, error) {
	var contestants []H2HContestant

	// Load agents (e.g. ci-reviewer.md)
	agentsDir := filepath.Join(repoPath, ".claude", "agents")
	if entries, err := os.ReadDir(agentsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}
			data, err := os.ReadFile(filepath.Join(agentsDir, entry.Name()))
			if err != nil {
				continue
			}
			prompt := string(data)
			agentName := strings.TrimSuffix(entry.Name(), ".md")
			c := H2HContestant{
				Name:   "forgent/" + agentName,
				Prompt: prompt,
				Words:  generator.CountWords(prompt),
			}

			// Try to load stage metadata from the agent YAML
			agentYAMLPath := filepath.Join(repoPath, "agents", agentName+".agent.yaml")
			if yamlData, err := os.ReadFile(agentYAMLPath); err == nil {
				if agent, err := yamlloader.ParseAgentYAML(string(yamlData)); err == nil && len(agent.Stages) > 0 {
					c.Stages = agent.Stages
					c.SkillSteps = loadSkillSteps(repoPath, agent.AllSkills())
				}
			}

			contestants = append(contestants, c)
		}
	}

	// Load the pr-reviewer skill
	skillPath := filepath.Join(repoPath, ".claude", "skills", "pr-reviewer", "SKILL.md")
	if data, err := os.ReadFile(skillPath); err == nil {
		prompt := string(data)
		contestants = append(contestants, H2HContestant{
			Name:   "forgent/pr-reviewer",
			Prompt: prompt,
			Words:  generator.CountWords(prompt),
		})
	}

	return contestants, nil
}

// loadSkillSteps loads strategy.steps from skill YAMLs for the given skill names.
func loadSkillSteps(repoPath string, skillNames []string) map[string][]string {
	steps := make(map[string][]string)
	for _, name := range skillNames {
		skillYAMLPath := filepath.Join(repoPath, "skills", name+".skill.yaml")
		data, err := os.ReadFile(skillYAMLPath)
		if err != nil {
			continue
		}
		skill, err := yamlloader.ParseSkillYAML(string(data))
		if err != nil {
			continue
		}
		if len(skill.Strategy.Steps) > 0 {
			steps[name] = skill.Strategy.Steps
		}
	}
	return steps
}

// LoadContestantFile loads a single .md file as an H2H contestant.
func LoadContestantFile(name, path string) (H2HContestant, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return H2HContestant{}, fmt.Errorf("read contestant %s: %w", name, err)
	}
	prompt := string(data)
	return H2HContestant{
		Name:   name,
		Prompt: prompt,
		Words:  generator.CountWords(prompt),
	}, nil
}

// LoadForgentContestants builds Forgent standard + compact prompts from imported specs.
func LoadForgentContestants(importedDir string) ([]H2HContestant, error) {
	var contestants []H2HContestant

	entries, err := os.ReadDir(importedDir)
	if err != nil {
		return nil, fmt.Errorf("read imported dir: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		source := entry.Name()

		// Load standard output
		stdDir := filepath.Join(importedDir, source, "output-standard")
		stdPrompt, err := readAllFilesInDir(stdDir)
		if err == nil && stdPrompt != "" {
			contestants = append(contestants, H2HContestant{
				Name:   source + "/forgent-standard",
				Prompt: stdPrompt,
				Words:  generator.CountWords(stdPrompt),
			})
		}

		// Load compact output
		compactDir := filepath.Join(importedDir, source, "output-compact")
		compactPrompt, err := readAllFilesInDir(compactDir)
		if err == nil && compactPrompt != "" {
			contestants = append(contestants, H2HContestant{
				Name:   source + "/forgent-compact",
				Prompt: compactPrompt,
				Words:  generator.CountWords(compactPrompt),
			})
		}
	}

	return contestants, nil
}

func readAllFilesInDir(dir string) (string, error) {
	var parts []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		parts = append(parts, string(data))
		return nil
	})
	if err != nil {
		return "", err
	}
	return strings.Join(parts, "\n\n---\n\n"), nil
}

// H2HTokenEntry holds the word count for a single source × variant.
type H2HTokenEntry struct {
	Source  string
	Variant string // "hand-written", "forgent-standard", "forgent-compact"
	Words   int
}

// H2HTokenResult holds the full token efficiency comparison.
type H2HTokenResult struct {
	Entries []H2HTokenEntry
	Sources []H2HSourceComparison
}

// H2HSourceComparison holds the comparison for a single source.
type H2HSourceComparison struct {
	Source          string
	HandWritten     int
	Standard        int
	Compact         int
	StandardSaving  float64 // percentage reduction vs hand-written
	CompactSaving   float64 // percentage reduction vs hand-written
}

// RunH2HTokens compares word counts across hand-written, forgent-standard, and forgent-compact
// variants without needing LLM calls.
func RunH2HTokens(fixturesDir, importedDir string) (*H2HTokenResult, error) {
	handWritten, err := LoadContestants(fixturesDir)
	if err != nil {
		return nil, fmt.Errorf("load hand-written: %w", err)
	}

	forgent, err := LoadForgentContestants(importedDir)
	if err != nil {
		return nil, fmt.Errorf("load forgent: %w", err)
	}

	result := &H2HTokenResult{}

	// Index all entries
	for _, c := range handWritten {
		result.Entries = append(result.Entries, H2HTokenEntry{
			Source:  c.Name,
			Variant: "hand-written",
			Words:   c.Words,
		})
	}
	for _, c := range forgent {
		variant := "forgent-standard"
		if strings.HasSuffix(c.Name, "/forgent-compact") {
			variant = "forgent-compact"
		}
		// Extract source name (e.g. "voltagent" from "voltagent/forgent-standard")
		source := strings.Split(c.Name, "/")[0]
		result.Entries = append(result.Entries, H2HTokenEntry{
			Source:  source,
			Variant: variant,
			Words:   c.Words,
		})
	}

	// Build per-source comparisons
	// Map hand-written names to imported source names.
	// Hand-written files use pattern: "{source}-{agent-name}.md" → contestant name "{source}-{agent-name}"
	// Imported dirs use just "{source}" (e.g., "voltagent", "lst97", "wshobson", "darcyegb")
	hwMap := make(map[string]int) // source prefix → words
	for _, c := range handWritten {
		hwMap[c.Name] = c.Words
	}

	// Group forgent variants by source
	type sourcePair struct {
		standard int
		compact  int
	}
	forgentMap := make(map[string]*sourcePair)
	for _, c := range forgent {
		parts := strings.SplitN(c.Name, "/", 2)
		source := parts[0]
		if forgentMap[source] == nil {
			forgentMap[source] = &sourcePair{}
		}
		if strings.HasSuffix(c.Name, "/forgent-standard") {
			forgentMap[source].standard = c.Words
		} else {
			forgentMap[source].compact = c.Words
		}
	}

	// Match hand-written to imported by prefix
	for source, pair := range forgentMap {
		comp := H2HSourceComparison{
			Source:   source,
			Standard: pair.standard,
			Compact:  pair.compact,
		}

		// Find matching hand-written agent
		for hwName, words := range hwMap {
			if strings.HasPrefix(hwName, source) {
				comp.HandWritten = words
				break
			}
		}

		if comp.HandWritten > 0 {
			comp.StandardSaving = (1.0 - float64(comp.Standard)/float64(comp.HandWritten)) * 100
			comp.CompactSaving = (1.0 - float64(comp.Compact)/float64(comp.HandWritten)) * 100
		}

		result.Sources = append(result.Sources, comp)
	}

	return result, nil
}

// EstimateCost estimates API cost in dollars from word counts.
// Uses Claude Sonnet 4 pricing via OpenRouter: $3/MTok input, $15/MTok output.
// Approximation: 1 word ≈ 1.33 tokens.
func EstimateCost(inputWords, outputWords int) float64 {
	const tokensPerWord = 1.33
	const inputPricePerMTok = 3.0  // $/MTok
	const outputPricePerMTok = 15.0 // $/MTok
	inputTokens := float64(inputWords) * tokensPerWord
	outputTokens := float64(outputWords) * tokensPerWord
	return (inputTokens*inputPricePerMTok + outputTokens*outputPricePerMTok) / 1_000_000
}

// H2HProgressFn is called after each contestant completes evaluation.
type H2HProgressFn func(completed, total int, contestant H2HContestantResult)

// Baselines returns baseline contestants with no domain-specific instructions.
// These measure the model's inherent capability, isolating the value added by agent prompts.
func Baselines() []H2HContestant {
	return []H2HContestant{
		{
			Name:   "baseline/no-prompt",
			Prompt: "",
			Words:  0,
		},
		{
			Name:   "baseline/minimal",
			Prompt: "You are an expert code reviewer.",
			Words:  generator.CountWords("You are an expert code reviewer."),
		},
	}
}

// RunH2HWithTasks evaluates contestants against provided tasks.
// passes controls the number of review passes per task (1 = single-pass, default).
func RunH2HWithTasks(contestants []H2HContestant, tasks []H2HTask, provider llm.Provider, passes int, progress H2HProgressFn) (*H2HResult, error) {
	if passes < 1 {
		passes = 1
	}
	result := &H2HResult{}

	for i, contestant := range contestants {
		cr := evalContestant(contestant, tasks, provider, passes)

		result.Contestants = append(result.Contestants, cr)

		if progress != nil {
			progress(i+1, len(contestants), cr)
		}
	}

	return result, nil
}

// evalContestant evaluates a single contestant against all tasks concurrently.
func evalContestant(contestant H2HContestant, tasks []H2HTask, provider llm.Provider, passes int) H2HContestantResult {
	cr := H2HContestantResult{
		Name:  contestant.Name,
		Words: contestant.Words,
	}

	type taskResult struct {
		index int
		tr    H2HTaskResult
	}

	results := make(chan taskResult, len(tasks))
	sem := make(chan struct{}, 6) // limit concurrency to 6

	var wg sync.WaitGroup
	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t H2HTask) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			tr := evalTaskMultiPass(contestant, t, provider, passes)
			results <- taskResult{index: idx, tr: tr}
		}(i, task)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect results in order
	ordered := make([]H2HTaskResult, len(tasks))
	for r := range results {
		ordered[r.index] = r.tr
	}

	totalScore := 0.0
	totalInputWords := 0
	totalOutputWords := 0

	for _, tr := range ordered {
		cr.Details = append(cr.Details, tr)
		totalScore += tr.Score
		totalInputWords += tr.InputWords
		totalOutputWords += tr.OutputWords
		cr.LLMCalls += tr.LLMCalls
		// Aggregate severity hits
		cr.Severity.CriticalHits += tr.Severity.CriticalHits
		cr.Severity.CriticalTotal += tr.Severity.CriticalTotal
		cr.Severity.HighHits += tr.Severity.HighHits
		cr.Severity.HighTotal += tr.Severity.HighTotal
		cr.Severity.MediumHits += tr.Severity.MediumHits
		cr.Severity.MediumTotal += tr.Severity.MediumTotal
		cr.Severity.LowHits += tr.Severity.LowHits
		cr.Severity.LowTotal += tr.Severity.LowTotal
	}

	if len(tasks) > 0 {
		cr.AvgScore = totalScore / float64(len(tasks))
	}
	cr.EstCost = EstimateCost(totalInputWords, totalOutputWords)
	if cr.EstCost > 0 {
		cr.CostScore = cr.AvgScore / cr.EstCost
	}
	cr.Tasks = len(tasks)

	return cr
}

// BuildReviewPrompt constructs the review prompt for a contestant+task pair.
// For tasks with inline Code, uses that directly.
// For tasks with a URL, fetches the PR diff.
func BuildReviewPrompt(contestantPrompt string, task H2HTask) string {
	code := task.Code
	if code == "" && task.URL != "" {
		diff, err := FetchPRDiff(task.URL)
		if err != nil {
			code = fmt.Sprintf("[Failed to fetch diff: %v]\nPR: %s — %s", err, task.URL, task.Title)
		} else {
			code = diff
		}
	}
	return fmt.Sprintf("%s\n\nReview the following code:\n```\n%s\n```", contestantPrompt, code)
}

// evalTask evaluates a single (contestant, task) pair.
func evalTask(contestant H2HContestant, task H2HTask, provider llm.Provider) H2HTaskResult {
	if contestant.BinaryPath != "" {
		return evalTaskBinary(contestant, task, provider)
	}

	reviewPrompt := BuildReviewPrompt(contestant.Prompt, task)
	reviewInputWords := generator.CountWords(reviewPrompt)

	response, err := provider.Complete(reviewPrompt)
	if err != nil {
		return H2HTaskResult{
			ContestantName: contestant.Name,
			TaskID:         task.ID,
			Reasoning:      fmt.Sprintf("error: %v", err),
			InputWords:     reviewInputWords,
			LLMCalls:       1,
		}
	}
	reviewOutputWords := generator.CountWords(response)

	judgeResult, err := JudgeH2HResponse(provider, task, response)
	if err != nil {
		return H2HTaskResult{
			ContestantName: contestant.Name,
			TaskID:         task.ID,
			Reasoning:      fmt.Sprintf("judge error: %v", err),
			InputWords:     reviewInputWords,
			OutputWords:    reviewOutputWords,
			LLMCalls:       1,
		}
	}

	return H2HTaskResult{
		ContestantName: contestant.Name,
		TaskID:         task.ID,
		Score:          judgeResult.WeightedScore,
		Reasoning:      judgeResult.Reasoning,
		Severity:       judgeResult.Severity,
		InputWords:     reviewInputWords + judgeResult.JudgeInputWords,
		OutputWords:    reviewOutputWords + judgeResult.JudgeOutputWords,
		LLMCalls:       2,
	}
}

// evalTaskBinary runs a compiled forgent runtime binary for a task and judges its review_comments output.
func evalTaskBinary(contestant H2HContestant, task H2HTask, provider llm.Provider) H2HTaskResult {
	// Fetch diff
	code := task.Code
	if code == "" && task.URL != "" {
		diff, err := FetchPRDiff(task.URL)
		if err != nil {
			return H2HTaskResult{
				ContestantName: contestant.Name,
				TaskID:         task.ID,
				Reasoning:      fmt.Sprintf("fetch diff error: %v", err),
				LLMCalls:       0,
			}
		}
		code = diff
	}

	// Write diff to temp file
	diffFile, err := os.CreateTemp("", "h2h-diff-*.patch")
	if err != nil {
		return H2HTaskResult{
			ContestantName: contestant.Name,
			TaskID:         task.ID,
			Reasoning:      fmt.Sprintf("tempfile error: %v", err),
		}
	}
	defer os.Remove(diffFile.Name())
	if _, err := diffFile.WriteString(code); err != nil {
		diffFile.Close()
		return H2HTaskResult{ContestantName: contestant.Name, TaskID: task.ID, Reasoning: fmt.Sprintf("write diff error: %v", err)}
	}
	diffFile.Close()

	// Create temp output dir
	outDir, err := os.MkdirTemp("", "h2h-out-*")
	if err != nil {
		return H2HTaskResult{ContestantName: contestant.Name, TaskID: task.ID, Reasoning: fmt.Sprintf("tempdir error: %v", err)}
	}
	defer os.RemoveAll(outDir)

	// Compute code_analysis from the diff (tree-sitter AST + pattern scanner).
	codeAnalysis, _ := analysis.Analyze(code, nil)
	analysisFile, err := os.CreateTemp("", "h2h-analysis-*.md")
	if err != nil {
		return H2HTaskResult{ContestantName: contestant.Name, TaskID: task.ID, Reasoning: fmt.Sprintf("tempfile error: %v", err)}
	}
	defer os.Remove(analysisFile.Name())
	analysisFile.WriteString(codeAnalysis)
	analysisFile.Close()

	// Run binary
	cmd := exec.Command(contestant.BinaryPath,
		"--input", "git_diff=@"+diffFile.Name(),
		"--input", "code_analysis=@"+analysisFile.Name(),
		"--input", "file_tree=",
		"--input", "source_code=",
		"--output", outDir,
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return H2HTaskResult{
			ContestantName: contestant.Name,
			TaskID:         task.ID,
			Reasoning:      fmt.Sprintf("binary error: %v\n%s", err, string(out)),
			LLMCalls:       6,
		}
	}

	// Read review output — try review_comments.md (plural) then review_comment.md (singular).
	reviewPath := filepath.Join(outDir, "review_comments.md")
	if _, statErr := os.Stat(reviewPath); os.IsNotExist(statErr) {
		reviewPath = filepath.Join(outDir, "review_comment.md")
	}
	reviewBytes, err := os.ReadFile(reviewPath)
	if err != nil {
		return H2HTaskResult{
			ContestantName: contestant.Name,
			TaskID:         task.ID,
			Reasoning:      fmt.Sprintf("read review_comments error: %v", err),
			LLMCalls:       6,
		}
	}
	response := string(reviewBytes)
	reviewOutputWords := generator.CountWords(response)

	judgeResult, err := JudgeH2HResponse(provider, task, response)
	if err != nil {
		return H2HTaskResult{
			ContestantName: contestant.Name,
			TaskID:         task.ID,
			Reasoning:      fmt.Sprintf("judge error: %v", err),
			OutputWords:    reviewOutputWords,
			LLMCalls:       7,
		}
	}

	return H2HTaskResult{
		ContestantName: contestant.Name,
		TaskID:         task.ID,
		Score:          judgeResult.WeightedScore,
		Reasoning:      judgeResult.Reasoning,
		Severity:       judgeResult.Severity,
		InputWords:     generator.CountWords(code) + judgeResult.JudgeInputWords,
		OutputWords:    reviewOutputWords + judgeResult.JudgeOutputWords,
		LLMCalls:       7, // 6 skills + 1 judge
	}
}

// LoadRuntimeContestants loads compiled forgent runtime binaries from .forgent/ as H2H contestants.
// Each subdirectory containing a binary named after the agent becomes a contestant.
func LoadRuntimeContestants(repoPath string) ([]H2HContestant, error) {
	forgentDir := filepath.Join(repoPath, ".forgent")
	entries, err := os.ReadDir(forgentDir)
	if err != nil {
		return nil, nil // no .forgent dir, silently skip
	}

	var contestants []H2HContestant
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		agentName := entry.Name()
		// Try common naming conventions: exact, hyphens, underscores
		candidates := []string{
			agentName,
			strings.ReplaceAll(agentName, "_", "-"),
			strings.ReplaceAll(agentName, "-", "_"),
		}
		binaryPath := ""
		for _, candidate := range candidates {
			p := filepath.Join(forgentDir, agentName, candidate)
			if _, err := os.Stat(p); err == nil {
				binaryPath = p
				break
			}
		}
		if binaryPath == "" {
			continue
		}
		contestants = append(contestants, H2HContestant{
			Name:       "runtime/" + agentName,
			BinaryPath: binaryPath,
		})
	}
	return contestants, nil
}
