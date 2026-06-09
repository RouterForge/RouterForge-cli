package cmd

import (
	"os"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/routerforge/cli/internal/orchestrator"
	"github.com/routerforge/cli/internal/storage"
	"github.com/spf13/cobra"
)

var lifecycleCmd = &cobra.Command{
	Use:   "lifecycle",
	Short: "Manage development lifecycle phases",
	Long:  `Manage the four-phase development lifecycle: Demo → Prototype → MVP → Production.`,
}

var lifecycleStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current lifecycle phase and gate status",
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

		currentPhase := orchestrator.LifecycleDemo
		pterm.DefaultSection.Printfln("Development Lifecycle")

		phases := []struct {
			phase       orchestrator.LifecyclePhase
			name        string
			deliverable string
		}{
			{orchestrator.LifecycleDemo, "Demo", "Product concept, initial user flow, demo artifacts"},
			{orchestrator.LifecyclePrototype, "Prototype", "Working prototype, technical architecture, core integrations"},
			{orchestrator.LifecycleMVP, "MVP", "Authentication, persistence, monitoring, core functionality, test coverage"},
			{orchestrator.LifecycleProduction, "Production", "Security reviews, performance testing, observability, deployment readiness"},
		}

		for _, p := range phases {
			marker := "  "
			if p.phase == currentPhase {
				marker = "▶ "
				pterm.Printfln("%s %s (CURRENT)", marker, p.name)
				pterm.Printfln("    Deliverable: %s", p.deliverable)
			} else if p.phase < currentPhase {
				marker = "✅"
				pterm.Printfln("%s %s", marker, p.name)
			} else {
				pterm.Printfln("%s %s", marker, p.name)
				pterm.Printfln("    Deliverable: %s", p.deliverable)
			}
		}

		pterm.DefaultSection.Println("Review Gates")
		gates := hm.ReviewGates().AllGates()
		for _, g := range gates {
			status := "❌"
			if g.Passed {
				status = "✅"
			}
			required := ""
			if g.Required {
				required = " (required)"
			}
			pterm.Printfln("  %s %s%s", status, g.Name, required)
		}
	},
}

var lifecycleAdvanceCmd = &cobra.Command{
	Use:   "advance",
	Short: "Advance to the next lifecycle phase",
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

		pterm.Info.Printfln("Current phase: %s", hm.LifecycleStr())

		if !hm.CanAdvanceLifecycle() {
			pterm.Warning.Println("Cannot advance: not all required review gates have passed.")
			pterm.Info.Println("Use 'routerforge gate approve <type>' to approve gates.")
			failed := hm.ReviewGates().GetFailedRequired()
			for _, g := range failed {
				pterm.Printfln("  ❌ %s (required)", g.Name)
			}
			return
		}

		if err := hm.AdvanceLifecycle(); err != nil {
			pterm.Error.Printfln("Failed to advance lifecycle: %v", err)
			return
		}

		pterm.Success.Printfln("Advanced to %s phase!", hm.LifecycleStr())
		pterm.Info.Printfln("Deliverable: %s", hm.LifecycleStr())
	},
}

func init() {
	rootCmd.AddCommand(lifecycleCmd)
	lifecycleCmd.AddCommand(lifecycleStatusCmd)
	lifecycleCmd.AddCommand(lifecycleAdvanceCmd)
}
