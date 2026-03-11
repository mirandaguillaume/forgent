package cmd

import (
	"github.com/spf13/cobra"
)

var version = "dev"

var rootCmd = &cobra.Command{
	Use:     "forgent",
	Short:   "Forgent — Forge agents from composable skill specs",
	Version: version,
}

func Execute() error {
	return rootCmd.Execute()
}
