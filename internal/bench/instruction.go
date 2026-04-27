package bench

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mirandaguillaume/forgent/internal/doctor/analyzer"
	"github.com/mirandaguillaume/forgent/internal/doctor/llmutil"
	"github.com/mirandaguillaume/forgent/internal/doctor/parser"
	"github.com/mirandaguillaume/forgent/internal/doctor/registry"
	"github.com/mirandaguillaume/forgent/internal/llm"
	"gopkg.in/yaml.v3"
)

// InstructionBenchOpts configures an InstructionBench run.
type InstructionBenchOpts struct {
	CorpusDir  string       // directory containing claude/ and copilot/ subdirectories
	Provider   llm.Provider // LLM provider for analysis (nil = static-only mode)
	GoldenPath string       // path to golden.yaml for false positive measurement (optional)
	Model      string       // model name for reporting metadata
	Registry   *registry.Registry
}

// InstructionBenchResult holds the aggregated results of an InstructionBench run.
type InstructionBenchResult struct {
	TotalFiles             int
	TotalFindings          int
	ByCategory             map[string]*CategoryResult
	ConfidenceDistribution map[string]int // "high" → count, "moderate" → count, "low" → count
	OverallScore           float64        // percentage of files with at least one finding
	FileResults            []FileResult   // per-file details
	GoldenFileCount        int            // number of files in the golden set (0 if no golden set)
}

// CategoryResult holds per-category metrics.
type CategoryResult struct {
	Count             int
	Accepted          int     // findings with confidence >= moderate (accumulated during collection)
	AcceptanceRate    float64 // Accepted / Count * 100
	FalsePositives    int     // count of findings not in golden set (if available)
	FalsePositiveRate float64 // FalsePositives / Count * 100
}

// FileResult holds per-file analysis results.
type FileResult struct {
	Path     string
	Format   string
	Findings []llmutil.Finding
	Error    error
}

// GoldenSet maps file paths to expected finding categories.
type GoldenSet struct {
	Files map[string]GoldenEntry `yaml:"files"`
}

// GoldenEntry holds the expected categories for a single file.
type GoldenEntry struct {
	ExpectedCategories []string `yaml:"expected_categories"`
}

// RunInstructionBench executes the benchmark against a corpus of agent files.
// It parses each file, runs the doctor analyzer, and collects metrics.
func RunInstructionBench(opts InstructionBenchOpts) (*InstructionBenchResult, error) {
	if opts.CorpusDir == "" {
		return nil, fmt.Errorf("corpus directory is required")
	}

	if opts.Registry == nil {
		reg, err := registry.Load("")
		if err != nil {
			return nil, fmt.Errorf("load registry: %w", err)
		}
		opts.Registry = reg
	}

	// Discover corpus files
	files, err := discoverCorpusFiles(opts.CorpusDir)
	if err != nil {
		return nil, fmt.Errorf("discover corpus: %w", err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("no .md files found in %s", opts.CorpusDir)
	}

	// Load golden set if provided
	var golden *GoldenSet
	if opts.GoldenPath != "" {
		golden, err = loadGoldenSet(opts.GoldenPath)
		if err != nil {
			return nil, fmt.Errorf("load golden set: %w", err)
		}
	}

	result := &InstructionBenchResult{
		ByCategory:             make(map[string]*CategoryResult),
		ConfidenceDistribution: make(map[string]int),
	}

	if golden != nil {
		result.GoldenFileCount = len(golden.Files)
	}

	for _, cf := range files {
		fr := analyzeCorpusFile(cf.path, cf.simPath, opts.Provider, opts.Registry)
		result.FileResults = append(result.FileResults, fr)
		result.TotalFiles++

		if fr.Error != nil {
			continue
		}

		for _, f := range fr.Findings {
			result.TotalFindings++

			// Category tracking
			cat, ok := result.ByCategory[f.Category]
			if !ok {
				cat = &CategoryResult{}
				result.ByCategory[f.Category] = cat
			}
			cat.Count++

			// Acceptance: accumulate during first pass (confidence >= moderate)
			conf := strings.ToLower(f.Confidence)
			if conf == "" {
				conf = "unknown"
			}
			if conf == "high" || conf == "moderate" {
				cat.Accepted++
			}

			// Confidence distribution
			result.ConfidenceDistribution[conf]++

			// False positive check against golden set
			if golden != nil {
				relPath := cf.relPath
				if entry, exists := golden.Files[relPath]; exists {
					if !containsString(entry.ExpectedCategories, f.Category) {
						cat.FalsePositives++
					}
				}
			}
		}
	}

	// Compute rates from accumulated counts
	for _, cat := range result.ByCategory {
		if cat.Count > 0 {
			cat.AcceptanceRate = float64(cat.Accepted) / float64(cat.Count) * 100
			cat.FalsePositiveRate = float64(cat.FalsePositives) / float64(cat.Count) * 100
		}
	}

	// Overall score: percentage of files that produced findings
	if result.TotalFiles > 0 {
		filesWithFindings := 0
		for _, fr := range result.FileResults {
			if len(fr.Findings) > 0 {
				filesWithFindings++
			}
		}
		result.OverallScore = float64(filesWithFindings) / float64(result.TotalFiles) * 100
	}

	return result, nil
}

type corpusFile struct {
	path    string // absolute path
	simPath string // simulated path for format detection
	relPath string // relative to corpus dir (for golden set matching)
}

func discoverCorpusFiles(dir string) ([]corpusFile, error) {
	var files []corpusFile

	subdirs := []struct {
		name    string
		simBase string
	}{
		{"claude", filepath.Join(".claude", "agents")},
		{"copilot", filepath.Join(".github", "agents")},
	}

	for _, sub := range subdirs {
		subDir := filepath.Join(dir, sub.name)
		entries, err := os.ReadDir(subDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read %s: %w", subDir, err)
		}

		for _, e := range entries {
			if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
				continue
			}
			files = append(files, corpusFile{
				path:    filepath.Join(subDir, e.Name()),
				simPath: filepath.Join(sub.simBase, e.Name()),
				relPath: filepath.Join(sub.name, e.Name()),
			})
		}
	}

	// Also check for flat .md files in the root (format auto-detected)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" || e.Name() == "README.md" {
			continue
		}
		files = append(files, corpusFile{
			path:    filepath.Join(dir, e.Name()),
			simPath: e.Name(),
			relPath: e.Name(),
		})
	}

	sort.Slice(files, func(i, j int) bool { return files[i].relPath < files[j].relPath })
	return files, nil
}

func analyzeCorpusFile(path, simPath string, provider llm.Provider, reg *registry.Registry) FileResult {
	content, err := os.ReadFile(path)
	if err != nil {
		return FileResult{Path: path, Error: fmt.Errorf("read: %w", err)}
	}

	p, err := parser.DetectFormat(simPath, content)
	if err != nil {
		return FileResult{Path: path, Error: fmt.Errorf("detect format: %w", err)}
	}

	analysis, err := p.Parse(content)
	if err != nil {
		return FileResult{Path: path, Format: p.Format(), Error: fmt.Errorf("parse: %w", err)}
	}

	fr := FileResult{
		Path:   path,
		Format: analysis.Format,
	}

	if provider != nil {
		findings, err := analyzer.Analyze(analysis, provider, reg)
		if err != nil {
			fr.Error = fmt.Errorf("analyze: %w", err)
			return fr
		}
		fr.Findings = findings
	}

	return fr
}

func loadGoldenSet(path string) (*GoldenSet, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var gs GoldenSet
	if err := yaml.Unmarshal(data, &gs); err != nil {
		return nil, fmt.Errorf("parse golden set: %w", err)
	}
	return &gs, nil
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
