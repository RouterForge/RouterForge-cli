package cmd

import (
	"os"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/routerforge/cli/internal/storage"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new RouterForge project",
	Long:  `Initialize a new RouterForge project in the current directory.`,
	Run: func(cmd *cobra.Command, args []string) {
		pterm.DefaultSection.Printfln("Initializing RouterForge project")

		projectDir, _ := os.Getwd()
		routerDir := filepath.Join(projectDir, ".routerforge")

		if _, err := os.Stat(routerDir); !os.IsNotExist(err) {
			pterm.Warning.Println("RouterForge already initialized in this directory")
			return
		}

		if err := os.MkdirAll(routerDir, 0755); err != nil {
			pterm.Error.Printfln("Failed to create .routerforge directory: %v", err)
			return
		}

		cfg := storage.DefaultConfig()
		cfg.ProjectDir = projectDir
		cfg.Model = modelFlag

		configPath := filepath.Join(routerDir, "config.json")
		if err := cfg.Save(configPath); err != nil {
			pterm.Error.Printfln("Failed to save config: %v", err)
			return
		}

		subdirs := []string{"projects", "sessions", "logs", "screenshots", "artifacts"}
		for _, d := range subdirs {
			if err := os.MkdirAll(filepath.Join(routerDir, d), 0755); err != nil {
				pterm.Warning.Printfln("Failed to create %s: %v", d, err)
			}
		}

		dbPath := filepath.Join(routerDir, "routerforge.db")
		store, err := storage.NewStore(dbPath)
		if err != nil {
			pterm.Error.Printfln("Failed to initialize database: %v", err)
			return
		}
		defer store.Close()

		pterm.Success.Printfln("RouterForge project initialized at %s", projectDir)
		pterm.Info.Printfln("  Model: %s", cfg.Model)
		pterm.Info.Printfln("  Data: %s", routerDir)
		pterm.Println()
		pterm.Info.Println("Run 'routerforge plan' to start planning your project")
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	initCmd.Flags().StringVarP(&modelFlag, "model", "m", "big-pickle", "Default AI model for agents")
}
