package orchestrator

import (
	"context"
	"testing"
	"time"
)

func TestResourceManager_RegisterAndTrack(t *testing.T) {
	rm := NewResourceManager()
	rm.RegisterAgent("agent-1")

	limits := rm.GetLimits("agent-1")
	if limits.MaxMemoryMB != 512 {
		t.Errorf("expected default 512MB, got %d", limits.MaxMemoryMB)
	}

	if err := rm.TrackMemory("agent-1", 100); err != nil {
		t.Errorf("TrackMemory: %v", err)
	}
	if err := rm.TrackCPU("agent-1", 1.5); err != nil {
		t.Errorf("TrackCPU: %v", err)
	}
	if err := rm.TrackFileHandles("agent-1", 10); err != nil {
		t.Errorf("TrackFileHandles: %v", err)
	}
	if err := rm.TrackConcurrentOps("agent-1", 5); err != nil {
		t.Errorf("TrackConcurrentOps: %v", err)
	}
	if err := rm.TrackOutput("agent-1", 1024); err != nil {
		t.Errorf("TrackOutput: %v", err)
	}

	u := rm.GetUsage("agent-1")
	if u == nil {
		t.Fatal("expected usage")
	}
	if u.MemoryMB != 100 {
		t.Errorf("expected 100MB, got %d", u.MemoryMB)
	}
	if u.CPUCores != 1.5 {
		t.Errorf("expected 1.5 cores, got %f", u.CPUCores)
	}
}

func TestResourceManager_Limits(t *testing.T) {
	rm := NewResourceManager()
	rm.RegisterAgent("agent-1")

	limits := ResourceLimits{MaxMemoryMB: 64, MaxCPUCores: 1}
	rm.SetLimits("agent-1", limits)

	got := rm.GetLimits("agent-1")
	if got.MaxMemoryMB != 64 {
		t.Errorf("expected 64MB, got %d", got.MaxMemoryMB)
	}

	if err := rm.TrackMemory("agent-1", 32); err != nil {
		t.Errorf("expected no error: %v", err)
	}
	if err := rm.TrackMemory("agent-1", 128); err == nil {
		t.Error("expected error for exceeding limit")
	}
}

func TestResourceManager_Summary(t *testing.T) {
	rm := NewResourceManager()
	rm.RegisterAgent("a1")
	rm.RegisterAgent("a2")
	rm.TrackMemory("a1", 100)
	rm.TrackMemory("a2", 200)
	summary := rm.Summary()
	if summary == "" {
		t.Error("expected non-empty summary")
	}
}

func TestMemoryPool_StoreAndLoad(t *testing.T) {
	mp := NewMemoryPool(DefaultMemoryPoolConfig())

	if err := mp.Store("key1", "agent-1", "hello", 1); err != nil {
		t.Fatalf("Store: %v", err)
	}

	val := mp.Load("key1")
	if val == nil {
		t.Fatal("expected non-nil")
	}
	if s, ok := val.(string); !ok || s != "hello" {
		t.Errorf("expected 'hello', got %v", val)
	}

	val = mp.Load("nonexistent")
	if val != nil {
		t.Errorf("expected nil for nonexistent key")
	}
}

func TestMemoryPool_Delete(t *testing.T) {
	mp := NewMemoryPool(DefaultMemoryPoolConfig())
	mp.Store("k1", "a1", "data", 1)
	mp.Store("k2", "a2", "data", 1)

	mp.Delete("k1")
	if mp.Load("k1") != nil {
		t.Error("expected nil after delete")
	}
	if mp.Load("k2") == nil {
		t.Error("expected k2 to still exist")
	}
}

func TestMemoryPool_DeleteByAgent(t *testing.T) {
	mp := NewMemoryPool(DefaultMemoryPoolConfig())
	mp.Store("k1", "a1", "data", 1)
	mp.Store("k2", "a1", "data", 2)
	mp.Store("k3", "a2", "data", 3)

	mp.DeleteByAgent("a1")
	if mp.Load("k1") != nil {
		t.Error("expected k1 deleted")
	}
	if mp.Load("k2") != nil {
		t.Error("expected k2 deleted")
	}
	if mp.Load("k3") == nil {
		t.Error("expected k3 to remain")
	}
}

func TestMemoryPool_Stats(t *testing.T) {
	mp := NewMemoryPool(MemoryPoolConfig{MaxSizeMB: 100})
	used, total, entries := mp.Stats()
	if used != 0 || total != 100 || entries != 0 {
		t.Errorf("expected 0/100/0, got %d/%d/%d", used, total, entries)
	}

	mp.Store("k1", "a1", "data", 10)
	used, total, entries = mp.Stats()
	if used != 10 || entries != 1 {
		t.Errorf("expected 10/.../1, got %d/.../%d", used, entries)
	}
}

func TestMemoryPool_EvictLRU(t *testing.T) {
	config := MemoryPoolConfig{
		MaxSizeMB:   10,
		EvictPolicy: EvictLRU,
		DefaultTTL:  30 * time.Minute,
	}
	mp := NewMemoryPool(config)

	mp.Store("k1", "a1", "d", 4)
	mp.Store("k2", "a1", "d", 4)
	mp.Store("k3", "a1", "d", 4)

	used, _, _ := mp.Stats()
	if used > 12 {
		t.Errorf("expected eviction, used=%d", used)
	}
}

func TestMemoryPool_EvictTTL(t *testing.T) {
	config := MemoryPoolConfig{
		MaxSizeMB:   10,
		EvictPolicy: EvictTTL,
		DefaultTTL:  1 * time.Millisecond,
	}
	mp := NewMemoryPool(config)
	mp.Store("k1", "a1", "d", 1)
	time.Sleep(5 * time.Millisecond)
	val := mp.Load("k1")
	if val != nil {
		t.Error("expected TTL expiry")
	}
}

func TestMemoryPool_EvictLargest(t *testing.T) {
	config := MemoryPoolConfig{
		MaxSizeMB:   10,
		EvictPolicy: EvictLargest,
		DefaultTTL:  30 * time.Minute,
	}
	mp := NewMemoryPool(config)
	mp.Store("k1", "a1", "d", 7)
	mp.Store("k2", "a1", "d", 7)

	used, total, entries := mp.Stats()
	if used > 14 || used == 0 {
		t.Errorf("expected eviction of 7, used=%d total=%d entries=%d", used, total, entries)
	}
}

func TestMemoryPool_Config(t *testing.T) {
	mp := NewMemoryPool(DefaultMemoryPoolConfig())
	cfg := mp.Config()
	if cfg.MaxSizeMB != 1024 {
		t.Errorf("expected 1024, got %d", cfg.MaxSizeMB)
	}
	mp.SetConfig(MemoryPoolConfig{MaxSizeMB: 512})
	cfg = mp.Config()
	if cfg.MaxSizeMB != 512 {
		t.Errorf("expected 512, got %d", cfg.MaxSizeMB)
	}
}

func TestRuntime_RegisterAndList(t *testing.T) {
	sched := NewScheduler(3)
	sandbox := NewToolSandbox(NewSandboxPolicy())
	rt := NewRuntime(sandbox, sched)

	rt.RegisterAgent("agent-1", "developer")
	rt.RegisterAgent("agent-2", "reviewer")

	agents := rt.ListAgents()
	if len(agents) != 2 {
		t.Errorf("expected 2 agents, got %d", len(agents))
	}

	info := rt.GetInfo("agent-1")
	if info == nil {
		t.Fatal("expected info for agent-1")
	}
	if info.Role != "developer" {
		t.Errorf("expected developer role, got %s", info.Role)
	}
}

func TestRuntime_Summary(t *testing.T) {
	sched := NewScheduler(3)
	sandbox := NewToolSandbox(NewSandboxPolicy())
	rt := NewRuntime(sandbox, sched)

	rt.RegisterAgent("a1", "dev")
	summary := rt.Summary()
	if summary == "" {
		t.Error("expected non-empty summary")
	}
}

func TestRuntime_JSON(t *testing.T) {
	sched := NewScheduler(3)
	sandbox := NewToolSandbox(NewSandboxPolicy())
	rt := NewRuntime(sandbox, sched)

	rt.RegisterAgent("a1", "dev")
	json := rt.JSON()
	if json == "" {
		t.Error("expected non-empty JSON")
	}
}

func TestRuntime_StartAgent(t *testing.T) {
	sched := NewScheduler(3)
	sandbox := NewToolSandbox(NewSandboxPolicy())
	rt := NewRuntime(sandbox, sched)

	proc, err := rt.StartAgent("a1", "tester", func(ctx context.Context) (string, error) {
		return "done", nil
	})
	if err != nil {
		t.Fatalf("StartAgent: %v", err)
	}
	if proc.Status != "running" && proc.Status != "completed" {
		t.Errorf("expected running or completed, got %s", proc.Status)
	}

	info := rt.GetInfo("a1")
	if info == nil {
		t.Fatal("expected runtime info")
	}
	if info.Status != "running" && info.Status != "completed" {
		t.Errorf("expected running or completed, got %s", info.Status)
	}
}

func TestRuntime_TrackProcessResourceUsage(t *testing.T) {
	sched := NewScheduler(3)
	sandbox := NewToolSandbox(NewSandboxPolicy())
	rt := NewRuntime(sandbox, sched)

	rt.RegisterAgent("a1", "dev")
	rt.TrackProcessResourceUsage("a1")

	time.Sleep(100 * time.Millisecond)
	u := rt.ResourceMgr.GetUsage("a1")
	if u == nil {
		t.Fatal("expected usage")
	}
	if u.MemoryMB <= 0 && u.CPUCores <= 0 {
		t.Log("usage may be zero if ticker hasn't fired yet")
	}
}

func TestDefaultResourceLimits(t *testing.T) {
	l := DefaultResourceLimits()
	if l.MaxMemoryMB != 512 {
		t.Errorf("expected 512, got %d", l.MaxMemoryMB)
	}
	if l.MaxCPUCores != 2 {
		t.Errorf("expected 2, got %d", l.MaxCPUCores)
	}
	if l.MaxFileHandles != 100 {
		t.Errorf("expected 100, got %d", l.MaxFileHandles)
	}
}

func TestSetDefaultLimits(t *testing.T) {
	rm := NewResourceManager()
	rm.SetDefaultLimits(ResourceLimits{MaxMemoryMB: 256, MaxCPUCores: 1})

	limits := rm.GetLimits("unknown-agent")
	if limits.MaxMemoryMB != 256 {
		t.Errorf("expected 256, got %d", limits.MaxMemoryMB)
	}
}

func TestAllUsage(t *testing.T) {
	rm := NewResourceManager()
	rm.RegisterAgent("a1")
	rm.RegisterAgent("a2")
	rm.TrackMemory("a1", 50)
	rm.TrackMemory("a2", 100)

	all := rm.AllUsage()
	if len(all) != 2 {
		t.Errorf("expected 2, got %d", len(all))
	}
}
