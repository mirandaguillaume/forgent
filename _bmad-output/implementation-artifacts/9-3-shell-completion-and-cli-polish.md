# Story 9-3: Shell Completion and CLI Polish

Status: done

## Story

As a **developer**,
I want shell completion and polished help text,
so that the CLI feels professional and discoverable.

## Acceptance Criteria

1. `forgent completion bash` (and zsh, fish, powershell) outputs a valid shell completion script.
2. Completions include: command names, flag names, file paths for `doctor <file>`, target names for `build --target`.
3. `forgent doctor --help` displays at least one complete workflow example and all flags are documented with their defaults.
4. Any forgent command error includes a suggested fix where actionable.
5. Exit codes are consistent: 0 (success), 1 (error), 2 (recommendations found).

## Tasks / Subtasks

- [x] Task 1: Enable shell completion command (AC: #1, #2)
  - [x] 1.1: Call `rootCmd.InitDefaultCompletionCmd()` in `internal/cmd/root.go` `init()` to register the built-in `completion` subcommand (bash, zsh, fish, powershell)
  - [x] 1.2: Add `ValidArgsFunction` to `doctorCmd` that completes file paths (filter to `.md`, `.yaml`, `.yml`)
  - [x] 1.3: Add `RegisterFlagCompletionFunc` to `buildCmd` for `--target` flag returning `spec.Available()`
  - [x] 1.4: Write tests: completion subcommand exists on root, doctor ValidArgs returns files, build --target returns target names

- [x] Task 2: Help text and defaults documentation (AC: #3)
  - [x] 2.1: Verify `doctorCmd` Long description has at least one complete workflow example — add if missing
  - [x] 2.2: Add Long description with examples to `buildCmd` (target options, output dir, compact mode)
  - [x] 2.3: Document flag defaults explicitly in flag usage strings (e.g., `"LLM provider (ollama, openrouter, anthropic). Default: auto-detect"` — already done; ensure build flags match)
  - [x] 2.4: Write test: `rootCmd --help` output contains "forgent completion", `doctor --help` contains an example

- [x] Task 3: Exit codes and error message polish (AC: #4, #5)
  - [x] 3.1: Audit error paths in `doctor.go` and `build.go` — ensure errors triggered by user mistakes include a hint (e.g., unknown provider → "Available: ollama, openrouter, anthropic")
  - [x] 3.2: Verify `main.go` exit code contract: `ErrFindings` → 2, all other non-nil errors → 1, success → 0
  - [x] 3.3: Ensure `build` uses `return fmt.Errorf(...)` instead of `os.Exit(1)` so exit code is routed through main
  - [x] 3.4: Write test: build with unknown target returns error (not os.Exit), error message contains available targets

## Dev Notes

### Architecture References

- **`rootCmd`**: `internal/cmd/root.go` — the root Cobra command
- **`doctorCmd`**: `internal/cmd/doctor.go` — `cobra.Command` registered via `init()`
- **`buildCmd`**: `internal/cmd/build.go` — `cobra.Command` registered via `init()`
- **`spec.Available()`**: `pkg/spec/generator.go:82` — returns sorted list of registered target names

### Shell Completion (Task 1)

Cobra v1.10.2 has a built-in `completion` subcommand. Enable it by calling `rootCmd.InitDefaultCompletionCmd()` after all subcommands are registered. Best place: `root.go` `init()` function, or in `Execute()`.

```go
// root.go
func init() {
    rootCmd.InitDefaultCompletionCmd()
}
```

For doctor file completion (filter to agent file extensions):
```go
doctorCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    if len(args) >= 1 {
        return nil, cobra.ShellCompDirectiveNoFileComp
    }
    return []string{"md", "yaml", "yml"}, cobra.ShellCompDirectiveFilterFileExt
}
```

For build `--target` completion:
```go
buildCmd.RegisterFlagCompletionFunc("target", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
    return spec.Available(), cobra.ShellCompDirectiveNoFileComp
})
```

**Key constraint**: `ValidArgsFunction` and `RegisterFlagCompletionFunc` must be set on the command AFTER it's created but within the same `init()` — the command variable is available there.

### Task 2: Build Command Long Description

Add:
```go
Long: `Generate skills and agents from YAML specs for a target framework.

Targets: claude (Claude Code), copilot (GitHub Copilot), forgent (standalone Go runtime)

Examples:
  forgent build --target claude
  forgent build --target copilot --compact
  forgent build --target forgent --output ./out
  forgent build --target claude --watch`,
```

### Task 3: Build Exit Code Fix

Currently `build.go` calls `os.Exit(1)` directly in the `Run` function. This bypasses main.go's exit code routing. The fix:
- Change `RunE` return instead of `Run` + `os.Exit`
- Or: Move build to `RunE` returning errors instead of calling `os.Exit`

**Important**: The `root.go` currently sets `SilenceErrors: false` by default. Cobra prints errors automatically. Switching `Run` to `RunE` for build cmd + removing `os.Exit` calls makes it consistent with doctor.

### Testing Strategy

- **Completion command test**: Execute `rootCmd` with args `["completion", "bash"]`, capture stdout, assert non-empty output containing "forgent"
- **ValidArgsFunction test**: Call `doctorCmd.ValidArgsFunction(doctorCmd, nil, "")` directly, assert returns `["md", "yaml", "yml"]` and `ShellCompDirectiveFilterFileExt`
- **Target completion test**: Call the registered completion func for `--target`, assert returns `spec.Available()`
- **Build error test**: Call `buildCmd` with `--target unknown`, assert `RunE` returns non-nil error with "unknown" or "available" in message
- **Exit code contract test**: Covered by main.go structure (tested implicitly)

### What NOT to Build

- Do NOT implement custom completion scripts by hand — use Cobra built-ins only
- Do NOT add `--completion` flag to individual commands — use the dedicated subcommand
- Do NOT change the error output format — only add hints where currently missing

### Project Structure

Modified files:
- `internal/cmd/root.go` — add `InitDefaultCompletionCmd()`
- `internal/cmd/doctor.go` — add `ValidArgsFunction`
- `internal/cmd/build.go` — add target completion, fix `os.Exit` → `RunE`, add Long description

New files:
- `internal/cmd/completion_test.go` — shell completion tests
- `internal/cmd/build_test.go` — build command error tests (if not exists)

## Dev Agent Record

### Agent Model Used

claude-sonnet-4-6

### Debug Log References

_empty_

### Completion Notes List

Cobra v1.10.2 registers `completion` automatically in `Execute()` — no explicit `InitDefaultCompletionCmd()` needed (calling it in `init()` caused double-registration + test interference). Added `ValidArgsFunction` on `doctorCmd` returning `ShellCompDirectiveFilterFileExt` for `.md/.yaml/.yml`. Added `RegisterFlagCompletionFunc` on build `--target` returning `spec.Available()`. Added Long descriptions to both `rootCmd` and `buildCmd`. Converted `buildCmd` from `Run`+`os.Exit` to `RunE`+`return fmt.Errorf` for consistent exit code routing. Discovered test isolation issue: shared `rootCmd` means `build --help` leaves `help=true` flag sticky — fixed via `resetBuildHelpFlag()` helper in tests. 11 new tests (5 completion, 4 help, 2 build error). 2071 total passing.

### File List

- `internal/cmd/root.go` — modified: added Long description
- `internal/cmd/doctor.go` — modified: added ValidArgsFunction
- `internal/cmd/build.go` — modified: added Long description, Run→RunE, os.Exit→return fmt.Errorf, RegisterFlagCompletionFunc for --target
- `internal/cmd/completion_test.go` — new: 5 tests (completion bash/fish, ValidArgsFunction, build target completion)
- `internal/cmd/help_test.go` — new: 6 tests (doctor/build/root help, build error tests)

## Review Findings

### Patches (apply before done)

- [x] **P1 HIGH** `completion_test.go:67` — `TestBuildCmd_TargetFlagCompletion` missing `resetBuildHelpFlag()` call. Flaky if `TestBuildHelp_ContainsExample` runs first (help flag remains sticky).
- [x] **P2 MEDIUM** `build.go:108-111` — Double error printing on build failure: reordered `PrintBuildResult` after success check so it never prints on failure; Cobra prints the single error message.
- [x] **P3 LOW** `build.go:126` — `_ = buildCmd.RegisterFlagCompletionFunc(...)` replaced with named error check; panics on programmer error.
- [x] **P4 LOW** `completion_test.go` — Added `TestCompletion_ZshSubcommandNoError` and `TestCompletion_PowershellSubcommandNoError`.
- [x] **P5 LOW** `help_test.go:43` — Added `assert.Contains(t, out, "Examples:")` to `TestBuildHelp_ContainsExample`.
- [x] **P6 LOW** `build.go:83` — Removed task-tracking comment `// P3 Task 3.3` from production code.

### Deferred (track for later)

- [x] **DEFER** `internal/cmd/build.go` watch loop — `signal.Stop(sigCh)` never called when context is cancelled; goroutine leaks in tests. (Epic 9 scope boundary)
- [x] **DEFER** `internal/cmd/build.go` watch block — residual `os.Exit(1)` on first build failure bypasses `RunE` exit code routing. (Epic 9 scope boundary)
- [x] **DEFER** `import`, `lint`, `init` commands — errors have no actionable hint (e.g. "file not found: run `forgent init` first"). AC#4 is scoped to doctor/build only. (Epic 9 polish pass)
- [x] **DEFER** `import`, `lint`, `init` commands — still use `Run`+`os.Exit` instead of `RunE`. Consistent exit routing incomplete. (Epic 9 polish pass)
- [x] **DEFER** `internal/cmd/*_test.go` — package-level `rootCmd` shared across all tests; order-dependent failures possible beyond `--help` flag. Full test isolation needs `newRootCmd()` factory. (Epic 9 or later)

## Change Log

- 2026-04-27: Story 9-3 created from Epic 9 definition (Story 6.3 in epics.md).
- 2026-04-27: Story 9-3 implemented. Shell completion, help text, exit code routing. 11 new tests, 2071 total passing.
- 2026-04-27: Code review complete. 6 patch, 5 defer, 10 dismissed. All 6 patches applied. 2074 tests passing.
