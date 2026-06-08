package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/routerforge/cli/internal/memory"
	"github.com/routerforge/cli/internal/orchestrator"
	"github.com/routerforge/cli/internal/storage"
	"github.com/routerforge/cli/internal/tui"
	"github.com/spf13/cobra"
)

var (
	profileFlag string
	tuiFlag     bool
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Execute the build pipeline",
	Long:  `Execute the full build pipeline: Execute agents and Review results.`,
	Run: func(cmd *cobra.Command, args []string) {
		pterm.DefaultSection.Printfln("RouterForge Build Pipeline")

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

		store, err := storage.NewStore(filepath.Join(routerDir, "routerforge.db"))
		if err == nil {
			defer store.Close()
		}

		teams := cfg.Pipeline.Teams
		profile := profileFlag
		if profile != "" && cfg.Pipeline.Profiles[profile] != nil {
			pterm.Info.Printfln("Using profile: %s", profile)
			teams = cfg.Pipeline.Profiles[profile].Teams
		}
		if profile == "" {
			profile = cfg.Pipeline.Profile
		}

		hm := orchestrator.NewHeadManager(cfg.Model)
		hm.SetMemory(memory.NewStore())

		if tuiFlag {
			p := tui.NewProgram(hm)
			if err := p.Run(); err != nil {
				pterm.Error.Printfln("TUI error: %v", err)
			}
			return
		}

		hm.AttachConsoleLogger()
		pterm.Info.Printfln("Building with model: %s", cfg.Model)
		pterm.Info.Printfln("Teams: %v", teams)

		pterm.Info.Println("Creating teams for execution...")

		teamMap := make(map[string]bool)
		for _, t := range teams {
			hm.CreateTeam(t)
			teamMap[t] = true
		}

		if teamMap["backend"] {
			backendTM := hm.Teams()["team-backend"]
			if backendTM != nil {
				backendTM.CreateMicroAgent("api_designer", []orchestrator.TaskDef{
					{Description: "Design REST API endpoints", Priority: "high"},
					{Description: "Define request/response schemas", Priority: "high"},
					{Description: "Document API", Priority: "medium"},
				}, cfg.Model)
				backendTM.CreateMicroAgent("db_schema_designer", []orchestrator.TaskDef{
					{Description: "Design database schema", Priority: "high"},
					{Description: "Define relationships", Priority: "high"},
				}, cfg.Model)
			}
		}

		if teamMap["frontend"] {
			frontendTM := hm.Teams()["team-frontend"]
			if frontendTM != nil {
				frontendTM.CreateMicroAgent("component_builder", []orchestrator.TaskDef{
					{Description: "Generate a complete SaaS landing page in a single HTML file", Priority: "high"},
					{Description: "Style the landing page with responsive dark theme CSS", Priority: "high"},
					{Description: "Add interactive JavaScript (smooth scroll, mobile menu, form handling)", Priority: "medium"},
				}, cfg.Model)
			}
		}

		if teamMap["security"] {
			securityTM := hm.Teams()["team-security"]
			if securityTM != nil {
				securityTM.CreateMicroAgent("security_reviewer", []orchestrator.TaskDef{
					{Description: "Review code for security vulnerabilities", Priority: "high"},
					{Description: "Check for OWASP Top 10 violations", Priority: "high"},
				}, cfg.Model)
			}
		}

		if teamMap["qa"] {
			qaTM := hm.Teams()["team-qa"]
			if qaTM != nil {
				qaTM.CreateMicroAgent("unit_test_writer", []orchestrator.TaskDef{
					{Description: "Write unit tests for core functions", Priority: "high"},
					{Description: "Write integration tests", Priority: "high"},
				}, cfg.Model)
			}
		}

		hm.SendMessage("head_manager", "all", "model_assignment", fmt.Sprintf("All agents using model: %s", cfg.Model))

		if err := hm.Execute(); err != nil {
			pterm.Error.Printfln("Execute phase failed: %v", err)
			return
		}

		if err := hm.Review(); err != nil {
			pterm.Error.Printfln("Review phase failed: %v", err)
			return
		}

		pterm.DefaultSection.Printfln("✅ Build Complete")
	},
}

func init() {
	buildCmd.Flags().StringVarP(&profileFlag, "profile", "p", "", "Pipeline profile (quick, full)")
	buildCmd.Flags().BoolVar(&tuiFlag, "tui", false, "Use terminal UI mode")
	rootCmd.AddCommand(buildCmd)
}
