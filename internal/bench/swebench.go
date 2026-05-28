package bench

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

	return parseHarnessResults(runID, predictionsPath)
}

// parseHarnessResults reads the harness output to find resolved instances.
// SWE-bench v4 writes: {model_name}.{run_id}.json in the current working directory.
func parseHarnessResults(runID, predictionsPath string) (map[string]bool, error) {
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
