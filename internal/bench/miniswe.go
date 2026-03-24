package bench

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// MiniSWEConfig holds configuration for the mini-SWE-agent wrapper.
type MiniSWEConfig struct {
	PythonBin string // path to python with mini-swe-agent installed
	Model     string // LLM model (e.g. "claude-sonnet-4-20250514")
	Workers   int    // parallel instances
	Subset    string // SWE-bench subset: "lite", "verified", etc.
	Slice     string // e.g. "0:5" for first 5 instances
	CostLimit float64
}

// DefaultMiniSWEConfig returns sensible defaults.
func DefaultMiniSWEConfig() MiniSWEConfig {
	return MiniSWEConfig{
		PythonBin: "/tmp/miniswe-env/bin/python",
		Model:     "claude-sonnet-4-20250514",
		Workers:   1,
		Subset:    "verified",
		Slice:     "",
		CostLimit: 3.0,
	}
}

// MiniSWERunResult holds the result of a mini-SWE-agent benchmark run.
type MiniSWERunResult struct {
	Name       string
	OutputDir  string
	Instances  int
	Resolved   int
	ResolveRate float64
	TotalCost  float64
}

// miniSWEOverrideConfig is the YAML structure for overriding mini-SWE-agent config.
type miniSWEOverrideConfig struct {
	Agent struct {
		SystemTemplate   string  `yaml:"system_template,omitempty"`
		InstanceTemplate string  `yaml:"instance_template,omitempty"`
		CostLimit        float64 `yaml:"cost_limit"`
	} `yaml:"agent"`
}

// RunMiniSWEBench runs mini-SWE-agent with a custom system prompt on SWE-bench.
// It injects the Forgent-generated prompt as system_template, keeping the standard
// instance_template and observation handling from mini-SWE-agent.
func RunMiniSWEBench(name, systemPrompt string, cfg MiniSWEConfig) (*MiniSWERunResult, error) {
	// Verify mini-SWE-agent is installed
	if err := CheckMiniSWEInstalled(cfg.PythonBin); err != nil {
		return nil, err
	}

	// Create temp dir for config and output
	tmpDir, err := os.MkdirTemp("", "forgent-miniswe-*")
	if err != nil {
		return nil, fmt.Errorf("create tmpdir: %w", err)
	}

	// Write override config with our system prompt
	overridePath := filepath.Join(tmpDir, "override.yaml")
	if err := writeMiniSWEOverride(overridePath, systemPrompt, cfg.CostLimit); err != nil {
		return nil, fmt.Errorf("write override config: %w", err)
	}

	// Output directory for trajectories
	outputDir := filepath.Join(tmpDir, "output")

	// Build mini-extra swebench command
	args := []string{
		"-m", "minisweagent.extras",
		"swebench",
		"--config", "swebench.yaml",
		"--config", overridePath,
		"--model", cfg.Model,
		"--subset", cfg.Subset,
		"--output", outputDir,
		"--workers", fmt.Sprintf("%d", cfg.Workers),
	}
	if cfg.Slice != "" {
		args = append(args, "--slice", cfg.Slice)
	}

	cmd := exec.Command(cfg.PythonBin, args...)
	cmd.Env = append(os.Environ(), "PYTHONUNBUFFERED=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("mini-swe-agent: %s (%w)", string(output), err)
	}

	// Parse results from output directory
	result := &MiniSWERunResult{
		Name:      name,
		OutputDir: outputDir,
	}

	result.Instances, result.Resolved, result.TotalCost = parseMiniSWEResults(outputDir)
	if result.Instances > 0 {
		result.ResolveRate = float64(result.Resolved) / float64(result.Instances) * 100
	}

	return result, nil
}

// RunMiniSWEComparison runs the baseline (default prompt) and Forgent prompt,
// then compares results.
func RunMiniSWEComparison(forgentPrompt string, cfg MiniSWEConfig, progress func(string, *MiniSWERunResult)) ([]*MiniSWERunResult, error) {
	var results []*MiniSWERunResult

	// 1. Run with default mini-SWE-agent prompt (baseline)
	baseline, err := RunMiniSWEBench("mini-swe-agent-default", "", cfg)
	if err != nil {
		return nil, fmt.Errorf("baseline run: %w", err)
	}
	results = append(results, baseline)
	if progress != nil {
		progress("baseline", baseline)
	}

	// 2. Run with Forgent-generated prompt
	forgent, err := RunMiniSWEBench("forgent-compact", forgentPrompt, cfg)
	if err != nil {
		return nil, fmt.Errorf("forgent run: %w", err)
	}
	results = append(results, forgent)
	if progress != nil {
		progress("forgent", forgent)
	}

	return results, nil
}

// CheckMiniSWEInstalled verifies mini-SWE-agent is available.
func CheckMiniSWEInstalled(pythonBin string) error {
	cmd := exec.Command(pythonBin, "-c", "import minisweagent; print(minisweagent.__version__)")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mini-swe-agent not installed at %s: %s (%w)", pythonBin, string(output), err)
	}
	return nil
}

// writeMiniSWEOverride writes a YAML config that overrides the system template.
// When systemPrompt is empty, the default mini-SWE-agent prompt is used.
func writeMiniSWEOverride(path, systemPrompt string, costLimit float64) error {
	var cfg miniSWEOverrideConfig
	cfg.Agent.CostLimit = costLimit

	if systemPrompt != "" {
		cfg.Agent.SystemTemplate = systemPrompt
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// parseMiniSWEResults reads trajectory files from a mini-SWE-agent output directory
// and counts instances and resolved tasks.
func parseMiniSWEResults(outputDir string) (instances, resolved int, totalCost float64) {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return 0, 0, 0
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		data, err := os.ReadFile(filepath.Join(outputDir, entry.Name()))
		if err != nil {
			continue
		}

		var traj struct {
			Info struct {
				ExitStatus string `json:"exit_status"`
				Submission string `json:"submission"`
				ModelStats struct {
					InstanceCost float64 `json:"instance_cost"`
				} `json:"model_stats"`
			} `json:"info"`
		}
		if err := json.Unmarshal(data, &traj); err != nil {
			continue
		}

		instances++
		totalCost += traj.Info.ModelStats.InstanceCost

		// A task is "resolved" if the agent submitted a non-empty patch
		// Real resolve rate requires running the SWE-bench harness on top
		if traj.Info.Submission != "" && traj.Info.ExitStatus != "LimitsExceeded" {
			resolved++
		}
	}

	return
}
