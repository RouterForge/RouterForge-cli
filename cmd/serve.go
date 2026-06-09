package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve [port]",
	Short: "Serve generated artifacts and traces",
	Long:  `Start an HTTP server that serves all generated artifacts, trace logs, and plan documents with a built-in dashboard.`,
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		port := "8080"
		if len(args) > 0 {
			port = args[0]
		}

		artifactDir := filepath.Join(".", ".routerforge", "artifacts")
		if _, err := os.Stat(artifactDir); os.IsNotExist(err) {
			pterm.Warning.Printf("No artifacts directory found at %s\n", artifactDir)
			pterm.Info.Println("Run 'routerforge build' first to generate artifacts")
			return
		}

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				serveDashboard(w, r, artifactDir)
				return
			}
			http.FileServer(http.Dir(artifactDir)).ServeHTTP(w, r)
		})

		go func() {
			time.Sleep(500 * time.Millisecond)
			resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%s/", port))
			if err != nil {
				pterm.Error.Printf("Self-test failed: %v\n", err)
				return
			}
			defer resp.Body.Close()
			pterm.Success.Printf("Self-test: HTTP %d\n", resp.StatusCode)
		}()

		pterm.Success.Printf("Serving artifacts on http://127.0.0.1:%s\n", port)
		pterm.Info.Printf("Artifact dir: %s\n", artifactDir)

		if err := http.ListenAndServe("127.0.0.1:"+port, nil); err != nil {
			pterm.Error.Printf("Server failed: %v\n", err)
		}
	},
}

func serveDashboard(w http.ResponseWriter, r *http.Request, dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		http.Error(w, "Cannot read artifacts", 500)
		return
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	var b strings.Builder
	b.WriteString(`<!DOCTYPE html><html><head><meta charset="utf-8"><title>RouterForge — Artifacts</title>
<style>body{font-family:system-ui,sans-serif;max-width:800px;margin:40px auto;padding:0 20px}
h1{color:#7C3AED}
.file{border:1px solid #e5e7eb;border-radius:8px;padding:12px 16px;margin:8px 0;display:flex;justify-content:space-between;align-items:center}
.file a{color:#3B82F6;text-decoration:none;font-weight:500}
.file a:hover{text-decoration:underline}
.size{color:#6B7280;font-size:14px}
.section{margin-top:32px}
h2{color:#374151;border-bottom:2px solid #e5e7eb;padding-bottom:8px}
</style></head><body>`)
	b.WriteString(fmt.Sprintf(`<h1>🔨 RouterForge Artifacts</h1>
<p>%d artifacts from the build pipeline</p>`, len(files)))

	b.WriteString(`<div class="section"><h2>📄 Documents</h2>`)
	for _, f := range files {
		if f == "trace.jsonl" {
			continue
		}
		info, _ := os.Stat(filepath.Join(dir, f))
		size := ""
		if info != nil {
			size = fmt.Sprintf("%d KB", info.Size()/1024+1)
		}
		b.WriteString(fmt.Sprintf(`<div class="file"><a href="/%s">%s</a><span class="size">%s</span></div>`, f, f, size))
	}
	b.WriteString("</div>")

	if hasFile(dir, "trace.jsonl") {
		b.WriteString(`<div class="section"><h2>📊 Execution Trace</h2><pre style="background:#f9fafb;border-radius:8px;padding:16px;overflow-x:auto;font-size:13px">`)
		data, _ := os.ReadFile(filepath.Join(dir, "trace.jsonl"))
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		start := 0
		if len(lines) > 50 {
			start = len(lines) - 50
			b.WriteString(fmt.Sprintf("<i>... %d earlier entries omitted</i>\n", start))
		}
		for _, line := range lines[start:] {
			var entry map[string]interface{}
			json.Unmarshal([]byte(line), &entry)
			if t, ok := entry["timestamp"].(string); ok && len(t) > 19 {
				entry["timestamp"] = t[11:19]
			}
			pretty, _ := json.Marshal(entry)
			b.WriteString(string(pretty) + "\n")
		}
		b.WriteString("</pre></div>")
	}

	b.WriteString(`</body></html>`)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(b.String()))
}

func hasFile(dir, name string) bool {
	_, err := os.Stat(filepath.Join(dir, name))
	return err == nil
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
