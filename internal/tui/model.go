package tui

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type LineType int

const (
	LineNormal LineType = iota
	LineSuccess
	LineError
	LineWarning
	LineInfo
	LineArtifact
	LineDecision
	LineChat
	LineDivider
	LinePhase
	LineTool
	LineAgentChat
	LineThinking
)

type Line struct {
	Time string
	Text string
	Type LineType
	Tags []string
}

type AgentTab struct {
	ID        string
	Title     string
	Role      string
	AgentType string
	Status    string
	Lines     []Line
	Scroll    int
	ParentID  string
	TasksDone int
	TasksCnt  int
	Icon      string
}

type Model struct {
	mu sync.RWMutex

	tabs      []AgentTab
	activeTab int

	width  int
	height int
	ready  bool
	quit   bool

	phase     string
	lifecycle string
	cost      float64
	tokens    int

	input     string
	inputMode bool

	internalLog []string
	startTime   time.Time

	pipelineStarted bool
	onFirstMessage  func(string)
}

const systemTabCount = 4

func NewModel() *Model {
	m := &Model{
		startTime: time.Now(),
	}
	now := time.Now().Format("15:04:05")
	m.tabs = []AgentTab{
		{
			ID:    "head_manager",
			Title: "Chat",
			Icon:  "\U0001F4AC",
			Lines: []Line{
				{Time: now, Text: "RouterForge 2.0 — AI Multi-Agent Operating System", Type: LinePhase},
				{Time: now, Text: "Welcome! I'm your Head Manager.", Type: LineChat},
				{Time: now, Text: "Tell me about the project you want to build.", Type: LineChat},
				{Time: now, Text: "Press Enter, describe your idea, then press Enter again to send.", Type: LineInfo},
			},
		},
		{
			ID:    "team-chat",
			Title: "Team Chat",
			Icon:  "\U0001F91D",
			Lines: []Line{
				{Time: now, Text: "Agent conversations will appear here.", Type: LineInfo},
			},
		},
		{
			ID:    "activity",
			Title: "Activity",
			Icon:  "\u2699\uFE0F",
			Lines: []Line{
				{Time: now, Text: "Tool usage, commands, and searches appear here.", Type: LineInfo},
			},
		},
		{
			ID:    "code-stream",
			Title: "Code",
			Icon:  "\U0001F4CB",
			Lines: []Line{
				{Time: now, Text: "Generated code and file changes appear here.", Type: LineInfo},
			},
		},
	}
	return m
}

func (m *Model) Init() tea.Cmd {
	return nil
}

type UserChatMsg struct {
	AgentID string
	Text    string
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case tea.KeyMsg:
		if m.inputMode {
			return m.handleInputKey(msg)
		}
		return m.handleNormalKey(msg)

	case UserChatMsg:
		m.addLineToTab(msg.AgentID, Line{
			Time: time.Now().Format("15:04:05"),
			Text: fmt.Sprintf("You > %s", msg.Text),
			Type: LineChat,
		})
	}

	return m, nil
}

func (m *Model) handleNormalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		m.quit = true
		return m, tea.Quit
	case "enter":
		m.inputMode = true
		m.input = ""
		return m, nil
	case "tab":
		if len(m.tabs) > 1 {
			m.activeTab = (m.activeTab + 1) % len(m.tabs)
		}
		return m, nil
	case "shift+tab":
		if len(m.tabs) > 1 {
			m.activeTab = (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
		}
		return m, nil
	case "left":
		if len(m.tabs) > 1 {
			m.activeTab = (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
		}
		return m, nil
	case "right":
		if len(m.tabs) > 1 {
			m.activeTab = (m.activeTab + 1) % len(m.tabs)
		}
		return m, nil
	case "h", "H":
		m.activeTab = 0
		return m, nil
	case "t", "T":
		if len(m.tabs) > 1 {
			m.activeTab = 1
		}
		return m, nil
	case "a", "A":
		if len(m.tabs) > 2 {
			m.activeTab = 2
		}
		return m, nil
	case "c", "C":
		if len(m.tabs) > 3 {
			m.activeTab = 3
		}
		return m, nil
	case "up":
		if m.activeTab < len(m.tabs) {
			if m.tabs[m.activeTab].Scroll > 0 {
				m.tabs[m.activeTab].Scroll--
			}
		}
		return m, nil
	case "down":
		if m.activeTab < len(m.tabs) {
			m.tabs[m.activeTab].Scroll++
		}
		return m, nil
	case "pgup":
		if m.activeTab < len(m.tabs) {
			m.tabs[m.activeTab].Scroll -= m.contentHeight() / 2
			if m.tabs[m.activeTab].Scroll < 0 {
				m.tabs[m.activeTab].Scroll = 0
			}
		}
		return m, nil
	case "pgdown":
		if m.activeTab < len(m.tabs) {
			m.tabs[m.activeTab].Scroll += m.contentHeight() / 2
		}
		return m, nil
	}

	if msg.String() >= "1" && msg.String() <= "9" {
		idx := int(msg.String()[0] - '1')
		if idx < len(m.tabs) {
			m.activeTab = idx
		}
		return m, nil
	}

	return m, nil
}

func (m *Model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter":
		text := strings.TrimSpace(m.input)
		m.inputMode = false
		if text != "" {
			tab := &m.tabs[m.activeTab]
			tab.Lines = append(tab.Lines, Line{
				Time: time.Now().Format("15:04:05"),
				Text: fmt.Sprintf("You > %s", text),
				Type: LineChat,
			})
			m.input = ""

			if m.activeTab == 0 && !m.pipelineStarted && m.onFirstMessage != nil {
				m.pipelineStarted = true
				tab.Lines = append(tab.Lines, Line{
					Time: time.Now().Format("15:04:05"),
					Text: "Got it! Assembling agent teams...",
					Type: LinePhase,
				})
				go m.onFirstMessage(text)
			}
		}
		return m, nil
	case "escape":
		m.inputMode = false
		m.input = ""
		return m, nil
	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
		}
		return m, nil
	default:
		if len(msg.String()) == 1 && msg.String()[0] >= 32 && msg.String()[0] < 127 {
			m.input += msg.String()
		}
		return m, nil
	}
}

func (m *Model) contentHeight() int {
	if !m.ready {
		return 10
	}
	overhead := 7
	if m.inputMode {
		overhead = 9
	}
	h := m.height - overhead
	if h < 3 {
		h = 3
	}
	return h
}

func (m *Model) View() string {
	if !m.ready {
		return "\n  Starting RouterForge 2.0..."
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var b strings.Builder

	b.WriteString(m.renderHeader())
	b.WriteString("\n")
	b.WriteString(m.renderTabs())
	b.WriteString("\n")
	b.WriteString(m.renderContent())
	b.WriteString("\n")

	if m.inputMode {
		b.WriteString(m.renderInput())
		b.WriteString("\n")
	}

	b.WriteString(m.renderStatusBar())
	b.WriteString("\n")
	b.WriteString(m.renderFooter())

	return b.String()
}

func (m *Model) renderHeader() string {
	title := " RouterForge 2.0 "
	style := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#7C3AED")).
		Padding(0, 1)

	titleBox := style.Render(title)

	life := ""
	if m.lifecycle != "" {
		life = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F59E0B")).
			Padding(0, 1).
			Render("[" + m.lifecycle + "]")
	}

	phase := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3B82F6")).
		Padding(0, 1).
		Render(m.phase)

	statusDot := "●"
	statusColor := "#10B981"
	status := "online"
	if m.quit {
		statusDot = "●"
		statusColor = "#EF4444"
		status = "offline"
	}
	statusBox := lipgloss.NewStyle().
		Foreground(lipgloss.Color(statusColor)).
		Render(fmt.Sprintf("%s %s", statusDot, status))

	right := lipgloss.NewStyle().
		Width(m.width - lipgloss.Width(titleBox) - lipgloss.Width(life) - lipgloss.Width(phase) - 5).
		Align(lipgloss.Right).
		Render(statusBox)

	return lipgloss.JoinHorizontal(lipgloss.Center, titleBox, life, phase, right)
}

func (m *Model) renderTabs() string {
	if len(m.tabs) == 0 {
		return ""
	}

	var tabBoxes []string
	for i, tab := range m.tabs {
		isActive := i == m.activeTab
		isSystem := i < systemTabCount && i > 0

		icon := tab.Icon
		if icon == "" {
			icon = "○"
		}
		if tab.Status == "active" || tab.Status == "running" {
			icon = "●"
		}

		fgColor := "#9CA3AF"
		if isActive {
			fgColor = "#FFFFFF"
		}
		style := lipgloss.NewStyle().
			Foreground(lipgloss.Color(fgColor)).
			Padding(0, 1)

		if isActive {
			bgColor := "#7C3AED"
			if isSystem {
				bgColor = "#374151"
			}
			style = style.Background(lipgloss.Color(bgColor))
		} else if isSystem {
			style = style.Foreground(lipgloss.Color("#6B7280"))
		}

		title := tab.Title
		if tab.TasksCnt > 0 {
			title = fmt.Sprintf("%s (%d/%d)", tab.Title, tab.TasksDone, tab.TasksCnt)
		}
		tabBoxes = append(tabBoxes, style.Render(fmt.Sprintf("%s %s", icon, title)))
	}

	tabLine := lipgloss.JoinHorizontal(lipgloss.Top, tabBoxes...)
	divider := strings.Repeat("─", m.width-lipgloss.Width(tabLine)-2)
	if divider == "" {
		divider = "─"
	}
	divStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#374151")).Render(divider)
	return lipgloss.JoinHorizontal(lipgloss.Top, tabLine, " ", divStyle)
}

func (m *Model) renderContent() string {
	if m.activeTab >= len(m.tabs) {
		return ""
	}

	tab := m.tabs[m.activeTab]
	contentHeight := m.contentHeight()

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7C3AED")).
		Padding(0, 1)

	headerParts := []string{fmt.Sprintf("  %s %s", tab.Icon, tab.Title)}
	if tab.Role != "" {
		headerParts = append(headerParts, fmt.Sprintf("(%s)", tab.Role))
	}
	if tab.TasksCnt > 0 {
		headerParts = append(headerParts, fmt.Sprintf("tasks: %d/%d", tab.TasksDone, tab.TasksCnt))
	}
	if tab.Status != "" {
		headerParts = append(headerParts, fmt.Sprintf("status: %s", tab.Status))
	}

	header := headerStyle.Render(strings.Join(headerParts, "  │  "))

	divStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#374151"))
	div := divStyle.Render(strings.Repeat("─", m.width-4))

	var contentLines []string
	contentLines = append(contentLines, header)
	contentLines = append(contentLines, div)

	if len(tab.Lines) == 0 {
		contentLines = append(contentLines,
			lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Italic(true).Render("  No activity yet..."))
	} else {
		start := 0
		if tab.Scroll > 0 {
			start = tab.Scroll
		}
		if start >= len(tab.Lines) {
			start = len(tab.Lines) - 1
			if start < 0 {
				start = 0
			}
		}

		visibleLines := tab.Lines[start:]
		if len(visibleLines) > contentHeight-3 {
			visibleLines = visibleLines[:contentHeight-3]
		}

		for _, line := range visibleLines {
			contentLines = append(contentLines, m.renderLine(line))
		}
	}

	spare := contentHeight - len(contentLines)
	for i := 0; i < spare; i++ {
		contentLines = append(contentLines, "")
	}

	paneStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#374151")).
		Width(m.width - 2).
		Padding(0, 1)

	return paneStyle.Render(strings.Join(contentLines, "\n"))
}

func (m *Model) renderLine(line Line) string {
	prefix := ""
	if line.Time != "" {
		prefix = fmt.Sprintf("[%s]", line.Time)
	}

	timeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))

	maxW := m.width - 14
	text := line.Text
	if maxW > 0 && len(text) > maxW {
		text = text[:maxW-3] + "..."
	}

	switch line.Type {
	case LineSuccess:
		return fmt.Sprintf("  %s %s %s",
			timeStyle.Render(prefix),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render("✓"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB")).Render(text))
	case LineError:
		return fmt.Sprintf("  %s %s %s",
			timeStyle.Render(prefix),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444")).Render("✗"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB")).Render(text))
	case LineWarning:
		return fmt.Sprintf("  %s %s %s",
			timeStyle.Render(prefix),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Render("△"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB")).Render(text))
	case LineInfo:
		return fmt.Sprintf("  %s %s %s",
			timeStyle.Render(prefix),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6")).Render("●"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB")).Render(text))
	case LineArtifact:
		return fmt.Sprintf("  %s %s %s",
			timeStyle.Render(prefix),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#8B5CF6")).Render("\U0001F4C4"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB")).Render(text))
	case LineDecision:
		return fmt.Sprintf("  %s %s %s",
			timeStyle.Render(prefix),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Render("\u26A1"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB")).Render(text))
	case LineChat:
		return fmt.Sprintf("  %s %s %s",
			timeStyle.Render(prefix),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED")).Render("\U0001F4AC"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Render(text))
	case LinePhase:
		return fmt.Sprintf("  %s %s %s",
			timeStyle.Render(prefix),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6")).Render("\u25B6"),
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Render(text))
	case LineTool:
		return fmt.Sprintf("  %s %s %s",
			timeStyle.Render(prefix),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B")).Render("\U0001F527"),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB")).Render(text))
	case LineAgentChat:
		return fmt.Sprintf("  %s %s",
			timeStyle.Render(prefix),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB")).Render(text))
	case LineThinking:
		return fmt.Sprintf("  %s %s %s",
			timeStyle.Render(prefix),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render("\U0001F9E0"),
			lipgloss.NewStyle().Italic(true).Foreground(lipgloss.Color("#9CA3AF")).Render(text))
	default:
		return fmt.Sprintf("  %s   %s",
			timeStyle.Render(prefix),
			lipgloss.NewStyle().Foreground(lipgloss.Color("#D1D5DB")).Render(text))
	}
}

func (m *Model) renderInput() string {
	inputBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#7C3AED")).
		Width(m.width-4).
		Padding(0, 1)

	prompt := "> "
	cursor := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C3AED")).
		Render("\u2588")

	text := m.input
	maxInputW := m.width - 10
	if len(text) > maxInputW {
		text = text[len(text)-maxInputW:]
	}

	return inputBox.Render(fmt.Sprintf("%s%s%s", prompt, text, cursor))
}

func (m *Model) renderStatusBar() string {
	phaseInfo := fmt.Sprintf("Phase: %s", m.phase)
	if m.phase == "" {
		phaseInfo = "Phase: idle"
	}

	agentCount := 0
	active := 0
	for _, t := range m.tabs {
		if t.AgentType == "team" || t.AgentType == "micro" {
			agentCount++
			if t.Status == "active" || t.Status == "running" {
				active++
			}
		}
	}
	agentsInfo := fmt.Sprintf("Agents: %d/%d", active, agentCount)

	costStr := fmt.Sprintf("$%.4f", m.cost)
	tokStr := fmt.Sprintf("%d tok", m.tokens)

	uptime := time.Since(m.startTime).Round(time.Second)
	uptimeStr := fmt.Sprintf("up %s", uptime)

	parts := []string{
		lipgloss.NewStyle().Foreground(lipgloss.Color("#3B82F6")).Render(phaseInfo),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render(agentsInfo),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981")).Render(costStr),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#9CA3AF")).Render(tokStr),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280")).Render(uptimeStr),
	}

	content := strings.Join(parts, "  │  ")

	spacer := m.width - lipgloss.Width(content) - 4
	if spacer < 1 {
		spacer = 1
	}
	padding := strings.Repeat(" ", spacer)

	style := lipgloss.NewStyle().
		Background(lipgloss.Color("#111111")).
		Width(m.width).
		Padding(0, 1)

	return style.Render(content + padding)
}

func (m *Model) renderFooter() string {
	hintStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#4B5563"))

	hints := []string{
		"Tab/\u2190\u2192:Switch",
		"Enter:Chat",
		"\u2191\u2193:Scroll",
		"0-9:Jump",
		"h:t/Chat",
		"q:Quit",
	}

	content := strings.Join(hints, "  ")
	spacer := m.width - lipgloss.Width(content) - 2
	if spacer < 1 {
		spacer = 1
	}
	padding := strings.Repeat(" ", spacer)

	return hintStyle.Render(" " + content + padding)
}

func agentRoleIcon(role string) string {
	r := strings.ToLower(role)
	switch {
	case strings.Contains(r, "frontend"), strings.Contains(r, "ui"), strings.Contains(r, "react"):
		return "\U0001F3A8"
	case strings.Contains(r, "backend"), strings.Contains(r, "api"), strings.Contains(r, "server"):
		return "\u2699\uFE0F"
	case strings.Contains(r, "qa"), strings.Contains(r, "test"), strings.Contains(r, "quality"):
		return "\U0001F9EA"
	case strings.Contains(r, "devops"), strings.Contains(r, "deploy"), strings.Contains(r, "infra"):
		return "\U0001F680"
	case strings.Contains(r, "security"), strings.Contains(r, "auth"):
		return "\U0001F512"
	case strings.Contains(r, "database"), strings.Contains(r, "db"), strings.Contains(r, "data"):
		return "\U0001F4BE"
	case strings.Contains(r, "design"), strings.Contains(r, "ux"):
		return "\U0001F3A8"
	case strings.Contains(r, "doc"), strings.Contains(r, "writer"), strings.Contains(r, "content"):
		return "\U0001F4DD"
	default:
		return "\U0001F916"
	}
}

func (m *Model) addLineToTab(tabID string, line Line) {
	for i := range m.tabs {
		if m.tabs[i].ID == tabID || m.tabs[i].Role == tabID {
			m.tabs[i].Lines = append(m.tabs[i].Lines, line)
			if len(m.tabs[i].Lines) > 200 {
				m.tabs[i].Lines = m.tabs[i].Lines[len(m.tabs[i].Lines)-200:]
			}
			visibleLines := m.contentHeight() - 3
			if len(m.tabs[i].Lines) > visibleLines {
				m.tabs[i].Scroll = len(m.tabs[i].Lines) - visibleLines
			}
			return
		}
	}
	if m.activeTab < len(m.tabs) {
		m.tabs[m.activeTab].Lines = append(m.tabs[m.activeTab].Lines, line)
		if len(m.tabs[m.activeTab].Lines) > 200 {
			m.tabs[m.activeTab].Lines = m.tabs[m.activeTab].Lines[len(m.tabs[m.activeTab].Lines)-200:]
		}
	}
}

func (m *Model) createTab(id, title, role, agentType, parentID string) {
	for _, t := range m.tabs {
		if t.ID == id {
			return
		}
	}
	m.tabs = append(m.tabs, AgentTab{
		ID:        id,
		Title:     title,
		Role:      role,
		AgentType: agentType,
		Status:    "created",
		ParentID:  parentID,
		Icon:      agentRoleIcon(role),
		Lines: []Line{
			{Time: time.Now().Format("15:04:05"), Text: fmt.Sprintf("%s agent created.", title), Type: LineInfo},
		},
	})
}

func (m *Model) updateTabStatus(id, status string) {
	for i := range m.tabs {
		if m.tabs[i].ID == id || m.tabs[i].Role == id {
			m.tabs[i].Status = status
			return
		}
	}
}

func (m *Model) logInternal(msg string) {
	m.internalLog = append(m.internalLog, msg)
	if len(m.internalLog) > 500 {
		m.internalLog = m.internalLog[len(m.internalLog)-500:]
	}
}
