---
stepsCompleted:
  - step-01-init
  - step-02-discovery
  - step-02b-vision
  - step-02c-executive-summary
  - step-03-success
  - step-04-journeys
  - step-05-domain
  - step-06-innovation
  - step-07-project-type
  - step-08-scoping
  - step-09-functional
  - step-10-nonfunctional
  - step-11-polish
  - step-12-complete
inputDocuments:
  - docs/index.md
  - docs/architecture.md
  - docs/project-overview.md
  - docs/development-guide.md
  - docs/source-tree-analysis.md
  - docs/ROADMAP.md
  - docs/plans/2026-03-11-research-agent-context-effectiveness.md
  - content/whitepaper/_index.md
workflowType: 'prd'
documentCounts:
  briefs: 0
  research: 1
  brainstorming: 0
  projectDocs: 6
  whitepaper: 1
classification:
  projectType: 'Agent Skill Specification, Compiler & Runtime'
  domain: 'Agent DevOps / Behavioral Standards'
  complexity: 'High'
  projectContext: 'Vision document — grounded in feasibility-proven codebase'
  oneLiner: 'Forgent makes any AI agent better — locally, privately, for free.'
  strategicInsights:
    - 'Doctor is the Trojan Horse — zero-friction front door, analyzes any agent format'
    - 'Compiler is the engine — YAML spec, 5 facets, multi-target compilation'
    - 'Runtime is the platform — standalone binary, graph engine, pluggable providers'
    - 'Local LLM for doctor — no API key, no cost, private by default'
    - 'Research-backed recommendations — every criterion cites published findings'
    - 'SOLID principles applied to agent skills — proven engineering'
    - 'Contract-level portability, not runtime behavioral equivalence'
    - 'Structural guarantees, not semantic — whitepaper is honest about limits'
    - 'Benchmark (SWE-bench) is evidence, not a product'
    - 'Existing codebase is feasibility proof, not the shipped product'
    - 'AGENTS.md is evidence of the problem, not competition'
    - 'Adoption pulled by need, never pushed'
  risks:
    - 'Doctor requires local LLM for meaningful analysis'
    - 'Format round-trip fidelity per target — needs extensive testing'
    - 'Better must be benchmarked — runtime vs text, not better text vs worse text'
    - 'No existing benchmark for instruction quality — InstructionBench fills the gap'
    - 'Doctor IS a Forgent runtime — the proof of concept is the product'
    - 'Text instructions give +4% at best — structural enforcement gives +20-33%'
    - 'Dry-run default required — rewriting user files is high-trust'
---

# Product Requirements Document - Forgent

*Vision document — grounded in feasibility-proven codebase*

**Author:** Guillaume
**Date:** 2026-03-27

## How to Read This Document

This is a vision document grounded in a feasibility-proven codebase. Sections build progressively: strategic context (Executive Summary through Innovation), implementation specification (Project Scoping through NFRs). The Functional Requirements are the capability contract — every feature must trace to an FR. The Non-Functional Requirements define quality attributes. For implementation sequencing, see Project Scoping & Phased Development.

## Executive Summary

Forgent makes any AI agent better — locally, privately, for free.

AI agent development has no engineering discipline. Skills are unvalidated text files. Orchestration relies on prompts that LLMs routinely ignore. Agent logic is trapped in platform-specific formats across 15+ incompatible tools, with no portability between providers. The industry recognizes the problem — the Linux Foundation is standardizing file locations (AGENTS.md), NIST has launched an AI Agent Standards Initiative — but no existing effort addresses behavioral contracts, typed composition, static validation, or cross-target compilation. The surface is being standardized. The substance remains the Far West.

**Why now:** The agent ecosystem has fragmented across 15+ tools with no behavioral standard. LLM providers are commoditizing rapidly. Research (CodeScout 2026, SkillsBench 2026) has proven that structured context and skill decomposition measurably improve agent performance. The conditions are right: fragmentation creates demand for unification, commoditization creates demand for portability, and the absence of standards creates space for a specification to emerge. This window won't close suddenly — like Docker and Terraform, the need for a neutral layer grows with fragmentation — but moving now establishes the formal foundation before competitors deepen.

Forgent addresses this through three product layers:

**1. Doctor — the front door.** `forgent doctor` analyzes any agent file — CLAUDE.md, Copilot agents, Cursor rules — and produces research-backed recommendations to improve it. Fixes are applied in-place, in the user's own format, via a local LLM. No migration, no new format to learn, no API key, no cost. Every recommendation cites published research: primacy bias ordering (Liu et al., 2024), structured context gains (CodeScout, 2026), skill decomposition benefits (SkillsBench, 2026). The user's agents get better. Their workflow doesn't change.

**2. Compiler — the engine.** When users need portability or composition guarantees, the underlying skill specification emerges. Skills are defined in YAML with explicit I/O contracts (`consumes`/`produces`) and five behavioral facets: context, strategy, guardrails, observability, and security. The compiler validates, lints, scores, enriches with codebase context, and compiles to multiple targets — Claude Code, GitHub Copilot, or any future platform. One spec, multiple targets, validated before any LLM runs. This layer is the formal foundation: the whitepaper proves 15 structural properties including decidable well-formedness, compatible substitution, canonical parallelization, and verified cross-target isomorphism. These guarantees are structural, not semantic — they ensure correct wiring and composition. Behavioral testing is the planned path to close the semantic gap.

**3. Runtime — the platform.** For workflows that demand deterministic execution, the Forgent runtime compiles skill specs into standalone Go binaries where a graph engine orchestrates LLM calls with structural guarantees. Execution order, contract enforcement, routing, and retry are handled by the runtime, not the LLM. Provider adapters make the LLM layer pluggable — Anthropic, OpenRouter, or local models via Ollama. The current formal foundation is a DAG; bounded cycles for iterative refinement are the planned extension.

Each layer delivers value independently and leads naturally to the next. Doctor gets users in the door with zero commitment. The compiler serves power users who want portability and validation. The runtime serves teams who need guarantees and vendor independence. Adoption is pulled by need, never pushed.

**Feasibility.** The existing codebase demonstrates that each layer is buildable: a working compiler with three build targets, a DAG engine with 12 node kinds, an enrichment pipeline using TF-IDF ranking and AST symbol extraction, an import pipeline with LLM decomposition, and 18 skill definitions powering 3 agent compositions. The whitepaper provides formal grounding with 15 proven propositions. This is a vision document — the code proves the vision is achievable.

The guiding design principle: **the spec is the constant — everything else is a variable.**

### What Makes This Special

**Zero-friction entry.** Doctor works on any agent format, runs locally, requires no migration. No other tool improves agents in-place without asking you to adopt a new format.

**Research-backed, not opinion-based.** Every recommendation traces to published research. Every structural guarantee traces to a formal proof. The whitepaper establishes credibility that no competing tool can match.

**LLM independence by design.** From doctor (local LLM) to runtime (pluggable providers), Forgent never locks you into a vendor. The spec is the constant. The model is a variable.

## Project Classification

| Field | Value |
|---|---|
| **Project Type** | Agent Skill Specification, Compiler & Runtime |
| **Domain** | Agent DevOps / Behavioral Standards |
| **Complexity** | High |
| **Project Context** | Vision document — grounded in feasibility-proven codebase |
| **Tech Stack** | Go 1.24, Cobra CLI, graph engine, tree-sitter AST, local LLM integration |
| **Build Targets** | Claude Code, GitHub Copilot, Standalone Go Runtime |
| **Foundation** | Whitepaper: 15 proven structural properties |

## Success Criteria

### User Success

- **Eye-opening diagnosis.** Doctor reveals problems the developer didn't know about — AND shows why text instructions alone have limits. The user thinks "I can't unsee this."
- **Measurable improvement.** For text-level fixes (primacy ordering, codebase context), the improved agent performs noticeably better. For structural issues, doctor clearly explains why instructions aren't enough and offers the runtime path.
- **Effortless and safe.** Dry-run by default — doctor never touches files without explicit consent. Confidence levels on every recommendation. Automatic backup on write. The user is never harmed by a false positive.
- **Natural escalation.** Doctor diagnoses the limits of text instructions and offers structural enforcement via the Forgent runtime. The upgrade path is pulled by evidence, not pushed by marketing.

### Business Success

- **3-month milestone:** Doctor works on Claude Code + GitHub Copilot formats. InstructionBench shows doctor recommendations are ≥ 70% useful. Runtime vs text-instruction benchmark shows measurable delta on complex tasks.
- **6-month milestone:** Doctor supports ≥ 3 formats. Runtime used internally for ≥ 1 real workflow (e.g., doctor itself as a Forgent agent). First public demo or blog post with aggregate analysis of real-world agent files.
- **12-month milestone:** External teams adopt the Forgent YAML spec. Ecosystem forming — community skills, contributed generators. The spec is what teams reach for when they need portable, composable, validated agent definitions.
- **Validation signal:** Adoption pulled by need, never pushed. Users arrive via doctor, stay for the spec, scale with the runtime.

### Technical Success

- **Doctor analysis quality.** ≥ 70% of recommendations accepted as valid by developers on a test corpus. Every recommendation cites published research. Confidence scoring (high/moderate/low) on every recommendation. Local LLM (Ollama) produces reliable classification.
- **Rewrite safety.** 100% of rewritten files parse and function correctly in their native tool. Zero tolerance for format-breaking rewrites. Verified on ≥ 30 real-world files per supported format.
- **Runtime vs text benchmark.** Forgent runtime agents outperform text-instruction agents on complex multi-step tasks. Controlled experiment: same model, same tools, same tasks. Delta is statistically significant or ≥ 15% on primary metric.
- **Doctor is a Forgent runtime.** Doctor itself is defined as a Forgent agent (skill YAML + agent YAML), compiled to the Forgent runtime with Ollama provider. Proves the entire stack: spec, compiler, DAG engine, runtime, provider adapter.

### Measurable Outcomes

| Metric | Target | Timeframe |
|---|---|---|
| Doctor format support | Claude Code + GitHub Copilot | MVP |
| Recommendation usefulness | ≥ 70% accepted as valid | MVP |
| Rewrite safety | 100% files parse correctly after rewrite | MVP |
| Default mode | Read-only analysis, no file changes | MVP |
| MVP recommendation set | 5-8 core recommendations, each research-backed | MVP |
| Runtime vs text delta | Statistically significant on complex tasks | MVP |
| Doctor as Forgent agent | Doctor is a compiled Forgent runtime | MVP |
| Corpus | ≥ 100 public agent files scraped from GitHub | MVP |
| Additional format support | ≥ 3 formats total | 6 months |
| Typed contracts | Phase 1 complete | 6 months |
| External spec adopters | ≥ 1 team, no prior relationship | 12 months |
| Community contributions | ≥ 1 external contribution | 12 months |

## Scope Boundaries

For detailed MVP definition, implementation sequencing, timeline, and phased roadmap, see **Project Scoping & Phased Development** below.

### Explicitly Out of MVP Scope

The following are deferred to Growth or Vision phases — do not implement for MVP:

- npm distribution, Homebrew tap
- Managed runtime hosting
- Behavioral testing (`forgent test`)
- Bounded DAG cycles
- VS Code extension, LSP
- Typed contracts (Phase 1 — moved to 6-month milestone)
- SWE-bench evaluation (Growth benchmark)
- Deep decomposition (`--deep`)
- Format version pinning
- `--json` / SARIF output
- Cursor, Windsurf, Gemini CLI format support
- `--export-yaml` (Growth — bridge from doctor to compiler)

## User Journeys

### Journey 1: Alex — The Solo Developer (Doctor User)

**Situation:** Alex is a backend developer who's been using Claude Code for 3 months. They wrote a CLAUDE.md with instructions for code review and a couple of skills they copied from a blog post. Their agents work... sometimes. Sometimes Claude skips the test step. Sometimes it ignores the "don't modify the database" rule. Alex doesn't know why — they assume it's the model being unpredictable.

**Opening Scene:** Alex hears about Forgent on Hacker News. The headline says "Make any AI agent better — one command." Skeptical, they install it: `go install github.com/mirandaguillaume/forgent@latest`.

**Rising Action:** Alex runs `forgent doctor .claude/agents/code-reviewer.md`. In 30 seconds, the report appears with 5 findings — guardrails buried in paragraph 3, no timeout, monolithic structure, no security declarations, and a note that text instructions are ignored 23% of the time on ordering tasks. Alex stares at finding #1. That's exactly why Claude keeps skipping the test step — the "run tests first" instruction is in the middle of the file, not at the top.

**Climax:** Alex runs `forgent doctor .claude/agents/code-reviewer.md --fix`. They review the diff — guardrails moved to the top, timeout added, security section included. They apply findings 1, 2, and 4. Skip 3 for now. Next code review, Claude runs the tests first. Every time.

**Resolution:** Alex starts running `forgent doctor` on every agent file. The tool becomes part of their workflow — like running a linter before committing. They tell their team lead about it.

---

### Journey 2: Alex — When Doctor Gets It Wrong (Failure Path)

**Situation:** Alex has been using `forgent doctor --fix` for a week. They trust it. Today they run it on a skill that uses a custom Claude Code hook triggered by a specific markdown section header.

**Opening Scene:** Alex runs `forgent doctor agent.md --fix` and applies all recommendations without reviewing the diff carefully. Doctor restructured the file for primacy bias — moving the guardrails section above the custom section header that triggers the hook.

**Rising Action:** Alex's agent breaks. The hook no longer fires because the section it matches was moved. Alex panics — the agent was working fine an hour ago.

**Climax:** Alex notices the terminal output: `Backup saved: agent.md.bak`. They run `cp agent.md.bak agent.md`. The original file is restored. Agent works again.

**Resolution:** Alex re-runs `forgent doctor agent.md` — read-only mode this time. They review each recommendation, apply 1, 2, and 4 but skip 3 (the restructuring that broke the hook). Doctor is useful but not omniscient. The trust isn't lost — it's calibrated. Alex uses `--diff` from now on.

---

### Journey 3: Priya — The Team Lead (Compiler User)

**Situation:** Priya leads a team of 6 developers. Half use Claude Code, half use GitHub Copilot. They've written agent skills on both platforms — independently, inconsistently.

**Opening Scene:** Alex shows Priya `forgent doctor`. She runs it across all 12 of her team's agent files. The aggregate picture is alarming: 6 out of 12 have no guardrails, 4 have inconsistent tool lists, and the Copilot agents are missing security declarations entirely. She thinks: "This is a team-wide problem, not an individual one."

**Rising Action:** Priya discovers `forgent doctor --export-yaml`. Doctor converts the agents into Forgent skill YAML — a portable format with explicit I/O contracts, guardrails, and security facets. She sees that the two "code reviewer" agents are actually different skills. She standardizes the YAML, then runs `forgent build --target claude` and `forgent build --target copilot`.

**Climax:** One spec. Two targets. Both outputs have identical guardrails, identical tool mappings, identical structure. The linter catches a dependency issue she'd never noticed — one skill consumed a type that nothing produced. She fixes it, rebuilds. Both platforms are now consistent and validated.

**Resolution:** Priya mandates Forgent for the team. The YAML specs go into the repo. Changes are PRed like code. `forgent lint` runs in CI. The consistency problems disappear.

---

### Journey 4: Marcus — The Engineering Director (Runtime User)

**Situation:** Marcus runs engineering at a 50-person startup. They've built a CI review pipeline using Claude Code agents. It works 80% of the time. The other 20%: Claude skips the linting step, outputs 47 review comments when the limit is 15, or decides to approve code that failed tests.

**Opening Scene:** Priya's team has been using Forgent for 2 months. She shows Marcus the `forgent doctor` report on their CI pipeline. Finding #5 reads: "Text instructions are ignored 23% of the time on ordering tasks. Structural enforcement available via Forgent runtime."

**Rising Action:** Marcus asks: "What's the runtime?" Priya shows him: `forgent build --target forgent --provider anthropic`. The pipeline compiles into a standalone Go binary. The DAG engine enforces execution order. The contracts enforce data flow. The guardrails are enforced by the runtime, not by hoping Claude follows them.

**Climax:** Marcus runs the compiled pipeline on the same PRs that failed with text-based agents. Structural failures drop to near-zero — every step executes in order, guardrails are enforced, the scorer can't approve without test results present. Output quality still depends on the model, but the pipeline itself never breaks. He swaps `--provider openrouter`, selects a cheaper model. Same pipeline. Same guarantees. 40% cost reduction.

**Resolution:** Marcus rolls out the Forgent runtime for all CI agent pipelines. The team evaluates models quarterly based on cost and quality — switching is a flag change, not a rewrite.

---

### Journey 5: Kenji — The Agent Framework Author (Runtime Power User)

**Situation:** Kenji built a documentation generation framework — a multi-step workflow with 8 agents that scan code, extract docstrings, generate API references, create tutorials, and assemble a documentation site. It runs on Claude Code using markdown instructions. The 8 agents are coordinated by a master prompt that says "run these in order." Sometimes they run out of order.

**Opening Scene:** Kenji learns about Forgent's runtime target. He realizes his 8 documentation agents are actually 8 skills with clear consumes/produces contracts.

**Rising Action:** Kenji defines each agent as a Forgent skill YAML. The linter immediately catches that his tutorial generator depends on `api_reference`, but that skill runs in parallel with the API reference generator — a race condition hidden in his text-based orchestration. He restructures the DAG.

**Climax:** He compiles with `forgent build --target forgent --provider ollama`. The documentation pipeline runs locally — no API costs during development. The DAG engine executes in the correct order, every time. He switches to `--provider anthropic` for production runs.

**Resolution:** Kenji publishes his documentation skills on GitHub as reusable Forgent YAML. Other teams compose them into their own workflows, swapping skills for their specific languages and frameworks.

---

### Journey 6: Dr. Lena — The Researcher (Benchmark User)

**Situation:** Dr. Lena is a PhD researcher studying agent effectiveness. She wants to measure whether agent instruction quality impacts task performance — but no benchmark exists for this.

**Opening Scene:** Lena discovers Forgent's InstructionBench — a purpose-built benchmark for agent instruction quality with a corpus of 100+ real-world agent files from public GitHub.

**Rising Action:** She uses `forgent doctor` to generate improved versions of the corpus — creating controlled experimental conditions (original vs doctor-improved). She runs both versions on representative tasks, measuring resolution rate, token usage, and pass@k consistency.

**Climax:** Her paper demonstrates that structurally improved instructions yield +12% consistency (pass^k) and -18% token usage with no regression in task completion.

**Resolution:** InstructionBench becomes a referenced benchmark in the agent evaluation literature. Other researchers contribute agent files to the corpus. The benchmark grows beyond Forgent's own tooling.

---

### Journey 7: Sam — The Contributor (Ecosystem Builder)

**Situation:** Sam is a developer who uses Cursor daily. Forgent doctor works on Claude and Copilot, but not on Cursor's `.mdc` rule files.

**Opening Scene:** Sam reads the Forgent architecture docs. Target generators implement `pkg/spec.TargetGenerator` and register via `init()`. The pattern is clear — each target is a pluggable module.

**Rising Action:** Sam writes a Cursor target: a parser for `.mdc` files and a generator that outputs improved `.mdc` files. They follow existing Claude and Copilot generators as templates. Tests validate format round-trip fidelity on 30 real Cursor rule files.

**Climax:** Sam opens a PR. CI runs, tests pass. The Forgent team reviews for format fidelity. It merges.

**Resolution:** `forgent doctor` now supports Cursor. Sam's contribution benefits every Cursor user. The ecosystem grows one target at a time, contributed by the people who use those platforms.

---

### Journey Requirements Summary

| Journey | Capabilities Revealed | Scope |
|---|---|---|
| **Alex — Happy Path** | Doctor analysis, research citations, dry-run, in-place rewrite, Ollama | MVP |
| **Alex — Failure Recovery** | Automatic backup, restore, granular apply, diff preview | MVP |
| **Priya — Team Lead** | Team-wide analysis, --export-yaml, multi-target build, CI lint | MVP + Growth |
| **Marcus — Eng Director** | Runtime compilation, DAG enforcement, provider swapping | Existing + MVP |
| **Kenji — Framework Author** | Skill YAML authoring, complex DAG, local dev with Ollama | Existing + MVP |
| **Dr. Lena — Researcher** | Doctor for experiment prep, InstructionBench, measurement methodology | MVP |
| **Sam — Contributor** | Target generator plugin, format parser, contribution workflow | Existing + Growth |

**Key insight:** Each journey naturally escalates commitment — doctor → compiler → runtime → ecosystem. No journey requires understanding the full stack. Each user gets value at their level and discovers the next level when they need it. The failure journey builds trust by showing recovery is easy and obvious.

## Domain-Specific Requirements

### Domain Context: Agent Behavioral Standards

Forgent operates in a pre-paradigmatic domain. No standard exists for defining, composing, or validating AI agent behaviors. The ecosystem has fragmented across 15+ tools with 4 incompatible format families (declarative markdown, rules-based, config-based, programmatic). The window for establishing a behavioral specification is open but contested — AGENTS.md (Linux Foundation), Open Agent Spec (Oracle), and NIST's AI Agent Standards Initiative are all early-stage efforts addressing surface-level concerns.

### Domain Risks & Mitigations

**1. Platform format instability.**
Platforms change agent formats without notice — Claude Code, Copilot, and Cursor are all actively evolving. A format change breaks Forgent's parsers and generators overnight.

*Mitigation:* The target generator plugin architecture isolates format-specific code. Each target is an independent module with its own parser and generator. A format change affects one module, not the entire system. Format fidelity tests (≥ 30 real files per target) detect breakage immediately. Community contributors distribute maintenance across the ecosystem. Format changes in major platforms are tracked via CI monitoring of platform documentation and sample files. Target-specific fixes are prioritized as P0 — broken format support is treated as a production outage. Format version pinning (Growth roadmap) allows users to stay on a known-good format version until ready to upgrade.

**2. Platform-native orchestration.**
Platforms are shipping their own orchestration — Claude Code hooks and subagents, Copilot agent mode, Cursor agent features. If platforms solve orchestration natively, Forgent's runtime value proposition weakens.

*Mitigation:* Platform-native orchestration is platform-specific — it deepens lock-in, not solves it. Forgent's runtime provides CROSS-PLATFORM orchestration with FORMAL guarantees (15 proven properties). As platforms diverge in their orchestration approaches, the need for a neutral specification layer increases. Additionally, the doctor front door is independent of the runtime — doctor's value doesn't depend on the runtime being needed.

**3. Competing specification.**
AGENTS.md, Oracle Open Agent Spec, or a NIST standard gains traction as the default agent definition format.

*Mitigation:* Forgent's formal foundation — 15 proven structural properties, a composition algebra, and a published whitepaper — creates a credibility moat that cannot be replicated quickly. Competing specs can copy the format; they cannot copy the proofs. Current competitors address surface concerns only. Forgent operates at a deeper layer and can coexist with or generate output compatible with any surface-layer standard that emerges.

**4. LLM capability leap.**
Models become near-perfect at following text instructions, reducing the value of structural enforcement.

*Mitigation:* Even with perfect instruction following, structural benefits remain: static validation before runtime (catching errors without LLM cost), composition guarantees (skills wire correctly), cross-target portability (one spec, multiple platforms), and cost optimization (DAG minimizes LLM calls). The value shifts from "enforcement" to "efficiency and portability" — but doesn't disappear. Additionally, doctor's value as a diagnostic tool is independent of enforcement — even perfect models benefit from well-structured instructions.

**5. Industry convergence.**
The agent ecosystem consolidates around one dominant platform, eliminating fragmentation. No fragmentation = no portability need = weakened product thesis.

*Mitigation:* The trend is toward MORE fragmentation, not less — Kiro, OpenCode, Zed AI, and others launched in 2025-2026. However, even in a single-platform world, doctor's diagnostic value persists (instruction quality matters regardless of platform count), the runtime's structural enforcement persists (orchestration problems are platform-independent), and the formal specification has standalone value as engineering discipline for agents.

### Domain Assumptions (Must Monitor)

| Assumption | Break Signal | Monitoring Mechanism |
|---|---|---|
| LLMs imperfect at instruction following | >99% adherence | Run InstructionBench on each major model release. If runtime vs text delta drops below 5%, assumption weakening. |
| Local LLMs capable enough | Doctor quality <50% | Track recommendation acceptance rate across releases. Downward trend triggers investigation. |
| Multiple providers coexist | One provider >80% share | Quarterly market report check. If >80% of Forgent users use one provider, leading indicator. |
| Agent definitions remain text-based | Visual/code-only primary | Monitor major platform releases. If 2+ platforms deprecate text definitions, trigger strategy review. |
| Fragmentation continues | One platform >70% adoption | Track agent tool landscape quarterly. Consolidation below 5 major tools triggers strategy review. |

### Technical Constraints (Domain-Specific)

**Format fragility.** Agent file formats are undocumented, convention-based, and evolving. There is no formal spec for CLAUDE.md structure, Copilot agent frontmatter, or Cursor `.mdc` format. Forgent must reverse-engineer these conventions and maintain fidelity through empirical testing, not spec compliance.

**LLM non-determinism.** Doctor uses a local LLM for analysis and rewriting. LLM outputs are non-deterministic — the same input may produce different recommendations on different runs. Recommendations must be validated for consistency (pass@k) and the tool must be transparent about confidence levels.

**Context window limits.** Doctor sends agent files to a local LLM for analysis. Large agent files (CLAUDE.md with embedded codebase context) can exceed 10,000 tokens. Local 7B models with 4K-8K context windows may require chunked analysis. Minimum recommended: local model with ≥ 8K token context window.

**Evaluation gap.** No industry benchmark exists for agent instruction quality. Forgent must create InstructionBench to prove its own claims — an unusual position where the product must build its own evaluation methodology because none exists.

**Pre-paradigmatic domain.** Best practices for agent behavioral design don't exist yet. Forgent's recommendations are grounded in research, but the research itself is early-stage (2024-2026). Recommendations may need to evolve as the field matures.

**Research registry versioning.** Research citations are maintained as a versioned registry, not hardcoded. When new research contradicts or supersedes existing citations, the registry is updated and doctor's recommendations adjust accordingly. Users can see which research version their recommendations are based on.

## Innovation & Novel Patterns

Forgent's innovation is not any single concept but the formal combination: declarative specification + proven structural properties + multi-target compilation + standalone runtime + in-place improvement. Each element has precedent in developer tooling. The combination, and the formal grounding, do not.

### Detected Innovation Areas

**1. Doctor as Trojan Horse — novel application of a proven pattern.**
The lint-and-fix pattern is established (ESLint, Prettier, gofmt). Applying it to agent definitions — reading any format, improving it in-place via local LLM, outputting the same format with research-backed improvements — is new. The invisible compiler (YAML as IR, not interface) means users benefit from the spec and formal properties without ever seeing them.

**2. Research-backed recommendations — first versioned academic registry in a dev tool.**
Citation-backed tooling exists (SonarQube/CWE, security scanners/CVE). A living, versioned academic research registry driving agent quality recommendations is new. Every recommendation traces to a specific published paper. The registry is updatable as the field evolves — not hardcoded heuristics.

**3. Doctor as own runtime — strategic self-validation.**
Bootstrapping (compiling the compiler with itself) is established practice (GCC, Go, Rust). The strategic value is that doctor IS the proof of concept — every `forgent doctor` run exercises the full stack (spec, compiler, DAG engine, runtime, provider adapter). No separate demo needed. The product proves the platform.

**4. InstructionBench — creating a missing benchmark category.**
Benchmark creation for new capability areas is established practice (SWE-bench for software engineering, HumanEval for code generation). A PRODUCT building its own evaluation methodology because none exists is unusual positioning. InstructionBench fills a real academic gap — no benchmark tests agent instruction quality in isolation. The circularity risk (self-serving benchmark) is mitigated by open corpus, reproducible methodology, and external contributions.

**5. Structural enforcement — the concept exists, the formalization is novel.**
Structural orchestration exists in multiple frameworks (DSPy compiles LLM calls, LangChain chains, CrewAI sequences). The FORMALIZATION — 15 proven structural properties from a declarative YAML specification — is genuinely novel. No competing approach has formal grounding. The compilation from declarative spec to structural enforcement (not code to code) is the differentiator.

**6. SOLID for agents — formal proof, not just analogy.**
Applying software engineering principles to new domains is standard practice. PROVING Liskov substitution (Proposition 2 — contravariant substitutability), dependency inversion (skills depend on contracts, not on each other), and composability (Proposition 7 — structural additivity) formally for agent skills is a novel academic contribution. The whitepaper doesn't borrow SOLID as metaphor — it proves SOLID properties hold.

### Market Context & Competitive Landscape

| Competitor | What they do | Overlap with Forgent | What Forgent adds |
|---|---|---|---|
| AGENTS.md (Linux Foundation) | Standardize file location/loading | Surface-level agent standards | Behavioral contracts, composition, formal proofs, compilation |
| Open Agent Spec (Oracle) | Declarative YAML agent format | Declarative spec concept | Formal properties, multi-target compilation, runtime, doctor |
| DSPy (Stanford) | Compile declarative LLM calls into optimized pipelines | Declarative I/O signatures, compilation | YAML (not Python), multi-target, formal proofs, agent-level (not call-level) |
| Semantic Kernel (Microsoft) | Programmatic agent orchestration, multi-provider | Multi-LLM provider support | Declarative spec (not code-first), formal proofs, compilation |
| AutoGen (Microsoft) | Multi-agent conversation framework, evolving toward declarative config | Multi-agent orchestration | Declarative spec, cross-target compilation, formal proofs, doctor |

**Honest positioning:** Forgent is the only tool that combines DECLARATIVE specification + FORMAL properties + MULTI-TARGET compilation into a STANDALONE runtime with IN-PLACE agent improvement. Competitors overlap on individual features but not on the full stack. Competitors are evolving — this landscape must be monitored.

### Validation Approach

| Innovation | How to validate |
|---|---|
| Doctor Trojan Horse | Track funnel: doctor users → spec adopters → runtime users. Adoption pulled, not pushed. |
| Research registry | ≥ 70% recommendation acceptance. Registry updates don't break existing recommendations. |
| Doctor as own runtime | Doctor works reliably = stack is proven. InstructionBench validates independently. |
| InstructionBench | External researchers use or cite the benchmark. Corpus grows beyond Forgent's own contributions. |
| Structural enforcement | Runtime vs text delta on InstructionBench is statistically significant. |
| SOLID formal proofs | Whitepaper withstands peer review. External implementations of the formal model emerge. |

### Risk Mitigation

| Innovation Risk | Mitigation |
|---|---|
| Doctor recommendations wrong too often | Defensive defaults (dry-run). Confidence levels. ≥ 70% acceptance threshold before shipping. |
| InstructionBench perceived as self-serving | Open corpus, reproducible methodology, external contributions invited. Benchmark has standalone value even without external adoption. |
| Structural enforcement thesis disproven | Doctor's diagnostic value persists regardless. Research registry adapts as evidence evolves. |
| SOLID framing feels academic/intimidating | Users never see the formal model. Doctor and compiler apply SOLID invisibly. Whitepaper is for credibility, not onboarding. |
| Recursive proof of concept is circular | InstructionBench validates doctor independently, breaking the circularity. |
| Competitors catch up | Formal proofs create a credibility moat that takes years to replicate. Speed of execution matters — establish the standard before competitors deepen. |

## Project Scoping & Phased Development

### MVP Strategy & Philosophy

**Approach: Platform MVP** — prove the entire Forgent stack works end-to-end through doctor as a Forgent runtime agent. The goal is complete stack validation: spec → compiler → DAG engine → runtime → provider adapter → user-facing product.

**Resource Model:** Solo architect with LLM development team. Guillaume makes design decisions, reviews output, and maintains architectural coherence. LLM agents handle implementation across parallel workstreams. The PRD and existing codebase are the implementation specification.

**De-risking priority:** Local LLM quality is the highest-risk assumption. The first engineering spike validates whether a 7B local model can reliably analyze agent files. Test first, decide later.

### Implementation Sequencing

Based on actual codebase gaps (existing infrastructure in parentheses):

```
Spike: LLM quality validation                    [1 week]
  → Phase A: Doctor Analysis                      [1 week]
      (adapts existing importer parsers + linter rules)
  → Phase C: Doctor as Forgent Runtime            [3-5 days]
      (runtime generator + DAG engine exist)
  → Phase D: Benchmark                            [1 week]
      (adapts existing H2H framework)
  → Phase B: Doctor Rewrite (--fix)               [1-2 weeks]
      (format-preserving rewrite — genuinely new, hardest part)

Total: 4-6 weeks with LLM team
```

**Ship order:** Analysis + runtime proof first (Phases A+C). Benchmark in parallel (Phase D). Rewrite follows (Phase B). Users get value from read-only analysis while rewrite is built. User feedback guides which recommendations get auto-fix priority.

### MVP Feature Set (Phase 1)

**Core Journey Supported:** Alex (solo dev happy path + failure recovery)

**Must-Have Capabilities:**

| Capability | Existing Code | Gap | Risk |
|---|---|---|---|
| Claude format parser | `internal/importer/` — source resolution, framework detection, frontmatter extraction | Adapt for analysis (not conversion) | Medium |
| Copilot format parser | Same importer — already detects `.github/` | Same adaptation | Medium |
| 5-8 research-backed recommendations | `internal/linter/` (6+ rules), `internal/analyzer/score.go` | Rules work on Forgent YAML, need rules for raw markdown | High (LLM quality) |
| Confidence levels | `internal/analyzer/score.go` — multi-facet scoring | Adapt from YAML to raw analysis | Low |
| Research citation registry | Nothing | Full gap — new system | Low (effort only) |
| Dry-run default + diff preview | Nothing for in-place rewrite | New, but Go diff libraries exist | Low |
| `--fix` with interactive selection + backup | `internal/cmd/import.go` has `--yes` pattern | Minimal interaction exists, need proper prompts | Medium |
| `--restore` from backup | Nothing | Trivial — file copy | Low |
| Doctor as Forgent agent YAML | `skills/` has 18 examples as templates | Design work — author 7 doctor skills | Medium |
| Compile to Forgent runtime | `internal/generator/forgent/` — full runtime generator | Exists and works | Low |
| Ollama provider | `internal/llm/openrouter.go` — routes to Ollama via OpenRouter | Dedicated Ollama adapter is new | Medium |
| API fallback | `internal/llm/anthropic.go` — direct Anthropic | Exists | Low |
| InstructionBench corpus | Nothing | GitHub scraping — new but low complexity | Low |
| Runtime vs text benchmark | `internal/bench/h2h.go` — H2H evaluation | Adapt for instruction-quality comparison | Medium |

**Engineering spike (before full MVP):**
- Test 3+ local models (Phi-3, Llama 3, Mistral) on 20 real agent files
- Measure recommendation accuracy vs cloud API baseline
- Decision gate: if local accuracy < 50% of cloud accuracy, pivot to hybrid approach

### Post-MVP Features

**Phase 2 (Growth — 6 months):**
- Doctor for Cursor, Windsurf, Gemini CLI formats
- `forgent doctor --deep` — full decomposition via cloud API
- Typed contracts — schema-level I/O validation
- Homebrew tap distribution
- `--json` output for CI integration
- Format version pinning
- Adapted SWE-bench evaluation
- Expanded Hugo site documentation
- Blog post: "We analyzed 100 CLAUDE.md files"

**Phase 3 (Vision — 12+ months):**
- Doctor as universal tool — every agent format
- Forgent YAML as industry standard
- InstructionBench as independent benchmark
- Community ecosystem — shared skills, contributed generators
- Bounded cycles in DAG engine
- `forgent test` — behavioral testing
- Runtime as production orchestration platform
- Multi-agent coordination via MCP/A2A
- npm wrapper distribution
- VS Code extension / LSP
- Managed runtime hosting (seed only)

### Risk Mitigation Strategy

**Technical Risks:**

| Risk | Probability | Impact | Mitigation |
|---|---|---|---|
| Local LLM quality insufficient | Medium-High | High | Engineering spike first. Measure before committing. API fallback ready. |
| Format round-trip breaks files | Medium | High | ≥ 30 test files per format. Dry-run default. Automatic backup. Zero tolerance. |
| Runtime-vs-text delta is zero | Medium | High | Careful experiment design. If zero, pivot to efficiency/consistency metrics. |
| Doctor-as-runtime adds complexity | Low | Medium | Existing infrastructure is stable. Risk is in doctor SKILLS, not platform. |

**Market Risks:**

| Risk | Probability | Impact | Mitigation |
|---|---|---|---|
| Nobody cares about agent instruction quality | Low | Fatal | Zero-friction entry reduces "caring" threshold. One command to try. |
| Platform format changes break doctor | High | Medium | Plugin architecture. P0 response time. CI monitoring. |
| Competing spec gains traction | Low-Medium | Medium | Credibility moat (formal proofs). Add competing spec as build target. |

**Resource Risks:**

| Risk | Probability | Impact | Mitigation |
|---|---|---|---|
| LLM team output quality | Medium | Medium | PRD as precise spec. Existing codebase as reference. Architect reviews all output. |
| Scope creep from vision ambition | Medium | Medium | Clear phase boundaries in this PRD. MVP scope is locked. |

## CLI Tool & Developer Tool Requirements

### Project-Type Overview

Forgent is a hybrid CLI tool + developer tool. The CLI is the primary interface for all three product layers (doctor, compiler, runtime). The developer tool aspects include the YAML spec format, the target generator plugin system, and the documentation/examples ecosystem. Distribution via Go install + goreleaser + Homebrew broadens reach beyond the Go ecosystem.

### Command Structure

**Doctor commands (MVP):**
```
forgent doctor <file>              # Analyze agent file, show recommendations (with diffs inline)
forgent doctor <file> --fix        # Interactive: select which recommendations to apply
forgent doctor <file> --restore    # Restore from backup
```

Note: The existing `forgent doctor` (lint + dependency + loop diagnostics on Forgent YAML) is replaced by the new doctor. The new doctor handles all formats including Forgent's own YAML — subsuming and deleting the old command.

**Existing commands (stable):**
```
forgent init                       # Scaffold project
forgent skill create <name>        # Create skill YAML
forgent lint [path]                # Lint skills
forgent score [path]               # Quality scoring
forgent build --target <t>         # Compile to target
forgent build --watch              # Live rebuild
forgent import <source>            # Import agent .md → skill YAML
```

### Interactivity Model

TTY-aware — detect environment and adapt:
- **Terminal (TTY):** Pretty colored output, interactive prompts for `--fix`, progress indicators
- **CI/Pipe (no TTY):** Plain text output, no interactive prompts, exit codes for scripting
- **Growth:** `--json` flag for structured output, SARIF for IDE integration

### Output Formats

- **MVP:** Human-readable terminal output only (colored via fatih/color)
- **Growth:** `--json` for structured output, SARIF for IDE integration
- **Vision:** LSP for skill YAML editing

### Installation & Distribution

| Method | Target Audience | Status |
|---|---|---|
| `go install github.com/mirandaguillaume/forgent@latest` | Go developers | Existing |
| GitHub releases (goreleaser) | All platforms, pre-built binaries | Existing |
| Homebrew tap | macOS/Linux developers | Growth |
| npm wrapper package | JavaScript/TypeScript developers | Vision |

### Configuration

- **Project config:** `forgent.yaml` in project root (existing via `forgent init`)
- **Skill definitions:** `skills/*.skill.yaml` and `agents/*.agent.yaml`
- **Doctor config (MVP design deliverable):** Recommendation severity thresholds, research registry version, default provider — schema TBD
- **Provider config:** LLM provider selection (`--provider` flag or config file)

### Documentation Strategy

**MVP:**
- README with doctor quick start
- Comprehensive `--help` on every command with examples
- One real-world example: run doctor on a CLAUDE.md, show before/after

**Growth:**
- Expand Hugo/Hextra site with doctor docs, skill examples, getting started guides
- Migration guides: "From CLAUDE.md to Forgent" and "From Copilot agents to Forgent"
- Skill example library for common use cases

### Implementation Considerations

**Ollama dependency detection:**
- On startup, check if Ollama is available
- If present: full doctor functionality with local LLM
- If absent: static-only analysis (superficial) with clear message + offer API fallback (`--provider anthropic` with API key)
- No user hits a dead end — local is default, API is fallback

**Shell completion:**
- Bash, Zsh, Fish, PowerShell completion via Cobra built-in
- Complete file paths for doctor, target names for build

**Error handling:**
- Clear error messages with suggested fixes
- `forgent doctor --debug` for verbose output
- Exit codes: 0 (success/no findings), 1 (error), 2 (recommendations found but not applied)

## Functional Requirements

### Agent Analysis

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

### Recommendations Engine

- **FR12:** Doctor produces 5-8 research-backed recommendations per agent file, each citing a specific published paper
- **FR13:** Each recommendation includes a confidence level (high, moderate, low)
- **FR14:** Each recommendation explains the issue, why it matters (with citation), and the suggested improvement
- **FR15:** Recommendations are sourced from a versioned research registry that can be updated
- **FR16:** Doctor can diagnose the limits of text-based instructions and suggest when structural enforcement would be more appropriate
- **FR17:** Doctor notifies the user when a newer research registry version is available
- **FR18:** Developer can update the research registry via `forgent doctor --update-registry`

### Agent Rewriting

- **FR19:** Doctor defaults to read-only analysis — no file modifications without explicit user consent
- **FR20:** Developer can preview the exact diff that each recommendation would produce
- **FR21:** Developer can select which recommendations to apply individually
- **FR22:** Doctor creates an automatic backup before any file modification
- **FR23:** Developer can restore an agent file from backup via `forgent doctor --restore`
- **FR24:** Rewritten Claude files retain all original frontmatter fields, preserve user custom sections, and pass Claude Code's internal parser validation
- **FR25:** Rewritten Copilot files retain all original frontmatter fields, preserve user custom sections, and pass Copilot's parser validation

### Skill Specification & Composition

- **FR26:** Developer can define a skill in YAML with explicit I/O contracts and five behavioral facets
- **FR27:** Developer can define an agent as a composition of skills with orchestration strategy
- **FR28:** Developer can scaffold a new project with `forgent init`
- **FR29:** Developer can create a new skill scaffold with `forgent skill create <name>`
- **FR30:** Developer can lint skills for structural issues
- **FR31:** Developer can score skill quality (0-100) with improvement suggestions
- **FR32:** Developer can validate that all skills in an agent compose correctly
- **FR33:** Skill and agent YAML specs include a version field with compiler-validated compatibility

### Multi-Target Compilation

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

### Runtime Execution

- **FR44:** Compiled Forgent runtime binary executes skill pipelines with deterministic ordering via DAG engine
- **FR45:** Runtime enforces I/O contracts — a skill cannot execute until its consumed types are available
- **FR46:** Runtime supports pluggable LLM providers via flag or config
- **FR47:** Runtime supports router nodes that branch execution based on intermediate results
- **FR48:** Runtime supports retry logic for failed skill executions
- **FR49:** Runtime reports clear error diagnostics on node failure — which node, what input, failure reason
- **FR50 (KEYSTONE):** Doctor itself is defined as a Forgent agent and compiled to the Forgent runtime — proving the platform through the product

### Benchmarking & Evidence

- **FR51:** Developer can run InstructionBench to measure recommendation quality across a corpus
- **FR52:** Developer can run a controlled runtime-vs-text experiment with reproducible results
- **FR53:** Contributor can run format fidelity tests for their target generator against a standard test corpus

### Ecosystem & Extensibility

- **FR54:** Contributor can add a new target generator by implementing `pkg/spec.TargetGenerator` and registering via `init()`
- **FR55:** Developer can list all available target generators via `forgent targets`
- **FR56:** Developer can export doctor's analysis as Forgent skill YAML files (`--export-yaml`)
- **FR57:** Developer can configure doctor defaults via config file or environment variables
- **FR58:** Developer can generate shell completion scripts for Bash, Zsh, Fish, and PowerShell

## Non-Functional Requirements

### Performance

- **NFR1:** Doctor analysis completes within 30 seconds for a typical agent file (< 5,000 tokens) on hardware meeting Ollama's recommended specs. Within 90 seconds on CPU-only hardware.
- **NFR2:** Doctor analysis completes within 60 seconds for large agent files (5,000-15,000 tokens) on recommended hardware.
- **NFR3:** Compiler build completes within 5 seconds for a project with ≤ 20 skills and 3 agents.
- **NFR4:** Enrichment pipeline completes within 15 seconds for a codebase of ≤ 50,000 LOC.
- **NFR5:** CLI startup time is under 200ms — tool feels instant.
- **NFR6:** Forgent binary size remains under 50MB for all platforms.
- **NFR7:** Doctor analysis uses less than 500MB RAM (excluding the Ollama process).

### Reliability

- **NFR8:** Format-preserving rewrite produces zero broken files — 100% of rewritten files retain valid YAML frontmatter (parseable by yaml.v3) and valid Markdown structure (no broken headers, lists, or code blocks) on a test corpus of ≥ 30 files per format.
- **NFR9:** Compiler produces identical output for identical input — deterministic builds verified by content hash comparison.
- **NFR10:** Backup is created BEFORE any file modification — write failure cannot result in data loss.
- **NFR11:** Doctor gracefully handles malformed agent files — produces partial analysis rather than crashing.
- **NFR12:** Runtime DAG execution handles node failures gracefully — clear error reporting, no silent failures, no orphan processes.
- **NFR13:** All existing tests pass after any change — `go test ./... -count=1 -race` is the CI gate.
- **NFR14:** Forgent upgrades preserve all user configuration, skill YAML files, and doctor config. No user data is lost during updates.
- **NFR15:** Concurrent doctor runs on the same file do not corrupt it — file operations use appropriate locking.
- **NFR16:** Research registry updates are verified by checksum — corrupted or tampered registries are detected and rejected.
- **NFR17:** Doctor backups are cleaned up after 7 days by default — `.bak` files don't accumulate indefinitely. Retention is configurable.

### Privacy & Data Handling

- **NFR18:** By default, neither agent file content nor codebase content scanned during enrichment leaves the developer's machine — local LLM is the default provider.
- **NFR19:** When API fallback is used, the user is explicitly informed that content will be sent to an external provider.
- **NFR20:** No telemetry or usage data is collected without explicit opt-in.
- **NFR21:** Doctor does not store or cache agent file content beyond the current execution.
- **NFR22:** When doctor makes any network request (API fallback, registry update), the target URL is visible to the user before the request is made. No silent network calls.

### Integration Compatibility

- **NFR23:** Doctor correctly parses agent files produced by current stable versions of Claude Code and GitHub Copilot, and supports formats from at least the last 6 months of platform releases.
- **NFR24:** Compiled output is compatible with the current stable versions of Claude Code and GitHub Copilot.
- **NFR25:** Ollama integration works with any model available via `ollama list` — not hardcoded to specific model names.
- **NFR26:** API fallback works with Anthropic Messages API and OpenRouter API.
- **NFR27:** Changes to the skill YAML schema maintain backward compatibility — existing skill files continue to work after upgrades. Breaking changes require a major version bump and migration guide.

### Developer Experience

- **NFR28:** Error messages include actionable suggestions — not just "failed" but "failed because X, try Y."
- **NFR29:** `--help` output for every command includes at least one usage example.
- **NFR30:** Exit codes are consistent and documented — 0 (success), 1 (error), 2 (recommendations found).
- **NFR31:** The tool works offline (with Ollama) — no network dependency for core functionality.
- **NFR32:** Forgent runs on macOS (arm64, amd64), Linux (arm64, amd64), and Windows (amd64) without additional dependencies beyond Ollama.
- **NFR33:** Doctor correctly handles UTF-8 encoded agent files including non-ASCII characters, emoji, and multilingual content without corruption during rewrite.
- **NFR34:** Doctor operations can be cancelled via Ctrl+C with clean exit — no orphan processes, no partial writes, backup remains intact.
- **NFR35:** Doctor LLM operations time out after a configurable duration (default: 120 seconds) with clear error message.
- **NFR36:** `--debug` mode produces structured diagnostic output sufficient to reproduce bugs — Ollama model used, prompt sent, response received, recommendation derivation steps.

### Testing

- **NFR37:** All new code maintains ≥ 80% test coverage. All packages have `*_test.go` files.
- **NFR38:** Race detector enabled in CI — `go test ./... -count=1 -race` passes cleanly.
