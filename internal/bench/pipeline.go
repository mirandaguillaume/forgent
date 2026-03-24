package bench

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mirandaguillaume/forgent/internal/builder"
	"github.com/mirandaguillaume/forgent/internal/generator"
	"github.com/mirandaguillaume/forgent/internal/importer"
	"github.com/mirandaguillaume/forgent/internal/llm"
	"github.com/mirandaguillaume/forgent/internal/scanner"
)

// PipelineResult holds the three contestants produced from one hand-written agent.
type PipelineResult struct {
	Source      string
	HandWritten H2HContestant
	Standard    H2HContestant
	Compact     H2HContestant
	ImportTime  time.Duration
	BuildTime   time.Duration
	Retries     int
}

// RunPipeline takes a hand-written agent .md file, imports it via LLM,
// builds standard + compact variants, and returns 3 contestants.
// It retries the import up to maxRetries times if the build fails (e.g. lint errors).
func RunPipeline(agentPath string, provider llm.Provider) (*PipelineResult, error) {
	return RunPipelineWithRetries(agentPath, provider, 2)
}

// RunPipelineWithRetries is RunPipeline with configurable retry count.
func RunPipelineWithRetries(agentPath string, provider llm.Provider, maxRetries int) (*PipelineResult, error) {
	data, err := os.ReadFile(agentPath)
	if err != nil {
		return nil, fmt.Errorf("read agent file: %w", err)
	}
	prompt := string(data)
	name := strings.TrimSuffix(filepath.Base(agentPath), ".md")

	result := &PipelineResult{
		Source: name,
		HandWritten: H2HContestant{
			Name:   name,
			Prompt: prompt,
			Words:  generator.CountWords(prompt),
		},
	}

	var lastErr string
	importStart := time.Now()

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			result.Retries = attempt
		}

		pr, buildErr := tryImportAndBuild(agentPath, name, provider)
		if buildErr == nil {
			result.ImportTime = time.Since(importStart)
			result.BuildTime = pr.buildTime
			result.Standard = pr.standard
			result.Compact = pr.compact
			return result, nil
		}
		lastErr = buildErr.Error()
	}

	result.ImportTime = time.Since(importStart)
	return nil, fmt.Errorf("pipeline failed after %d retries: %s", maxRetries, lastErr)
}

type buildOutput struct {
	standard  H2HContestant
	compact   H2HContestant
	buildTime time.Duration
}

func tryImportAndBuild(agentPath, name string, provider llm.Provider) (*buildOutput, error) {
	tmpDir, err := os.MkdirTemp("", "forgent-bench-*")
	if err != nil {
		return nil, fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Import
	importResult := importer.RunImport(importer.ImportOptions{
		Source:   agentPath,
		Provider: provider,
		MinScore: 60,
	})
	if !importResult.Success {
		return nil, fmt.Errorf("import failed: %s", importResult.Error)
	}

	// Write to disk
	if _, err := importer.WriteImportResult(importResult, tmpDir); err != nil {
		return nil, fmt.Errorf("write import result: %w", err)
	}

	skillsDir := filepath.Join(tmpDir, "skills")
	agentsDir := filepath.Join(tmpDir, "agents")
	os.MkdirAll(agentsDir, 0755)

	// Build standard
	buildStart := time.Now()
	stdOutputDir := filepath.Join(tmpDir, "output-standard")
	stdBuild := builder.RunBuildWithOptions(skillsDir, agentsDir, stdOutputDir, "claude", scanner.EnrichNone, false)
	if !stdBuild.Success {
		return nil, fmt.Errorf("standard build: %s", stdBuild.Error)
	}

	// Build compact
	compactOutputDir := filepath.Join(tmpDir, "output-compact")
	compactBuild := builder.RunBuildWithOptions(skillsDir, agentsDir, compactOutputDir, "claude", scanner.EnrichNone, true)
	if !compactBuild.Success {
		return nil, fmt.Errorf("compact build: %s", compactBuild.Error)
	}
	buildTime := time.Since(buildStart)

	// Read output
	stdPrompt, err := readAllFilesInDir(stdOutputDir)
	if err != nil {
		return nil, fmt.Errorf("read standard output: %w", err)
	}
	compactPrompt, err := readAllFilesInDir(compactOutputDir)
	if err != nil {
		return nil, fmt.Errorf("read compact output: %w", err)
	}

	return &buildOutput{
		standard: H2HContestant{
			Name:   name + "/forgent-standard",
			Prompt: stdPrompt,
			Words:  generator.CountWords(stdPrompt),
		},
		compact: H2HContestant{
			Name:   name + "/forgent-compact",
			Prompt: compactPrompt,
			Words:  generator.CountWords(compactPrompt),
		},
		buildTime: buildTime,
	}, nil
}
