package cmd

import (
	"fmt"

	"github.com/mirandaguillaume/forgent/internal/doctor/parser"
	"github.com/mirandaguillaume/forgent/pkg/spec"
	"github.com/spf13/cobra"
)

func init() {
	targetsCmd := &cobra.Command{
		Use:   "targets",
		Short: "List registered build targets and format parsers",
		Run: func(cmd *cobra.Command, args []string) {
			out := cmd.OutOrStdout()
			fmt.Fprintln(out, "Build targets:")
			for _, t := range spec.Available() {
				fmt.Fprintf(out, "  %s\n", t)
			}
			fmt.Fprintln(out)
			fmt.Fprintln(out, "Format parsers:")
			for _, p := range parser.RegisteredFormats() {
				fmt.Fprintf(out, "  %s\n", p)
			}
		},
	}
	rootCmd.AddCommand(targetsCmd)
}
