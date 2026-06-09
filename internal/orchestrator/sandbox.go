package orchestrator

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
)

type SandboxPolicy struct {
	mu            sync.RWMutex
	allowedDirs   []string
	allowedCmds   []string
	blockedCmds   []string
	allowedHosts  []string
	maxFileSize   int64
	allowNetwork  bool
	allowWriting  bool
}

func NewSandboxPolicy() *SandboxPolicy {
	return &SandboxPolicy{
		maxFileSize:  10 * 1024 * 1024,
		allowNetwork: true,
		allowWriting: true,
	}
}

func (sp *SandboxPolicy) AllowDirectory(dir string) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.allowedDirs = append(sp.allowedDirs, dir)
}

func (sp *SandboxPolicy) AllowCommand(cmd string) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.allowedCmds = append(sp.allowedCmds, cmd)
}

func (sp *SandboxPolicy) BlockCommand(cmd string) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.blockedCmds = append(sp.blockedCmds, cmd)
}

func (sp *SandboxPolicy) SetMaxFileSize(size int64) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.maxFileSize = size
}

func (sp *SandboxPolicy) SetAllowNetwork(v bool) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.allowNetwork = v
}

func (sp *SandboxPolicy) SetAllowWriting(v bool) {
	sp.mu.Lock()
	defer sp.mu.Unlock()
	sp.allowWriting = v
}

type ToolSandbox struct {
	policy     *SandboxPolicy
	agentScope map[string]*AgentSandbox
	mu         sync.Mutex
}

type AgentSandbox struct {
	AgentID     string
	AllowedDirs []string
	AllowedCmds []string
	BlockedCmds []string
	MaxFileSize int64
	CanNetwork  bool
	CanWrite    bool
}

func NewToolSandbox(policy *SandboxPolicy) *ToolSandbox {
	return &ToolSandbox{
		policy:     policy,
		agentScope: make(map[string]*AgentSandbox),
	}
}

func (ts *ToolSandbox) RegisterAgent(agentID string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.agentScope[agentID] = &AgentSandbox{
		AgentID:     agentID,
		AllowedDirs: append([]string{}, ts.policy.allowedDirs...),
		AllowedCmds: append([]string{}, ts.policy.allowedCmds...),
		BlockedCmds: append([]string{}, ts.policy.blockedCmds...),
		MaxFileSize: ts.policy.maxFileSize,
		CanNetwork:  ts.policy.allowNetwork,
		CanWrite:    ts.policy.allowWriting,
	}
}

func (ts *ToolSandbox) ConfigureAgent(agentID string, fn func(*AgentSandbox)) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	if s, ok := ts.agentScope[agentID]; ok {
		fn(s)
	}
}

func (ts *ToolSandbox) ValidateFileRead(agentID, filePath string) error {
	ts.mu.Lock()
	sandbox, ok := ts.agentScope[agentID]
	ts.mu.Unlock()
	if !ok {
		return nil
	}
	return validatePath(sandbox.AllowedDirs, filePath)
}

func (ts *ToolSandbox) ValidateFileWrite(agentID, filePath string) error {
	ts.mu.Lock()
	sandbox, ok := ts.agentScope[agentID]
	ts.mu.Unlock()
	if !ok {
		return nil
	}
	if !sandbox.CanWrite {
		return fmt.Errorf("agent %s has no write permission", agentID)
	}
	return validatePath(sandbox.AllowedDirs, filePath)
}

func (ts *ToolSandbox) ValidateCommand(agentID, command string) error {
	ts.mu.Lock()
	sandbox, ok := ts.agentScope[agentID]
	ts.mu.Unlock()
	if !ok {
		return nil
	}

	cmdName := strings.Fields(command)[0]

	for _, blocked := range sandbox.BlockedCmds {
		if strings.Contains(command, blocked) {
			return fmt.Errorf("command %s contains blocked pattern %s", command, blocked)
		}
	}

	if len(sandbox.AllowedCmds) > 0 {
		allowed := false
		for _, a := range sandbox.AllowedCmds {
			if strings.HasPrefix(cmdName, a) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("command %s not in allowed list", cmdName)
		}
	}

	return nil
}

func (ts *ToolSandbox) ValidateNetwork(agentID string) error {
	ts.mu.Lock()
	sandbox, ok := ts.agentScope[agentID]
	ts.mu.Unlock()
	if !ok {
		return nil
	}
	if !sandbox.CanNetwork {
		return fmt.Errorf("agent %s has no network access", agentID)
	}
	return nil
}

func (ts *ToolSandbox) ValidateFileSize(agentID string, size int64) error {
	ts.mu.Lock()
	sandbox, ok := ts.agentScope[agentID]
	ts.mu.Unlock()
	if !ok {
		return nil
	}
	if size > sandbox.MaxFileSize {
		return fmt.Errorf("file size %d exceeds limit %d", size, sandbox.MaxFileSize)
	}
	return nil
}

func validatePath(allowedDirs []string, target string) error {
	if len(allowedDirs) == 0 {
		return nil
	}
	abs, err := filepath.Abs(target)
	if err != nil {
		return err
	}
	for _, dir := range allowedDirs {
		absDir, err := filepath.Abs(dir)
		if err != nil {
			continue
		}
		if strings.HasPrefix(abs, absDir) {
			return nil
		}
	}
	return fmt.Errorf("path %s not in allowed directories", target)
}
