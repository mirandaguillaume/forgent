package experiment

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"text/template"
	"time"

	"github.com/mirandaguillaume/forgent/internal/llm"
	"github.com/mirandaguillaume/forgent/pkg/budget"
	"github.com/mirandaguillaume/forgent/pkg/dag"
	"github.com/mirandaguillaume/forgent/pkg/qualitygate"
)

// effectiveGuardrails returns the guardrails to enforce for a given step.
// Step-level guardrails take precedence when declared; otherwise the task-level
// guardrails are used, preserving full backward compatibility (NFR40).
func effectiveGuardrails(step StepDef, task TaskDefinition) []string {
	if step.DisableTaskGuardrails {
		return nil
	}
	if len(step.Guardrails) > 0 {
		return step.Guardrails
	}
	return task.Guardrails
}

// RunRuntimeExperiment executes all tasks as Forgent DAG agents.
// Each task is converted to a DAG where steps are nodes with consumes/produces
// contracts. The DAG engine handles ordering, parallelism, and retry.
func RunRuntimeExperiment(config ExperimentConfig) ([]TaskResult, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	total := len(config.Tasks) * config.RunsPerTask
	done := 0
	label := config.Condition
	if label == "" {
		label = "runtime"
	}
	var results []TaskResult
	for i, task := range config.Tasks {
		skip := config.SkipMap[task.ID]
		for run := 0; run < config.RunsPerTask; run++ {
			done++
			if run < skip {
				fmt.Fprintf(os.Stderr, "  [%s] task %d/%d (%s, %d steps) run %d/%d (%d/%d total) (skipped, resumed)\n",
					label, i+1, len(config.Tasks), task.ID, len(task.Steps), run+1, config.RunsPerTask, done, total)
				continue
			}
			fmt.Fprintf(os.Stderr, "  [%s] task %d/%d (%s, %d steps) run %d/%d (%d/%d total)\n",
				label, i+1, len(config.Tasks), task.ID, len(task.Steps), run+1, config.RunsPerTask, done, total)
			retriever := config.ContextRetriever
			if retriever == nil {
				retriever = NilRetriever{}
			}
			condition := label
			result := runRuntimeTask(task, config.Provider, retriever, condition)
			results = append(results, result)
			if config.OnResult != nil {
				config.OnResult(result, run)
			}
			if IsAPIFatalError(result.Error) {
				fmt.Fprintf(os.Stderr, "  ✗ fatal API error, aborting experiment: %v\n", result.Error)
				return results, fmt.Errorf("%w: %v", ErrAPIFatal, result.Error)
			}
		}
	}
	return results, nil
}

func runRuntimeTask(task TaskDefinition, provider llm.Provider, retriever ContextRetriever, condition string) TaskResult {
	start := time.Now()

	// Collect ALL node outputs (not just terminal) via a shared map + callback.
	allOutputs := make(map[string]any)
	var mu sync.Mutex
	var llmCalls int32

	// Wrap provider to count LLM calls
	countingProvider := &callCountingProvider{inner: provider, calls: &llmCalls}

	// Build sandbox from workspace (if tool-based task)
	var sandbox *SandboxFS
	if task.Workspace != nil {
		sandbox = buildSandboxFromWorkspace(task)
	} else {
		sandbox = NewSandboxFS(nil, nil, SandboxPolicy{})
	}

	d, nodeOutputKeys, err := buildTaskDAG(task, countingProvider, sandbox, retriever, func(nodeID string, out map[string]any) {
		mu.Lock()
		for k, v := range out {
			allOutputs[k] = v
		}
		mu.Unlock()
	})
	if err != nil {
		return TaskResult{
			TaskID:    task.ID,
			Condition: condition,
			Error:     fmt.Errorf("build DAG: %w", err),
		}
	}
	_ = nodeOutputKeys

	inputs := map[string]any{
		"task_input": task.Input,
	}

	_, err = d.Execute(context.Background(), dag.WithInputs(inputs))
	wallClock := time.Since(start).Milliseconds()

	if err != nil {
		return TaskResult{
			TaskID:      task.ID,
			Condition:   condition,
			WallClockMs: wallClock,
			LLMCalls:    int(llmCalls),
			Error:       fmt.Errorf("execute DAG: %w", err),
		}
	}

	result := evaluateRuntimeOutputs(task, allOutputs)
	result.Condition = condition
	result.WallClockMs = wallClock

	// Attach sandbox metrics if tool-based
	if sandbox != nil {
		result.ActionLog = sandbox.Log()
		result.ActionMetrics = ComputeActionMetrics(sandbox.Log())
	}
	result.LLMCalls = int(llmCalls)
	return result
}

// callCountingProvider wraps an LLM provider to count calls.
type callCountingProvider struct {
	inner llm.Provider
	calls *int32
}

func (p *callCountingProvider) Complete(prompt string) (string, error) {
	atomic.AddInt32(p.calls, 1)
	return p.inner.Complete(prompt)
}

// outputCapture is called after each node completes with its outputs.
type outputCapture func(nodeID string, out map[string]any)

// buildTaskDAG constructs a DAG from a task definition.
// Each step becomes a node; consumes/produces contracts wire the topology.
// The capture callback receives all node outputs (not just terminal).
//
// Four step behaviors are supported:
//   - llm (default): calls the LLM provider, guardrails enforced by code after each call
//   - llm with tools: LLM + tool-call loop through sandbox (Story 5-4)
//   - static: executes a Go text/template against inputs, no LLM call
//   - assert: runs a guardrail check against inputs, fails if violated
func buildTaskDAG(task TaskDefinition, provider llm.Provider, sandbox *SandboxFS, retriever ContextRetriever, capture outputCapture) (*dag.DAG, map[string][]string, error) {
	var nodes []*dag.Node
	outputKeys := make(map[string][]string) // nodeID → produces keys

	// Write task_context.md to sandbox so any step can access full task context via file_read.
	// For workspace tasks, buildSandboxFromWorkspace pre-populates this file (handles ReadOnly);
	// for non-workspace tasks this Write creates it fresh.
	if sandbox != nil {
		if err := sandbox.Write("dag", "task_context.md", taskContextContent(task)); err != nil {
			fmt.Fprintf(os.Stderr, "warn: sandbox write task_context.md for %s: %v\n", task.ID, err)
		}
	}

	for _, step := range task.Steps {
		stepCopy := step // capture for closure
		consumes := step.Consumes
		if len(consumes) == 0 && len(nodes) == 0 {
			consumes = []SlotSpec{{Key: "task_input"}}
		}
		produces := defaultProduces(step.ID, step.Produces)
		outputKeys[step.ID] = SlotKeys(produces)

		var runFn func(ctx context.Context, inputs map[string]any) (map[string]any, error)

		switch stepCopy.Kind {
		case StepKindStatic:
			// Static: execute Go template against inputs — no LLM, no tokens
			runFn = func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
				tmpl, err := template.New(stepCopy.ID).Parse(stepCopy.Template)
				if err != nil {
					return nil, fmt.Errorf("step %s: parse template: %w", stepCopy.ID, err)
				}
				data := map[string]any{"inputs": inputs}
				var buf bytes.Buffer
				if err := tmpl.Execute(&buf, data); err != nil {
					return nil, fmt.Errorf("step %s: execute template: %w", stepCopy.ID, err)
				}
				out := make(map[string]any)
				for _, p := range stepCopy.Produces {
					out[p.Key] = buf.String()
				}
				if capture != nil {
					capture(stepCopy.ID, out)
				}
				return out, nil
			}

		case StepKindAssert:
			// Assert: verify guardrails/checks against inputs — no LLM, fail on violation
			runFn = func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
				// Aggregate all input text
				var allText strings.Builder
				for _, v := range inputs {
					if s, ok := v.(string); ok {
						allText.WriteString(s)
						allText.WriteString("\n")
					}
				}
				lower := strings.ToLower(allText.String())

				// Run the named check
				switch stepCopy.Check {
				case "guardrails_respected":
					for _, g := range effectiveGuardrails(stepCopy, task) {
						if !checkGuardrail(lower, g) {
							return nil, fmt.Errorf("step %s: guardrail violated: %s", stepCopy.ID, g)
						}
					}
				case "all_inputs_present":
					for k, v := range inputs {
						if s, ok := v.(string); ok && strings.TrimSpace(s) == "" {
							return nil, fmt.Errorf("step %s: empty input: %s", stepCopy.ID, k)
						}
					}
				case "no_empty_steps":
					for k, v := range inputs {
						if s, ok := v.(string); ok && len(strings.TrimSpace(s)) < 10 {
							return nil, fmt.Errorf("step %s: suspiciously short input %s (%d chars)", stepCopy.ID, k, len(s))
						}
					}
				default:
					// Unknown check — pass through (lenient)
				}

				out := make(map[string]any)
				for _, p := range stepCopy.Produces {
					out[p.Key] = allText.String()
				}
				if capture != nil {
					capture(stepCopy.ID, out)
				}
				return out, nil
			}

		default:
			// LLM step: call provider with guardrail enforcement + retry
			// All LLM steps use file-based slot passing via the sandbox:
			// consumed slots are written to slots/{key}.md before the LLM call,
			// and the prompt lists file paths instead of embedding content inline.
			// file_read is always available so the LLM can pull context on demand.
			stepSandbox := sandbox // capture sandbox reference for closure
			stepConsumes := consumes
			runFn = func(ctx context.Context, inputs map[string]any) (map[string]any, error) {
				// Write consumed slot values to sandbox files before calling the LLM.
				if stepSandbox != nil {
					for k, v := range inputs {
						if s, ok := v.(string); ok && k != "task_input" {
							if err := stepSandbox.Write(stepCopy.ID, fmt.Sprintf("slots/%s.md", k), []byte(s)); err != nil {
								fmt.Fprintf(os.Stderr, "warn: sandbox write input slot %s for step %s: %v\n", k, stepCopy.ID, err)
							}
						}
					}
				}

				// Build base prompt: minimum required = step instructions + file paths.
				var base strings.Builder
				base.WriteString(fmt.Sprintf("# Task: %s\n\n", task.Name))
				base.WriteString(fmt.Sprintf("## Step: %s\n\n%s\n\n", stepCopy.Name, stepCopy.Prompt))

				// Declared tools (from step YAML) + always include file_read for context access.
				// Deduplicate so file_read isn't listed twice if the step already declares it.
				seenTools := map[string]bool{"file_read": true}
				allTools := []string{"file_read"}
				for _, t := range stepCopy.Tools {
					if !seenTools[t] {
						seenTools[t] = true
						allTools = append(allTools, t)
					}
				}
				base.WriteString("## Available Tools\n\n")
				base.WriteString("Use XML format: `<tool name=\"tool_name\" path=\"file_path\"/>`\n\n")
				for _, t := range allTools {
					base.WriteString(fmt.Sprintf("- %s\n", t))
				}
				base.WriteString("\n")

				stepGuardrails := effectiveGuardrails(stepCopy, task)
				if len(stepGuardrails) > 0 {
					base.WriteString("## Constraints (MUST respect)\n\n")
					for _, g := range stepGuardrails {
						base.WriteString(fmt.Sprintf("- %s\n", g))
					}
					base.WriteString("\n")
				}

				// Context injection: pre-loaded excerpts from consumed slots (if retriever set).
				// The retriever can read sandbox files because slot writes happened above.
				if retriever != nil {
					if excerpt, _ := retriever.Retrieve(stepCopy, stepSandbox, stepConsumes); excerpt != "" {
						base.WriteString("## Context (pre-loaded excerpt)\n\n")
						base.WriteString(excerpt)
					}
				}

				// File listing replaces inline input injection — LLM reads on demand.
				base.WriteString("## Available files (use file_read to access)\n")
				base.WriteString("- task_context.md — full task context and original input\n")
				for _, slot := range stepConsumes {
					if slot.Key != "task_input" {
						base.WriteString(fmt.Sprintf("- slots/%s.md — output from: %s\n", slot.Key, slot.Key))
					}
				}
				base.WriteString("\n")
				basePrompt := base.String()

				// Try up to 3 times, checking guardrails after each attempt.
				maxAttempts := 3
				var lastResponse string
				for attempt := 0; attempt < maxAttempts; attempt++ {
					prompt := basePrompt
					if attempt > 0 {
						prompt += fmt.Sprintf("\n\n[Previous attempt violated constraints. Retry %d/%d. Strictly respect ALL constraints listed above.]\n", attempt+1, maxAttempts)
					}

					response, err := provider.Complete(prompt)
					if err != nil {
						return nil, fmt.Errorf("step %s attempt %d: %w", stepCopy.ID, attempt+1, err)
					}

					// Tool-call loop: always active when sandbox is present.
					// file_read is always allowed; declared Tools add further capabilities.
					if stepSandbox != nil {
						allowedTools := map[string]bool{"file_read": true}
						for _, t := range stepCopy.Tools {
							allowedTools[t] = true
						}
						const maxToolIterations = 5
						toolPrompt := prompt
						for iter := 0; iter < maxToolIterations; iter++ {
							calls := ParseToolCalls(response)
							if len(calls) == 0 {
								break
							}
							var results []ToolResult
							for _, call := range calls {
								if !allowedTools[call.Name] {
									results = append(results, ToolResult{Name: call.Name, Blocked: true, Reason: fmt.Sprintf("tool %q not available in this step (allowed: %v)", call.Name, stepCopy.Tools)})
									continue
								}
								results = append(results, ExecuteToolWithPolicy(stepSandbox, stepCopy.ID, call, stepCopy.Sandbox))
							}
							toolPrompt += "\n\n" + FormatToolResults(results) +
								"\nContinue your analysis based on the tool results above. If you need more tools, use them. Otherwise, provide your final answer.\n"

							response, err = provider.Complete(toolPrompt)
							if err != nil {
								return nil, fmt.Errorf("step %s tool iteration %d: %w", stepCopy.ID, iter+1, err)
							}
						}
					}

					lastResponse = response

					// Check guardrails by code — use step-level when declared, task-level otherwise.
					if len(stepGuardrails) > 0 {
						lower := strings.ToLower(response)
						allPassed := true
						for _, g := range stepGuardrails {
							if !checkGuardrail(lower, g) {
								allPassed = false
								break
							}
						}
						if allPassed {
							break
						}
					} else {
						break
					}
				}

				// Write step output to sandbox so downstream steps can file_read it.
				if stepSandbox != nil {
					for _, p := range stepCopy.Produces {
						if err := stepSandbox.Write(stepCopy.ID, fmt.Sprintf("slots/%s.md", p.Key), []byte(lastResponse)); err != nil {
							fmt.Fprintf(os.Stderr, "warn: sandbox write produces slot %s for step %s: %v\n", p.Key, stepCopy.ID, err)
						}
					}
				}

				out := make(map[string]any)
				for _, p := range stepCopy.Produces {
					out[p.Key] = lastResponse
				}
				if capture != nil {
					capture(stepCopy.ID, out)
				}
				return out, nil
			}
		}

		node := &dag.Node{
			ID:       step.ID,
			Kind:     dag.KindTask,
			Consumes: SlotKeys(consumes),
			Produces: SlotKeys(produces),
			Timeout:  120 * time.Second,
			Run:      runFn,
		}

		// Harness: attach middleware guards to all LLM nodes (always on — the
		// harness IS the runtime's value proposition)
		if stepCopy.Kind == StepKindLLM {
			// Ensure context files are accessible via file_read even when the step
			// declares a restrictive per-step sandbox policy. ExecuteToolWithPolicy
			// checks stepCopy.Sandbox.AllowedPaths before the sandbox-level glob list,
			// so we must augment the per-step policy here.
			if stepCopy.Sandbox != nil {
				augmented := make([]string, len(stepCopy.Sandbox.AllowedPaths), len(stepCopy.Sandbox.AllowedPaths)+2)
				copy(augmented, stepCopy.Sandbox.AllowedPaths)
				augmented = append(augmented, "task_context.md", "slots/**")
				stepCopy.Sandbox = &StepSandboxPolicy{AllowedPaths: augmented}
			}

			var middlewares []dag.Middleware

			// Quality gate: enforce format template from SlotSpec contracts (if declared).
			// expected_keywords is kept on StepDef for evaluation metrics only, not enforcement.
			var producesTemplate, consumesTemplate string
			for _, slot := range stepCopy.Produces {
				if slot.TemplateFile != "" && task.TaskDir == "" {
					return nil, nil, fmt.Errorf("step %s: template_file %q requires TaskDir (load task via LoadTasks)", stepCopy.ID, slot.TemplateFile)
				}
				tmpl, err := slot.ResolveTemplate(task.TaskDir)
				if err != nil {
					return nil, nil, fmt.Errorf("step %s produces: %w", stepCopy.ID, err)
				}
				if tmpl != "" {
					producesTemplate = tmpl
					break
				}
			}
			for _, slot := range stepCopy.Consumes {
				if slot.TemplateFile != "" && task.TaskDir == "" {
					return nil, nil, fmt.Errorf("step %s: template_file %q requires TaskDir (load task via LoadTasks)", stepCopy.ID, slot.TemplateFile)
				}
				tmpl, err := slot.ResolveTemplate(task.TaskDir)
				if err != nil {
					return nil, nil, fmt.Errorf("step %s consumes: %w", stepCopy.ID, err)
				}
				if tmpl != "" {
					consumesTemplate = tmpl
					break
				}
			}
			policy := &qualitygate.QualityGatePolicy{
				ProducesTemplate: producesTemplate,
				ConsumesTemplate: consumesTemplate,
			}
			middlewares = append(middlewares, qualitygate.QualityGateMiddleware(policy))

			// Budget guard: default limits
			middlewares = append(middlewares, budget.BudgetMiddleware(&budget.BudgetPolicy{
				MaxTokens: 8192,
				Timeout:   60 * time.Second,
			}))

			node.Middlewares = middlewares
			node.MaxRetries = 2
		}

		nodes = append(nodes, node)
	}

	d, err := dag.New(nodes...)
	return d, outputKeys, err
}

// taskContextContent builds the content of task_context.md from a task definition.
// This file is always present in the sandbox so any step can access full task context
// via file_read without it being injected inline into the prompt.
func taskContextContent(task TaskDefinition) []byte {
	var ctx strings.Builder
	fmt.Fprintf(&ctx, "# Task: %s\n\n", task.Name)
	if task.Description != "" {
		fmt.Fprintf(&ctx, "## Description\n%s\n\n", task.Description)
	}
	fmt.Fprintf(&ctx, "## Original Input\n%s\n\n", task.Input)
	if len(task.Guardrails) > 0 {
		ctx.WriteString("## Guardrails\n")
		for _, g := range task.Guardrails {
			fmt.Fprintf(&ctx, "- %s\n", g)
		}
	}
	return []byte(ctx.String())
}

// buildSandboxFromWorkspace creates a SandboxFS from a task's workspace definition.
func buildSandboxFromWorkspace(task TaskDefinition) *SandboxFS {
	files := make(map[string][]byte)
	var traps []string

	// Pre-populate task_context.md so it exists even in ReadOnly sandboxes.
	// buildTaskDAG will also Write it (overwrite), but for ReadOnly workspaces
	// this pre-population is the only way the file gets created.
	files["task_context.md"] = taskContextContent(task)

	if task.Workspace != nil {
		for _, f := range task.Workspace.Files {
			files[f.Path] = []byte(f.Content)
		}
		for _, t := range task.Workspace.Traps {
			files[t.Path] = []byte(t.Content) // traps are readable files that SHOULD be denied
			traps = append(traps, t.Path)
		}
	}

	// Build policy from step-level sandbox rules (union of all steps)
	var allowedGlobs []string
	seen := make(map[string]bool)
	for _, step := range task.Steps {
		if step.Sandbox != nil {
			for _, p := range step.Sandbox.AllowedPaths {
				if !seen[p] {
					allowedGlobs = append(allowedGlobs, p)
					seen[p] = true
				}
			}
		}
	}

	// When the policy is already restrictive, also allow task context and slot files.
	if len(allowedGlobs) > 0 {
		if !seen["task_context.md"] {
			allowedGlobs = append(allowedGlobs, "task_context.md")
		}
		if !seen["slots/**"] {
			allowedGlobs = append(allowedGlobs, "slots/**")
		}
	}

	policy := SandboxPolicy{
		AllowedGlobs:        allowedGlobs,
		DeniedGlobs:         task.Workspace.DeniedPaths,
		DeniedWritePatterns: task.Workspace.DeniedWritePatterns,
		MaxFileSize:         task.Workspace.MaxFileSize,
		ReadOnly:            task.Workspace.ReadOnly,
	}
	return NewSandboxFS(files, traps, policy)
}

// evaluateRuntimeOutputs checks the DAG outputs against the task definition.
func evaluateRuntimeOutputs(task TaskDefinition, outputs map[string]any) TaskResult {
	result := TaskResult{
		TaskID:    task.ID,
		Condition: "runtime",
	}

	// Collect all output text
	var allOutput strings.Builder
	totalTokens := 0

	for _, step := range task.Steps {
		sr := StepResult{StepID: step.ID}

		// Check each produce key for this step
		for _, slot := range defaultProduces(step.ID, step.Produces) {
			key := slot.Key
			if val, ok := outputs[key]; ok {
				if s, ok := val.(string); ok && len(s) > 0 {
					sr.Completed = true
					sr.Output = s
					allOutput.WriteString(s)
					allOutput.WriteString("\n")
					sr.TokensUsed = len(s) / 4
					totalTokens += sr.TokensUsed
				}
			}
		}

		result.StepResults = append(result.StepResults, sr)
	}

	result.TokensUsed = totalTokens

	// Completion
	completed := 0
	for _, sr := range result.StepResults {
		if sr.Completed {
			completed++
		}
	}
	result.Completed = completed == len(task.Steps)

	// Ordering: DAG enforces order, so if all completed, order is correct
	if result.Completed {
		result.OrderingScore = 1.0
	} else if len(task.Steps) > 0 {
		result.OrderingScore = float64(completed) / float64(len(task.Steps))
	}

	// GuardrailScore: per-step evaluation using effectiveGuardrails.
	// Each step's output is checked against its own effective guardrails (step-level
	// when declared, task-level as fallback). Steps with no effective guardrails are skipped.
	totalGuardrails := 0
	respectedGuardrails := 0
	for i, step := range task.Steps {
		gs := effectiveGuardrails(step, task)
		if len(gs) == 0 {
			continue
		}
		var output string
		if i < len(result.StepResults) {
			output = strings.ToLower(result.StepResults[i].Output)
		}
		for _, g := range gs {
			totalGuardrails++
			if checkGuardrail(output, g) {
				respectedGuardrails++
			}
		}
	}
	if totalGuardrails > 0 {
		result.GuardrailScore = float64(respectedGuardrails) / float64(totalGuardrails)
	} else {
		result.GuardrailScore = 1.0
	}

	return result
}
