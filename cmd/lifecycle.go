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
	Short: "Manage the unified project lifecycle",
	Long:  `Manage the unified lifecycle: Runtime Flow (Idle→Understand→Design→Execute→Repair→Review) and Project Maturity (Demo→Prototype→MVP→Production).`,
}

var lifecycleStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show lifecycle status (runtime flow + project maturity)",
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

		// --- Runtime Flow ---
		pterm.DefaultSection.Printfln("Runtime Flow")
		flowPhases := []struct {
			phase orchestrator.RuntimePhase
			name  string
		}{
			{orchestrator.RuntimeIdle, "Idle"},
			{orchestrator.RuntimeUnderstand, "Understand"},
			{orchestrator.RuntimeDesign, "Design"},
			{orchestrator.RuntimeExecute, "Execute"},
			{orchestrator.RuntimeRepair, "Repair"},
			{orchestrator.RuntimeReview, "Review"},
		}

		currentFlow := hm.State()
		for _, p := range flowPhases {
			marker := "  "
			if p.phase == currentFlow {
				marker = "▶ "
				pterm.Printfln("%s %s (CURRENT)", marker, p.name)
			} else if p.phase < currentFlow {
				pterm.Printfln("  ✅ %s", p.name)
			} else {
				pterm.Printfln("  %s %s", marker, p.name)
			}
		}

		// --- Project Maturity ---
		pterm.DefaultSection.Printfln("Project Maturity")
		maturityPhases := []struct {
			phase       orchestrator.MaturityStage
			name        string
			deliverable string
		}{
			{orchestrator.MaturityDemo, "Demo", "Product concept, initial user flow, demo artifacts"},
			{orchestrator.MaturityPrototype, "Prototype", "Working prototype, technical architecture, core integrations"},
			{orchestrator.MaturityMVP, "MVP", "Authentication, persistence, monitoring, core functionality, test coverage"},
			{orchestrator.MaturityProduction, "Production", "Security reviews, performance testing, observability, deployment readiness"},
		}

		currentMaturity := hm.LifecyclePhase()
		for _, p := range maturityPhases {
			marker := "  "
			if p.phase == currentMaturity {
				marker = "▶ "
				pterm.Printfln("%s %s (CURRENT)", marker, p.name)
				pterm.Printfln("    Deliverable: %s", p.deliverable)
			} else if p.phase < currentMaturity {
				pterm.Printfln("  ✅ %s", p.name)
			} else {
				pterm.Printfln("  %s %s", marker, p.name)
				pterm.Printfln("    Deliverable: %s", p.deliverable)
			}
		}

		// --- Review Gates ---
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

		pterm.Info.Printfln("Gates: %s", hm.ReviewGates().Summary())
	},
}

var lifecycleAdvanceCmd = &cobra.Command{
	Use:   "advance",
	Short: "Advance to the next project maturity stage",
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

		pterm.Info.Printfln("Current maturity stage: %s", hm.LifecycleStr())

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
			pterm.Error.Printfln("Failed to advance maturity: %v", err)
			return
		}

		pterm.Success.Printfln("Project advanced to %s stage!", hm.LifecycleStr())
		pterm.Info.Printfln("Deliverable: %s", hm.LifecycleStr())
	},
}

func init() {
	rootCmd.AddCommand(lifecycleCmd)
	lifecycleCmd.AddCommand(lifecycleStatusCmd)
	lifecycleCmd.AddCommand(lifecycleAdvanceCmd)
}
