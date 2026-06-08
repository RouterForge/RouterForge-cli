package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type PhaseStatus struct {
	Name   string
	Active bool
	Done   bool
}

type AgentStatus struct {
	ID     string
	Role   string
	Status string
}

type LogEntry struct {
	Time    time.Time
	Message string
}

type Model struct {
	mu      sync.RWMutex
	phase   string
	phases  []PhaseStatus
	agents  []AgentStatus
	logs    []LogEntry
	width   int
	height  int
	ready   bool
	quitting bool
	err     error
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7C3AED")).
			Padding(0, 1)

	phaseStyle = lipgloss.NewStyle().
			Width(20).
			Padding(0, 1)

	activePhaseStyle = lipgloss.NewStyle().
				Width(20).
				Padding(0, 1).
				Background(lipgloss.Color("#7C3AED")).
				Foreground(lipgloss.Color("#FFFFFF"))

	donePhaseStyle = lipgloss.NewStyle().
			Width(20).
			Padding(0, 1).
			Foreground(lipgloss.Color("#10B981"))

	agentRunningStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#3B82F6"))

	agentDoneStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#10B981"))

	agentFailedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#EF4444"))

	logStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9CA3AF"))

	logStyle2 = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#D1D5DB"))

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			Padding(0, 1)
)

func NewModel() *Model {
	return &Model{
		phases: []PhaseStatus{
			{Name: "Idle"},
			{Name: "Understand"},
			{Name: "Design"},
			{Name: "Execute"},
			{Name: "Review"},
		},
	}
}

func (m *Model) SetPhase(p string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.phase = p
	for i := range m.phases {
		if m.phases[i].Name == p {
			m.phases[i].Active = true
		} else if m.phases[i].Active {
			m.phases[i].Active = false
			m.phases[i].Done = true
		}
	}
}

func (m *Model) AddAgent(id, role, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.agents = append(m.agents, AgentStatus{ID: id, Role: role, Status: status})
}

func (m *Model) UpdateAgent(id, status string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.agents {
		if m.agents[i].ID == id {
			m.agents[i].Status = status
			return
		}
	}
}

func (m *Model) AddLog(msg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, LogEntry{Time: time.Now(), Message: msg})
	if len(m.logs) > 100 {
		m.logs = m.logs[len(m.logs)-100:]
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m *Model) View() string {
	if !m.ready {
		return "\n  Loading..."
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var b strings.Builder
	b.WriteString(titleStyle.Render(" RouterForge "))
	b.WriteString("\n\n")

	b.WriteString(renderPhases(m.phases))
	b.WriteString("\n")

	b.WriteString(renderAgents(m.agents))
	b.WriteString("\n")

	b.WriteString(renderLogs(m.logs, m.width))
	b.WriteString("\n")
	b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render(" q:quit "))

	return b.String()
}

func renderPhases(phases []PhaseStatus) string {
	var parts []string
	for _, p := range phases {
		switch {
		case p.Done:
			parts = append(parts, donePhaseStyle.Render("✅ "+p.Name))
		case p.Active:
			parts = append(parts, activePhaseStyle.Render("▶ "+p.Name))
		default:
			parts = append(parts, phaseStyle.Render("  "+p.Name))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

func renderAgents(agents []AgentStatus) string {
	if len(agents) == 0 {
		return borderStyle.Render(" No agents yet ")
	}
	var lines []string
	lines = append(lines, " Agents:")
	for _, a := range agents {
		var s string
		switch a.Status {
		case "active", "running":
			s = agentRunningStyle.Render(fmt.Sprintf("  ◉ %s (%s)", a.Role, a.Status))
		case "completed":
			s = agentDoneStyle.Render(fmt.Sprintf("  ✅ %s (%s)", a.Role, a.Status))
		case "failed":
			s = agentFailedStyle.Render(fmt.Sprintf("  ✗ %s (%s)", a.Role, a.Status))
		default:
			s = fmt.Sprintf("  ○ %s (%s)", a.Role, a.Status)
		}
		lines = append(lines, s)
	}
	return borderStyle.Render(strings.Join(lines, "\n"))
}

func renderLogs(logs []LogEntry, width int) string {
	if len(logs) == 0 {
		return borderStyle.Render(" No logs yet ")
	}
	show := logs
	if len(show) > 10 {
		show = show[len(show)-10:]
	}
	var lines []string
	lines = append(lines, " Logs:")
	for _, l := range show {
		t := l.Time.Format("15:04:05")
		msg := l.Message
		if len(msg) > width-20 {
			msg = msg[:width-23] + "..."
		}
		lines = append(lines, logStyle.Render(fmt.Sprintf("  %s", t))+" "+logStyle2.Render(msg))
	}
	return borderStyle.Render(strings.Join(lines, "\n"))
}
