// Package rewrite implements the LLM-powered agent file rewriter for doctor --fix.
// It takes an original file and selected recommendations, asks the LLM to produce
// an improved version, and returns the raw rewritten content.
package rewrite

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/mirandaguillaume/forgent/internal/doctor/llmutil"
	"github.com/mirandaguillaume/forgent/internal/llm"
)

// maxPromptContentBytes limits the original file content included in the prompt.
// Files larger than this are truncated with a notice to the LLM.
const maxPromptContentBytes = 32768 // ~8K tokens

// Rewrite sends the original file and selected recommendations to the LLM
// and returns the complete rewritten file content.
func Rewrite(original []byte, findings []llmutil.Finding, format string, provider llm.Provider) ([]byte, error) {
	if len(findings) == 0 {
		return original, nil
	}

	prompt := buildPrompt(original, findings, format)

	response, err := provider.Complete(prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM rewrite failed: %w", err)
	}

	content := extractFileContent(response)
	if strings.TrimSpace(content) == "" {
		return nil, fmt.Errorf("LLM returned empty rewrite")
	}

	return []byte(content), nil
}

func buildPrompt(original []byte, findings []llmutil.Finding, format string) string {
	var b strings.Builder

	b.WriteString("You are rewriting an agent definition file. Apply ONLY the selected improvements below.\n\n")
	b.WriteString("RULES:\n")
	b.WriteString("1. Preserve ALL frontmatter fields exactly — do not add, remove, or rename any\n")
	b.WriteString("2. Preserve ALL existing sections — you may add content but never remove sections\n")
	b.WriteString("3. Preserve the file's original formatting style (indentation, spacing)\n")
	b.WriteString("4. Output the COMPLETE rewritten file — not a diff, not a partial\n")
	b.WriteString("5. Preserve all non-ASCII characters, emoji, and special characters exactly\n\n")

	b.WriteString(fmt.Sprintf("FORMAT: %s\n\n", format))

	b.WriteString("ORIGINAL FILE:\n---\n")
	content := string(original)
	if len(original) > maxPromptContentBytes {
		content = string(original[:maxPromptContentBytes]) + "\n\n[... truncated for prompt size ...]"
	}
	b.WriteString(content)
	b.WriteString("\n---\n\n")

	b.WriteString("IMPROVEMENTS TO APPLY:\n")
	for i, f := range findings {
		b.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, f.Category, f.Issue))
		if f.SuggestedImprovement != "" {
			b.WriteString(fmt.Sprintf("   Suggestion: %s\n", f.SuggestedImprovement))
		}
	}
	b.WriteString("\nOUTPUT THE COMPLETE REWRITTEN FILE:\n")

	return b.String()
}

// fenceRe matches markdown code fences.
var fenceRe = regexp.MustCompile("(?s)```(?:markdown|md|ya?ml)?\\s*\\n(.*?)\\n```")

// extractFileContent strips markdown fences and LLM preamble from the response.
// Local models often prepend "Here is the rewritten file:" before the content.
func extractFileContent(response string) string {
	// Try code fences first
	matches := fenceRe.FindStringSubmatch(response)
	if len(matches) > 1 {
		return stripPreamble(matches[1])
	}
	return stripPreamble(strings.TrimSpace(response))
}

// frontmatterStartRe matches the start of YAML frontmatter: --- followed by a key: value line.
var frontmatterStartRe = regexp.MustCompile(`(?m)^---\n[a-zA-Z_][a-zA-Z0-9_]*:`)

// stripPreamble removes any text before the first YAML frontmatter block.
// Handles LLM preamble like "Here is the rewritten agent definition file:"
func stripPreamble(content string) string {
	if strings.HasPrefix(strings.TrimLeft(content, " \t\r\n"), "---\n") {
		return content
	}

	loc := frontmatterStartRe.FindStringIndex(content)
	if loc != nil && loc[0] > 0 {
		return content[loc[0]:]
	}
	return content
}
