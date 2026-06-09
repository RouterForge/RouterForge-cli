package tool

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/routerforge/cli/internal/engine"
)

var ErrUnknownTool = errors.New("unknown tool")

type Tool interface {
	Name() string
	Description() string
	Schema() json.RawMessage
	Execute(ctx context.Context, params json.RawMessage) (string, error)
}

type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

func (r *Registry) Register(t Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[t.Name()] = t
}

func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

func (r *Registry) Definitions() []interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]interface{}, 0, len(r.tools))
	for _, t := range r.tools {
		var schema interface{}
		if err := json.Unmarshal(t.Schema(), &schema); err != nil {
			continue
		}
		out = append(out, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        t.Name(),
				"description": t.Description(),
				"parameters":  schema,
			},
		})
	}
	return out
}

type PermissionRule struct {
	ToolPattern string `json:"tool_pattern"`
	Effect      string `json:"effect"`
}

type PermissionEvaluator struct {
	rules []PermissionRule
}

func NewPermissionEvaluator() *PermissionEvaluator {
	return &PermissionEvaluator{}
}

func (p *PermissionEvaluator) AddRule(r PermissionRule) {
	p.rules = append(p.rules, r)
}

func (p *PermissionEvaluator) Evaluate(toolName string) string {
	for _, r := range p.rules {
		if r.ToolPattern == toolName {
			return r.Effect
		}
	}
	return "allow"
}

func (p *PermissionEvaluator) CanExecute(toolName string) bool {
	return p.Evaluate(toolName) != "deny"
}

type registeredTool struct {
	name        string
	description string
	schema      json.RawMessage
	run         func(ctx context.Context, params json.RawMessage) (string, error)
}

func (t *registeredTool) Name() string                        { return t.name }
func (t *registeredTool) Description() string                 { return t.description }
func (t *registeredTool) Schema() json.RawMessage              { return t.schema }
func (t *registeredTool) Execute(ctx context.Context, params json.RawMessage) (string, error) {
	return t.run(ctx, params)
}

type toolParams struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout"`
	Path    string `json:"path"`
	Content string `json:"content"`
	Pattern string `json:"pattern"`
	URL     string `json:"url"`
}

func sk(n, d string, s json.RawMessage, fn func(ctx context.Context, p json.RawMessage) (string, error)) Tool {
	return &registeredTool{name: n, description: d, schema: s, run: fn}
}

func RegisterAll(r *Registry) {
	registerShell(r)
	registerFile(r)
	registerSearch(r)
	registerWebFetch(r)
	registerAskUser(r)
	registerWebSearch(r)
	registerAPICall(r)
	registerBrowserAction(r)
}

func registerShell(r *Registry) {
	r.Register(sk("run_command", "Execute a shell command and return its output",
		json.RawMessage(`{"type":"object","properties":{"command":{"type":"string"},"timeout":{"type":"integer","default":30}},"required":["command"]}`),
		func(ctx context.Context, raw json.RawMessage) (string, error) {
			var p toolParams
			json.Unmarshal(raw, &p)
			if p.Timeout <= 0 { p.Timeout = 30 }
			cmdCtx, cancel := context.WithTimeout(ctx, time.Duration(p.Timeout)*time.Second)
			defer cancel()
			cmd := exec.CommandContext(cmdCtx, "bash", "-c", p.Command)
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Sprintf("exit: %v\n%s", err, string(out)), nil
			}
			return strings.TrimSpace(string(out)), nil
		},
	))
}

func registerFile(r *Registry) {
	r.Register(sk("read_file", "Read the contents of a file",
		json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"}},"required":["path"]}`),
		func(ctx context.Context, raw json.RawMessage) (string, error) {
			var p toolParams
			json.Unmarshal(raw, &p)
			data, err := os.ReadFile(p.Path)
			if err != nil {
				return "", err
			}
			return string(data), nil
		},
	))

	r.Register(sk("write_file", "Write content to a file",
		json.RawMessage(`{"type":"object","properties":{"path":{"type":"string"},"content":{"type":"string"}},"required":["path","content"]}`),
		func(ctx context.Context, raw json.RawMessage) (string, error) {
			var p toolParams
			json.Unmarshal(raw, &p)
			os.MkdirAll(filepath.Dir(p.Path), 0755)
			if err := os.WriteFile(p.Path, []byte(p.Content), 0644); err != nil {
				return "", err
			}
			return fmt.Sprintf("wrote %d bytes to %s", len(p.Content), p.Path), nil
		},
	))
}

func registerSearch(r *Registry) {
	r.Register(sk("search_code", "Search for text patterns in the codebase",
		json.RawMessage(`{"type":"object","properties":{"pattern":{"type":"string"},"path":{"type":"string","default":"."}},"required":["pattern"]}`),
		func(ctx context.Context, raw json.RawMessage) (string, error) {
			var p toolParams
			json.Unmarshal(raw, &p)
			if p.Path == "" { p.Path = "." }
			cmd := exec.CommandContext(ctx, "grep", "-rn", p.Pattern, p.Path)
			out, err := cmd.CombinedOutput()
			if err != nil {
				if len(out) > 0 { return string(out), nil }
				return "no matches found", nil
			}
			return strings.TrimSpace(string(out)), nil
		},
	))

	r.Register(sk("glob_files", "Find files matching a glob pattern",
		json.RawMessage(`{"type":"object","properties":{"pattern":{"type":"string"},"path":{"type":"string","default":"."}},"required":["pattern"]}`),
		func(ctx context.Context, raw json.RawMessage) (string, error) {
			var p toolParams
			json.Unmarshal(raw, &p)
			if p.Path == "" { p.Path = "." }
			matches, _ := filepath.Glob(filepath.Join(p.Path, p.Pattern))
			if len(matches) == 0 { return "no files matched", nil }
			return strings.Join(matches, "\n"), nil
		},
	))
}

func registerWebFetch(r *Registry) {
	r.Register(sk("web_fetch", "Fetch a URL and return its content as text",
		json.RawMessage(`{"type":"object","properties":{"url":{"type":"string"},"timeout":{"type":"integer","default":15}},"required":["url"]}`),
		func(ctx context.Context, raw json.RawMessage) (string, error) {
			var p toolParams
			json.Unmarshal(raw, &p)
			if p.Timeout <= 0 { p.Timeout = 15 }
			reqCtx, cancel := context.WithTimeout(ctx, time.Duration(p.Timeout)*time.Second)
			defer cancel()
			req, _ := http.NewRequestWithContext(reqCtx, "GET", p.URL, nil)
			req.Header.Set("User-Agent", "RouterForge/1.0")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return "", err
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
			return fmt.Sprintf("HTTP %d\n\n%s", resp.StatusCode, string(body)), nil
		},
	))
}

func registerAskUser(r *Registry) {
	r.Register(sk("ask_user", "Ask the user a question for input",
		json.RawMessage(`{"type":"object","properties":{"question":{"type":"string"}},"required":["question"]}`),
		func(ctx context.Context, raw json.RawMessage) (string, error) {
			var q struct{ Question string `json:"question"` }
			json.Unmarshal(raw, &q)
			fmt.Print("\n🤔 " + q.Question + " [y/N]: ")
			var answer string
			fmt.Scanln(&answer)
			if answer == "" { answer = "no" }
			return "User response: " + answer, nil
		},
	))
}

func registerWebSearch(r *Registry) {
	r.Register(sk("web_search", "Search the web and return results",
		json.RawMessage(`{"type":"object","properties":{"query":{"type":"string"},"count":{"type":"integer","default":5}},"required":["query"]}`),
		func(ctx context.Context, raw json.RawMessage) (string, error) {
			var p struct {
				Query string `json:"query"`
				Count int    `json:"count"`
			}
			json.Unmarshal(raw, &p)
			if p.Count <= 0 { p.Count = 5 }

			req, _ := http.NewRequestWithContext(ctx, "GET",
				fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1", url.QueryEscape(p.Query)), nil)
			req.Header.Set("User-Agent", "RouterForge/1.0")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return "", fmt.Errorf("search failed: %w", err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
			return fmt.Sprintf("Search results for '%s':\n%s", p.Query, string(body)), nil
		},
	))
}

func registerAPICall(r *Registry) {
	r.Register(sk("api_call", "Make an HTTP API request to an external service",
		json.RawMessage(`{"type":"object","properties":{"url":{"type":"string"},"method":{"type":"string","default":"GET"},"headers":{"type":"object","default":{}},"body":{"type":"string","default":""}},"required":["url"]}`),
		func(ctx context.Context, raw json.RawMessage) (string, error) {
			var p struct {
				URL     string            `json:"url"`
				Method  string            `json:"method"`
				Headers map[string]string `json:"headers"`
				Body    string            `json:"body"`
			}
			json.Unmarshal(raw, &p)
			if p.Method == "" { p.Method = "GET" }

			var reqBody io.Reader
			if p.Body != "" {
				reqBody = strings.NewReader(p.Body)
			}
			req, err := http.NewRequestWithContext(ctx, p.Method, p.URL, reqBody)
			if err != nil {
				return "", fmt.Errorf("request failed: %w", err)
			}
			for k, v := range p.Headers {
				req.Header.Set(k, v)
			}
			req.Header.Set("User-Agent", "RouterForge/1.0")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return "", fmt.Errorf("api call failed: %w", err)
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
			return fmt.Sprintf("HTTP %d\n\n%s", resp.StatusCode, string(body)), nil
		},
	))
}

func registerBrowserAction(r *Registry) {
	r.Register(sk("browser_action", "Control a headless browser (navigate, click, type, screenshot, html, evaluate)",
		json.RawMessage(`{"type":"object","properties":{"action":{"type":"string","enum":["navigate","click","type","screenshot","html","evaluate","cookies","viewport"]},"value":{"type":"string"},"selector":{"type":"string","default":""}},"required":["action","value"]}`),
		func(ctx context.Context, raw json.RawMessage) (string, error) {
			var p struct {
				Action   string `json:"action"`
				Value    string `json:"value"`
				Selector string `json:"selector"`
			}
			json.Unmarshal(raw, &p)

			screenshotDir := filepath.Join(".", ".routerforge", "screenshots")
			be := engine.NewBrowserEngine(screenshotDir)
			if err := be.Launch(); err != nil {
				return "", fmt.Errorf("browser launch failed: %w", err)
			}
			defer be.Close()

			switch p.Action {
			case "navigate":
				if err := be.Navigate(p.Value); err != nil {
					return "", fmt.Errorf("navigate: %w", err)
				}
				return fmt.Sprintf("Navigated to %s", p.Value), nil
			case "click":
				if err := be.Click(p.Selector); err != nil {
					return "", fmt.Errorf("click: %w", err)
				}
				return fmt.Sprintf("Clicked %s", p.Selector), nil
			case "type":
				if err := be.Type(p.Selector, p.Value); err != nil {
					return "", fmt.Errorf("type: %w", err)
				}
				return fmt.Sprintf("Typed '%s' into %s", p.Value, p.Selector), nil
			case "screenshot":
				filename, err := be.Screenshot(p.Value)
				if err != nil {
					return "", fmt.Errorf("screenshot: %w", err)
				}
				return fmt.Sprintf("Screenshot saved to %s", filename), nil
			case "html":
				html, err := be.HTML()
				if err != nil {
					return "", fmt.Errorf("html: %w", err)
				}
				return fmt.Sprintf("Page HTML (%d bytes):\n%s", len(html), html[:min(len(html), 5000)]), nil
			case "evaluate":
				result, err := be.Evaluate(p.Value)
				if err != nil {
					return "", fmt.Errorf("evaluate: %w", err)
				}
				return fmt.Sprintf("Result: %v", result), nil
			case "cookies":
				cookies, err := be.GetCookies()
				if err != nil {
					return "", fmt.Errorf("cookies: %w", err)
				}
				return fmt.Sprintf("Cookies (%d): %+v", len(cookies), cookies), nil
			case "viewport":
				return "Viewport requires SetViewport(width, height). Use browser CLI.", nil
			default:
				return "", fmt.Errorf("unknown browser action: %s", p.Action)
			}
		},
	))
}
