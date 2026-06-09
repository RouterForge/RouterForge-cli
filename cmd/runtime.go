package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pterm/pterm"
	"github.com/routerforge/cli/internal/orchestrator"
	"github.com/routerforge/cli/internal/storage"
	"github.com/spf13/cobra"
)

func loadHeadManager() *orchestrator.HeadManager {
	projectDir, _ := os.Getwd()
	routerDir := filepath.Join(projectDir, ".routerforge")
	configPath := filepath.Join(routerDir, "config.json")
	cfg, err := storage.LoadConfig(configPath)
	if err != nil {
		cfg = storage.DefaultConfig()
	}
	return orchestrator.NewHeadManager(cfg.Model)
}

var runtimeCmd = &cobra.Command{
	Use:   "runtime",
	Short: "OS-level runtime: resource limits, memory pool, agent telemetry",
	Run: func(cmd *cobra.Command, args []string) {
		hm := loadHeadManager()
		r := hm.Runtime()
		pterm.DefaultSection.Println("Runtime Summary")
		pterm.Println(r.Summary())
	},
}

var runtimeLimitsCmd = &cobra.Command{
	Use:   "limits <agent-id> [max-memory-mb] [max-cpu-cores]",
	Short: "View or set resource limits for an agent",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		hm := loadHeadManager()
		rm := hm.ResourceManager()
		agentID := args[0]

		if len(args) >= 3 {
			memMB := 0
			cpuCores := 0
			fmt.Sscanf(args[1], "%d", &memMB)
			fmt.Sscanf(args[2], "%d", &cpuCores)
			limits := orchestrator.DefaultResourceLimits()
			if memMB > 0 {
				limits.MaxMemoryMB = memMB
			}
			if cpuCores > 0 {
				limits.MaxCPUCores = cpuCores
			}
			rm.SetLimits(agentID, limits)
			pterm.Success.Printfln("Set limits for %s: memory=%dMB, cpu=%d cores", agentID, limits.MaxMemoryMB, limits.MaxCPUCores)
			return
		}

		limits := rm.GetLimits(agentID)
		usage := rm.GetUsage(agentID)
		pterm.DefaultSection.Printfln("Resources for %s", agentID)
		pterm.Printfln("  Limits:   memory=%dMB, cpu=%d cores, files=%d, ops=%d",
			limits.MaxMemoryMB, limits.MaxCPUCores, limits.MaxFileHandles, limits.MaxConcurrentOps)
		if usage != nil {
			pterm.Printfln("  Usage:    memory=%dMB, cpu=%.1f cores, files=%d, ops=%d",
				usage.MemoryMB, usage.CPUCores, usage.FileHandles, usage.ConcurrentOps)
		}
	},
}

var runtimePoolCmd = &cobra.Command{
	Use:   "pool",
	Short: "Show memory pool status",
	Run: func(cmd *cobra.Command, args []string) {
		hm := loadHeadManager()
		mp := hm.MemoryPool()
		usedMB, totalMB, entries := mp.Stats()
		config := mp.Config()

		pterm.DefaultSection.Println("Memory Pool")
		pterm.Printfln("  Size:     %d / %d MB (%d%%)", usedMB, totalMB, usedMB*100/totalMB)
		pterm.Printfln("  Entries:  %d", entries)
		pterm.Printfln("  Policy:   evict=%d, ttl=%v, compress_after=%dMB",
			config.EvictPolicy, config.DefaultTTL, config.CompressAfterMB)

		if entryList := mp.Entries(); len(entryList) > 0 {
			pterm.Println()
			pterm.DefaultSection.Println("Pool Entries")
			for _, e := range entryList {
				pterm.Printfln("  %s: agent=%s, size=%dMB, expires=%s",
					e.Key, e.AgentID, e.SizeMB, e.ExpiresAt.Format("15:04:05"))
			}
		}
	},
}

var runtimeAgentsCmd = &cobra.Command{
	Use:   "agents",
	Short: "List all agents with resource usage",
	Run: func(cmd *cobra.Command, args []string) {
		hm := loadHeadManager()
		r := hm.Runtime()
		agents := r.ListAgents()

		pterm.DefaultSection.Printfln("Agents (%d)", len(agents))
		for _, a := range agents {
			pterm.Printfln("  %s [%s] role=%s", a.AgentID[:8], a.Status, a.Role)
			if a.Usage != nil {
				pterm.Printfln("    mem=%dMB cpu=%.1f files=%d ops=%d",
					a.Usage.MemoryMB, a.Usage.CPUCores, a.Usage.FileHandles, a.Usage.ConcurrentOps)
			}
		}
	},
}

var runtimeJSONCmd = &cobra.Command{
	Use:   "json",
	Short: "Full runtime state as JSON",
	Run: func(cmd *cobra.Command, args []string) {
		hm := loadHeadManager()
		fmt.Println(hm.Runtime().JSON())
	},
}

func init() {
	runtimeCmd.AddCommand(runtimeLimitsCmd)
	runtimeCmd.AddCommand(runtimePoolCmd)
	runtimeCmd.AddCommand(runtimeAgentsCmd)
	runtimeCmd.AddCommand(runtimeJSONCmd)
	rootCmd.AddCommand(runtimeCmd)
}
