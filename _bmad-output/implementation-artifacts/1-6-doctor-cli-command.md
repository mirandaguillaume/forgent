# Story 1.6: Doctor CLI Command with TTY-Aware Output

Status: done

## Story

As a **developer**,
I want to run `forgent doctor agent.md` and see clear, colored recommendations,
so that I can quickly understand what to improve in my agent files.

## Acceptance Criteria

1. `forgent doctor <file>` auto-detects format, runs analysis, displays colored recommendations
2. Each recommendation shows: confidence badge [HIGH/MOD/LOW], issue title, explanation, suggested improvement
3. In CI pipe (no TTY): plain text, no color codes, suitable for log capture
4. Exit codes: 0 = no findings, 1 = error, 2 = recommendations found
5. No Ollama + no API key: static analysis output + "Install Ollama for full analysis" message
6. Ollama running but no model: detects available models or prompts `ollama pull <model>`
7. Provider selection via `--provider` flag (default: ollama, fallback: openrouter/anthropic)
8. Replaces the existing `forgent doctor` command completely (old diagnostics subsumed)
9. All existing tests pass (`go test ./... -count=1 -short`)

## Tasks / Subtasks

- [ ] Task 1: Rewrite doctor Cobra command (AC: #7, #8)
  - [ ] 1.1: Rewrite `internal/cmd/doctor.go` — replace existing diagnostics with new doctor
  - [ ] 1.2: Define flags: `--provider` (default "ollama"), `--model` (default from ADR-8), `--debug`
  - [ ] 1.3: Wire up provider selection: detect Ollama → try API fallback → static fallback
  - [ ] 1.4: Call format parser registry `DetectFormat()` on the input file
  - [ ] 1.5: Call analyzer `Analyze()` with parsed AgentAnalysis + selected provider
  - [ ] 1.6: Handle errors at each step with actionable messages (NFR28)
- [ ] Task 2: TTY-aware output rendering (AC: #1, #2, #3)
  - [ ] 2.1: Detect TTY via `os.Stdout` isatty check (use `golang.org/x/term` or `mattn/go-isatty`)
  - [ ] 2.2: Terminal mode: colored output via fatih/color — confidence badges, section headers, citations
  - [ ] 2.3: Pipe mode: plain text, no ANSI codes, same content structure
  - [ ] 2.4: Format each finding as: `[HIGH] Issue title\n  Why: explanation\n  Fix: suggested improvement`
- [ ] Task 3: Provider fallback chain (AC: #5, #6)
  - [ ] 3.1: Ollama detection: try `GET http://localhost:11434/api/tags` — if responds, Ollama is running
  - [ ] 3.2: Model detection: parse response from `/api/tags`, check if recommended model (or any model) is available
  - [ ] 3.3: No Ollama: check for `OPENROUTER_API_KEY` or `ANTHROPIC_API_KEY` env vars → use API provider
  - [ ] 3.4: No LLM at all: run static-only analysis (format detection + basic structure checks) + message
  - [ ] 3.5: Write tests for each fallback path
- [ ] Task 4: Exit codes and error handling (AC: #4, #9)
  - [ ] 4.1: Exit 0: command succeeds, no findings
  - [ ] 4.2: Exit 1: error (file not found, provider error, parse failure)
  - [ ] 4.3: Exit 2: findings found but not applied (read-only mode)
  - [ ] 4.4: Verify `go build ./cmd/forgent` still works
  - [ ] 4.5: Verify all existing tests pass

## Dev Notes

### Existing Doctor Command to Replace

The current `internal/cmd/doctor.go` runs lint + dependency + loop analysis on Forgent YAML files. The new doctor replaces it completely:

```go
// OLD: forgent doctor [path] — lint + analyze Forgent YAML
// NEW: forgent doctor <file> — analyze ANY agent file with LLM
```

The old functionality is subsumed by the Forgent YAML parser (Story 1.4) — when doctor analyzes a `.skill.yaml` file, it runs the linter and analyzer internally via the Forgent parser.

### Command Wiring — This is the Integration Point

This story wires together everything from Stories 1.2-1.5:

```go
func runDoctor(cmd *cobra.Command, args []string) error {
    filePath := args[0]
    content, err := os.ReadFile(filePath)
    // ...

    // 1. Detect format (Story 1.3 — parser registry)
    p, err := parser.DetectFormat(filePath, content)

    // 2. Parse agent file (Story 1.3/1.4 — format parsers)
    analysis, err := p.Parse(content)

    // 3. Select provider (Story 1.2 — Ollama adapter)
    provider, err := selectProvider(cmd)

    // 4. Analyze (Story 1.5 — analyzer skill)
    findings, err := analyzer.Analyze(analysis, provider)

    // 5. Render output (this story)
    renderFindings(findings, isTTY())

    // 6. Exit code
    if len(findings) > 0 { os.Exit(2) }
    return nil
}
```

### TTY Detection

Two options:
- `golang.org/x/term`: `term.IsTerminal(int(os.Stdout.Fd()))`
- `mattn/go-isatty`: `isatty.IsTerminal(os.Stdout.Fd())`

Check if either is already in `go.mod`. If not, `golang.org/x/term` is preferred (official Go package).

fatih/color already handles TTY detection internally (`color.NoColor` is set automatically). So for colored output, fatih/color may be sufficient without an explicit isatty check. But for exit code behavior and output format switching, explicit detection is needed.

### Output Format — Terminal Mode

```
forgent doctor .claude/agents/code-reviewer.md

5 recommendations found:

[HIGH] Missing guardrails section
  Your agent has no explicit behavioral constraints (timeouts, output limits).
  Research: Liu et al. (2024) — primacy bias improves constraint adherence.
  Fix: Add a guardrails section at the top with timeout and output limits.

[HIGH] No security declarations
  Only 14.5% of agent files address security (Agent READMEs, 2025).
  Fix: Add security declarations for filesystem and network access.

[MOD]  Monolithic structure — consider decomposition
  SkillsBench (2026): 2-3 skill compositions outperform monoliths by 18.6pp.
  Fix: Split into focused skills with explicit I/O contracts.

[LOW]  Instruction ordering not optimized
  Critical instructions are buried after general content.
  Fix: Move constraints and key instructions to the top of the file.

[INFO] Structural enforcement available
  Text instructions are ignored ~23% of the time on ordering tasks.
  Consider: Forgent runtime for deterministic execution guarantees.
```

### Output Format — Pipe Mode

Same content, no color, no badges:

```
5 recommendations found:
HIGH: Missing guardrails section — no explicit behavioral constraints
HIGH: No security declarations — filesystem/network access undeclared
MOD: Monolithic structure — consider decomposition into focused skills
LOW: Instruction ordering not optimized — move critical content to top
INFO: Structural enforcement available via Forgent runtime
```

### Provider Fallback Chain

```
1. --provider flag set? → use that provider
2. Ollama reachable at localhost:11434? → use Ollama
3. OPENROUTER_API_KEY set? → use OpenRouter
4. ANTHROPIC_API_KEY set? → use Anthropic
5. None available → static analysis only + guidance message
```

For Ollama model detection (AC #6), call `/api/tags`:

```go
func detectOllamaModels() ([]string, error) {
    resp, err := http.Get("http://localhost:11434/api/tags")
    // Parse response, extract model names
    // Return list of available models
}
```

If recommended model not available, pick the first available model. If no models at all, tell user to pull one.

### Previous Story Intelligence

From Stories 1.2-1.5:
- Ollama adapter at `internal/llm/ollama.go` — provider ready
- Format parsers at `internal/doctor/parser/` — Claude, Copilot, Forgent parsers ready
- Analyzer at `internal/doctor/analyzer/` — Analyze() function ready
- LLM utils at `internal/doctor/llmutil/` — ExtractYAML, ParseFindings ready

This story is pure integration + UI — no new domain logic.

### What NOT to Build

- Do NOT implement `--fix` (rewrite) — that's Epic 3
- Do NOT implement `--export-yaml` — that's Story 6.2
- Do NOT implement directory analysis — that's Story 2.3
- Do NOT implement research registry integration — that's Story 2.1
- DO implement the full read-only doctor experience end-to-end

### Project Structure

Modified files:
- `internal/cmd/doctor.go` — REWRITE (replace old doctor)

New files:
- `internal/cmd/doctor_test.go` — command tests (or extend existing)
- `internal/doctor/render.go` — finding output renderer (TTY + pipe modes)
- `internal/doctor/render_test.go` — renderer tests

### Dependencies Check

Verify these are in go.mod (should be already):
- `github.com/fatih/color` — colored output (existing)
- `github.com/spf13/cobra` — CLI framework (existing)
- `golang.org/x/term` or `mattn/go-isatty` — TTY detection (check if needed)

### References

- [Source: _bmad-output/planning-artifacts/architecture.md#CLI Tool Requirements] — command structure, interactivity model
- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-5] — Ollama adapter with actionable errors
- [Source: _bmad-output/planning-artifacts/architecture.md#ADR-7] — Config precedence (flags > env > config > defaults)
- [Source: _bmad-output/planning-artifacts/architecture.md#Data Flow] — Full doctor execution flow diagram
- [Source: _bmad-output/planning-artifacts/prd.md#FR7] — TTY-aware output
- [Source: _bmad-output/planning-artifacts/prd.md#FR10] — Static fallback
- [Source: _bmad-output/planning-artifacts/prd.md#FR11] — Ollama model detection
- [Source: _bmad-output/planning-artifacts/prd.md#NFR28-30] — Actionable errors, help examples, exit codes
- [Source: internal/cmd/doctor.go] — Existing doctor command to replace

## Dev Agent Record

### Agent Model Used

{{agent_model_name_version}}

### Debug Log References

### Completion Notes List

### File List
