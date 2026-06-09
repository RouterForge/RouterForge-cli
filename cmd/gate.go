package cmd

import (
	"os"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/routerforge/cli/internal/orchestrator"
	"github.com/routerforge/cli/internal/storage"
	"github.com/spf13/cobra"
)

var gateCmd = &cobra.Command{
	Use:   "gate",
	Short: "Manage review gates and approvals",
}

var gateApproveCmd = &cobra.Command{
	Use:   "approve <gate-type> [notes]",
	Short: "Approve a review gate",
	Long: `Approve a review gate to pass governance requirements.
Gate types: architecture_review, security_review, testing_requirement, documentation, approval_workflow`,
	Args: cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		projectDir, _ := os.Getwd()
		routerDir := filepath.Join(projectDir, ".routerforge")

		if _, err := os.Stat(routerDir); os.IsNotExist(err) {
			pterm.Warning.Println("No RouterForge project found. Run 'routerforge init' first.")
			return
		}

		gateType := orchestrator.GateType(args[0])
		notes := ""
		if len(args) > 1 {
			notes = args[1]
		}

		configPath := filepath.Join(routerDir, "config.json")
		cfg, err := storage.LoadConfig(configPath)
		if err != nil {
			cfg = storage.DefaultConfig()
		}

		hm := orchestrator.NewHeadManager(cfg.Model)
		hm.ApproveGate(gateType, "user", notes)
		pterm.Success.Printfln("Gate '%s' approved.", gateType)
		pterm.Info.Printfln("Gates summary: %s", hm.ReviewGates().Summary())
	},
}

var gateListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all review gates and their status",
	Run: func(cmd *cobra.Command, args []string) {
		projectDir, _ := os.Getwd()
		routerDir := filepath.Join(projectDir, ".routerforge")

		if _, err := os.Stat(routerDir); os.IsNotExist(err) {
			pterm.Warning.Println("No RouterForge project found. Run 'routerforge init' first.")
			return
		}

		configPath := filepath.Join(routerDir, "config.json")
		cfg, err := storage.LoadConfig(configPath)
		if err != nil {
			cfg = storage.DefaultConfig()
		}

		hm := orchestrator.NewHeadManager(cfg.Model)
		pterm.DefaultSection.Println("Review Gates")
		for _, g := range hm.ReviewGates().AllGates() {
			status := "❌"
			if g.Passed {
				status = "✅"
			}
			req := ""
			if g.Required {
				req = " (required)"
			}
			pterm.Printfln("  %s %s%s", status, g.Name, req)
			pterm.Printfln("    %s", g.Description)
			if g.Passed {
				pterm.Printfln("    Approved by: %s at %s", g.ApprovedBy, g.ApprovedAt)
			}
		}
	},
}

func init() {
	rootCmd.AddCommand(gateCmd)
	gateCmd.AddCommand(gateApproveCmd)
	gateCmd.AddCommand(gateListCmd)
}
