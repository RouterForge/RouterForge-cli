package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect",
	Short: "Inspect all observable execution data from the last build",
	Long: `Displays all available evidence from the most recent build:
  - Plan & summary
  - Trace events
  - Validation results
  - Decisions
  - Cost & token usage
  - File manifest
  - LLM conversation logs
  - Agent messages`,
	RunE: func(cmd *cobra.Command, args []string) error {
		projectDir, _ := os.Getwd()
		artifactsDir := filepath.Join(projectDir, ".routerforge", "artifacts")

		if _, err := os.Stat(artifactsDir); os.IsNotExist(err) {
			return fmt.Errorf("no artifacts found at %s (run 'routerforge build' first)", artifactsDir)
		}

		fmt.Println("============================================================")
		fmt.Println("  RouterForge Inspect — Execution Evidence")
		fmt.Println("============================================================")
		fmt.Println()

		displayJSON := func(label, path string) {
			data, err := os.ReadFile(path)
			if err != nil {
				fmt.Printf("  %s: %v\n", label, err)
				return
			}
			var pretty interface{}
			if err := json.Unmarshal(data, &pretty); err != nil {
				fmt.Printf("  %s: (raw) %s\n", label, string(data))
				return
			}
			formatted, _ := json.MarshalIndent(pretty, "    ", "  ")
			fmt.Printf("  %s (%s):\n", label, path)
			fmt.Printf("    %s\n", string(formatted))
		}

		displayTextFile := func(label, path string, maxLines int) {
			data, err := os.ReadFile(path)
			if err != nil {
				fmt.Printf("  %s: %v\n", label, err)
				return
			}
			lines := strings.Split(string(data), "\n")
			if len(lines) > maxLines {
				lines = lines[:maxLines]
				fmt.Printf("  %s (%s): %d lines (showing first %d)\n", label, path, len(strings.Split(string(data), "\n")), maxLines)
			} else {
				fmt.Printf("  %s (%s): %d lines\n", label, path, len(lines))
			}
			for _, l := range lines {
				fmt.Printf("    %s\n", l)
			}
		}

		// 1. Summary
		summaryPath := filepath.Join(artifactsDir, "summary.json")
		if _, err := os.Stat(summaryPath); err == nil {
			fmt.Println("--- Build Summary ---")
			displayJSON("summary", summaryPath)
			fmt.Println()
		}

		// 2. Plan
		planPath := filepath.Join(artifactsDir, "plan.json")
		if _, err := os.Stat(planPath); err == nil {
			fmt.Println("--- Plan ---")
			displayJSON("plan", planPath)
			fmt.Println()
		}

		// 3. Trace
		tracePath := filepath.Join(artifactsDir, "trace.jsonl")
		if _, err := os.Stat(tracePath); err == nil {
			fmt.Println("--- Trace Events ---")
			displayTextFile("trace", tracePath, 50)
			fmt.Println()
		}

		// 4. Validation results
		validationFiles, _ := filepath.Glob(filepath.Join(artifactsDir, "validation-*.json"))
		if len(validationFiles) > 0 {
			fmt.Println("--- Validation Results ---")
			for _, vf := range validationFiles {
				displayJSON(filepath.Base(vf), vf)
				fmt.Println()
			}
			fmt.Println()
		}

		// 5. Decisions
		decPath := filepath.Join(artifactsDir, "decisions.json")
		if _, err := os.Stat(decPath); err == nil {
			fmt.Println("--- Decisions ---")
			displayJSON("decisions", decPath)
			fmt.Println()
		}

		// 6. Cost & token usage
		costPath := filepath.Join(artifactsDir, "cost.json")
		if _, err := os.Stat(costPath); err == nil {
			fmt.Println("--- Cost Summary ---")
			displayJSON("cost", costPath)
			fmt.Println()
		}
		costEntriesPath := filepath.Join(artifactsDir, "cost_entries.json")
		if _, err := os.Stat(costEntriesPath); err == nil {
			fmt.Println("--- Cost Entries (per LLM call) ---")
			displayJSON("cost_entries", costEntriesPath)
			fmt.Println()
		}

		// 7. File manifest
		fmPath := filepath.Join(artifactsDir, "file_manifest.json")
		if _, err := os.Stat(fmPath); err == nil {
			fmt.Println("--- Generated Files ---")
			displayJSON("file_manifest", fmPath)
			fmt.Println()
		}

		// 8. LLM Conversation logs
		convDir := filepath.Join(artifactsDir, "conversations")
		if convEntries, err := filepath.Glob(filepath.Join(convDir, "conversation_*.json")); err == nil && len(convEntries) > 0 {
			fmt.Printf("--- LLM Conversations (%d total) ---\n", len(convEntries))
			// Show first 3 in full, rest as summary
			for i, ce := range convEntries {
				if i < 3 {
					displayJSON(fmt.Sprintf("conversation %d", i+1), ce)
				} else {
					data, _ := os.ReadFile(ce)
					var entry map[string]interface{}
					json.Unmarshal(data, &entry)
					fmt.Printf("  conversation %d: agent=%s phase=%s\n", i+1, entry["agent_id"], entry["phase"])
				}
			}
			if len(convEntries) > 3 {
				fmt.Printf("  ... and %d more\n", len(convEntries)-3)
			}
			fmt.Println()
		}

		// 9. Agent messages (from memory, if available)
		msgPath := filepath.Join(artifactsDir, "messages.json")
		if _, err := os.Stat(msgPath); err == nil {
			fmt.Println("--- Agent Messages ---")
			displayJSON("messages", msgPath)
			fmt.Println()
		}

		// 10. Raw build log
		buildLog := filepath.Join(projectDir, "build.log")
		if _, err := os.Stat(buildLog); err == nil {
			fmt.Println("--- Build Log (last 30 lines) ---")
			displayTextFile("build.log", buildLog, 30)
			fmt.Println()
		}

		fmt.Println("============================================================")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(inspectCmd)
}
