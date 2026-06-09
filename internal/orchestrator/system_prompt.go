package orchestrator

import (
	"encoding/json"
	"fmt"

	"github.com/routerforge/cli/internal/engine"
	"github.com/routerforge/cli/internal/tool"
	"github.com/routerforge/cli/pkg/models"
)

func GenerateSystemPrompt(role, description string, tools []string, tasks []string, model string) string {
	base := fmt.Sprintf(`You are a %s agent.

Mission: %s

Your capabilities:
- Model: %s
- Tools available: %s
- Tasks assigned: %v

Operational rules:
1. Complete your assigned tasks autonomously
2. Use available tools to accomplish your goals
3. Report results clearly after each task
4. Escalate immediately if blocked or if you need clarification
5. Stay within your role scope — do not take over other agents' work
6. Generate production-quality output every time`,
		role, description, model, tools, tasks)

	return base
}

func GenerateSystemPromptFromLLM(role, description, goal string, tools []string, tasks []string, model string) string {
	prompt := fmt.Sprintf(`Generate a concise, actionable system prompt for a software engineering agent with these details:

Role: %s
Description: %s
Project Goal: %s
Available Tools: %v
Assigned Tasks: %v

The system prompt should:
1. Be 3-5 sentences
2. Define the agent's mission clearly
3. Specify constraints and success criteria
4. Reference the available tools
5. Tell the agent how to report results

Return ONLY the system prompt text, no JSON, no markdown.`,
		role, description, goal, tools, tasks)

	llm := engine.NewLLMClient(model)
	result, err := llm.Chat("You generate concise system prompts for AI agents. Return plain text only.", prompt)
	if err != nil {
		return GenerateSystemPrompt(role, description, tools, tasks, model)
	}
	return result
}

func BuildAgentFromPlan(projectGoal, model string, pa models.PlanAgent, parentID string) *models.Agent {
	sysPrompt := GenerateSystemPrompt(pa.Role, pa.Description, pa.Tools, pa.Tasks, model)

	agentTasks := make([]models.Task, len(pa.Tasks))
	for i, t := range pa.Tasks {
		agentTasks[i] = models.Task{
			ID:          fmt.Sprintf("%s-%d", pa.Role, i),
			Description: t,
			Status:      models.TaskPending,
			Priority:    "medium",
		}
	}

	return &models.Agent{
		ID:              fmt.Sprintf("agent-%s", pa.Role),
		Role:            pa.Role,
		Type:            models.TypeMicroAgent,
		ParentID:        parentID,
		Model:           model,
		Status:          models.StatusCreated,
		SystemPrompt:    sysPrompt,
		Tasks:           agentTasks,
		SuccessCriteria: pa.SuccessCriteria,
		Tools:           pa.Tools,
		MemoryScope:     []string{"project", pa.Role},
		ReportingFormat: "json",
	}
}

func BuildToolRegistry(toolNames []string) *tool.Registry {
	r := tool.NewRegistry()
	tool.RegisterAll(r)

	allowed := make(map[string]bool)
	for _, n := range toolNames {
		allowed[n] = true
	}

	perm := tool.NewPermissionEvaluator()
	for _, t := range r.List() {
		if !allowed[t.Name()] {
			perm.AddRule(tool.PermissionRule{
				ToolPattern: t.Name(),
				Effect:      "deny",
			})
		}
	}

	return r
}

func MarshalPlanAgent(pa models.PlanAgent) ([]byte, error) {
	return json.MarshalIndent(pa, "", "  ")
}
