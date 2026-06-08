package cmd

import (
	"os"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/routerforge/cli/internal/orchestrator"
	"github.com/routerforge/cli/internal/storage"
	"github.com/spf13/cobra"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Start the planning phase",
	Long:  `Start the interactive planning phase to understand the project and design the architecture.`,
	Run: func(cmd *cobra.Command, args []string) {
		pterm.DefaultSection.Printfln("RouterForge Planning Phase")

		projectDir, _ := os.Getwd()
		routerDir := filepath.Join(projectDir, ".routerforge")

		if _, err := os.Stat(routerDir); os.IsNotExist(err) {
			pterm.Warning.Println("No RouterForge project found. Run 'routerforge init' first.")
			return
		}

		configPath := filepath.Join(routerDir, "config.json")
		cfg, err := storage.LoadConfig(configPath)
		if err != nil {
			pterm.Warning.Printfln("Using default config: %v", err)
			cfg = storage.DefaultConfig()
		}

		if modelFlag != "big-pickle" {
			cfg.Model = modelFlag
		}

		hm := orchestrator.NewHeadManager(cfg.Model)

		pterm.Info.Printfln("Using model: %s", cfg.Model)
		pterm.Println()

		if err := hm.Understand(); err != nil {
			pterm.Error.Printfln("Understanding phase failed: %v", err)
			return
		}

		if err := hm.Design(); err != nil {
			pterm.Error.Printfln("Design phase failed: %v", err)
			return
		}

		store, err := storage.NewStore(filepath.Join(routerDir, "routerforge.db"))
		if err == nil {
			store.Put("project", hm.Project().ID, hm.Project())
			store.Close()
		}

		pterm.Success.Printfln("Planning complete. Run 'routerforge build' to execute.")
	},
}

func init() {
	rootCmd.AddCommand(planCmd)
}
