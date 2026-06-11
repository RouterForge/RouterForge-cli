package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/routerforge/cli/internal/agent"
	"github.com/routerforge/cli/internal/engine"
	"github.com/routerforge/cli/internal/event"
	"github.com/routerforge/cli/internal/orchestrator"
)

type Program struct {
	model *Model
	hm    *orchestrator.HeadManager

	mu            sync.RWMutex
	agentTabMap   map[string]string
	teamTabMap    map[string]string
	chatHistory   []engine.ChatMessage
	projectReady  bool
	pipelineLaunched bool
}

func NewProgram(hm *orchestrator.HeadManager) *Program {
	m := NewModel()
	p := &Program{
		model:       m,
		hm:          hm,
		agentTabMap: make(map[string]string),
		teamTabMap:  make(map[string]string),
	}

	m.onFirstMessage = func(text string) {
		p.model.mu.Lock()
		if strings.HasPrefix(text, "/") {
			p.handleCommand(text)
			p.model.mu.Unlock()
			return
		}
		if p.model.pipelineRunning {
			p.model.addLineToTab("head_manager", Line{
				Time: time.Now().Format("15:04:05"),
				Text: "Pipeline is already running. Please wait...",
				Type: LineInfo,
			})
			p.model.mu.Unlock()
			return
		}
		p.model.mu.Unlock()

		systemPrompt := `You are RouterForge Head Manager, an AI project architect.

Your job: determine if the user wants to chat or build a project.

- If greeting or casual chat: respond warmly and ask if they want to build something. Keep it brief.
- If they want to build a project: ask clarifying questions one at a time. Gather: project type, tech stack, main features, special requirements.
- When you have enough info to build: include "---READY---" on its own line, then summarize what you understood.
- Be concise. Ask ONE question at a time.`

		var historyBuilder strings.Builder
		for _, msg := range p.chatHistory {
			historyBuilder.WriteString(msg.Role + ": " + msg.Content + "\n")
		}
		historyBuilder.WriteString("user: " + text)
		userPrompt := historyBuilder.String()

		llm := engine.NewLLMClient(p.hm.Model())
		llm.AgentID = "head_manager"
		llm.Phase = "chat"
		response, err := llm.Chat(systemPrompt, userPrompt)
		if err != nil {
			p.model.mu.Lock()
			p.model.addLineToTab("head_manager", Line{
				Time: time.Now().Format("15:04:05"),
				Text: fmt.Sprintf("Error: %s", err.Error()),
				Type: LineError,
			})
			p.model.mu.Unlock()
			return
		}

		p.chatHistory = append(p.chatHistory,
			engine.ChatMessage{Role: "user", Content: text},
			engine.ChatMessage{Role: "assistant", Content: response})

		if strings.Contains(response, "---READY---") {
			p.launchPipeline(text, response)
		} else {
			p.model.mu.Lock()
			p.model.addLineToTab("head_manager", Line{
				Time: time.Now().Format("15:04:05"),
				Text: response,
				Type: LineChat,
			})
			p.model.mu.Unlock()
		}
	}

	return p
}

func (p *Program) handleCommand(text string) {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return
	}
	cmd := strings.ToLower(parts[0])
	args := parts[1:]

	ts := time.Now().Format("15:04:05")

	switch cmd {
	case "/help", "/?":
		help := "" +
			"  /help, /h, /?   show this help\n" +
			"  /exit, /quit, /q quit RouterForge\n" +
			"  /reload, /reset  restart the session\n" +
			"  /cancel, /stop   cancel running pipeline\n" +
			"  /clear           clear current tab\n\n" +
			"  /build <desc>    start project pipeline\n" +
			"  /plan            run planning phase\n" +
			"  /design          run design phase\n" +
			"  /execute         run execute phase\n" +
			"  /review          run review phase\n" +
			"  /repair          run repair phase\n" +
			"  /deploy          run deploy checks\n\n" +
			"  /chat            switch to chat mode\n" +
			"  /project         switch to project mode\n" +
			"  /research <q>    research mode (soon)\n" +
			"  /learn           interactive tutorial\n\n" +
			"  /status          show system state\n" +
			"  /phase           show current phase\n" +
			"  /agents          list active agents\n" +
			"  /tasks           list task progress\n" +
			"  /cost            show cost breakdown\n" +
			"  /tokens          show token usage\n" +
			"  /uptime          show session uptime\n" +
			"  /version         show RouterForge version\n\n" +
			"  /models          list available models\n" +
			"  /model <name>    switch AI model\n" +
			"  /provider <name> switch AI provider\n\n" +
			"  /next            next tab\n" +
			"  /prev            previous tab\n" +
			"  /tab <n>         jump to tab number\n" +
			"  /home            go to head manager\n" +
			"  /teamchat        go to team chat\n" +
			"  /activity        go to activity tab\n" +
			"  /code            go to code stream\n\n" +
			"  /save            save session state\n" +
			"  /export          export conversation\n" +
			"  /inspect         inspect artifacts\n" +
			"  /analyze         analyze repository"
		p.model.addLineToTab("head_manager", Line{Time: ts, Text: help, Type: LineInfo})

	case "/exit", "/quit", "/q":
		p.model.addLineToTab("head_manager", Line{Time: ts, Text: "Shutting down...", Type: LineInfo})
		p.model.quit = true
		go func() { time.Sleep(100 * time.Millisecond); tea.Quit() }()

	case "/reload", "/reset":
		p.chatHistory = nil
		p.model.pipelineRunning = false
		p.pipelineLaunched = false
		p.model.phase = ""
		p.model.addLineToTab("head_manager", Line{Time: ts, Text: "Session reloaded.", Type: LinePhase})

	case "/cancel", "/stop":
		if !p.model.pipelineRunning {
			p.model.addLineToTab("head_manager", Line{Time: ts, Text: "No pipeline running.", Type: LineInfo})
			return
		}
		p.model.pipelineRunning = false
		p.pipelineLaunched = false
		p.model.phase = ""
		p.model.addLineToTab("head_manager", Line{Time: ts, Text: "Pipeline cancelled.", Type: LineWarning})

	case "/clear":
		p.model.tabs[p.model.activeTab].Lines = nil
		p.model.tabs[p.model.activeTab].Scroll = 0
		p.model.addLineToTab("head_manager", Line{Time: ts, Text: "Cleared.", Type: LineInfo})

	case "/build":
		if p.model.pipelineRunning {
			p.model.addLineToTab("head_manager", Line{Time: ts, Text: "Pipeline already running.", Type: LineInfo})
			return
		}
		desc := strings.Join(args, " ")
		if desc == "" {
			p.model.addLineToTab("head_manager", Line{Time: ts, Text: "Usage: /build <project description>\nExample: /build a REST API in Go", Type: LineInfo})
			return
		}
		p.model.mu.Unlock()
		p.launchPipeline(desc, desc)
		p.model.mu.Lock()

	case "/plan", "/design", "/execute", "/review", "/repair", "/deploy":
		phase := strings.TrimPrefix(cmd, "/")
		if p.model.pipelineRunning {
			p.model.addLineToTab("head_manager", Line{Time: ts, Text: fmt.Sprintf("Pipeline already running. Use /cancel first."), Type: LineInfo})
			return
		}
		p.model.addLineToTab("head_manager", Line{Time: ts, Text: fmt.Sprintf("Starting %s phase... Use /build for full pipeline.", phase), Type: LinePhase})
		p.model.mu.Unlock()
		p.launchPipeline(phase+" phase", phase+" phase")
		p.model.mu.Lock()

	case "/chat":
		p.chatHistory = nil
		p.model.addLineToTab("head_manager", Line{Time: ts, Text: "Chat mode. Ask me anything.", Type: LineChat})

	case "/project":
		p.chatHistory = nil
		p.model.addLineToTab("head_manager", Line{Time: ts, Text: "Project mode. Describe what you want to build.", Type: LinePhase})

	case "/research":
		q := strings.Join(args, " ")
		if q == "" {
			p.model.addLineToTab("head_manager", Line{Time: ts, Text: "Usage: /research <query>\nExample: /research AI agents for code review", Type: LineInfo})
			return
		}
		p.model.addLineToTab("head_manager", Line{Time: ts, Text: fmt.Sprintf("Researching: %s\nComing soon.", q), Type: LineInfo})

	case "/learn":
		p.model.addLineToTab("head_manager", Line{Time: ts, Text: "" +
			"RouterForge Quick Tour:\n\n" +
			"1. Type a message or use /build to start a project\n" +
			"2. Watch agents form in Team Chat tab\n" +
			"3. See tools run in Activity tab\n" +
			"4. View generated code in Code tab\n" +
			"5. Use /help anytime for command list",
			Type: LineInfo})

	case "/status":
		s := "Idle"
		if p.model.pipelineRunning {
			s = "Running"
		}
		agents := 0
		for _, t := range p.model.tabs {
			if t.AgentType == "team" || t.AgentType == "micro" {
				agents++
			}
		}
		p.model.addLineToTab("head_manager", Line{Time: ts,
			Text: fmt.Sprintf("Status: %s | Agents: %d | Tabs: %d | Phase: %s",
				s, agents, len(p.model.tabs), p.model.phase), Type: LineInfo})

	case "/phase":
		p.model.addLineToTab("head_manager", Line{Time: ts,
			Text: fmt.Sprintf("Current phase: %s", p.model.phase), Type: LineInfo})

	case "/agents":
		lines := []string{"Active agents:"}
		for _, t := range p.model.tabs {
			if t.AgentType == "team" || t.AgentType == "micro" {
				lines = append(lines, fmt.Sprintf("  %s %s [%s]", t.Icon, t.Title, t.Status))
			}
		}
		if len(lines) == 1 {
			lines = append(lines, "  No agents yet. Start a project with /build")
		}
		p.model.addLineToTab("head_manager", Line{Time: ts, Text: strings.Join(lines, "\n"), Type: LineInfo})

	case "/tasks":
		total, done := 0, 0
		for _, t := range p.model.tabs {
			total += t.TasksCnt
			done += t.TasksDone
		}
		p.model.addLineToTab("head_manager", Line{Time: ts,
			Text: fmt.Sprintf("Tasks: %d/%d completed", done, total), Type: LineInfo})

	case "/cost":
		p.model.addLineToTab("head_manager", Line{Time: ts,
			Text: fmt.Sprintf("Total cost: $%.4f", p.model.cost), Type: LineInfo})

	case "/tokens":
		p.model.addLineToTab("head_manager", Line{Time: ts,
			Text: fmt.Sprintf("Total tokens: %d", p.model.tokens), Type: LineInfo})

	case "/uptime":
		up := time.Since(p.model.startTime).Round(time.Second)
		p.model.addLineToTab("head_manager", Line{Time: ts,
			Text: fmt.Sprintf("Uptime: %s", up), Type: LineInfo})

	case "/version":
		p.model.addLineToTab("head_manager", Line{Time: ts,
			Text: "RouterForge 2.1.14 — AI Multi-Agent Operating System", Type: LineInfo})

	case "/models":
		p.model.addLineToTab("head_manager", Line{Time: ts,
			Text: fmt.Sprintf("Current: %s\n\nFree models:\n  big-pickle, deepseek-v4-flash-free\n  mimo-v2.5-free, nemotron-3-super-free\n  nemotron-3-ultra-free\n\nUse /model <name> to switch.", p.hm.Model()), Type: LineInfo})

	case "/model":
		if len(args) == 0 {
			p.model.addLineToTab("head_manager", Line{Time: ts,
				Text: fmt.Sprintf("Current model: %s\nUse: /model <name>", p.hm.Model()), Type: LineInfo})
			return
		}
		newModel := args[0]
		p.model.addLineToTab("head_manager", Line{Time: ts,
			Text: fmt.Sprintf("Model switching coming soon. Requested: %s", newModel), Type: LineInfo})

	case "/provider":
		if len(args) == 0 {
			p.model.addLineToTab("head_manager", Line{Time: ts,
				Text: "Current provider: OpenCode\nAvailable: opencode, openai", Type: LineInfo})
			return
		}
		p.model.addLineToTab("head_manager", Line{Time: ts,
			Text: fmt.Sprintf("Provider switching coming soon. Requested: %s", args[0]), Type: LineInfo})

	case "/next", "/n":
		if len(p.model.tabs) > 1 {
			p.model.activeTab = (p.model.activeTab + 1) % len(p.model.tabs)
		}

	case "/prev", "/p":
		if len(p.model.tabs) > 1 {
			p.model.activeTab = (p.model.activeTab - 1 + len(p.model.tabs)) % len(p.model.tabs)
		}

	case "/tab", "/t":
		if len(args) == 0 {
			p.model.addLineToTab("head_manager", Line{Time: ts,
				Text: fmt.Sprintf("Current tab: %d/%d", p.model.activeTab+1, len(p.model.tabs)), Type: LineInfo})
			return
		}
		n := 0
		fmt.Sscanf(args[0], "%d", &n)
		if n > 0 && n <= len(p.model.tabs) {
			p.model.activeTab = n - 1
		}

	case "/home", "/head", "/h":
		p.model.activeTab = 0

	case "/teamchat", "/team":
		if len(p.model.tabs) > 1 {
			p.model.activeTab = 1
		}

	case "/activity", "/act":
		if len(p.model.tabs) > 2 {
			p.model.activeTab = 2
		}

	case "/code", "/stream":
		if len(p.model.tabs) > 3 {
			p.model.activeTab = 3
		}

	case "/save":
		p.model.addLineToTab("head_manager", Line{Time: ts,
			Text: "Session saved.", Type: LineSuccess})

	case "/export":
		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("RouterForge Chat Export\n%s\n\n", time.Now().Format("2006-01-02 15:04:05")))
		for _, line := range p.model.tabs[0].Lines {
			sb.WriteString(fmt.Sprintf("[%s] %s\n", line.Time, line.Text))
		}
		p.model.addLineToTab("head_manager", Line{Time: ts,
			Text: "Conversation exported to clipboard (simulated). Use /save for persistence.", Type: LineInfo})

	case "/inspect":
		p.model.addLineToTab("head_manager", Line{Time: ts,
			Text: "Use: routerforge inspect (CLI command)", Type: LineInfo})

	case "/analyze":
		p.model.addLineToTab("head_manager", Line{Time: ts,
			Text: "Use: routerforge analyze <path> (CLI command)", Type: LineInfo})

	default:
		p.model.addLineToTab("head_manager", Line{Time: ts,
			Text: fmt.Sprintf("Unknown: %s\nType /help for command list.", cmd), Type: LineWarning})
	}
}

func (p *Program) launchPipeline(text string, llmResponse string) {
	p.model.mu.Lock()
	if p.pipelineLaunched {
		p.model.mu.Unlock()
		return
	}
	p.pipelineLaunched = true
	p.model.pipelineRunning = true

	parts := strings.SplitN(llmResponse, "---READY---", 2)
	summary := strings.TrimSpace(parts[len(parts)-1])
	if summary == "" {
		summary = text
	}

	p.model.addLineToTab("head_manager", Line{
		Time: time.Now().Format("15:04:05"),
		Text: fmt.Sprintf("Starting project: %s", summary),
		Type: LinePhase,
	})
	p.model.mu.Unlock()

	proj := p.hm.Project()
	proj.Name = extractName(text)
	proj.Goal = summary
	proj.Description = summary
	if proj.TechStack == "" {
		proj.TechStack = "Go, React, TypeScript"
	}

	p.hm.SetUserProxy(agent.NewSilentUserProxy())

	p.model.mu.Lock()
	p.model.addLineToTab("team-chat", Line{
		Time: time.Now().Format("15:04:05"),
		Text: fmt.Sprintf("Project initialized: %s", proj.Name),
		Type: LinePhase,
	})
	p.model.mu.Unlock()

	p.hm.Bus().Subscribe("*", func(evt event.Event) {
		p.handleEvent(evt)
	})

	p.hm.RunFullPipeline()
}

func (p *Program) tabForID(id string) string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if id == "" || id == "head_manager" {
		return "head_manager"
	}
	if tab, ok := p.agentTabMap[id]; ok {
		return tab
	}
	if tab, ok := p.teamTabMap[id]; ok {
		return tab
	}
	lookup := strings.TrimPrefix(id, "team-")
	for k, v := range p.teamTabMap {
		if strings.Contains(id, k) || strings.Contains(k, id) {
			return v
		}
		if lookup != "" && (strings.Contains(v, lookup) || strings.Contains(lookup, v)) {
			return v
		}
	}
	return "head_manager"
}

func (p *Program) addLine(tabID string, line Line) {
	p.model.mu.Lock()
	defer p.model.mu.Unlock()
	p.model.addLineToTab(tabID, line)
}

func (p *Program) handleEvent(evt event.Event) {
	ts := time.Now().Format("15:04:05")
	source := evt.Source
	payload := fmt.Sprintf("%v", evt.Payload)

	switch evt.Type {
	case event.EvtPhaseChanged:
		phase := payload
		if m, ok := evt.Payload.(map[string]string); ok {
			phase = m["phase"]
		}
		p.model.mu.Lock()
		p.model.phase = phase
		p.model.mu.Unlock()
		p.addLine("head_manager", Line{
			Time: ts,
			Text: fmt.Sprintf("Phase: %s", phase),
			Type: LinePhase,
		})

	case event.EvtAgentCreated:
		if strings.Contains(payload, "team:") {
			parts := strings.SplitN(payload, " ", 3)
			if len(parts) >= 2 {
				domain := strings.TrimPrefix(parts[1], "team:")
				domain = strings.TrimSpace(domain)
				if domain != "" {
					tabID := fmt.Sprintf("team-%s", domain)
					title := fmt.Sprintf("%s Team", capitalize(domain))

					p.mu.Lock()
					p.teamTabMap[tabID] = tabID
					p.mu.Unlock()

					p.model.mu.Lock()
					p.model.createTab(tabID, title, domain, "team", "head_manager")
					p.model.mu.Unlock()

					p.addLine(tabID, Line{
						Time: ts,
						Text: fmt.Sprintf("Team %s created and ready.", domain),
						Type: LineSuccess,
					})
					p.addLine("team-chat", Line{
						Time: ts,
						Text: fmt.Sprintf("\U0001F91D %s team assembled", capitalize(domain)),
						Type: LineAgentChat,
					})
				}
			}
		} else if strings.Contains(payload, "micro-agent") && source != "" {
			p.mu.RLock()
			parentTab, ok := p.agentTabMap[source]
			if !ok {
				for k, v := range p.teamTabMap {
					if strings.Contains(source, k) || strings.Contains(k, source) {
						parentTab = v
						break
					}
				}
			}
			if !ok {
				parentTab = "head_manager"
			}
			p.mu.RUnlock()

			role := strings.TrimPrefix(payload, "micro-agent ")
			role = strings.TrimSuffix(role, " started")
			role = strings.TrimSpace(role)

			p.mu.Lock()
			p.agentTabMap[source] = parentTab
			p.mu.Unlock()

			p.addLine(parentTab, Line{
				Time: ts,
				Text: fmt.Sprintf("Agent online: %s", role),
				Type: LineSuccess,
			})
			p.addLine("team-chat", Line{
				Time: ts,
				Text: fmt.Sprintf("%s Agent %s: Ready to work", agentRoleIcon(role), capitalize(role)),
				Type: LineAgentChat,
			})
		} else {
			p.addLine("head_manager", Line{
				Time: ts,
				Text: fmt.Sprintf("Agent: %s", payload),
				Type: LineInfo,
			})
		}

	case event.EvtTaskStarted:
		p.addLine("team-chat", Line{
			Time: ts,
			Text: fmt.Sprintf("%s Agent %s: Starting task \u2014 %s",
				agentRoleIcon(p.roleForID(source)),
				capitalize(p.roleForID(source)),
				payload),
			Type: LineAgentChat,
		})
		tabID := p.tabForID(source)
		p.addLine(tabID, Line{
			Time: ts,
			Text: fmt.Sprintf("Task started: %s", payload),
			Type: LineInfo,
		})

	case event.EvtTaskCompleted:
		p.addLine("team-chat", Line{
			Time: ts,
			Text: fmt.Sprintf("%s Agent %s: Task complete \u2713",
				agentRoleIcon(p.roleForID(source)),
				capitalize(p.roleForID(source))),
			Type: LineAgentChat,
		})
		tabID := p.tabForID(source)
		p.addLine(tabID, Line{
			Time: ts,
			Text: fmt.Sprintf("Task completed: %s", payload),
			Type: LineSuccess,
		})

		p.model.mu.Lock()
		for i := range p.model.tabs {
			if p.model.tabs[i].ID == tabID {
				p.model.tabs[i].TasksDone++
				break
			}
		}
		p.model.mu.Unlock()

	case event.EvtTaskFailed:
		p.addLine("team-chat", Line{
			Time: ts,
			Text: fmt.Sprintf("%s Agent %s: Task failed \u2717 \u2014 %s",
				agentRoleIcon(p.roleForID(source)),
				capitalize(p.roleForID(source)),
				payload),
			Type: LineAgentChat,
		})
		tabID := p.tabForID(source)
		p.addLine(tabID, Line{
			Time: ts,
			Text: fmt.Sprintf("Task failed: %s", payload),
			Type: LineError,
		})

	case event.EvtEscalation:
		p.addLine("head_manager", Line{
			Time: ts,
			Text: fmt.Sprintf("Escalation: %s", payload),
			Type: LineWarning,
		})
		p.addLine("team-chat", Line{
			Time: ts,
			Text: fmt.Sprintf("\u26A0\uFE0F Agent %s escalated: %s",
				capitalize(p.roleForID(source)), payload),
			Type: LineWarning,
		})

	case event.EvtArtifactCreated:
		p.addLine("code-stream", Line{
			Time: ts,
			Text: fmt.Sprintf("%s %s", agentRoleIcon(p.roleForID(source)), payload),
			Type: LineArtifact,
		})
		tabID := p.tabForID(source)
		if tabID != "code-stream" {
			p.addLine(tabID, Line{
				Time: ts,
				Text: fmt.Sprintf("Generated: %s", payload),
				Type: LineArtifact,
			})
		}

	case event.EvtToolExecuted:
		p.addLine("activity", Line{
			Time: ts,
			Text: fmt.Sprintf("\U0001F527 [%s] %s", source, payload),
			Type: LineTool,
		})

	case event.EvtModelCalled:
		p.model.mu.Lock()
		p.model.logInternal(fmt.Sprintf("model[%s]: %s", source, payload))
		p.model.mu.Unlock()
		p.addLine("team-chat", Line{
			Time: ts,
			Text: fmt.Sprintf("%s is thinking...", capitalize(p.roleForID(source))),
			Type: LineThinking,
		})

	case event.EvtAgentDied:
		tabID := p.tabForID(source)
		p.addLine(tabID, Line{
			Time: ts,
			Text: fmt.Sprintf("Agent failed: %s", payload),
			Type: LineError,
		})
		p.addLine("team-chat", Line{
			Time: ts,
			Text: fmt.Sprintf("\u274C Agent %s failed: %s",
				capitalize(p.roleForID(source)), payload),
			Type: LineError,
		})
	}
}

func (p *Program) roleForID(id string) string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if tab, ok := p.teamTabMap[id]; ok {
		return strings.TrimPrefix(tab, "team-")
	}
	if tab, ok := p.agentTabMap[id]; ok {
		return strings.TrimPrefix(tab, "team-")
	}
	return id
}

func (p *Program) Run() error {
	prog := tea.NewProgram(p.model)
	_, err := prog.Run()
	return err
}

func capitalize(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func extractName(description string) string {
	words := strings.Fields(description)
	if len(words) > 5 {
		words = words[:5]
	}
	name := strings.Join(words, " ")
	return name
}


