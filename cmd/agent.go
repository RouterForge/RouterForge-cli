package cmd

import (
	"github.com/pterm/pterm"
	"github.com/routerforge/cli/internal/agent"
	"github.com/spf13/cobra"
)

var agentCmd = &cobra.Command{
	Use:   "agent",
	Short: "Agent management commands",
}

var agentListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available agent templates",
	Run: func(cmd *cobra.Command, args []string) {
		pterm.DefaultSection.Printfln("Available Agent Templates")
		roles := agent.ListRoles()
		if len(roles) == 0 {
			pterm.Info.Println("No agent templates found")
			return
		}
		for _, role := range roles {
			t, _ := agent.GetTemplate(role)
			pterm.Println(" • " + role)
			pterm.Printfln("    Tools: %v", t.Tools)
			pterm.Printfln("    Scope: %v", t.MemoryScope)
		}
	},
}

func init() {
	rootCmd.AddCommand(agentCmd)
	agentCmd.AddCommand(agentListCmd)
}
