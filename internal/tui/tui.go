// Package tui provides a beautiful interactive terminal interface for Loom.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/uttufy/loom/internal/beadsclient"
	"github.com/uttufy/loom/internal/config"
	"github.com/uttufy/loom/internal/scorer"
)

// ViewMode represents the current view
type ViewMode int

const (
	ViewDashboard ViewMode = iota
	ViewLocks
	ViewPatterns
	ViewScore
	ViewDetail
	ViewHelp
)

// Model is the main TUI model
type Model struct {
	// State
	ready   bool
	loading bool
	err     error
	width   int
	height  int

	// Data
	tasks    []*scorer.ScoredIssue
	locks    []FileLockInfo
	patterns []PatternInfo
	selected int

	// Views
	currentView ViewMode
	prevView    ViewMode // For returning from popups

	// Detail view data
	selectedTask      *scorer.ScoredIssue
	selectedBreakdown *scorer.ScoreBreakdown

	// Clients
	client *beadsclient.Client
	scorer *scorer.Scorer
	config *config.Config

	// Timing
	lastRefresh time.Time
}

// FileLockInfo represents file lock information for display
type FileLockInfo struct {
	FilePath  string
	IssueID   string
	AgentID   string
	ClaimedAt time.Time
	ExpiresAt time.Time
}

// PatternInfo represents a pattern for display
type PatternInfo struct {
	ID          string
	Name        string
	Description string
	SuccessRate float64
	UseCount    int
}

// Messages for tea.Cmd
type (
	// tasksLoadedMsg is sent when tasks are loaded
	tasksLoadedMsg struct {
		tasks []*scorer.ScoredIssue
		err   error
	}

	// locksLoadedMsg is sent when locks are loaded
	locksLoadedMsg struct {
		locks []FileLockInfo
		err   error
	}

	// patternsLoadedMsg is sent when patterns are loaded
	patternsLoadedMsg struct {
		patterns []PatternInfo
		err      error
	}

	// taskClaimedMsg is sent when a task is claimed
	taskClaimedMsg struct {
		issueID string
		err     error
	}

	// tickMsg is sent periodically for refresh
	tickMsg time.Time
)

// New creates a new TUI model
func New(client *beadsclient.Client, scorer *scorer.Scorer, cfg *config.Config) Model {
	return Model{
		client:      client,
		scorer:      scorer,
		config:      cfg,
		currentView: ViewDashboard,
		loading:     true,
	}
}

// Init initializes the TUI
func (m Model) Init() tea.Cmd {
	return tea.Batch(
		m.loadTasks(),
		tickCmd(),
	)
}

// Update handles events
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true

	case tasksLoadedMsg:
		m.loading = false
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.tasks = msg.tasks
			m.lastRefresh = time.Now()
		}

	case locksLoadedMsg:
		if msg.err == nil {
			m.locks = msg.locks
		}

	case patternsLoadedMsg:
		if msg.err == nil {
			m.patterns = msg.patterns
		}

	case taskClaimedMsg:
		if msg.err == nil {
			// Refresh tasks after claiming
			cmds = append(cmds, m.loadTasks())
		}

	case tickMsg:
		// Auto-refresh every 30 seconds
		if time.Since(m.lastRefresh) > 30*time.Second {
			cmds = append(cmds, m.loadTasks())
		}
		cmds = append(cmds, tickCmd())
	}

	return m, tea.Batch(cmds...)
}

// View renders the TUI
func (m Model) View() string {
	if !m.ready {
		return m.renderLoading()
	}

	if m.loading && len(m.tasks) == 0 {
		return m.renderLoading()
	}

	if m.err != nil {
		return m.renderError()
	}

	// Handle popup views
	switch m.currentView {
	case ViewHelp:
		return m.renderHelpOverlay()
	case ViewScore:
		return m.renderScoreOverlay()
	case ViewDetail:
		return m.renderDetailOverlay()
	}

	// Main views
	switch m.currentView {
	case ViewLocks:
		return m.renderLocksView()
	case ViewPatterns:
		return m.renderPatternsView()
	default:
		return m.renderDashboard()
	}
}

// handleKeyPress handles keyboard input
func (m Model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle popup views first
	switch m.currentView {
	case ViewHelp:
		if msg.String() == "q" || msg.String() == "?" || msg.String() == "esc" {
			m.currentView = m.prevView
			return m, nil
		}
		return m, nil

	case ViewScore, ViewDetail:
		if msg.String() == "q" || msg.String() == "esc" {
			m.currentView = m.prevView
			return m, nil
		}
		return m, nil
	}

	// Handle main views
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "up", "k":
		if m.selected > 0 {
			m.selected--
		}

	case "down", "j":
		if m.selected < len(m.tasks)-1 {
			m.selected++
		}

	case "enter", " ":
		return m.handleSelect()

	case "s":
		return m.showScoreBreakdown()

	case "d":
		return m.showTaskDetail()

	case "r":
		m.loading = true
		return m, m.loadTasks()

	case "l":
		m.currentView = ViewLocks
		return m, m.loadLocks()

	case "p":
		m.currentView = ViewPatterns
		return m, m.loadPatterns()

	case "t":
		// Toggle between ready and all tasks (future feature)

	case "f":
		// Filter (future feature)

	case "?":
		m.prevView = m.currentView
		m.currentView = ViewHelp

	case "esc":
		if m.currentView != ViewDashboard {
			m.currentView = ViewDashboard
		}
	}

	return m, nil
}

// handleSelect handles task selection/claiming
func (m Model) handleSelect() (tea.Model, tea.Cmd) {
	if len(m.tasks) == 0 || m.selected >= len(m.tasks) {
		return m, nil
	}

	task := m.tasks[m.selected]
	return m, m.claimTask(task.Issue.ID)
}

// showScoreBreakdown shows the score breakdown popup
func (m Model) showScoreBreakdown() (tea.Model, tea.Cmd) {
	if len(m.tasks) == 0 || m.selected >= len(m.tasks) {
		return m, nil
	}

	task := m.tasks[m.selected]
	m.selectedTask = task
	m.prevView = m.currentView
	m.currentView = ViewScore

	// Get detailed breakdown
	if m.scorer != nil {
		breakdown, err := m.scorer.ScoreTaskWithBreakdown(context.Background(), task.Issue)
		if err == nil {
			m.selectedBreakdown = breakdown
		}
	}

	return m, nil
}

// showTaskDetail shows the task detail popup
func (m Model) showTaskDetail() (tea.Model, tea.Cmd) {
	if len(m.tasks) == 0 || m.selected >= len(m.tasks) {
		return m, nil
	}

	task := m.tasks[m.selected]
	m.selectedTask = task
	m.prevView = m.currentView
	m.currentView = ViewDetail

	return m, nil
}

// loadTasks loads tasks from beads
func (m Model) loadTasks() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return tasksLoadedMsg{err: fmt.Errorf("beads client not initialized")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		issues, err := m.client.Ready(ctx, beadsclient.WorkFilter{})
		if err != nil {
			return tasksLoadedMsg{err: err}
		}

		// Score and rank tasks
		var scored []*scorer.ScoredIssue
		if m.scorer != nil {
			scored, err = m.scorer.Rank(ctx, issues)
			if err != nil {
				return tasksLoadedMsg{err: err}
			}
		} else {
			// No scorer, just wrap issues
			scored = make([]*scorer.ScoredIssue, len(issues))
			for i, issue := range issues {
				scored[i] = &scorer.ScoredIssue{Issue: issue, Score: 0}
			}
		}

		return tasksLoadedMsg{tasks: scored}
	}
}

// loadLocks loads file locks
func (m Model) loadLocks() tea.Cmd {
	return func() tea.Msg {
		// TODO: Implement actual lock loading from coordinator
		return locksLoadedMsg{locks: []FileLockInfo{}}
	}
}

// loadPatterns loads learned patterns
func (m Model) loadPatterns() tea.Cmd {
	return func() tea.Msg {
		// TODO: Implement actual pattern loading from retrospective store
		return patternsLoadedMsg{patterns: []PatternInfo{}}
	}
}

// claimTask claims a task
func (m Model) claimTask(issueID string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return taskClaimedMsg{err: fmt.Errorf("beads client not initialized")}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		err := m.client.Claim(ctx, issueID, "loom-tui")
		return taskClaimedMsg{issueID: issueID, err: err}
	}
}

// tickCmd returns a periodic tick command
func tickCmd() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// renderLoading renders the loading screen
func (m Model) renderLoading() string {
	logo := m.renderLogo()

	loadingStyle := lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Bold(true)

	spinner := "⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
	idx := int(time.Now().UnixNano()/100000000) % len(spinner)

	loading := loadingStyle.Render(fmt.Sprintf("\n%s Loading tasks...", string(spinner[idx])))

	return lipgloss.JoinVertical(lipgloss.Center,
		logo,
		loading,
	)
}

// renderError renders the error screen
func (m Model) renderError() string {
	logo := m.renderLogo()

	errStyle := lipgloss.NewStyle().
		Foreground(ColorError).
		Bold(true).
		Margin(1, 0)

	errMsg := errStyle.Render(fmt.Sprintf("⚠ Error: %v", m.err))

	helpStyle := lipgloss.NewStyle().
		Foreground(ColorMuted)

	help := helpStyle.Render("\nPress 'r' to retry or 'q' to quit")

	return lipgloss.JoinVertical(lipgloss.Center,
		logo,
		errMsg,
		help,
	)
}

// renderLogo renders the ASCII logo
func (m Model) renderLogo() string {
	logo := `

██╗░░░░░░█████╗░░█████╗░███╗░░░███╗
██║░░░░░██╔══██╗██╔══██╗████╗░████║
██║░░░░░██║░░██║██║░░██║██╔████╔██║
██║░░░░░██║░░██║██║░░██║██║╚██╔╝██║
███████╗╚█████╔╝╚█████╔╝██║░╚═╝░██║
╚══════╝░╚════╝░░╚════╝░╚═╝░░░░░╚═╝
`

	tagline := "Beads are the task, Loom is what weaves them together"

	logoRendered := LogoStyle.Render(logo)
	taglineRendered := TaglineStyle.Render(tagline)

	return lipgloss.JoinVertical(lipgloss.Center, logoRendered, taglineRendered)
}

// renderDashboard renders the main dashboard view
func (m Model) renderDashboard() string {
	var sections []string

	// Header
	header := m.renderHeader()
	sections = append(sections, header)

	// Task list
	taskList := m.renderTaskList()
	sections = append(sections, taskList)

	// Status bar
	statusBar := m.renderStatusBar()
	sections = append(sections, statusBar)

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderHeader renders the header section
func (m Model) renderHeader() string {
	// Title
	title := TitleStyle.Render("📋 Ready Tasks")

	// Count badge
	count := fmt.Sprintf("%d tasks", len(m.tasks))
	countBadge := lipgloss.NewStyle().
		Foreground(ColorSecondary).
		Render(count)

	// Combine
	return lipgloss.JoinHorizontal(lipgloss.Left, title, " ", countBadge)
}

// renderTaskList renders the list of tasks
func (m Model) renderTaskList() string {
	if len(m.tasks) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(ColorMuted).
			Italic(true).
			Margin(1, 0)

		return emptyStyle.Render("No ready tasks found. Great job! 🎉")
	}

	var items []string
	maxVisible := m.height - 10 // Leave room for header and status bar

	start := 0
	end := len(m.tasks)

	// Scroll window
	if len(m.tasks) > maxVisible {
		if m.selected >= maxVisible {
			start = m.selected - maxVisible + 1
		}
		end = start + maxVisible
		if end > len(m.tasks) {
			end = len(m.tasks)
		}
	}

	for i := start; i < end; i++ {
		task := m.tasks[i]
		selected := i == m.selected
		items = append(items, m.renderTaskCard(task, selected, i))
	}

	return strings.Join(items, "\n")
}

// renderTaskCard renders a single task card
func (m Model) renderTaskCard(task *scorer.ScoredIssue, selected bool, index int) string {
	issue := task.Issue

	// Selection indicator
	indicator := "  "
	if selected {
		indicator = SelectedCardStyle.Foreground(ColorSecondary).Render("▸ ")
	}

	// ID
	id := IDStyle.Render(issue.ID)

	// Priority badge
	priorityBadge := GetPriorityStyle(issue.Priority).Render(GetPriorityText(issue.Priority))

	// Type icon
	typeIcon := GetTypeIcon(string(issue.IssueType))

	// Title
	titleStyle := NormalStyle
	if selected {
		titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF"))
	}
	title := titleStyle.Render(issue.Title)

	// Score badge
	scoreBadge := ScoreStyle.Render(fmt.Sprintf("Score: %d", task.Score))

	// First line
	line1 := lipgloss.JoinHorizontal(lipgloss.Left,
		indicator,
		id, " ",
		priorityBadge, " ",
		typeIcon, " ",
		title, " ",
		scoreBadge,
	)

	// Metadata line
	var meta []string
	if len(issue.Blocks) > 0 {
		meta = append(meta, fmt.Sprintf("Blocks: %d", len(issue.Blocks)))
	}
	age := formatAge(issue.CreatedAt)
	meta = append(meta, fmt.Sprintf("Age: %s", age))
	meta = append(meta, fmt.Sprintf("Type: %s", issue.IssueType))

	metaStr := strings.Join(meta, " │ ")
	metaLine := MutedStyle.Render("    │ " + metaStr)

	return line1 + "\n" + metaLine
}

// renderStatusBar renders the bottom status bar
func (m Model) renderStatusBar() string {
	// Key hints
	keys := []struct {
		key  string
		desc string
	}{
		{"↑/↓", "Navigate"},
		{"Enter", "Claim"},
		{"s", "Score"},
		{"d", "Details"},
		{"r", "Refresh"},
		{"l", "Locks"},
		{"p", "Patterns"},
		{"?", "Help"},
		{"q", "Quit"},
	}

	var hints []string
	for _, k := range keys {
		key := StatusBarKeyStyle.Render(k.key)
		desc := StatusBarDescStyle.Render(k.desc)
		hints = append(hints, lipgloss.JoinHorizontal(lipgloss.Left, key, desc))
	}

	hintsLine := strings.Join(hints, " ")

	return StatusBarStyle.Render(hintsLine)
}

// renderHelpOverlay renders the help overlay
func (m Model) renderHelpOverlay() string {
	// Semi-transparent background overlay
	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(1, 2).
		Width(60)

	title := TitleStyle.Render("⌨ Keyboard Shortcuts")

	shortcuts := []struct {
		keys string
		desc string
	}{
		{"↑/k", "Move up in list"},
		{"↓/j", "Move down in list"},
		{"Enter/Space", "Claim selected task"},
		{"s", "Show score breakdown"},
		{"d", "Show task details"},
		{"r", "Refresh task list"},
		{"l", "View file locks"},
		{"p", "View learned patterns"},
		{"t", "Toggle ready/all tasks"},
		{"f", "Filter tasks"},
		{"?", "Show this help"},
		{"Esc", "Close popup / Return to dashboard"},
		{"q/Ctrl+C", "Quit Loom"},
	}

	var lines []string
	lines = append(lines, title)
	lines = append(lines, "")

	for _, s := range shortcuts {
		keyStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent).
			Width(15)

		descStyle := MutedStyle

		line := lipgloss.JoinHorizontal(lipgloss.Left,
			keyStyle.Render(s.keys),
			descStyle.Render(s.desc),
		)
		lines = append(lines, line)
	}

	lines = append(lines, "")
	lines = append(lines, MutedStyle.Render("Press any key to close"))

	return overlayStyle.Render(strings.Join(lines, "\n"))
}

// renderScoreOverlay renders the score breakdown overlay
func (m Model) renderScoreOverlay() string {
	if m.selectedTask == nil {
		return "No task selected"
	}

	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(1, 2).
		Width(60)

	title := TitleStyle.Render(fmt.Sprintf("📊 Score Breakdown: %s", m.selectedTask.Issue.ID))
	taskTitle := NormalStyle.Render(m.selectedTask.Issue.Title)

	var content []string
	content = append(content, title)
	content = append(content, taskTitle)
	content = append(content, "")

	if m.selectedBreakdown != nil {
		b := m.selectedBreakdown

		// Total score
		totalStyle := lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary).
			Render(fmt.Sprintf("Total Score: %d", b.Total))
		content = append(content, totalStyle)
		content = append(content, "")

		// Breakdown
		breakdownItems := []struct {
			name  string
			value int
			desc  string
		}{
			{"Blocking bonus", b.BlockingBonus, fmt.Sprintf("(blocks %d tasks)", b.BlockedCount)},
			{"Priority bonus", b.PriorityBonus, ""},
			{"Staleness bonus", b.StalenessBonus, fmt.Sprintf("(stale: %v)", b.IsStale)},
			{"Failure penalty", -b.FailurePenalty, fmt.Sprintf("(failed: %v)", b.HasFailedBefore)},
		}

		for _, item := range breakdownItems {
			valueStr := fmt.Sprintf("%+d", item.value)
			if item.value > 0 {
				valueStr = SuccessStyle.Render(valueStr)
			} else if item.value < 0 {
				valueStr = ErrorStyle.Render(valueStr)
			}

			line := lipgloss.JoinHorizontal(lipgloss.Left,
				MutedStyle.Render(item.name),
				" ",
				valueStr,
				" ",
				DimmedStyle.Render(item.desc),
			)
			content = append(content, line)
		}
	}

	content = append(content, "")
	content = append(content, MutedStyle.Render("Press Esc or q to close"))

	return overlayStyle.Render(strings.Join(content, "\n"))
}

// renderDetailOverlay renders the task detail overlay
func (m Model) renderDetailOverlay() string {
	if m.selectedTask == nil {
		return "No task selected"
	}

	issue := m.selectedTask.Issue

	overlayStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ColorPrimary).
		Padding(1, 2).
		Width(70)

	title := TitleStyle.Render(fmt.Sprintf("📋 Task Details: %s", issue.ID))

	var content []string
	content = append(content, title)
	content = append(content, "")

	// Title
	content = append(content, NormalStyle.Bold(true).Render(issue.Title))
	content = append(content, "")

	// Status and Priority
	statusIcon := GetStatusIcon(string(issue.Status))
	priorityBadge := GetPriorityStyle(issue.Priority).Render(GetPriorityLabel(issue.Priority))
	typeIcon := GetTypeIcon(string(issue.IssueType))

	meta := lipgloss.JoinHorizontal(lipgloss.Left,
		MutedStyle.Render("Status: "),
		NormalStyle.Render(statusIcon+" "+string(issue.Status)),
		MutedStyle.Render("  Priority: "),
		priorityBadge,
		MutedStyle.Render("  Type: "),
		NormalStyle.Render(typeIcon+" "+string(issue.IssueType)),
	)
	content = append(content, meta)
	content = append(content, "")

	// Description
	if issue.Description != "" {
		content = append(content, MutedStyle.Render("Description:"))
		content = append(content, NormalStyle.Render(issue.Description))
		content = append(content, "")
	}

	// Dates
	content = append(content, MutedStyle.Render(fmt.Sprintf("Created: %s", formatTime(issue.CreatedAt))))
	content = append(content, MutedStyle.Render(fmt.Sprintf("Updated: %s", formatTime(issue.UpdatedAt))))

	// Dependencies
	if len(issue.Blocks) > 0 {
		content = append(content, "")
		content = append(content, MutedStyle.Render(fmt.Sprintf("Blocks: %s", strings.Join(issue.Blocks, ", "))))
	}
	if len(issue.BlockedBy) > 0 {
		content = append(content, MutedStyle.Render(fmt.Sprintf("Blocked by: %s", strings.Join(issue.BlockedBy, ", "))))
	}

	// Labels
	if len(issue.Labels) > 0 {
		content = append(content, "")
		content = append(content, MutedStyle.Render(fmt.Sprintf("Labels: %s", strings.Join(issue.Labels, ", "))))
	}

	content = append(content, "")
	content = append(content, MutedStyle.Render("Press Esc or q to close"))

	return overlayStyle.Render(strings.Join(content, "\n"))
}

// renderLocksView renders the file locks view
func (m Model) renderLocksView() string {
	title := TitleStyle.Render("🔒 File Locks")

	if len(m.locks) == 0 {
		empty := MutedStyle.Italic(true).Render("\nNo active file locks")
		return title + empty + "\n\n" + m.renderStatusBar()
	}

	var lines []string
	lines = append(lines, title)
	lines = append(lines, "")

	for _, lock := range m.locks {
		filePath := NormalStyle.Render(lock.FilePath)
		issueID := IDStyle.Render(lock.IssueID)
		agentID := MutedStyle.Render(lock.AgentID)
		expires := MutedStyle.Render(fmt.Sprintf("Expires: %s", formatTime(lock.ExpiresAt)))

		line := fmt.Sprintf("  %s\n    %s │ %s │ %s", filePath, issueID, agentID, expires)
		lines = append(lines, line)
	}

	lines = append(lines, "")
	lines = append(lines, m.renderStatusBar())

	return strings.Join(lines, "\n")
}

// renderPatternsView renders the learned patterns view
func (m Model) renderPatternsView() string {
	title := TitleStyle.Render("🧠 Learned Patterns")

	if len(m.patterns) == 0 {
		empty := MutedStyle.Italic(true).Render("\nNo patterns learned yet")
		return title + empty + "\n\n" + m.renderStatusBar()
	}

	var lines []string
	lines = append(lines, title)
	lines = append(lines, "")

	for i, pattern := range m.patterns {
		num := lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true).
			Render(fmt.Sprintf("%d.", i+1))

		name := NormalStyle.Bold(true).Render(pattern.Name)
		desc := MutedStyle.Render(pattern.Description)
		stats := lipgloss.NewStyle().
			Foreground(ColorSecondary).
			Render(fmt.Sprintf("Success: %.0f%% │ Used: %d times", pattern.SuccessRate*100, pattern.UseCount))

		lines = append(lines, fmt.Sprintf("%s %s\n    %s\n    %s", num, name, desc, stats))
	}

	lines = append(lines, "")
	lines = append(lines, m.renderStatusBar())

	return strings.Join(lines, "\n")
}

// Helper functions

func formatAge(t time.Time) string {
	d := time.Since(t)

	if d < time.Minute {
		return "just now"
	} else if d < time.Hour {
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	} else if d < 7*24*time.Hour {
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	} else if d < 30*24*time.Hour {
		return fmt.Sprintf("%dw ago", int(d.Hours()/(24*7)))
	}
	return fmt.Sprintf("%dmo ago", int(d.Hours()/(24*30)))
}

func formatTime(t time.Time) string {
	return t.Format("2006-01-02 15:04")
}
