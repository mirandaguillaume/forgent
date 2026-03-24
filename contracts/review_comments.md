Provide your review as a structured list. For each finding:

1. **Severity**: critical | warning | suggestion
2. **Location**: `file:line`
3. **Issue**: One-sentence description
4. **Fix**: Concrete code suggestion or action
5. **Rationale**: Why this matters

Group findings by severity (critical first). End with a summary count.

---

# Review Comments — feat/scanner-enricher-bench → main

Branch : `feat/scanner-enricher-bench`
Tests  : 706/706 pass
Lint   : 103 mineurs, 0 critiques

---

## Critiques

Aucun problème bloquant identifié.

---

## Avertissements

### 1

**Severity**: warning
**Location**: `internal/generator/claude/agent.go:66`
**Issue**: Quand un skill figure dans `agent.Skills` mais pas dans `resolvedSkills`, le générateur émet l'en-tête `### Step N: X` suivi d'une ligne vide sans bloc `Launch a subagent`, produisant une étape vide dans le markdown final.
**Fix**: Déplacer l'ajout de l'en-tête à l'intérieur de la branche `if s.Skill == skillName` afin qu'il ne soit émis que si le skill est effectivement résolu, ou ajouter un commentaire `(skill not resolved)` explicite.

```go
// Avant (ligne 66-88)
lines = append(lines, fmt.Sprintf("### Step %d: %s", i+1, generator.ToTitle(skillName)))
effort := model.EffortMedium
for _, s := range resolvedSkills {
    if s.Skill == skillName {
        lines = append(lines, "Launch a subagent:")
        ...
        break
    }
}
lines = append(lines, "")

// Après — déplacer l'en-tête à l'intérieur du if
for _, s := range resolvedSkills {
    if s.Skill == skillName {
        lines = append(lines, fmt.Sprintf("### Step %d: %s", i+1, generator.ToTitle(skillName)))
        lines = append(lines, "Launch a subagent:")
        ...
        break
    }
}
```
**Rationale**: Le même problème existe dans `internal/generator/copilot/agent.go:66`. Un skill non résolu produit du markdown silencieusement cassé sans que le build échoue ni n'avertisse au niveau du générateur.

---

### 2

**Severity**: warning
**Location**: `internal/generator/copilot/effort.go:6`
**Issue**: `EffortToModel` du package `copilot` retourne `"haiku"`, `"sonnet"`, `"opus"` — des noms de modèles Anthropic — alors que GitHub Copilot utilise des modèles GPT/OpenAI.
**Fix**: Mapper vers les identifiants corrects pour Copilot (ex. `"gpt-4o-mini"`, `"gpt-4o"`, `"o1"`) ou documenter explicitement que ces noms sont intentionnellement génériques et laissés à l'utilisateur.
**Rationale**: Un agent Copilot généré avec `Model: haiku` sera incompréhensible pour un utilisateur GitHub Copilot ; cela peut induire des erreurs de configuration silencieuses.

---

### 3

**Severity**: warning
**Location**: `internal/generator/copilot/effort.go` (fichier entier)
**Issue**: `EffortToModel` est dupliqué à l'identique dans `internal/generator/claude/effort.go` et `internal/generator/copilot/effort.go` sans factorisation.
**Fix**: Extraire la logique commune dans `internal/generator/effort.go` (package `generator`) et appeler `generator.EffortToModel` depuis chaque package, ou au moins ajouter un commentaire signalant la divergence intentionnelle (noms de modèles différents par cible).
**Rationale**: Toute évolution future (ajout d'un niveau `EffortUltra`) devra être dupliquée manuellement, avec risque de désynchronisation.

---

### 4

**Severity**: warning
**Location**: `internal/generator/claude/agent.go:44`
**Issue**: La section `## Execution` et le texte d'orchestration sont toujours émis, même quand `resolvedSkills` est vide (ex. aucun skill chargé), ce qui produit un agent avec `Execute 2 skills sequentially...` mais aucun step.
**Fix**: Conditionner l'émission du bloc `## Execution` à `len(resolvedSkills) > 0`, ou au moins à `len(agent.Skills) > 0`.
**Rationale**: Le test `TestGenerateAgentMd_NoSkills` passe `nil` comme `resolvedSkills` mais `agent.Skills` reste `["code-review","security-scan"]` — l'output contient le texte d'orchestration sans aucune étape concrète.

---

### 5

**Severity**: warning
**Location**: `pkg/model/skill.go:143–148` (suppression de `DependencyFacet` / `DependsOn`)
**Issue**: Le champ `DependsOn DependencyFacet` a été supprimé du modèle `SkillBehavior` sans versioning ni note de migration : les fichiers `.skill.yaml` existants contenant `depends_on:` seront silencieusement ignorés par le loader YAML.
**Fix**: Ajouter un avertissement dans le loader YAML ou le linter quand un champ `depends_on` est détecté dans le fichier source (strict mode YAML ou vérification explicite).
**Rationale**: Un breaking change silencieux sur le format YAML de définition des skills peut casser des projets utilisateurs sans message d'erreur.

---

## Suggestions

### 6

**Severity**: suggestion
**Location**: `internal/generator/copilot/agent.go:35`
**Issue**: La valeur `tools: ["task"]` est un littéral de chaîne Go brut non construit programmatiquement, contrairement à `GenerateCompactCopilotAgentMd` (ligne 138) qui génère le format `tools: ["read", ...]` via `fmt.Sprintf`.
**Fix**: Utiliser une constante ou une construction cohérente :

```go
const copilotTaskTool = `tools: ["task"]`
lines = append(lines, copilotTaskTool)
```
**Rationale**: Incohérence de style entre les deux fonctions du même fichier ; risque d'erreur typographique lors d'une future modification.

---

### 7

**Severity**: suggestion
**Location**: `internal/generator/claude/agent.go:41`
**Issue**: Le corps de l'agent en mode dispatch ne contient plus la `description` de l'agent (remplacée par le template fixe `An orchestrator that coordinates N specialized subagents`), alors que `GenerateCompactAgentMd` (ligne 150) conserve la description dans le corps.
**Fix**: Soit inclure la description dans les deux modes de façon cohérente :

```go
lines = append(lines, fmt.Sprintf("You are %s. %s Orchestrates %d specialized subagents.",
    generator.ToTitle(agent.Agent), agent.Description, len(agent.Skills)))
```
soit documenter que le mode dispatch n'expose pas la description dans le corps.
**Rationale**: Incohérence comportementale entre `--compact` et mode standard ; peut surprendre les utilisateurs qui personnalisent la description.

---

### 8

**Severity**: suggestion
**Location**: `internal/generator/copilot/effort.go` (absence de fichier `effort_test.go`)
**Issue**: `EffortToModel` dans le package `copilot` n'a aucun test unitaire dédié, alors que le package `claude` possède `effort_test.go`.
**Fix**: Créer `internal/generator/copilot/effort_test.go` avec les quatre cas (`light`, `medium`, `heavy`, `""`).
**Rationale**: Couverture asymétrique entre les deux packages pour une fonction identique.

---

### 9

**Severity**: suggestion
**Location**: `internal/generator/claude/agent.go:114`
**Issue**: `formatIOPointers` est une fonction privée triviale (2 lignes) qui duplique partiellement la logique de formatage d'I/O déjà présente dans `internal/generator/util.go`.
**Fix**: Vérifier si `generator.FormatIO` ou une fonction similaire existe dans `util.go` et consolider, ou au moins ajouter un commentaire de justification.
**Rationale**: Fragmentation des helpers de formatage entre `util.go` (package `generator`) et les sous-packages.

---

### 10

**Severity**: suggestion
**Location**: `internal/generator/claude/agent_test.go:112`
**Issue**: `TestGenerateAgentMd_NoSkills` passe `resolvedSkills=nil` mais l'agent a toujours `Skills: ["code-review","security-scan"]` — la condition testée est donc `resolvedSkills vide` et non `agent.Skills vide` ; le cas `agent.Skills=[]` + `resolvedSkills=[]` n'est pas couvert.
**Fix**: Ajouter un test explicite :

```go
func TestGenerateAgentMd_EmptySkillsList(t *testing.T) {
    agent := testAgent()
    agent.Skills = nil
    md := claude.GenerateAgentMd(agent, nil, ".claude")
    assert.NotContains(t, md, "## Execution")
}
```
**Rationale**: La validation `ValidateAgent` rejette `len(Skills)==0`, mais ce chemin de génération reste non couvert.

---

### 11

**Severity**: suggestion
**Location**: `internal/generator/claude/agent_test.go` (absence de test d'étape orpheline)
**Issue**: Aucun test ne vérifie le comportement quand `agent.Skills` référence un skill absent de `resolvedSkills` (ex. skill déclaré mais non chargé).
**Fix**: Ajouter :

```go
func TestGenerateAgentMd_UnresolvedSkill_NoOrphanBlock(t *testing.T) {
    agent := testAgent() // Skills: ["code-review", "security-scan"]
    // Ne passer qu'un seul des deux skills
    md := claude.GenerateAgentMd(agent, testResolvedSkills()[:1], ".claude")
    // Vérifier que l'étape sans skill résolu ne produit pas de header vide
    assert.NotContains(t, md, "### Step 2: Security Scan\n\n")
}
```
**Rationale**: Le comportement actuel (header + ligne vide sans contenu) est silencieux et non spécifié.

---

### 12

**Severity**: suggestion
**Location**: `pkg/model/skill.go:169-178`
**Issue**: `WhenToUseFacet.IsEmpty()` ne vérifie que `len == 0` pour trois champs, ce qui est correct, mais la méthode n'est pas testée isolément — elle est uniquement exercée indirectement via les règles du linter et le score.
**Fix**: Ajouter un test unitaire minimal dans `pkg/model/skill_test.go` pour `IsEmpty()` avec un cas vide et un cas non-vide.
**Rationale**: Méthode publique sur un type de modèle central, sans test direct.

---

## Résumé

| Sévérité   | Nombre |
|------------|--------|
| Critique   | 0      |
| Avertissement | 5   |
| Suggestion | 7      |
| **Total**  | **12** |

Les tests passent tous (706/706). Les points les plus risqués sont l'étape orpheline en cas de skill non résolu (#1, #11) et le breaking change silencieux sur `depends_on` (#5). Le mapping de modèles Copilot (#2) est fonctionnellement incorrect pour les utilisateurs GitHub Copilot.
