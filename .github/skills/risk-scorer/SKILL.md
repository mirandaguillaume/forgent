---
name: risk-scorer
description: diff-first-based skill consuming git_diff, test_results, lint_results to produce risk_score
---

# Risk Scorer

## Guardrails
- timeout: 3min
- require_risk_score

## Context
Consumes: git_diff, test_results, lint_results
Produces: risk_score
Memory: short-term

## Output Format

### Output: Risk_score

Output a risk score as:

**Risk: N/10** — one-sentence justification

Where N is based on:
- 0-3: Low risk (cosmetic, style)
- 4-6: Medium risk (logic, edge cases)
- 7-10: High risk (security, data loss, breaking changes)

---

## Résultat — branche `feat/scanner-enricher-bench` (2026-03-16)

**Risk: 7/10** — Ruptures d'interface publique sur `pkg/spec.TargetGenerator` et le comportement de `GenerateAgentMd`, combinées à une surface de changement exceptionnellement large (147 fichiers, 5 nouveaux packages, clients LLM en production), compensées par une couverture de tests solide à 92%+ et 706 tests verts.

### Détail par dimension

| Dimension               | Score | Justification |
|-------------------------|-------|---------------|
| Portée des changements  | 8/10  | 147 fichiers, 22 607 insertions, 5 nouveaux packages entiers |
| Changements cassants    | 7/10  | `TargetGenerator` éclatée en 4 interfaces ; `DependencyFacet` supprimé ; format orchestrateur modifié |
| Couverture de tests     | 2/10  | Facteur mitigant — 706/706 tests verts, 92%+ de couverture sur les packages modifiés |
| Nouvelles fonctionnalités | 5/10 | `EffortLevel`, format subagent, bench framework, scanner/enricher avec clients LLM réels |
| Résultats lint          | 1/10  | Facteur mitigant — 103 issues mineures, 0 critique |

## Strategy
Approach: diff-first
Tools: read_file, grep

### Steps
1. analyze git diff scope and complexity
2. cross-reference with test coverage
3. cross-reference with lint severity
4. assign risk score (0-10)

## Security
- Filesystem: read-only
- Network: none
