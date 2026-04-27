---
stepsCompleted: [1, 2, 3, 'party-integration', 4, 5, 6]
inputDocuments:
  - _bmad-output/planning-artifacts/research/market-agent-spec-tools-2026-03-31.md
  - _bmad-output/planning-artifacts/research/domain-agent-behavioral-standards-2026-03-31.md
  - internal/doctor/registry/data/research-registry.yaml
workflowType: 'research'
lastStep: 1
research_type: 'technical'
research_topic: 'Agent File Quality Analysis — Metrics, LLM-as-Judge, Static Analysis'
research_goals: 'Improve Doctor scoring + Design InstructionBench (Epics 4-7)'
user_name: 'Guillaume'
date: '2026-03-31'
web_research_enabled: true
source_verification: true
---

# Research Report: technical

**Date:** 2026-03-31
**Author:** Guillaume
**Research Type:** technical

---

## Research Overview

This research investigates how to measure and improve AI agent specification file quality through three complementary lenses: quality metrics, LLM-as-Judge patterns, and static analysis. The goal is to upgrade Forgent Doctor from a recommendation-list tool to a calibrated quality diagnostic with defensible scoring, and to design InstructionBench — the first benchmark proving that Doctor recommendations improve agent performance.

**Key finding:** No validated quality metric for agent specification files exists in the literature. The field is pre-paradigmatic. Forgent Doctor's 24-category research-backed registry is among the most structured approaches, but no tool — including Doctor — has proven that file quality metrics correlate with agent performance. InstructionBench would be the first to close this gap.

**Methodology:** Web search verification of all claims. 30+ papers and tools analyzed. All findings cross-referenced. Party mode sessions with 4 analysts challenged every conclusion.

---

## Technical Research Scope Confirmation

**Research Topic:** Agent File Quality Analysis — Metrics, LLM-as-Judge, Static Analysis
**Research Goals:** Improve Doctor scoring + Design InstructionBench (Epics 4-7)

**Technical Research Scope:**

- Architecture Analysis - design patterns, frameworks, system architecture
- Implementation Approaches - development methodologies, coding patterns
- Technology Stack - languages, frameworks, tools, platforms
- Integration Patterns - APIs, protocols, interoperability
- Performance Considerations - scalability, optimization, patterns

**Three Focus Areas:**

1. Agent File Quality Metrics — scoring rubrics, benchmarks, correlation with agent performance
2. LLM-as-Judge Patterns — calibration, rubric design, biases, cost/quality tradeoffs
3. Static Analysis of Structured Markdown — AST parsing, rule engines, hybrid static+LLM

**Research Methodology:**

- Current web data with rigorous source verification
- Multi-source validation for critical technical claims
- Confidence level framework for uncertain information
- Actionable specs — each section ends with implementation recommendations for Forgent Doctor

**Scope Confirmed:** 2026-03-31

---

## Technology Stack Analysis

### Area 1: Agent File Quality Metrics

#### State of the art: No validated rubric exists

No peer-reviewed, validated rubric specifically for scoring agent specification file quality has been published. What exists:

**Agent Skill Report (2026)** — 673 skills scored across 6 LLM-as-judge dimensions:
1. Clarity, 2. Actionability, 3. Token efficiency, 4. Scope discipline, 5. Directive precision, 6. Novelty.
Key finding: skills cluster tightly on "craft" dimensions (1-5) but spread independently on novelty — craft is table-stakes, novelty differentiates.
Source: [agentskillreport.com](https://agentskillreport.com/)

**DETAIL Framework (Kim, 2025)** — measures prompt specificity via perplexity (L1 vague=18.9, L2 moderate=30.3, L3 detailed=45.7). Specificity improves accuracy, especially for smaller models.
Source: [arXiv:2512.02246](https://arxiv.org/abs/2512.02246)

**Readability metrics (Chatlatanagulchai et al., 2025)** — only study measuring readability of agent files:
| Agent Tool | Median Flesch RE | Interpretation |
|---|---|---|
| Claude Code | 16.6 | "Very difficult" |
| GitHub Copilot | 26.6 | "Very difficult" |
| OpenAI Codex | 39.6 | "Difficult" |
Source: [arXiv:2511.12884](https://arxiv.org/abs/2511.12884)

**Key gap:** No benchmark exists specifically for measuring agent FILE quality. SWE-bench, AGENTIF, SkillsBench all measure agent runtime performance, not intrinsic file quality. "InstructionBench" does not exist as a named benchmark.

**Correlation evidence (weak):**
- Gloaguen et al.: presence vs absence gives +4% (human-written) or -3% (LLM-generated). No quality metric correlated.
- SkillsBench: 2-3 focused skills yield +20.0pp; 4+ drops to +5.2pp. Composition granularity is the strongest signal.
- No study connects a static quality metric to downstream agent performance.

#### Implementation recommendation for Doctor

Forgent's registry of 24 research-backed categories + score.go (8 weighted facets) is among the most structured approaches that exist — but unvalidated against downstream performance. InstructionBench (Epic 5) would be the first benchmark to close this gap.

---

### Area 2: LLM-as-Judge Patterns

#### Foundational findings

**Zheng et al. (NeurIPS 2023)** — GPT-4 as judge matches human preference at >80% agreement (same as human inter-annotator rate). Three systematic biases: position bias, verbosity bias, self-enhancement bias.
Source: [arXiv:2306.05685](https://arxiv.org/abs/2306.05685)

**Known biases:**
| Bias | Severity | Mitigation |
|---|---|---|
| Position bias | High | Swap response positions, average scores |
| Verbosity bias | High | Length penalties; rubric: "conciseness counts" |
| Self-preference bias | Moderate-High | Ensemble of different model families; steering vectors (up to 97% reduction, Panickssery et al.) |
| Leniency bias | Moderate | Minority-veto ensembles; calibration against human baselines |
| Format bias | Moderate | Explicit rubric stating format ≠ quality |

Source bias mitigation: [arXiv:2410.21819](https://arxiv.org/abs/2410.21819)

#### Key frameworks

**LLM-Rubric (Hashemi et al., ACL 2024)** — manually constructed rubric + LLM probability distributions + small neural network. RMS error <0.5 on 1-4 scale. 2x improvement over uncalibrated.
Source: [ACL 2024.acl-long.745](https://aclanthology.org/2024.acl-long.745/)

**AdaRubric (2026)** — task-adaptive rubric generation. Pearson r=0.79 human correlation. Krippendorff's alpha=0.83.
Source: [arXiv:2603.21362](https://arxiv.org/abs/2603.21362)

**RULERS (Hong et al., 2026)** — compiles natural language rubrics into versioned, immutable bundles with deterministic evidence rules. Smaller models rival frontier judges with precise rubrics.
Source: [arXiv:2601.08654](https://arxiv.org/abs/2601.08654)

**Trust or Escalate (ICLR 2025)** — cascaded evaluation. Start cheap (Mistral-7B), escalate to expensive only when confidence is low. >80% human agreement with ~80% cheap-model coverage.
Source: [arXiv:2407.18370](https://arxiv.org/abs/2407.18370)

**LLM-as-Judge on a Budget (Saha et al., 2026)** — variance-adaptive allocation. Factual questions have σ²~0.0001, subjective questions σ²~10. Concentrate multiple queries where uncertainty is highest.
Source: [arXiv:2602.15481](https://arxiv.org/abs/2602.15481)

#### Critical finding: CoT-before-score

Explanation must come BEFORE the score (G-Eval, arXiv:2303.16634). If score is produced first, the explanation is rationalized, not genuine reasoning. Improves alignment by 10-15%.

#### Doctor vs LLM-as-Judge best practices

| Dimension | Doctor (current) | Best practice | Gap |
|---|---|---|---|
| Scoring paradigm | Pointwise, reference-free | Pointwise + reference-based hybrid | Add gold-standard exemplars |
| Scale | 3-point confidence (H/M/L) | 5-point per dimension + anchored rubric | Confidence ≠ quality score |
| CoT reasoning | Not required from LLM | Explanation before score | Add `reasoning` field |
| Rubric precision | Questions ("Are there...?") | Evidence-anchored rubrics (RULERS) | Major upgrade opportunity |
| Calibration | None | Few-shot anchoring, human correction loop | Biggest gap |
| Multi-judge | Single LLM call | Panel or cascaded evaluation | Cascade: local structural + frontier quality |
| Bias mitigation | None | Position randomization, length penalties | Free to add |
| Cost optimization | Single call with all categories | Per-category routing + caching | Add caching by file hash |

#### Implementation recommendation for Doctor

**Phase 1 (free wins):** Add `reasoning` field before score in prompt. Add file-hash caching. Add explicit bias mitigation instructions to prompt.
**Phase 2 (calibration):** Curate 10-20 gold-standard agent files scored by humans. Measure Cohen's Kappa. Inject best examples as few-shot.
**Phase 3 (cascade):** Tier 0 (static, zero-cost) → Tier 1 (local binary pass/fail) → Tier 2 (frontier quality scoring).

---

### Area 3: Static Analysis of Structured Markdown

#### Recommended library: goldmark (Go)

| Library | AST Walking | CommonMark | Extensible | Maintained | Frontmatter |
|---|---|---|---|---|---|
| **goldmark** | Full `ast.Walk()` enter/exit | Yes | Plugin system | Active (v1.7.12) | Via goldmark-meta |
| gomarkdown | `ast.WalkFunc()` | Partial | Limited | Active | Manual |
| blackfriday v2 | `Node.Walk()` | No | No | Stale (2020) | No |

Performance: Parse + full AST walk = **75 microseconds** on a typical agent file. Real-time analysis is trivial.

#### Static checks immediately implementable

**Phase 1 — No new dependencies (pure Go):**
1. Section presence/absence (walk existing `parser.Section` list)
2. Instruction ordering (critical sections first?)
3. Keyword density (security, testing, guardrails keywords)
4. Duplicate instruction detection (Jaccard similarity ≥ 0.5)
5. Position risk warning (critical content in middle)
6. Vague instruction patterns (regex: "if needed", "be thorough", 17 patterns)

**Phase 2 — Add goldmark:**
7. Proper AST-based parsing (replaces manual `parseSections()`)
8. Code block exclusion from prose analysis
9. Link/reference validation (do referenced files exist?)
10. Readability metrics (Flesch-Kincaid, Gunning Fog — pure Go formulas)
11. Section-level token budgets (tiktoken-go or `chars/4 * 0.78`)

**Phase 3 — Hybrid pipeline:**
12. Progressive display (static instant → LLM streaming)
13. Gate decision (skip LLM on critical structural issues)
14. Merged rendering of static + LLM findings
15. Result caching by content hash

#### Token counting

| Method | vs tiktoken cl100k_base | Error |
|---|---|---|
| tiktoken-go (exact) | baseline | 0% |
| chars/4 | +28% overestimate | high |
| chars/4 * 0.78 | ~0% | acceptable |

Source: benchmarked on typical agent file text.

#### Hybrid architecture

```
Static Analysis (goldmark AST) — instant (<1ms)
├── Structure checks (section presence, ordering)
├── Token budget (per-section estimation)
├── Keyword density (security, testing, etc.)
├── Duplicate detection (Jaccard similarity)
├── Readability metrics (Flesch, Fog)
└── Link validation (file existence)
         │
    Gate Decision — skip LLM if:
    - Critical structural issues found
    - File too large (>50K tokens)
    - No LLM provider available
         │
LLM Analysis (semantic) — 2-10s
├── Vague instruction quality
├── Contradictory logic detection
├── Missing context assessment
├── Decomposition advice
└── Security gap analysis
         │
Merged Results (static + LLM)
```

#### Static vs LLM division

| Check | Static | LLM | Rationale |
|---|---|---|---|
| Section presence | ✅ | | Deterministic AST check |
| Instruction ordering | ✅ | | Position in AST |
| Token budget | ✅ | | Arithmetic |
| Readability | ✅ | | Formula-based |
| Duplicate detection | ✅ | | Jaccard similarity |
| Keyword density | ✅ | | Word counting |
| Link validation | ✅ | | File existence |
| Frontmatter schema | ✅ | | Schema validation |
| Vague instructions | | ✅ | Semantic understanding |
| Contradictory logic | | ✅ | Requires reasoning |
| Missing context | | ✅ | Domain knowledge |
| Decomposition advice | | ✅ | Design judgment |
| Security gaps | ✅ partial | ✅ | Keywords + semantic |

#### Recommended pattern for Doctor static checks

```go
type StaticCheck interface {
    ID() string
    Category() string       // maps to registry category
    DefaultSeverity() string
    Check(analysis *parser.AgentAnalysis) []llmutil.Finding
}
// Same init() self-registration pattern as internal/linter/
```

Produces `llmutil.Finding` objects identical to LLM output — seamless merging.

---

### Cross-Area Synthesis: What to Build

| Priority | What | Area | Impact | Effort |
|---|---|---|---|---|
| **P0** | Add `reasoning` field to LLM prompt (CoT-before-score) | LLM-Judge | +10-15% alignment | Low |
| **P0** | Result caching by file content hash | LLM-Judge | Cost reduction | Low |
| **P1** | 6 static checks (Phase 1, no deps) | Static | Instant feedback | Medium |
| **P1** | Progressive display (static → LLM) | Static | UX improvement | Medium |
| **P2** | goldmark AST migration | Static | Enables Phase 2 checks | Medium |
| **P2** | 5-point rubric with evidence anchoring | LLM-Judge | Calibrated scoring | Medium |
| **P3** | Gold-standard calibration set (10-20 files) | Metrics | Validated scoring | High |
| **P3** | InstructionBench (Epic 5) | Metrics | First-of-kind benchmark | High |
| **P3** | Cascaded evaluation (local → frontier) | LLM-Judge | Cost optimization | Medium |

---

## Sources

### Quality Metrics
- [Agent Skill Report](https://agentskillreport.com/)
- [DETAIL Framework — arXiv:2512.02246](https://arxiv.org/abs/2512.02246)
- [Agent GPA Framework — arXiv:2510.08847](https://arxiv.org/abs/2510.08847)
- [LLM-Rubric — ACL 2024](https://aclanthology.org/2024.acl-long.745/)
- [AdaRubric — arXiv:2603.21362](https://arxiv.org/abs/2603.21362)
- [Autorubric — arXiv:2603.00077](https://arxiv.org/abs/2603.00077)
- [AGENTIF — arXiv:2505.16944](https://arxiv.org/abs/2505.16944)
- [SWE-ContextBench — arXiv:2602.08316](https://arxiv.org/abs/2602.08316)
- [PECS Scale — Sage Journals](https://journals.sagepub.com/doi/10.1177/02666669251336455)
- [Agent Behavioral Contracts — arXiv:2602.22302](https://arxiv.org/abs/2602.22302)
- [Agentic Rubrics (Scale AI) — arXiv:2601.04171](https://arxiv.org/abs/2601.04171)

### LLM-as-Judge
- [Zheng et al. — Judging LLM-as-a-Judge — arXiv:2306.05685](https://arxiv.org/abs/2306.05685)
- [Gu et al. — Survey on LLM-as-a-Judge — arXiv:2411.15594](https://arxiv.org/abs/2411.15594)
- [Li et al. — LLMs-as-Judges Survey — arXiv:2412.05579](https://arxiv.org/abs/2412.05579)
- [LLM-as-a-Judge for SE — arXiv:2510.24367](https://arxiv.org/abs/2510.24367)
- [Self-Preference Bias — arXiv:2410.21819](https://arxiv.org/abs/2410.21819)
- [Judge's Verdict — arXiv:2510.09738](https://arxiv.org/abs/2510.09738)
- [Trust or Escalate — arXiv:2407.18370](https://arxiv.org/abs/2407.18370)
- [RULERS — arXiv:2601.08654](https://arxiv.org/abs/2601.08654)
- [G-Eval — arXiv:2303.16634](https://arxiv.org/abs/2303.16634)
- [LLM-as-Judge on a Budget — arXiv:2602.15481](https://arxiv.org/abs/2602.15481)
- [JudgeLM — arXiv:2310.17631](https://arxiv.org/abs/2310.17631)
- [Prometheus 2 — arXiv:2405.01535](https://arxiv.org/abs/2405.01535)
- [CMU — Rating Indeterminacy](https://blog.ml.cmu.edu/2025/12/09/validating-llm-as-a-judge-systems-under-rating-indeterminacy/)
- [Anthropic — Demystifying Evals](https://www.anthropic.com/engineering/demystifying-evals-for-ai-agents)
- [DeepEval](https://github.com/confident-ai/deepeval)
- [Awesome LLMs-as-Judges](https://github.com/CSHaitao/Awesome-LLMs-as-Judges)

### Static Analysis
- [goldmark](https://github.com/yuin/goldmark)
- [goldmark-meta](https://github.com/yuin/goldmark-meta)
- [vale](https://github.com/errata-ai/vale)
- [markdownlint](https://github.com/DavidAnson/markdownlint)
- [tiktoken-go](https://github.com/pkoukk/tiktoken-go)

## Party Mode Decisions (from technology stack analysis)

### Decision 1: Scoring Model — No single aggregate score

| Level | Type | Output | Foundation |
|---|---|---|---|
| **Structural Score** | ~17 checks pass/fail | `14/17 (82%)` | Research-backed, deterministic, defensible |
| **Indicators** | Continuous metrics | Readability 22.4, Specificity 0.71, Tokens 3420 | Informative, not scored |
| **Quality Findings** | Prioritized list | `4 findings (2 HIGH, 2 MOD)` | LLM analysis, not numerically scored |

Rationale: No validated quality metric correlates with agent performance. Aggregate scores (like AgentLinter's B+) use arbitrary weights. Structural pass/fail is defensible. Quality scoring deferred to post-InstructionBench.

### Decision 2: Check Inventory — 68 checks identified, 54 static

| Category | Count | Static | LLM |
|---|---|---|---|
| Section presence | 15 | 15 | 0 |
| Quantitative (readability, tokens, length) | 8 | 8 | 0 |
| Heuristic (regex, similarity, keywords) | 15 | 15 | 0 |
| Format-specific (Claude/Copilot/Forgent) | 8 | 6 | 2 |
| Temporal (git-aware) | 4 | 4 | 0 |
| Internal consistency | 6 | 2 | 4 |
| Agent smells (anti-patterns) | 6 | 4 | 2 |
| **Total** | **62 new + 24 existing LLM** | **54** | **8 + 24** |

### Decision 3: Tag-based activation

```
forgent doctor agent.md              → 17 structural + LLM (default)
forgent doctor agent.md --thorough   → 54 static + LLM (all checks)
forgent doctor agent.md --quick      → 17 structural only (no LLM, instant)
forgent doctor agent.md --security   → secret scanning + injection only
```

### Decision 4: Check implementation pattern

```go
type StaticCheck interface {
    ID() string
    Tags() []string  // "default", "thorough", "security", "git-aware"
    Category() string
    DefaultSeverity() string
    Check(analysis *parser.AgentAnalysis) []llmutil.Finding
}
```

Same `init()` self-registration as `internal/linter/`. Produces `llmutil.Finding` for seamless merging with LLM results.

### Decision 5: Priority order

| Priority | What | Why |
|---|---|---|
| P0 | CoT-before-score in LLM prompt | +10-15% alignment, zero code change |
| P0 | Result caching by file hash | Cost reduction |
| P0 | Secret scanning (17 regex) | Safety — must-have |
| P1 | InstructionBench (Epic 5) | Proves Doctor works |
| P1 | Progressive display (Story 7.1) | UX + competitive |
| P1 | 3 core static checks (presence, vague, ordering) | Structural scoring |
| P2 | Remaining static checks | Full structural score |
| P2 | goldmark AST migration | Enables advanced checks |
| P3 | Calibration set + 5-point rubric | Post-InstructionBench |
| P3 | Git-aware checks | Competitive with Packmind |

### Decision 6: Agent smells (unique to Doctor)

| Anti-pattern | Detection |
|---|---|
| God agent | >3000 words + >8 responsibilities |
| Copy-paste agent | Jaccard >80% vs known templates |
| LLM-generated agent | Distinctive LLM writing patterns |
| Checkbox agent | All sections present but each <20 words |
| Over-constrained agent | >10 prohibitions, few positive permissions |

### Decision 7: Format-specific checks

| Check | Format |
|---|---|
| Tools vs MCP servers match | Claude |
| Skill folder structure valid | Claude Skills |
| `applyTo` glob matches files | Claude Rules |
| Token limit warning | Copilot |
| File in correct directory | All |
| 5 facets present | Forgent YAML |
| DAG topology valid | Forgent YAML |

---

## Integration Patterns Analysis

### Pattern 1: Unified Check Interface (from DeepEval)

DeepEval's key insight: **no type distinction between code-based and LLM-based metrics at the interface level**. Both implement the same `BaseMetric` with `measure() -> float`. The LLM call is an implementation detail.

Applied to Doctor:
```go
type Check interface {
    Name() string
    Category() string           // maps to registry category
    Kind() CheckKind            // KindStatic | KindLLM
    Tags() []string             // "default", "thorough", "security", "git-aware"
    DefaultSeverity() Severity
    Run(ctx context.Context, input *parser.AgentAnalysis) ([]llmutil.Finding, error)
}
```

Static and LLM checks produce identical `llmutil.Finding` objects — the caller never needs to know which kind ran.

### Pattern 2: Tag-Based Filtering (from golangci-lint)

golangci-lint uses a `Groups` map on each linter config with a Manager that resolves enabled linters via group → enable/disable overrides:

```go
type CheckRegistry struct {
    checks map[string]Check
    groups map[string][]string // "default" -> [check names]
}

func (r *CheckRegistry) Resolve(mode string, enable, disable []string) []Check
```

Maps to Doctor's `--quick` / `--thorough` / `--security` flags:
- `--quick` resolves group "default" with `Kind==KindStatic` filter
- `--thorough` resolves group "all"
- `--security` resolves group "security"
- Default resolves group "default" (static + LLM)

### Pattern 3: Two-Phase Pipeline with Progressive Output

```
Phase 1: Static (instant)
├── Resolve checks: filter by tags + Kind==KindStatic
├── Run all static checks concurrently
├── Render structural score immediately: "14/17 ✅"
└── Render indicators: readability, tokens, specificity

Phase 2: LLM (2-10s)
├── Check cache: sha256(content + prompt_template + model)
├── On cache hit: return cached findings
├── On cache miss: call LLM, cache response
├── Parse findings from LLM response
└── Render quality findings incrementally

Phase 3: Merge + Quality Gate
├── Combine static + LLM findings
├── Post-process: dedup, severity override, sort
├── Evaluate quality gate conditions
└── Return exit code (0=pass, 1=error, 2=findings)
```

### Pattern 4: Quality Gate (from SonarQube)

SonarQube uses AND-logic conditions with per-metric thresholds. No weighted scores — binary pass/fail per condition:

```go
type GateCondition struct {
    Metric    string  // "structural_score", "high_findings", "secrets_found"
    Operator  string  // ">", "<", ">="
    Threshold float64
}

type QualityGate struct {
    Conditions []GateCondition
}

// Gate passes iff ALL conditions pass
func (g *QualityGate) Evaluate(report *AggregateReport) (bool, []string)
```

Default gate for Doctor:
- `structural_score >= 0.7` (at least 70% of checks pass)
- `secrets_found == 0` (no secrets in agent files)
- `high_findings <= 3` (max 3 high-severity LLM findings)

### Pattern 5: Content-Hash Caching (from ESLint + Promptfoo)

ESLint caches by `hash(fileContent + configHash)`. Promptfoo caches LLM responses, not assertions. Applied to Doctor:

- **Static checks: never cache** (fast enough at <1ms, always fresh)
- **LLM response: cache aggressively** by `sha256(fileContent + promptTemplate + modelName)`
- **Cache invalidation:** file content changes → all cache invalidated; model changes → LLM cache invalidated; registry version changes → LLM cache invalidated
- **Never cache files with --fix** (result would be stale after modification)

Storage: `.forgent/cache/doctor/` with JSON entries. TTL: 7 days.

### Pattern 6: Processor Pipeline (from golangci-lint)

golangci-lint post-processes issues through a chain of Processors. Applied to Doctor:

```
Raw findings
  → DeduplicateProcessor (remove duplicate findings across static+LLM)
  → SeverityOverrideProcessor (user config overrides severity)
  → SuppressionProcessor (inline comments: <!-- forgent-disable security -->)
  → SortProcessor (HIGH first, then MOD, then LOW)
  → LimitProcessor (max N findings per category)
  → SourceProcessor (attach source line context)
```

### Pattern 7: Shared Expensive Computation (from golangci-lint metalinter)

golangci-lint merges compatible linters that share expensive pre-computation (AST parsing). Applied to Doctor:

- **Parse once:** goldmark AST parsed once, shared across all static checks
- **Enrich once:** codebase context enrichment runs once, results shared with LLM prompt
- **Frontmatter once:** YAML frontmatter extracted once, shared with all checks that inspect metadata

### Integration Architecture Summary

```
                    ┌─────────────────────┐
                    │  Format Detection    │ ← existing parser/registry.go
                    │  + Parse             │
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │  Check Resolution   │ ← tag-based filtering
                    │  (mode + flags)     │
                    └──────────┬──────────┘
                               │
              ┌────────────────┼────────────────┐
              │                                  │
    ┌─────────▼─────────┐           ┌───────────▼───────────┐
    │  Static Checks    │           │  LLM Check            │
    │  (concurrent)     │           │  (cache-aware)        │
    │  → structural     │           │  → CoT-before-score   │
    │  → indicators     │           │  → 24 categories      │
    │  → security       │           │  → reasoning field    │
    └─────────┬─────────┘           └───────────┬───────────┘
              │ instant                          │ 2-10s
              │ render immediately               │ render incrementally
              └────────────────┬────────────────┘
                               │
                    ┌──────────▼──────────┐
                    │  Processor Pipeline │ ← dedup, sort, suppress
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │  Quality Gate       │ ← AND-logic conditions
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │  Exit Code          │ ← 0/1/2
                    └─────────────────────┘
```

### Parking Lot

- **DG (Directed Graphs with cycles) for Doctor:** Could enable analyze → fix → re-analyze → validate loops. Deferred — DG is Phase 3 roadmap for the runtime, not Doctor's analysis pipeline.

---

## Integration Patterns — Party Mode Decisions

### CI/CD Integration

- `--ci` flag: auto-detects (static-only if no API key, full if FORGENT_PROVIDER set)
- Quality gate conditions configurable in `.forgent/config.yaml`
- Default gate: `secrets_found==0` (blocking) + `structural_score>=0.5` (blocking) + `high_findings<=5` (warning)
- GitHub annotations output via `::error` / `::warning` format
- JSON output via `--format json` for custom integration

### Cache Design

- Location: `.forgent/cache/doctor/<hash>.json`
- Key: `sha256(fileContent + promptTemplate + modelName + registryVersion)`
- TTL: 7 days
- Static checks: never cached (fast enough)
- LLM responses: cached aggressively
- Invalidation: content change, model change, registry update, `--fix` purges
- Impact: directory mode 10 files goes from 5min → <1s on repeat runs

### Code Integration Plan

New files (4):
- `internal/doctor/static/` — static check package with `init()` registration
- `internal/doctor/pipeline.go` — orchestrates static → LLM → merge → gate
- `internal/doctor/cache.go` — content-hash LLM response cache
- `internal/doctor/gate.go` — quality gate with AND-logic conditions

Migrated (3):
- `context/enricher.go` gap detection → `static/presence.go`
- `context/enricher.go` CheckTextOrdering → `static/ordering.go`
- `validate/gate.go` → `gate.go` (replaced by condition-based gate)

Adapted (2):
- `render.go` — extended for structural score + indicators + findings
- `analyzer/analyzer.go` — wrapped as a KindLLM Check

### Progressive Rendering

- TTY: structural instant → spinner → findings incremental
- CI/pipe: sequential plain text
- JSON: structured output at end
- LLM findings parsed incrementally by `\n  - category:` block separator

---

## Architectural Patterns

### Doctor Architecture Evolution

**Current (v1 — Epics 1-3):**
```
Parse → Enrich → LLM Analyze → Render findings
                                └→ --fix: Rewrite → Validate → Apply
```

**Target (v2 — Epics 6-7):**
```
Parse → Static Checks (instant) → Render structural
     → Cache Check → LLM Analyze → Render quality
     → Merge → Post-process → Quality Gate → Exit code
                                └→ --fix: Rewrite → Validate → Apply
```

**Future (v3 — post-InstructionBench):**
```
Parse → Static → Calibrated LLM Judge → Validated Score
     → InstructionBench regression suite
     → Empirically-weighted dimensions
```

### Architectural Decision Records (new)

| ADR | Decision | Rationale |
|-----|----------|-----------|
| ADR-10 | No aggregate quality score | No validated metric; avoid false precision |
| ADR-11 | 3-layer output (structural + indicators + findings) | Defensible structural score + informative metrics + LLM depth |
| ADR-12 | Tag-based check activation | Supports --quick/--thorough/--security without multiple codepaths |
| ADR-13 | Static+LLM hybrid pipeline | 70% of checks are static (instant); LLM for semantic understanding |
| ADR-14 | Content-hash LLM cache | O(1) repeat runs; cache key includes model+registry version |
| ADR-15 | SonarQube-style quality gate | AND-logic conditions, no weighted scoring, CI-friendly |
| ADR-16 | CoT-before-score in LLM prompt | +10-15% alignment per G-Eval literature |

### InstructionBench Architecture (Epic 5)

**Design:** Controlled before/after experiment on SWE-bench tasks.

```
Corpus: 30-50 SWE-bench Verified tasks (Python, deterministic tests)

Conditions:
  A: No agent file (baseline)
  B: Original agent file (as-is from GitHub)
  C: Doctor-enhanced (after forgent doctor --fix)
  D: Doctor-enhanced + skill decomposition (2-3 skills)

Runner:
  For each (task, condition):
    1. Set up SWE-bench docker container
    2. Inject agent file into workspace
    3. Run agent (Claude Code or Codex) on task
    4. Run SWE-bench test suite
    5. Record: pass/fail, tokens, time, tool calls

Analysis:
  - Paired Wilcoxon signed-rank test (same tasks, different conditions)
  - Need ≥30 tasks for statistical power
  - Metrics: resolution rate, token consumption, wall-clock time

Estimated cost: 50 tasks × 4 conditions × ~$0.50/run = ~$100
```

**Key insight from research:** Deterministic verifiers (Docker + programmatic assertions) are more reliable than LLM-as-Judge for measuring agent performance. SkillsBench proved this approach works.

---

## Implementation Research

### Implementation Roadmap

| Phase | Scope | Sprint | Dependencies |
|-------|-------|--------|-------------|
| **Phase 0: Quick wins** | CoT prompt + cache | Now | None |
| **Phase 1: Static foundation** | `static/` package, 3 core checks, progressive display | Epic 7 | Phase 0 |
| **Phase 2: Full static** | All 17 structural checks, indicators, secret scanning | Epic 7 | Phase 1 |
| **Phase 3: Pipeline** | `pipeline.go`, quality gate, CI mode | Epic 6-7 | Phase 2 |
| **Phase 4: InstructionBench** | Benchmark design + execution | Epic 5 | Phase 1 (for before/after) |
| **Phase 5: Calibration** | Gold standards, 5-point rubric, Cohen's Kappa | Post-Epic 5 | Phase 4 results |

### Phase 0: Quick Wins (no architecture change)

**Task 0.1: CoT-before-score prompt**
- File: `internal/doctor/analyzer/analyzer.go` → `buildPrompt()`
- Change: add `reasoning` field before `confidence` in YAML example template
- Backward compatible: `llmutil/parse.go` ignores unknown fields
- Effort: 30 min

**Task 0.2: File-hash cache**
- New file: `internal/doctor/cache.go`
- Key: `sha256(content + prompt_template_hash + model + registry_version)`
- Storage: `.forgent/cache/doctor/<hash8>.json`
- TTL: 7 days, purge on `--fix`
- Effort: 2-3 hours

### Phase 1: Static Foundation

**Task 1.1: Check interface + registry**
- New package: `internal/doctor/static/`
- `check.go`: interface, Tags(), registration
- Pattern: same `init()` as `internal/linter/`
- Effort: 2 hours

**Task 1.2: Section presence check**
- Migrate from `context/enricher.go` gap detection
- 15 section checks (12 existing + identity, goals, constraints)
- Effort: 2 hours

**Task 1.3: Vague pattern detection**
- 17 regex patterns from AgentLinter research
- Must exclude code blocks (`inCodeFence` tracking already exists)
- Effort: 1-2 hours

**Task 1.4: Ordering check**
- Migrate from `context/enricher.go` CheckTextOrdering
- Check: critical sections (identity, guardrails, security) in first 33% of file
- Effort: 1 hour

**Task 1.5: Progressive display**
- Modify `render.go`: structural score + indicators first, then LLM findings
- Spinner during LLM analysis
- Effort: 3-4 hours

### Phase 2: Full Static Suite

**Task 2.1: Secret scanning**
- 17+ regex patterns: `sk-[a-zA-Z0-9]{48}` (OpenAI), `sk-ant-[a-zA-Z0-9-]{95}` (Anthropic), `AKIA[0-9A-Z]{16}` (AWS), `ghp_[a-zA-Z0-9]{36}` (GitHub), etc.
- Severity: always HIGH
- Effort: 2 hours

**Task 2.2: Compressible padding detection**
- Patterns: "Make sure to", "It is important to", "Please note that", "Keep in mind that"
- Suggests removal to reduce token waste
- Effort: 1 hour

**Task 2.3: Readability metrics**
- Flesch-Kincaid, Gunning Fog (pure Go, no dependency)
- Syllable counting via vowel-group heuristic
- Displayed as indicators, not pass/fail
- Effort: 2 hours

**Task 2.4: Token estimation**
- `chars/4 * 0.78` corrected heuristic (or tiktoken-go for exact)
- Per-section breakdown
- Total file budget warning (>8K warn, >16K critical)
- Effort: 1-2 hours

**Task 2.5: Instruction specificity ratio**
- Directive words ("must", "always", "never") / advisory ("may", "consider", "could")
- Displayed as indicator (0.0-1.0 scale)
- Effort: 1 hour

**Task 2.6: Duplicate detection**
- Jaccard similarity on instruction-like lines
- Threshold: ≥0.5 = near-duplicate warning
- Effort: 2 hours

### Phase 3: Pipeline + Gate

**Task 3.1: pipeline.go**
- Orchestrate: resolve checks → run static → render → run LLM (cached) → merge → post-process → gate
- Effort: 4-6 hours

**Task 3.2: gate.go**
- AND-logic conditions from config
- Default: secrets==0, structural≥50%, high≤5
- Effort: 2 hours

**Task 3.3: CI mode**
- `--ci` flag: auto-detect, plain text, quality gate, exit codes
- `--format json` for programmatic integration
- GitHub annotations output
- Effort: 3-4 hours

### Technology Adoption

| Dependency | Version | Purpose | When |
|---|---|---|---|
| goldmark | v1.7+ | Markdown AST (replaces manual parsing) | Phase 2+ optional |
| goldmark-meta | v1.1+ | YAML frontmatter extraction | With goldmark |
| tiktoken-go | v0.1+ | Accurate token counting | Phase 2 optional (heuristic works) |
| None new for Phase 0-1 | | Pure Go, no new deps | Immediate |

### Total Estimated Effort

| Phase | Tasks | Effort |
|---|---|---|
| Phase 0 | 2 tasks | ~3 hours |
| Phase 1 | 5 tasks | ~10 hours |
| Phase 2 | 6 tasks | ~10 hours |
| Phase 3 | 3 tasks | ~10 hours |
| **Total v2 pipeline** | **16 tasks** | **~33 hours** |

---

## Executive Summary

### The 10 things we learned

1. **No validated quality metric exists** for agent files. Every benchmark measures agent runtime, not file quality.
2. **Structure beats content.** 2-3 focused skills = +20pp; text instructions = +4% at best.
3. **70% of useful checks are static** — section presence, ordering, duplicates, keywords, secrets, readability. LLM needed for only 30% (semantic quality).
4. **CoT-before-score is a free +10-15%** alignment improvement. Change the prompt, not the code.
5. **Content-hash caching** makes repeat runs instant. Directory mode goes from 5min to <1s.
6. **Small local models (7B) are unreliable judges** for nuanced quality scoring. Use them for binary structural checks only.
7. **AgentLinter is the direct competitor** but has fabricated citations. Packmind is adjacent (platform, not diagnostic).
8. **68 checks identified**, 54 static. Organized by tags for --quick/--thorough/--security modes.
9. **3-layer output** (structural score + indicators + findings) is defensible. Single aggregate scores are not.
10. **InstructionBench is the missing piece** — first benchmark to prove file quality → agent performance correlation.

### What to build next

| Priority | Action | Effort | Impact |
|----------|--------|--------|--------|
| **Now** | CoT prompt change | 30 min | Better LLM analysis |
| **Now** | LLM response cache | 3 hours | Instant repeat runs |
| **Sprint 4** | InstructionBench design | 1 week | Proves Doctor works |
| **Sprint 5** | Static check package (17 checks) | 1 week | Structural scoring |
| **Sprint 5** | Progressive display | 3-4 hours | UX leap |
| **Sprint 6** | Full pipeline (gate, CI, JSON) | 1 week | Production-ready |

### What NOT to build

- Single aggregate score (0-100) — arbitrary weights, no validation
- 385 format rules like agnix — format compliance is not quality
- Proprietary platform like Packmind — stay open, stay local
- Fine-tuned judge model — insufficient training data, premature optimization

---

**Research complete.** Document: `_bmad-output/planning-artifacts/research/technical-agent-quality-analysis-research-2026-03-31.md`
