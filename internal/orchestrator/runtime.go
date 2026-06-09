package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"time"
)

type ResourceLimits struct {
	MaxMemoryMB       int `json:"max_memory_mb"`
	MaxCPUCores       int `json:"max_cpu_cores"`
	MaxFileHandles    int `json:"max_file_handles"`
	MaxConcurrentOps  int `json:"max_concurrent_ops"`
	MaxOutputBytes    int `json:"max_output_bytes"`
}

func DefaultResourceLimits() ResourceLimits {
	return ResourceLimits{
		MaxMemoryMB:      512,
		MaxCPUCores:      2,
		MaxFileHandles:   100,
		MaxConcurrentOps: 10,
		MaxOutputBytes:   10 * 1024 * 1024,
	}
}

type ResourceUsage struct {
	AgentID        string    `json:"agent_id"`
	MemoryMB       int       `json:"memory_mb"`
	CPUCores       float64   `json:"cpu_cores"`
	FileHandles    int       `json:"file_handles"`
	ConcurrentOps  int       `json:"concurrent_ops"`
	OutputBytes    int       `json:"output_bytes"`
	LastUpdated    time.Time `json:"last_updated"`
}

type ResourceManager struct {
	mu      sync.Mutex
	limits  map[string]ResourceLimits
	usage   map[string]*ResourceUsage
	defaultLimits ResourceLimits
}

func NewResourceManager() *ResourceManager {
	return &ResourceManager{
		limits:        make(map[string]ResourceLimits),
		usage:         make(map[string]*ResourceUsage),
		defaultLimits: DefaultResourceLimits(),
	}
}

func (rm *ResourceManager) SetDefaultLimits(limits ResourceLimits) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.defaultLimits = limits
}

func (rm *ResourceManager) SetLimits(agentID string, limits ResourceLimits) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.limits[agentID] = limits
}

func (rm *ResourceManager) GetLimits(agentID string) ResourceLimits {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	if l, ok := rm.limits[agentID]; ok {
		return l
	}
	return rm.defaultLimits
}

func (rm *ResourceManager) RegisterAgent(agentID string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	if _, ok := rm.usage[agentID]; !ok {
		rm.usage[agentID] = &ResourceUsage{
			AgentID: agentID,
			LastUpdated: time.Now(),
		}
	}
}

func (rm *ResourceManager) TrackMemory(agentID string, mb int) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	limits := rm.defaultLimits
	if l, ok := rm.limits[agentID]; ok {
		limits = l
	}
	u, ok := rm.usage[agentID]
	if !ok {
		u = &ResourceUsage{AgentID: agentID}
		rm.usage[agentID] = u
	}
	u.MemoryMB = mb
	u.LastUpdated = time.Now()
	if mb > limits.MaxMemoryMB {
		return fmt.Errorf("agent %s memory %dMB exceeds limit %dMB", agentID, mb, limits.MaxMemoryMB)
	}
	return nil
}

func (rm *ResourceManager) TrackCPU(agentID string, cores float64) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	limits := rm.defaultLimits
	if l, ok := rm.limits[agentID]; ok {
		limits = l
	}
	u, ok := rm.usage[agentID]
	if !ok {
		u = &ResourceUsage{AgentID: agentID}
		rm.usage[agentID] = u
	}
	u.CPUCores = cores
	u.LastUpdated = time.Now()
	if cores > float64(limits.MaxCPUCores) {
		return fmt.Errorf("agent %s CPU %.1f cores exceeds limit %d", agentID, cores, limits.MaxCPUCores)
	}
	return nil
}

func (rm *ResourceManager) TrackFileHandles(agentID string, n int) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	limits := rm.defaultLimits
	if l, ok := rm.limits[agentID]; ok {
		limits = l
	}
	u, ok := rm.usage[agentID]
	if !ok {
		u = &ResourceUsage{AgentID: agentID}
		rm.usage[agentID] = u
	}
	u.FileHandles = n
	u.LastUpdated = time.Now()
	if n > limits.MaxFileHandles {
		return fmt.Errorf("agent %s file handles %d exceeds limit %d", agentID, n, limits.MaxFileHandles)
	}
	return nil
}

func (rm *ResourceManager) TrackConcurrentOps(agentID string, n int) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	limits := rm.defaultLimits
	if l, ok := rm.limits[agentID]; ok {
		limits = l
	}
	u, ok := rm.usage[agentID]
	if !ok {
		u = &ResourceUsage{AgentID: agentID}
		rm.usage[agentID] = u
	}
	u.ConcurrentOps = n
	u.LastUpdated = time.Now()
	if n > limits.MaxConcurrentOps {
		return fmt.Errorf("agent %s concurrent operations %d exceeds limit %d", agentID, n, limits.MaxConcurrentOps)
	}
	return nil
}

func (rm *ResourceManager) TrackOutput(agentID string, bytes int) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	limits := rm.defaultLimits
	if l, ok := rm.limits[agentID]; ok {
		limits = l
	}
	u, ok := rm.usage[agentID]
	if !ok {
		u = &ResourceUsage{AgentID: agentID}
		rm.usage[agentID] = u
	}
	u.OutputBytes += bytes
	u.LastUpdated = time.Now()
	if u.OutputBytes > limits.MaxOutputBytes {
		return fmt.Errorf("agent %s output %d bytes exceeds limit %d", agentID, u.OutputBytes, limits.MaxOutputBytes)
	}
	return nil
}

func (rm *ResourceManager) GetUsage(agentID string) *ResourceUsage {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	u, ok := rm.usage[agentID]
	if !ok {
		return nil
	}
	dup := *u
	return &dup
}

func (rm *ResourceManager) AllUsage() []*ResourceUsage {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	out := make([]*ResourceUsage, 0, len(rm.usage))
	for _, u := range rm.usage {
		dup := *u
		out = append(out, &dup)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].AgentID < out[j].AgentID
	})
	return out
}

func (rm *ResourceManager) Summary() string {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	totalMem := 0
	totalCPU := 0.0
	var agents []string
	for id, u := range rm.usage {
		agents = append(agents, id)
		totalMem += u.MemoryMB
		totalCPU += u.CPUCores
		_ = id
	}
	return fmt.Sprintf("Agents: %d | Total Memory: %d MB | Total CPU: %.1f cores", len(agents), totalMem, totalCPU)
}

type EvictionPolicy int

const (
	EvictLRU EvictionPolicy = iota
	EvictTTL
	EvictLargest
)

type MemoryPoolConfig struct {
	MaxSizeMB       int            `json:"max_size_mb"`
	EvictPolicy     EvictionPolicy `json:"evict_policy"`
	DefaultTTL      time.Duration  `json:"default_ttl"`
	CompressAfterMB int            `json:"compress_after_mb"`
}

func DefaultMemoryPoolConfig() MemoryPoolConfig {
	return MemoryPoolConfig{
		MaxSizeMB:       1024,
		EvictPolicy:     EvictLRU,
		DefaultTTL:      30 * time.Minute,
		CompressAfterMB: 100,
	}
}

type PoolEntry struct {
	Key       string      `json:"key"`
	AgentID   string      `json:"agent_id"`
	SizeMB    int         `json:"size_mb"`
	Data      interface{} `json:"-"`
	CreatedAt time.Time   `json:"created_at"`
	LastUsed  time.Time   `json:"last_used"`
	ExpiresAt time.Time   `json:"expires_at"`
	Compressed bool       `json:"compressed"`
}

type MemoryPool struct {
	mu       sync.Mutex
	entries  map[string]*PoolEntry
	config   MemoryPoolConfig
	usedMB   int
}

func NewMemoryPool(config MemoryPoolConfig) *MemoryPool {
	return &MemoryPool{
		entries: make(map[string]*PoolEntry),
		config:  config,
	}
}

func (mp *MemoryPool) Config() MemoryPoolConfig {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	return mp.config
}

func (mp *MemoryPool) SetConfig(config MemoryPoolConfig) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	mp.config = config
}

func (mp *MemoryPool) Store(key, agentID string, data interface{}, sizeMB int) error {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	if mp.usedMB+sizeMB > mp.config.MaxSizeMB {
		mp.evict(mp.usedMB + sizeMB - mp.config.MaxSizeMB)
	}

	if mp.usedMB+sizeMB > mp.config.MaxSizeMB {
		return fmt.Errorf("memory pool full: need %dMB but max is %dMB and cannot evict enough", sizeMB, mp.config.MaxSizeMB)
	}

	now := time.Now()
	entry := &PoolEntry{
		Key:       key,
		AgentID:   agentID,
		SizeMB:    sizeMB,
		Data:      data,
		CreatedAt: now,
		LastUsed:  now,
		ExpiresAt: now.Add(mp.config.DefaultTTL),
		Compressed: sizeMB > mp.config.CompressAfterMB,
	}
	mp.entries[key] = entry
	mp.usedMB += sizeMB
	return nil
}

func (mp *MemoryPool) Load(key string) interface{} {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	entry, ok := mp.entries[key]
	if !ok {
		return nil
	}
	if time.Now().After(entry.ExpiresAt) {
		delete(mp.entries, key)
		mp.usedMB -= entry.SizeMB
		return nil
	}
	entry.LastUsed = time.Now()
	return entry.Data
}

func (mp *MemoryPool) Delete(key string) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	if entry, ok := mp.entries[key]; ok {
		mp.usedMB -= entry.SizeMB
		delete(mp.entries, key)
	}
}

func (mp *MemoryPool) DeleteByAgent(agentID string) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	for key, entry := range mp.entries {
		if entry.AgentID == agentID {
			mp.usedMB -= entry.SizeMB
			delete(mp.entries, key)
		}
	}
}

func (mp *MemoryPool) Stats() (usedMB, totalMB int, entryCount int) {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	return mp.usedMB, mp.config.MaxSizeMB, len(mp.entries)
}

func (mp *MemoryPool) Entries() []*PoolEntry {
	mp.mu.Lock()
	defer mp.mu.Unlock()
	out := make([]*PoolEntry, 0, len(mp.entries))
	for _, e := range mp.entries {
		dup := *e
		dup.Data = nil
		out = append(out, &dup)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].LastUsed.After(out[j].LastUsed)
	})
	return out
}

func (mp *MemoryPool) evict(needMB int) {
	var candidates []*PoolEntry
	for _, e := range mp.entries {
		candidates = append(candidates, e)
	}

	switch mp.config.EvictPolicy {
	case EvictLRU:
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].LastUsed.Before(candidates[j].LastUsed)
		})
	case EvictTTL:
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].ExpiresAt.Before(candidates[j].ExpiresAt)
		})
	case EvictLargest:
		sort.Slice(candidates, func(i, j int) bool {
			return candidates[i].SizeMB > candidates[j].SizeMB
		})
	}

	evicted := 0
	for _, e := range candidates {
		if evicted >= needMB {
			break
		}
		delete(mp.entries, e.Key)
		mp.usedMB -= e.SizeMB
		evicted += e.SizeMB
	}
}

type AgentRuntimeInfo struct {
	AgentID      string           `json:"agent_id"`
	Role         string           `json:"role"`
	Status       string           `json:"status"`
	Limits       ResourceLimits   `json:"limits"`
	Usage        *ResourceUsage   `json:"usage"`
	MemoryMB     int              `json:"memory_mb"`
	RunningSince time.Time        `json:"running_since"`
}

type Runtime struct {
	mu            sync.Mutex
	ResourceMgr   *ResourceManager
	MemoryPool    *MemoryPool
	Sandbox       *ToolSandbox
	scheduler     *Scheduler
	agents        map[string]*AgentRuntimeInfo
	startedAt     time.Time
}

func NewRuntime(sandbox *ToolSandbox, scheduler *Scheduler) *Runtime {
	return &Runtime{
		ResourceMgr: NewResourceManager(),
		MemoryPool:  NewMemoryPool(DefaultMemoryPoolConfig()),
		Sandbox:     sandbox,
		scheduler:   scheduler,
		agents:      make(map[string]*AgentRuntimeInfo),
		startedAt:   time.Now(),
	}
}

func (rt *Runtime) RegisterAgent(agentID, role string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.ResourceMgr.RegisterAgent(agentID)
	if rt.Sandbox != nil {
		rt.Sandbox.RegisterAgent(agentID)
	}
	rt.agents[agentID] = &AgentRuntimeInfo{
		AgentID:      agentID,
		Role:         role,
		Status:       "registered",
		Limits:       DefaultResourceLimits(),
		RunningSince: time.Now(),
	}
}

func (rt *Runtime) StartAgent(agentID, role string, fn func(ctxWithRuntime context.Context) (string, error)) (*AgentProcess, error) {
	rt.RegisterAgent(agentID, role)

	rt.mu.Lock()
	if info, ok := rt.agents[agentID]; ok {
		info.Status = "running"
	}
	rt.mu.Unlock()

	return rt.scheduler.Schedule(agentID, role, func(ctx context.Context) (string, error) {
		return fn(ctx)
	})
}

func (rt *Runtime) GetInfo(agentID string) *AgentRuntimeInfo {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	info, ok := rt.agents[agentID]
	if !ok {
		return nil
	}
	info.Usage = rt.ResourceMgr.GetUsage(agentID)
	info.Limits = rt.ResourceMgr.GetLimits(agentID)
	return info
}

func (rt *Runtime) ListAgents() []*AgentRuntimeInfo {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	out := make([]*AgentRuntimeInfo, 0, len(rt.agents))
	for _, info := range rt.agents {
		dup := *info
		dup.Usage = rt.ResourceMgr.GetUsage(info.AgentID)
		dup.Limits = rt.ResourceMgr.GetLimits(info.AgentID)
		out = append(out, &dup)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].AgentID < out[j].AgentID
	})
	return out
}

func (rt *Runtime) Summary() string {
	rt.mu.Lock()
	agentCount := len(rt.agents)
	rt.mu.Unlock()
	schedulerProcesses := rt.scheduler.List()
	running := 0
	for _, p := range schedulerProcesses {
		if p.Status == "running" {
			running++
		}
	}
	usedMB, totalMB, entries := rt.MemoryPool.Stats()
	resSummary := rt.ResourceMgr.Summary()
	uptime := time.Since(rt.startedAt).Round(time.Second)
	return fmt.Sprintf("Runtime uptime: %s | %s | Pool: %d/%d MB (%d entries) | Agents: %d (%d running)",
		uptime, resSummary, usedMB, totalMB, entries, agentCount, running)
}

func (rt *Runtime) JSON() string {
	rt.mu.Lock()
	agents := make([]*AgentRuntimeInfo, 0, len(rt.agents))
	for _, info := range rt.agents {
		dup := *info
		dup.Usage = rt.ResourceMgr.GetUsage(info.AgentID)
		dup.Limits = rt.ResourceMgr.GetLimits(info.AgentID)
		agents = append(agents, &dup)
	}
	rt.mu.Unlock()

	usedMB, totalMB, entryCount := rt.MemoryPool.Stats()
	procs := rt.scheduler.List()

	data := map[string]interface{}{
		"uptime":   time.Since(rt.startedAt).String(),
		"agents":   agents,
		"processes": procs,
		"memory_pool": map[string]interface{}{
			"used_mb":     usedMB,
			"total_mb":    totalMB,
			"entry_count": entryCount,
		},
		"resource_usage": rt.ResourceMgr.AllUsage(),
	}
	b, _ := json.MarshalIndent(data, "", "  ")
	return string(b)
}

func (rt *Runtime) TrackProcessResourceUsage(agentID string) {
	rt.mu.Lock()
	info, ok := rt.agents[agentID]
	rt.mu.Unlock()
	if !ok {
		return
	}
	_ = info

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			rt.mu.Lock()
			curInfo, exists := rt.agents[agentID]
			rt.mu.Unlock()
			if !exists || curInfo.Status != "running" {
				return
			}

			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)
			memoryMB := int(memStats.Alloc / 1024 / 1024)
			cpuCores := float64(runtime.NumGoroutine()) / 100.0

			_ = rt.ResourceMgr.TrackMemory(agentID, memoryMB)
			_ = rt.ResourceMgr.TrackCPU(agentID, cpuCores)
		}
	}()
}
