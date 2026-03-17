# Forgent Runtime Target — Design

**Date:** 2026-03-17
**Status:** Draft
**Goal:** Add `forgent build --target forgent` that compiles skill/agent YAMLs into a standalone Go program — a tailored runtime for that specific agent's DAG.

## Problem

Forgent already compiles to Claude Code (markdown) and GitHub Copilot (markdown). But the DAG engine (`pkg/dag`) we just built has no user-facing entrypoint. There's no way to actually run an agent's DAG.

The `forgent` build target closes this gap: it outputs a Go program that IS the runtime for a specific agent, with all prompts, edges, retry config, and input discovery baked in. You compile and run it directly — no YAML interpreter needed.

## Architecture

```
forgent build --target forgent
         │
         ▼
  skills/*.yaml + agents/*.yaml
         │
    ┌────┴───────────────────────────────────────┐
    │  internal/generator/forgent/forgent.go     │
    │  implements spec.Generator + AgentGenerator│
    └────┬───────────────────────────────────────┘
         │
         ▼
  .forgent/<agent_name>/main.go    ← standalone Go program
```

The generated `main.go`:
- Imports `pkg/dag` and `internal/llm`
- Constructs `*dag.Node` for each skill with prompts hardcoded
- Auto-wires the DAG via `dag.New()`
- Discovers inputs from the working directory
- Executes the DAG layer-parallel
- Prints terminal node outputs

Usage:
```bash
forgent build --target forgent           # generates .forgent/<agent>/main.go
cd .forgent/ci-reviewer && go run .      # run it
# or
go build -o ci-reviewer .forgent/ci-reviewer/ && ./ci-reviewer
```

## Generator Design

Implements the existing `spec.Generator` + `spec.AgentGenerator` interfaces. No `SkillGenerator` needed — skills are inlined into the agent program (always compact).

```go
// internal/generator/forgent/forgent.go
package forgent

import "github.com/mirandaguillaume/forgent/pkg/spec"

type forgentGenerator struct{}

func init() {
    spec.Register("forgent", func() spec.Generator {
        return &forgentGenerator{}
    })
}

func (g *forgentGenerator) Target() string          { return "forgent" }
func (g *forgentGenerator) DefaultOutputDir() string { return ".forgent" }
func (g *forgentGenerator) ContextDir() string       { return "" }
```

### Generated Code Structure

For an agent `ci-reviewer` with skills `[ts-linter, type-checker, review-commenter]`:

```go
// .forgent/ci_reviewer/main.go
package main

import (
    "context"
    "fmt"
    "os"
    "os/exec"
    "strings"

    "github.com/mirandaguillaume/forgent/internal/llm"
    "github.com/mirandaguillaume/forgent/pkg/dag"
)

func main() {
    apiKey := os.Getenv("ANTHROPIC_API_KEY")
    if apiKey == "" {
        fmt.Fprintln(os.Stderr, "ANTHROPIC_API_KEY is required")
        os.Exit(1)
    }
    provider, _ := llm.GetProvider("anthropic", apiKey)

    nodes := []*dag.Node{
        {
            ID:       "ts-linter",
            Consumes: []string{"file_tree", "source_code"},
            Produces: []string{"lint_results"},
            Run: makeSkillRunner(provider,
                "You are a TypeScript linter.\n\n"+
                "## Guardrails\n- Only report real issues\n\n"+
                "## Input\nFile tree:\n{{ .file_tree }}\n\nSource:\n{{ .source_code }}\n\n"+
                "## Output\nProduce: lint_results",
                []string{"lint_results"},
            ),
        },
        {
            ID:       "type-checker",
            Consumes: []string{"file_tree", "source_code"},
            Produces: []string{"type_errors"},
            Run: makeSkillRunner(provider,
                "You are a type checker...",
                []string{"type_errors"},
            ),
        },
        {
            ID:       "review-commenter",
            Consumes: []string{"git_diff", "lint_results", "type_errors"},
            Produces: []string{"review_comments"},
            Run: makeSkillRunner(provider,
                "You are a code reviewer...",
                []string{"review_comments"},
            ),
        },
    }

    d, err := dag.New(nodes...)
    if err != nil {
        fmt.Fprintf(os.Stderr, "DAG error: %v\n", err)
        os.Exit(1)
    }

    inputs := discoverInputs([]string{"file_tree", "source_code", "git_diff"})

    results, err := d.Execute(context.Background(), dag.WithInputs(inputs))
    if err != nil {
        fmt.Fprintf(os.Stderr, "Execution error: %v\n", err)
        os.Exit(1)
    }

    for k, v := range results {
        fmt.Printf("=== %s ===\n%v\n\n", k, v)
    }
}

// makeSkillRunner creates a dag.Node Run function that renders a prompt template
// with input values and calls the LLM provider.
func makeSkillRunner(
    provider llm.Provider,
    promptTemplate string,
    produces []string,
) func(context.Context, map[string]any) (map[string]any, error) {
    return func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
        prompt := promptTemplate
        for k, v := range inputs {
            prompt = strings.ReplaceAll(prompt, "{{ ."+k+" }}", fmt.Sprintf("%v", v))
        }
        response, err := provider.Complete(prompt)
        if err != nil {
            return nil, err
        }
        out := make(map[string]any, len(produces))
        for _, p := range produces {
            out[p] = response
        }
        return out, nil
    }
}

// discoverInputs auto-discovers input values from the working directory.
func discoverInputs(needed []string) map[string]any {
    inputs := make(map[string]any)
    for _, key := range needed {
        switch key {
        case "git_diff":
            out, err := exec.Command("git", "diff", "HEAD").Output()
            if err == nil {
                inputs[key] = string(out)
            }
        case "file_tree":
            out, err := exec.Command("git", "ls-files").Output()
            if err == nil {
                inputs[key] = string(out)
            }
        case "source_code":
            // read files from git ls-files, concatenate (with limits)
            inputs[key] = "(source code discovery)"
        case "pr_diff":
            out, err := exec.Command("gh", "pr", "diff").Output()
            if err == nil {
                inputs[key] = string(out)
            }
        case "pr_url":
            out, err := exec.Command("gh", "pr", "view", "--json", "url", "-q", ".url").Output()
            if err == nil {
                inputs[key] = strings.TrimSpace(string(out))
            }
        }
    }
    return inputs
}
```

### Prompt Construction

The prompt template for each skill is built from:
1. **Skill approach** — the strategy.approach field
2. **Steps** — numbered steps from strategy.steps
3. **Guardrails** — injected as constraints
4. **I/O contract** — explicit listing of consumes/produces types
5. **Examples** — if present, included as demonstrations
6. **Template variables** — `{{ .type_name }}` for each consumed type, replaced at runtime with actual input values

```go
// internal/generator/forgent/prompt.go
func buildPromptTemplate(skill model.SkillBehavior) string {
    var b strings.Builder
    b.WriteString("You are: " + skill.Skill + "\n\n")

    if skill.Strategy.Approach != "" {
        b.WriteString("## Approach\n" + skill.Strategy.Approach + "\n\n")
    }

    if len(skill.Strategy.Steps) > 0 {
        b.WriteString("## Steps\n")
        for i, step := range skill.Strategy.Steps {
            fmt.Fprintf(&b, "%d. %s\n", i+1, step)
        }
        b.WriteString("\n")
    }

    if len(skill.Guardrails) > 0 {
        b.WriteString("## Guardrails\n")
        for _, g := range skill.Guardrails {
            if s, ok := g.StringValue(); ok {
                b.WriteString("- " + s + "\n")
            }
        }
        b.WriteString("\n")
    }

    b.WriteString("## Input\n")
    for _, c := range skill.Context.Consumes {
        fmt.Fprintf(&b, "%s:\n{{ .%s }}\n\n", c, c)
    }

    b.WriteString("## Output\n")
    b.WriteString("Produce: " + strings.Join(skill.Context.Produces, ", ") + "\n")

    return b.String()
}
```

### go.mod Generation

The generated program needs a `go.mod` that references `pkg/dag` and `internal/llm`. Two approaches:

**Option A: Module replace directive** — the generated `go.mod` uses a `replace` directive pointing to the forgent repo root:
```
module ci-reviewer

go 1.22

require github.com/mirandaguillaume/forgent v0.0.0

replace github.com/mirandaguillaume/forgent => ../..
```

**Option B: Vendored runtime** — copy `pkg/dag/` and `internal/llm/` into the generated directory as standalone packages with no external dependency.

Recommended: **Option A** for v1 (simpler, works when forgent is installed locally). Option B for a future `--standalone` flag.

## What the Generator Produces

For each agent, `GenerateAgent()` returns the `main.go` content. The builder writes it to `.forgent/<agent_name>/main.go`. Additionally, the generator writes a `go.mod` file.

**Output files per agent:**
```
.forgent/
  ci_reviewer/
    main.go         ← generated Go program
    go.mod          ← module with replace directive
```

## Integration with Existing Builder

The builder (`internal/builder/builder.go`) already handles:
1. Loading skills from `skills/` directory
2. Loading agents from `agents/` directory
3. Resolving skill names to `SkillBehavior` structs
4. Calling `GenerateAgent(agent, resolvedSkills, outputDir)`
5. Writing output to `AgentPath()`

The forgent generator fits this pipeline exactly. The only difference: it doesn't implement `SkillGenerator` (skills are inlined), and `AgentPath()` returns `<agent_name>/main.go`.

The builder will also need to write the `go.mod` file — this can be done via the existing `GenerateAgent()` by having the generator return both files, or by having the generator write additional files directly.

## File Structure

```
internal/
  generator/
    forgent/
      forgent.go        # Generator registration, Target(), DefaultOutputDir()
      forgent_test.go
      agent.go           # GenerateAgent() — emits main.go code
      agent_test.go
      prompt.go          # buildPromptTemplate() — skill → prompt string
      prompt_test.go
```

## Testing Strategy

1. **Unit test `buildPromptTemplate()`** — given a SkillBehavior, produces correct prompt with template variables
2. **Unit test `GenerateAgent()`** — given an agent + skills, produces valid Go code
3. **Compilation test** — the generated code compiles (`go build` in a temp dir)
4. **Integration test** — build a test agent, run with a mock LLM provider, verify DAG execution order

## Future Targets

The same architecture scales to other frameworks:

| Target | Output | Language |
|--------|--------|----------|
| `forgent` | `.forgent/<agent>/main.go` | Go |
| `langchain` | `langchain/<agent>.py` | Python |
| `langgraph` | `langgraph/<agent>.py` | Python |
| `airflow` | `dags/<agent>.py` | Python |
| `temporal` | `temporal/<agent>/workflow.go` | Go |

Each implements `spec.AgentGenerator` and emits framework-specific code from the same skill/agent YAMLs.
