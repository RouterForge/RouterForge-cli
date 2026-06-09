package engine

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type TerminalSession struct {
	ID        string
	Workdir   string
	CreatedAt time.Time
	History   []TerminalCommand
}

type TerminalCommand struct {
	Command  string
	Output   string
	ExitCode int
	Duration time.Duration
	Time     time.Time
}

type TerminalManager struct {
	mu       sync.Mutex
	sessions map[string]*TerminalSession
	maxSize  int
}

func NewTerminalManager(maxSize int) *TerminalManager {
	return &TerminalManager{
		sessions: make(map[string]*TerminalSession),
		maxSize:  maxSize,
	}
}

func (tm *TerminalManager) CreateSession(workdir string) string {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if len(tm.sessions) >= tm.maxSize {
		var oldestID string
		var oldestTime time.Time
		for id, s := range tm.sessions {
			if oldestID == "" || s.CreatedAt.Before(oldestTime) {
				oldestID = id
				oldestTime = s.CreatedAt
			}
		}
		delete(tm.sessions, oldestID)
	}

	id := fmt.Sprintf("term-%d", time.Now().UnixNano())
	tm.sessions[id] = &TerminalSession{
		ID:        id,
		Workdir:   workdir,
		CreatedAt: time.Now(),
	}
	return id
}

func (tm *TerminalManager) Execute(sessionID, command string, timeout time.Duration) (*TerminalCommand, error) {
	tm.mu.Lock()
	session, ok := tm.sessions[sessionID]
	tm.mu.Unlock()
	if !ok {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	start := time.Now()
	cmdCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, "bash", "-c", command)
	cmd.Dir = session.Workdir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("exec failed: %w", err)
		}
	}

	output := strings.TrimSpace(stdout.String())
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n--- stderr ---\n" + strings.TrimSpace(stderr.String())
		} else {
			output = strings.TrimSpace(stderr.String())
		}
	}

	tc := &TerminalCommand{
		Command:  command,
		Output:   output,
		ExitCode: exitCode,
		Duration: duration,
		Time:     time.Now(),
	}

	tm.mu.Lock()
	session.History = append(session.History, *tc)
	tm.mu.Unlock()

	return tc, nil
}

func (tm *TerminalManager) GetSession(id string) *TerminalSession {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	return tm.sessions[id]
}

func (tm *TerminalManager) ListSessions() []string {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	var ids []string
	for id := range tm.sessions {
		ids = append(ids, id)
	}
	return ids
}

func (tm *TerminalManager) CloseSession(id string) {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	delete(tm.sessions, id)
}

func (tm *TerminalManager) CloseAll() {
	tm.mu.Lock()
	defer tm.mu.Unlock()
	tm.sessions = make(map[string]*TerminalSession)
}

func (s *TerminalSession) HistoryText() string {
	var b strings.Builder
	for _, c := range s.History {
		b.WriteString(fmt.Sprintf("$ %s\n%s\n", c.Command, c.Output))
	}
	return b.String()
}

func RunCommand(workdir, command string, timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), err
	}
	return strings.TrimSpace(string(out)), nil
}

func RunCommandWithDir(workdir, command string) (string, int, error) {
	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = workdir
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return "", -1, err
		}
	}
	return strings.TrimSpace(string(out)), exitCode, nil
}
