package cmd

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/mirandaguillaume/forgent/internal/doctor"
	"github.com/mirandaguillaume/forgent/internal/doctor/analyzer"
	doctorctx "github.com/mirandaguillaume/forgent/internal/doctor/context"
	"github.com/mirandaguillaume/forgent/internal/doctor/llmutil"
	"github.com/mirandaguillaume/forgent/internal/doctor/parser"
	"github.com/mirandaguillaume/forgent/internal/doctor/registry"
	"github.com/mirandaguillaume/forgent/internal/doctor/rewrite"
	docindex "github.com/mirandaguillaume/forgent/internal/doctor/index"
	"github.com/mirandaguillaume/forgent/internal/doctor/scaffold"
	"github.com/mirandaguillaume/forgent/internal/doctor/static"
	"github.com/mirandaguillaume/forgent/internal/scanner"
	"github.com/mirandaguillaume/forgent/internal/doctor/validate"
	"github.com/mirandaguillaume/forgent/internal/fileops"
	"github.com/mirandaguillaume/forgent/internal/llm"
	"github.com/spf13/cobra"
)

// ErrFindings is returned when doctor finds issues. Callers should
// inspect this with errors.Is and set exit code 2 themselves.
var ErrFindings = errors.New("doctor found issues")

func init() {
	var providerFlag string
	var modelFlag string
	var debugFlag bool
	var updateRegistryFlag bool
	var fixFlag bool
	var forceFlag bool
	var noCacheFlag bool
	var quickFlag bool
	var ciFlag bool
	var thoroughFlag bool
	var securityFlag bool
	var formatFlag string

	doctorCmd := &cobra.Command{
		Use:   "doctor <file|directory>",
		Short: "Analyze and improve agent definition files",
		Long: `Analyze any agent file (Claude Code, GitHub Copilot, Forgent YAML) and
produce research-backed recommendations to improve it.

When given a directory, discovers all agent files recursively and
produces an aggregate summary after individual file results.

Examples:
  forgent doctor .claude/agents/code-reviewer.md
  forgent doctor .github/agents/dash.agent.md
  forgent doctor skills/review-commenter.skill.yaml
  forgent doctor .claude/agents/                      # directory mode
  forgent doctor agent.md --provider openrouter
  forgent doctor agent.md --fix                       # interactive rewrite
  forgent doctor agent.md --fix --force               # skip diff-size check
  forgent doctor --update-registry                    # download latest registry`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, _ := os.Getwd()

			// Load research registry
			reg, err := registry.Load(cwd)
			if err != nil {
				return fmt.Errorf("load research registry: %w", err)
			}

			if debugFlag {
				fmt.Fprintf(os.Stderr, "[DEBUG] Registry: version=%s source=%s\n", reg.Version, reg.Source)
				if reg.SourcePath != "" {
					fmt.Fprintf(os.Stderr, "[DEBUG] Registry path: %s\n", reg.SourcePath)
				}
			}

			// --update-registry: download and exit (no analysis)
			if updateRegistryFlag {
				return registry.Update(reg, cwd)
			}

			if len(args) == 0 {
				return fmt.Errorf("requires a file or directory argument (or --update-registry)")
			}

			target := args[0]

			// --fix: scaffold AGENTS.md index + .agents/*.md
			if fixFlag {
				return runScaffoldMode(target, debugFlag, reg)
			}

			// Determine if target is file or directory
			info, err := os.Stat(target)
			if err != nil {
				return fmt.Errorf("cannot access %s: %w", target, err)
			}

			// Determine analysis mode
			mode := "default"
			if quickFlag {
				mode = "quick"
			} else if thoroughFlag {
				mode = "thorough"
			} else if securityFlag {
				mode = "security"
			}

			// CI mode: force plain text, auto-fallback to quick if no LLM
			if ciFlag {
				color.NoColor = true
			}

			// Check if target is an index file
			var indexContent []byte
			isIndex := false
			if !info.IsDir() {
				if c, err := os.ReadFile(target); err == nil {
					indexContent = c
					isIndex = docindex.IsIndex(c)
				}
			}

			var analysisErr error
			if isIndex {
				analysisErr = runIndexModeWithContent(target, indexContent, mode, formatFlag, ciFlag, reg)
			} else if mode == "quick" && formatFlag == "" && !ciFlag {
				analysisErr = runQuickMode(target, reg)
			} else if mode == "quick" && (formatFlag != "" || ciFlag) {
				// Quick mode but with structured output — use pipeline
				analysisErr = runPipelineMode(target, providerFlag, modelFlag, debugFlag, noCacheFlag, ciFlag, mode, formatFlag, reg)
			} else if info.IsDir() {
				analysisErr = runDirectoryMode(target, providerFlag, modelFlag, debugFlag, noCacheFlag, reg)
			} else {
				analysisErr = runPipelineMode(target, providerFlag, modelFlag, debugFlag, noCacheFlag, ciFlag, mode, formatFlag, reg)
			}

			// Show registry update notification once, after all findings
			notifyRegistryUpdate(reg)

			return analysisErr
		},
	}

	doctorCmd.Flags().StringVar(&providerFlag, "provider", "", "LLM provider (ollama, openrouter, anthropic). Default: auto-detect")
	doctorCmd.Flags().StringVar(&modelFlag, "model", "", "LLM model name (provider-specific)")
	doctorCmd.Flags().BoolVar(&debugFlag, "debug", false, "Show debug output on stderr")
	doctorCmd.Flags().BoolVar(&updateRegistryFlag, "update-registry", false, "Download the latest research registry and exit")
	doctorCmd.Flags().BoolVar(&fixFlag, "fix", false, "Generate AGENTS.md index + specialized files from agent file")
	doctorCmd.Flags().BoolVar(&forceFlag, "force", false, "Skip diff-size sanity check (use with --fix)")
	doctorCmd.Flags().BoolVar(&noCacheFlag, "no-cache", false, "Bypass LLM response cache")
	doctorCmd.Flags().BoolVar(&quickFlag, "quick", false, "Static checks only — no LLM, instant results")
	doctorCmd.Flags().BoolVar(&ciFlag, "ci", false, "CI mode: plain text, quality gate, GitHub annotations")
	doctorCmd.Flags().BoolVar(&thoroughFlag, "thorough", false, "Run all checks including thorough-tagged")
	doctorCmd.Flags().BoolVar(&securityFlag, "security", false, "Run security-tagged checks only")
	doctorCmd.Flags().StringVar(&formatFlag, "format", "", "Output format (json)")
	rootCmd.AddCommand(doctorCmd)
}

// notifyRegistryUpdate prints a one-time notification if a newer registry is available.
func notifyRegistryUpdate(reg *registry.Registry) {
	if !reg.NeedsUpdate() {
		return
	}

	tty := doctor.IsTTY()
	if tty {
		fmt.Fprintf(os.Stderr, "\n%s Research registry update available (current: %s, latest: %s).\n",
			color.YellowString("[INFO]"), reg.Version, reg.LatestVersion)
		fmt.Fprintf(os.Stderr, "  Run: %s\n", color.CyanString("forgent doctor --update-registry"))
	} else {
		fmt.Fprintf(os.Stderr, "\nINFO: Research registry update available (current: %s, latest: %s).\n", reg.Version, reg.LatestVersion)
		fmt.Fprintf(os.Stderr, "  Run: forgent doctor --update-registry\n")
	}
}

// runQuickMode runs static checks only — no LLM, instant results.
func runQuickMode(target string, reg *registry.Registry) error {
	info, err := os.Stat(target)
	if err != nil {
		return fmt.Errorf("cannot access %s: %w", target, err)
	}

	var files []string
	if info.IsDir() {
		files, err = doctor.DiscoverAgentFiles(target)
		if err != nil {
			return fmt.Errorf("discover agent files: %w", err)
		}
	} else {
		files = []string{target}
	}

	if len(files) == 0 {
		fmt.Println("No agent files found.")
		return nil
	}

	tty := doctor.IsTTY()
	totalFindings := 0

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", f, err)
			continue
		}

		p, err := parser.DetectFormat(f, content)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Cannot detect format of %s: %v\n", f, err)
			continue
		}

		analysis, err := p.Parse(content)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Parse error in %s: %v\n", f, err)
			continue
		}

		// Run static checks only
		findings := static.RunChecks(analysis, "quick")
		structural := doctor.ComputeStructural(findings, sectionMappingCount)
		doctor.RenderStructural(structural, f, tty)

		// Also show non-presence findings (vague, ordering)
		var otherFindings []llmutil.Finding
		for _, finding := range findings {
			if !strings.Contains(finding.Issue, "Missing") {
				otherFindings = append(otherFindings, finding)
			}
		}
		if len(otherFindings) > 0 {
			doctor.RenderFindings(otherFindings, f, tty, reg)
		}
		totalFindings += len(findings)
	}

	if totalFindings > 0 {
		return ErrFindings
	}
	return nil
}

// sectionMappingCount for structural score — must match static/presence.go (15 sections).
// ComputeStructural adds +1 for secrets check internally.
const sectionMappingCount = 15

// runIndexModeWithContent analyzes an AGENTS.md index by resolving links and computing aggregate score.
func runIndexModeWithContent(target string, content []byte, mode, formatFlag string, ciFlag bool, reg *registry.Registry) error {
	baseDir := filepath.Dir(target)
	resolved := docindex.ResolveIndex(content, baseDir)
	tty := doctor.IsTTY() && !ciFlag

	// Show index resolution
	if tty {
		fmt.Printf("\nIndex: %s (%d referenced files)\n", target, len(resolved))
	} else {
		fmt.Printf("Index: %s (%d referenced files)\n", target, len(resolved))
	}

	// Show missing file findings
	missingFindings := docindex.MissingFiles(resolved)
	for _, f := range missingFindings {
		if tty {
			fmt.Printf("  %s %s\n", color.RedString("x"), f.Issue)
		} else {
			fmt.Printf("  MISSING: %s\n", f.Issue)
		}
	}

	// Compute aggregate score
	agg := docindex.ScoreIndex(resolved, mode)

	// Render
	if formatFlag == "json" {
		jsonBytes, err := docindex.AggregateToJSON(agg, target)
		if err != nil {
			return fmt.Errorf("JSON output: %w", err)
		}
		fmt.Println(string(jsonBytes))
	} else {
		fmt.Print(docindex.FormatAggregate(agg))
	}

	if len(missingFindings) > 0 || agg.Covered < agg.Total {
		return ErrFindings
	}
	return nil
}

// runScaffoldMode generates AGENTS.md index + .agents/*.md from the original file.
func runScaffoldMode(target string, debugFlag bool, reg *registry.Registry) error {
	content, err := os.ReadFile(target)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", target, err)
	}

	// Check if this is already an index — scaffold only missing files
	if docindex.IsIndex(content) {
		return scaffoldMissing(target, content, debugFlag)
	}

	// Parse the original file
	p, pErr := parser.DetectFormat(target, content)
	var analysis *parser.AgentAnalysis
	if pErr == nil {
		analysis, _ = p.Parse(content)
	}
	if analysis == nil {
		analysis = &parser.AgentAnalysis{Format: "unknown", RawContent: content}
	}

	// Scan codebase for context
	projectRoot := doctorctx.DetectProjectRoot(filepath.Dir(target))
	var ctx *scanner.CodebaseContext
	if projectRoot != "" {
		ctx, _ = scanner.ScanCodebase(projectRoot)
	}

	// Generate scaffold
	result, err := scaffold.Scaffold(analysis, ctx)
	if err != nil {
		return fmt.Errorf("scaffold: %w", err)
	}

	// Write files
	baseDir := filepath.Dir(target)
	if baseDir == "." {
		baseDir, _ = os.Getwd()
	}

	tty := doctor.IsTTY()

	// Write AGENTS.md (skip if exists)
	indexPath := filepath.Join(baseDir, "AGENTS.md")
	if _, err := os.Stat(indexPath); err == nil {
		if tty {
			fmt.Printf("  %s %s (already exists, skipped)\n", color.YellowString("~"), indexPath)
		} else {
			fmt.Printf("  ~ %s (already exists, skipped)\n", indexPath)
		}
	} else {
		if err := os.WriteFile(indexPath, result.IndexContent, 0644); err != nil {
			return fmt.Errorf("write index: %w", err)
		}
		if tty {
			fmt.Printf("  %s %s\n", color.GreenString("+"), indexPath)
		} else {
			fmt.Printf("  + %s\n", indexPath)
		}
	}

	// Write .agents/*.md (skip existing)
	written := 0
	for relPath, content := range result.Files {
		fullPath := filepath.Join(baseDir, relPath)
		if _, err := os.Stat(fullPath); err == nil {
			if tty {
				fmt.Printf("  %s %s (already exists, skipped)\n", color.YellowString("~"), fullPath)
			} else {
				fmt.Printf("  ~ %s (already exists, skipped)\n", fullPath)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			return fmt.Errorf("create dir for %s: %w", relPath, err)
		}
		if err := os.WriteFile(fullPath, content, 0644); err != nil {
			return fmt.Errorf("write %s: %w", relPath, err)
		}
		if tty {
			fmt.Printf("  %s %s\n", color.GreenString("+"), fullPath)
		} else {
			fmt.Printf("  + %s\n", fullPath)
		}
		written++
	}

	if written == 0 {
		fmt.Println("\nAll files already exist. Nothing written.")
	} else if tty {
		color.Green("\nScaffold complete: %d files written (%d migrated, %d templated)",
			written, result.MigratedCount, result.TemplatedCount)
	} else {
		fmt.Printf("\nScaffold complete: %d files written (%d migrated, %d templated)\n",
			written, result.MigratedCount, result.TemplatedCount)
	}

	return nil
}

// scaffoldMissing generates only the missing files referenced in an existing index.
func scaffoldMissing(indexPath string, indexContent []byte, debugFlag bool) error {
	baseDir := filepath.Dir(indexPath)
	resolved := docindex.ResolveIndex(indexContent, baseDir)

	tty := doctor.IsTTY()
	created := 0

	for _, rf := range resolved {
		if !rf.Missing {
			continue
		}
		// Find matching template
		fullPath := filepath.Join(baseDir, rf.Path)
		for _, sf := range scaffold.DefaultFiles {
			if strings.Contains(rf.Path, sf.Name) {
				content := sf.Title + " — TODO: customize for your project.\n"
				if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
					return fmt.Errorf("create dir: %w", err)
				}
				if err := os.WriteFile(fullPath, []byte("# "+content), 0644); err != nil {
					return fmt.Errorf("write %s: %w", rf.Path, err)
				}
				if tty {
					fmt.Printf("  %s %s (new)\n", color.GreenString("+"), fullPath)
				} else {
					fmt.Printf("  + %s (new)\n", fullPath)
				}
				created++
				break
			}
		}
	}

	if created == 0 {
		fmt.Println("All referenced files exist. Nothing to scaffold.")
	} else if tty {
		color.Green("\n%d missing files created.", created)
	} else {
		fmt.Printf("\n%d missing files created.\n", created)
	}

	return nil
}

// runPipelineMode runs the full pipeline: static → LLM → post-process → gate.
func runPipelineMode(filePath, providerFlag, modelFlag string, debugFlag, noCacheFlag, ciFlag bool, mode, formatFlag string, reg *registry.Registry) error {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", filePath, err)
	}
	p, err := parser.DetectFormat(filePath, content)
	if err != nil {
		return fmt.Errorf("cannot detect format of %s: %w", filePath, err)
	}
	analysis, err := p.Parse(content)
	if err != nil {
		return fmt.Errorf("parse error in %s: %w", filePath, err)
	}

	tty := doctor.IsTTY() && !ciFlag

	// Phase 1: Run static checks + render structural (instant)
	opts := doctor.PipelineOpts{
		Mode:         mode,
		SectionCount: sectionMappingCount,
	}

	staticFindings := static.RunChecks(analysis, mode)
	structural := doctor.ComputeStructural(staticFindings, sectionMappingCount)
	if !ciFlag || formatFlag == "" {
		doctor.RenderStructural(structural, filePath, tty)
	}

	// Phase 2: LLM analysis (unless CI with no provider)
	var llmFindings []llmutil.Finding
	_, providerName, provErr := selectProvider(providerFlag, modelFlag, debugFlag)
	if provErr != nil {
		if ciFlag {
			if debugFlag {
				fmt.Fprintf(os.Stderr, "[DEBUG] CI mode: no LLM available, static only\n")
			}
		} else if tty {
			fmt.Fprintf(os.Stderr, "\nNo LLM provider available — showing static analysis only.\n")
		}
	} else {
		if tty {
			fmt.Fprintf(os.Stderr, "\nAnalyzing with LLM...\n")
		}
		llmFindings, _, err = analyzeFile(filePath, providerFlag, modelFlag, debugFlag, noCacheFlag, reg)
		if err != nil {
			if debugFlag {
				fmt.Fprintf(os.Stderr, "[DEBUG] LLM analysis error: %v\n", err)
			}
		}
		_ = providerName // used only for debug
	}

	// Phase 3: Run full pipeline
	report := doctor.RunPipeline(analysis, llmFindings, opts)
	report.FilePath = filePath

	// Phase 4: Output
	if formatFlag == "json" {
		jsonBytes, err := doctor.ToJSON(report, filePath)
		if err != nil {
			return fmt.Errorf("JSON output: %w", err)
		}
		fmt.Println(string(jsonBytes))
	} else if ciFlag {
		doctor.RenderGitHubAnnotations(report.AllFindings, filePath)
		doctor.RenderGateAnnotation(report.GateResult)
	} else {
		doctor.RenderAntiPatterns(report.AllFindings, tty)
		doctor.RenderFindings(report.AllFindings, filePath, tty, reg)
	}

	// Gate result determines exit code — only fail on gate failure
	if !report.GateResult.Pass {
		return ErrFindings
	}
	return nil
}

// runFileMode analyzes a single file with progressive display: static instant → LLM.
func runFileMode(filePath, providerFlag, modelFlag string, debugFlag, noCacheFlag bool, reg *registry.Registry) error {
	// Read and parse for static checks first
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", filePath, err)
	}
	p, err := parser.DetectFormat(filePath, content)
	if err != nil {
		return fmt.Errorf("cannot detect format of %s: %w", filePath, err)
	}
	analysis, err := p.Parse(content)
	if err != nil {
		return fmt.Errorf("parse error in %s: %w", filePath, err)
	}

	tty := doctor.IsTTY()

	// Phase 1: Static checks (instant)
	staticFindings := static.RunChecks(analysis, "default")
	structural := doctor.ComputeStructural(staticFindings, sectionMappingCount)
	doctor.RenderStructural(structural, filePath, tty)

	// Phase 2: LLM analysis
	if tty {
		fmt.Fprintf(os.Stderr, "\nAnalyzing with LLM...\n")
	}
	findings, _, err := analyzeFile(filePath, providerFlag, modelFlag, debugFlag, noCacheFlag, reg)
	if err != nil {
		return err
	}

	doctor.RenderFindings(findings, filePath, tty, reg)

	totalFindings := len(staticFindings) + len(findings)
	if totalFindings > 0 {
		return ErrFindings
	}
	return nil
}

// runDirectoryMode discovers agent files, analyzes each, and renders an aggregate summary.
func runDirectoryMode(dirPath, providerFlag, modelFlag string, debugFlag, noCacheFlag bool, reg *registry.Registry) error {
	files, err := doctor.DiscoverAgentFiles(dirPath)
	if err != nil {
		return fmt.Errorf("discover agent files in %s: %w", dirPath, err)
	}

	if len(files) == 0 {
		tty := doctor.IsTTY()
		if tty {
			color.Green("No agent files found in %s", dirPath)
		} else {
			fmt.Printf("No agent files found in %s\n", dirPath)
		}
		return nil
	}

	tty := doctor.IsTTY()
	var results []doctor.FileResult
	totalFindings := 0

	for i, f := range files {
		if debugFlag {
			fmt.Fprintf(os.Stderr, "[DEBUG] Analyzing file %d of %d: %s\n", i+1, len(files), f)
		} else {
			fmt.Fprintf(os.Stderr, "Analyzing file %d of %d: %s\n", i+1, len(files), f)
		}

		findings, format, err := analyzeFile(f, providerFlag, modelFlag, debugFlag, noCacheFlag, reg)
		result := doctor.FileResult{Path: f, Format: format, Error: err}
		if err != nil {
			fmt.Fprintf(os.Stderr, "  Error: %s\n", err)
		} else {
			result.Findings = findings
			doctor.RenderFindings(findings, f, tty, reg)
			totalFindings += len(findings)
		}
		results = append(results, result)
	}

	// Cross-file consistency analysis (if >1 file and LLM available)
	if len(results) > 1 {
		provider, _, provErr := selectProvider(providerFlag, modelFlag, debugFlag)
		if provErr == nil {
			var analyses []*parser.AgentAnalysis
			var filePaths []string
			for _, r := range results {
				if r.Error == nil && r.Findings != nil {
					// Re-parse for cross-file (lightweight — already in memory conceptually)
					content, err := os.ReadFile(r.Path)
					if err != nil {
						continue
					}
					p, err := parser.DetectFormat(r.Path, content)
					if err != nil {
						continue
					}
					a, err := p.Parse(content)
					if err != nil {
						continue
					}
					analyses = append(analyses, a)
					filePaths = append(filePaths, r.Path)
				}
			}
			if len(analyses) > 1 {
				if debugFlag {
					fmt.Fprintf(os.Stderr, "[DEBUG] Running cross-file consistency on %d files\n", len(analyses))
				}
				crossFindings, err := analyzer.AnalyzeCrossFile(analyses, filePaths, provider)
				if err != nil {
					if debugFlag {
						fmt.Fprintf(os.Stderr, "[DEBUG] Cross-file error: %v\n", err)
					}
				} else if len(crossFindings) > 0 {
					doctor.RenderConsistency(crossFindings, tty, reg)
					totalFindings += len(crossFindings)
				}
			}
		}
	}

	// Aggregate and render summary
	report := doctor.Aggregate(results)
	doctor.RenderAggregate(report, tty)

	if totalFindings > 0 {
		return ErrFindings
	}
	return nil
}

// analyzeFile runs the full analysis pipeline on a single file.
// Returns findings, detected format name, and any error.
func analyzeFile(filePath, providerFlag, modelFlag string, debugFlag, noCacheFlag bool, reg *registry.Registry) ([]llmutil.Finding, string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, "", fmt.Errorf("cannot read %s: %w", filePath, err)
	}

	if debugFlag {
		fmt.Fprintf(os.Stderr, "[DEBUG] Registered parsers: %v\n", parser.RegisteredFormats())
	}

	if debugFlag {
		for _, name := range parser.RegisteredFormats() {
			pp, _ := parser.Get(name)
			if pp != nil {
				detected := pp.Detect(filePath, content)
				fmt.Fprintf(os.Stderr, "[DEBUG] Parser %s.Detect(%s) = %v\n", name, filePath, detected)
			}
		}
	}

	p, err := parser.DetectFormat(filePath, content)
	if err != nil {
		return nil, "", fmt.Errorf("cannot detect format of %s: %w\nSupported formats: Claude Code, GitHub Copilot, Forgent YAML", filePath, err)
	}

	if debugFlag {
		fmt.Fprintf(os.Stderr, "[DEBUG] Detected format: %s\n", p.Format())
	}

	analysis, err := p.Parse(content)
	if err != nil {
		return nil, p.Format(), fmt.Errorf("parse error in %s: %w", filePath, err)
	}

	if debugFlag {
		fmt.Fprintf(os.Stderr, "[DEBUG] Parsed: %d sections, %d tools, %d frontmatter fields\n",
			len(analysis.Sections), len(analysis.Tools), len(analysis.Frontmatter))
	}

	// Select provider
	provider, providerName, err := selectProvider(providerFlag, modelFlag, debugFlag)
	if err != nil {
		// No LLM available — static analysis + context enrichment only
		if debugFlag {
			fmt.Fprintf(os.Stderr, "[DEBUG] No LLM: %s\n", err)
		}
		var findings []llmutil.Finding

		// Context enrichment (includes text ordering check)
		projectRoot := doctorctx.DetectProjectRoot(filepath.Dir(filePath))
		ctxFindings, _ := doctorctx.Enrich(analysis, projectRoot)
		findings = append(findings, ctxFindings...)

		// Show warnings
		tty := doctor.IsTTY()
		for _, w := range analysis.Warnings {
			if tty {
				fmt.Printf("%s %s\n", color.YellowString("[WARN]"), w)
			} else {
				fmt.Printf("WARN: %s\n", w)
			}
		}

		return findings, p.Format(), nil
	}

	if debugFlag {
		fmt.Fprintf(os.Stderr, "[DEBUG] Using provider: %s\n", providerName)
	}

	// Cache setup — use file's directory as fallback if Getwd fails
	cwd, err := os.Getwd()
	if err != nil {
		cwd = filepath.Dir(filePath)
	}
	cacheDir := filepath.Join(cwd, ".forgent", "cache", "doctor")
	cache := doctor.NewCache(cacheDir)
	// Use canonical provider name for cache key (strip "(auto-detected)" etc.)
	cacheProviderName := canonicalProvider(providerName)
	cacheKey := doctor.CacheKey(content, reg.PromptHash(), cacheProviderName, reg.Version)

	// Check cache before LLM call
	if !noCacheFlag {
		if cached := cache.Get(cacheKey); cached != nil {
			if debugFlag {
				fmt.Fprintf(os.Stderr, "[DEBUG] Cache hit for %s (key: %s)\n", filePath, cacheKey)
			}
			findings := cached

			// Still run context enrichment on cached results
			projectRoot := doctorctx.DetectProjectRoot(filepath.Dir(filePath))
			ctxFindings, _ := doctorctx.Enrich(analysis, projectRoot)
			findings = append(findings, ctxFindings...)

			return findings, p.Format(), nil
		}
		if debugFlag {
			fmt.Fprintf(os.Stderr, "[DEBUG] Cache miss for %s (key: %s)\n", filePath, cacheKey)
		}
	}

	// Analyze with LLM
	findings, err := analyzer.Analyze(analysis, provider, reg)
	if err != nil {
		return nil, p.Format(), fmt.Errorf("analysis failed: %w", err)
	}

	// Store in cache
	cacheEntry := doctor.CacheEntry{
		ContentHash:     fmt.Sprintf("%x", sha256.Sum256(content))[:16],
		PromptHash:      reg.PromptHash(),
		Model:           providerName,
		RegistryVersion: reg.Version,
		Timestamp:       time.Now(),
		Findings:        findings,
	}
	if err := cache.Put(cacheKey, cacheEntry); err != nil && debugFlag {
		fmt.Fprintf(os.Stderr, "[DEBUG] Cache write error: %v\n", err)
	}

	// Context enrichment (includes text ordering check)
	projectRoot := doctorctx.DetectProjectRoot(filepath.Dir(filePath))
	if debugFlag {
		if projectRoot != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG] Project root: %s\n", projectRoot)
		} else {
			fmt.Fprintf(os.Stderr, "[DEBUG] No project root detected — skipping context enrichment\n")
		}
	}
	ctxFindings, ctxErr := doctorctx.Enrich(analysis, projectRoot)
	if ctxErr != nil && debugFlag {
		fmt.Fprintf(os.Stderr, "[DEBUG] Context enrichment error: %v\n", ctxErr)
	}
	if debugFlag {
		fmt.Fprintf(os.Stderr, "[DEBUG] Context findings: %d\n", len(ctxFindings))
	}
	findings = append(findings, ctxFindings...)

	// Show warnings
	tty := doctor.IsTTY()
	for _, w := range analysis.Warnings {
		if tty {
			fmt.Printf("%s %s\n", color.YellowString("[WARN]"), w)
		} else {
			fmt.Printf("WARN: %s\n", w)
		}
	}

	return findings, p.Format(), nil
}

// runFixMode analyzes a file, prompts the user to select recommendations,
// rewrites the file via LLM, validates through the quality gate, and applies via SafeWrite.
func runFixMode(filePath, providerFlag, modelFlag string, debugFlag, forceFlag, noCacheFlag bool, reg *registry.Registry) error {
	// Analyze
	findings, format, err := analyzeFile(filePath, providerFlag, modelFlag, debugFlag, noCacheFlag, reg)
	if err != nil {
		return err
	}

	tty := doctor.IsTTY()
	doctor.RenderFindings(findings, filePath, tty, reg)

	if len(findings) == 0 {
		return nil
	}

	// Need an LLM provider for rewrite
	provider, _, err := selectProvider(providerFlag, modelFlag, debugFlag)
	if err != nil {
		return fmt.Errorf("--fix requires an LLM provider: %w", err)
	}

	// Interactive selection
	selected, err := rewrite.SelectRecommendations(findings, tty, nil)
	if err != nil {
		return fmt.Errorf("selection: %w", err)
	}
	if len(selected) == 0 {
		fmt.Fprintf(os.Stderr, "No recommendations selected.\n")
		return nil
	}

	// Read original content
	original, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	// Rewrite via LLM
	fmt.Fprintf(os.Stderr, "\nRewriting with %d selected recommendation(s)...\n", len(selected))
	rewritten, err := rewrite.Rewrite(original, selected, format, provider)
	if err != nil {
		return fmt.Errorf("rewrite: %w", err)
	}

	// Show diff
	diff := rewrite.GenerateDiff(original, rewritten, tty)
	if diff != "" {
		fmt.Fprintf(os.Stderr, "\n%s\n", diff)
	} else {
		fmt.Fprintf(os.Stderr, "No changes produced by rewrite.\n")
		return nil
	}

	// Quality gate
	gateResult := validate.Validate(original, rewritten, validate.GateOptions{Force: forceFlag})
	if !gateResult.Passed {
		fmt.Fprintf(os.Stderr, "\nQuality gate FAILED — file NOT modified:\n")
		for _, f := range gateResult.Failures {
			fmt.Fprintf(os.Stderr, "  [%s] %s\n", f.Check, f.Detail)
		}
		fmt.Fprintf(os.Stderr, "\nTry applying recommendations individually or use --force to skip diff-size check.\n")
		return fmt.Errorf("quality gate rejected rewrite (%d failures)", len(gateResult.Failures))
	}

	// SafeWrite with signal handling
	ctx, cancel := fileops.WithCleanup(context.Background())
	defer cancel()

	if err := fileops.SafeWrite(ctx, filePath, rewritten); err != nil {
		return fmt.Errorf("safe write: %w", err)
	}

	// Purge cache for this file (content changed)
	fixCwd, cwdErr := os.Getwd()
	if cwdErr != nil {
		fixCwd = filepath.Dir(filePath)
	}
	fixCache := doctor.NewCache(filepath.Join(fixCwd, ".forgent", "cache", "doctor"))
	contentHash := fmt.Sprintf("%x", sha256.Sum256(original))[:16]
	fixCache.Purge(contentHash)

	if tty {
		color.Green("\nFile updated: %s (backup: %s.bak)", filePath, filePath)
	} else {
		fmt.Printf("File updated: %s (backup: %s.bak)\n", filePath, filePath)
	}

	// Cleanup old backups
	dir := filepath.Dir(filePath)
	if cleaned, _ := fileops.CleanupBackups(dir, fileops.DefaultBackupMaxAge); cleaned > 0 && debugFlag {
		fmt.Fprintf(os.Stderr, "[DEBUG] Cleaned %d old backup(s)\n", cleaned)
	}

	return nil
}

// selectProvider chooses an LLM provider using the fallback chain:
// 1. Explicit --provider flag
// 2. Ollama (if running locally)
// 3. OPENROUTER_API_KEY env var
// 4. ANTHROPIC_API_KEY env var
// 5. Error (no provider available)
func selectProvider(providerFlag, modelFlag string, debug bool) (llm.Provider, string, error) {
	if providerFlag != "" {
		apiKey := ""
		switch providerFlag {
		case "ollama":
			// No API key needed
		case "openrouter":
			apiKey = os.Getenv("OPENROUTER_API_KEY")
			if apiKey == "" {
				return nil, "", fmt.Errorf("OPENROUTER_API_KEY not set")
			}
		case "anthropic":
			apiKey = os.Getenv("ANTHROPIC_API_KEY")
			if apiKey == "" {
				return nil, "", fmt.Errorf("ANTHROPIC_API_KEY not set")
			}
		default:
			return nil, "", fmt.Errorf("unknown provider %q (available: ollama, openrouter, anthropic)", providerFlag)
		}
		p, err := llm.GetProvider(providerFlag, apiKey)
		if err != nil {
			return nil, "", err
		}
		return p, providerFlag, nil
	}

	if ollamaRunning(debug) {
		p, err := llm.GetProvider("ollama", "")
		if err == nil {
			return p, "ollama (auto-detected)", nil
		}
	}

	if key := os.Getenv("OPENROUTER_API_KEY"); key != "" {
		p, err := llm.GetProvider("openrouter", key)
		if err == nil {
			return p, "openrouter (via OPENROUTER_API_KEY)", nil
		}
	}

	if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		p, err := llm.GetProvider("anthropic", key)
		if err == nil {
			return p, "anthropic (via ANTHROPIC_API_KEY)", nil
		}
	}

	return nil, "", fmt.Errorf("no LLM provider available. Install Ollama or set OPENROUTER_API_KEY / ANTHROPIC_API_KEY")
}

func ollamaRunning(debug bool) bool {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:11434/api/tags")
	if err != nil {
		if debug {
			fmt.Fprintf(os.Stderr, "[DEBUG] Ollama not detected: %v\n", err)
		}
		return false
	}
	defer resp.Body.Close()

	if debug {
		var tags struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}
		if json.NewDecoder(resp.Body).Decode(&tags) == nil {
			var names []string
			for _, m := range tags.Models {
				names = append(names, m.Name)
			}
			fmt.Fprintf(os.Stderr, "[DEBUG] Ollama running, models: %v\n", names)
		}
	}

	return resp.StatusCode == http.StatusOK
}

func renderStaticAnalysis(analysis *parser.AgentAnalysis, filePath string) {
	tty := doctor.IsTTY()

	if tty {
		bold := color.New(color.Bold)
		bold.Printf("\nStatic analysis for %s (format: %s):\n\n", filePath, analysis.Format)
	} else {
		fmt.Printf("Static analysis for %s (format: %s):\n", filePath, analysis.Format)
	}

	fmt.Printf("  Sections: %d\n", len(analysis.Sections))
	fmt.Printf("  Tools: %d", len(analysis.Tools))
	if len(analysis.Tools) > 0 {
		fmt.Printf(" (%s)", joinMax(analysis.Tools, 5))
	}
	fmt.Println()
	fmt.Printf("  Frontmatter fields: %d\n", len(analysis.Frontmatter))

	for _, w := range analysis.Warnings {
		if tty {
			fmt.Printf("  %s %s\n", color.YellowString("[WARN]"), w)
		} else {
			fmt.Printf("  WARN: %s\n", w)
		}
	}

	fmt.Println("\n  For full LLM-powered analysis, install Ollama: https://ollama.com")
}

// canonicalProvider extracts the short provider name from display names
// like "ollama (auto-detected)" or "openrouter (via OPENROUTER_API_KEY)".
func canonicalProvider(name string) string {
	if idx := strings.Index(name, " ("); idx > 0 {
		return name[:idx]
	}
	return name
}

func joinMax(items []string, max int) string {
	if len(items) <= max {
		return fmt.Sprintf("%v", items)
	}
	return fmt.Sprintf("%v and %d more", items[:max], len(items)-max)
}
