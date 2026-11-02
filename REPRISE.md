# Reprise — forgent (dossier local `ax-cli`)

## À quoi sert le projet

Forgent est un compilateur de skills d'agents : on décrit des skills en YAML avec des contrats explicites (`consumes`/`produces`, garde-fous, sécurité, permissions de fichiers, politique réseau, bornes de boucle), puis `forgent build` les compile vers Claude Code, GitHub Copilot, Cursor ou un runtime Go autonome (`--target forgent`, moteur DAG exécutable dans `pkg/dag`). Il fournit aussi `lint`, `score`, `doctor` (analyse par LLM, 12 contrôles), `classify` (facettes d'instructions), `check` (estimation de conformité par harnais) et `import` (décompose des fichiers `.md` d'agents en specs). Positionnement retenu : compilateur de harnais multi-cibles (souple en Markdown, strict en runtime Go, analyse seule), pas un orchestrateur. Un binaire interne `forgent-bench` porte les expériences (garde-fous, DAG, SWE-bench). Le dossier local s'appelle `ax-cli` : ancien nom du projet.

## État au 2026-11-02

- Dépôt GitHub public `mirandaguillaume/forgent` ; branche locale courante : `chore/clean-history-testdata` (`ee2bf82`, 2026-05-28, « refactor(check): replace fake percentages with qualitative risk levels »). Dernier travail : cible Cursor, estimateur de conformité (`check`), génération de `CLAUDE.md` pour la cible claude, infrastructure d'expérience Epic 10 (SWE-bench, métriques de sécurité), validation typée/AST (`pkg/qualitygate`), sandbox réseau.
- Historique réécrit : les datasets de bench ont été retirés de l'historique local. Résultat, la branche par défaut distante `origin/main` (`4d9b9a9`, 2026-03-24) n'a plus le même historique que le `main` local (`0f21f58`) : 128 commits de chaque côté depuis la base `bdc1542`. Le `main` réécrit est poussé sous `origin/main-local-2026-11-02` ; `chore/clean-history-testdata` (177 commits d'avance sur `origin/main`) est identique à sa copie distante.
- Sauvegardes poussées le 2026-11-02 : `wip/sauvegarde-2026-11-02` (`cd58e70`, un commit au-dessus de la branche courante : 11 fichiers `epic-10-*` / `experiment-sonnet-baseline-converted.json` et le binaire `forgent-bench` reconstruit, 36 Mo vers 39 Mo) ; `feat/scanner-enricher-bench-local-2026-11-02` (`deee12a`, divergente de `origin/feat/scanner-enricher-bench`) ; `origin/stash/2026-03-11-0` (copie du stash). Le stash local `stash@{0}` (WIP sur `ci/auto-release-pipeline`) existe toujours.
- Autres branches locales anciennes : `master`, `feat/go-rewrite`, `feat/ax-cli-mvp`, `ci/auto-release-pipeline`, `release-please--branches--main` (toutes en retard sur `main`).
- Epic 10 (preuve que l'application structurelle bat les garde-fous par mots-clés) : stories 10-0 à 10-11 terminées ; 10-9 donne sur 50 tâches Martian CR-Bench un rappel de 22 % (texte), 34 % (runtime), 41 % (spécialistes), avec 10 % de tokens en moins ; l'enforcement T5 est reporté après le MVP (risque « théorie B »).

## Installer et lancer

Prérequis : Go 1.24.2 (`go.mod` ; le README annonce 1.22+).

```bash
go build ./cmd/forgent          # CLI ; go install github.com/mirandaguillaume/forgent/cmd/forgent@latest
go build ./cmd/forgent-bench    # binaire de bench interne
go test ./...                   # ~2300 tests, race-clean d'après le dernier bilan
go vet ./...
forgent init && forgent skill create search-web --tools web_search,read_url
forgent lint && forgent build [--target claude|copilot|cursor|forgent] [--compact] [--watch]
forgent classify <fichier> --provider anthropic     # LLM par défaut, --quick = statique
forgent import <source> [--yes]
```

Mutation : `.gremlins.yaml` (gremlins). Livraison : `release-please` + GoReleaser (`.goreleaser.yaml`, workflows `ci.yaml` et `release.yaml`). Site de doc Hugo : `hugo.yaml`, `content/`, `layouts/`. Documents de cadrage : `docs/THESIS.md`, `docs/ROADMAP.md`, `docs/architecture.md`, `_bmad-output/planning-artifacts/` (PRD, architecture, épics) et `_bmad-output/implementation-artifacts/` (stories, rétros, `sprint-status.yaml`).

## Où j'en étais et prochaines étapes

1. Story 10-12 (validation SWE-bench, `in-progress`) : le code est écrit et 12 correctifs de revue sont appliqués ; restent T3 (test de fumée du harnais Docker sur 5 tâches, bloqué faute de Docker et du paquet Python `swebench`) et T4 (run de 30 puis 300 tâches SWE-Bench Lite, 3 conditions, intervalle de confiance bootstrap déjà implémenté ; bloqué faute de budget API). Ensuite passer `10-12` et `sprint-status.yaml` à `done`, clore l'epic 10. Voir `_bmad-output/implementation-artifacts/10-12-swebench-validation.md` et `sprint-change-proposal-2026-05-13-swebench.md`.
2. Aligner la notion de « tour » : une itération Forgent peut englober plusieurs appels d'outils, alors que SWE-Bench compte un tour par action-observation ; exposer chaque appel ou publier les deux métriques.
3. Run T5 de l'expérience 10-9 (preuve FR45, comparaison avec la baseline Sonnet du 2026-04-16 sur 5 tâches) reporté ; commande de conversion de la baseline : `forgent-bench convert-baseline experiment-sonnet-baseline.text.json experiment-sonnet-baseline.runtime.json experiment-sonnet-baseline-converted.json`.
4. Trancher le sort de l'historique : soit remplacer `origin/main` par le `main` réécrit (`origin/main-local-2026-11-02`), soit repartir d'`origin/main` en réappliquant le travail ; ensuite supprimer de la branche de sauvegarde le binaire et les fichiers de résultats.
5. `docs/ROADMAP.md`, phases 1 à 3 : contrats typés pour `consumes`/`produces`, champs de documentation de skill, versionnage sémantique, inférence d'effort, garde sur étapes orphelines, `forgent test`, groupes parallèles, DAG v2 (race, fallback, map-reduce, HITL), graphes cycliques, hiérarchie multi-niveaux, `import` par lot. La cible Cursor y figure encore comme future alors qu'elle est livrée.

## À fournir hors dépôt

- Clés LLM : `ANTHROPIC_FORGENT_API_KEY`, `OPENROUTER_FORGENT_API_KEY` (prioritaires), `ANTHROPIC_API_KEY`, `OPENROUTER_API_KEY`, `FORGENT_API_KEY`, `FORGENT_MODEL` ; Ollama local sans clé. Le `.envrc` local (ignoré) définit les deux premières ; ne jamais le committer.
- Datasets de bench non versionnés (22 Mo dans `internal/bench/testdata/` : SWE-bench Lite 30 tâches, Martian, FeatureBench, `eval_tasks.yaml`), plus `logs/`, `results/`, `predictions-*.jsonl`, `bench-h2h.json` : tous ignorés. Ils proviennent de jeux publics tiers (SWE-bench Lite, Martian CR-Bench) à retélécharger ; aucune recette de régénération n'est écrite dans le dépôt. Un fichier SWE-bench contenait un jeton tiers qui avait déclenché la protection GitHub : ne jamais committer ces fichiers.
- Documents locaux ignorés : `docs/plans/` (27 des 32 notes de conception datées de mars 2026 ne sont pas suivies : ast-context, copilot, go-rewrite, solid-skills, container-runtime, h2h-benchmark, whitepaper…). À commiter ou archiver pour ne pas les perdre.
- Services : Docker et paquet Python `swebench` pour le harnais ; budget API pour les runs.
- Fichiers non suivis à la racine (`epic-10-*`, `experiment-sonnet-baseline-converted.json`) : résultats d'expérience, à ne pas confondre avec du code ; les rapports `experiment-*.md` sont, eux, suivis.

## Pièges connus

- Ollama est détecté avant les clés Anthropic : toujours passer `--provider anthropic` explicitement (import ou classify), sinon le modèle local, lent et moins précis, est choisi.
- La classification statique donnait 100 % de faux positifs « Context » sur les serveurs MCP ; l'import s'appuie donc sur `ClassifyLLM` (Haiku) pour ancrer la décomposition (résultat : 7 skills et un agent à 94/100 sur le CLAUDE.md de mcp-servers).
- Anti-patrons relevés à la rétro de l'epic 9 : « swallow & continue » (erreurs avalées) et `rootCmd` partagé entre tests ; l'epic 10 est censé les avoir résorbés.
- Le binaire `forgent-bench` (36 Mo) et `.forgent/ci_reviewer/ci-reviewer` sont suivis par git : éviter de les recommitter à chaque build.
- Données de baseline et jeux de bench : jamais dans git (règle confirmée après l'incident du jeton).
- Mode équipe d'agents non recommandé sur ce projet (consommation de tokens, processus non appliqué).
