package orchestrator

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type AgentProcess struct {
	AgentID    string    `json:"agent_id"`
	Role       string    `json:"role"`
	Status     string    `json:"status"` // running, completed, failed, cancelled
	StartedAt  time.Time `json:"started_at"`
	FinishedAt time.Time `json:"finished_at,omitempty"`
	Error      string    `json:"error,omitempty"`
	Result     string    `json:"result,omitempty"`
}

type Scheduler struct {
	mu             sync.Mutex
	processes      map[string]*AgentProcess
	maxConcurrent  int
	semaphore      chan struct{}
	wg             sync.WaitGroup
	ctx            context.Context
	cancel         context.CancelFunc
	running        bool
}

func NewScheduler(maxConcurrent int) *Scheduler {
	return &Scheduler{
		processes:     make(map[string]*AgentProcess),
		maxConcurrent: maxConcurrent,
	}
}

func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return
	}
	s.ctx, s.cancel = context.WithCancel(context.Background())
	s.semaphore = make(chan struct{}, s.maxConcurrent)
	s.running = true
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	s.running = false
	s.mu.Unlock()
	s.wg.Wait()
}

func (s *Scheduler) Schedule(agentID, role string, fn func(ctx context.Context) (string, error)) (*AgentProcess, error) {
	s.mu.Lock()
	if !s.running {
		s.Start()
	}
	s.mu.Unlock()

	proc := &AgentProcess{
		AgentID:   agentID,
		Role:      role,
		Status:    "running",
		StartedAt: time.Now(),
	}

	s.mu.Lock()
	s.processes[agentID] = proc
	s.mu.Unlock()

	s.semaphore <- struct{}{}
	s.wg.Add(1)

	go func() {
		defer s.wg.Done()
		defer func() { <-s.semaphore }()

		select {
		case <-s.ctx.Done():
			s.mu.Lock()
			proc.Status = "cancelled"
			proc.Error = "scheduler stopped"
			proc.FinishedAt = time.Now()
			s.mu.Unlock()
			return
		default:
		}

		result, err := fn(s.ctx)

		s.mu.Lock()
		proc.FinishedAt = time.Now()
		if err != nil {
			proc.Status = "failed"
			proc.Error = err.Error()
		} else {
			proc.Status = "completed"
			proc.Result = result
		}
		s.mu.Unlock()
	}()

	return proc, nil
}

func (s *Scheduler) Cancel(agentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	proc, ok := s.processes[agentID]
	if !ok {
		return fmt.Errorf("process %s not found", agentID)
	}
	if proc.Status == "running" {
		proc.Status = "cancelled"
		proc.FinishedAt = time.Now()
	}
	return nil
}

func (s *Scheduler) Status(agentID string) (*AgentProcess, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	proc, ok := s.processes[agentID]
	if !ok {
		return nil, fmt.Errorf("process %s not found", agentID)
	}
	return proc, nil
}

func (s *Scheduler) List() []*AgentProcess {
	s.mu.Lock()
	defer s.mu.Unlock()
	var list []*AgentProcess
	for _, p := range s.processes {
		list = append(list, p)
	}
	return list
}

func (s *Scheduler) RunningCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for _, p := range s.processes {
		if p.Status == "running" {
			count++
		}
	}
	return count
}

func (s *Scheduler) WaitAll() {
	s.wg.Wait()
}

func (s *Scheduler) WaitFor(agentID string, timeout time.Duration) (*AgentProcess, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		proc, err := s.Status(agentID)
		if err != nil {
			return nil, err
		}
		if proc.Status != "running" {
			return proc, nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return nil, fmt.Errorf("timeout waiting for %s", agentID)
}
