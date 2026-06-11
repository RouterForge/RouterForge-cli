package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type ChatMessage struct {
	Role       string     `json:"role"`
	Content    string     `json:"content"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
}

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type Usage struct {
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	Cost             float64 `json:"cost"`
}

func (u *Usage) Add(other Usage) {
	u.PromptTokens += other.PromptTokens
	u.CompletionTokens += other.CompletionTokens
	u.TotalTokens += other.TotalTokens
	u.Cost += other.Cost
}

type LLMResponse struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls,omitempty"`
	Usage     Usage      `json:"usage"`
}

type CostCallback func(model, agentID, phase string, usage Usage)

type ConversationEntry struct {
	Timestamp    string `json:"timestamp"`
	AgentID      string `json:"agent_id"`
	Phase        string `json:"phase"`
	Model        string `json:"model"`
	SystemPrompt string `json:"system_prompt"`
	UserPrompt   string `json:"user_prompt"`
	Response     string `json:"response"`
	Error        string `json:"error,omitempty"`
}

type LLMClient struct {
	BaseURL        string
	Model          string
	Client         *http.Client
	CostHandler    CostCallback
	AgentID        string
	Phase          string
	ConversationsDir string
}

func NewLLMClient(model string) *LLMClient {
	return &LLMClient{
		BaseURL: "https://opencode.ai/zen/v1",
		Model:   model,
		Client:  &http.Client{Timeout: 300 * time.Second},
	}
}

func (c *LLMClient) Chat(systemPrompt, userPrompt string) (string, error) {
	return c.chatWithRetry(systemPrompt, userPrompt, 2)
}

func (c *LLMClient) saveConversation(systemPrompt, userPrompt, response, errMsg string) {
	if c.ConversationsDir == "" {
		return
	}
	os.MkdirAll(c.ConversationsDir, 0755)
	entry := ConversationEntry{
		Timestamp:    time.Now().UTC().Format(time.RFC3339Nano),
		AgentID:      c.AgentID,
		Phase:        c.Phase,
		Model:        c.Model,
		SystemPrompt: systemPrompt,
		UserPrompt:   userPrompt,
		Response:     response,
		Error:        errMsg,
	}
	data, _ := json.MarshalIndent(entry, "", "  ")
	filename := fmt.Sprintf("conversation_%s_%s_%d.json", c.AgentID, c.Phase, time.Now().UnixNano())
	os.WriteFile(filepath.Join(c.ConversationsDir, filename), data, 0644)
}

func (c *LLMClient) chatWithRetry(systemPrompt, userPrompt string, retries int) (string, error) {
	req := map[string]interface{}{
		"model": c.Model,
		"messages": []map[string]string{
			{"role": "system", "content": systemPrompt},
			{"role": "user", "content": userPrompt},
		},
	}

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", c.BaseURL+"/chat/completions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		if retries > 0 && (strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "deadline exceeded")) {
			return c.chatWithRetry(systemPrompt, userPrompt, retries-1)
		}
		c.saveConversation(systemPrompt, userPrompt, "", err.Error())
		return "", fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		c.saveConversation(systemPrompt, userPrompt, "", fmt.Sprintf("HTTP %d: %s", resp.StatusCode, string(respBody)))
		return "", fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
		Cost string `json:"cost"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		c.saveConversation(systemPrompt, userPrompt, "", fmt.Sprintf("parse: %v", err))
		return "", fmt.Errorf("parse: %w", err)
	}
	if len(result.Choices) == 0 {
		c.saveConversation(systemPrompt, userPrompt, "", "no choices")
		return "", fmt.Errorf("no choices")
	}

	response := strings.TrimSpace(result.Choices[0].Message.Content)
	c.saveConversation(systemPrompt, userPrompt, response, "")

	if c.CostHandler != nil {
		cost := 0.0
		if result.Cost != "" {
			fmt.Sscanf(result.Cost, "%f", &cost)
		}
		c.CostHandler(c.Model, c.AgentID, c.Phase, Usage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
			Cost:             cost,
		})
	}

	return response, nil
}

func (c *LLMClient) ChatWithTools(messages []interface{}, toolDefs []interface{}) (*LLMResponse, error) {
	req := map[string]interface{}{
		"model":    c.Model,
		"messages": messages,
	}
	if len(toolDefs) > 0 {
		req["tools"] = toolDefs
	}

	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequest("POST", c.BaseURL+"/chat/completions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("api call: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content   string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("no choices")
	}

	msg := result.Choices[0].Message
	lr := &LLMResponse{
		Content: strings.TrimSpace(msg.Content),
		Usage: Usage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		},
	}
	for _, tc := range msg.ToolCalls {
		t := ToolCall{ID: tc.ID, Type: tc.Type}
		t.Function.Name = tc.Function.Name
		t.Function.Arguments = tc.Function.Arguments
		lr.ToolCalls = append(lr.ToolCalls, t)
	}
	return lr, nil
}

func (c *LLMClient) GenerateCode(role, task string) (string, error) {
	system := fmt.Sprintf(`You are a %s agent. Generate production-quality code only. No explanations, no markdown fences — just raw code.`, role)
	return c.Chat(system, task)
}

func (c *LLMClient) GenerateHTML(spec string) (string, error) {
	system := `You are a senior frontend developer. Generate a complete, high-quality single HTML file. Return ONLY raw HTML.`
	return c.Chat(system, spec)
}
