package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/mirandaguillaume/forgent/internal/bench"
	"github.com/mirandaguillaume/forgent/internal/bench/experiment"
	"github.com/mirandaguillaume/forgent/internal/bench/formal"
	"github.com/mirandaguillaume/forgent/internal/doctor/registry"
	"github.com/mirandaguillaume/forgent/internal/llm"
	yamlloader "github.com/mirandaguillaume/forgent/internal/yaml"
	"github.com/mirandaguillaume/forgent/pkg/model"
	"github.com/spf13/cobra"
)

func init() {
	convertBaselineCmd := &cobra.Command{
		Use:   "convert-baseline <text.json> <runtime.json> <out.json>",
		Short: "Convert legacy experiment JSON files to a BaselineSnapshot for --baseline",
		Long: `Reads pre-Epic-10 experiment result JSON files ([]TaskResultJSON) and produces
a BaselineSnapshot compatible with forgent-bench bench --baseline.

Because legacy files predate NodeViolations telemetry, ContractComplianceRate is
proxied from GuardrailScore (fraction of guardrails respected). Step-level rates
are derived from step completion fractions across runs.`,
		Args: cobra.ExactArgs(3),
		RunE: runConvertBaseline,
	}
	rootCmd.AddCommand(convertBaselineCmd)

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
	benchCmd.Flags().String("corpus", "", "Corpus directory for InstructionBench (instruction level)")
	benchCmd.Flags().String("golden", "", "Golden set YAML for false positive measurement (instruction level)")
	benchCmd.Flags().String("tasks-dir", "", "Task definitions directory (experiment level)")
	benchCmd.Flags().String("task", "", "Single task YAML workflow template (experiment level, use with --experiment-corpus)")
	benchCmd.Flags().String("experiment-corpus", "", "Corpus directory to instantiate the --task template for each entry (experiment level)")
	benchCmd.Flags().Int("runs", 1, "Runs per task for experiment level")
	benchCmd.Flags().String("resume", "", "Resume from checkpoint file (empty with --output = auto-detect)")
	benchCmd.Flags().String("retriever", "none", "Context retriever strategy for experiment level: none|prefix|ranked")
	benchCmd.Flags().String("conditions", "", "Comma-separated conditions for swebench-experiment: text,runtime (default: text,runtime)")
	benchCmd.Flags().Bool("agent", false, "Enable agent condition (multi-turn sequential, one LLM call per step, no DAG enforcement)")
	benchCmd.Flags().String("baseline", "", "Path to a baseline JSON snapshot (saved by --save-baseline) for per-node delta computation. Overrides BENCH_BASELINE env var.")
	benchCmd.Flags().String("save-baseline", "", "Save per-node compliance metrics to this path after the run (for future --baseline comparisons).")
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
	case "mutate":
		return runMutateBench(cmd, repoPath)
	case "h2h":
		return runH2HBench(cmd, repoPath)
	case "h2h-tokens":
		return runH2HTokensBench(cmd, repoPath)
	case "swebench-experiment":
		return runSWEBenchExperimentCmd(cmd, repoPath)
	case "instruction":
		return runInstructionBench(cmd)
	case "experiment":
		return runExperimentBench(cmd, repoPath)
	case "all":
		return runAllStructural(cmd, repoPath)
	default:
		return fmt.Errorf("unknown level %q (use 'proxy', 'agent', 'token', 'determinism', 'isomorphism', 'formal', 'eval', 'consistency', 'swebench-experiment', 'mutate', 'h2h', 'h2h-tokens', 'instruction', 'experiment', or 'all')", level)
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
	evalTasksFile, _ := cmd.Flags().GetString("eval-tasks")
	tasksDir, _ := cmd.Flags().GetString("tasks-dir")

	if evalTasksFile == "" {
		return fmt.Errorf("--eval-tasks is required for eval level")
	}
	if tasksDir == "" {
		return fmt.Errorf("--tasks-dir is required for eval level (task template for composed run)")
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

	// Load task template — pick the first code-review task (or just the first task).
	taskDefs, err := experiment.LoadTasks(tasksDir)
	if err != nil {
		return fmt.Errorf("load task template from %s: %w", tasksDir, err)
	}
	if len(taskDefs) == 0 {
		return fmt.Errorf("no task definitions found in %s", tasksDir)
	}
	template := taskDefs[0]
	for _, td := range taskDefs {
		if td.Category == "claude-code-workflow" {
			template = td
			break
		}
	}

	// composedFn: inject eval input as pr_diff.patch and run the full pipeline.
	composedFn := func(input string) (string, error) {
		taskCopy := template
		if taskCopy.Workspace != nil {
			wsCopy := *taskCopy.Workspace
			newFiles := make([]experiment.WorkspaceFile, len(wsCopy.Files))
			copy(newFiles, wsCopy.Files)
			for i := range newFiles {
				if newFiles[i].Path == "pr_diff.patch" {
					newFiles[i].Content = input
				}
			}
			wsCopy.Files = newFiles
			taskCopy.Workspace = &wsCopy
		}
		results, err := experiment.RunRuntimeExperiment(experiment.ExperimentConfig{
			Tasks:    []experiment.TaskDefinition{taskCopy},
			Provider: provider,
		})
		if err != nil {
			return "", err
		}
		if len(results) == 0 {
			return "", fmt.Errorf("no results from experiment pipeline")
		}
		return results[0].CombinedOutput, nil
	}

	monolithicPrompt := "You are a code reviewer. Review the code for bugs, security issues, performance problems, and best practice violations. Be thorough and specific."

	result, err := bench.RunEval(tasks, composedFn, monolithicPrompt, provider)
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

// runSWEBenchExperimentCmd implements --level swebench-experiment.
// It bridges the experiment framework (text/runtime conditions via resolve-issue.yaml)
// with the SWE-bench patch generation + harness evaluation pipeline.
//
// Usage:
//
//	forgent-bench bench . --level swebench-experiment --swe-size quick5 --task <path/to/resolve-issue.yaml>
//	forgent-bench bench . --level swebench-experiment --swe-size lite --harness
//	forgent-bench bench . --level swebench-experiment --swe-size lite --resume <checkpoint.jsonl>
func runSWEBenchExperimentCmd(cmd *cobra.Command, repoPath string) error {
	provider, err := resolveLLMProvider(cmd)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	bold := color.New(color.Bold)
	bold.Fprintf(out, "SWE-bench Experiment (text vs runtime conditions)\n\n")

	// Load SWE-bench tasks from JSONL.
	tasksPath := resolveSWETasksPath(cmd, repoPath)
	sweTasks, err := bench.LoadSWETasks(tasksPath)
	if err != nil {
		return fmt.Errorf("load swebench tasks: %w", err)
	}
	fmt.Fprintf(out, "Tasks: %d (from %s)\n", len(sweTasks), filepath.Base(tasksPath))

	// Load resolve-issue.yaml as the task template.
	// Default location: internal/bench/experiment/tasks/resolve-issue.yaml.
	taskFile, _ := cmd.Flags().GetString("task")
	if taskFile == "" {
		taskFile = filepath.Join(repoPath, "internal/bench/experiment/tasks/resolve-issue.yaml")
	} else if !filepath.IsAbs(taskFile) {
		taskFile = filepath.Join(repoPath, taskFile)
	}
	taskDir := filepath.Dir(taskFile)
	allTasks, err := experiment.LoadTasks(taskDir)
	if err != nil {
		return fmt.Errorf("load task template: %w", err)
	}
	var template experiment.TaskDefinition
	var found bool
	for _, t := range allTasks {
		if t.ID == "resolve-issue" {
			template = t
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("resolve-issue.yaml not found in %s", taskDir)
	}
	fmt.Fprintf(out, "Task template: %s (%d steps, %d tool_nodes)\n",
		template.ID, len(template.Steps), len(template.ToolNodes))

	// Parse conditions from flag or default.
	conditionsStr, _ := cmd.Flags().GetString("conditions")
	var conditions []string
	if conditionsStr != "" {
		for _, c := range strings.Split(conditionsStr, ",") {
			conditions = append(conditions, strings.TrimSpace(c))
		}
	}

	// Resolve checkpoint.
	resumePath, _ := cmd.Flags().GetString("resume")
	outputPath, _ := cmd.Flags().GetString("output")
	checkpointPath := resumePath
	if checkpointPath == "" && outputPath != "" {
		ext := filepath.Ext(outputPath)
		base := outputPath
		if ext != "" {
			base = strings.TrimSuffix(outputPath, ext)
		}
		checkpointPath = base + ".checkpoint.jsonl"
	}
	if checkpointPath != "" {
		fmt.Fprintf(out, "Checkpoint: %s\n", checkpointPath)
	}

	// Harness setup.
	harnessEnabled, _ := cmd.Flags().GetBool("harness")
	var harnessCfg *bench.SWEHarnessConfig
	if harnessEnabled {
		h := bench.DefaultHarnessConfig()
		// Update dataset name based on --swe-size.
		sweSize, _ := cmd.Flags().GetString("swe-size")
		switch sweSize {
		case "lite":
			h.DatasetName = "princeton-nlp/SWE-bench_Lite"
		case "verified":
			h.DatasetName = "princeton-nlp/SWE-bench_Verified"
		case "pro":
			h.DatasetName = "ScaleAI/SWE-bench_Pro"
		}
		harnessCfg = &h
		fmt.Fprintf(out, "Harness: %s\n", h.DatasetName)
	}

	fmt.Fprintf(out, "\nRunning %d tasks", len(sweTasks))
	if len(conditions) > 0 {
		fmt.Fprintf(out, " × %s", strings.Join(conditions, "/"))
	} else {
		fmt.Fprintf(out, " × text/runtime")
	}
	fmt.Fprintf(out, "...\n\n")

	expCfg := bench.SWEBenchExperimentConfig{
		Tasks:         sweTasks,
		Template:      template,
		Provider:      provider,
		Conditions:    conditions,
		CheckpointPath: checkpointPath,
		HarnessConfig:  harnessCfg,
		StepModelOverride: resolveStepModelFactory(cmd),
		OnProgress: func(task, condition string, result bench.SWETaskResult) {
			status := "no patch"
			if result.Patch != "" {
				status = "patch"
			}
			if result.Applied {
				status = "applied"
			}
			if result.TestPassed {
				status = "resolved"
			}
			if result.Error != "" {
				status = "error: " + result.Error
			}
			fmt.Fprintf(out, "  [%s/%s] %s\n", condition, status, task)
		},
	}

	result, err := bench.RunSWEBenchExperiment(expCfg)
	if err != nil {
		return fmt.Errorf("swebench experiment: %w", err)
	}

	// Print non-fatal warnings (e.g. harness failures).
	for _, w := range result.Warnings {
		fmt.Fprintf(out, "WARN: %s\n", w)
	}

	// Print results table.
	// "Formatted" = patch != "" (format check only, not a real git apply).
	// "Resolved"  = TestPassed via official harness (requires --harness).
	fmt.Fprintf(out, "\n%-12s %8s %10s %8s\n", "Condition", "Patched", "Formatted", "Resolved")
	fmt.Fprintf(out, "%s\n", strings.Repeat("-", 44))
	finalConditions := conditions
	if len(finalConditions) == 0 {
		finalConditions = []string{"text", "runtime"}
	}
	for _, cond := range finalConditions {
		fmt.Fprintf(out, "%-12s %7.1f%% %9.1f%% %7.1f%%\n",
			cond,
			result.PatchRate(cond),
			result.ApplyRate(cond),
			result.ResolveRate(cond),
		)
	}

	// Compute and print bootstrap CI for delta between first two conditions.
	if len(finalConditions) >= 2 {
		condA, condB := finalConditions[0], finalConditions[1]
		resA := result.Results[condA]
		resB := result.Results[condB]
		if len(resA) == len(resB) && len(resA) > 0 {
			resolvedA := make([]bool, len(resA))
			resolvedB := make([]bool, len(resB))
			for i := range resA {
				if harnessEnabled {
					resolvedA[i] = resA[i].TestPassed
					resolvedB[i] = resB[i].TestPassed
				} else {
					resolvedA[i] = resA[i].Applied
					resolvedB[i] = resB[i].Applied
				}
			}
			lo, hi := experiment.BootstrapCI(resolvedA, resolvedB, 10_000)
			var deltaA, deltaB float64
			if harnessEnabled {
				deltaA = result.ResolveRate(condA)
				deltaB = result.ResolveRate(condB)
			} else {
				deltaA = result.ApplyRate(condA)
				deltaB = result.ApplyRate(condB)
			}
			delta := deltaB - deltaA
			metric := "formatted"
			if harnessEnabled {
				metric = "resolved"
			}
			fmt.Fprintf(out, "\nDelta %s (%s vs %s): %+.1f pp\n", metric, condB, condA, delta)
			fmt.Fprintf(out, "Bootstrap 95%% CI: [%+.1f%%, %+.1f%%]\n", lo*100, hi*100)
			if lo > 0 || hi < 0 {
				fmt.Fprintf(out, "Statistically significant: YES (CI does not include 0)\n")
			} else {
				fmt.Fprintf(out, "Statistically significant: NO (CI includes 0)\n")
			}
		}
	}

	// Write predictions.jsonl per condition.
	if outputPath != "" {
		for _, cond := range finalConditions {
			ext := filepath.Ext(outputPath)
			base := outputPath
			if ext != "" {
				base = strings.TrimSuffix(outputPath, ext)
			}
			predPath := base + "-predictions-" + cond + ".jsonl"
			if wErr := bench.WriteSWEPredictions(predPath, cond, result.Results[cond]); wErr != nil {
				fmt.Fprintf(out, "warn: write predictions for %s: %v\n", cond, wErr)
			} else {
				fmt.Fprintf(out, "Predictions written: %s\n", predPath)
			}
		}
	}

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

// resolveEnvKey returns the first non-empty value among the given env var names.
func resolveEnvKey(names ...string) string {
	for _, name := range names {
		if v := os.Getenv(name); v != "" {
			return v
		}
	}
	return ""
}

// resolveLLMProvider picks the LLM provider from --provider flag or env vars.
func resolveLLMProvider(cmd *cobra.Command) (llm.Provider, error) {
	providerName, _ := cmd.Flags().GetString("provider")

	// Auto-detect from env if not explicitly set
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

	modelName, _ := cmd.Flags().GetString("model")
	return llm.GetProviderWithModel(providerName, apiKey, modelName)
}

// resolveStepModelFactory returns a factory that creates providers with per-step
// model overrides. Uses the same provider name and API key as resolveLLMProvider.
func resolveStepModelFactory(cmd *cobra.Command) func(model string) (llm.Provider, error) {
	providerName, _ := cmd.Flags().GetString("provider")
	if providerName == "" {
		if resolveEnvKey("ANTHROPIC_API_KEY", "ANTHROPIC_FORGENT_API_KEY") != "" {
			providerName = "anthropic"
		} else if resolveEnvKey("OPENROUTER_API_KEY", "OPENROUTER_FORGENT_API_KEY") != "" {
			providerName = "openrouter"
		}
	}
	var apiKey string
	switch providerName {
	case "anthropic":
		apiKey = resolveEnvKey("ANTHROPIC_API_KEY", "ANTHROPIC_FORGENT_API_KEY")
	case "openrouter":
		apiKey = resolveEnvKey("OPENROUTER_API_KEY", "OPENROUTER_FORGENT_API_KEY")
	}
	if providerName == "" || apiKey == "" {
		return nil
	}
	return func(model string) (llm.Provider, error) {
		return llm.GetProviderWithModel(providerName, apiKey, model)
	}
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
		if jsonErr := json.Unmarshal(data, &report); jsonErr != nil {
			fmt.Fprintf(os.Stderr, "warn: existing H2H report at %s is corrupt, rebuilding: %v\n", path, jsonErr)
			report = H2HReport{}
		}
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

func runInstructionBench(cmd *cobra.Command) error {
	corpusDir, _ := cmd.Flags().GetString("corpus")
	if corpusDir == "" {
		return fmt.Errorf("--corpus is required for instruction level")
	}

	goldenPath, _ := cmd.Flags().GetString("golden")
	modelName, _ := cmd.Flags().GetString("model")

	var provider llm.Provider
	var err error
	provider, err = resolveLLMProvider(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: no LLM provider available (%v) — running static-only\n", err)
		provider = nil
	}

	reg, err := registry.Load("")
	if err != nil {
		return fmt.Errorf("load registry: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Running InstructionBench on %s...\n", corpusDir)
	result, err := bench.RunInstructionBench(bench.InstructionBenchOpts{
		CorpusDir:  corpusDir,
		Provider:   provider,
		GoldenPath: goldenPath,
		Registry:   reg,
		Model:      modelName,
	})
	if err != nil {
		return fmt.Errorf("instruction bench: %w", err)
	}

	report := bench.FormatInstructionReport(result, bench.InstructionReportOpts{
		Model:       modelName,
		CorpusDir:   corpusDir,
		Version:     reg.Version,
		GoldenFiles: result.GoldenFileCount,
	})

	outputPath, _ := cmd.Flags().GetString("output")
	if outputPath != "" {
		if err := os.WriteFile(outputPath, []byte(report), 0644); err != nil {
			return fmt.Errorf("write report: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Report written to %s\n", outputPath)
	} else {
		fmt.Print(report)
	}

	return nil
}

func runExperimentBench(cmd *cobra.Command, repoPath string) error {
	tasksDir, _ := cmd.Flags().GetString("tasks-dir")
	taskFile, _ := cmd.Flags().GetString("task")
	corpusDir, _ := cmd.Flags().GetString("experiment-corpus")

	// Resolve relative paths against repoPath so the binary can be run from any directory.
	if taskFile != "" && !filepath.IsAbs(taskFile) {
		taskFile = filepath.Join(repoPath, taskFile)
	}
	if corpusDir != "" && !filepath.IsAbs(corpusDir) {
		corpusDir = filepath.Join(repoPath, corpusDir)
	}
	if tasksDir != "" && !filepath.IsAbs(tasksDir) {
		tasksDir = filepath.Join(repoPath, tasksDir)
	}

	if tasksDir == "" && taskFile == "" {
		return fmt.Errorf("--tasks-dir or --task is required for experiment level")
	}
	if taskFile != "" && corpusDir == "" {
		return fmt.Errorf("--task requires --experiment-corpus")
	}

	modelName, _ := cmd.Flags().GetString("model")
	runs, _ := cmd.Flags().GetInt("runs")
	outputPath, _ := cmd.Flags().GetString("output")
	resumePath, _ := cmd.Flags().GetString("resume")
	harnessEnabled, _ := cmd.Flags().GetBool("harness")
	retrieverFlag, _ := cmd.Flags().GetString("retriever")
	agentEnabled, _ := cmd.Flags().GetBool("agent")

	// Build third-condition retriever (nil = no third condition)
	var thirdRetriever experiment.ContextRetriever
	var thirdConditionLabel string
	switch retrieverFlag {
	case "prefix":
		thirdRetriever = experiment.PrefixRetriever{MaxTokens: 300}
		thirdConditionLabel = "prefix"
	case "ranked":
		thirdRetriever = experiment.RankedRetriever{MaxTokens: 500, TopK: 5}
		thirdConditionLabel = "ranked"
	}

	provider, err := resolveLLMProvider(cmd)
	if err != nil {
		return fmt.Errorf("experiment requires an LLM provider: %w", err)
	}

	var allTasks []experiment.TaskDefinition
	if taskFile != "" {
		allTasks, err = experiment.LoadCorpus(taskFile, corpusDir)
		if err != nil {
			return fmt.Errorf("load corpus: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Loaded %d corpus entries from %s\n", len(allTasks), corpusDir)
	} else {
		allTasks, err = experiment.LoadTasks(tasksDir)
		if err != nil {
			return fmt.Errorf("load tasks: %w", err)
		}
	}

	// Filter tasks by --only flag
	onlyFlag, _ := cmd.Flags().GetString("only")
	tasks := allTasks
	if onlyFlag != "" {
		allowed := make(map[string]bool)
		for _, name := range strings.Split(onlyFlag, ",") {
			allowed[strings.TrimSpace(name)] = true
		}
		tasks = nil
		for _, t := range allTasks {
			if allowed[t.ID] {
				tasks = append(tasks, t)
			}
		}
		if len(tasks) == 0 {
			return fmt.Errorf("--only matched no tasks (available: %d)", len(allTasks))
		}
		fmt.Fprintf(os.Stderr, "Filtered to %d/%d tasks\n", len(tasks), len(allTasks))
	}

	// Resolve checkpoint path
	checkpointPath := resumePath
	if checkpointPath == "" && outputPath != "" {
		checkpointPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".checkpoint.jsonl"
	}

	// Load existing checkpoint for resume
	var existingEntries []experiment.CheckpointEntry
	var priorTextResults, priorRuntimeResults []experiment.TaskResult

	if resumePath != "" {
		if _, statErr := os.Stat(resumePath); statErr == nil {
			existingEntries, err = experiment.LoadCheckpoint(resumePath)
			if err != nil {
				return fmt.Errorf("load checkpoint: %w", err)
			}
			priorTextResults = experiment.ExtractResults(existingEntries, "text")
			priorRuntimeResults = experiment.ExtractResults(existingEntries, "runtime")
			fmt.Fprintf(os.Stderr, "Resumed from checkpoint: %d text results, %d runtime results\n",
				len(priorTextResults), len(priorRuntimeResults))
		}
	} else if checkpointPath != "" {
		// Auto-detect: try to load checkpoint alongside output
		if _, statErr := os.Stat(checkpointPath); statErr == nil {
			existingEntries, err = experiment.LoadCheckpoint(checkpointPath)
			if err != nil {
				return fmt.Errorf("load checkpoint: %w", err)
			}
			priorTextResults = experiment.ExtractResults(existingEntries, "text")
			priorRuntimeResults = experiment.ExtractResults(existingEntries, "runtime")
			fmt.Fprintf(os.Stderr, "Auto-resumed from checkpoint: %d text results, %d runtime results\n",
				len(priorTextResults), len(priorRuntimeResults))
		}
	}

	// Open checkpoint writer
	var cpWriter *experiment.CheckpointWriter
	if checkpointPath != "" {
		cpWriter, err = experiment.NewCheckpointWriter(checkpointPath)
		if err != nil {
			return fmt.Errorf("open checkpoint: %w", err)
		}
		defer cpWriter.Close()
	}

	textSkipMap := experiment.BuildSkipMap(existingEntries, "text")
	runtimeSkipMap := experiment.BuildSkipMap(existingEntries, "runtime")
	agentSkipMap := experiment.BuildSkipMap(existingEntries, "agent")
	var rankedSkipMap map[string]int
	if thirdConditionLabel != "" {
		rankedSkipMap = experiment.BuildSkipMap(existingEntries, thirdConditionLabel)
	}

	if harnessEnabled {
		fmt.Fprintf(os.Stderr, "Running experiment: %d tasks, %d runs per task (per-task pairing, harness enabled)...\n", len(tasks), runs)
	} else {
		fmt.Fprintf(os.Stderr, "Running experiment: %d tasks, %d runs per task (per-task pairing)...\n", len(tasks), runs)
	}
	if agentEnabled {
		fmt.Fprintf(os.Stderr, "Agent condition enabled: multi-turn sequential (1 LLM call/step, no DAG enforcement)\n")
	}
	if thirdRetriever != nil {
		fmt.Fprintf(os.Stderr, "Third condition: %s retriever\n", thirdConditionLabel)
	}

	var priorRankedResults []experiment.TaskResult
	if thirdConditionLabel != "" {
		priorRankedResults = experiment.ExtractResults(existingEntries, thirdConditionLabel)
	}

	// Per-task execution: text + runtime [+ agent] [+ ranked] for each task before moving to next.
	var newTextResults, newRuntimeResults, newAgentResults, newRankedResults []experiment.TaskResult
	var priorAgentResults []experiment.TaskResult
	if agentEnabled {
		priorAgentResults = experiment.ExtractResults(existingEntries, "agent")
	}

	for i, task := range tasks {
		fmt.Fprintf(os.Stderr, "\n── Task %d/%d: %s ──\n", i+1, len(tasks), task.ID)

		// Run all conditions concurrently for this task.
		var (
			textRes  []experiment.TaskResult
			rtRes    []experiment.TaskResult
			agentRes []experiment.TaskResult
			textErr  error
			rtErr    error
			agentErr error
			mu       sync.Mutex
		)
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			cfg := experiment.ExperimentConfig{
				Tasks:       []experiment.TaskDefinition{task},
				Provider:    provider,
				Model:       modelName,
				RunsPerTask: runs,
				SkipMap:     map[string]int{task.ID: textSkipMap[task.ID]},
				OnResult: func(r experiment.TaskResult, run int) {
					if cpWriter != nil {
						mu.Lock()
						defer mu.Unlock()
						if err := cpWriter.Append(experiment.CheckpointEntry{TaskID: r.TaskID, Condition: "text", Run: run, Result: experiment.ToResultJSON(r)}); err != nil {
							fmt.Fprintf(os.Stderr, "warn: checkpoint append text/%s/run%d: %v\n", r.TaskID, run, err)
						}
					}
				},
			}
			textRes, textErr = experiment.RunTextExperiment(cfg)
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			cfg := experiment.ExperimentConfig{
				Tasks:             []experiment.TaskDefinition{task},
				Provider:          provider,
				Model:             modelName,
				RunsPerTask:       runs,
				SkipMap:           map[string]int{task.ID: runtimeSkipMap[task.ID]},
				StepModelOverride: resolveStepModelFactory(cmd),
				OnResult: func(r experiment.TaskResult, run int) {
					if cpWriter != nil {
						mu.Lock()
						defer mu.Unlock()
						if err := cpWriter.Append(experiment.CheckpointEntry{TaskID: r.TaskID, Condition: "runtime", Run: run, Result: experiment.ToResultJSON(r)}); err != nil {
							fmt.Fprintf(os.Stderr, "warn: checkpoint append runtime/%s/run%d: %v\n", r.TaskID, run, err)
						}
					}
				},
			}
			rtRes, rtErr = experiment.RunRuntimeExperiment(cfg)
		}()

		if agentEnabled {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cfg := experiment.ExperimentConfig{
					Tasks:       []experiment.TaskDefinition{task},
					Provider:    provider,
					Model:       modelName,
					RunsPerTask: runs,
					SkipMap:     map[string]int{task.ID: agentSkipMap[task.ID]},
					Condition:   "agent",
					OnResult: func(r experiment.TaskResult, run int) {
						if cpWriter != nil {
							mu.Lock()
							defer mu.Unlock()
							if err := cpWriter.Append(experiment.CheckpointEntry{TaskID: r.TaskID, Condition: "agent", Run: run, Result: experiment.ToResultJSON(r)}); err != nil {
								fmt.Fprintf(os.Stderr, "warn: checkpoint append agent/%s/run%d: %v\n", r.TaskID, run, err)
							}
						}
					},
				}
				agentRes, agentErr = experiment.RunAgentExperiment(cfg)
			}()
		}

		wg.Wait()

		newTextResults = append(newTextResults, textRes...)
		newRuntimeResults = append(newRuntimeResults, rtRes...)
		newAgentResults = append(newAgentResults, agentRes...)

		// Check for fatal API errors — any condition aborting aborts the whole run.
		for _, e := range []error{textErr, rtErr, agentErr} {
			if errors.Is(e, experiment.ErrAPIFatal) {
				fmt.Fprintf(os.Stderr, "  ✗ API credits/auth exhausted — saving partial results\n")
				goto doneLoop
			}
		}
		for cond, e := range map[string]error{"text": textErr, "runtime": rtErr, "agent": agentErr} {
			if e != nil && !errors.Is(e, experiment.ErrAPIFatal) {
				return fmt.Errorf("%s experiment task %s: %w", cond, task.ID, e)
			}
		}

		// Third condition (ranked/prefix) if requested
		var rkRes []experiment.TaskResult
		if thirdRetriever != nil {
			condLabel := thirdConditionLabel
			singleRankedConfig := experiment.ExperimentConfig{
				Tasks:            []experiment.TaskDefinition{task},
				Provider:         provider,
				Model:            modelName,
				RunsPerTask:      runs,
				SkipMap:          map[string]int{task.ID: rankedSkipMap[task.ID]},
				ContextRetriever: thirdRetriever,
				Condition:        condLabel,
				OnResult: func(r experiment.TaskResult, run int) {
					if cpWriter != nil {
						if err := cpWriter.Append(experiment.CheckpointEntry{
							TaskID:    r.TaskID,
							Condition: condLabel,
							Run:       run,
							Result:    experiment.ToResultJSON(r),
						}); err != nil {
							fmt.Fprintf(os.Stderr, "warn: checkpoint append %s/%s/run%d: %v\n", condLabel, r.TaskID, run, err)
						}
					}
				},
			}
			var rkErr error
			rkRes, rkErr = experiment.RunRuntimeExperiment(singleRankedConfig)
			newRankedResults = append(newRankedResults, rkRes...)
			if rkErr != nil {
				if errors.Is(rkErr, experiment.ErrAPIFatal) {
					fmt.Fprintf(os.Stderr, "  ✗ API credits/auth exhausted during %s condition — saving partial results\n", condLabel)
					break
				}
				return fmt.Errorf("%s experiment task %s: %w", condLabel, task.ID, rkErr)
			}
		}

		completedMsg := fmt.Sprintf("  ✓ %s complete: %d text + %d runtime runs", task.ID, len(textRes), len(rtRes))
		if agentEnabled {
			completedMsg += fmt.Sprintf(" + %d agent runs", len(agentRes))
		}
		if thirdRetriever != nil {
			completedMsg += fmt.Sprintf(" + %d %s runs", len(rkRes), thirdConditionLabel)
		}
		fmt.Fprintf(os.Stderr, "%s\n", completedMsg)

		// Intermediate results
		accumText := append(priorTextResults, newTextResults...)
		accumRT := append(priorRuntimeResults, newRuntimeResults...)
		if len(accumText) > 0 && len(accumRT) > 0 {
			interim := experiment.Compare(accumText, accumRT)
			dm := interim.DetectionRate
			textComp := experiment.CompletionRatePublic(accumText)
			rtComp := experiment.CompletionRatePublic(accumRT)
			interimMsg := fmt.Sprintf("  ── Interim (%d/%d tasks) ── completion: text %.0f%% / runtime %.0f%%",
				i+1, len(tasks), textComp*100, rtComp*100)
			if agentEnabled {
				agentAccum := append(priorAgentResults, newAgentResults...)
				agentComp := experiment.CompletionRatePublic(agentAccum)
				interimMsg += fmt.Sprintf(" / agent %.0f%%", agentComp*100)
			}
			fmt.Fprintf(os.Stderr, "%s | detection: %d/%d caught", interimMsg, dm.AnyCaught, dm.AnyFailures)
			if dm.AnyFailures > 0 {
				fmt.Fprintf(os.Stderr, " (%.0f%%)", dm.AnyRate*100)
			}
			fmt.Fprintf(os.Stderr, "\n")
		}
	}
doneLoop:

	textResults := append(priorTextResults, newTextResults...)
	runtimeResults := append(priorRuntimeResults, newRuntimeResults...)
	agentResults := append(priorAgentResults, newAgentResults...)
	rankedResults := append(priorRankedResults, newRankedResults...)

	// Flush condition snapshots
	if outputPath != "" {
		textJSON := make([]experiment.TaskResultJSON, len(textResults))
		for i, r := range textResults {
			textJSON[i] = experiment.ToResultJSON(r)
		}
		flushPath := strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".text.json"
		if data, jsonErr := json.MarshalIndent(textJSON, "", "  "); jsonErr == nil {
			if err := os.WriteFile(flushPath, data, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "warn: flush text snapshot to %s: %v\n", flushPath, err)
			}
		}

		rtJSON := make([]experiment.TaskResultJSON, len(runtimeResults))
		for i, r := range runtimeResults {
			rtJSON[i] = experiment.ToResultJSON(r)
		}
		flushPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".runtime.json"
		if data, jsonErr := json.MarshalIndent(rtJSON, "", "  "); jsonErr == nil {
			if err := os.WriteFile(flushPath, data, 0644); err != nil {
				fmt.Fprintf(os.Stderr, "warn: flush runtime snapshot to %s: %v\n", flushPath, err)
			}
		}

		if len(agentResults) > 0 {
			agJSON := make([]experiment.TaskResultJSON, len(agentResults))
			for i, r := range agentResults {
				agJSON[i] = experiment.ToResultJSON(r)
			}
			flushPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + ".agent.json"
			if data, jsonErr := json.MarshalIndent(agJSON, "", "  "); jsonErr == nil {
				if err := os.WriteFile(flushPath, data, 0644); err != nil {
					fmt.Fprintf(os.Stderr, "warn: flush agent snapshot to %s: %v\n", flushPath, err)
				}
			}
		}

		if len(rankedResults) > 0 {
			rkJSON := make([]experiment.TaskResultJSON, len(rankedResults))
			for i, r := range rankedResults {
				rkJSON[i] = experiment.ToResultJSON(r)
			}
			flushPath = strings.TrimSuffix(outputPath, filepath.Ext(outputPath)) + fmt.Sprintf(".%s.json", thirdConditionLabel)
			if data, jsonErr := json.MarshalIndent(rkJSON, "", "  "); jsonErr == nil {
				if err := os.WriteFile(flushPath, data, 0644); err != nil {
					fmt.Fprintf(os.Stderr, "warn: flush %s snapshot to %s: %v\n", thirdConditionLabel, flushPath, err)
				}
			}
		}
	}

	// Compare + report
	fmt.Fprintf(os.Stderr, "Comparing results...\n")
	var comparison *experiment.ComparisonResult
	switch {
	case len(agentResults) > 0 && len(rankedResults) > 0:
		// Both agent and ranked: prefer agent as the primary third condition.
		comparison = experiment.CompareThree(textResults, runtimeResults, agentResults)
	case len(agentResults) > 0:
		comparison = experiment.CompareThree(textResults, runtimeResults, agentResults)
	case len(rankedResults) > 0:
		comparison = experiment.CompareThree(textResults, runtimeResults, rankedResults)
	default:
		comparison = experiment.Compare(textResults, runtimeResults)
	}

	// Baseline delta: flag > env var > none.
	baselinePath, _ := cmd.Flags().GetString("baseline")
	if baselinePath == "" {
		baselinePath = os.Getenv("BENCH_BASELINE")
	}
	if baselinePath != "" {
		if bs, bErr := experiment.LoadBaseline(baselinePath); bErr != nil {
			fmt.Fprintf(os.Stderr, "warn: baseline unavailable (%s): %v\n", baselinePath, bErr)
		} else {
			experiment.ApplyBaseline(comparison.NodeMetrics, bs)
		}
	}

	// Save baseline if requested.
	saveBaselinePath, _ := cmd.Flags().GetString("save-baseline")
	if saveBaselinePath != "" {
		if sErr := experiment.SaveBaseline(saveBaselinePath, comparison, "forgent-bench run"); sErr != nil {
			fmt.Fprintf(os.Stderr, "warn: save-baseline failed (%s): %v\n", saveBaselinePath, sErr)
		} else {
			fmt.Fprintf(os.Stderr, "Baseline saved to %s\n", saveBaselinePath)
		}
	}

	report := experiment.FormatReport(comparison, experiment.ReportOpts{
		Model:        modelName,
		BaselinePath: baselinePath,
	})

	if outputPath != "" {
		if err := os.WriteFile(outputPath, []byte(report), 0644); err != nil {
			return fmt.Errorf("write report: %w", err)
		}
		fmt.Fprintf(os.Stderr, "Report written to %s\n", outputPath)
	} else {
		fmt.Print(report)
	}

	return nil
}

// readAllFiles concatenates all files in a directory tree.



func runConvertBaseline(_ *cobra.Command, args []string) error {
	textPath, runtimePath, outPath := args[0], args[1], args[2]

	// Parse both legacy JSON files as []TaskResultJSON.
	readLegacy := func(path string) ([]experiment.TaskResultJSON, error) {
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		var entries []experiment.TaskResultJSON
		if err := json.Unmarshal(data, &entries); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		return entries, nil
	}

	textEntries, err := readLegacy(textPath)
	if err != nil {
		return err
	}
	runtimeEntries, err := readLegacy(runtimePath)
	if err != nil {
		return err
	}
	all := append(textEntries, runtimeEntries...)

	// Accumulate per (taskID, condition): guardrail_score sum + step completion counts.
	type key struct{ taskID, condition string }
	type acc struct {
		ccrSum        float64
		runs          int
		stepCompleted map[string]int
		stepTotal     map[string]int
	}
	accs := map[key]*acc{}
	for _, e := range all {
		k := key{e.TaskID, e.Condition}
		if accs[k] == nil {
			accs[k] = &acc{
				stepCompleted: map[string]int{},
				stepTotal:     map[string]int{},
			}
		}
		a := accs[k]
		a.ccrSum += e.GuardrailScore // CCR proxy: 1.0 - guardrail_failure_fraction = guardrail_score
		a.runs++
		for _, sr := range e.StepResults {
			a.stepTotal[sr.StepID]++
			if sr.Completed {
				a.stepCompleted[sr.StepID]++
			}
		}
	}

	// Build NodeComplianceRates: task-level (for T1.3 assertion) + step-level (for per-node delta).
	nodeRates := map[string]map[string]float64{}
	setRate := func(nodeID, condition string, rate float64) {
		if nodeRates[nodeID] == nil {
			nodeRates[nodeID] = map[string]float64{}
		}
		nodeRates[nodeID][condition] = rate
	}
	for k, a := range accs {
		if a.runs == 0 {
			continue
		}
		avgCCR := a.ccrSum / float64(a.runs)
		setRate(k.taskID, k.condition, avgCCR)
		for stepID, total := range a.stepTotal {
			if total == 0 {
				continue
			}
			setRate(stepID, k.condition, float64(a.stepCompleted[stepID])/float64(total))
		}
	}

	bs := experiment.BaselineSnapshot{
		CapturedAt:          "2026-04-16",
		Description:         "Sonnet baseline (pre-Epic-10 keyword guardrails; CCR proxied from GuardrailScore)",
		NodeComplianceRates: nodeRates,
	}
	if err := experiment.WriteBaselineSnapshot(outPath, bs); err != nil {
		return fmt.Errorf("write baseline: %w", err)
	}

	fmt.Printf("Baseline written to %s (%d node/condition pairs)\n", outPath, countPairs(nodeRates))
	return nil
}

func countPairs(nodeRates map[string]map[string]float64) int {
	n := 0
	for _, conds := range nodeRates {
		n += len(conds)
	}
	return n
}
