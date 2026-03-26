# Changelog

## [0.6.0](https://github.com/mirandaguillaume/forgent/compare/v0.5.0...v0.6.0) (2026-03-24)


### Features

* add --compact flag to reduce structural overhead 117% → 14% ([3f2d965](https://github.com/mirandaguillaume/forgent/commit/3f2d965139afa9017774239e3f43ad9f01694a52))
* add 10 granular code-review SMB skill specs ([dca3755](https://github.com/mirandaguillaume/forgent/commit/dca3755e4f146cd297ad7333ce3bd77b9ed82b6e))
* add AllSkills() helper for flat and staged agents ([787262e](https://github.com/mirandaguillaume/forgent/commit/787262e73938c0a937f912be1d0d5ddc11398c50))
* add effort level to all skills and template ([24c7ee0](https://github.com/mirandaguillaume/forgent/commit/24c7ee0f7dc816b3e568656f3071d0308a46977b))
* add EffortToModel mapping (light→haiku, medium→sonnet, heavy→opus) ([4524947](https://github.com/mirandaguillaume/forgent/commit/452494719cc0267593041f64623617c91c8f7750))
* add Stage struct and Stages field to AgentComposition ([483354f](https://github.com/mirandaguillaume/forgent/commit/483354fde790c44d5bb9b47955c0dbfea5e7d4cc))
* add strategy.effort field (light/medium/heavy) to skill model ([af040d9](https://github.com/mirandaguillaume/forgent/commit/af040d945262a0f06a9901c25c3a4aa7087d0235))
* **build:** register forgent as build target ([289e77b](https://github.com/mirandaguillaume/forgent/commit/289e77b97bdc0be6f2b9cf8764ace9265a88f731))
* Claude generator supports staged agent pipelines ([f9f5ddf](https://github.com/mirandaguillaume/forgent/commit/f9f5ddfe144c77173507a5a1029afc191de8c4db))
* copilot agent generator with subagent dispatch format ([3e0107d](https://github.com/mirandaguillaume/forgent/commit/3e0107de6e067a2a772b11eb4edda307f8c199e8))
* Copilot generator supports staged agent pipelines ([8b3cf41](https://github.com/mirandaguillaume/forgent/commit/8b3cf411ced4ce1372be23c1fbba0f40e4513cc4))
* **dag:** add core types — Node, DAG, auto-wiring ([8c3fa0e](https://github.com/mirandaguillaume/forgent/commit/8c3fa0e0d0e398994f3681377fa19298ceea078d))
* **dag:** add cycle detection and topology hint validation ([97a3395](https://github.com/mirandaguillaume/forgent/commit/97a3395a2cfafb373d78589bffac9dd2b9a61b3b))
* **dag:** add layer decomposition — T1-T10 topologies ([8479e36](https://github.com/mirandaguillaume/forgent/commit/8479e36c8de1e5ba531591ff315868a946cf03ba))
* **dag:** add layer-parallel execution engine with router support ([8ba8287](https://github.com/mirandaguillaume/forgent/commit/8ba8287e6bd42d39a95aa3b6894a38667e938d05))
* **forgent-target:** add --input and --output flags to generated runtime ([f45e66f](https://github.com/mirandaguillaume/forgent/commit/f45e66f84c7beb29ae4d470e770ec64435684eb5))
* **forgent-target:** add Go code generator for agent runtime ([6154ebb](https://github.com/mirandaguillaume/forgent/commit/6154ebb9bb59bd77af1ce7e6bd58d0ba52c1d96f))
* **forgent-target:** add OpenRouter support — OPENROUTER_API_KEY with model selection ([2173d94](https://github.com/mirandaguillaume/forgent/commit/2173d941ff45806d83771374b3f65cb5d0c3a216))
* **forgent-target:** add prompt template builder ([372332f](https://github.com/mirandaguillaume/forgent/commit/372332f0a3319c1a440b86203da11a6becdc4b4b))
* **forgent-target:** register forgent as build target ([b7c0859](https://github.com/mirandaguillaume/forgent/commit/b7c08593e942d6aa9b11cca9faa66b6ba61ff56c))
* generate orchestrator agents with Task() dispatch and per-skill model ([d7955d7](https://github.com/mirandaguillaume/forgent/commit/d7955d7d453ac70526c437904fe2c45625f719b9))
* update docs, generators, DAG engine, bench framework and importers ([7fb396b](https://github.com/mirandaguillaume/forgent/commit/7fb396be2c217fe4bf7d7bf450fc0a33752e2d1d))
* validate staged agents with mutual exclusivity and stage rules ([a9d7f70](https://github.com/mirandaguillaume/forgent/commit/a9d7f70e227ff45751a3b1994119ba7280720934))


### Bug Fixes

* **dag:** add arg validation, duplicate producer check, cycle guard, copy returns ([998ab18](https://github.com/mirandaguillaume/forgent/commit/998ab18728733353b22a64638b2a696fd405895b))
* **forgent-target:** add test coverage for header comment and module sanitization ([26de0d5](https://github.com/mirandaguillaume/forgent/commit/26de0d5b0f778968721050a0738b234f843088ae))
* **forgent-target:** fix sanitisation inconsistency and ContextDir ([f0b0167](https://github.com/mirandaguillaume/forgent/commit/f0b016773a52b88b7d58bda1de9a008d459a84cc))
* **forgent-target:** forward ctx to HTTP request, check go mod tidy error ([630b587](https://github.com/mirandaguillaume/forgent/commit/630b587b997e535f3345acb02bfd4feff49337e5))
* **forgent-target:** guard empty consumes/produces, use FormatGuardrail ([10fca22](https://github.com/mirandaguillaume/forgent/commit/10fca229c42f09d2f479155f1c91a8366cacc6e0))
* **forgent-target:** use agent name in output, format source, sanitize module name ([f30cde0](https://github.com/mirandaguillaume/forgent/commit/f30cde03e299b00cf328add692ce42f856830bcc))

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
