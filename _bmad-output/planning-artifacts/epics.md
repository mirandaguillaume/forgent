---
stepsCompleted:
  - step-01-validate-prerequisites
  - step-02-design-epics
  - step-03-create-stories
  - step-04-final-validation
status: 'complete'
completedAt: '2026-03-30'
inputDocuments:
  - _bmad-output/planning-artifacts/prd.md
  - _bmad-output/planning-artifacts/architecture.md
  - docs/plans/ (31 design documents)
---

# Forgent - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for Forgent, decomposing the requirements from the PRD and Architecture into implementable stories.

## Requirements Inventory

### Functional Requirements

- **FR1:** Developer can run doctor on a Claude Code agent file and receive a structured analysis report
- **FR2:** Developer can run doctor on a GitHub Copilot agent file and receive a structured analysis report
- **FR3:** Developer can run doctor on a Forgent skill YAML file and receive a structured analysis report
- **FR4:** Doctor can detect the agent file format automatically without user specification
- **FR5:** Doctor can analyze agent files using a local LLM (Ollama) without requiring an API key or network access
- **FR6:** Doctor can fall back to a cloud API provider when no local LLM is available
- **FR7:** Doctor adapts its output based on environment — interactive in terminal (TTY), plain text in CI/pipe
- **FR8:** Developer can run doctor on a directory to analyze all agent files within it, producing an aggregate report
- **FR9:** Doctor can enrich recommendations with codebase context when run inside a project directory
- **FR10:** Doctor provides useful static analysis output when neither local LLM nor cloud API is available
- **FR11:** Doctor detects available Ollama models and selects an appropriate one, or prompts the user to pull a recommended model
- **FR12:** Doctor produces 5-8 research-backed recommendations per agent file, each citing a specific published paper
- **FR13:** Each recommendation includes a confidence level (high, moderate, low)
- **FR14:** Each recommendation explains the issue, why it matters (with citation), and the suggested improvement
- **FR15:** Recommendations are sourced from a versioned research registry that can be updated
- **FR16:** Doctor can diagnose the limits of text-based instructions and suggest when structural enforcement would be more appropriate
- **FR17:** Doctor notifies the user when a newer research registry version is available
- **FR18:** Developer can update the research registry via `forgent doctor --update-registry`
- **FR19:** Doctor defaults to read-only analysis — no file modifications without explicit user consent
- **FR20:** Developer can preview the exact diff that each recommendation would produce
- **FR21:** Developer can select which recommendations to apply individually
- **FR22:** Doctor creates an automatic backup before any file modification
- **FR23:** Developer can restore an agent file from backup via `forgent doctor --restore`
- **FR24:** Rewritten Claude files retain all original frontmatter fields, preserve user custom sections, and pass Claude Code's internal parser validation
- **FR25:** Rewritten Copilot files retain all original frontmatter fields, preserve user custom sections, and pass Copilot's parser validation
- **FR26:** Developer can define a skill in YAML with explicit I/O contracts and five behavioral facets
- **FR27:** Developer can define an agent as a composition of skills with orchestration strategy
- **FR28:** Developer can scaffold a new project with `forgent init`
- **FR29:** Developer can create a new skill scaffold with `forgent skill create <name>`
- **FR30:** Developer can lint skills for structural issues
- **FR31:** Developer can score skill quality (0-100) with improvement suggestions
- **FR32:** Developer can validate that all skills in an agent compose correctly
- **FR33:** Skill and agent YAML specs include a version field with compiler-validated compatibility
- **FR34:** Developer can compile skill/agent YAML to Claude Code format
- **FR35:** Developer can compile skill/agent YAML to GitHub Copilot format
- **FR36:** Developer can compile skill/agent YAML to a standalone Go runtime binary
- **FR37:** Developer can compile to multiple targets in a single build command
- **FR38:** Developer can enrich compiled output with structured codebase context
- **FR39:** Developer can watch for file changes and rebuild automatically
- **FR40:** Developer can import existing agent .md files into Forgent skill YAML
- **FR41:** Import pipeline scores generated YAML quality and retries decomposition if below threshold
- **FR42:** Compiler produces identical output when given identical input — deterministic builds
- **FR43:** Developer can pin a target format version so platform updates don't affect output until explicit upgrade
- **FR44:** Compiled Forgent runtime binary executes skill pipelines with deterministic ordering via DAG engine
- **FR45:** Runtime enforces I/O contracts — a skill cannot execute until its consumed types are available
- **FR46:** Runtime supports pluggable LLM providers via flag or config
- **FR47:** Runtime supports router nodes that branch execution based on intermediate results
- **FR48:** Runtime supports retry logic for failed skill executions
- **FR49:** Runtime reports clear error diagnostics on node failure — which node, what input, failure reason
- **FR50 (KEYSTONE):** Doctor itself is defined as a Forgent agent and compiled to the Forgent runtime — proving the platform through the product
- **FR51:** Developer can run InstructionBench to measure recommendation quality across a corpus
- **FR52:** Developer can run a controlled runtime-vs-text experiment with reproducible results
- **FR53:** Contributor can run format fidelity tests for their target generator against a standard test corpus
- **FR54:** Contributor can add a new target generator by implementing `pkg/spec.TargetGenerator` and registering via `init()`
- **FR55:** Developer can list all available target generators via `forgent targets`
- **FR56:** Developer can export doctor's analysis as Forgent skill YAML files (`--export-yaml`)
- **FR57:** Developer can configure doctor defaults via config file or environment variables
- **FR58:** Developer can generate shell completion scripts for Bash, Zsh, Fish, and PowerShell

### Non-Functional Requirements

- **NFR1:** Doctor analysis completes within 30 seconds for typical files on recommended hardware. 90 seconds on CPU-only.
- **NFR2:** Doctor analysis completes within 60 seconds for large files on recommended hardware.
- **NFR3:** Compiler build completes within 5 seconds for ≤ 20 skills and 3 agents.
- **NFR4:** Enrichment pipeline completes within 15 seconds for ≤ 50,000 LOC.
- **NFR5:** CLI startup under 200ms.
- **NFR6:** Binary size under 50MB.
- **NFR7:** Doctor uses < 500MB RAM (excluding Ollama).
- **NFR8:** Zero broken rewrites — 100% valid frontmatter and markdown on ≥ 30 file test corpus per format.
- **NFR9:** Deterministic builds — identical input → identical output via content hash.
- **NFR10:** Backup created BEFORE any file modification.
- **NFR11:** Graceful handling of malformed agent files — partial analysis, no crash.
- **NFR12:** Runtime DAG handles node failures gracefully.
- **NFR13:** CI gate: `go test ./... -count=1 -race`.
- **NFR14:** Upgrades preserve user config and data.
- **NFR15:** Concurrent doctor runs don't corrupt files — file locking.
- **NFR16:** Research registry checksum verification.
- **NFR17:** Backup cleanup after 7 days (configurable).
- **NFR18:** Local-first — content never leaves machine by default.
- **NFR19:** Explicit consent before sending to API.
- **NFR20:** No telemetry without opt-in.
- **NFR21:** No caching of agent content beyond execution.
- **NFR22:** Network request URLs visible before sending.
- **NFR23:** Parses formats from last 6 months of platform releases.
- **NFR24:** Compiled output compatible with current platform versions.
- **NFR25:** Works with any Ollama model — not hardcoded.
- **NFR26:** API fallback works with Anthropic + OpenRouter.
- **NFR27:** Backward-compatible YAML schema — breaking changes require major version.
- **NFR28:** Actionable error messages with suggestions.
- **NFR29:** --help includes usage examples.
- **NFR30:** Consistent exit codes: 0/1/2.
- **NFR31:** Works offline with Ollama.
- **NFR32:** Cross-platform: macOS (arm64/amd64), Linux (arm64/amd64), Windows (amd64).
- **NFR33:** UTF-8 handling — no corruption on non-ASCII content.
- **NFR34:** Ctrl+C clean exit — no orphan processes, no partial writes.
- **NFR35:** LLM timeout: 120s default, configurable.
- **NFR36:** --debug mode with full diagnostic output.
- **NFR37:** ≥ 80% test coverage.
- **NFR38:** Race detector in CI.

### Additional Requirements (Architecture)

- ADR-1: Doctor skills embedded via `//go:embed`, DAG constructed at runtime in single binary
- ADR-2: Four-check quality gate (frontmatter validity, field preservation, markdown structure, section preservation) + diff-size sanity check. Fail-closed.
- ADR-3: Research registry embedded via `//go:embed` with local override at `.forgent/research-registry.yaml`
- ADR-4: 5-skill DAG topology — format-detector(CODE) → analyzer(LLM) ‖ context-enricher(CODE) → recommendation-builder(CODE) → rewriter(LLM, conditional)
- ADR-5: Ollama direct HTTP adapter (`internal/llm/ollama.go`) with actionable error messages for: not running, model missing, malformed output, timeout
- ADR-6: FormatParser interface + registry pattern (`internal/doctor/parser/`). Detect, Parse, Validate, Rewrite methods per format.
- ADR-7: Config precedence: CLI flags > env vars > config file > defaults
- Engineering spike required: test 3+ local models on 20 real agent files before full MVP
- Implementation sequence: Spike → Phase A (analysis) → Phase C (runtime) → Phase D (benchmark) → Phase B (rewrite)
- Context-enricher graceful degradation when no project root detected
- Testdata from real public GitHub repos with source attribution
- New packages: `internal/doctor/`, `internal/fileops/`, `internal/llm/ollama.go`, `internal/bench/instruction.go`

### UX Design Requirements

Not applicable — CLI tool. UX defined in PRD CLI Requirements section and NFRs 28-36 (Developer Experience).

### FR Coverage Map

| FR | Epic | Description |
|---|---|---|
| FR1 | Epic 1 | Doctor analyzes Claude files |
| FR2 | Epic 1 | Doctor analyzes Copilot files |
| FR3 | Epic 1 | Doctor analyzes Forgent YAML files |
| FR4 | Epic 1 | Auto-detect format |
| FR5 | Epic 1 | Local LLM analysis (Ollama) |
| FR6 | Epic 1 | API fallback |
| FR7 | Epic 1 | TTY-aware output |
| FR8 | Epic 2 | Directory analysis |
| FR9 | Epic 2 | Codebase context enrichment |
| FR10 | Epic 1 | Static fallback (no LLM) |
| FR11 | Epic 1 | Ollama model detection |
| FR12 | Epic 1 | 5-8 recommendations |
| FR13 | Epic 1 | Confidence levels |
| FR14 | Epic 1 | Recommendation explanations |
| FR15 | Epic 2 | Versioned research registry |
| FR16 | Epic 2 | Diagnose text instruction limits |
| FR17 | Epic 2 | Registry version notification |
| FR18 | Epic 2 | Registry update command |
| FR19 | Epic 3 | Read-only default |
| FR20 | Epic 3 | Diff preview |
| FR21 | Epic 3 | Granular apply |
| FR22 | Epic 3 | Automatic backup |
| FR23 | Epic 3 | Restore from backup |
| FR24 | Epic 3 | Format-preserving Claude rewrite |
| FR25 | Epic 3 | Format-preserving Copilot rewrite |
| FR26-FR32 | Existing | Skill spec, composition, lint, score |
| FR33 | Deferred | Typed contracts (Growth) |
| FR34-FR42 | Existing | Multi-target compilation |
| FR43 | Deferred | Format version pinning (Growth) |
| FR44 | Epic 4 | DAG execution (existing, exercised) |
| FR45 | Epic 4 | Contract enforcement (existing, exercised) |
| FR46 | Epic 4 | Pluggable providers (existing, exercised) |
| FR47 | Epic 4 | Router nodes (existing, exercised) |
| FR48 | Epic 4 | Retry logic (existing, exercised) |
| FR49 | Epic 4 | Error diagnostics (existing, exercised) |
| FR50 | Epic 4 | KEYSTONE: Doctor as Forgent runtime |
| FR51 | Epic 5 | InstructionBench |
| FR52 | Epic 5 | Runtime-vs-text experiment |
| FR53 | Epic 5 | Format fidelity test suite |
| FR54 | Epic 6 | Target generator plugin |
| FR55 | Epic 6 | List targets command |
| FR56 | Epic 6 | Export to YAML |
| FR57 | Epic 6 | Doctor config |
| FR58 | Epic 6 | Shell completion |

**Coverage:** 58/58 FRs mapped (40 to MVP epics, 10 existing, 2 deferred Growth, 6 existing exercised by Epic 4).

## Epic List

### Epic 1: Core Doctor Analysis
Developer runs `forgent doctor agent.md` and gets actionable recommendations — powered by a local LLM, no API key needed.
**FRs covered:** FR1, FR2, FR3, FR4, FR5, FR6, FR7, FR10, FR11, FR12, FR13, FR14

### Epic 2: Enriched Research-Backed Analysis
Recommendations include codebase context from project scanning, research citations from a versioned registry, confidence levels, and text-instruction-limit diagnosis.
**FRs covered:** FR8, FR9, FR15, FR16, FR17, FR18

### Epic 3: Safe Agent Rewrite
Developer runs `forgent doctor agent.md --fix`, reviews diffs per recommendation, selects which to apply, and agent files are improved in-place with automatic backup and quality gate validation. Zero broken files.
**FRs covered:** FR19, FR20, FR21, FR22, FR23, FR24, FR25

### Epic 4: Doctor as Forgent Runtime Agent
Doctor is reimplemented as a Forgent agent — 5 skill YAMLs, 1 agent YAML, embedded via go:embed, DAG constructed at runtime. Every `forgent doctor` run now exercises the full Forgent stack: spec, compiler, DAG engine, runtime, provider adapter.
**FRs covered:** FR44, FR45, FR46, FR47, FR48, FR49, FR50 (KEYSTONE)

### Epic 5: Benchmark & Evidence
InstructionBench measures recommendation quality across a corpus of real agent files. Runtime-vs-text experiment proves structural enforcement outperforms text instructions. Format fidelity tests validate contributed target generators.
**FRs covered:** FR51, FR52, FR53

### Epic 6: Ecosystem & Developer Experience
Developers can configure doctor defaults, list available targets, export analysis as YAML, and generate shell completion. Contributors have a documented workflow for adding new format targets.
**FRs covered:** FR54, FR55, FR56, FR57, FR58

---

## Epic 1: Core Doctor Analysis

Developer runs `forgent doctor agent.md` and gets actionable recommendations — powered by a local LLM, no API key needed.

### Story 1.1: Engineering Spike — Validate Local LLM Quality

As a **product architect**,
I want to test whether a 7B local model can reliably analyze agent files,
So that I can make an informed decision about the local-first architecture before committing to full implementation.

**Acceptance Criteria:**

**Given** 20 real CLAUDE.md and Copilot agent files scraped from public GitHub repos
**When** I send each file to Ollama (Phi-3, Llama 3, Mistral) with a structured analysis prompt
**Then** the model produces parseable YAML output for ≥ 80% of files
**And** the product architect rates ≥ 60% of findings as valid (not false positives)
**And** analysis completes within 30 seconds per file on recommended hardware
**And** results are documented with model name, accuracy rate, and decision gate outcome

**Given** local model accuracy is below 50% of cloud API baseline
**When** the spike results are reviewed
**Then** the decision gate triggers: pivot to hybrid approach (local for simple checks, cloud for complex analysis) and document the decision

### Story 1.2: Ollama Provider Adapter

As a **developer**,
I want Forgent to communicate directly with my local Ollama instance,
So that I can analyze agent files without an API key or network dependency.

**Acceptance Criteria:**

**Given** Ollama is running locally with a model available
**When** the Ollama adapter sends a completion request
**Then** it receives a response via Ollama's REST API (localhost:11434)
**And** the adapter implements the existing `Provider` interface in `internal/llm/`

**Given** Ollama is not running
**When** the adapter attempts to connect
**Then** it returns an actionable error: "Ollama not running. Start with `ollama serve` or use `--provider anthropic`"

**Given** the requested model is not pulled
**When** the adapter receives a model-not-found error
**Then** it returns: "Model not available. Run `ollama pull <model>` or use `--model <available_model>`"

**Given** an LLM call exceeds 120 seconds
**When** the timeout is reached
**Then** the operation is cancelled with: "Analysis timed out. Try a smaller model or use `--provider anthropic`"

### Story 1.3: Format Parser Registry and Claude Parser

As a **developer**,
I want Forgent to automatically detect and parse my Claude Code agent files,
So that doctor can analyze them without me specifying the format.

**Acceptance Criteria:**

**Given** a `FormatParser` interface with `Detect()`, `Parse()`, `Validate()` methods
**When** a parser is registered via `init()` + `Register()`
**Then** it follows the same plugin pattern as `pkg/spec.TargetGenerator`

**Given** a file at `.claude/agents/code-reviewer.md` with YAML frontmatter
**When** the Claude parser's `Detect()` is called
**Then** it returns true based on path pattern and frontmatter structure

**Given** a valid Claude agent file
**When** the Claude parser's `Parse()` is called
**Then** it extracts: frontmatter fields, section headers, section content, tool references, and overall structure
**And** returns an `AgentAnalysis` struct for the analyzer skill

**Given** ≥ 5 real Claude agent files in `internal/doctor/parser/testdata/claude/`
**When** tests run against all testdata files
**Then** all files are detected and parsed correctly with source attribution in README

### Story 1.4: Copilot and Forgent YAML Parsers

As a **developer**,
I want Forgent to analyze my GitHub Copilot agents and Forgent skill YAML files,
So that doctor works across all formats I use.

**Acceptance Criteria:**

**Given** a file at `.github/agents/code-reviewer.agent.md`
**When** the Copilot parser's `Detect()` is called
**Then** it returns true based on path pattern and frontmatter structure

**Given** a valid Copilot agent file
**When** parsed
**Then** it extracts frontmatter (including 1024-char description limit), sections, and tool references
**And** ≥ 5 real Copilot files in testdata with source attribution

**Given** a file matching `*.skill.yaml` or `*.agent.yaml`
**When** the Forgent parser's `Detect()` is called
**Then** it returns true and parses using existing `internal/yaml/` loader
**And** subsumes the old `forgent doctor` diagnostics (lint + dependency + loop analysis)

### Story 1.5: Analyzer Skill and LLM Utilities

As a **developer**,
I want doctor to analyze my agent file and identify specific issues,
So that I know what to improve with research-backed evidence.

**Acceptance Criteria:**

**Given** a parsed `AgentAnalysis` from any format parser
**When** the analyzer skill sends a structured prompt to the LLM provider
**Then** it returns YAML-formatted findings with: category, issue, confidence, citation_id, current_state, suggested_improvement

**Given** the LLM returns output wrapped in markdown code fences
**When** `llmutil.ExtractYAML()` processes the output
**Then** it strips fences and returns clean YAML parseable by yaml.v3
**And** handles variations: ```yaml, ```yml, ```, or no fences

**Given** the analyzer produces findings
**When** recommendations are built
**Then** each recommendation includes a confidence level (high/moderate/low)
**And** each recommendation explains the issue with a research citation
**And** at least 5 recommendation types are supported (guardrails, security, ordering, decomposition, context)

### Story 1.6: Doctor CLI Command with TTY-Aware Output

As a **developer**,
I want to run `forgent doctor agent.md` and see clear, colored recommendations,
So that I can quickly understand what to improve in my agent files.

**Acceptance Criteria:**

**Given** a developer runs `forgent doctor agent.md` in a terminal
**When** the command executes
**Then** it auto-detects the file format, runs analysis, and displays colored recommendations via fatih/color
**And** each recommendation shows: confidence badge, issue title, explanation, citation, suggested improvement

**Given** the command runs in a CI pipe (no TTY)
**When** output is generated
**Then** it produces plain text without color codes, suitable for log capture
**And** exit code 0 = no findings, 1 = error, 2 = recommendations found

**Given** no Ollama and no API key configured
**When** doctor runs
**Then** it produces static analysis output (format validation, section structure) with a message: "Install Ollama for full analysis"

**Given** Ollama is running but no suitable model is pulled
**When** doctor starts
**Then** it detects available models and either selects one or prompts: "Run `ollama pull phi3` for full analysis"

---

## Epic 2: Enriched Research-Backed Analysis

Recommendations include codebase context from project scanning, research citations from a versioned registry, confidence levels, and text-instruction-limit diagnosis.

### Story 2.1: Research Registry — Embedded with Local Override

As a **developer**,
I want every recommendation to cite a specific published paper,
So that I can trust the suggestions are grounded in evidence, not opinions.

**Acceptance Criteria:**

**Given** a research registry YAML file embedded via `//go:embed` in `internal/doctor/registry/data/`
**When** the registry is loaded at startup
**Then** it provides ~7 entries, each with: id, title, citation, paper, url, finding, confidence, detection_prompt, suggestion_prompt

**Given** a local override file exists at `.forgent/research-registry.yaml`
**When** the registry loads
**Then** the local override takes precedence over the embedded default

**Given** the embedded registry has version "2026.03"
**When** a newer local override with version "2026.06" is present
**Then** doctor uses the newer version and reports which version is active in `--debug` mode

**Given** the registry is loaded
**When** recommendations are generated
**Then** each recommendation's `citation_id` resolves to a registry entry with full citation details

### Story 2.2: Codebase Context Enrichment

As a **developer**,
I want doctor to understand my project when analyzing agent files,
So that recommendations include context-specific suggestions like "your project has 47 test files not referenced by the agent."

**Acceptance Criteria:**

**Given** `forgent doctor agent.md` is run inside a project directory
**When** the context-enricher skill executes
**Then** it uses the existing `internal/scanner/` to scan the codebase (TF-IDF, entropy, AST symbols)
**And** produces context-specific recommendations (e.g., unreferenced test files, missing codebase context)

**Given** `forgent doctor ~/Downloads/agent.md` is run outside any project
**When** the context-enricher detects no project root
**Then** it produces empty results gracefully — no crash, no error
**And** the recommendation-builder proceeds with analysis-only recommendations

**Given** a project with ≤ 50,000 LOC
**When** the context-enricher runs
**Then** it completes within 15 seconds

### Story 2.3: Directory Analysis and Advanced Features

As a **team lead**,
I want to run doctor on a directory and see aggregate results across all agent files,
So that I can identify team-wide patterns.

**Acceptance Criteria:**

**Given** a developer runs `forgent doctor .claude/agents/`
**When** the directory contains multiple agent files
**Then** doctor analyzes each file and produces an aggregate summary: total files, issues by category, files with no guardrails, files with no security

**Given** recommendations are generated
**When** doctor detects the agent relies entirely on text instructions for execution ordering
**Then** it includes a finding: "Text instructions are ignored X% of the time on ordering tasks. Structural enforcement available via Forgent runtime."

**Given** the embedded registry version is older than the latest published version
**When** doctor runs
**Then** it displays a notification: "Research registry update available. Run `forgent doctor --update-registry`"

**Given** a developer runs `forgent doctor --update-registry`
**When** executed
**Then** it downloads the latest registry to `.forgent/research-registry.yaml` with checksum verification
**And** the target URL is displayed before the download begins

---

## Epic 3: Safe Agent Rewrite

Developer runs `forgent doctor agent.md --fix`, reviews diffs per recommendation, selects which to apply, and agent files are improved in-place with automatic backup and quality gate validation. Zero broken files.

### Story 3.1: Defensive File Operations

As a **developer**,
I want automatic backups and safe writes for any file doctor modifies,
So that I can always recover if something goes wrong.

**Acceptance Criteria:**

**Given** the `internal/fileops/` package with `SafeWrite()` and `Restore()` functions
**When** `SafeWrite(path, content)` is called
**Then** it creates a backup at `path.bak`, writes to a temp file, then atomically renames to the target
**And** if any step fails, the original file is preserved

**Given** a developer runs `forgent doctor agent.md --restore`
**When** a `.bak` file exists for the target
**Then** the original is restored from backup

**Given** two concurrent `forgent doctor --fix` runs target the same file
**When** both attempt to write
**Then** file locking prevents corruption — second run waits or fails gracefully

**Given** a user presses Ctrl+C during a `--fix` operation
**When** the interrupt is caught
**Then** no partial write occurs, backup remains intact, clean exit

**Given** backups older than 7 days exist
**When** doctor runs
**Then** old `.bak` files are cleaned up (configurable retention)

### Story 3.2: Quality Gate Validation

As a **developer**,
I want doctor to verify every rewrite before applying it,
So that my agent files are never broken by a bad recommendation.

**Acceptance Criteria:**

**Given** the LLM produces a rewritten agent file
**When** the quality gate validates it
**Then** it checks: (1) frontmatter is valid YAML, (2) all original frontmatter fields are preserved, (3) markdown structure is well-formed, (4) all user custom sections are preserved

**Given** the rewrite changes more than 50% of the file content
**When** the diff-size sanity check runs
**Then** it flags the rewrite as suspicious and requires explicit `--force` to apply

**Given** any quality gate check fails
**When** the validation runs
**Then** the rewrite is REJECTED — original file untouched, error shown with specific failure reason

**Given** a test corpus of ≥ 30 real Claude files and ≥ 30 real Copilot files
**When** rewrites are applied and validated
**Then** 100% of rewritten files pass all quality gate checks

### Story 3.3: Interactive Rewrite Experience

As a **developer**,
I want to preview diffs and choose which recommendations to apply,
So that I stay in control of every change to my agent files.

**Acceptance Criteria:**

**Given** doctor has produced recommendations for a file
**When** the developer runs `forgent doctor agent.md` (no --fix)
**Then** recommendations are displayed read-only — no file modifications

**Given** the developer runs `forgent doctor agent.md --fix`
**When** recommendations are available
**Then** each recommendation shows its diff (the exact changes it would make)
**And** the developer is prompted to select which recommendations to apply

**Given** the developer selects recommendations 1, 3, and 5 but not 2 and 4
**When** the selections are applied
**Then** only selected recommendations are merged into the rewrite
**And** the quality gate validates the merged result before writing
**And** `SafeWrite()` creates backup and writes atomically

**Given** the merged rewrite fails all quality gate checks
**When** the user selected recommendations that conflict
**Then** doctor reports which recommendations conflict, suggests applying them individually, and does NOT write the file

**Given** a Claude agent file with frontmatter and custom sections
**When** recommendations are applied
**Then** the rewritten file retains all original frontmatter fields and preserves custom sections

**Given** the file contains UTF-8 non-ASCII characters or emoji
**When** the rewrite is applied
**Then** no content corruption occurs

---

## Epic 4: Doctor as Forgent Runtime Agent

Doctor is reimplemented as a Forgent agent — 5 skill YAMLs, 1 agent YAML, embedded via go:embed, DAG constructed at runtime. Proves the entire Forgent stack.

### Story 4.1: Doctor Skill and Agent YAML Definitions

As a **product architect**,
I want doctor defined as real Forgent skill and agent YAML files,
So that the doctor specification is the same format as any Forgent agent.

**Acceptance Criteria:**

**Given** 5 doctor skill YAMLs in `skills/doctor/`
**When** each skill is defined
**Then** it has: name, version, context (consumes/produces), strategy (approach: code or llm), guardrails, security
**And** format-detector, context-enricher, recommendation-builder use `approach: code`
**And** analyzer, rewriter use `approach: llm`

**Given** a doctor agent YAML in `agents/doctor.agent.yaml`
**When** defined
**Then** it composes the 5 skills with proper consumes/produces contracts
**And** `forgent lint` validates the doctor agent with zero errors
**And** `forgent score` rates the doctor agent quality

### Story 4.2: Embedded YAML and Runtime DAG Construction

As a **developer**,
I want `forgent doctor` to internally construct and execute a real Forgent DAG,
So that every doctor run proves the Forgent runtime works.

**Acceptance Criteria:**

**Given** doctor skill and agent YAMLs embedded via `//go:embed`
**When** `forgent doctor agent.md` is called
**Then** the Cobra command loads embedded specs, parses them with `internal/yaml/`, and constructs a DAG with `pkg/dag.New()`

**Given** the constructed DAG
**When** auto-wiring resolves consumes/produces
**Then** the topology matches ADR-4:
- Layer 0: format-detector (code)
- Layer 1: analyzer (LLM) ‖ context-enricher (code) — parallel
- Layer 2: recommendation-builder (code)
- Layer 3: rewriter (LLM, conditional on --fix)

**Given** the DAG executes with Ollama
**When** the provider is local
**Then** concurrency is set to 1 (serialize LLM calls)

**Given** a test corpus of ≥ 10 agent files
**When** both the direct Go doctor and the DAG-based doctor analyze each file
**Then** recommendations match exactly: same issues, same categories, same confidence levels, same citations
**And** this equivalence test runs in CI on every commit

### Story 4.3: Runtime Error Handling and Diagnostics

As a **developer**,
I want clear error messages when a doctor DAG node fails,
So that I can diagnose and report issues.

**Acceptance Criteria:**

**Given** a skill node fails during DAG execution
**When** the error propagates
**Then** the runtime reports: which node failed, what input it received, and the failure reason

**Given** the rewriter node is conditional on `--fix`
**When** doctor runs without `--fix`
**Then** Layer 3 is skipped entirely — the DAG terminates after recommendation-builder

**Given** a retry-configured node fails
**When** retries are exhausted
**Then** the error includes: node name, attempt count, all failure reasons

**Given** `--debug` mode
**When** the DAG executes
**Then** each node logs: start time, input data keys, LLM prompt (if applicable), output data keys, duration

---

## Epic 5: Benchmark & Evidence

InstructionBench measures recommendation quality. Runtime-vs-text experiment proves structural enforcement outperforms text instructions. Format fidelity tests validate contributed targets.

### Story 5.1: InstructionBench — Agent File Corpus and Quality Measurement

As a **product architect**,
I want to measure doctor's recommendation quality against a corpus of real agent files,
So that I can prove doctor helps and publish the results.

**Acceptance Criteria:**

**Given** a corpus of ≥ 100 real agent files scraped from public GitHub repos
**When** stored in a dedicated location (internal tooling)
**Then** files are organized by format (Claude, Copilot) with source attribution
**And** only files from repos with permissive licenses (MIT, Apache, unlicensed public) are included
**And** source repo URL is recorded per file

**Given** doctor runs against the corpus
**When** InstructionBench collects results
**Then** it measures: total recommendations, acceptance rate per category, confidence distribution, false positive rate
**And** results are reproducible — same corpus + same config → same measurements

**Given** InstructionBench results
**When** formatted for publication
**Then** output includes: methodology description, corpus statistics, per-category results, overall quality score

### Story 5.2: Runtime-vs-Text Controlled Experiment

As a **product architect**,
I want to prove that Forgent runtime agents outperform text-instruction agents on complex tasks,
So that the core thesis has evidence.

**Acceptance Criteria:**

**Given** a set of complex multi-step tasks (code review, documentation, analysis)
**When** the same tasks are run with: (a) text-instruction agent and (b) Forgent runtime agent
**Then** both use the same LLM model and same tools — the ONLY variable is orchestration method

**Given** experiment results
**When** compared
**Then** metrics include: task completion rate, step ordering accuracy, guardrail adherence, token usage
**And** the delta is reported with statistical significance or confidence interval

**Given** the experiment methodology
**When** documented
**Then** a third party can reproduce the experiment and verify results

### Story 5.3: Format Fidelity Test Suite

As a **contributor**,
I want to validate that my target generator preserves format fidelity,
So that my contributed parser doesn't break user files.

**Acceptance Criteria:**

**Given** a standard test corpus per format (≥ 30 real files each)
**When** a contributor runs `go test ./internal/doctor/parser/...`
**Then** all registered parsers are tested against their testdata

**Given** a new format parser (e.g., Cursor)
**When** the contributor adds testdata and registers via `init()`
**Then** the fidelity test suite automatically includes the new parser
**And** CI validates it on every PR

---

## Epic 6: Runtime Harness — Enforcement by Construction

PIVOT 2026-04-16: Replaces original "Ecosystem & DX" epic (moved to Epic 9). Party mode decision.
Rationale: Experiment results show text beats runtime on guardrails without programmatic enforcement.
The runtime's value is not in "better LLM responses" but in "programmatically preventing bad actions."

The Forgent runtime enforces constraints during LLM execution — not after. Each skill YAML declares its security policy, and the runtime builds a middleware pipeline around each DAG node that intercepts, validates, and constrains the LLM at every step.

### Story 6.1: Middleware Pipeline in DAG Node

As a **runtime developer**,
I want each DAG node's Run function to pass through a configurable middleware chain,
So that enforcement layers (file access, budget, quality, PII) can be composed declaratively.

**Acceptance Criteria:**

**Given** a DAG node with middlewares configured
**When** the node executes
**Then** inputs pass through pre-middlewares, the handler runs, and outputs pass through post-middlewares

**Given** a pre-middleware returns an error
**When** the node executes
**Then** the handler is NOT called, the error propagates, and the DAG retry mechanism applies

**Given** a post-middleware rejects the output
**When** the node has MaxRetry > 0
**Then** the entire chain (including the LLM call) is retried

**Given** no middlewares are configured
**When** the node executes
**Then** behavior is identical to current (zero overhead)

### Story 6.2: File Access Control from Skill YAML

As a **skill author**,
I want to declare file read/write permissions in my skill YAML security section,
So that the runtime prevents the LLM from accessing files outside the declared scope.

**Acceptance Criteria:**

**Given** a skill YAML with `security.file_access.read: ["src/**/*.go"]`
**When** the LLM step tries to read a file outside that glob
**Then** the read is blocked and the LLM receives a "permission denied" feedback

**Given** a skill YAML with `security.file_access.deny: ["**/.env", "**/credentials*"]`
**When** the LLM step tries to read or write a denied file
**Then** the action is blocked regardless of other allow rules

**Given** no `security.file_access` is configured in the skill YAML
**When** the LLM step runs
**Then** the existing SandboxFS behavior applies (or no restriction if no sandbox)

### Story 6.3: Budget Guards (Token Cap + Timeout)

As a **platform operator**,
I want per-step token and time limits declared in guardrails,
So that a runaway LLM step cannot exhaust the API budget or hang indefinitely.

**Acceptance Criteria:**

**Given** a step with `guardrails.max_tokens: 4000`
**When** the LLM response exceeds 4000 tokens
**Then** the response is truncated and a warning is logged

**Given** a step with `guardrails.timeout: 30s`
**When** the LLM call exceeds 30 seconds
**Then** the call is cancelled via context, the step fails, and retry applies if configured

**Given** no budget guardrails are configured
**When** the step runs
**Then** default limits apply (provider defaults)

### Story 6.4: Quality Gate Inline (Keywords + LLM Validator)

As a **skill author**,
I want expected_keywords enforced inline during step execution with automatic retry,
So that a step that drifts from its purpose is caught and retried before polluting downstream steps.

**Acceptance Criteria:**

**Given** a step with `expected_keywords: [vulnerability, remediation]`
**When** the LLM output does not contain all expected keywords
**Then** the step fails with a descriptive error and the DAG retries it

**Given** a step with `strategy.validation: haiku` (LLM validator)
**When** the LLM output passes keyword check
**Then** a lightweight LLM call validates semantic correctness (PASS/FAIL)

**Given** a step with `strategy.validation_mode: keywords` (L1 only)
**When** keywords are present
**Then** no LLM validation call is made (cost savings)

### Story 6.5: PII/Secrets Output Filter

As a **security-conscious operator**,
I want LLM outputs scanned for PII and secrets before being passed to the next step,
So that sensitive data cannot leak through the agent pipeline.

**Acceptance Criteria:**

**Given** an LLM step produces output containing a pattern matching a secret (API key, password assignment)
**When** the output passes through the PII filter middleware
**Then** the finding is logged, the output is redacted or the step fails (configurable)

**Given** a skill YAML with `security.output_filter: strict`
**When** a secret pattern is detected in output
**Then** the step fails and retries (giving the LLM a chance to produce clean output)

**Given** a skill YAML with `security.output_filter: redact`
**When** a secret pattern is detected
**Then** the pattern is replaced with `[REDACTED]` and the output continues downstream

### Story 6.6: Harness Experiment — Closed-Loop Control Proof

As a **product architect**,
I want to run a 4-condition experiment (model × harness matrix) proving that cheap models with harness match expensive models without,
So that we validate the core thesis: closed-loop control makes cheap LLMs reliable.

**Acceptance Criteria:**

**Given** the 4-condition matrix:
| Condition | Model | Harness |
|-----------|-------|---------|
| A | Sonnet | OFF (text only) |
| B | Haiku | OFF (text only) |
| C | Haiku | ON (middleware + quality gate + retry) |
| D | Sonnet | ON (harness) |

**When** the same 5 experiment tasks run under all 4 conditions (3 runs each)
**Then** condition C (Haiku+harness) achieves guardrail adherence within 5% of condition A (Sonnet alone)

**Given** the cost data (tokens × model pricing)
**When** compared across conditions
**Then** condition C costs less than 50% of condition A for equivalent quality

**Given** the experiment report
**When** compared to v3 baseline (Haiku text 93% vs Haiku runtime 78%)
**Then** the harness-enabled runtime surpasses both baselines

---

## Epic 9: Ecosystem & Developer Experience (deferred from original Epic 6)

Developers can configure doctor, contributors can add new format targets, and the CLI provides polished DX with shell completion, target listing, and export capabilities.

### Story 9.1: Doctor Configuration (was 6.1)

As a **developer**,
I want to configure doctor defaults so I don't have to pass flags every time,
So that my preferred provider, model, and thresholds are remembered.

**Acceptance Criteria:**

**Given** a config file at `.forgent/config.yaml`
**When** it contains doctor settings (provider, model, confidence_threshold, registry_version)
**Then** doctor uses these as defaults, overridable by CLI flags and env vars

**Given** `FORGENT_PROVIDER=anthropic` is set as an environment variable
**When** doctor runs without `--provider` flag
**Then** it uses the env var, overriding the config file default

**Given** `--provider ollama` is passed as a CLI flag
**When** doctor runs
**Then** the flag overrides both env var and config file

### Story 6.2: Target Listing and Export

As a **developer**,
I want to see what format targets are available and export doctor's analysis as YAML,
So that I can discover capabilities and bridge into the compiler when ready.

**Acceptance Criteria:**

**Given** a developer runs `forgent targets`
**When** executed
**Then** it lists all registered target generators and format parsers
**And** contributed targets are included if registered

**Given** a developer runs `forgent doctor agent.md --export-yaml`
**When** analysis is complete
**Then** doctor outputs the internal analysis as Forgent skill YAML files
**And** the exported YAML is valid and can be compiled with `forgent build`

### Story 6.3: Shell Completion and CLI Polish

As a **developer**,
I want shell completion and polished help text,
So that the CLI feels professional and discoverable.

**Acceptance Criteria:**

**Given** a developer runs `forgent completion bash` (or zsh, fish, powershell)
**When** executed
**Then** it outputs a shell completion script
**And** completions include: command names, flag names, file paths for doctor, target names for build

**Given** a developer runs `forgent doctor --help`
**When** help is displayed
**Then** it includes at least one usage example showing a complete doctor workflow
**And** all flags are documented with their defaults

**Given** any forgent command encounters an error
**When** the error is displayed
**Then** it includes a suggested fix
**And** exit codes are consistent: 0 (success), 1 (error), 2 (recommendations found)

---

## Epic 7: Static Analysis + Quality Pipeline (from Technical Research)

> *Architecture sourced from technical research (2026-03-31): LLM-as-Judge patterns, static analysis of markdown, quality metrics. Party mode decisions on scoring model, check inventory, integration patterns. ~33 hours estimated.*
>
> **FRs covered:** FR10, FR8, FR9, plus new: structural scoring, quality gate, CI mode, agent smells

### Story 7.0: Quick Wins — CoT Prompt + LLM Cache

As a **developer**,
I want Doctor's LLM analysis to be more accurate and faster on repeat runs,
So that I get better recommendations and don't wait for unchanged files.

**Acceptance Criteria:**

**Given** the LLM prompt template
**When** the YAML example output is generated
**Then** it includes a `reasoning` field BEFORE the `confidence` field (CoT-before-score pattern)

**Given** a developer runs `forgent doctor agent.md` a second time without modifying the file
**When** the cache key matches (same content + model + registry version)
**Then** cached findings are returned in <100ms without calling the LLM

**Given** `--fix` is applied
**When** the file is modified
**Then** the cache is purged for that file

**Research backing:** G-Eval (arXiv:2303.16634): +10-15% alignment. ESLint: content-hash caching.
**Effort:** ~3 hours

### Story 7.1: Static Check Foundation + Progressive Display

As a **developer**,
I want to see instant structural feedback while the LLM analysis runs,
So that I get immediate value and the tool feels responsive.

**Acceptance Criteria:**

**Given** a developer runs `forgent doctor agent.md`
**When** the format parser and enricher have completed (< 0.1s)
**Then** structural findings are displayed immediately:
  - Format detected
  - Section count
  - Missing sections (security, guardrails, examples, testing, error handling)
  - Primacy bias check (critical instructions at top?)

**Given** structural findings are displayed
**When** the LLM analysis is still running
**Then** a progress indicator shows LLM analysis status
**And** LLM findings appear incrementally as they are parsed

**Competitive context:** AgentLinter is instant (static only, no LLM). This closes the perceived speed gap while keeping LLM depth. Uses existing code: `context/enricher.go` gap detection + `CheckTextOrdering()`.

**Given** `--quick` flag is set
**When** doctor runs
**Then** only static checks execute (no LLM call), structural score is displayed, exit code reflects result

**Research backing:** Liu et al. (arXiv:2307.03172), Chatlatanagulchai et al. (arXiv:2511.12884), AgentLinter competitive baseline.
**Effort:** ~10 hours

### Story 7.2: Full Static Suite — Secrets, Readability, Tokens, Duplicates

As a **developer**,
I want Doctor to catch secrets, measure readability, estimate token costs, and detect duplicates without needing an LLM,
So that I get comprehensive structural analysis instantly.

**Acceptance Criteria:**

**Given** an agent file contains an API key pattern (e.g., `sk-ant-...`)
**When** doctor runs (any mode)
**Then** it flags the secret as HIGH severity

**Given** doctor completes static analysis
**When** results are rendered
**Then** indicators show: Readability (Flesch), Specificity ratio, Token count per section

**Given** two instruction lines have Jaccard similarity ≥0.5
**When** thorough mode is active
**Then** both lines are flagged as near-duplicates

**Research backing:** Gloaguen et al. (+20% token cost), Chatlatanagulchai et al. (Flesch 16.6), Snyk (36% skill secret exposure).
**Effort:** ~10 hours

### Story 7.3: Pipeline Orchestration + Quality Gate + CI Mode

As a **developer**,
I want Doctor to produce a pass/fail quality gate I can use in CI,
So that my team enforces agent file quality on every PR.

**Acceptance Criteria:**

**Given** a quality gate with conditions: `secrets==0`, `structural>=50%`, `high<=5`
**When** doctor evaluates the report
**Then** gate passes only if ALL blocking conditions pass (AND-logic)

**Given** `--ci` flag in a GitHub Action without Ollama
**When** doctor runs
**Then** it auto-falls back to --quick, outputs GitHub annotations, returns exit code 0/1/2

**Given** `--format json`
**When** doctor completes
**Then** output is structured JSON: structural_score, indicators, findings, gate result

**Research backing:** SonarQube quality gate, golangci-lint tag filtering, Promptfoo threshold-based pass/fail.
**Effort:** ~10 hours

### Story 7.4: Cross-File Consistency + Codebase Drift Detection

As a **developer**,
I want Doctor to detect contradictions between my agent files and warn about stale references,
So that my configuration stays consistent and up-to-date.

**Acceptance Criteria:**

**Given** `forgent doctor .` finds multiple agent files
**When** LLM analysis runs
**Then** cross-file consistency is checked: contradictions, persona conflicts, permission mismatches

**Given** an agent file references `src/utils/auth.go`
**When** that file doesn't exist
**Then** doctor flags: "Referenced file not found"

**Given** `--thorough` mode in a git repo
**When** agent file not modified in >90 days
**Then** warning: "Agent file not modified in X days"

**Research backing:** Chatlatanagulchai et al. (67.4% modified multiple times), Packmind drift detection.
**Effort:** ~8 hours

### Story 7.5: Agent Smell Detection

As a **developer**,
I want Doctor to detect anti-patterns in my agent files,
So that I avoid common mistakes that reduce agent effectiveness.

**Acceptance Criteria:**

**Given** `--thorough` mode
**When** file has >3000 words AND >8 sections
**Then** "God agent" smell detected

**Given** file has >80% Jaccard similarity with default templates
**Then** "Copy-paste agent" smell detected

**Given** file has >3 LLM transition phrases ("Additionally", "Furthermore") AND >5 numbered lists
**Then** "LLM-generated agent" smell detected (citing Gloaguen -3% finding)

**Given** >10 prohibitions AND <3 positive permissions
**Then** "Over-constrained agent" smell detected

**Research backing:** Gloaguen et al. (LLM-generated = -3%), SkillsBench (4+ skills = diminishing returns).
**Effort:** ~6 hours

---

## Epic 8: Scaffold + Index Architecture (from bench learnings)

> *Architecture sourced from bench analysis (121 agent files, avg 18% structural score) and party mode decisions on AGENTS.md index pattern. ~15 hours estimated.*

### Story 8.1: Scaffold Fix + Index Resolution

As a **developer**,
I want `forgent doctor --fix` to generate an AGENTS.md index + specialized files,
So that my monolithic agent file is decomposed into focused, discoverable files.

**Acceptance Criteria:**

**Given** `forgent doctor agent.md --fix`
**When** the file is not already an index
**Then** generates AGENTS.md index + 6 specialized .agents/*.md files, content migrated

**Given** the scaffold runs
**When** files already exist
**Then** existing files are skipped (no overwrite)

**Research backing:** SkillsBench (2-3 skills > monolithic), Vasilopoulos (hot/cold tiers).
**Effort:** ~8 hours. **Status: done.**

### Story 8.2: Index-Aware Scoring + Aggregate

As a **developer**,
I want Doctor to score an AGENTS.md index by resolving its links and computing an aggregate,
So that I see the real quality of my decomposed agent configuration.

**Acceptance Criteria:**

**Given** `forgent doctor AGENTS.md`
**When** the file contains markdown links to .agents/*.md files
**Then** Doctor resolves each link and analyzes each file

**Given** each linked file is analyzed
**When** results are aggregated
**Then** each file is scored on its relevant categories only (not all 16)
**And** aggregate = unique sections covered / total sections

**Given** a linked file is missing
**Then** HIGH finding with path

**Given** multiple files are resolved
**When** LLM is available
**Then** cross-file consistency runs on all resolved files

**Effort:** ~4 hours

### Story 8.3: Complete Templates + 100% Scaffold Score

As a **developer**,
I want the scaffold templates to cover all 15 section categories,
So that a freshly scaffolded project starts at 100% structural score.

**Acceptance Criteria:**

**Given** `--fix` scaffolds a project
**When** the aggregate score is computed
**Then** all 15 categories are covered (examples, goals, memory_management, workflow_triggers added)

**Given** codebase context is available from scanner
**When** templates are generated
**Then** test commands, build commands, directory structure, language are injected

**Effort:** ~2 hours
