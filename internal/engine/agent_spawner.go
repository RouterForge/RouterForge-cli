package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type SpawnedAgent struct {
	ID            string    `json:"id"`
	Role          string    `json:"role"`
	SystemPrompt  string    `json:"system_prompt"`
	Model         string    `json:"model"`
	Status        string    `json:"status"`
	ParentID      string    `json:"parent_id"`
	CreatedAt     time.Time `json:"created_at"`
	Result        string    `json:"result,omitempty"`
}

type AgentSpawner struct {
	model       string
	baseURL     string
	spawned     map[string]*SpawnedAgent
}

func NewAgentSpawner(model string) *AgentSpawner {
	return &AgentSpawner{
		model:   model,
		baseURL: "https://opencode.ai/zen/v1",
		spawned: make(map[string]*SpawnedAgent),
	}
}

func (as *AgentSpawner) Spawn(role, task string, parentID string) (*SpawnedAgent, error) {
	id := fmt.Sprintf("spawn-%s-%d", strings.ReplaceAll(role, " ", "_"), time.Now().UnixNano())

	llm := NewLLMClient(as.model)
	systemPrompt := fmt.Sprintf(`You are a %s sub-agent spawned by parent %s.
Your mission: %s
You are autonomous. Complete the task and return your result. Do not ask for help unless absolutely necessary.`, role, parentID, task)

	result, err := llm.Chat(systemPrompt, task)
	if err != nil {
		return nil, fmt.Errorf("spawn agent %s: %w", role, err)
	}

	agent := &SpawnedAgent{
		ID:           id,
		Role:         role,
		SystemPrompt: systemPrompt,
		Model:        as.model,
		Status:       "completed",
		ParentID:     parentID,
		CreatedAt:    time.Now(),
		Result:       result,
	}
	as.spawned[id] = agent
	return agent, nil
}

func (as *AgentSpawner) SpawnWithTools(role, task string, parentID string, toolNames []string) (*SpawnedAgent, error) {
	id := fmt.Sprintf("spawn-%s-%d", strings.ReplaceAll(role, " ", "_"), time.Now().UnixNano())

	prompt := fmt.Sprintf(`Role: %s
Parent: %s
Task: %s
Available tools: %v

Complete the task using the tools available. Return your final result.`,
		role, parentID, task, toolNames)

	llm := NewLLMClient(as.model)
	result, err := llm.Chat(
		fmt.Sprintf("You are a %s sub-agent with tool access.", role),
		prompt,
	)
	if err != nil {
		return nil, fmt.Errorf("spawn agent with tools: %w", err)
	}

	agent := &SpawnedAgent{
		ID:           id,
		Role:         role,
		SystemPrompt: prompt,
		Model:        as.model,
		Status:       "completed",
		ParentID:     parentID,
		CreatedAt:    time.Now(),
		Result:       result,
	}
	as.spawned[id] = agent
	return agent, nil
}

func (as *AgentSpawner) SpawnBatch(agents []struct {
	Role     string `json:"role"`
	Task     string `json:"task"`
	ParentID string `json:"parent_id"`
}) []*SpawnedAgent {
	results := make([]*SpawnedAgent, 0, len(agents))
	for _, a := range agents {
		spawned, err := as.Spawn(a.Role, a.Task, a.ParentID)
		if err != nil {
			spawned = &SpawnedAgent{
				ID:       fmt.Sprintf("spawn-%s-failed", a.Role),
				Role:     a.Role,
				Status:   "failed",
				ParentID: a.ParentID,
			}
		}
		results = append(results, spawned)
	}
	return results
}

func (as *AgentSpawner) GetAgent(id string) *SpawnedAgent {
	return as.spawned[id]
}

func (as *AgentSpawner) ListAgents() []*SpawnedAgent {
	out := make([]*SpawnedAgent, 0, len(as.spawned))
	for _, a := range as.spawned {
		out = append(out, a)
	}
	return out
}

func (as *AgentSpawner) ToJSON() string {
	b, _ := json.MarshalIndent(as.spawned, "", "  ")
	return string(b)
}

type SubAgentTask struct {
	ctx    context.Context
	role   string
	task   string
	result chan string
}

func RunSubAgent(ctx context.Context, model, role, task string) string {
	llm := NewLLMClient(model)
	systemPrompt := fmt.Sprintf("You are a %s sub-agent. Complete the assigned task autonomously.", role)
	result, err := llm.Chat(systemPrompt, task)
	if err != nil {
		return fmt.Sprintf("sub-agent %s failed: %v", role, err)
	}
	return result
}
