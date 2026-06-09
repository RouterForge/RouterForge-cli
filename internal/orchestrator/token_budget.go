package orchestrator

import (
	"fmt"
	"sync"
	"time"
)

type TokenBudget struct {
	mu              sync.Mutex
	agentBudgets    map[string]*AgentTokenBudget
	phaseBudgets    map[string]*PhaseTokenBudget
	globalMaxTokens int
	globalUsed      int
}

type AgentTokenBudget struct {
	AgentID     string
	MaxTokens   int
	UsedTokens  int
	Warnings    []string
	Phase       string
}

type PhaseTokenBudget struct {
	Phase       string
	MaxTokens   int
	UsedTokens  int
}

func NewTokenBudget(globalMaxTokens int) *TokenBudget {
	return &TokenBudget{
		agentBudgets:    make(map[string]*AgentTokenBudget),
		phaseBudgets:    make(map[string]*PhaseTokenBudget),
		globalMaxTokens: globalMaxTokens,
	}
}

func (tb *TokenBudget) SetAgentBudget(agentID string, maxTokens int) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.agentBudgets[agentID] = &AgentTokenBudget{
		AgentID:   agentID,
		MaxTokens: maxTokens,
	}
}

func (tb *TokenBudget) SetPhaseBudget(phase string, maxTokens int) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.phaseBudgets[phase] = &PhaseTokenBudget{
		Phase:     phase,
		MaxTokens: maxTokens,
	}
}

func (tb *TokenBudget) RecordUsage(agentID, phase string, tokens int) error {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	tb.globalUsed += tokens

	if bg, ok := tb.agentBudgets[agentID]; ok {
		bg.UsedTokens += tokens
		bg.Phase = phase
		ratio := float64(bg.UsedTokens) / float64(bg.MaxTokens)
		if ratio >= 0.9 {
			warning := fmt.Sprintf("agent %s at %.0f%% of token budget (%d/%d)",
				agentID, ratio*100, bg.UsedTokens, bg.MaxTokens)
			bg.Warnings = append(bg.Warnings, warning)
			if bg.UsedTokens >= bg.MaxTokens {
				return fmt.Errorf("agent %s exceeded token budget: %d/%d",
					agentID, bg.UsedTokens, bg.MaxTokens)
			}
		}
	}

	if bp, ok := tb.phaseBudgets[phase]; ok {
		bp.UsedTokens += tokens
		if bp.UsedTokens >= bp.MaxTokens {
			return fmt.Errorf("phase %s exceeded token budget: %d/%d",
				phase, bp.UsedTokens, bp.MaxTokens)
		}
	}

	if tb.globalMaxTokens > 0 && tb.globalUsed >= tb.globalMaxTokens {
		return fmt.Errorf("global token budget exceeded: %d/%d",
			tb.globalUsed, tb.globalMaxTokens)
	}

	return nil
}

func (tb *TokenBudget) AgentUsage(agentID string) *AgentTokenBudget {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.agentBudgets[agentID]
}

func (tb *TokenBudget) GlobalUsage() (int, int) {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	return tb.globalUsed, tb.globalMaxTokens
}

func (tb *TokenBudget) PhaseUsage() map[string]int {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	result := make(map[string]int)
	for p, b := range tb.phaseBudgets {
		result[p] = b.UsedTokens
	}
	return result
}

func (tb *TokenBudget) Reset() {
	tb.mu.Lock()
	defer tb.mu.Unlock()
	tb.globalUsed = 0
	for _, b := range tb.agentBudgets {
		b.UsedTokens = 0
		b.Warnings = nil
	}
	for _, b := range tb.phaseBudgets {
		b.UsedTokens = 0
	}
}

type TokenTracker struct {
	Entries []TokenEntry
	mu      sync.Mutex
}

type TokenEntry struct {
	AgentID string    `json:"agent_id"`
	Phase   string    `json:"phase"`
	TaskID  string    `json:"task_id"`
	Prompt  int       `json:"prompt_tokens"`
	Output  int       `json:"output_tokens"`
	Total   int       `json:"total_tokens"`
	Time    time.Time `json:"time"`
}

func NewTokenTracker() *TokenTracker {
	return &TokenTracker{}
}

func (tt *TokenTracker) Track(agentID, phase, taskID string, prompt, output, total int) {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	tt.Entries = append(tt.Entries, TokenEntry{
		AgentID: agentID,
		Phase:   phase,
		TaskID:  taskID,
		Prompt:  prompt,
		Output:  output,
		Total:   total,
		Time:    time.Now(),
	})
}

func (tt *TokenTracker) TotalTokens() int {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	total := 0
	for _, e := range tt.Entries {
		total += e.Total
	}
	return total
}

func (tt *TokenTracker) ByAgent(agentID string) []TokenEntry {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	var out []TokenEntry
	for _, e := range tt.Entries {
		if e.AgentID == agentID {
			out = append(out, e)
		}
	}
	return out
}

func (tt *TokenTracker) Summary() map[string]int {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	summary := make(map[string]int)
	for _, e := range tt.Entries {
		summary[e.AgentID] += e.Total
	}
	return summary
}
