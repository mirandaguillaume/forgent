package cmd

import (
	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:   "forgent",
	Short: "Forgent — Forge agents from composable skill specs",
	Long: `Forgent compiles agent skill specs (YAML) into framework-native formats.

Agents are compositions of Skill Behaviors — reusable behavioral units
with 5 facets: Context, Strategy, Guardrails, Observability, Security.

Examples:
  forgent doctor agent.md              # analyze an agent definition
  forgent build --target claude        # compile for Claude Code
  forgent targets                      # list registered targets and parsers
  forgent completion bash > ~/.forgent-completion.bash`,
	Version: version,
}

func Execute() error {
	return rootCmd.Execute()
}
