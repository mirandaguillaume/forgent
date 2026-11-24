# Changelog

## [0.6.0](https://github.com/mirandaguillaume/forgent/compare/v0.5.0...v0.6.0) (2026-11-24)


### Features

* add --compact flag to reduce structural overhead 117% → 14% ([4548451](https://github.com/mirandaguillaume/forgent/commit/454845130aca8b0f0c2728c87b1db5732c06a984))
* add 10 granular code-review SMB skill specs ([8e821ce](https://github.com/mirandaguillaume/forgent/commit/8e821ce9a188c29aadfc630a49fec065b2ea7b60))
* add AllSkills() helper for flat and staged agents ([08cf8ff](https://github.com/mirandaguillaume/forgent/commit/08cf8ff6969c1e71cb75ed4d2eed63228cb228c5))
* add effort level to all skills and template ([9cb2944](https://github.com/mirandaguillaume/forgent/commit/9cb2944e447bfb0dfeeb883634bd380018fd69da))
* add EffortToModel mapping (light→haiku, medium→sonnet, heavy→opus) ([966351d](https://github.com/mirandaguillaume/forgent/commit/966351d554436f5fc4f36d4f16f6aab250f65c0c))
* add Stage struct and Stages field to AgentComposition ([346e6a6](https://github.com/mirandaguillaume/forgent/commit/346e6a67b9d708121b8c99eb59e3747338c0ae5d))
* add strategy.effort field (light/medium/heavy) to skill model ([08f315d](https://github.com/mirandaguillaume/forgent/commit/08f315d950c5f069fc5a27058bbf3f47fd0ef8f0))
* **build:** register forgent as build target ([3653381](https://github.com/mirandaguillaume/forgent/commit/3653381526c4cd218753b24458791e956b55c52a))
* Claude generator supports staged agent pipelines ([5e92418](https://github.com/mirandaguillaume/forgent/commit/5e92418d00694ac8bc2d8495b1c525c639d76592))
* copilot agent generator with subagent dispatch format ([1ddd02b](https://github.com/mirandaguillaume/forgent/commit/1ddd02bf6e5fd5c5bb23035538dba17445b660bf))
* Copilot generator supports staged agent pipelines ([c0d9f1b](https://github.com/mirandaguillaume/forgent/commit/c0d9f1b26ee4c7314826315e4998ab9e0077ff46))
* **dag:** add core types — Node, DAG, auto-wiring ([43e1170](https://github.com/mirandaguillaume/forgent/commit/43e117077100ce679c0fb9b246b9b20c38206461))
* **dag:** add cycle detection and topology hint validation ([ba70de7](https://github.com/mirandaguillaume/forgent/commit/ba70de785a3836635a392fbd37f590d86abb2cd7))
* **dag:** add layer decomposition — T1-T10 topologies ([d31ed24](https://github.com/mirandaguillaume/forgent/commit/d31ed24a5219a1d7b928933397846c8c8ca7f625))
* **dag:** add layer-parallel execution engine with router support ([d6cb902](https://github.com/mirandaguillaume/forgent/commit/d6cb902874c82c61884263ab79dbf2f549775351))
* **forgent-target:** add --input and --output flags to generated runtime ([21f17dc](https://github.com/mirandaguillaume/forgent/commit/21f17dcc252df173338125058fe0d995bb8203d8))
* **forgent-target:** add Go code generator for agent runtime ([438ff17](https://github.com/mirandaguillaume/forgent/commit/438ff177b33715ec9eb011a149d9f9779fd28e21))
* **forgent-target:** add OpenRouter support — OPENROUTER_API_KEY with model selection ([3d90766](https://github.com/mirandaguillaume/forgent/commit/3d90766f337d2cf233bd1e8d71ab609d0041fa4b))
* **forgent-target:** add prompt template builder ([013851b](https://github.com/mirandaguillaume/forgent/commit/013851b799efba02a302e3e8f30aa2dd9bdfad6a))
* **forgent-target:** register forgent as build target ([897605d](https://github.com/mirandaguillaume/forgent/commit/897605d6e81512678f1623a529c3ff17e2cf3874))
* generate orchestrator agents with Task() dispatch and per-skill model ([9fb4a33](https://github.com/mirandaguillaume/forgent/commit/9fb4a33dece47b5fc25c226e752596c17e5506dc))
* update docs, generators, DAG engine, bench framework and importers ([6e0376e](https://github.com/mirandaguillaume/forgent/commit/6e0376eb9cb6a1837c17e8f62fbec95af58b8504))
* validate staged agents with mutual exclusivity and stage rules ([cebc829](https://github.com/mirandaguillaume/forgent/commit/cebc82978dd651c06c3d474f0cfc000c0aad00bd))


### Bug Fixes

* **dag:** add arg validation, duplicate producer check, cycle guard, copy returns ([f80cca1](https://github.com/mirandaguillaume/forgent/commit/f80cca16c2efb283ff91876804601995ef647b66))
* **forgent-target:** add test coverage for header comment and module sanitization ([e3eaf40](https://github.com/mirandaguillaume/forgent/commit/e3eaf40ea08b9f44714c67cc5b6387a4addc8d1a))
* **forgent-target:** fix sanitisation inconsistency and ContextDir ([dfb3b08](https://github.com/mirandaguillaume/forgent/commit/dfb3b083e2a9f8f978a9d866405b54869cc6a1ef))
* **forgent-target:** forward ctx to HTTP request, check go mod tidy error ([e28cf4d](https://github.com/mirandaguillaume/forgent/commit/e28cf4defb320ec63c5859d3ca621b62b84e942b))
* **forgent-target:** guard empty consumes/produces, use FormatGuardrail ([a330d26](https://github.com/mirandaguillaume/forgent/commit/a330d26223497bfc81efbfa98ddda8acf57e3c25))
* **forgent-target:** use agent name in output, format source, sanitize module name ([6cf755f](https://github.com/mirandaguillaume/forgent/commit/6cf755f9b5cc71fcd8ff0cd6cf8ffba3249e7b19))

## [0.5.0](https://github.com/mirandaguillaume/forgent/compare/v0.4.0...v0.5.0) (2026-03-15)


### Features

* whitepaper v2 — formal properties, frontiers, and experience ([#15](https://github.com/mirandaguillaume/forgent/issues/15)) ([ea3f72f](https://github.com/mirandaguillaume/forgent/commit/ea3f72fedff41fcffb221774c6e50bd69a7af563))

## [0.4.0](https://github.com/mirandaguillaume/forgent/compare/v0.3.0...v0.4.0) (2026-03-13)


### Features

* forgent import — LLM-powered agent decomposition pipeline ([#13](https://github.com/mirandaguillaume/forgent/issues/13)) ([5a216e0](https://github.com/mirandaguillaume/forgent/commit/5a216e0ab7fb9b694970db172e9a902ea74238ef))

## [0.3.0](https://github.com/mirandaguillaume/forgent/compare/v0.2.0...v0.3.0) (2026-03-12)


### Features

* codebase scanner, enricher, bench + whitepaper rewrite ([9d65407](https://github.com/mirandaguillaume/forgent/commit/9d65407a4eebad6cb7ecc32e82effa94dc7b7535))

## [0.2.0](https://github.com/mirandaguillaume/forgent/compare/v0.1.0...v0.2.0) (2026-03-12)


### Features

* add codebase scanner, enricher and bench command ([70ac475](https://github.com/mirandaguillaume/forgent/commit/70ac47522db6234c94bea08c33a445104d87c191))
* **build:** auto-enrich skills that consume codebase_index ([25b1182](https://github.com/mirandaguillaume/forgent/commit/25b1182270ca44c9d2d7b85197965c6f59d4177e))
* **linter:** add producesMatchesDescription SRP lint rule ([eb4c655](https://github.com/mirandaguillaume/forgent/commit/eb4c655bbcd0a420651a7c8423378b1e14007def))
* **linter:** add singleProducesOutput SRP lint rule ([5ef5589](https://github.com/mirandaguillaume/forgent/commit/5ef55899a8c03e49461a0b74a7d7133228367b5c))
* **linter:** add skillNameMatchesOutput SRP lint rule ([34c56d6](https://github.com/mirandaguillaume/forgent/commit/34c56d6fb6ab396c4262335a43f3c1192c34455d))
* **model:** add when_to_use, anti_patterns and examples facets ([a82a5c5](https://github.com/mirandaguillaume/forgent/commit/a82a5c521055920b4d9becd7eac9911404aa7922))
* SOLID skills + codebase scanner/enricher/bench ([7068630](https://github.com/mirandaguillaume/forgent/commit/70686303f535571d44d6ebbe22ed4d913c7e4d0f))


### Bug Fixes

* address code review issues ([135dbce](https://github.com/mirandaguillaume/forgent/commit/135dbce2b4fdf2fba6b5af2c3aed2910e78073fb))
* **linter:** correct skillNameMatchesOutput facet to context ([9fb9abe](https://github.com/mirandaguillaume/forgent/commit/9fb9abef61ba7e68ea231360340f5bd3ba4177e5))

## [0.1.0](https://github.com/mirandaguillaume/forgent/compare/v0.0.0...v0.1.0) (2026-03-11)


### Features

* add analyzers (dependency, loop, trace, score, ordering) ([670d251](https://github.com/mirandaguillaume/forgent/commit/670d251e2970162e8a0c4cdda4231769a0189c2a))
* add build command with multi-target support ([628f02f](https://github.com/mirandaguillaume/forgent/commit/628f02f4c608e386e7602a37b4dbf42b26a244a3))
* add build, watch, score commands and skill ordering ([e575461](https://github.com/mirandaguillaume/forgent/commit/e575461ac0d9c5c8fac08689877ccb7eb26cc9f4))
* add Claude generators (skill, agent, toolmap) ([ffbfdb0](https://github.com/mirandaguillaume/forgent/commit/ffbfdb0ee6c84f7c3d661b14c17e11ea66f19a54))
* add Copilot generators (skill, agent, instructions, tool map) ([d0c457e](https://github.com/mirandaguillaume/forgent/commit/d0c457e79875f2af04fdc93d7e5b53cd42f75425))
* add Copilot generators (skill, agent, instructions, toolmap) ([22fc788](https://github.com/mirandaguillaume/forgent/commit/22fc7881763096d872e439d55f651e78b2be28b8))
* add doctor, trace, and score commands ([64a4323](https://github.com/mirandaguillaume/forgent/commit/64a43232e67fad51d3bd12d49ea9f1ea36c1df87))
* add Hugo site with Hextra theme and landing page ([baf487b](https://github.com/mirandaguillaume/forgent/commit/baf487bf8be06bdc90d26f31be2c07af92cdd31d))
* add init command with embedded templates ([370b4ce](https://github.com/mirandaguillaume/forgent/commit/370b4ce1751b27c56bc45f02ce9a6b14f51ad2b0))
* add linter rules and lint command ([9c141af](https://github.com/mirandaguillaume/forgent/commit/9c141af1d1503398b58089d4945e6552871afe3e))
* add skill create command ([2af370d](https://github.com/mirandaguillaume/forgent/commit/2af370d5e1c7dd5d99966be6592dd5d9e61e8697))
* add SkillBehavior, AgentComposition models with validation ([e883120](https://github.com/mirandaguillaume/forgent/commit/e883120c4c0604527c409c515fcd58e853db0ef4))
* add TargetGenerator interface and registry ([b8adb57](https://github.com/mirandaguillaume/forgent/commit/b8adb57e6d3db7c97600e8bb5710831bbf567a4a))
* add TargetGenerator interface and registry ([f505e05](https://github.com/mirandaguillaume/forgent/commit/f505e059e9939d14cf6dc37931168e2b93cadb32))
* add watch command and GoReleaser config ([d84cfa2](https://github.com/mirandaguillaume/forgent/commit/d84cfa271d52529507b1fc07f9b3fc5b6f6619d6))
* add YAML loader with validation ([17ad47a](https://github.com/mirandaguillaume/forgent/commit/17ad47a4a1f61a52edfdf9e440d5585420683afe))
* **ax:** add --target flag to build command, defaults to claude → .claude/ ([63bee27](https://github.com/mirandaguillaume/forgent/commit/63bee271c6fed0b349331c46dc2ed652760094ce))
* **ax:** add 'ax build' command generating Claude Code skills and agents ([b806629](https://github.com/mirandaguillaume/forgent/commit/b8066298b4bbaf2eccc7424ba6de9b5aeb9e81ca))
* **ax:** add 'ax doctor' command with aggregated diagnostics ([3e9ab6c](https://github.com/mirandaguillaume/forgent/commit/3e9ab6c69640946bac1361eaff3839143f866314))
* **ax:** add 'ax init' command with project scaffolding ([38f3a5f](https://github.com/mirandaguillaume/forgent/commit/38f3a5fff00aaba51e7760dcd81b4675a55786ff))
* **ax:** add 'ax lint' command for skill quality checks ([4e6b01a](https://github.com/mirandaguillaume/forgent/commit/4e6b01aa86ad7f122214a4a58b27ba04355afb12))
* **ax:** add 'ax skill create' command ([0f36fe4](https://github.com/mirandaguillaume/forgent/commit/0f36fe41ee23b131b9e13234ba5845e0d8779210))
* **ax:** add 'ax trace' command with JSONL trace analysis ([14732e5](https://github.com/mirandaguillaume/forgent/commit/14732e5e105167ca9f65dfb7442ad0fb9cea00c9))
* **ax:** add agent generator for Claude Code agent.md output ([a6e60d0](https://github.com/mirandaguillaume/forgent/commit/a6e60d0de3f365cf601ab344f05cb51f06a20e1f))
* **ax:** add AX linter rules engine with 4 rules ([400c57e](https://github.com/mirandaguillaume/forgent/commit/400c57edb0187138fa73533717fc0f1d0bd9ba66))
* **ax:** add dependency analyzer with cycle and context validation ([b3e4dee](https://github.com/mirandaguillaume/forgent/commit/b3e4deebe596f08a6151ca0ca205dec3906e1917))
* **ax:** add JSON Schema validation for skills and agents ([dc9ad95](https://github.com/mirandaguillaume/forgent/commit/dc9ad9541a7782029e6a2a931ad9ed93183cfb30))
* **ax:** add loop risk detector for self-referencing skills ([366bee1](https://github.com/mirandaguillaume/forgent/commit/366bee181785fcdd929033e2be022fb4f93d451b))
* **ax:** add skill generator for Claude Code SKILL.md output ([68be5a5](https://github.com/mirandaguillaume/forgent/commit/68be5a59b77a01f1ab3e2deddc8b7415803daf1e))
* **ax:** add YAML parsing with schema validation ([2493039](https://github.com/mirandaguillaume/forgent/commit/24930395e7eaafc89e3e96871dc165888a449897))
* **ax:** define Skill Behavior Model types with 6 facets ([fe79807](https://github.com/mirandaguillaume/forgent/commit/fe79807466fefab2b5cdab8dd8ab1dff34298e9a))
* **ax:** scaffold ax-cli project with TypeScript + Commander ([be1d26d](https://github.com/mirandaguillaume/forgent/commit/be1d26d602e7498b4add4ecda7a94bb57a4b8e73))
* initialize Go module with Cobra skeleton, remove TypeScript ([37f6317](https://github.com/mirandaguillaume/forgent/commit/37f631790c34bf39636c96c3ece65012f0c87ada))
* **site:** redesign landing page with forge-inspired theme ([143e1a0](https://github.com/mirandaguillaume/forgent/commit/143e1a0f1de071105331591f582e314c5c405c04))
* wrap claude generators and refactor build.ts to use TargetGenerator ([de8fafb](https://github.com/mirandaguillaume/forgent/commit/de8fafb2c9e80e424a3bdf5687edc587eb95946a))


### Bug Fixes

* address code review findings from Go rewrite ([ddc2068](https://github.com/mirandaguillaume/forgent/commit/ddc206822a6b2842ee6a41090d44e69d0bd5bbe4))
* **ci:** configure release-please for v0.x and fix goreleaser deprecations ([419ee1b](https://github.com/mirandaguillaume/forgent/commit/419ee1b0d113e6ea9a3522a86f724de46f8451d7))
* **ci:** disable bump-patch-for-minor-pre-major to get v0.1.0 ([129f6d0](https://github.com/mirandaguillaume/forgent/commit/129f6d0d5e1978a3a3573c136c231dc140b3e7b0))
* **ci:** reset changelog for v0.1.0 initial release ([1d503e2](https://github.com/mirandaguillaume/forgent/commit/1d503e2b18f323b64da7d1d71de71e4ed0157a04))
* **ci:** set initial version to 0.0.0 for first release as v0.1.0 ([33f4a88](https://github.com/mirandaguillaume/forgent/commit/33f4a880c496752b0d6fec81ca1778001f836d39))
* resolve circular import in target generator registration ([e3eda79](https://github.com/mirandaguillaume/forgent/commit/e3eda79e9b6d06469d1b0ea1c4db6ffdcf7e4d70))
* **site:** custom layout + complete design overhaul for landing page ([278c225](https://github.com/mirandaguillaume/forgent/commit/278c2259b73b669608d77acf41af445264dd5266))
* **site:** force dark background on all landing page sections ([f82d474](https://github.com/mirandaguillaume/forgent/commit/f82d4741c77cecc4147f4ece68868494f5f277da))

## Changelog
