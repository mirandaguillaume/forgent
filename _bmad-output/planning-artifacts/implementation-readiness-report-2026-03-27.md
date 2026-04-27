---
stepsCompleted:
  - step-01-document-discovery
date: '2026-03-27'
project: 'ax-cli'
documents:
  prd: '_bmad-output/planning-artifacts/prd.md'
  architecture: null
  epics: null
  ux: null
---

# Implementation Readiness Assessment Report

**Date:** 2026-03-27
**Project:** ax-cli (Forgent)

## Document Inventory

| Document | Status | Path |
|---|---|---|
| PRD | Found | `_bmad-output/planning-artifacts/prd.md` |
| Architecture | Not found | — |
| Epics & Stories | Not found | — |
| UX Design | Not found | — |

## PRD Analysis

### Functional Requirements

**Agent Analysis (11 FRs):** FR1-FR11 — doctor analyzes Claude/Copilot/Forgent YAML files, auto-detects format, local LLM + API fallback, TTY-aware, directory analysis, codebase enrichment, static fallback, Ollama model detection.

**Recommendations Engine (7 FRs):** FR12-FR18 — 5-8 research-backed recommendations, confidence levels, research registry (versioned, updatable), diagnoses text instruction limits, registry update notification + command.

**Agent Rewriting (7 FRs):** FR19-FR25 — dry-run default, diff preview, granular apply, automatic backup, restore, format-preserving rewrite for Claude and Copilot (frontmatter, custom sections, parser validation).

**Skill Specification & Composition (8 FRs):** FR26-FR33 — YAML with 5 facets, agent composition, scaffold, lint, score, composition validation, version field with compatibility.

**Multi-Target Compilation (10 FRs):** FR34-FR43 — Claude/Copilot/runtime targets, multi-target build, enrichment, watch mode, import with quality scoring, deterministic builds, format version pinning.

**Runtime Execution (7 FRs):** FR44-FR50 — DAG execution, contract enforcement, pluggable providers, router nodes, retry, error diagnostics, **FR50 (KEYSTONE): doctor as Forgent runtime agent**.

**Benchmarking & Evidence (3 FRs):** FR51-FR53 — InstructionBench, runtime-vs-text experiment, format fidelity test suite.

**Ecosystem & Extensibility (5 FRs):** FR54-FR58 — target generator plugin, list targets, export-yaml, doctor config, shell completion.

**Total FRs: 58**

### Non-Functional Requirements

**Performance (7 NFRs):** NFR1-NFR7 — doctor analysis 30s/60s/90s, compiler 5s, enrichment 15s, startup 200ms, binary <50MB, RAM <500MB.

**Reliability (10 NFRs):** NFR8-NFR17 — zero broken rewrites, deterministic builds, backup-before-write, graceful malformed handling, DAG failure handling, CI gate, upgrade preservation, concurrent safety, registry integrity, backup cleanup.

**Privacy & Data Handling (5 NFRs):** NFR18-NFR22 — local by default, explicit API consent, no telemetry, no caching, network transparency.

**Integration Compatibility (5 NFRs):** NFR23-NFR27 — platform version support (6-month window), compiled output compatibility, any Ollama model, Anthropic + OpenRouter APIs, backward-compatible YAML schema.

**Developer Experience (9 NFRs):** NFR28-NFR36 — actionable errors, help examples, exit codes, offline capable, cross-platform, UTF-8 handling, Ctrl+C cancellation, LLM timeout, debug mode.

**Testing (2 NFRs):** NFR37-NFR38 — ≥80% coverage, race detector in CI.

**Total NFRs: 38**

### Additional Requirements

**Domain constraints identified in PRD:**
- 5 domain risks with mitigations (format instability, platform orchestration, competing specs, LLM capability leap, industry convergence)
- 5 monitored assumptions with break signals and monitoring mechanisms
- 6 technical constraints (format fragility, LLM non-determinism, context window limits, evaluation gap, pre-paradigmatic domain, research registry versioning)

**Scoping constraints:**
- MVP thesis explicitly stated: "A Forgent runtime agent that analyzes any AI agent file can diagnose instruction limits and measurably outperform text-based agents on complex tasks through structural enforcement."
- Engineering spike required before full MVP (LLM quality validation)
- Implementation sequence: Spike → Phase A (analysis) → Phase C (runtime) → Phase D (benchmark) → Phase B (rewrite)
- Resource model: solo architect + LLM development team, 4-6 weeks

**Explicit out-of-MVP exclusions (12 items):** npm, Homebrew, managed hosting, behavioral testing, bounded cycles, VS Code/LSP, typed contracts, SWE-bench, deep decomposition, format version pinning, JSON/SARIF output, additional format support.

### PRD Completeness Assessment

**Strengths:**
- Comprehensive FR coverage (58 FRs across 8 capability areas)
- Measurable NFRs (38 NFRs across 6 categories with specific targets)
- Clear MVP thesis and scope boundaries
- Explicit out-of-scope list prevents scope creep
- Implementation sequencing based on actual codebase gap analysis
- 7 user journeys covering happy path, failure path, and 5 user types
- Domain risks with monitoring mechanisms
- Honest innovation framing (what's novel vs what's existing pattern)

**Gaps for implementation readiness:**
- No Architecture document — technical design decisions not yet formalized
- No Epics & Stories — work breakdown not yet created
- No UX Design — CLI interaction patterns not formalized (may not be needed for CLI tool)
- FR50 (KEYSTONE) depends on all runtime FRs — critical path not mapped
- Doctor config schema (FR57) marked "TBD" — needs design before implementation

## Epic Coverage Validation

**Status: NOT POSSIBLE — No epics document exists.**

All 58 FRs are currently uncovered by epics. Epic & story creation is a prerequisite before implementation can begin.

### Coverage Statistics

- Total PRD FRs: 58
- FRs covered in epics: 0
- Coverage percentage: 0%

### Recommendation

Run `bmad-create-epics-and-stories` to create the epic breakdown from this PRD before assessing implementation readiness.

## UX Alignment Assessment

### UX Document Status

Not found — and **not required**. Forgent is a CLI tool. The user experience is terminal-based: commands, flags, colored output, interactive prompts for `--fix`, diff previews. This UX is fully specified in:
- CLI Tool Requirements section (command structure, interactivity model, output formats)
- NFRs 28-36 (Developer Experience: actionable errors, help examples, exit codes, cross-platform, UTF-8, Ctrl+C, timeout, debug mode)
- User journeys (Alex's happy path and failure path describe the complete CLI experience)

### Alignment Issues

None — no visual UI exists or is implied. The CLI interaction model is well-defined in the PRD.

### Warnings

None. A separate UX design document would be appropriate if/when the Vision phase includes a VS Code extension or web dashboard. Not needed for MVP.

## Epic Quality Review

**Status: NOT POSSIBLE — No epics document exists.**

Cannot validate epic structure, story quality, dependency analysis, or acceptance criteria without an epics & stories document. This is the primary blocker for implementation readiness.

## Summary and Recommendations

### Overall Readiness Status

**NOT READY** — PRD is complete and high quality, but architecture and epics do not exist yet. Implementation cannot begin without a work breakdown.

### Findings Summary

| Area | Status | Issues |
|---|---|---|
| PRD | Complete | 58 FRs, 38 NFRs, comprehensive vision document |
| Architecture | Missing | No technical design document |
| Epics & Stories | Missing | 0% FR coverage — primary blocker |
| UX Design | Not needed | CLI tool — UX defined in PRD (CLI Requirements + NFRs) |

### Critical Issues Requiring Immediate Action

1. **No epics or stories exist.** All 58 FRs have zero implementation coverage. Cannot begin development without a work breakdown.
2. **No architecture document.** Technical design decisions (doctor skill decomposition, Ollama integration pattern, format parser architecture, research registry design, doctor config schema) are not formalized.
3. **Doctor config schema marked "TBD"** in PRD (FR57). Needs design before implementation.

### What's Strong

- PRD is comprehensive with clear MVP thesis, explicit scope boundaries, and honest risk assessment
- 58 FRs across 8 capability areas form a solid capability contract
- 38 NFRs with specific, measurable targets
- Implementation sequencing based on actual codebase gap analysis (Spike → A → C → D → B)
- Existing codebase provides substantial infrastructure (compiler, DAG engine, runtime generator, enrichment pipeline, import pipeline)
- 7 user journeys including failure recovery path

### Recommended Next Steps

1. **Create Architecture** — run `bmad-create-architecture` to formalize technical design decisions for doctor, format parsers, research registry, and Ollama integration
2. **Create Epics & Stories** — run `bmad-create-epics-and-stories` to break the 58 FRs into implementable work items following the Spike → Phase A → C → D → B sequencing
3. **Design doctor config schema** — resolve the TBD in FR57 as part of architecture
4. **Run engineering spike** — validate local LLM quality on 20 real agent files before committing to full MVP (highest risk item)
5. **Re-run this readiness check** after architecture and epics are created

### Final Note

This assessment identified 3 critical issues, all related to missing downstream documents (architecture, epics). The PRD itself is thorough and implementation-ready — it provides a solid foundation for the next workflow steps. The existing codebase significantly reduces implementation risk. Address the missing documents, then re-assess.
