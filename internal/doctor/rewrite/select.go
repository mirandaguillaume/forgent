package rewrite

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/mirandaguillaume/forgent/internal/doctor/llmutil"
)

// SelectRecommendations displays findings and prompts the user to select which
// to apply. In non-TTY mode, all recommendations are selected automatically.
// The reader parameter allows testing with simulated stdin.
func SelectRecommendations(findings []llmutil.Finding, tty bool, reader io.Reader) ([]llmutil.Finding, error) {
	if len(findings) == 0 {
		return nil, nil
	}

	// Non-TTY: apply all
	if !tty {
		return findings, nil
	}

	if reader == nil {
		reader = os.Stdin
	}

	// Display numbered list
	bold := color.New(color.Bold)
	bold.Fprintf(os.Stderr, "\nSelect recommendations to apply:\n\n")
	for i, f := range findings {
		fmt.Fprintf(os.Stderr, "  %d. [%s] %s\n", i+1, f.Confidence, f.Issue)
		if f.SuggestedImprovement != "" {
			fmt.Fprintf(os.Stderr, "     Fix: %s\n", f.SuggestedImprovement)
		}
	}

	fmt.Fprintf(os.Stderr, "\nEnter numbers to apply (e.g., 1,3,5) or 'all': ")

	scanner := bufio.NewScanner(reader)
	if !scanner.Scan() {
		return nil, fmt.Errorf("no input received")
	}

	input := strings.TrimSpace(scanner.Text())
	if input == "" {
		return nil, fmt.Errorf("empty selection")
	}

	if strings.ToLower(input) == "all" {
		return findings, nil
	}

	return parseSelection(input, findings)
}

func parseSelection(input string, findings []llmutil.Finding) ([]llmutil.Finding, error) {
	parts := strings.Split(input, ",")
	var selected []llmutil.Finding
	seen := make(map[int]bool)

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid selection %q: expected a number", p)
		}
		if n < 1 || n > len(findings) {
			return nil, fmt.Errorf("selection %d out of range (1-%d)", n, len(findings))
		}
		if !seen[n] {
			seen[n] = true
			selected = append(selected, findings[n-1])
		}
	}

	if len(selected) == 0 {
		return nil, fmt.Errorf("no valid selections")
	}
	return selected, nil
}
