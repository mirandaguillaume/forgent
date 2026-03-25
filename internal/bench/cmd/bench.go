package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/mirandaguillaume/forgent/internal/bench"
	"github.com/mirandaguillaume/forgent/internal/builder"
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
	benchCmd.Flags().String("level", "proxy", "Benchmark level: proxy, agent, token, determinism, isomorphism, formal, eval, consistency, swebench, swebench-h2h, swebench-live, mutate, h2h, h2h-tokens, all")
	benchCmd.Flags().String("tasks", "", "YAML file with navigation tasks (agent level only)")
	benchCmd.Flags().Int("samples", 100, "Number of files to sample (proxy level only)")
	benchCmd.Flags().String("model", "haiku", "Claude model for agent bench")
	benchCmd.Flags().String("skills", "skills", "Skills directory")
	benchCmd.Flags().String("agents", "agents", "Agents directory")
	benchCmd.Flags().Int("passes", 5, "Number of passes for consistency bench")
	benchCmd.Flags().String("eval-tasks", "", "YAML file with evaluation tasks (eval level only)")
	benchCmd.Flags().String("provider", "", "LLM provider: anthropic or openrouter (auto-detected from env)")
	benchCmd.Flags().String("fixtures", "", "Directory with hand-written agent .md files (overrides default for h2h/swebench levels)")
	benchCmd.Flags().String("swe-size", "verified", "SWE-bench dataset size: quick5, sample, lite, verified, live")
	benchCmd.Flags().Bool("harness", false, "Run official SWE-bench harness after generating patches (requires Docker + swebench)")
	benchCmd.Flags().String("predictions", "", "Path to predictions.jsonl for swebench-eval level")
	benchCmd.Flags().String("only", "", "Comma-separated list of contestant names to run (e.g. 'epoch-v2-original,handcrafted')")
	benchCmd.Flags().String("martian", "internal/bench/testdata/martian/", "Path to Martian golden comments directory")
	benchCmd.Flags().Int("review-passes", 1, "Number of review passes per task in H2H bench (1=single-pass)")
	benchCmd.Flags().String("output", "", "Write h2h results to this JSON file (appends if file exists)")
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
	case "h2h":
		return runH2HBench(cmd, repoPath)
	case "h2h-tokens":
		return runH2HTokensBench(cmd, repoPath)
	case "swebench-h2h":
		return runSWEBenchH2HCmd(cmd, repoPath)
	case "swebench-live":
		return runSWEBenchLiveCmd(cmd, repoPath)
	case "swebench-eval":
		return runSWEBenchEvalCmd(cmd)
	case "report":
		return runReportCmd(cmd, repoPath)
	case "all":
		return runAllStructural(cmd, repoPath)
	default:
		return fmt.Errorf("unknown level %q (use 'proxy', 'agent', 'token', 'determinism', 'isomorphism', 'formal', 'eval', 'consistency', 'swebench', 'swebench-h2h', 'swebench-live', 'swebench-eval', 'mutate', 'h2h', 'h2h-tokens', or 'all')", level)
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

	buildResult := builder.RunBuild(
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

	buildResult := builder.RunBuild(
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

	tasksPath := resolveSWETasksPath(cmd, repoPath)
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

// resolveFixturesDir returns the --fixtures flag value if set, otherwise the given default.
func resolveFixturesDir(cmd *cobra.Command, defaultDir string) string {
	dir, _ := cmd.Flags().GetString("fixtures")
	if dir != "" {
		return dir
	}
	return defaultDir
}

// resolveSWETasksPath picks the SWE-bench JSONL dataset based on --swe-size flag.
func resolveSWETasksPath(cmd *cobra.Command, repoPath string) string {
	size, _ := cmd.Flags().GetString("swe-size")
	base := filepath.Join(repoPath, "internal/bench/testdata")
	switch size {
	case "quick5":
		return filepath.Join(base, "swebench_quick5.jsonl")
	case "sample":
		return filepath.Join(base, "swebench_sample.jsonl")
	case "lite":
		return filepath.Join(base, "swebench_lite_30.jsonl")
	case "live":
		return filepath.Join(base, "swebench_live.jsonl")
	default: // "verified" or empty
		return filepath.Join(base, "swebench_verified_500.jsonl")
	}
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

func runSWEBenchH2HCmd(cmd *cobra.Command, repoPath string) error {
	provider, err := resolveLLMProvider(cmd)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	bold := color.New(color.Bold)
	bold.Fprintf(out, "SWE-bench Head-to-Head Benchmark (live pipeline)\n\n")

	tasksPath, _ := cmd.Flags().GetString("tasks")
	if tasksPath == "" {
		tasksPath = resolveSWETasksPath(cmd, repoPath)
	}

	// Phase 1: Pipeline each hand-written SWE-bench agent → import → build
	fixturesDir := resolveFixturesDir(cmd, filepath.Join(repoPath, "internal/bench/fixtures/swebench-agents"))
	agentFiles, err := filepath.Glob(filepath.Join(fixturesDir, "*.md"))
	if err != nil {
		return fmt.Errorf("glob swe agent files: %w", err)
	}
	if len(agentFiles) == 0 {
		return fmt.Errorf("no agent .md files found in %s", fixturesDir)
	}

	var allContestants []bench.SWEContestant
	for _, agentPath := range agentFiles {
		name := strings.TrimSuffix(filepath.Base(agentPath), ".md")
		fmt.Fprintf(out, "Pipeline: %s ...", name)

		pr, pErr := bench.RunPipeline(agentPath, provider)
		if pErr != nil {
			fmt.Fprintf(out, " FAIL (%v)\n", pErr)
			continue
		}
		fmt.Fprintf(out, " ok (import %s, build %s) → %dw / %dw / %dw\n",
			pr.ImportTime.Round(time.Millisecond), pr.BuildTime.Round(time.Millisecond),
			pr.HandWritten.Words, pr.Standard.Words, pr.Compact.Words)

		allContestants = append(allContestants,
			bench.SWEContestant{Name: pr.HandWritten.Name, Prompt: pr.HandWritten.Prompt, Words: pr.HandWritten.Words},
			bench.SWEContestant{Name: pr.Standard.Name, Prompt: pr.Standard.Prompt, Words: pr.Standard.Words},
			bench.SWEContestant{Name: pr.Compact.Name, Prompt: pr.Compact.Prompt, Words: pr.Compact.Words},
		)
	}

	if len(allContestants) == 0 {
		return fmt.Errorf("no contestants produced — all pipelines failed")
	}

	// Phase 2: SWE-bench evaluation
	// Detect if the real SWE-bench harness is available
	harnessBin := "/tmp/swebench-env/bin/python"
	useHarness := false
	if _, statErr := os.Stat(harnessBin); statErr == nil {
		// Verify swebench is importable
		checkCmd := exec.Command(harnessBin, "-c", "import swebench")
		if checkCmd.Run() == nil {
			useHarness = true
		}
	}

	// Pick dataset name for harness based on --swe-size
	sweSize, _ := cmd.Flags().GetString("swe-size")
	datasetName := "princeton-nlp/SWE-bench_Verified"
	switch sweSize {
	case "lite":
		datasetName = "princeton-nlp/SWE-bench_Lite"
	case "live":
		datasetName = "SWE-bench-Live/SWE-bench-Live"
	}

	if useHarness {
		bold.Fprintf(out, "Harness: swebench (Docker + real tests) ✓\n\n")
	} else {
		fmt.Fprintf(out, "Harness: git apply --check only (install swebench for real resolve rate)\n\n")
	}

	fmt.Fprintf(out, "%-35s %8s %8s %8s %10s %6s\n",
		"Contestant", "Patched", "Applied", "Resolved", "Efficiency", "Words")
	fmt.Fprintf(out, "%s\n", strings.Repeat("-", 83))

	progress := func(completed, total int, c bench.SWEContestantResult) {
		applyColor := color.New(color.FgGreen)
		if c.ApplyRate < 30 {
			applyColor = color.New(color.FgYellow)
		}
		if c.ApplyRate < 10 {
			applyColor = color.New(color.FgRed)
		}
		resolveStr := "—"
		if c.ResolveRate > 0 {
			resolveStr = fmt.Sprintf("%.0f%%", c.ResolveRate)
		}
		fmt.Fprintf(out, "%-35s %7.0f%% ", c.Name, c.PatchRate)
		applyColor.Fprintf(out, "%7.0f%%", c.ApplyRate)
		fmt.Fprintf(out, " %8s %10.1f %6d  [%d/%d]\n",
			resolveStr, c.Efficiency, c.Words, completed, total)
	}

	var result *bench.SWEH2HResult
	if useHarness {
		cfg := bench.SWEHarnessConfig{
			PythonBin:   harnessBin,
			DatasetName: datasetName,
			MaxWorkers:  4,
		}
		result, err = bench.RunSWEBenchH2HWithHarness(allContestants, tasksPath, provider, cfg, progress)
	} else {
		result, err = bench.RunSWEBenchH2HWithContestants(allContestants, tasksPath, provider, progress)
	}
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "\n")
	bold.Fprintf(out, "Task Details:\n")
	for _, c := range result.Contestants {
		fmt.Fprintf(out, "\n  %s:\n", c.Name)
		for _, d := range c.Details {
			status := "FAIL"
			if d.TestPassed {
				status = "RSLV"
			} else if d.Applied {
				status = "APPL"
			} else if d.Patch != "" {
				status = "NOAP"
			}
			errInfo := ""
			if d.Error != "" && len(d.Error) > 60 {
				errInfo = " (" + d.Error[:60] + "...)"
			} else if d.Error != "" {
				errInfo = " (" + d.Error + ")"
			}
			fmt.Fprintf(out, "    [%s] %-40s%s\n", status, d.InstanceID, errInfo)
		}
	}

	return nil
}

func runSWEBenchLiveCmd(cmd *cobra.Command, repoPath string) error {
	provider, err := resolveLLMProvider(cmd)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	bold := color.New(color.Bold)
	bold.Fprintf(out, "SWE-bench Live Benchmark (Claude Code runtime)\n\n")

	// Check Claude CLI is available
	cfg := bench.DefaultClaudeRunConfig()
	if err := bench.CheckClaudeInstalled(cfg.ClaudeBin); err != nil {
		return fmt.Errorf("claude CLI not available: %w", err)
	}

	// Configure model from flags
	modelName, _ := cmd.Flags().GetString("model")
	cfg.Model = modelName

	// Load SWE-bench tasks
	tasksPath := resolveSWETasksPath(cmd, repoPath)
	tasks, err := bench.LoadSWEBenchTasks(tasksPath)
	if err != nil {
		return fmt.Errorf("load tasks: %w", err)
	}

	// Apply size limit
	sweSize, _ := cmd.Flags().GetString("swe-size")
	switch sweSize {
	case "quick5":
		if len(tasks) > 5 {
			tasks = tasks[:5]
		}
	case "sample":
		if len(tasks) > 10 {
			tasks = tasks[:10]
		}
	}

	fmt.Fprintf(out, "Tasks: %d (from %s)\n", len(tasks), filepath.Base(tasksPath))
	fmt.Fprintf(out, "Model: %s\n", cfg.Model)
	fmt.Fprintf(out, "Budget: $%.2f per task\n\n", cfg.MaxBudgetUSD)

	// Parse --only filter early to skip unnecessary work
	onlyFlag, _ := cmd.Flags().GetString("only")
	var allowedSet map[string]bool
	if onlyFlag != "" {
		allowedSet = make(map[string]bool)
		for _, name := range strings.Split(onlyFlag, ",") {
			allowedSet[strings.TrimSpace(name)] = true
		}
	}
	wantContestant := func(name string) bool {
		return allowedSet == nil || allowedSet[name]
	}

	// Phase 1: Load Epoch v2.0 scaffold (baseline)
	fixturesDir := resolveFixturesDir(cmd, filepath.Join(repoPath, "internal/bench/fixtures/swebench-agents"))
	scaffoldPath := filepath.Join(fixturesDir, "epoch-v2-scaffold.md")
	if _, statErr := os.Stat(scaffoldPath); statErr != nil {
		return fmt.Errorf("epoch-v2-scaffold.md not found at %s", scaffoldPath)
	}

	scaffoldData, err := os.ReadFile(scaffoldPath)
	if err != nil {
		return fmt.Errorf("read scaffold: %w", err)
	}
	scaffoldPrompt := strings.TrimSpace(string(scaffoldData))
	scaffoldWords := len(strings.Fields(scaffoldPrompt))

	var contestants []bench.ClaudeContestant

	if wantContestant("epoch-v2-original") {
		contestants = append(contestants, bench.ClaudeContestant{Name: "epoch-v2-original", Prompt: scaffoldPrompt, Words: scaffoldWords})
		fmt.Fprintf(out, "  Epoch v2 scaffold: %d words\n", scaffoldWords)
	}

	// Phase 1a: Import via Forgent pipeline (only if imported contestants are wanted)
	needImport := wantContestant("forgent-imported") || wantContestant("forgent-imported-compact")
	if needImport {
		fmt.Fprintf(out, "Phase 1a: Import Epoch v2 scaffold via Forgent pipeline...\n")
		pr, pErr := bench.RunPipeline(scaffoldPath, provider)
		if pErr != nil {
			return fmt.Errorf("pipeline failed: %w", pErr)
		}
		fmt.Fprintf(out, "  Import: %s, Build: %s\n", pr.ImportTime.Round(time.Millisecond), pr.BuildTime.Round(time.Millisecond))
		if wantContestant("forgent-imported") {
			fmt.Fprintf(out, "  Forgent imported:  %d words\n", pr.Standard.Words)
			contestants = append(contestants, bench.ClaudeContestant{Name: "forgent-imported", Prompt: pr.Standard.Prompt, Words: pr.Standard.Words})
		}
		if wantContestant("forgent-imported-compact") {
			fmt.Fprintf(out, "  Forgent imported-compact: %d words\n", pr.Compact.Words)
			contestants = append(contestants, bench.ClaudeContestant{Name: "forgent-imported-compact", Prompt: pr.Compact.Prompt, Words: pr.Compact.Words})
		}
	}

	// Phase 1b: Build hand-crafted skills (deterministic, no LLM)
	needHandcrafted := wantContestant("handcrafted") || wantContestant("handcrafted-compact")
	handcraftedDir := filepath.Join(filepath.Dir(fixturesDir), "handcrafted-swe")
	handSkillsDir := filepath.Join(handcraftedDir, "skills")
	handAgentsDir := filepath.Join(handcraftedDir, "agents")
	if needHandcrafted {
		if _, statErr := os.Stat(handSkillsDir); statErr != nil {
			return fmt.Errorf("handcrafted skills not found at %s", handSkillsDir)
		}
		fmt.Fprintf(out, "Phase 1b: Build hand-crafted skills...\n")
		for _, compact := range []bool{false, true} {
			name := "handcrafted"
			if compact {
				name = "handcrafted-compact"
			}
			if !wantContestant(name) {
				continue
			}

			tmpOut, tErr := os.MkdirTemp("", "forgent-handcrafted-*")
			if tErr != nil {
				continue
			}
			defer os.RemoveAll(tmpOut)

			br := builder.RunBuildWithOptions(handSkillsDir, handAgentsDir, tmpOut, "claude", scanner.EnrichNone, compact)
			if !br.Success {
				fmt.Fprintf(out, "  Warning: handcrafted build failed: %s\n", br.Error)
				continue
			}

			// Read generated agent prompt
			agentFiles, _ := filepath.Glob(filepath.Join(tmpOut, "agents", "*.md"))
			if len(agentFiles) == 0 {
				continue
			}
			agentData, rErr := os.ReadFile(agentFiles[0])
			if rErr != nil {
				continue
			}
			prompt := strings.TrimSpace(string(agentData))
			words := len(strings.Fields(prompt))
			fmt.Fprintf(out, "  %s: %d words\n", name, words)
			contestants = append(contestants, bench.ClaudeContestant{Name: name, Prompt: prompt, Words: words})
		}
	}
	fmt.Fprintf(out, "\n")

	if len(contestants) == 0 {
		return fmt.Errorf("no contestants selected (check --only flag)")
	}

	// Phase 2: Run comparison via Claude Code
	bold.Fprintf(out, "Phase 2: Running Claude Code comparison (%d contestants x %d tasks)...\n\n", len(contestants), len(tasks))

	results, err := bench.RunClaudeComparison(
		contestants, tasks, cfg,
		func(label string, r bench.ClaudeRunContestantResult) {
			fmt.Fprintf(out, "  %s: %d/%d patched, %d/%d files match gold (overlap %.0f%%)\n",
				r.Name, r.Patched, r.Tasks, r.FilesMatch, r.Tasks, r.AvgOverlap)
		},
	)
	if err != nil {
		return err
	}

	// Phase 3: Results table
	fmt.Fprintf(out, "\n")
	bold.Fprintf(out, "Results:\n")
	fmt.Fprintf(out, "%-25s %6s %8s %10s %9s %6s\n", "System", "Tasks", "Patched", "FilesMatch", "Overlap", "Words")
	fmt.Fprintf(out, "%s\n", strings.Repeat("-", 70))
	for _, r := range results {
		fmt.Fprintf(out, "%-25s %6d %7.0f%% %9.0f%% %8.0f%% %6d\n",
			r.Name, r.Tasks, r.PatchRate, r.MatchRate, r.AvgOverlap, r.Words)
	}

	// Compare each Forgent variant against baseline (first contestant)
	if len(results) >= 2 {
		baseline := results[0]
		fmt.Fprintf(out, "\n")
		for _, r := range results[1:] {
			diff := r.MatchRate - baseline.MatchRate
			label := fmt.Sprintf("%s vs %s:", r.Name, baseline.Name)
			if diff > 0 {
				color.New(color.FgGreen, color.Bold).Fprintf(out, "%s +%.1f%% file match rate\n", label, diff)
			} else if diff < 0 {
				color.New(color.FgRed).Fprintf(out, "%s %.1f%% file match rate\n", label, diff)
			} else {
				fmt.Fprintf(out, "%s same file match rate\n", label)
			}
		}

		// Detail per task (all contestants)
		fmt.Fprintf(out, "\nTask Details:\n")
		// Header
		fmt.Fprintf(out, "  %-40s", "Instance")
		for _, r := range results {
			fmt.Fprintf(out, " %-18s", r.Name)
		}
		fmt.Fprintf(out, "\n  %s\n", strings.Repeat("-", 40+19*len(results)))
		// Rows
		for i := range tasks {
			fmt.Fprintf(out, "  %-40s", tasks[i].InstanceID)
			for _, r := range results {
				fmt.Fprintf(out, " %-18s", taskLabel(r.Details, i))
			}
			fmt.Fprintf(out, "\n")
			// Errors
			for _, r := range results {
				if i < len(r.Details) && r.Details[i].Error != "" && !r.Details[i].Applied {
					fmt.Fprintf(out, "    %s: %s\n", r.Name, truncateBenchStr(r.Details[i].Error, 90))
				}
			}
		}

		fmt.Fprintf(out, "\n  Legend: EXACT=identical to gold, MATCH=right files, PARTIAL=some files, WRONG=wrong files, FAIL=no patch\n")
	}

	// Phase 4: Export predictions.jsonl for each contestant
	fmt.Fprintf(out, "\n")
	for _, r := range results {
		predFile := fmt.Sprintf("predictions-%s.jsonl", strings.ReplaceAll(r.Name, "/", "_"))
		if err := bench.WriteClaudePredictions(predFile, r.Name, r.Details); err != nil {
			fmt.Fprintf(out, "Warning: failed to write %s: %v\n", predFile, err)
			continue
		}
		bold.Fprintf(out, "Predictions: %s\n", predFile)
	}

	// Phase 5: Optionally run SWE-bench harness
	runHarness, _ := cmd.Flags().GetBool("harness")
	if runHarness {
		fmt.Fprintf(out, "\n")
		bold.Fprintf(out, "Phase 5: Running SWE-bench harness...\n")

		harnessCfg := bench.DefaultHarnessConfig()
		for _, r := range results {
			predFile := fmt.Sprintf("predictions-%s.jsonl", strings.ReplaceAll(r.Name, "/", "_"))
			runID := strings.ReplaceAll(r.Name, "/", "_")
			fmt.Fprintf(out, "  Evaluating %s...", r.Name)

			resolved, hErr := bench.RunHarnessEvaluation(harnessCfg, predFile, runID)
			if hErr != nil {
				fmt.Fprintf(out, " FAIL (%v)\n", hErr)
				continue
			}
			resolveRate := float64(len(resolved)) / float64(r.Tasks) * 100
			fmt.Fprintf(out, " %d/%d resolved (%.1f%%)\n", len(resolved), r.Tasks, resolveRate)
		}
	} else {
		fmt.Fprintf(out, "\nTip: re-evaluate with harness: forgent bench . --level swebench-eval --predictions predictions-<name>.jsonl\n")
	}

	return nil
}

func runSWEBenchEvalCmd(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	bold := color.New(color.Bold)

	predictionsPath, _ := cmd.Flags().GetString("predictions")
	if predictionsPath == "" {
		return fmt.Errorf("--predictions flag is required for swebench-eval level")
	}
	if _, err := os.Stat(predictionsPath); err != nil {
		return fmt.Errorf("predictions file not found: %s", predictionsPath)
	}

	bold.Fprintf(out, "SWE-bench Harness Evaluation\n\n")
	fmt.Fprintf(out, "Predictions: %s\n", predictionsPath)

	harnessCfg := bench.DefaultHarnessConfig()
	sweSize, _ := cmd.Flags().GetString("swe-size")
	switch sweSize {
	case "lite":
		harnessCfg.DatasetName = "princeton-nlp/SWE-bench_Lite"
	case "live":
		harnessCfg.DatasetName = "princeton-nlp/SWE-bench"
	default:
		harnessCfg.DatasetName = "princeton-nlp/SWE-bench_Verified"
	}
	fmt.Fprintf(out, "Dataset: %s\n\n", harnessCfg.DatasetName)

	runID := "forgent-eval"
	fmt.Fprintf(out, "Running harness (this may take a while)...\n")
	resolved, err := bench.RunHarnessEvaluation(harnessCfg, predictionsPath, runID)
	if err != nil {
		return fmt.Errorf("harness evaluation failed: %w", err)
	}

	bold.Fprintf(out, "\nResults: %d instances resolved\n", len(resolved))
	if len(resolved) > 0 {
		fmt.Fprintf(out, "\nResolved instances:\n")
		for id := range resolved {
			fmt.Fprintf(out, "  %s\n", id)
		}
	}

	return nil
}

func runH2HBench(cmd *cobra.Command, repoPath string) error {
	provider, err := resolveLLMProvider(cmd)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	bold := color.New(color.Bold)
	bold.Fprintf(out, "Head-to-Head Benchmark\n\n")

	// Load contestants: baselines + hand-written + Forgent-built agents/skills
	fixturesDir := resolveFixturesDir(cmd, filepath.Join(repoPath, "internal/bench/fixtures/contestants"))

	handWritten, err := bench.LoadContestants(fixturesDir)
	if err != nil {
		return fmt.Errorf("load hand-written contestants: %w", err)
	}

	forgentBuilt, err := bench.LoadForgentBuiltContestants(repoPath)
	if err != nil {
		return fmt.Errorf("load forgent-built contestants: %w", err)
	}

	// Load superpower code-reviewer if available
	homeDir, _ := os.UserHomeDir()
	superpowerPath := filepath.Join(homeDir, ".claude", "plugins", "cache",
		"claude-plugins-official", "superpowers", "4.3.1", "agents", "code-reviewer.md")
	if sp, err := bench.LoadContestantFile("superpower/code-reviewer", superpowerPath); err == nil {
		forgentBuilt = append(forgentBuilt, sp)
	}

	// Load compiled runtime binaries from .forgent/
	runtimeContestants, _ := bench.LoadRuntimeContestants(repoPath)

	allContestants := bench.Baselines()
	allContestants = append(allContestants, handWritten...)
	allContestants = append(allContestants, forgentBuilt...)
	allContestants = append(allContestants, runtimeContestants...)

	// Apply --only filter if set
	if onlyFlag, _ := cmd.Flags().GetString("only"); onlyFlag != "" {
		allowed := make(map[string]bool)
		for _, name := range strings.Split(onlyFlag, ",") {
			allowed[strings.TrimSpace(name)] = true
		}
		var filtered []bench.H2HContestant
		for _, c := range allContestants {
			if allowed[c.Name] {
				filtered = append(filtered, c)
			}
		}
		allContestants = filtered
	}

	if len(allContestants) == 0 {
		return fmt.Errorf("no contestants found (check --only flag or %s)", fixturesDir)
	}

	fmt.Fprintf(out, "Loaded %d contestants (%d hand-written, %d forgent-built, %d runtime, %d baselines)\n",
		len(allContestants), len(handWritten), len(forgentBuilt), len(runtimeContestants), len(bench.Baselines()))

	// Load tasks from Martian golden comments
	martianDir, _ := cmd.Flags().GetString("martian")
	tasks, err := bench.MartianTasks(filepath.Join(repoPath, martianDir))
	if err != nil {
		return fmt.Errorf("load martian tasks: %w", err)
	}
	totalCriteria := 0
	for _, t := range tasks {
		totalCriteria += len(t.Criteria)
	}
	fmt.Fprintf(out, "Loaded %d Martian tasks (%d criteria)\n\n", len(tasks), totalCriteria)

	// Phase 2: H2H evaluation
	reviewPasses, _ := cmd.Flags().GetInt("review-passes")
	if reviewPasses < 1 {
		reviewPasses = 1
	}
	if reviewPasses > 1 {
		fmt.Fprintf(out, "Review passes: %d\n", reviewPasses)
	}

	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "%-35s %7s %5s %5s %5s %5s %5s %6s %7s %6s\n",
		"Contestant", "Score", "Crit", "High", "Med", "Low", "Tasks", "Turns", "Cost", "Sc/$")
	fmt.Fprintf(out, "%s\n", strings.Repeat("-", 96))

	progress := func(completed, total int, c bench.H2HContestantResult) {
		scoreColor := color.New(color.FgGreen)
		if c.AvgScore < 70 {
			scoreColor = color.New(color.FgYellow)
		}
		if c.AvgScore < 50 {
			scoreColor = color.New(color.FgRed)
		}
		fmt.Fprintf(out, "%-35s ", c.Name)
		scoreColor.Fprintf(out, "%6.1f%%", c.AvgScore)
		fmt.Fprintf(out, " %4.0f%% %4.0f%% %4.0f%% %4.0f%%    %2d %5d $%5.3f %6.1f  [%d/%d]\n",
			c.Severity.Rate("Critical"), c.Severity.Rate("High"),
			c.Severity.Rate("Medium"), c.Severity.Rate("Low"),
			c.Tasks, c.LLMCalls, c.EstCost, c.CostScore, completed, total)
	}

	result, err := bench.RunH2HWithTasks(allContestants, tasks, provider, reviewPasses, progress)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "\n")

	// Show task-level details
	bold.Fprintf(out, "Task Details:\n")
	for _, c := range result.Contestants {
		fmt.Fprintf(out, "\n  %s:\n", c.Name)
		for _, d := range c.Details {
			fmt.Fprintf(out, "    %-25s  score=%.0f%%  %s\n",
				d.TaskID, d.Score, d.Reasoning)
		}
	}

	// Persist to JSON if --output is set
	if outputPath, _ := cmd.Flags().GetString("output"); outputPath != "" {
		if err := appendH2HReport(outputPath, result); err != nil {
			fmt.Fprintf(out, "\nWarning: failed to write report to %s: %v\n", outputPath, err)
		} else {
			fmt.Fprintf(out, "\nReport written to %s\n", outputPath)
		}
	}

	return nil
}

// H2HReport is the persisted format for h2h bench results.
type H2HReport struct {
	RunAt       time.Time                   `json:"run_at"`
	Contestants []bench.H2HContestantResult `json:"contestants"`
}

// appendH2HReport merges new results into an existing JSON report file (or creates it).
func appendH2HReport(path string, result *bench.H2HResult) error {
	var report H2HReport
	if data, err := os.ReadFile(path); err == nil {
		_ = json.Unmarshal(data, &report)
	}

	// Upsert contestants by name
	byName := make(map[string]int, len(report.Contestants))
	for i, c := range report.Contestants {
		byName[c.Name] = i
	}
	for _, c := range result.Contestants {
		if idx, ok := byName[c.Name]; ok {
			report.Contestants[idx] = c
		} else {
			report.Contestants = append(report.Contestants, c)
		}
	}
	report.RunAt = time.Now()

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func runH2HTokensBench(cmd *cobra.Command, repoPath string) error {
	provider, err := resolveLLMProvider(cmd)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	bold := color.New(color.Bold)
	bold.Fprintf(out, "Head-to-Head Token Efficiency Benchmark (live pipeline)\n\n")

	// Pipeline each hand-written agent → import → build
	fixturesDir := resolveFixturesDir(cmd, filepath.Join(repoPath, "internal/bench/fixtures/contestants"))
	agentFiles, err := filepath.Glob(filepath.Join(fixturesDir, "*.md"))
	if err != nil {
		return fmt.Errorf("glob agent files: %w", err)
	}
	if len(agentFiles) == 0 {
		return fmt.Errorf("no agent .md files found in %s", fixturesDir)
	}

	type sourceComparison struct {
		Source      string
		HandWritten int
		Standard    int
		Compact     int
	}

	var comparisons []sourceComparison
	for _, agentPath := range agentFiles {
		name := strings.TrimSuffix(filepath.Base(agentPath), ".md")
		fmt.Fprintf(out, "Pipeline: %s ...", name)

		pr, err := bench.RunPipeline(agentPath, provider)
		if err != nil {
			fmt.Fprintf(out, " FAIL (%v)\n", err)
			continue
		}
		fmt.Fprintf(out, " ok (%dw → std:%dw, compact:%dw)\n",
			pr.HandWritten.Words, pr.Standard.Words, pr.Compact.Words)

		comparisons = append(comparisons, sourceComparison{
			Source:      pr.Source,
			HandWritten: pr.HandWritten.Words,
			Standard:    pr.Standard.Words,
			Compact:     pr.Compact.Words,
		})
	}

	if len(comparisons) == 0 {
		return fmt.Errorf("no comparisons produced — all pipelines failed")
	}

	// Print results table
	fmt.Fprintf(out, "\n")
	fmt.Fprintf(out, "%-30s %12s %12s %12s %10s %10s\n",
		"Source", "Hand-Written", "Standard", "Compact", "Std Δ", "Cpt Δ")
	fmt.Fprintf(out, "%s\n", strings.Repeat("-", 90))

	totalHW, totalStd, totalCompact := 0, 0, 0
	for _, src := range comparisons {
		stdDelta := float64(src.Standard-src.HandWritten) / float64(src.HandWritten) * 100
		cptDelta := float64(src.Compact-src.HandWritten) / float64(src.HandWritten) * 100

		stdColor := color.New(color.FgGreen)
		if stdDelta > 50 {
			stdColor = color.New(color.FgYellow)
		}
		if stdDelta > 100 {
			stdColor = color.New(color.FgRed)
		}
		cptColor := color.New(color.FgGreen)
		if cptDelta > 50 {
			cptColor = color.New(color.FgYellow)
		}

		fmt.Fprintf(out, "%-30s %10d w %10d w %10d w ",
			src.Source, src.HandWritten, src.Standard, src.Compact)
		stdColor.Fprintf(out, "%+8.0f%%", stdDelta)
		fmt.Fprintf(out, "  ")
		cptColor.Fprintf(out, "%+8.0f%%", cptDelta)
		fmt.Fprintln(out)

		totalHW += src.HandWritten
		totalStd += src.Standard
		totalCompact += src.Compact
	}

	fmt.Fprintf(out, "%s\n", strings.Repeat("-", 90))
	avgStdDelta := float64(totalStd-totalHW) / float64(totalHW) * 100
	avgCptDelta := float64(totalCompact-totalHW) / float64(totalHW) * 100

	totalColor := color.New(color.Bold)
	fmt.Fprintf(out, "%-30s %10d w %10d w %10d w ", "TOTAL", totalHW, totalStd, totalCompact)
	totalColor.Fprintf(out, "%+8.0f%%  %+8.0f%%\n", avgStdDelta, avgCptDelta)

	return nil
}

// readAllFiles concatenates all files in a directory tree.

func runReportCmd(cmd *cobra.Command, repoPath string) error {
	provider, err := resolveLLMProvider(cmd)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	bold := color.New(color.Bold)
	bold.Fprintf(out, "Generating Full Benchmark Report\n\n")

	// Phase 1: Pipeline all agents
	fixturesDir := resolveFixturesDir(cmd, filepath.Join(repoPath, "internal/bench/fixtures/swebench-agents"))
	agentFiles, err := filepath.Glob(filepath.Join(fixturesDir, "*.md"))
	if err != nil {
		return fmt.Errorf("glob agent files: %w", err)
	}
	if len(agentFiles) == 0 {
		return fmt.Errorf("no agent .md files found in %s", fixturesDir)
	}

	var allContestants []bench.SWEContestant
	var agentNames []string
	for _, agentPath := range agentFiles {
		name := strings.TrimSuffix(filepath.Base(agentPath), ".md")
		fmt.Fprintf(out, "Pipeline: %s ...", name)

		pr, pErr := bench.RunPipeline(agentPath, provider)
		if pErr != nil {
			fmt.Fprintf(out, " FAIL (%v)\n", pErr)
			continue
		}
		fmt.Fprintf(out, " ok → %dw / %dw / %dw\n",
			pr.HandWritten.Words, pr.Standard.Words, pr.Compact.Words)
		agentNames = append(agentNames, name)

		allContestants = append(allContestants,
			bench.SWEContestant{Name: pr.HandWritten.Name, Prompt: pr.HandWritten.Prompt, Words: pr.HandWritten.Words},
			bench.SWEContestant{Name: pr.Standard.Name, Prompt: pr.Standard.Prompt, Words: pr.Standard.Words},
			bench.SWEContestant{Name: pr.Compact.Name, Prompt: pr.Compact.Prompt, Words: pr.Compact.Words},
		)
	}

	if len(allContestants) == 0 {
		return fmt.Errorf("no contestants produced — all pipelines failed")
	}

	report := bench.BenchReport{
		Timestamp: time.Now(),
		Agents:    agentNames,
		Errors:    make(map[string]string),
	}

	// Phase 2: Run SWE-bench
	fmt.Fprintf(out, "\nRunning SWE-bench...")
	tasksPath := resolveSWETasksPath(cmd, repoPath)
	sweResult, sweErr := bench.RunSWEBenchH2HWithContestants(allContestants, tasksPath, provider, nil)
	if sweErr != nil {
		fmt.Fprintf(out, " FAIL (%v)\n", sweErr)
		report.Errors["swebench"] = sweErr.Error()
	} else {
		fmt.Fprintf(out, " done (%d tasks)\n", sweResult.Tasks)
		report.SWEBench = sweResult
	}

	// Phase 3: Generate report
	md := bench.GenerateReport(report)
	reportPath := fmt.Sprintf("bench-report-%s.md", time.Now().Format("2006-01-02"))
	if err := os.WriteFile(reportPath, []byte(md), 0644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	fmt.Fprintf(out, "\n")
	bold.Fprintf(out, "Report written to %s\n", reportPath)
	fmt.Fprintf(out, "\n%s", md)

	return nil
}

func taskLabel(details []bench.ClaudeRunResult, i int) string {
	if i >= len(details) {
		return "FAIL"
	}
	d := details[i]
	if !d.Applied {
		return "FAIL"
	}
	if d.ExactMatch {
		return "EXACT (100%)"
	}
	if d.FilesMatch {
		return fmt.Sprintf("MATCH (%.0f%%)", d.FileOverlap*100)
	}
	if d.FileOverlap > 0 {
		return fmt.Sprintf("PARTIAL (%.0f%%)", d.FileOverlap*100)
	}
	return "WRONG (0%)"
}

func truncateBenchStr(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

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
