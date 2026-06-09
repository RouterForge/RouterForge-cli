package cmd

import (
	"github.com/pterm/pterm"
	"github.com/routerforge/cli/internal/engine"
	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Agent management commands",
}

var agentSpawnCmd = &cobra.Command{
	Use:   "spawn <role> <task>",
	Short: "Spawn a dynamic sub-agent",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		role := args[0]
		task := args[1]

		spawner := engine.NewAgentSpawner(modelFlag)
		agent, err := spawner.Spawn(role, task, "cli")
		if err != nil {
			pterm.Error.Printfln("Spawn failed: %v", err)
			return
		}
		pterm.Success.Printfln("Spawned agent %s (%s)", agent.Role, agent.ID)
		pterm.Info.Println(agent.Result)
	},
}

func init() {
	rootCmd.AddCommand(agentCmd)
	agentCmd.AddCommand(agentSpawnCmd)
}
