# Story 6.1: Doctor Configuration

Status: done

## Story

As a **developer**,
I want to configure doctor defaults so I don't have to pass flags every time,
so that my preferred provider, model, and thresholds are remembered.

## Acceptance Criteria

1. Config file at `.forgent/config.yaml` with doctor settings: provider, model, confidence_threshold, backup_retention, debug
2. Environment variables override config file: `FORGENT_PROVIDER`, `FORGENT_MODEL`, `FORGENT_DEBUG`
3. CLI flags override both env vars and config file (ADR-7 precedence: flags > env > config > defaults)
4. Doctor uses resolved config as defaults, overridable per-run
5. All existing tests pass (`go test ./... -count=1 -short`)

## Tasks / Subtasks

- [x] Task 1: Config file support (AC: #1)
  - [x] 1.1: Create `internal/config/config.go` — `Config` struct, `Load(dir string) (*Config, error)`
  - [x] 1.2: `Config` struct fields: Provider, Model, ConfidenceThreshold, BackupRetention (duration), Debug, RegistryPath
  - [x] 1.3: Load reads `.forgent/config.yaml` from current dir, then parent dirs (up to home or filesystem root)
  - [x] 1.4: If no config file found, return zero-value Config (all defaults)
  - [x] 1.5: Write tests for: config present, config absent, malformed config (actionable error), nested directory lookup

- [x] Task 2: Environment variable override (AC: #2)
  - [x] 2.1: Add `ApplyEnv(cfg *Config)` — reads `FORGENT_PROVIDER`, `FORGENT_MODEL`, `FORGENT_DEBUG` and overrides Config fields
  - [x] 2.2: Only override non-empty env vars (empty string = not set)
  - [x] 2.3: Write tests with `t.Setenv()` for each env var

- [x] Task 3: CLI flag override and resolution (AC: #3, #4)
  - [x] 3.1: Add `ApplyFlags(cfg *Config, provider, model string, debug bool)` — override Config with non-default flag values
  - [x] 3.2: Resolution order: `Load()` → `ApplyEnv()` → `ApplyFlags()` → final Config
  - [x] 3.3: Wire into `internal/cmd/doctor.go` — replace hardcoded defaults with config resolution
  - [x] 3.4: In debug mode, log resolved config source for each field: "provider: ollama (from: config file)"
  - [x] 3.5: Write integration test verifying full precedence chain

- [x] Task 4: Config file format and documentation (AC: #1)
  - [x] 4.1: Define config YAML schema (under `doctor:` key with all fields)
  - [x] 4.2: Add `forgent doctor --help` text mentioning config file location and env vars
  - [x] 4.3: Sample config template shown in help output

## Dev Notes

### Architecture References

- **ADR-7 (Configuration):** "CLI flags > environment variables > config file (.forgent/config.yaml) > defaults. Standard pattern."
- **FR57:** "Developer can configure doctor defaults via config file or environment variables."
- **NFR14:** "Forgent upgrades preserve all user configuration."

### Config File Location

Config lives at `.forgent/config.yaml` in the project root (same directory as `.forgent/research-registry.yaml`). This keeps all Forgent-specific config in one place. The `.forgent/` directory is created by `forgent init` and also used for local registry override.

### No Viper

The architecture says "Viper or manual resolution for env/config." Given the small number of config fields (< 10), manual resolution is simpler and avoids a heavy dependency. Use `os.Getenv()` and `yaml.v3` directly.

### Dependencies

- **Epic 1-2:** Doctor command exists with flag handling. This story adds config resolution underneath.

### Previous Story Intelligence (from Epic 1 retro)

- **Action #4:** "Always review before done."
- **Action #5:** "No undocumented scope creep" — config fields are defined here. Do not add more without discussion.

### What NOT to Build

- Do NOT implement `forgent config` command — direct file editing is sufficient for MVP
- Do NOT implement Viper — manual resolution is simpler for < 10 fields
- Do NOT implement config validation beyond YAML parsing — invalid values fall through to existing flag validation
- Do NOT implement config migration — first version, no backwards compatibility needed yet

### Project Structure

New files:
- `internal/config/config.go` — Config struct, Load, ApplyEnv, ApplyFlags
- `internal/config/config_test.go`

Modified files:
- `internal/cmd/doctor.go` — wire config resolution into provider/model selection

## Testing Strategy

- **Config loading tests:** File present (parsed correctly), file absent (defaults), malformed YAML (actionable error), nested directory lookup.
- **Env override tests:** Each env var overrides corresponding field. Empty env var = no override. Use `t.Setenv()`.
- **Flag override tests:** Flags override env and config. Non-default flags only.
- **Precedence integration test:** Config file sets provider=ollama, env sets provider=anthropic, flag sets provider=openrouter → final = openrouter.
- **Debug output test:** Verify config source logging in debug mode.
- **CI gate:** `go test ./... -count=1 -race`

## References

- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-7] — Configuration resolution precedence
- [Source: _bmad-output/planning-artifacts/prd.md#FR57] — Doctor config via file or env vars
- [Source: _bmad-output/planning-artifacts/prd.md#NFR14] — Upgrades preserve user config

## Dev Agent Record

### Agent Model Used

claude-sonnet-4-6

### Debug Log References

_empty_

### Completion Notes List

Implemented `internal/config` package with `Config` struct (under `doctor:` YAML key), `Load` (walks from `dir` up to filesystem root looking for `.forgent/config.yaml`), `ApplyEnv` (FORGENT_PROVIDER, FORGENT_MODEL, FORGENT_DEBUG), and `ApplyFlags` (non-zero values only). Wired into `internal/cmd/doctor.go` via a config resolution block at the start of `RunE` that captures pre-resolution flag values for source tracking, applies the three layers, and logs the resolved source per field in debug mode. Help text updated with config file location, env vars, and a sample YAML template. Key decision: debug source logging checks the pre-resolution original flag value and env keys directly to determine which layer provided each resolved value. 2039 tests passing, 11 new.

### File List

- `internal/config/config.go` — new: Config, DoctorConfig, Load, ApplyEnv, ApplyFlags
- `internal/config/config_test.go` — new: 11 tests covering all loading scenarios, env overrides, and precedence chain
- `internal/cmd/doctor.go` — modified: fconfig import, config resolution block in RunE, updated Long help text

### Review Findings

- [x] [Review][Decision] D1: Walk past $HOME boundary — resolved: stop at `os.UserHomeDir()` before filesystem root. [internal/config/config.go]
- [x] [Review][Patch] P1: Debug escalation-only — fixed: `ApplyEnv` handles "false"/"0"; `ApplyFlags` uses `debugChanged bool` + `cmd.Flags().Changed("debug")` in doctor.go [internal/config/config.go, internal/cmd/doctor.go]
- [x] [Review][Patch] P2: `cfgErr` silently swallowed — fixed: always print to stderr when `cfgErr != nil`. [internal/cmd/doctor.go]
- [x] [Review][Patch] P3: Permission errors silently treated as "not found" — fixed: `os.IsNotExist(err)` check propagates non-ENOENT errors. [internal/config/config.go]
- [x] [Review][Patch] P4: `logResolved` only covers `provider`/`model` — fixed: added `debug`, `confidence_threshold`, `backup_retention`, `registry_path` logging. [internal/cmd/doctor.go]
- [x] [Review][Patch] P5: Missing tests — fixed: added 5 tests for FORGENT_DEBUG=1/false/0 and debug flag precedence. [internal/config/config_test.go]

## Change Log

- 2026-04-27: Story 9-1 (doctor configuration) implemented. 11 new tests, 2039 total passing.
- 2026-04-27: Code review complete. 1 decision (D1→patch), 5 patches, 0 deferred, 9 dismissed. All 6 applied. 16 config tests, 2041 total passing.
