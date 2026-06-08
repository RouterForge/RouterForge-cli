package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type OpenCodeProvider struct {
	baseURL  string
	client   *http.Client
}

func NewOpenCodeProvider() *OpenCodeProvider {
	return &OpenCodeProvider{
		baseURL: "https://opencode.ai/zen/v1",
		client:  &http.Client{Timeout: 180 * time.Second},
	}
}

func (p *OpenCodeProvider) Name() string { return "opencode" }

func (p *OpenCodeProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	req.Stream = false
	body, _ := json.Marshal(req)
	httpReq, _ := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &ProviderError{Message: "opencode api call", Cause: err}
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, &ProviderError{Message: fmt.Sprintf("API %d", resp.StatusCode), Cause: fmt.Errorf("%s", string(respBody))}
	}

	var openAIResp struct {
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

	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return nil, &ProviderError{Message: "parse response", Cause: err}
	}
	if len(openAIResp.Choices) == 0 {
		return nil, &ProviderError{Message: "no choices in response"}
	}

	msg := openAIResp.Choices[0].Message
	cr := &ChatResponse{
		Message: Message{
			Role:    RoleAssistant,
			Content: strings.TrimSpace(msg.Content),
		},
		Usage: Usage{
			PromptTokens:     openAIResp.Usage.PromptTokens,
			CompletionTokens: openAIResp.Usage.CompletionTokens,
			TotalTokens:      openAIResp.Usage.TotalTokens,
		},
	}
	for _, tc := range msg.ToolCalls {
		t := ToolCall{ID: tc.ID, Type: tc.Type}
		t.Function.Name = tc.Function.Name
		t.Function.Arguments = tc.Function.Arguments
		cr.Message.ToolCalls = append(cr.Message.ToolCalls, t)
	}
	return cr, nil
}

func (p *OpenCodeProvider) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error) {
	req.Stream = true
	body, _ := json.Marshal(req)

	httpReq, _ := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, &ProviderError{Message: "stream api call", Cause: err}
	}

	ch := make(chan StreamEvent, 64)

	go func() {
		defer resp.Body.Close()
		defer close(ch)

		dec := json.NewDecoder(resp.Body)
		for {
			var line string
			if err := dec.Decode(&line); err != nil {
				if err == io.EOF {
					break
				}
				ch <- StreamEvent{Type: StreamError, Error: err}
				return
			}
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				ch <- StreamEvent{Type: StreamDone, Done: true}
				return
			}

			var chunk struct {
				Choices []struct {
					Delta struct {
						Content   string `json:"content"`
						ToolCalls []struct {
							Index    int    `json:"index"`
							ID       string `json:"id"`
							Type     string `json:"type"`
							Function struct {
								Name      string `json:"name"`
								Arguments string `json:"arguments"`
							} `json:"function"`
						} `json:"tool_calls"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}
			for _, c := range chunk.Choices {
				if c.Delta.Content != "" {
					ch <- StreamEvent{Type: StreamToken, Data: c.Delta.Content}
				}
				for _, tc := range c.Delta.ToolCalls {
					if tc.Function.Name != "" {
						ch <- StreamEvent{Type: StreamToolCallBegin, Index: tc.Index, ToolID: tc.ID, ToolName: tc.Function.Name}
					}
					if tc.Function.Arguments != "" {
						ch <- StreamEvent{Type: StreamToolCallDelta, Index: tc.Index, Data: tc.Function.Arguments}
					}
				}
			}
		}
		ch <- StreamEvent{Type: StreamDone, Done: true}
	}()

	return ch, nil
}

func (p *OpenCodeProvider) ModelInfo(model string) (*ModelInfo, error) {
	return &ModelInfo{
		Provider:       "opencode",
		Name:           model,
		SupportsStream: true,
		SupportsTools:  true,
		SupportsJSON:   false,
		MaxTokens:      4096,
		ContextWindow:  128000,
	}, nil
}
