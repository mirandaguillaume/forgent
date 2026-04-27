# Story 6.3: Shell Completion and CLI Polish

Status: ready-for-dev

## Story

As a **developer**,
I want shell completion and polished help text,
so that the CLI feels professional and discoverable.

## Acceptance Criteria

1. `forgent completion bash` (and zsh, fish, powershell) outputs a shell completion script (FR58)
2. Completions include: command names, flag names, file paths for doctor, target names for build
3. `forgent doctor --help` includes at least one usage example showing a complete workflow
4. All flags are documented with their defaults
5. All error messages include a suggested fix (NFR28)
6. Exit codes are consistent: 0 (success), 1 (error), 2 (recommendations found) (NFR30)
7. All existing tests pass (`go test ./... -count=1 -short`)

## Tasks / Subtasks

- [ ] Task 1: Shell completion (AC: #1, #2)
  - [ ] 1.1: Create `internal/cmd/completion.go` — `completionCmd` using Cobra's built-in completion generators
  - [ ] 1.2: `forgent completion bash` → `rootCmd.GenBashCompletion(os.Stdout)`
  - [ ] 1.3: `forgent completion zsh` → `rootCmd.GenZshCompletion(os.Stdout)`
  - [ ] 1.4: `forgent completion fish` → `rootCmd.GenFishCompletion(os.Stdout, true)`
  - [ ] 1.5: `forgent completion powershell` → `rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)`
  - [ ] 1.6: Add custom completions: `--provider` suggests [ollama, openrouter, anthropic], `--target` suggests registered targets
  - [ ] 1.7: Doctor file argument: complete with file paths (default Cobra behavior)
  - [ ] 1.8: Write tests verifying each shell output is non-empty and contains expected keywords

- [ ] Task 2: Help text polish (AC: #3, #4)
  - [ ] 2.1: Update `doctorCmd.Long` with full usage examples:
    ```
    forgent doctor .claude/agents/reviewer.md
    forgent doctor .github/agents/ --provider ollama
    forgent doctor agent.md --fix --force
    forgent doctor --update-registry
    ```
  - [ ] 2.2: Review all commands' `Short` and `Long` descriptions for clarity
  - [ ] 2.3: Ensure all flags have description text and default values shown
  - [ ] 2.4: Add `--help` examples to: build, lint, score, init, skill create, import, targets
  - [ ] 2.5: Write test that `--help` output contains "Example" for doctor and build commands

- [ ] Task 3: Error message audit (AC: #5)
  - [ ] 3.1: Audit all `fmt.Errorf` and error returns in `internal/cmd/` for actionable messages
  - [ ] 3.2: Pattern: "Error: <what happened>. <suggestion>." e.g., "Error: cannot read file. Check path and permissions."
  - [ ] 3.3: Audit provider selection errors — already good from Epic 1, verify no regressions
  - [ ] 3.4: Audit parser detection errors — add suggestion for unsupported formats
  - [ ] 3.5: Write tests for key error paths verifying suggestion text is present

- [ ] Task 4: Exit code consistency (AC: #6)
  - [ ] 4.1: Audit all `os.Exit()` calls across all commands
  - [ ] 4.2: Ensure: 0 = success/no findings, 1 = error, 2 = recommendations found (doctor only)
  - [ ] 4.3: Other commands (build, lint, score): 0 = success, 1 = error
  - [ ] 4.4: Document exit codes in `--help` for doctor command
  - [ ] 4.5: Write tests verifying exit codes for each scenario

## Dev Notes

### Architecture References

- **FR58:** "Developer can generate shell completion scripts for Bash, Zsh, Fish, and PowerShell."
- **NFR28:** "Actionable error messages with suggestions."
- **NFR29:** "--help includes usage examples."
- **NFR30:** "Consistent exit codes: 0/1/2."
- **Cobra built-in:** Cobra has built-in shell completion generation — use it directly.

### Cobra Completion

Cobra provides completion generators out of the box. Custom completions for specific flags use `RegisterFlagCompletionFunc`:

```go
doctorCmd.RegisterFlagCompletionFunc("provider", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    return []string{"ollama", "openrouter", "anthropic"}, cobra.ShellCompDirectiveNoFileComp
})
```

### Dependencies

- **All previous stories:** This is the final polish story. All commands should be functional.

### Previous Story Intelligence (from Epic 1 retro)

- **Action #4:** "Always review before done."
- **NFR28 already partially addressed:** Epic 1 Ollama adapter has actionable error messages. This story audits the rest.

### What NOT to Build

- Do NOT implement man page generation — shell completion is sufficient
- Do NOT implement auto-update for completion scripts — manual install instructions
- Do NOT implement interactive tutorial — help text examples are sufficient
- Do NOT implement i18n — English only

### Project Structure

New files:
- `internal/cmd/completion.go` — completionCmd with bash/zsh/fish/powershell

Modified files:
- `internal/cmd/doctor.go` — updated help text, flag descriptions
- `internal/cmd/build.go` — updated help text, examples
- `internal/cmd/lint.go` — updated help text
- `internal/cmd/score.go` — updated help text
- `internal/cmd/root.go` — ensure consistent error handling

## Testing Strategy

- **Completion output tests:** Each shell (bash, zsh, fish, powershell) produces non-empty output containing expected command names.
- **Custom completion tests:** `--provider` completes with ollama/openrouter/anthropic.
- **Help text tests:** `--help` contains "Example" section for doctor and build commands.
- **Error message tests:** Key error paths produce messages containing suggestions.
- **Exit code tests:** Use `exec.Command` to run forgent subprocesses and verify exit codes.
- **CI gate:** `go test ./... -count=1 -race`

## References

- [Source: _bmad-output/planning-artifacts/prd.md#FR58] — Shell completion
- [Source: _bmad-output/planning-artifacts/prd.md#NFR28] — Actionable error messages
- [Source: _bmad-output/planning-artifacts/prd.md#NFR29] — Help with examples
- [Source: _bmad-output/planning-artifacts/prd.md#NFR30] — Consistent exit codes
- [Source: github.com/spf13/cobra] — Built-in completion generation

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
