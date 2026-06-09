package orchestrator

import (
	"fmt"
	"math"
	"sync"
	"time"
)

type CostEntry struct {
	Model      string    `json:"model"`
	AgentID    string    `json:"agent_id"`
	Phase      string    `json:"phase"`
	PromptTok  int       `json:"prompt_tokens"`
	OutputTok  int       `json:"output_tokens"`
	TotalTok   int       `json:"total_tokens"`
	CostUSD    float64   `json:"cost_usd"`
	Time       time.Time `json:"time"`
}

type CostTracker struct {
	mu      sync.Mutex
	entries []CostEntry
}

func NewCostTracker() *CostTracker {
	return &CostTracker{}
}

func (ct *CostTracker) Track(model, agentID, phase string, promptTok, outputTok, totalTok int, cost float64) {
	rate := ModelCostPerToken(model)
	if cost == 0 {
		cost = rate * float64(totalTok)
	}
	ct.mu.Lock()
	defer ct.mu.Unlock()
	ct.entries = append(ct.entries, CostEntry{
		Model:     model,
		AgentID:   agentID,
		Phase:     phase,
		PromptTok: promptTok,
		OutputTok: outputTok,
		TotalTok:  totalTok,
		CostUSD:   cost,
		Time:      time.Now(),
	})
}

func (ct *CostTracker) TotalCost() float64 {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	total := 0.0
	for _, e := range ct.entries {
		total += e.CostUSD
	}
	return math.Round(total*100000) / 100000
}

func (ct *CostTracker) TotalTokens() int {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	total := 0
	for _, e := range ct.entries {
		total += e.TotalTok
	}
	return total
}

func (ct *CostTracker) ByPhase(phase string) float64 {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	total := 0.0
	for _, e := range ct.entries {
		if e.Phase == phase {
			total += e.CostUSD
		}
	}
	return math.Round(total*100000) / 100000
}

func (ct *CostTracker) ByAgent(agentID string) float64 {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	total := 0.0
	for _, e := range ct.entries {
		if e.AgentID == agentID {
			total += e.CostUSD
		}
	}
	return math.Round(total*100000) / 100000
}

func (ct *CostTracker) Summary() map[string]interface{} {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	byModel := make(map[string]float64)
	byPhase := make(map[string]float64)
	byAgent := make(map[string]float64)
	total := 0.0
	totalTok := 0
	for _, e := range ct.entries {
		byModel[e.Model] += e.CostUSD
		byPhase[e.Phase] += e.CostUSD
		byAgent[e.AgentID] += e.CostUSD
		total += e.CostUSD
		totalTok += e.TotalTok
	}
	return map[string]interface{}{
		"total_cost": math.Round(total*100000) / 100000,
		"total_tokens": totalTok,
		"by_model":   byModel,
		"by_phase":   byPhase,
		"by_agent":   byAgent,
		"entries":    len(ct.entries),
	}
}

func (ct *CostTracker) Entries() []CostEntry {
	ct.mu.Lock()
	defer ct.mu.Unlock()
	out := make([]CostEntry, len(ct.entries))
	copy(out, ct.entries)
	return out
}

func ModelCostPerToken(model string) float64 {
	freeModels := map[string]bool{
		"big-pickle":              true,
		"deepseek-v4-flash-free":  true,
		"mimo-v2.5-free":         true,
		"nemotron-3-super-free":  true,
		"nemotron-3-ultra-free":  true,
	}
	if freeModels[model] {
		return 0.0
	}
	return 0.000002
}

func ModelCostDisplay(model string) string {
	cost := ModelCostPerToken(model)
	if cost == 0 {
		return "FREE"
	}
	return fmt.Sprintf("$%.6f/token", cost)
}
