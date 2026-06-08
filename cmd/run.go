package cmd

import (
	"fmt"

	"github.com/pterm/pterm"
	"github.com/routerforge/cli/internal/engine"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run a shell command",
	Long:  `Execute a shell command in the project directory.`,
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		command := args[0]
		if len(args) > 1 {
			for _, a := range args[1:] {
				command += " " + a
			}
		}

		pterm.Info.Printfln("Running: %s", command)

		runner := engine.NewShellRunner(".")
		result, err := runner.Run(cmd.Context(), command)
		if err != nil {
			pterm.Error.Printfln("Command failed: %v", err)
			return
		}

		if result.Stdout != "" {
			fmt.Println(result.Stdout)
		}
		if result.Stderr != "" {
			pterm.Warning.Printfln("Stderr: %s", result.Stderr)
		}
		pterm.Success.Printfln("Exit code: %d (%v)", result.ExitCode, result.Duration)
	},
}

func init() {
	rootCmd.AddCommand(runCmd)
}
