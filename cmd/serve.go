package cmd

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve [port]",
	Short: "Serve generated artifacts",
	Long:  `Start an HTTP server to serve the generated landing page and test it.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		port := "8080"
		if len(args) > 0 {
			port = args[0]
		}

		artifactPath := filepath.Join(".", ".routerforge", "artifacts", "index.html")
		if _, err := os.Stat(artifactPath); os.IsNotExist(err) {
			pterm.Warning.Printf("No generated index.html found at %s\n", artifactPath)
			pterm.Info.Println("Run 'routerforge build' first to generate artifacts")
			return
		}

		pterm.Success.Printf("Serving generated page on http://127.0.0.1:%s\n", port)
		pterm.Info.Printf("Artifact: %s\n", artifactPath)

		fs := http.FileServer(http.Dir(filepath.Dir(artifactPath)))
		http.Handle("/", fs)

		// Self-test in a goroutine
		go func() {
			// Small delay to let server start
			resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/", port))
			if err != nil {
				pterm.Error.Printf("Self-test failed: %v\n", err)
				return
			}
			defer resp.Body.Close()
			pterm.Success.Printf("Self-test: HTTP %d\n", resp.StatusCode)
		}()

		if err := http.ListenAndServe("127.0.0.1:"+port, nil); err != nil {
			pterm.Error.Printf("Server failed: %v\n", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
