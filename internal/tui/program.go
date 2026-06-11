package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/routerforge/cli/internal/agent"
	"github.com/routerforge/cli/internal/event"
	"github.com/routerforge/cli/internal/orchestrator"
)

type Program struct {
	model *Model
	hm    *orchestrator.HeadManager

	mu           sync.RWMutex
	agentTabMap  map[string]string
	teamTabMap   map[string]string
}

func NewProgram(hm *orchestrator.HeadManager) *Program {
	m := NewModel()
	p := &Program{
		model:       m,
		hm:          hm,
		agentTabMap: make(map[string]string),
		teamTabMap:  make(map[string]string),
	}

	m.onFirstMessage = func(description string) {
		proj := p.hm.Project()
		proj.Name = extractName(description)
		proj.Goal = description
		proj.Description = description
		if proj.TechStack == "" {
			proj.TechStack = "Go, React, TypeScript"
		}

		p.hm.SetUserProxy(agent.NewSilentUserProxy())

		p.model.mu.Lock()
		p.model.addLineToTab("head_manager", Line{
			Time: time.Now().Format("15:04:05"),
			Text: fmt.Sprintf("Project: %s | %s", proj.Name, proj.Goal),
			Type: LineInfo,
		})
		p.model.addLineToTab("team-chat", Line{
			Time: time.Now().Format("15:04:05"),
			Text: fmt.Sprintf("Project initialized: %s", proj.Name),
			Type: LinePhase,
		})
		p.model.mu.Unlock()

		hm.Bus().Subscribe("*", func(evt event.Event) {
			p.handleEvent(evt)
		})

		p.hm.RunFullPipeline()
	}

	return p
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
			Text: fmt.Sprintf("Phase transition: %s", phase),
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
			Text: fmt.Sprintf("%s Agent %s: Starting task — %s",
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
			Text: fmt.Sprintf("%s Agent %s: Task complete ✓",
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
			Text: fmt.Sprintf("%s Agent %s: Task failed ✗ — %s",
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
