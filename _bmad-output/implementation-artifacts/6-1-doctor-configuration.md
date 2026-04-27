# Story 6.1: Doctor Configuration

Status: ready-for-dev

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

- [ ] Task 1: Config file support (AC: #1)
  - [ ] 1.1: Create `internal/config/config.go` — `Config` struct, `Load(dir string) (*Config, error)`
  - [ ] 1.2: `Config` struct fields: Provider, Model, ConfidenceThreshold, BackupRetention (duration), Debug, RegistryPath
  - [ ] 1.3: Load reads `.forgent/config.yaml` from current dir, then parent dirs (up to home or filesystem root)
  - [ ] 1.4: If no config file found, return zero-value Config (all defaults)
  - [ ] 1.5: Write tests for: config present, config absent, malformed config (actionable error), nested directory lookup

- [ ] Task 2: Environment variable override (AC: #2)
  - [ ] 2.1: Add `ApplyEnv(cfg *Config)` — reads `FORGENT_PROVIDER`, `FORGENT_MODEL`, `FORGENT_DEBUG` and overrides Config fields
  - [ ] 2.2: Only override non-empty env vars (empty string = not set)
  - [ ] 2.3: Write tests with `t.Setenv()` for each env var

- [ ] Task 3: CLI flag override and resolution (AC: #3, #4)
  - [ ] 3.1: Add `ApplyFlags(cfg *Config, provider, model string, debug bool)` — override Config with non-default flag values
  - [ ] 3.2: Resolution order: `Load()` → `ApplyEnv()` → `ApplyFlags()` → final Config
  - [ ] 3.3: Wire into `internal/cmd/doctor.go` — replace hardcoded defaults with config resolution
  - [ ] 3.4: In debug mode, log resolved config source for each field: "provider: ollama (from: config file)"
  - [ ] 3.5: Write integration test verifying full precedence chain

- [ ] Task 4: Config file format and documentation (AC: #1)
  - [ ] 4.1: Define config YAML schema:
    ```yaml
    doctor:
      provider: ollama
      model: llama4-scout
      confidence_threshold: moderate
      backup_retention: 168h  # 7 days
      debug: false
    ```
  - [ ] 4.2: Add `forgent doctor --help` text mentioning config file location and env vars
  - [ ] 4.3: Create a sample config file template shown in help output

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

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
