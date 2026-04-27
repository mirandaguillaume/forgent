# Market Research: Agent Specification & Quality Tools (v3 — verified sources + competitive analysis)

**Date:** 2026-03-31 (v3)
**Methodology:** Sections 1-2 : uniquement données vérifiables — papers peer-reviewed, études empiriques, données GitHub mesurables. Projections d'analystes exclues. Sections 3+ : les outils identifiés via analyse compétitive sont marqués séparément. Les claims non vérifiés en peer-review sont signalés.

---

## 1. Données vérifiées sur l'adoption

### Adoption des AI coding tools (source : études empiriques)

| Donnée | Source | Fiabilité | Caveat |
|---|---|---|---|
| ~90% des repos configurés utilisent des context files (CLAUDE.md, AGENTS.md) | Configuring Agentic AI (2,926 repos) | **Haute** — mesure directe GitHub | Biais : repos qui configurent, pas tous les repos |
| CLAUDE.md dans 45.4% des repos configurés | Idem | **Haute** | Idem |
| AGENTS.md dans 40.6% des repos configurés | Idem | **Haute** | Idem |
| 2,303 context files analysés dans 1,925 repos | Agent READMEs study | **Haute** — étude empirique large | 3 outils seulement (Claude, Codex, Copilot) |
| 67.4% des context files modifiés plusieurs fois | Agent READMEs study | **Haute** | Modifié ≠ bien maintenu |
| Skills utilisés dans seulement 5.4% des repos (158 repos) | Configuring Agentic AI | **Haute** | Adoption avancée très faible |
| 83.3% des Skills n'ont aucune ressource additionnelle | Idem | **Haute** | "Documentation, not automation" |

### Impact mesuré des context files (source : études contrôlées)

| Donnée | Source | Fiabilité | Caveat |
|---|---|---|---|
| AGENTS.md réduit le temps médian de 28.64% | Lulla et al., 2026 (124 PRs, 10 repos) | **Moyenne** — petit échantillon, un seul agent (Codex) | Pas reproductible sur d'autres agents |
| Output tokens réduits de 16.58% (médian) | Idem | **Moyenne** | Idem |
| Résultats statistiquement significatifs (p<0.05) | Idem | **Haute** — méthodo correcte | Petit échantillon |
| Text instruction files : +4% au mieux, parfois négatif | ETH Zurich (138 instances, 12 repos) | **Haute** — peer-reviewed, méthodologie rigoureuse | SWE-bench seulement, Python seulement |
| LLM-generated context : -3% (négatif) | Idem | **Haute** | |
| Structured codebase context : +20% | CodeScout (SWE-bench) | **Haute** | Python seulement, pas open-source |
| RepoGraph : +32.8% avg across frameworks | RepoGraph (SWE-bench) | **Haute** — open-source, reproductible | Python seulement, GPT-4 seulement |
| 2-3 skills > monolith de 18.6pp | SkillsBench (84 tasks, 11 domains) | **Haute** — peer-reviewed | Définition de "skill" différente de Forgent |
| Smaller model + skills > larger model sans skills | SkillsBench (Haiku 27.7% vs Opus 22.0%) | **Haute** | |

### Ce qu'on NE SAIT PAS (données non vérifiées)

| Claim courant | Pourquoi douteux |
|---|---|
| "$7.6B marché AI Agents 2025" | Définition "AI Agent" trop large, inclut chatbots et RPA |
| "73% des teams utilisent AI tools daily" | Survey auto-sélectionné (Pragmatic Engineer readers = tech-forward) |
| "4% des commits GitHub par Claude Code" | First-party data Anthropic, pas vérifié par tiers |
| "$2.5B Claude Code run rate" | Estimation externe, jamais confirmé |
| "60,000 projets AGENTS.md" | Chiffre marketing, non vérifié en peer-review. L'étude Configuring Agentic AI a trouvé 2,926 repos avec configuration |

---

## 2. Le problème vérifié

### Problème 1 : Les context files sont de mauvaise qualité

**Evidence :**
- ETH Zurich : LLM-generated context files ont un impact NÉGATIF (-3%) — [arXiv:2602.11988](https://arxiv.org/abs/2602.11988)
- Agent READMEs : seulement 14.5% abordent la sécurité, 14.5% la performance — [arXiv:2511.12884](https://arxiv.org/abs/2511.12884)
- 83.3% des Skills n'ont aucune ressource — "documentation, not automation" — [arXiv:2602.14690](https://arxiv.org/abs/2602.14690)
- Readability Flesch score de 16.6 — "difficulté document légal" — [arXiv:2511.12884](https://arxiv.org/abs/2511.12884)

### Problème 2 : Peu d'outils de diagnostic pre-runtime, aucun ne couvre tout

**Evidence :**
- 5 outils d'évaluation post-exécution existent (Maxim AI, LangSmith, Arize, Langfuse, Galileo) — tous POST-exécution
- Au moins 6 outils de diagnostic pre-runtime identifiés (voir Section 3.4) — ils adressent des facettes spécifiques (biais, clarté, conformité format)
- **Aucun outil existant ne combine** : (a) multi-format natif (CLAUDE.md + AGENTS.md + Copilot + YAML), (b) catégories research-backed vérifiées par des papers peer-reviewed, (c) exécution locale (Ollama/local-first)
- Le paper "Configuring Agentic AI" conclut : "little empirical evidence on which configuration strategies are most effective" — [arXiv:2602.14690](https://arxiv.org/abs/2602.14690)

### Problème 3 : La structure aide, le texte seul non

**Evidence :**
- Text instructions : +4% au mieux (ETH Zurich)
- Structured context : +20-33% (CodeScout, RepoGraph)
- 2-3 skills composés > monolith (SkillsBench : +18.6pp)
- Structure compense la capacité du modèle (SkillsBench : Haiku + skills > Opus sans skills)

---

## 3. Paysage concurrentiel

### 3.1 Formats déclaratifs (données vérifiables)

| Format | Backer | Evidence d'adoption | Open source |
|---|---|---|---|
| AGENTS.md | Linux Foundation (OpenAI, Anthropic, Block) | 40.6% des repos configurés (Configuring Agentic AI) | Oui |
| Agent Skills | Anthropic | 5.4% des repos configurés | Oui |
| Agent Format | Snap | Pas de données d'adoption publiques | Oui |
| Open Agent Spec | Oracle/Prime Vector | Pas de données d'adoption publiques | Oui |
| Microsoft Declarative | Microsoft | Pas de données d'adoption indépendantes | Non |
| ADL | Academic | Paper seulement | Non |
| Auton AgenticFormat | Academic | Paper seulement | Non |
| Forgent | Guillaume Miranda | Pas d'adoption externe | Oui |

**Observation :** Seuls AGENTS.md et Agent Skills ont des données d'adoption mesurées par des études indépendantes.

### 3.2 Protocoles de communication

| Protocole | Backer | Evidence |
|---|---|---|
| MCP | Anthropic → Linux Foundation | De facto standard (supporté par tous les outils majeurs) |
| A2A | Google + 50 partners | Liste de partners publique, spec ouverte |
| ACP | Community | Mentionné dans surveys, adoption non mesurée |

### 3.3 Formal contracts (academic)

| Paper | Key concept | Status |
|---|---|---|
| Agent Contracts (Ye & Tan, 2026) | 7-tuple avec resource budgets | Accepté COINE/AAMAS 2026. Pas d'implémentation. |
| Behavioral Contracts (Li et al., 2026) | Hard/soft invariants + ContractSpec DSL | Paper. Pas d'implémentation publique. |
| Prompt Injection Patterns (Hossain et al., 2025) | 6 design patterns | Accepté IEEE S&P 2026. Patterns, pas un format. |

### 3.4 Quality Diagnostic Tools (pre-runtime)

> *Découverts via analyse compétitive, mars 2026. Claims des auteurs non vérifiés en peer-review sauf mention contraire.*

#### AgentLinter (seojoonkim) — Concurrent direct

- **Type :** Web app + CLI npm (`npx agentlinter`)
- **Approche :** 100% statique (regex + heuristiques, 0 LLM). 11 dimensions pondérées : Structure 10%, Clarity 15%, Completeness 10%, Security 13%, Consistency 6%, Memory 8%, Runtime Config 8%, Skill Safety 8%, Remote-Ready 5%, Blueprint 7%, Freshness 10%. Grading S/A/B/C/D/F.
- **Formats :** CLAUDE.md, AGENTS.md, .cursorrules, copilot-instructions.md, OpenClaw, Moltbot, Clawdbot
- **Auteur :** Simon Seojoon Kim (CEO de Hashed, fonds VC blockchain coréen). Side project solo (48/49 commits).
- **Traction :** 44 GitHub stars, ~2,574 npm downloads total, 5-25 downloads/jour
- **Citations research :**
  - "Gloaguen et al. (2026) — A Taxonomy of Agent Instruction Failures" — **TITRE INEXISTANT.** Thibaud Gloaguen est un vrai chercheur (premier auteur de arXiv:2602.11988, ETH Zurich SRI Lab) mais le paper "A Taxonomy of Agent Instruction Failures" n'existe pas. Son vrai paper est "Evaluating AGENTS.md: Are Repository-Level Context Files Helpful for Coding Agents?" — un titre complètement différent. Citation semi-hallucinée : nom réel, titre inventé.
  - Song et al. (TMLR 2026) "LLM Reasoning Failures" — **Paper réel** (arXiv:2602.06176) mais sur les failures de raisonnement LLM (math, logique), **pas sur le linting de fichiers agent**. Citation embellied.
  - Anthropic CLAUDE.md Best Practices — **Légitime.**
- **Faiblesses :** Citation fabriquée, version inflation (npm 0.3.3 vs README v2.3.0), analyse statique seulement (ne peut pas évaluer la qualité sémantique), ~30% des dimensions spécifiques à OpenClaw, mainteneur solo.
- **Forces vs Doctor :** Zero-config (pas besoin d'Ollama), cross-file consistency checks, secret scanning (17 regex), token budget analysis, instantané.

#### Packmind Context-Evaluator — Concurrent adjacent (funnel vers SaaS)

- **Type :** CLI binaire + web app (context-evaluator.ai)
- **Approche :** LLM-powered via agent CLI externe (Claude Code, Cursor, Copilot, Codex). 17 évaluateurs (13 erreurs + 4 suggestions). Génère des `.patch`.
- **Formats :** CLAUDE.md, AGENTS.md, .cursorrules, copilot-instructions.md, SKILL.md
- **Auteur :** Packmind (ex-Promyze, fondée 2016 Bordeaux). 13 employés, $1M seed (Nov 2023, profitable avant le round). AQUITI Gestion + Le Village by CA. SOC 2 Type II.
- **Traction :** 20 GitHub stars (context-evaluator), 252 stars (plateforme), 135 commits
- **Modèle :** Open-core Apache-2.0 mais split agressif — le moteur de linting, governance, change management sont **stubbés** dans l'édition OSS (`throw Error('Method not implemented.')`). 7/8 packages propriétaires n'existent pas dans le repo public. Communauté quasi-inexistante (2 contributeurs externes, 3 commits).
- **Citations research :** Aucune citation académique. Approche "empirique" basée sur l'analyse de fichiers réels.
- **Faiblesses :** Dépendance parasitaire à un agent CLI tiers (si Claude Code change son CLI, ça casse), nécessite abonnement agent ($20+/mois), Node 22 + Bun requis, pas de mode local/offline.
- **Forces vs Doctor :** Détection drift codebase, intégration dans plateforme de distribution enterprise, remediation pipeline 4 phases.
- **Stratégie :** Funnel vers plateforme payante. Prediction : intégration du context-evaluator dans le tier Enterprise.

#### skill-validator (agent-ecosystem) — Concurrent partiel

- **Type :** CLI open-source
- **Approche :** Content density, instruction specificity, novelty scoring, LLM-as-Judge. Multi-axes.
- **Formats :** Agent Skills uniquement (pas AGENTS.md, pas CLAUDE.md)
- **Citations research :** Référence implicite à ETH Zurich (skills qui restent common knowledge dégradent la performance)
- **Scope limité :** Skills only — pas un diagnostic d'agent files génériques.

#### PromptDoctor (académique) — Wildcard

- **Type :** VS Code extension
- **Approche :** Détection de biais, réparation de prompts, hardening prompt injection. De-biased 68.29%, hardened 41.81%, improved 37.1%.
- **Paper :** [arXiv:2501.12521](https://arxiv.org/abs/2501.12521) — **Peer-reviewed**, 2,173 prompts analysés.
- **Scope :** Prompts individuels, pas agent specification files. Si les auteurs élargissent, concurrent sérieux grâce à la crédibilité académique.

#### Prompt Linter (Ignire, VS Code) — Non concurrent

- **Type :** Extension VS Code
- **Approche :** Role clarity, conflits logiques, complexité, emphasis overuse. Uses Claude for analysis.
- **Scope :** Fichiers `.prompt` uniquement. Pas AGENTS.md/CLAUDE.md natif.

#### TaskLint — Non concurrent

- **Type :** Web app
- **Approche :** Scoring rubric-based de descriptions de tâches AI.
- **Scope :** Task descriptions, pas agent specification files.

### 3.5 Format Linting Tools

> *Focus conformité de format, pas qualité de contenu.*

| Outil | Règles | Formats | Notes |
|---|---|---|---|
| agnix (agent-sh) | 385 | CLAUDE.md, AGENTS.md, SKILL.md, hooks, MCP | Plus grand linter de format. VS Code/Neovim/Zed. Règles sourcées des specs officielles, pas de research-backing. |
| ccat (nwiizo) | N/A | CLAUDE.md uniquement | Analyse structurelle (imports circulaires, taille, secrets). Mono-format. |
| agentmd (PyPI) | N/A | Agent markdown files | Validation de schéma. CLI + library Python. |
| SkillCheck | N/A | Agent Skills (Anthropic) | Browser-based (WebAssembly). Pro tier : anti-slop, WCAG, security. |

### 3.6 Security Scanning Tools (agent/skill-specific)

| Outil | Auteur | Approche | Notes |
|---|---|---|---|
| Snyk Agent Scan | Snyk | 15+ risks, LLM judges + rules | ToxicSkills study : 36% des skills ont du prompt injection, 1,467 payloads malveillants |
| Cisco Skill Scanner | Cisco | Pattern + LLM + dataflow | Enterprise-grade, Codex Skills + Cursor Agent Skills |
| Sentry Skill Scanner | Sentry | Analyse statique | Prompt injection, code malveillant, permissions excessives, secrets |

**Observation :** Le security scanning agent est un segment émergent porté par des vendors sécurité établis. Complémentaire à Doctor, pas concurrent.

---

## 4. Position de Forgent (honnête, corrigée)

### Ce que Forgent a et que personne d'autre n'a

| Claim | Vérifié ? | Comment vérifier |
|---|---|---|
| ~~Seul outil de pre-runtime quality diagnostic~~ | **FAUX** — au moins 6 outils identifiés (Section 3.4) | Voir AgentLinter, Packmind, skill-validator, Prompt Linter, PromptDoctor, TaskLint |
| **Seul outil combinant 24 catégories research-backed vérifiées + multi-format natif + local-first (Ollama)** | **Oui** — aucun concurrent identifié combine ces 3 axes | Vérifier (a) registry YAML avec citations, (b) parsers multi-format, (c) support Ollama |
| Multi-target compilation (YAML → Claude + Copilot + runtime) | **Oui** — code vérifiable | `forgent build --target claude/copilot/forgent` |
| 15 preuves formelles | **Oui** — whitepaper publié | Vérifier les preuves dans le whitepaper |
| Runtime avec DAG engine + provider independence | **Oui** — code vérifiable | `forgent build --target forgent` |

### Détail du positionnement unique (3 axes)

1. **Research-backed categories (24)** : Chaque catégorie du registry `doctor` cite un paper peer-reviewed avec arXiv URL. Aucun concurrent ne documente ce niveau de traçabilité académique. AgentLinter cite une référence fabriquée et une loosely-connected. Packmind n'a aucune citation.
2. **Multi-format natif** : `doctor` parse nativement CLAUDE.md, AGENTS.md, Copilot agent files, et Forgent YAML. Les concurrents ciblent soit les prompts génériques (PromptDoctor, Prompt Linter), soit un format unique (ccat = CLAUDE.md, SkillCheck = Agent Skills), soit dépendent d'un agent tiers (Packmind).
3. **Local-first (Ollama)** : `doctor` fonctionne avec Ollama en local, sans envoyer les specs agent à un cloud tiers. Packmind nécessite un agent CLI cloud. AgentLinter est local mais statique (regex, pas d'analyse sémantique).

### Ce que Forgent n'a PAS

| Gap | Qui l'a | Priorité |
|---|---|---|
| Adoption externe | AGENTS.md (40.6%), Agent Skills (5.4%) | Critique — doctor fonctionne déjà sur ces formats |
| Cross-file consistency analysis | AgentLinter (cross-file checks) | Haute — Doctor analyse un fichier à la fois |
| Token budget estimation | AgentLinter (section weights, padding) | Moyenne — aligné ETH Zurich +19% tokens |
| Linting format (385 règles) | agnix | Moyenne — complémentaire (qualité vs conformité) |
| Security scanning dédié | Snyk, Cisco, Sentry | Basse — doctor couvre la sécurité comme 1 des 24 catégories |
| Détection drift codebase | Packmind Context-Evaluator | Moyenne — feature différenciante de Packmind |
| Zero-config (pas besoin d'Ollama) | AgentLinter (static, instant) | Moyenne — affichage progressif (structural instant → LLM) comme alternative |
| Resource budgets | Agent Contracts (paper) | Moyenne — le runtime peut l'implémenter |
| Hard/soft invariants | Behavioral Contracts (paper) | Moyenne — enrichir guardrails |
| Governance neutre (Linux Foundation) | AGENTS.md, MCP | Basse — Forgent est complémentaire |
| MCP/A2A integration | Écosystème | Basse — futur |

---

## 5. Analyse concurrentielle approfondie

### 5.1 Matrice concurrentielle directe (même business = diagnostic qualité agent files)

| | Forgent Doctor | AgentLinter | Packmind CE | skill-validator |
|---|---|---|---|---|
| **Moteur** | LLM (sémantique) | Static (regex) | LLM via agent tiers | LLM-as-Judge |
| **Catégories** | 24 | 11 | 17 | ~6 |
| **Citations vérifiées** | 12 papers arXiv | 0 (1 fabriquée, 1 embellied) | 0 | 1 (implicite) |
| **Multi-format** | 4 formats natifs | 7+ (dont OpenClaw/Moltbot) | 5 formats | Skills only |
| **Local/offline** | Ollama | Oui (static) | Non (agent cloud) | Configurable |
| **Rewrite/fix** | --fix + quality gate | --fix (basique) | .patch files | Non |
| **Cross-file** | Non | Oui | Oui | Non |
| **Coût** | Gratuit (Ollama) | Gratuit (static) | Gratuit mais nécessite agent $20+/mois | Gratuit |
| **Distribution** | Go binary | npm package | Node+Bun binary | CLI |
| **Mainteneur** | Guillaume Miranda | Solo (VC crypto, side project) | 13 employés, $1M seed | Community |
| **Maturité** | 3 epics, 14 stories | v0.3.3, 49 commits | v0.4.3, 135 commits | Early |

### 5.2 Carte du marché

```
FORMAT LINTING              QUALITY DIAGNOSTIC              SECURITY SCANNING
(syntaxe/conformité)        (efficacité comportementale)    (vulnérabilités)

agnix (385 rules)           Doctor (24 cat, LLM, research)  Snyk Agent Scan
ccat (structural)           AgentLinter (11 dim, static)    Cisco Skill Scanner
agentmd (schema)            Packmind CE (17 eval, LLM)      Sentry Skill Scanner
SkillCheck (browser)        skill-validator (density, LLM)
                            PromptDoctor (academic, prompts)

         POST-EXECUTION                    DISTRIBUTION/GOVERNANCE
         (runtime evaluation)              (standards management)

         Maxim AI, LangSmith              Packmind Platform
         Arize, Langfuse, Galileo         (open-shell, enterprise payant)
```

### 5.3 Menaces et opportunités

| Scénario | Probabilité | Impact | Action |
|---|---|---|---|
| Packmind intègre context-evaluator dans Enterprise payant | Élevée | Moyen — renforce argument "gratuit et local" | Communiquer la différence |
| Packmind ajoute des citations research-backed | Faible (pas leur ADN) | Élevé | Maintenir avance avec InstructionBench (Epic 5) |
| Packmind ajoute Ollama/local-first | Très faible (archi parasitaire) | Élevé | Non applicable |
| AgentLinter meurt (side project solo) | Élevée | Positif | Récupérer ses utilisateurs |
| PromptDoctor s'élargit aux agent files | Faible-Moyenne | Élevé (crédibilité académique) | Publier InstructionBench en peer-review |
| agnix ajoute du quality diagnostic | Faible (focus format) | Moyen | Positionner complémentarité |
| Snyk/Cisco ajoutent du quality diagnostic | Faible (focus sécurité) | Moyen | Pas le même marché |

---

## 6. Stratégie fondée sur les données (corrigée)

### Ce que les données disent

1. **Les context files sont partout** (90% des repos configurés) **mais de mauvaise qualité** (14.5% security, Flesch 16.6, 83% sans ressources)
2. **Le texte seul ne suffit pas** (+4% au mieux) **mais la structure aide** (+20-33%)
3. **Des outils de diagnostic pre-runtime existent** (au moins 6) **mais aucun ne combine multi-format + research-backed + local-first**
4. **Les garanties formelles n'existent qu'en théorie** (papers sans implémentation)
5. **Le linting format et le security scanning sont des segments émergents** avec des acteurs dédiés
6. **Le concurrent direct le plus visible (AgentLinter) a une citation fabriquée** — fragilité de crédibilité

### Ce que ça implique pour Forgent

- **Doctor (transversal)** : le marché est les ~3,000+ repos qui ont des context files aujourd'hui (et croissant). Pas les "millions de devs" — c'est du hype. **La pression compétitive existe mais est fragmentée** : chaque outil adresse un sous-problème.
- **Différenciation clé** : ne PAS se positionner comme "seul outil de diagnostic" (c'est faux et vérifiable). Se positionner comme **"diagnostic le plus complet et tracé académiquement, multi-format, local-first"**. C'est vérifiable et défendable.
- **Risque compétitif** : AgentLinter et skill-validator pourraient ajouter du multi-format. Packmind pourrait ajouter des citations. Le moat est la **profondeur du registry** (24 catégories, citations vérifiées) et l'**exécution locale**.
- **Complémentarité** : agnix (linting format) et Snyk/Cisco/Sentry (sécurité) sont complémentaires. Forgent peut recommander ces outils et se concentrer sur le diagnostic qualité.
- **Features à considérer** : cross-file analysis (gap vs AgentLinter), token budget estimation (aligné ETH Zurich), affichage progressif (structural instant → LLM analysis).
- **Runtime (concurrent)** : le marché des garanties structurelles n'est pas mesurable aujourd'hui — c'est un marché à créer.
- **Le whitepaper + 24 catégories research-backed** restent le credibility moat. Mais ce n'est plus un moat d'unicité totale — c'est un moat de **profondeur et de traçabilité**.

---

## Sources (vérifiées, peer-reviewed ou études empiriques)

- [ETH Zurich — Evaluating AGENTS.md (arXiv:2602.11988)](https://arxiv.org/abs/2602.11988)
- [Agent READMEs — Empirical Study (arXiv:2511.12884)](https://arxiv.org/abs/2511.12884)
- [CodeScout — Knowledge Graph (arXiv:2603.05744)](https://arxiv.org/abs/2603.05744)
- [RepoGraph — Code Graph as Tool (arXiv:2410.14684)](https://arxiv.org/abs/2410.14684)
- [SkillsBench — Benchmarking Skills (arXiv:2602.12670)](https://arxiv.org/abs/2602.12670)
- [Codified Context — 3-Tier Infrastructure (arXiv:2602.20478)](https://arxiv.org/abs/2602.20478)
- [On the Impact of AGENTS.md (arXiv:2601.20404)](https://arxiv.org/abs/2601.20404)
- [Configuring Agentic AI Coding Tools (arXiv:2602.14690)](https://arxiv.org/abs/2602.14690)
- [Context Engineering in OSS (arXiv:2510.21413)](https://arxiv.org/abs/2510.21413)
- [Agent Contracts (arXiv:2601.08815)](https://arxiv.org/abs/2601.08815)
- [Behavioral Contracts (arXiv:2602.22302)](https://arxiv.org/abs/2602.22302)
- [Prompt Injection Patterns (arXiv:2506.08837)](https://arxiv.org/abs/2506.08837)
- [Anthropic — Demystifying evals (anthropic.com)](https://www.anthropic.com/engineering/demystifying-evals-for-ai-agents)
- [Confucius Code Agent (arXiv:2512.10398)](https://arxiv.org/abs/2512.10398)
- [PromptDoctor (arXiv:2501.12521)](https://arxiv.org/abs/2501.12521)

## Sources additionnelles (analyse compétitive, non peer-reviewed)

> *Outils identifiés via analyse compétitive directe (GitHub, npm, VS Code Marketplace, sites produit). Claims des auteurs non vérifiés indépendamment sauf PromptDoctor.*

- AgentLinter — [GitHub](https://github.com/seojoonkim/agentlinter) | [agentlinter.com](https://agentlinter.com) — 11 dimensions, citation "Gloaguen 2026" non trouvée
- Packmind Context-Evaluator — [GitHub](https://github.com/PackmindHub/context-evaluator) | [context-evaluator.ai](https://context-evaluator.ai) — LLM via agent CLI
- Packmind Platform — [GitHub](https://github.com/PackmindHub/packmind) | [packmind.com](https://packmind.com) — open-shell, $1M seed
- skill-validator — [GitHub](https://github.com/agent-ecosystem/skill-validator) — content density + LLM-as-Judge
- Prompt Linter — [VS Code Marketplace](https://marketplace.visualstudio.com/items?itemName=Ignire.prompt-linter) — fichiers .prompt
- TaskLint — [tasklint.dev](https://www.tasklint.dev/) — task descriptions
- agnix — [GitHub](https://github.com/agent-sh/agnix) — 385 règles, multi-éditeur
- ccat — [GitHub](https://github.com/nwiizo/ccat) — analyse structurelle CLAUDE.md
- agentmd — [PyPI](https://pypi.org/project/agentmd/) — schema validation
- SkillCheck — [getskillcheck.com](https://www.getskillcheck.com/) — browser-based
- Snyk Agent Scan — [GitHub](https://github.com/snyk/agent-scan) — security scanning skills/MCP
- Cisco Skill Scanner — [GitHub](https://github.com/cisco-ai-defense/skill-scanner) — pattern + LLM + dataflow
- Sentry Skill Scanner — [GitHub](https://github.com/getsentry/skills) — analyse statique skills
