package cmd

import (
	"fmt"
	"os"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var (
	modelFlag string
	dataDir   string
)

var rootCmd = &cobra.Command{
	Use:   "routerforge",
	Short: "AI Software Company Operating System",
	Long: `RouterForge is an AI Software Company Operating System.
It creates and manages dynamic multi-agent teams to build software projects.
Inspired by OpenCode, Claude Code, Gemini CLI, and Devin.`,
	Run: func(cmd *cobra.Command, args []string) {
		pterm.DefaultBigText.WithLetters(pterm.NewLettersFromString("RouterForge")).Render()
		pterm.Println()
		pterm.Info.Println("Use 'routerforge init' to start a new project")
		pterm.Info.Println("Use 'routerforge plan' to start the planning phase")
		pterm.Info.Println("Use 'routerforge build' to execute the build pipeline")
		pterm.Info.Println("Use 'routerforge --help' for all commands")
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&modelFlag, "model", "m", "big-pickle", "AI model to use for agents")
	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", "", "Data directory for RouterForge state")
}
