# Source Tree Analysis

> Generated: 2026-03-27 | Scan Level: exhaustive

## Project Root

```
forgent/
├── cmd/                          # Binary entry points
│   ├── forgent/
│   │   └── main.go               # CLI entry point → internal/cmd.Execute()
│   └── forgent-bench/
│       └── main.go               # Benchmark CLI (internal tooling)
│
├── pkg/                          # Public API packages (importable)
│   ├── model/                    # Skill & agent domain types
│   │   ├── skill.go              # SkillBehavior, all 5 facets (Context, Strategy, Guardrails, Observability, Security)
│   │   ├── agent.go              # AgentComposition, Stage, OrchestrationStrategy
│   │   ├── validate.go           # ValidateSkill(), ValidateAgent()
│   │   └── guardrail_checker.go  # GuardrailChecker interface
│   ├── spec/                     # Generator plugin registry
│   │   └── generator.go          # Generator/SkillGenerator/AgentGenerator interfaces, Register(), Get()
│   ├── dag/                      # Executable DAG engine
│   │   ├── dag.go                # DAG struct, New() with auto-wiring, AddEdge with cycle detection
│   │   ├── node_kinds.go         # RunFunc, NodeConfig, 12 node kinds (task/router/merge/map/filter/reduce/...)
│   │   ├── edge.go               # Edge, EdgeKey, back-edge support
│   │   ├── layers.go             # Layers() — Kahn's + longest-path for parallel scheduling
│   │   ├── execute.go            # Execute() — layer-based or event-driven scheduler, retry, 12 kind handlers
│   │   ├── datastore.go          # Thread-safe key-value inter-node data passing
│   │   └── validate.go           # HasCycle(), ValidateTopology(), FindCycles(), 6 named topologies
│   └── analysis/                 # Diff analysis pipeline
│       ├── analysis.go           # Analyze() — top-level orchestrator
│       ├── diff.go               # ParseDiff() — unified diff parser, language detection
│       ├── ast.go                # ParseAST() — tree-sitter multi-language symbol extraction
│       ├── patterns.go           # ScanPatterns() — heuristic defect/security detection
│       └── render.go             # Render() — markdown output for LLM consumption
│
├── internal/                     # Private implementation packages
│   ├── cmd/                      # Cobra CLI command handlers
│   │   ├── root.go               # Root command, version
│   │   ├── init.go               # `forgent init` — project scaffolding
│   │   ├── lint.go               # `forgent lint` — skill linting
│   │   ├── doctor.go             # `forgent doctor` — full diagnostics + health score
│   │   ├── score.go              # `forgent score` — design quality scoring
│   │   ├── skill_create.go       # `forgent skill create` — YAML template generation
│   │   ├── build.go              # `forgent build` — code generation orchestration
│   │   ├── watch.go              # `forgent build --watch` — fsnotify live reload
│   │   └── import.go             # `forgent import` — agent .md → skill YAML conversion
│   │
│   ├── builder/                  # Build orchestration
│   │   └── builder.go            # RunBuild() pipeline: parse → lint → scan → generate
│   │
│   ├── linter/                   # Lint rule engine
│   │   ├── registry.go           # Rule interface, Register(), LintSkill()
│   │   ├── rules.go              # Core rules (no-empty-tools, has-guardrails)
│   │   ├── rules_strategy.go     # Strategy validation
│   │   ├── rules_guardrails.go   # Security-needs-guardrails
│   │   ├── rules_observability.go # Observable-outputs
│   │   └── rules_when_to_use.go  # Has-when-to-use
│   │
│   ├── analyzer/                 # Dependency & quality analysis
│   │   ├── dependency.go         # Circular/missing/unmet dependency detection
│   │   ├── loop.go               # Loop risk detection (self-reference, no-timeout)
│   │   ├── score.go              # Multi-facet quality scoring (0-100)
│   │   └── ordering.go           # Skill execution order validation
│   │
│   ├── generator/                # Target-specific code generators
│   │   ├── contracts.go          # Shared contract loading/formatting
│   │   ├── util.go               # Shared formatting utilities
│   │   ├── claude/               # Claude Code target
│   │   │   ├── claude.go         # claudeGenerator — registers "claude" target
│   │   │   ├── skill.go          # GenerateSkillMd() — SKILL.md with frontmatter
│   │   │   ├── agent.go          # GenerateAgentMd() — multi-file & compact modes
│   │   │   ├── toolmap.go        # Generic → Claude tool mapping
│   │   │   └── effort.go         # Effort → model mapping (haiku/sonnet/opus)
│   │   ├── copilot/              # GitHub Copilot target
│   │   │   ├── copilot.go        # copilotGenerator — registers "copilot" target
│   │   │   ├── skill.go          # GenerateCopilotSkillMd() — 1024-char limit
│   │   │   ├── agent.go          # GenerateCopilotAgentMd() — .agent.md format
│   │   │   ├── instructions.go   # copilot-instructions.md generation
│   │   │   ├── toolmap.go        # Generic → Copilot tool mapping
│   │   │   └── effort.go         # Effort → model mapping
│   │   └── forgent/              # Standalone Go runtime target
│   │       ├── forgent.go        # forgentGenerator — registers "forgent" target
│   │       ├── agent.go          # GenerateAgentGo() — executable main.go with DAG
│   │       └── prompt.go         # BuildPromptTemplate() — runtime prompt construction
│   │
│   ├── scanner/                  # Codebase scanner
│   │   ├── scan.go               # ScanCodebase() — filesystem walk, dependency extraction
│   │   ├── symbols.go            # ExtractSymbols() — Go AST symbol extraction
│   │   ├── stratify.go           # Neyman proportional allocation for directory budgets
│   │   ├── entropy.go            # Shannon entropy for adaptive file caps
│   │   ├── tfidf.go              # TF-IDF directory importance ranking
│   │   └── types.go              # EnrichMode, CodebaseContext types
│   │
│   ├── enricher/                 # Codebase context injection
│   │   └── enricher.go           # RenderIndex/Inline/Pointer modes for skill enrichment
│   │
│   ├── importer/                 # Agent .md import pipeline
│   │   ├── source.go             # Source resolution (file/dir/URL/Vercel)
│   │   ├── frontmatter.go        # YAML frontmatter extraction
│   │   └── toolmap.go            # Reverse tool mapping (framework → generic)
│   │
│   ├── llm/                      # LLM provider abstraction
│   │   ├── provider.go           # Provider interface (Complete method)
│   │   ├── registry.go           # Provider factory registry
│   │   ├── anthropic.go          # Anthropic Messages API implementation
│   │   └── openrouter.go         # OpenRouter API implementation
│   │
│   ├── yaml/                     # YAML parsing
│   │   └── loader.go             # ParseSkillYAML(), ParseAgentYAML()
│   │
│   ├── templates/                # Embedded templates
│   │   └── embed.go              # SkillTemplate(), AgentTemplate()
│   │
│   └── bench/                    # Benchmark framework (internal tooling)
│       ├── bench.go              # Core benchmark types, Claude CLI integration
│       ├── agent.go              # Agent navigation benchmark
│       ├── proxy.go              # Index reachability benchmark
│       ├── consistency.go        # Response consistency (pass@k)
│       ├── determinism.go        # Build reproducibility
│       ├── eval.go               # LLM-as-judge evaluation
│       ├── clauderun.go          # SWE-bench integration
│       ├── h2h.go                # Head-to-head code review evaluation
│       ├── h2h_multipass.go      # Multi-pass review strategies
│       ├── h2h_judge.go          # Severity-weighted judging
│       ├── martian.go            # Martian golden comment dataset
│       ├── miniswe.go            # mini-SWE-agent integration
│       ├── pipeline.go           # End-to-end import+build pipeline
│       ├── report.go             # Markdown report generation
│       └── formal/
│           └── graph.go          # Formal property analysis (P10-P15)
│
├── skills/                       # Skill YAML definitions (18 skills)
│   ├── bug-scanner.skill.yaml
│   ├── claudemd-auditor.skill.yaml
│   ├── claudemd-discoverer.skill.yaml
│   ├── code-comment-auditor.skill.yaml
│   ├── coverage-reporter.skill.yaml
│   ├── git-history-reviewer.skill.yaml
│   ├── issue-confidence-scorer.skill.yaml
│   ├── pr-commenter.skill.yaml
│   ├── pr-eligibility-checker.skill.yaml
│   ├── pr-history-reviewer.skill.yaml
│   ├── pr-summarizer.skill.yaml
│   ├── pr-triage.skill.yaml
│   ├── review-commenter.skill.yaml
│   ├── risk-scorer.skill.yaml
│   ├── source-fetcher.skill.yaml
│   ├── tdd-runner.skill.yaml
│   ├── ts-linter.skill.yaml
│   └── type-checker.skill.yaml
│
├── agents/                       # Agent YAML definitions (3 agents)
│   ├── code-reviewer.agent.yaml  # Multi-stage PR review pipeline
│   ├── ci-reviewer.agent.yaml    # CI/CD-focused review
│   └── staged-reviewer.agent.yaml # Staged review orchestration
│
├── contracts/                    # Output format contracts (markdown)
│   ├── risk_score.md             # Risk scoring format (0-10)
│   └── review_comments.md        # Structured review comment format
│
├── templates/                    # Scaffold templates
│   ├── skill.yaml                # Skill YAML template
│   └── agent.yaml                # Agent YAML template
│
├── docs/                         # Documentation
│   ├── ROADMAP.md                # Product roadmap (3 phases)
│   └── plans/                    # Design docs & implementation plans (32 files)
│
├── .github/                      # GitHub configuration
│   ├── workflows/
│   │   ├── ci.yaml               # CI: vet + test + build + Hugo site
│   │   └── release.yaml          # Release automation
│   ├── copilot-instructions.md   # Generated Copilot instructions
│   ├── agents/                   # Generated Copilot agents (2)
│   └── skills/                   # Generated Copilot skills (15)
│
├── .claude/                      # Claude Code configuration
│   ├── agents/                   # Generated Claude agents
│   └── skills/                   # Generated Claude skills + BMad skills
│
├── .forgent/                     # Generated Forgent runtime binaries (3)
│   ├── ci_reviewer/
│   ├── code_reviewer/
│   └── staged_reviewer/
│
├── go.mod                        # Go module: github.com/mirandaguillaume/forgent
├── go.sum
├── CLAUDE.md                     # AI agent instructions
├── README.md                     # Project readme
├── CHANGELOG.md                  # Change log
└── LICENSE                       # License
```

## Critical Directories

| Directory | Purpose |
|-----------|---------|
| `cmd/forgent/` | CLI binary entry point |
| `pkg/model/` | Core domain types — SkillBehavior and AgentComposition |
| `pkg/spec/` | Generator plugin interface and registry |
| `pkg/dag/` | Executable DAG engine with 12 node kinds |
| `pkg/analysis/` | Diff analysis pipeline (AST + patterns) |
| `internal/cmd/` | All CLI command implementations |
| `internal/builder/` | Build orchestration pipeline |
| `internal/generator/` | Three target generators (Claude, Copilot, Forgent) |
| `internal/scanner/` | Intelligent codebase scanning with budget allocation |
| `internal/importer/` | Agent .md → skill YAML conversion pipeline |
| `skills/` | Source skill definitions (compiled to all targets) |
| `agents/` | Source agent compositions (compiled to all targets) |
| `contracts/` | Output format specifications shared across skills |

## Entry Points

| Binary | Entry | Purpose |
|--------|-------|---------|
| `forgent` | `cmd/forgent/main.go` | Main CLI tool |
| `forgent-bench` | `cmd/forgent-bench/main.go` | Internal benchmark runner |
