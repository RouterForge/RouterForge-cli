package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pterm/pterm"
	"github.com/routerforge/cli/internal/repo"
	"github.com/spf13/cobra"
)

var analyzeCmd = &cobra.Command{
	Use:   "analyze",
	Short: "Repository Intelligence Engine",
	Long:  `Analyze repositories, detect patterns, build capability matrices, and get integration recommendations.`,
}

var cloneCmd = &cobra.Command{
	Use:   "clone <url> [name]",
	Short: "Clone a repository for analysis",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		url := args[0]
		name := filepath.Base(url)
		if len(args) > 1 {
			name = args[1]
		}
		name = strings.TrimSuffix(name, ".git")

		baseDir := filepath.Join(".", ".routerforge", "repos")
		mgr := repo.NewManager(baseDir)

		pterm.Info.Printfln("Cloning %s as %s...", url, name)
		info, err := mgr.Clone(url, name)
		if err != nil {
			pterm.Error.Printfln("Clone failed: %v", err)
			return
		}
		pterm.Success.Printfln("Cloned %s at commit %s", info.Name, info.CommitSHA)
	},
}

var detectCmd = &cobra.Command{
	Use:   "detect <path>",
	Short: "Detect language and patterns in a codebase",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]

		sa := &repo.SourceAnalyzer{}
		lang := sa.DetectLanguage(path)
		pterm.Info.Printfln("Detected language: %s", lang)

		pd := &repo.PatternDetector{}
		patterns := pd.Detect(path)
		if len(patterns) == 0 {
			pterm.Warning.Println("No patterns detected")
			return
		}
		pterm.DefaultSection.Printfln("Detected %d patterns", len(patterns))
		for _, p := range patterns {
			pterm.Printfln("  • %s (confidence: %.0f%%)", p.Name, p.Confidence*100)
			for _, ind := range p.Indicators {
				pterm.Printfln("    └ %s", ind)
			}
		}
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List cloned repositories",
	Run: func(cmd *cobra.Command, args []string) {
		baseDir := filepath.Join(".", ".routerforge", "repos")
		mgr := repo.NewManager(baseDir)
		repos, err := mgr.List()
		if err != nil {
			if os.IsNotExist(err) {
				pterm.Warning.Println("No repos directory found. Clone a repo first.")
				return
			}
			pterm.Error.Printfln("Error: %v", err)
			return
		}
		if len(repos) == 0 {
			pterm.Warning.Println("No repositories cloned yet")
			return
		}
		pterm.DefaultSection.Printfln("Cloned repositories (%d)", len(repos))
		for _, r := range repos {
			info, err := mgr.Info(r)
			if err != nil {
				pterm.Printfln("  • %s (error: %v)", r, err)
				continue
			}
			pterm.Printfln("  • %s @ %s (%s)", info.Name, info.Branch, info.CommitSHA[:8])
		}
	},
}

var matrixCmd = &cobra.Command{
	Use:   "matrix",
	Short: "Build a feature matrix from cloned repos",
	Run: func(cmd *cobra.Command, args []string) {
		baseDir := filepath.Join(".", ".routerforge", "repos")
		mgr := repo.NewManager(baseDir)
		repos, err := mgr.List()
		if err != nil || len(repos) == 0 {
			pterm.Warning.Println("No repos cloned. Use 'routerforge analyze clone' first.")
			return
		}

		fm := repo.NewFeatureMatrix()
		for _, name := range repos {
			fm.AddRepo(name)
		}

		pd := &repo.PatternDetector{}
		for _, name := range repos {
			r, err := mgr.Info(name)
			if err != nil {
				continue
			}
			patterns := pd.Detect(r.LocalPath)
			for _, p := range patterns {
				fm.SetFeature(name, p.Name, true)
			}
		}

		pterm.DefaultSection.Println("Feature Matrix")
		fmt.Println(fm.Markdown())
	},
}

var recommendCmd = &cobra.Command{
	Use:   "recommend <features...>",
	Short: "Get integration recommendations based on needed features",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		g := repo.NewCapabilityGraph()
		g.Add("containerized", "Runs in Docker containers", nil)
		g.Add("ci_cd", "Has CI/CD pipelines", nil)
		g.Add("language_go", "Written in Go", nil)
		g.Add("language_python", "Written in Python", nil)
		g.Add("language_ts", "Written in TypeScript", nil)
		g.Add("web_framework", "Has web framework", []string{"language_go", "language_python", "language_ts"})
		g.Add("api_server", "Has API server", []string{"web_framework"})
		g.Add("database", "Has database support", []string{"api_server"})
		g.Add("auth_system", "Has authentication", []string{"api_server"})
		g.Add("cli_tool", "Is a CLI application", []string{"language_go", "language_python", "language_ts"})
		g.Add("plugin_system", "Has plugin/extension system", []string{"cli_tool"})

		r := repo.NewRecommender(g)
		order := r.Recommend(args)

		pterm.DefaultSection.Printfln("Integration order for: %s", strings.Join(args, ", "))
		for i, item := range order {
			pterm.Printfln("  %d. %s", i+1, item)
		}

		if b, err := json.MarshalIndent(order, "", "  "); err == nil {
			pterm.Info.Println("JSON: " + string(b))
		}
	},
}

func init() {
	analyzeCmd.AddCommand(cloneCmd)
	analyzeCmd.AddCommand(detectCmd)
	analyzeCmd.AddCommand(listCmd)
	analyzeCmd.AddCommand(matrixCmd)
	analyzeCmd.AddCommand(recommendCmd)
	rootCmd.AddCommand(analyzeCmd)
}
