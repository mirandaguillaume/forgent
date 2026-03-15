package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/mirandaguillaume/forgent/internal/bench"
	"github.com/mirandaguillaume/forgent/internal/bench/formal"
	"github.com/mirandaguillaume/forgent/internal/llm"
	"github.com/mirandaguillaume/forgent/internal/scanner"
	yamlloader "github.com/mirandaguillaume/forgent/internal/yaml"
	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/spf13/cobra"
)

func init() {
	benchCmd := &cobra.Command{
		Use:   "bench <repo-path>",
		Short: "Benchmark agent composition quality",
		Long:  "Evaluates agent composition properties: token overhead, determinism, isomorphism, formal properties, and LLM-based quality.",
		Args:  cobra.ExactArgs(1),
		RunE:  runBench,
	}
	benchCmd.Flags().String("level", "proxy", "Benchmark level: proxy, agent, token, determinism, isomorphism, formal, eval, consistency, swebench, mutate, all")
	benchCmd.Flags().String("tasks", "", "YAML file with navigation tasks (agent level only)")
	benchCmd.Flags().Int("samples", 100, "Number of files to sample (proxy level only)")
	benchCmd.Flags().String("model", "haiku", "Claude model for agent bench")
	benchCmd.Flags().String("skills", "skills", "Skills directory")
	benchCmd.Flags().String("agents", "agents", "Agents directory")
	benchCmd.Flags().Int("passes", 5, "Number of passes for consistency bench")
	benchCmd.Flags().String("eval-tasks", "", "YAML file with evaluation tasks (eval level only)")
	benchCmd.Flags().String("provider", "", "LLM provider: anthropic or openrouter (auto-detected from env)")
	rootCmd.AddCommand(benchCmd)
}

func runBench(cmd *cobra.Command, args []string) error {
	repoPath, err := filepath.Abs(args[0])
	if err != nil {
		return err
	}

	level, _ := cmd.Flags().GetString("level")

	switch level {
	case "proxy":
		return runProxyBench(cmd, repoPath)
	case "agent":
		return runAgentBench(cmd, repoPath)
	case "token":
		return runTokenBench(cmd, repoPath)
	case "determinism":
		return runDeterminismBench(cmd, repoPath)
	case "isomorphism":
		return runIsomorphismBench(cmd, repoPath)
	case "formal":
		return runFormalBench(cmd, repoPath)
	case "eval":
		return runEvalBench(cmd, repoPath)
	case "consistency":
		return runConsistencyBench(cmd, repoPath)
	case "swebench":
		return runSWEBenchCmd(cmd, repoPath)
	case "mutate":
		return runMutateBench(cmd, repoPath)
	case "all":
		return runAllStructural(cmd, repoPath)
	default:
		return fmt.Errorf("unknown level %q (use 'proxy', 'agent', 'token', 'determinism', 'isomorphism', 'formal', 'eval', 'consistency', 'swebench', 'mutate', or 'all')", level)
	}
}

func runProxyBench(cmd *cobra.Command, repoPath string) error {
	samples, _ := cmd.Flags().GetInt("samples")

	bold := color.New(color.Bold)
	bold.Fprintf(cmd.OutOrStdout(), "Proxy Benchmark: %s\n", repoPath)

	result, err := bench.RunProxy(repoPath, samples, time.Now().UnixNano())
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "  Source files:   %d\n", result.TotalSourceFiles)
	fmt.Fprintf(cmd.OutOrStdout(), "  Sampled:        %d\n", result.SampledFiles)

	reachColor := color.New(color.FgGreen)
	if result.Reachability < 80 {
		reachColor = color.New(color.FgYellow)
	}
	if result.Reachability < 60 {
		reachColor = color.New(color.FgRed)
	}
	reachColor.Fprintf(cmd.OutOrStdout(), "  Reachable:      %d/%d (%.1f%%)\n",
		result.ReachableFiles, result.SampledFiles, result.Reachability)

	fmt.Fprintf(cmd.OutOrStdout(), "  Index entries:  %d\n", result.IndexEntries)
	fmt.Fprintf(cmd.OutOrStdout(), "  Index size:     %d bytes\n", result.IndexBytes)

	return nil
}

func runAgentBench(cmd *cobra.Command, repoPath string) error {
	if !bench.ClaudeAvailable() {
		return fmt.Errorf("claude CLI not found in PATH (required for agent benchmark)")
	}

	model, _ := cmd.Flags().GetString("model")
	tasksFile, _ := cmd.Flags().GetString("tasks")

	bold := color.New(color.Bold)
	bold.Fprintf(cmd.OutOrStdout(), "Agent Benchmark: %s\n", repoPath)

	var tasks []bench.Task
	if tasksFile != "" {
		var err error
		tasks, err = bench.LoadTasks(tasksFile)
		if err != nil {
			return err
		}
	}

	result, err := bench.RunAgent(repoPath, tasks, model)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "  Tasks:          %d\n", result.Tasks)

	hitColor := color.New(color.FgGreen)
	if result.HitRate < 70 {
		hitColor = color.New(color.FgYellow)
	}
	if result.HitRate < 50 {
		hitColor = color.New(color.FgRed)
	}
	hitColor.Fprintf(cmd.OutOrStdout(), "  Hits:           %d (%.1f%%)\n", result.Hits, result.HitRate)

	if result.Misses > 0 {
		color.New(color.FgYellow).Fprintf(cmd.OutOrStdout(), "  Misses:         %d\n", result.Misses)
	}
	if result.Errors > 0 {
		color.New(color.FgRed).Fprintf(cmd.OutOrStdout(), "  Errors:         %d\n", result.Errors)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "  Avg latency:    %s\n", result.AvgLatency.Round(time.Millisecond))

	if result.TotalTokens > 0 {
		avgTokens := result.TotalTokens / result.Tasks
		fmt.Fprintf(cmd.OutOrStdout(), "  Avg tokens/task: %d\n", avgTokens)
		fmt.Fprintf(cmd.OutOrStdout(), "  Total cost:     $%.4f\n", result.TotalCost)
		fmt.Fprintf(cmd.OutOrStdout(), "  Cost/task:      $%.4f\n", result.TotalCost/float64(result.Tasks))
	}

	for _, d := range result.Details {
		if !d.Hit && d.Err == nil {
			fmt.Fprintf(cmd.OutOrStdout(), "  MISS: %q → %q\n", d.Query, d.Response)
		}
	}

	return nil
}

func runTokenBench(cmd *cobra.Command, repoPath string) error {
	skillsDir, _ := cmd.Flags().GetString("skills")
	agentsDir, _ := cmd.Flags().GetString("agents")

	bold := color.New(color.Bold)
	bold.Fprintf(cmd.OutOrStdout(), "Token Overhead Benchmark\n")

	for _, target := range []string{"claude", "copilot"} {
		result, err := bench.RunTokenOverhead(
			filepath.Join(repoPath, skillsDir),
			filepath.Join(repoPath, agentsDir),
			target)
		if err != nil {
			return fmt.Errorf("token bench (%s): %w", target, err)
		}

		overheadColor := color.New(color.FgGreen)
		if result.OverheadPct > 150 {
			overheadColor = color.New(color.FgYellow)
		}
		if result.OverheadPct > 200 {
			overheadColor = color.New(color.FgRed)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "  [%s] ", target)
		overheadColor.Fprintf(cmd.OutOrStdout(), "%.1f%% overhead", result.OverheadPct)
		fmt.Fprintf(cmd.OutOrStdout(), " (%d composed / %d monolithic words, %d files)\n",
			result.ComposedWords, result.MonolithicWords, result.ComposedFiles)
	}
	return nil
}

func runDeterminismBench(cmd *cobra.Command, repoPath string) error {
	skillsDir, _ := cmd.Flags().GetString("skills")
	agentsDir, _ := cmd.Flags().GetString("agents")

	bold := color.New(color.Bold)
	bold.Fprintf(cmd.OutOrStdout(), "Determinism Benchmark\n")

	for _, target := range []string{"claude", "copilot"} {
		result, err := bench.RunDeterminism(
			filepath.Join(repoPath, skillsDir),
			filepath.Join(repoPath, agentsDir),
			target, 3)
		if err != nil {
			return fmt.Errorf("determinism bench (%s): %w", target, err)
		}

		status := color.New(color.FgGreen)
		label := "PASS"
		if !result.Identical {
			status = color.New(color.FgRed)
			label = "FAIL"
		}

		fmt.Fprintf(cmd.OutOrStdout(), "  [%s] ", target)
		status.Fprintf(cmd.OutOrStdout(), "%s", label)
		fmt.Fprintf(cmd.OutOrStdout(), " (%d runs, %d diffs)\n", result.Runs, result.DiffCount)
	}
	return nil
}

func runIsomorphismBench(cmd *cobra.Command, repoPath string) error {
	skillsDir, _ := cmd.Flags().GetString("skills")
	agentsDir, _ := cmd.Flags().GetString("agents")

	bold := color.New(color.Bold)
	bold.Fprintf(cmd.OutOrStdout(), "Cross-Target Isomorphism Benchmark\n")

	result, err := bench.RunIsomorphism(
		filepath.Join(repoPath, skillsDir),
		filepath.Join(repoPath, agentsDir))
	if err != nil {
		return err
	}

	statusColor := color.New(color.FgGreen)
	if result.StructureScore < 1.0 {
		statusColor = color.New(color.FgRed)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "  Skill names match:  %v\n", result.SkillNamesMatch)
	fmt.Fprintf(cmd.OutOrStdout(), "  I/O contracts match: %v\n", result.IOContractsMatch)
	statusColor.Fprintf(cmd.OutOrStdout(), "  Structure score:    %.1f%%\n", result.StructureScore*100)
	fmt.Fprintf(cmd.OutOrStdout(), "  Claude skills:      %d\n", len(result.ClaudeSkills))
	fmt.Fprintf(cmd.OutOrStdout(), "  Copilot skills:     %d\n", len(result.CopilotSkills))

	return nil
}

func runFormalBench(cmd *cobra.Command, repoPath string) error {
	skillsDir, _ := cmd.Flags().GetString("skills")
	agentsDir, _ := cmd.Flags().GetString("agents")

	bold := color.New(color.Bold)
	bold.Fprintf(cmd.OutOrStdout(), "Formal Property Benchmark\n")

	// Load skills and agents
	skills, agents, err := loadSkillsAndAgents(
		filepath.Join(repoPath, skillsDir),
		filepath.Join(repoPath, agentsDir))
	if err != nil {
		return err
	}

	pass := color.New(color.FgGreen)
	fail := color.New(color.FgRed)

	for _, agent := range agents {
		var agentSkills []model.SkillBehavior
		for _, sName := range agent.Skills {
			for _, s := range skills {
				if s.Skill == sName {
					agentSkills = append(agentSkills, s)
				}
			}
		}

		fmt.Fprintf(cmd.OutOrStdout(), "  Agent: %s (%d skills)\n", agent.Agent, len(agentSkills))

		g := formal.BuildGraph(agent, agentSkills)

		// P10: Reachability
		allReachable := true
		for _, s := range agentSkills {
			if !g.Reachable(formal.Top, s.Skill) {
				allReachable = false
				break
			}
		}
		printProp(cmd, pass, fail, "P10 Reachability", allReachable)

		// Cor 3.1: Layers
		layers := g.Layers()
		printProp(cmd, pass, fail, "Cor3.1 Layer decomposition", len(layers) > 0)
		fmt.Fprintf(cmd.OutOrStdout(), "         %d layers, width %d\n", len(layers), g.Width())

		// P14: Parallel independence
		p14Ok := true
		for _, layer := range layers {
			var layerSkills []model.SkillBehavior
			for _, sName := range layer {
				for _, s := range agentSkills {
					if s.Skill == sName {
						layerSkills = append(layerSkills, s)
					}
				}
			}
			if !formal.DisjointProduces(layerSkills) {
				p14Ok = false
			}
		}
		printProp(cmd, pass, fail, "P14 Parallel independence", p14Ok)

		// P13: Containment
		p13Ok := true
		maxFS := model.AccessNone
		maxNet := model.NetworkNone
		for _, s := range agentSkills {
			if accessOrder[s.Security.Filesystem] > accessOrder[maxFS] {
				maxFS = s.Security.Filesystem
			}
			if networkOrder[s.Security.Network] > networkOrder[maxNet] {
				maxNet = s.Security.Network
			}
		}
		for _, s := range agentSkills {
			if !formal.AccessLevelContained(s.Security.Filesystem, maxFS) ||
				!formal.NetworkContained(s.Security.Network, maxNet) {
				p13Ok = false
			}
		}
		printProp(cmd, pass, fail, "P13 Containment", p13Ok)
	}

	return nil
}

var accessOrder = map[model.AccessLevel]int{
	model.AccessNone:      0,
	model.AccessReadOnly:  1,
	model.AccessReadWrite: 2,
	model.AccessFull:      3,
}

var networkOrder = map[model.NetworkAccess]int{
	model.NetworkNone:      0,
	model.NetworkAllowlist: 1,
	model.NetworkFull:      2,
}

func printProp(cmd *cobra.Command, pass, fail *color.Color, name string, ok bool) {
	if ok {
		pass.Fprintf(cmd.OutOrStdout(), "    PASS")
	} else {
		fail.Fprintf(cmd.OutOrStdout(), "    FAIL")
	}
	fmt.Fprintf(cmd.OutOrStdout(), " %s\n", name)
}

func runEvalBench(cmd *cobra.Command, repoPath string) error {
	skillsDir, _ := cmd.Flags().GetString("skills")
	agentsDir, _ := cmd.Flags().GetString("agents")
	evalTasksFile, _ := cmd.Flags().GetString("eval-tasks")

	if evalTasksFile == "" {
		return fmt.Errorf("--eval-tasks is required for eval level")
	}

	provider, err := resolveLLMProvider(cmd)
	if err != nil {
		return err
	}

	bold := color.New(color.Bold)
	bold.Fprintf(cmd.OutOrStdout(), "LLM-as-Judge Evaluation Benchmark\n")

	tasks, err := bench.LoadEvalTasks(evalTasksFile)
	if err != nil {
		return err
	}

	// Build composed prompt
	tmpDir, err := os.MkdirTemp("", "bench-eval-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	buildResult := RunBuild(
		filepath.Join(repoPath, skillsDir),
		filepath.Join(repoPath, agentsDir),
		tmpDir, "claude", scanner.EnrichNone)
	if !buildResult.Success {
		return fmt.Errorf("build failed: %s", buildResult.Error)
	}

	composedPrompt, err := readAllFiles(tmpDir)
	if err != nil {
		return err
	}

	monolithicPrompt := "You are a code reviewer. Review the code for bugs, security issues, performance problems, and best practice violations. Be thorough and specific."

	result, err := bench.RunEval(tasks, composedPrompt, monolithicPrompt, provider)
	if err != nil {
		return err
	}

	for _, r := range result.Results {
		winLabel := "TIE"
		if r.ComposedWins {
			winLabel = "WIN"
		} else if r.ComposedScore < r.MonolithicScore {
			winLabel = "LOSS"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  [%s] %s: Composed=%d, Monolithic=%d\n",
			winLabel, r.TaskID, r.ComposedScore, r.MonolithicScore)
	}

	winColor := color.New(color.FgGreen)
	if result.WinRate < 50 {
		winColor = color.New(color.FgYellow)
	}
	winColor.Fprintf(cmd.OutOrStdout(), "  Win rate: %.1f%% (%d/%d wins, %d ties)\n",
		result.WinRate, result.ComposedWins, result.Tasks, result.Ties)

	return nil
}

func runConsistencyBench(cmd *cobra.Command, repoPath string) error {
	evalTasksFile, _ := cmd.Flags().GetString("eval-tasks")
	passes, _ := cmd.Flags().GetInt("passes")

	if evalTasksFile == "" {
		return fmt.Errorf("--eval-tasks is required for consistency level")
	}

	provider, err := resolveLLMProvider(cmd)
	if err != nil {
		return err
	}

	bold := color.New(color.Bold)
	bold.Fprintf(cmd.OutOrStdout(), "Pass@k Consistency Benchmark (k=%d)\n", passes)

	tasks, err := bench.LoadEvalTasks(evalTasksFile)
	if err != nil {
		return err
	}

	prompt := "You are a code reviewer. Review the code for bugs, security issues, and best practice violations."

	for _, task := range tasks {
		result, err := bench.RunConsistency(task, prompt, provider, passes)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "  [ERROR] %s: %v\n", task.ID, err)
			continue
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  %s: %.0f%% consistency (%d unique / %d runs, avg %d chars)\n",
			result.Task, result.ConsistencyRate*100, result.UniqueResponses, result.Runs, result.AvgResponseLen)
	}

	return nil
}

func runSWEBenchCmd(cmd *cobra.Command, repoPath string) error {
	skillsDir, _ := cmd.Flags().GetString("skills")
	agentsDir, _ := cmd.Flags().GetString("agents")

	provider, err := resolveLLMProvider(cmd)
	if err != nil {
		return err
	}

	bold := color.New(color.Bold)
	bold.Fprintf(cmd.OutOrStdout(), "SWE-bench Benchmark\n")

	// Build composed prompt
	tmpDir, err := os.MkdirTemp("", "bench-swe-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	buildResult := RunBuild(
		filepath.Join(repoPath, skillsDir),
		filepath.Join(repoPath, agentsDir),
		tmpDir, "claude", scanner.EnrichNone)
	if !buildResult.Success {
		return fmt.Errorf("build failed: %s", buildResult.Error)
	}

	composedPrompt, err := readAllFiles(tmpDir)
	if err != nil {
		return err
	}

	tasksPath := filepath.Join(repoPath, "internal/bench/testdata/swebench_sample.jsonl")
	result, err := bench.RunSWEBench(tasksPath, composedPrompt, provider)
	if err != nil {
		return err
	}

	for _, d := range result.Details {
		status := "FAIL"
		statusColor := color.New(color.FgRed)
		if d.Applied {
			status = "PASS"
			statusColor = color.New(color.FgGreen)
		}
		statusColor.Fprintf(cmd.OutOrStdout(), "  [%s]", status)
		fmt.Fprintf(cmd.OutOrStdout(), " %s", d.InstanceID)
		if d.Error != "" {
			fmt.Fprintf(cmd.OutOrStdout(), " (%s)", d.Error)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	fmt.Fprintf(cmd.OutOrStdout(), "  Resolved: %d/%d (%.1f%%)\n", result.Resolved, result.Tasks, result.Rate)

	return nil
}

func runAllStructural(cmd *cobra.Command, repoPath string) error {
	for _, fn := range []struct {
		name string
		run  func(*cobra.Command, string) error
	}{
		{"token", runTokenBench},
		{"determinism", runDeterminismBench},
		{"isomorphism", runIsomorphismBench},
		{"formal", runFormalBench},
	} {
		if err := fn.run(cmd, repoPath); err != nil {
			return fmt.Errorf("%s: %w", fn.name, err)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}
	return nil
}

// loadSkillsAndAgents reads all skill and agent YAMLs from directories.
func loadSkillsAndAgents(skillsDir, agentsDir string) ([]model.SkillBehavior, []model.AgentComposition, error) {
	var skills []model.SkillBehavior
	if entries, err := os.ReadDir(skillsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(skillsDir, entry.Name()))
			if err != nil {
				continue
			}
			skill, err := yamlloader.ParseSkillYAML(string(data))
			if err != nil {
				continue
			}
			skills = append(skills, skill)
		}
	}

	var agents []model.AgentComposition
	if entries, err := os.ReadDir(agentsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() || filepath.Ext(entry.Name()) != ".yaml" {
				continue
			}
			data, err := os.ReadFile(filepath.Join(agentsDir, entry.Name()))
			if err != nil {
				continue
			}
			agent, err := yamlloader.ParseAgentYAML(string(data))
			if err != nil {
				continue
			}
			agents = append(agents, agent)
		}
	}

	return skills, agents, nil
}

// resolveLLMProvider picks the LLM provider from --provider flag or env vars.
func resolveLLMProvider(cmd *cobra.Command) (llm.Provider, error) {
	providerName, _ := cmd.Flags().GetString("provider")

	// Auto-detect from env if not explicitly set
	if providerName == "" {
		if os.Getenv("ANTHROPIC_API_KEY") != "" {
			providerName = "anthropic"
		} else if os.Getenv("OPENROUTER_API_KEY") != "" {
			providerName = "openrouter"
		} else {
			return nil, fmt.Errorf("no LLM API key found — set ANTHROPIC_API_KEY or OPENROUTER_API_KEY")
		}
	}

	var apiKey string
	switch providerName {
	case "anthropic":
		apiKey = os.Getenv("ANTHROPIC_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY required for provider %q", providerName)
		}
	case "openrouter":
		apiKey = os.Getenv("OPENROUTER_API_KEY")
		if apiKey == "" {
			return nil, fmt.Errorf("OPENROUTER_API_KEY required for provider %q", providerName)
		}
	default:
		return nil, fmt.Errorf("unknown provider %q (use 'anthropic' or 'openrouter')", providerName)
	}

	return llm.GetProvider(providerName, apiKey)
}

func runMutateBench(cmd *cobra.Command, repoPath string) error {
	bold := color.New(color.Bold)
	bold.Fprintln(cmd.OutOrStdout(), "Mutation Testing (gremlins)")

	// Check if gremlins is installed
	gremlinsPath, err := exec.LookPath("gremlins")
	if err != nil {
		return fmt.Errorf("gremlins not found in PATH — install with: go install github.com/go-gremlins/gremlins/cmd/gremlins@latest")
	}
	fmt.Fprintf(cmd.OutOrStdout(), "  Using: %s\n", gremlinsPath)

	// Build args
	args := []string{"unleash",
		"--coverpkg", "./internal/bench/...,./internal/bench/formal/...,./internal/analyzer/...,./internal/linter/...,./pkg/model/...",
	}

	// Check for --dry-run via env var
	if os.Getenv("GREMLINS_DRY_RUN") == "1" {
		args = append(args, "--dry-run")
	}

	gremlinsCmd := exec.Command(gremlinsPath, args...)
	gremlinsCmd.Dir = repoPath
	gremlinsCmd.Stdout = cmd.OutOrStdout()
	gremlinsCmd.Stderr = cmd.ErrOrStderr()

	fmt.Fprintf(cmd.OutOrStdout(), "  Running: gremlins %s\n\n", strings.Join(args, " "))
	if err := gremlinsCmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			switch exitErr.ExitCode() {
			case 10:
				color.New(color.FgYellow).Fprintln(cmd.OutOrStdout(), "\nTest efficacy below threshold")
			case 11:
				color.New(color.FgYellow).Fprintln(cmd.OutOrStdout(), "\nMutant coverage below threshold")
			default:
				return fmt.Errorf("gremlins exited with code %d", exitErr.ExitCode())
			}
			return nil // threshold warnings are not hard errors
		}
		return fmt.Errorf("gremlins: %w", err)
	}

	color.New(color.FgGreen).Fprintln(cmd.OutOrStdout(), "\nMutation testing passed all thresholds")
	return nil
}

// readAllFiles concatenates all files in a directory tree.
func readAllFiles(dir string) (string, error) {
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
	result := ""
	for i, p := range parts {
		if i > 0 {
			result += "\n\n---\n\n"
		}
		result += p
	}
	return result, nil
}
