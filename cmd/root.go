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
	modelFlag string
	dataDir   string
)

var rootCmd = &cobra.Command{
	Use:   "routerforge",
	Short: "AI multi-agent operating system",
	Long: `RouterForge is an AI multi-agent operating system.
It creates and manages dynamic agent teams to build software projects.
Just type 'routerforge' and start chatting with your Head Manager.`,
	Run: func(cmd *cobra.Command, args []string) {
		projectDir, _ := os.Getwd()
		routerDir := filepath.Join(projectDir, ".routerforge")

		if _, err := os.Stat(routerDir); os.IsNotExist(err) {
			pterm.Info.Println("First run — initializing RouterForge project...")
			if err := os.MkdirAll(routerDir, 0755); err != nil {
				pterm.Error.Printfln("Failed to create .routerforge directory: %v", err)
				return
			}
			cfg := storage.DefaultConfig()
			if modelFlag != "big-pickle" {
				cfg.Model = modelFlag
			}
			if err := cfg.Save(filepath.Join(routerDir, "config.json")); err != nil {
				pterm.Warning.Printfln("Could not save config: %v", err)
			}
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

		hm := orchestrator.NewHeadManager(cfg.Model)
		hm.SetMemory(memory.NewStore())
		hm.SetTracePath(filepath.Join(routerDir, "artifacts", "trace.jsonl"))
		hm.SetProjectDir(projectDir)
		hm.SetConversationsDir(filepath.Join(routerDir, "artifacts", "conversations"))

		p := tui.NewProgram(hm)
		if err := p.Run(); err != nil {
			pterm.Error.Printfln("Error: %v", err)
		}
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&modelFlag, "model", "m", "big-pickle", "AI model to use for agents")
	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", "", "Data directory for RouterForge state")
}
