package tui

import (
	"github.com/charmbracelet/bubbletea"
	"github.com/routerforge/cli/internal/event"
	"github.com/routerforge/cli/internal/orchestrator"
)

type pipelineMsg struct{}

type Program struct {
	model *Model
	hm    *orchestrator.HeadManager
}

func NewProgram(hm *orchestrator.HeadManager) *Program {
	m := NewModel()

	hm.Bus().Subscribe("*", func(evt event.Event) {
		switch evt.Type {
		case event.EvtPhaseChanged:
			if p, ok := evt.Payload.(map[string]string); ok {
				m.SetPhase(p["phase"])
			}
		case event.EvtAgentCreated:
			if s, ok := evt.Payload.(string); ok {
				m.AddAgent("", "", s)
			}
		case event.EvtTaskStarted:
			m.AddLog("task started: " + toString(evt.Payload))
		case event.EvtTaskCompleted:
			m.AddLog("task completed: " + toString(evt.Payload))
		case event.EvtTaskFailed:
			m.AddLog("task failed: " + toString(evt.Payload))
		case event.EvtEscalation:
			m.AddLog("escalation: " + toString(evt.Payload))
		}
	})

	return &Program{model: m, hm: hm}
}

func (p *Program) Run() error {
	go func() {
		p.hm.AttachConsoleLogger()
		p.hm.RunFullPipeline()
	}()

	prog := tea.NewProgram(p.model)
	_, err := prog.Run()
	return err
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
