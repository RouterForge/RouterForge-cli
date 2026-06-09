package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

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
		hm.SetTracePath(filepath.Join(routerDir, "artifacts", "trace.jsonl"))

		if tuiFlag {
			p := tui.NewProgram(hm)
			if err := p.Run(); err != nil {
				pterm.Error.Printfln("TUI error: %v", err)
			}
			return
		}

		hm.AttachConsoleLogger()
		pterm.Info.Printfln("Building with model: %s", cfg.Model)
		pterm.Info.Printfln("Profile: %s (teams: %v)", profile, teams)

		pterm.Info.Println("Designing team structure from requirements...")
		if err := hm.Design(); err != nil {
			pterm.Warning.Printfln("Design fallback: %v", err)
			pterm.Info.Println("Creating teams from profile config...")
			for _, t := range teams {
				hm.CreateTeam(t, "")
			}
		}

		pterm.Info.Println("Starting execution...")
		if err := hm.Execute(); err != nil {
			pterm.Error.Printfln("Execute phase failed: %v", err)
			return
		}

		if err := hm.Review(); err != nil {
			pterm.Error.Printfln("Review phase failed: %v", err)
			return
		}

		saveArtifactSummary(hm, routerDir)
		pterm.DefaultSection.Printfln("✅ Build Complete")
		pterm.Info.Printfln("📄 Trace: %s", filepath.Join(routerDir, "artifacts", "trace.jsonl"))
		pterm.Info.Printfln("📄 Plan: %s", filepath.Join(routerDir, "artifacts", "plan.json"))
		pterm.Info.Printfln("🌐 Dashboard: routerforge serve")
	},
}

func saveArtifactSummary(hm *orchestrator.HeadManager, routerDir string) {
	artifactsDir := filepath.Join(routerDir, "artifacts")
	summary := map[string]interface{}{
		"project":     hm.Project().Name,
		"model":       hm.Model(),
		"teams":       len(hm.Teams()),
		"decisions":   len(hm.Decisions()),
		"plan":        hm.Plan() != nil,
		"phases":      hm.StateHistory(),
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}
	data, _ := json.MarshalIndent(summary, "", "  ")
	os.WriteFile(filepath.Join(artifactsDir, "summary.json"), data, 0644)
}

func init() {
	buildCmd.Flags().StringVarP(&profileFlag, "profile", "p", "", "Pipeline profile (quick, full)")
	buildCmd.Flags().BoolVar(&tuiFlag, "tui", false, "Use terminal UI mode")
	rootCmd.AddCommand(buildCmd)
}
