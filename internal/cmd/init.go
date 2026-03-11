package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/fatih/color"
	"github.com/mirandaguillaume/forgent/internal/templates"
	"github.com/spf13/cobra"
)

// InitResult holds the result of an init operation.
type InitResult struct {
	AlreadyInitialized bool
	Path               string
}

// InitProject initializes a Forgent project in the given directory.
func InitProject(targetDir string) InitResult {
	configPath := filepath.Join(targetDir, "forgent.yaml")

	if _, err := os.Stat(configPath); err == nil {
		return InitResult{AlreadyInitialized: true, Path: targetDir}
	}

	os.MkdirAll(filepath.Join(targetDir, "skills"), 0755)
	os.MkdirAll(filepath.Join(targetDir, "agents"), 0755)

	config := "# Forgent Project Configuration\nversion: \"0.1.0\"\nskills_dir: skills\nagents_dir: agents\n"
	os.WriteFile(configPath, []byte(config), 0644)

	tmpl, err := templates.SkillTemplate()
	if err == nil {
		os.WriteFile(filepath.Join(targetDir, "skills", "example.skill.yaml"), tmpl, 0644)
	}

	return InitResult{AlreadyInitialized: false, Path: targetDir}
}

func init() {
	initCmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Initialize a Forgent project in the current directory",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}
			result := InitProject(path)
			if result.AlreadyInitialized {
				fmt.Println(color.YellowString("Forgent project already initialized."))
			} else {
				fmt.Println(color.GreenString("Forgent project initialized at"), result.Path)
				fmt.Println("  Created: forgent.yaml, skills/, agents/")
			}
		},
	}
	rootCmd.AddCommand(initCmd)
}
