package forgent

import (
	"fmt"
	"strings"

	"github.com/mirandaguillaume/forgent/pkg/model"
)

// GenerateAgentGo generates a complete Go main.go program for an agent's DAG runtime.
func GenerateAgentGo(agent model.AgentComposition, skills []model.SkillBehavior) string {
	var b strings.Builder

	// Collect all consumed types across skills that have no internal producer
	allProduced := map[string]bool{}
	for _, s := range skills {
		for _, p := range s.Context.Produces {
			allProduced[p] = true
		}
	}
	externalInputs := []string{}
	seen := map[string]bool{}
	for _, s := range skills {
		for _, c := range s.Context.Consumes {
			if !allProduced[c] && !seen[c] {
				externalInputs = append(externalInputs, c)
				seen[c] = true
			}
		}
	}

	// Package + imports
	b.WriteString("package main\n\n")
	b.WriteString("import (\n")
	b.WriteString("\t\"context\"\n")
	b.WriteString("\t\"fmt\"\n")
	b.WriteString("\t\"os\"\n")
	b.WriteString("\t\"os/exec\"\n")
	b.WriteString("\t\"strings\"\n")
	b.WriteString("\t\"time\"\n\n")
	b.WriteString("\t\"github.com/mirandaguillaume/forgent/internal/llm\"\n")
	b.WriteString("\t\"github.com/mirandaguillaume/forgent/pkg/dag\"\n")
	b.WriteString(")\n\n")

	// main function
	b.WriteString("func main() {\n")
	b.WriteString("\tapiKey := os.Getenv(\"ANTHROPIC_API_KEY\")\n")
	b.WriteString("\tif apiKey == \"\" {\n")
	b.WriteString("\t\tfmt.Fprintln(os.Stderr, \"ANTHROPIC_API_KEY is required\")\n")
	b.WriteString("\t\tos.Exit(1)\n")
	b.WriteString("\t}\n")
	b.WriteString("\tprovider, err := llm.GetProvider(\"anthropic\", apiKey)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\tfmt.Fprintf(os.Stderr, \"provider error: %v\\n\", err)\n")
	b.WriteString("\t\tos.Exit(1)\n")
	b.WriteString("\t}\n\n")

	// Nodes
	b.WriteString("\tnodes := []*dag.Node{\n")
	for _, s := range skills {
		prompt := BuildPromptTemplate(s)
		escapedPrompt := strings.ReplaceAll(prompt, "`", "` + \"`\" + `")

		b.WriteString("\t\t{\n")
		fmt.Fprintf(&b, "\t\t\tID: %q,\n", s.Skill)

		// Consumes
		if len(s.Context.Consumes) > 0 {
			consumeStrs := make([]string, len(s.Context.Consumes))
			for i, c := range s.Context.Consumes {
				consumeStrs[i] = fmt.Sprintf("%q", c)
			}
			fmt.Fprintf(&b, "\t\t\tConsumes: []string{%s},\n", strings.Join(consumeStrs, ", "))
		}

		// Produces
		if len(s.Context.Produces) > 0 {
			produceStrs := make([]string, len(s.Context.Produces))
			for i, p := range s.Context.Produces {
				produceStrs[i] = fmt.Sprintf("%q", p)
			}
			fmt.Fprintf(&b, "\t\t\tProduces: []string{%s},\n", strings.Join(produceStrs, ", "))
		}

		// Retry / timeout (defaults: 0) — use time.Duration(0) to avoid vet warnings
		fmt.Fprintf(&b, "\t\t\tMaxRetries: %d,\n", 0)
		fmt.Fprintf(&b, "\t\t\tTimeout:    time.Duration(%d),\n", 0)

		// Run function
		fmt.Fprintf(&b, "\t\t\tRun: makeSkillRunner(provider, `%s`, []string{", escapedPrompt)
		for i, p := range s.Context.Produces {
			if i > 0 {
				b.WriteString(", ")
			}
			fmt.Fprintf(&b, "%q", p)
		}
		b.WriteString("}),\n")
		b.WriteString("\t\t},\n")
	}
	b.WriteString("\t}\n\n")

	// Build and execute DAG
	b.WriteString("\td, err := dag.New(nodes...)\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\tfmt.Fprintf(os.Stderr, \"DAG error: %v\\n\", err)\n")
	b.WriteString("\t\tos.Exit(1)\n")
	b.WriteString("\t}\n\n")

	// Discover inputs
	fmt.Fprintf(&b, "\tinputs := discoverInputs([]string{%s})\n\n",
		joinQuoted(externalInputs))

	b.WriteString("\tresults, err := d.Execute(context.Background(), dag.WithInputs(inputs))\n")
	b.WriteString("\tif err != nil {\n")
	b.WriteString("\t\tfmt.Fprintf(os.Stderr, \"Execution error: %v\\n\", err)\n")
	b.WriteString("\t\tos.Exit(1)\n")
	b.WriteString("\t}\n\n")
	b.WriteString("\tfor k, v := range results {\n")
	b.WriteString("\t\tfmt.Printf(\"=== %s ===\\n%v\\n\\n\", k, v)\n")
	b.WriteString("\t}\n")
	b.WriteString("}\n\n")

	// makeSkillRunner helper
	b.WriteString(makeSkillRunnerCode())

	// discoverInputs helper
	b.WriteString(discoverInputsCode())

	return b.String()
}

// GenerateGoMod generates a go.mod file for the generated runtime.
func GenerateGoMod(agentName string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "module %s\n\n", agentName)
	b.WriteString("go 1.22\n\n")
	b.WriteString("require github.com/mirandaguillaume/forgent v0.0.0\n\n")
	b.WriteString("replace github.com/mirandaguillaume/forgent => ../..\n")
	return b.String()
}

func joinQuoted(items []string) string {
	quoted := make([]string, len(items))
	for i, s := range items {
		quoted[i] = fmt.Sprintf("%q", s)
	}
	return strings.Join(quoted, ", ")
}

func makeSkillRunnerCode() string {
	return `func makeSkillRunner(
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

`
}

func discoverInputsCode() string {
	return `func discoverInputs(needed []string) map[string]any {
	inputs := make(map[string]any)
	for _, key := range needed {
		switch key {
		case "git_diff":
			if out, err := exec.Command("git", "diff", "HEAD").Output(); err == nil {
				inputs[key] = string(out)
			}
		case "file_tree":
			if out, err := exec.Command("git", "ls-files").Output(); err == nil {
				inputs[key] = string(out)
			}
		case "source_code":
			if out, err := exec.Command("git", "ls-files").Output(); err == nil {
				files := strings.Split(strings.TrimSpace(string(out)), "\n")
				var sb strings.Builder
				for _, f := range files {
					if content, err := os.ReadFile(f); err == nil {
						fmt.Fprintf(&sb, "=== %s ===\n%s\n", f, string(content))
						if sb.Len() > 100000 {
							break
						}
					}
				}
				inputs[key] = sb.String()
			}
		case "pr_diff":
			if out, err := exec.Command("gh", "pr", "diff").Output(); err == nil {
				inputs[key] = string(out)
			}
		case "pr_url":
			if out, err := exec.Command("gh", "pr", "view", "--json", "url", "-q", ".url").Output(); err == nil {
				inputs[key] = strings.TrimSpace(string(out))
			}
		}
	}
	return inputs
}
`
}
