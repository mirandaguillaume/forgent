# Forgent Architecture

> Generated: 2026-03-27 | Type: CLI monolith | Language: Go 1.24

## Executive Summary

Forgent is a skill compiler that translates declarative agent skill definitions (YAML) into framework-native formats. It follows a pipeline architecture: **parse → validate → analyze → enrich → generate**. The project is organized as a standard Go CLI with public packages (`pkg/`) exposing reusable domain types and engines, and private packages (`internal/`) implementing CLI commands, generators, and tooling.

## Technology Stack

| Category | Technology | Version | Purpose |
|----------|-----------|---------|---------|
| Language | Go | 1.24.2 | Primary language |
| CLI Framework | Cobra | 1.10.2 | Command parsing and help |
| YAML | gopkg.in/yaml.v3 | 3.0.1 | Skill/agent definition parsing |
| Testing | testify | 1.11.1 | Assertions and test suites |
| Output | fatih/color | 1.18.0 | Colored terminal output |
| File Watching | fsnotify | 1.9.0 | Live rebuild on changes |
| AST Parsing | gotreesitter | 0.9.2 | Multi-language symbol extraction |
| Documentation | Hugo + Hextra | 0.12.1 | Project website |
| CI/CD | GitHub Actions | — | Test, build, deploy |
| Releases | release-please + goreleaser | — | Automated versioning and binary distribution |

## Architecture Pattern

**Pipeline with Plugin Registry** — the build flow is a linear pipeline, but code generators are pluggable via a factory registry pattern (`pkg/spec`). Each generator registers itself in `init()` and is instantiated on demand.

```
┌─────────┐    ┌────────┐    ┌──────────┐    ┌─────────┐    ┌───────────┐
│  Parse   │───▶│  Lint  │───▶│ Analyze  │───▶│ Enrich  │───▶│ Generate  │
│  (YAML)  │    │ (Rules)│    │(Deps/Score)│   │(Scanner)│    │(Target)   │
└─────────┘    └────────┘    └──────────┘    └─────────┘    └───────────┘
     ▲                                                           │
     │                                                    ┌──────┴──────┐
  .skill.yaml                                             │   Target    │
  .agent.yaml                                             │  Registry   │
                                                          ├─────────────┤
                                                          │ claude      │
                                                          │ copilot     │
                                                          │ forgent     │
                                                          └─────────────┘
```

## Core Domain Model

### SkillBehavior (pkg/model/skill.go)

The fundamental unit. A skill is a pure interface with 5 behavioral facets:

| Facet | Responsibility | Key Fields |
|-------|---------------|------------|
| **Context** | I/O contract | consumes, produces, memory type |
| **Strategy** | Execution plan | tools, approach, steps, effort level |
| **Guardrails** | Constraints | rules (string or map), limits |
| **Observability** | Monitoring | trace level, metrics list |
| **Security** | Permissions | filesystem, network, secrets, sandbox |

Additional facets: Negotiation (conflict handling), WhenToUse (triggers), AntiPatterns, Examples.

### AgentComposition (pkg/model/agent.go)

Composes skills into an orchestrated workflow:

- **Flat mode**: `skills: [a, b, c]` with a single orchestration strategy
- **Staged mode**: `stages:` with named groups, each with its own strategy (sequential, parallel, parallel-then-merge, adaptive)
- **I/O contract**: `consumes` (agent inputs) and `produces` (agent outputs)

### Generator Interface (pkg/spec/generator.go)

```go
type Generator interface {
    Target() string
    DefaultOutputDir() string
}

type SkillGenerator interface {
    GenerateSkill(skill model.SkillBehavior) string
    SkillPath(name string) string
}

type AgentGenerator interface {
    GenerateAgent(agent model.AgentComposition, skills []model.SkillBehavior, outputDir string) string
    AgentPath(name string) string
}
```

Optional interfaces: `InstructionsGenerator` (Copilot-only), `Configurable` (compact mode, contracts).

## Package Dependencies

```
cmd/forgent
  └── internal/cmd
        ├── internal/builder
        │     ├── internal/yaml       (parse)
        │     ├── internal/linter     (validate)
        │     ├── internal/scanner    (codebase context)
        │     ├── internal/enricher   (context injection)
        │     └── internal/generator  (code gen)
        │           ├── claude/
        │           ├── copilot/
        │           └── forgent/
        ├── internal/analyzer         (deps, loops, scores)
        └── internal/importer         (agent .md import)
              └── internal/llm        (Anthropic, OpenRouter)

pkg/model      ← used by all internal packages
pkg/spec       ← generator registry
pkg/dag        ← standalone execution engine
pkg/analysis   ← diff analysis pipeline
```

## Build Targets

### Claude Code (`--target claude`)
- **Output**: `.claude/skills/<name>/SKILL.md`, `.claude/agents/<name>.md`
- **Format**: Markdown with YAML frontmatter (name, description, tools)
- **Design**: Guardrails placed first (primacy bias), tool names mapped to Claude-specific tools
- **Compact mode**: Single-file agent with all skills inlined as 4-line terse blocks

### GitHub Copilot (`--target copilot`)
- **Output**: `.github/skills/<name>/SKILL.md`, `.github/agents/<name>.agent.md`, `.github/copilot-instructions.md`
- **Format**: Markdown with 1024-char description limit, JSON array tool syntax
- **Design**: Explicit when-to-use sections, master instructions file aggregating all skills/agents

### Forgent Runtime (`--target forgent`)
- **Output**: `.forgent/<name>/main.go` + `go.mod` + `go.sum`
- **Format**: Executable Go binary using `pkg/dag` for orchestration
- **Design**: Auto-wired DAG from skill consumes/produces, LLM provider abstraction (Anthropic + OpenRouter), CLI flag-based input discovery

## DAG Engine (pkg/dag)

Standalone execution engine with no domain dependencies:

- **Auto-wiring**: if node A produces type T and node B consumes T, edge A→B is created automatically
- **12 node kinds**: task, router, merge, map, filter, flatmap, reduce, batch, zip, gate, fallback, race, cache
- **Scheduling**: Layer-based (acyclic) or event-driven (cyclic with bounded back-edges)
- **Concurrency**: Configurable via `WithConcurrency(n)`, thread-safe data store
- **Validation**: 6 named topologies (pipeline, fan-out, fan-in, diamond, wide-parallel, hourglass)

## Analysis Pipeline (pkg/analysis)

Used by generated Forgent runtime agents for code review:

1. **ParseDiff** — unified diff → structured files with language detection
2. **ParseAST** — tree-sitter multi-language symbol extraction (Go, Python, Java, TypeScript, JavaScript, Ruby)
3. **ScanPatterns** — heuristic detection: null safety, type errors, logic bugs, concurrency, resource handling, security
4. **Render** — markdown output for LLM consumption

## Codebase Scanner (internal/scanner)

Intelligent codebase context generation for skill enrichment:

- **Neyman proportional allocation** — distributes directory budget across top-level strata
- **Shannon entropy** — adapts file display count based on file-type diversity
- **TF-IDF ranking** — identifies uniquely informative directories
- **Symbol extraction** — Go AST parsing for exported types/functions

Two enrichment modes:
- **Index mode**: Separate context files with pointers in SKILL.md
- **Inline mode**: Context embedded directly in generated SKILL.md

## Import Pipeline (internal/importer)

Converts existing agent definitions into Forgent skill YAML:

1. **Source resolution**: file path, directory, URL, or `vercel:<name>`
2. **Framework detection**: Claude (`.claude/`) or Copilot (`.github/`)
3. **Frontmatter extraction**: YAML metadata from agent .md files
4. **LLM decomposition**: Anthropic/OpenRouter decomposes monolithic agents into skills
5. **Reverse tool mapping**: Framework-specific → generic tool names
6. **Quality scoring**: Generated skills scored (0-100), retry if below threshold

## Benchmark Framework (internal/bench)

Internal tooling for evaluating Forgent-generated agents:

| Benchmark | What it measures |
|-----------|-----------------|
| **Proxy reachability** | Index coverage of codebase files |
| **Agent navigation** | Claude CLI task completion accuracy |
| **Consistency** | Response stability (pass@k) |
| **Determinism** | Build output reproducibility |
| **H2H evaluation** | Code review quality (severity-weighted scoring) |
| **Multi-pass** | Iterative review refinement |
| **SWE-bench** | Software engineering task resolution |
| **Isomorphism** | Cross-target output equivalence (Claude vs Copilot) |
| **Formal properties** | DAG correctness (P10-P15 from whitepaper) |

## Testing Strategy

- **Unit tests**: All packages have `*_test.go` files with testify assertions
- **Integration tests**: `internal/importer/integration_test.go`, `internal/bench/*_test.go`
- **CI**: `go vet ./...` + `go test ./... -count=1 -race` on every PR
- **Test pattern**: Table-driven tests, temp directory fixtures
- **Coverage areas**: Model validation, generator output, DAG execution, linter rules, import pipeline
