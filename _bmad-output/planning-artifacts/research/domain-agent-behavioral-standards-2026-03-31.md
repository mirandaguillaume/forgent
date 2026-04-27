# Domain Research: Agent Behavioral Standards and Specifications

**Date:** 2026-03-31
**Scope:** Comprehensive mapping of all published papers, specs, frameworks, and standards that define how AI agents are specified, composed, and governed. Focus: 2024-2026.

---

## 1. Declarative Agent Specification Formats

### 1.1 Agent Format (Snap, 2025)
- **What:** Declarative, runtime-agnostic YAML standard for defining AI agent systems
- **Origin:** Snap (Snapchat) — three engineering teams independently building agents led to standardization need
- **Governance:** Modeled after OpenAPI Initiative and CNCF — neutral stewardship
- **Scope:** Agent identity, interface, tools, constraints, governance, execution strategy
- **URL:** [agentformat.org](https://agentformat.org/) | [Snap Engineering Blog](https://eng.snap.com/agent-format)

### 1.2 Open Agent Specification (Oracle, 2024-2025)
- **What:** Declarative, framework-agnostic YAML for defining agents and workflows
- **Origin:** Oracle, with Prime Vector taking over public-facing project
- **Scope:** Unified representation across AutoGen, LangGraph, OCI Agents without reimplementation
- **Paper:** [arXiv:2510.04173](https://arxiv.org/abs/2510.04173)
- **GitHub:** [prime-vector/open-agent-spec](https://github.com/prime-vector/open-agent-spec)

### 1.3 Agent Skills (Anthropic, 2025)
- **What:** Folder-based skill format — SKILL.md with YAML frontmatter + structured markdown body
- **Origin:** Anthropic (Claude Code) — released as open standard, adopted by OpenAI Codex CLI
- **Scope:** Task-specific instructions, scripts, resources. Discovery via frontmatter.
- **Spec:** [agentskills.io](https://agentskills.io/specification)
- **GitHub:** [anthropics/skills](https://github.com/anthropics/skills)

### 1.4 AGENTS.md (Linux Foundation, 2025)
- **What:** Standardized file location and loading for agent context files
- **Origin:** Community initiative, now under Linux Foundation / Agentic AI Foundation
- **Scope:** Surface-level — WHERE files live, HOW they're loaded. No behavioral contracts.
- **URL:** [agents.md](https://agents.md/)

### 1.5 ADL — Agent Declarative Language (2025)
- **What:** YAML-based language for declaring agent chatbots and their relationships
- **Origin:** Academic (Sirui Zeng et al.)
- **Scope:** Agent definitions, inter-agent routing, conversation flows
- **Paper:** [arXiv:2504.14787](https://arxiv.org/abs/2504.14787)

### 1.6 Auton AgenticFormat (2026)
- **What:** Language-agnostic declarative schema separating "Cognitive Blueprint" from "Runtime Engine"
- **Origin:** Academic
- **Scope:** Agent interface, tool bindings, memory configuration, safety constraints
- **Paper:** [arXiv:2602.23720](https://arxiv.org/abs/2602.23720)

### 1.7 Microsoft Declarative Agents (2025-2026)
- **What:** YAML/JSON-based agent configuration in Microsoft Agent Framework
- **Origin:** Microsoft (Copilot ecosystem)
- **Scope:** Agent config without programmatic code — easier to define, modify, share
- **URL:** [Microsoft Learn — Declarative Agents](https://learn.microsoft.com/en-us/agent-framework/agents/declarative)

### 1.8 Forgent Skill Behavior Model (2026)
- **What:** YAML-based skill spec with 5 behavioral facets + agent composition + multi-target compilation
- **Origin:** Guillaume Miranda
- **Scope:** I/O contracts, guardrails, security, observability, orchestration. Compiles to Claude, Copilot, standalone runtime.
- **Whitepaper:** 15 formal proofs, composition algebra
- **GitHub:** [mirandaguillaume/forgent](https://github.com/mirandaguillaume/forgent)

---

## 2. Formal Contract Frameworks (Academic)

### 2.1 Agent Contracts (Ye & Tan, 2026)
- **What:** 7-tuple formal framework C=(I,O,S,R,T,Φ,Ψ) — inputs, outputs, skills, resources, temporal, success criteria, termination
- **Key contribution:** Resource conservation laws for multi-agent delegation (∑R_i ≤ R_parent)
- **Status:** Accepted at COINE 2026 / AAMAS 2026. No implementation.
- **Paper:** [arXiv:2601.08815](https://arxiv.org/abs/2601.08815)

### 2.2 Agent Behavioral Contracts (Li et al., 2026)
- **What:** 6-tuple behavioral contracts with hard/soft invariants + recovery mechanisms
- **Key contribution:** Compositionality theorem (assume-guarantee reasoning), ContractSpec YAML-based DSL
- **Status:** Academic. ContractSpec DSL proposed but not implemented.
- **Paper:** [arXiv:2602.22302](https://arxiv.org/abs/2602.22302)

### 2.3 Design Patterns for Prompt Injection Security (Hossain et al., 2025)
- **What:** 6 design patterns providing provable resistance to prompt injection
- **Key patterns:** Action-Selector, Plan-Then-Execute, Map-Reduce, Dual-LLM, Code-Then-Execute, Context-Minimization
- **Status:** IEEE S&P 2026. Patterns, not a spec format.
- **Paper:** [arXiv:2506.08837](https://arxiv.org/abs/2506.08837)

---

## 3. Communication & Interoperability Protocols

### 3.1 MCP — Model Context Protocol (Anthropic, 2024)
- **What:** Standardizes LLM agent ↔ external tool/API interaction
- **Scope:** Tool access, resource provision, prompt injection boundaries
- **Governance:** Linux Foundation
- **URL:** [modelcontextprotocol.io](https://modelcontextprotocol.io/)

### 3.2 A2A — Agent-to-Agent Protocol (Google, 2025)
- **What:** Inter-agent communication — discovery, negotiation, delegation across frameworks
- **Scope:** Agent Cards (JSON capability descriptions), task management, streaming
- **Partners:** 50+ companies (Google, Atlassian, Salesforce, SAP, MongoDB)
- **URL:** [a2a-protocol.org](https://a2a-protocol.org/latest/)

### 3.3 ACP — Agent Communication Protocol (2025)
- **What:** Lightweight messaging protocol for agent-to-agent communication
- **Scope:** Complement to MCP (tools) and A2A (discovery)
- **Survey:** [arXiv:2505.02279](https://arxiv.org/abs/2505.02279)

### 3.4 ANP — Agent Network Protocol (2025)
- **What:** Decentralized agent discovery and collaboration
- **Scope:** Network-level agent interoperability
- **Survey:** [arXiv:2505.02279](https://arxiv.org/abs/2505.02279)

---

## 4. Industry Standards & Governance

### 4.1 NIST AI Agent Standards Initiative (2026)
- **What:** Integrator role — unifying agent protocols, security frameworks, interoperability standards
- **Scope:** Evaluation and certification system. Incorporates A2A, MCP, OWASP, IEEE P2894.
- **RFI:** Request for Information on AI Agent Security (January 2026)
- **URL:** [NIST CAISI](https://www.nist.gov/caisi/ai-agent-standards-initiative)

### 4.2 OWASP Top 10 for Agentic Applications (2026)
- **What:** 10 critical security risk categories for AI agents
- **Risks:** Goal hijacking, tool misuse, privilege abuse, missing guardrails, data poisoning, resource exhaustion, supply chain, prompt injection, data disclosure, over-reliance on autonomy
- **Principle:** "Least Agency" — agents receive minimum autonomy for authorized tasks
- **URL:** [OWASP Agentic Top 10](https://openclawai.io/blog/owasp-top-10-agentic-applications-2026-security/)

### 4.3 IEEE P2894 (in progress)
- **What:** Standard for agent interoperability
- **Status:** Referenced by NIST as part of the integrator framework

---

## 5. Programmatic Agent Frameworks (Code-First)

### 5.1 DSPy (Stanford, 2024)
- **What:** Compile declarative LLM calls into optimized pipelines
- **Approach:** Signatures (I/O declarations), modules, optimizers
- **Paper:** [arXiv:2310.03714](https://arxiv.org/abs/2310.03714)

### 5.2 LangGraph (LangChain, 2024-2025)
- **What:** Graph-based agent orchestration with state machines
- **Approach:** Nodes, edges, conditional routing, human-in-the-loop

### 5.3 AutoGen (Microsoft, 2023-2026)
- **What:** Multi-agent conversation framework
- **Evolution:** Moving toward declarative config in AutoGen 0.4+
- **Paper:** [arXiv:2308.08155](https://arxiv.org/abs/2308.08155)

### 5.4 Semantic Kernel (Microsoft, 2023-2026)
- **What:** Programmatic agent orchestration, multi-provider
- **Approach:** C#/Python SDK, plugins, planners

### 5.5 CrewAI (2024-2025)
- **What:** Role-based multi-agent orchestration
- **Approach:** Agent config files, task definitions, crew composition

---

## 6. Research on Agent Context Quality

### 6.1 ETH Zurich — Evaluating AGENTS.md (2026)
- **Finding:** Text instruction files give +4% at best, sometimes negative. +19% token cost.
- **Paper:** [arXiv:2602.11988](https://arxiv.org/abs/2602.11988)

### 6.2 Agent READMEs — Empirical Study (2025)
- **Finding:** 2,303 context files analyzed. 67.4% modified multiple times. Security (14.5%) and performance (14.5%) rarely specified.
- **Paper:** [arXiv:2511.12884](https://arxiv.org/abs/2511.12884)

### 6.3 CodeScout — Pre-exploration via Knowledge Graph (2026)
- **Finding:** Pre-compiled structured context gives +20% on SWE-bench. Weak agents benefit +51.9%.
- **Paper:** [arXiv:2603.05744](https://arxiv.org/abs/2603.05744)

### 6.4 Codified Context — 3-Tier Infrastructure (2026)
- **Finding:** Hot/warm/cold context tiers. 24.2% of codebase as structured context reduces human prompts.
- **Paper:** [arXiv:2602.20478](https://arxiv.org/abs/2602.20478)

### 6.5 SkillsBench — Benchmarking Skills (2026)
- **Finding:** 2-3 skills optimal (+18.6pp). Self-generated skills negative. Structure compensates for model capability.
- **Paper:** [arXiv:2602.12670](https://arxiv.org/abs/2602.12670)

### 6.6 Context Engineering in OSS (2025)
- **Finding:** No established content structure. Field is pre-paradigmatic.
- **Paper:** [arXiv:2510.21413](https://arxiv.org/abs/2510.21413)

---

## 7. Landscape Summary

| Category | Count | Key Players |
|---|---|---|
| Declarative specs | 8 | Agent Format, Open Agent Spec, Agent Skills, AGENTS.md, ADL, Auton, Microsoft, Forgent |
| Formal contracts | 3 | Agent Contracts, Behavioral Contracts, Prompt Injection Patterns |
| Communication protocols | 4 | MCP, A2A, ACP, ANP |
| Governance standards | 3 | NIST, OWASP, IEEE P2894 |
| Code-first frameworks | 5 | DSPy, LangGraph, AutoGen, Semantic Kernel, CrewAI |
| Context quality research | 6 | ETH Zurich, Agent READMEs, CodeScout, Codified Context, SkillsBench, Context Engineering |
| **Total** | **29** | |

### What no one has (except Forgent):
- **Multi-target compilation** — one spec → multiple platform outputs
- **Doctor** — automated quality diagnostic with research-backed recommendations
- **Formal proofs** on a YAML-based declarative spec (15 propositions)
- **Working runtime** from a declarative spec with DAG execution
- **Enrichment pipeline** — codebase context injection at build time
