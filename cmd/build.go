package cmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/routerforge/cli/internal/memory"
	"github.com/routerforge/cli/internal/orchestrator"
	"github.com/routerforge/cli/internal/storage"
	"github.com/routerforge/cli/internal/tui"
	"github.com/routerforge/cli/pkg/models"
	"github.com/spf13/cobra"
)

var (
	profileFlag       string
	tuiFlag           bool
	repairRetriesFlag int
)

var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Execute the project lifecycle runtime flow",
	Long:  `Execute the runtime flow: Understand → Design → Execute → Repair → Review.`,
	Run: func(cmd *cobra.Command, args []string) {
		pterm.DefaultSection.Printfln("RouterForge Lifecycle Runtime Flow")

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

		teams := cfg.Lifecycle.Teams
		profile := profileFlag
		if profile != "" && cfg.Lifecycle.Profiles[profile] != nil {
			pterm.Info.Printfln("Using profile: %s", profile)
			teams = cfg.Lifecycle.Profiles[profile].Teams
		}
		if profile == "" {
			profile = cfg.Lifecycle.Profile
		}

		hm := orchestrator.NewHeadManager(cfg.Model)
		hm.SetMemory(memory.NewStore())
		hm.SetTracePath(filepath.Join(routerDir, "artifacts", "trace.jsonl"))
		hm.SetProjectDir(projectDir)
		hm.SetConversationsDir(filepath.Join(routerDir, "artifacts", "conversations"))

		if tuiFlag {
			pterm.Info.Println("Starting RouterForge multi-agent operating system...")
			p := tui.NewProgram(hm)
			if err := p.Run(); err != nil {
				pterm.Error.Printfln("TUI error: %v", err)
			}
			return
		}

		hm.AttachConsoleLogger()
		pterm.Info.Printfln("Building with model: %s", cfg.Model)
		pterm.Info.Printfln("Profile: %s (teams: %v)", profile, teams)

		// Try loading saved plan first
		planArtifact := filepath.Join(routerDir, "artifacts", "plan.json")
		loadedSaved := false
		if data, err := os.ReadFile(planArtifact); err == nil {
			var savedPlan models.Plan
			if err := json.Unmarshal(data, &savedPlan); err == nil && len(savedPlan.Teams) > 0 {
				pterm.Info.Printfln("Loaded saved plan from %s", planArtifact)
				hm.RestorePlan(&savedPlan)
				loadedSaved = true
			}
		}

		if !loadedSaved {
			pterm.Info.Println("Designing team structure from requirements...")
			if err := hm.Design(); err != nil {
				pterm.Warning.Printfln("Design fallback: %v", err)
				pterm.Info.Println("Creating teams from profile config...")
				for _, t := range teams {
					hm.CreateTeam(t, "")
				}
			}
		}

		pterm.Info.Println("Starting execution...")
		if err := hm.Execute(); err != nil {
			pterm.Warning.Printfln("Execute phase had errors: %v", err)
		}

		if err := hm.RepairUntilValid(repairRetriesFlag); err != nil {
			pterm.Error.Printfln("Repair loop failed: %v", err)
			return
		}

		if err := hm.Review(); err != nil {
			pterm.Warning.Printfln("Review phase: %v", err)
		}

		saveArtifactSummary(hm, routerDir)
		pterm.DefaultSection.Printfln("Lifecycle Runtime Flow Complete")
		pterm.Info.Printfln("Trace: %s", filepath.Join(routerDir, "artifacts", "trace.jsonl"))
		pterm.Info.Printfln("Plan: %s", filepath.Join(routerDir, "artifacts", "plan.json"))
		pterm.Info.Printfln("Dashboard: routerforge serve")
	},
}

func saveArtifactSummary(hm *orchestrator.HeadManager, routerDir string) {
	artifactsDir := filepath.Join(routerDir, "artifacts")
	os.MkdirAll(artifactsDir, 0755)

	summary := map[string]interface{}{
		"project":      hm.Project().Name,
		"model":        hm.Model(),
		"teams":        len(hm.Teams()),
		"decisions":    len(hm.Decisions()),
		"plan":         hm.Plan() != nil,
		"phases":       hm.StateHistory(),
		"generated_at": time.Now().UTC().Format(time.RFC3339),
	}
	data, _ := json.MarshalIndent(summary, "", "  ")
	os.WriteFile(filepath.Join(artifactsDir, "summary.json"), data, 0644)

	decData, _ := json.MarshalIndent(hm.Decisions(), "", "  ")
	os.WriteFile(filepath.Join(artifactsDir, "decisions.json"), decData, 0644)

	msgData, _ := json.MarshalIndent(hm.Messages(), "", "  ")
	os.WriteFile(filepath.Join(artifactsDir, "messages.json"), msgData, 0644)

	if hm.CostTracker() != nil {
		costData, _ := json.MarshalIndent(hm.CostTracker().Summary(), "", "  ")
		os.WriteFile(filepath.Join(artifactsDir, "cost.json"), costData, 0644)
		entryData, _ := json.MarshalIndent(hm.CostTracker().Entries(), "", "  ")
		os.WriteFile(filepath.Join(artifactsDir, "cost_entries.json"), entryData, 0644)
	}

	projectDir := hm.ProjectDir()
	if projectDir == "" {
		projectDir = "."
	}
	var files []string
	filepath.Walk(projectDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			if info != nil && info.IsDir() && (info.Name() == ".routerforge" || info.Name() == "node_modules" || info.Name() == ".git") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(info.Name(), ".log") {
			files = append(files, path)
		}
		return nil
	})
	fmData, _ := json.MarshalIndent(map[string]interface{}{
		"files":     files,
		"count":     len(files),
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}, "", "  ")
	os.WriteFile(filepath.Join(artifactsDir, "file_manifest.json"), fmData, 0644)
}

func init() {
	buildCmd.Flags().StringVarP(&profileFlag, "profile", "p", "", "Lifecycle profile (quick, full)")
	buildCmd.Flags().BoolVar(&tuiFlag, "tui", false, "RouterForge multi-agent OS interface (recommended)")
	buildCmd.Flags().IntVar(&repairRetriesFlag, "repair-retries", 2, "Maximum repair attempts after validation failure")
	rootCmd.AddCommand(buildCmd)
}
