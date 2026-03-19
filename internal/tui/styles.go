// Package tui provides a beautiful interactive terminal interface for Loom.
package tui

import "github.com/charmbracelet/lipgloss"

// Color palette
var (
	// Primary colors
	ColorPrimary   = lipgloss.Color("#7D56F4") // Purple
	ColorSecondary = lipgloss.Color("#04B575") // Green
	ColorAccent    = lipgloss.Color("#00BFFF") // Cyan

	// Status colors
	ColorSuccess = lipgloss.Color("#04B575") // Green
	ColorWarning = lipgloss.Color("#FFCC00") // Yellow
	ColorError   = lipgloss.Color("#FF5555") // Red
	ColorInfo    = lipgloss.Color("#00BFFF") // Cyan

	// UI colors
	ColorMuted    = lipgloss.Color("#626262") // Gray
	ColorDimmed   = lipgloss.Color("#3C3C3C") // Dark gray
	ColorBorder   = lipgloss.Color("#4A4A4A") // Border gray
	ColorBgDark   = lipgloss.Color("#1A1A2E") // Dark blue background
	ColorBgLight  = lipgloss.Color("#2A2A4E") // Lighter background
	ColorBgCard   = lipgloss.Color("#252540") // Card background

	// Priority colors
	PriorityP0 = lipgloss.Color("#FF5555") // Red - Critical
	PriorityP1 = lipgloss.Color("#FFAA00") // Orange - High
	PriorityP2 = lipgloss.Color("#FFCC00") // Yellow - Medium
	PriorityP3 = lipgloss.Color("#04B575") // Green - Low
	PriorityP4 = lipgloss.Color("#626262") // Gray - Backlog

	// Type colors
	TypeFeature = lipgloss.Color("#7D56F4") // Purple
	TypeBug     = lipgloss.Color("#FF5555") // Red
	TypeTask    = lipgloss.Color("#00BFFF") // Cyan
	TypeChore   = lipgloss.Color("#626262") // Gray
	TypeEpic    = lipgloss.Color("#FFAA00") // Orange
)

// Base styles
var (
	// Title style for main headings
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Margin(0, 0, 1, 0)

	// Subtitle style for secondary headings
	SubtitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary).
			Margin(1, 0, 0, 0)

	// Normal text style
	NormalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF"))

	// Muted text style for less important info
	MutedStyle = lipgloss.NewStyle().
			Foreground(ColorMuted)

	// Dimmed text style for very subtle info
	DimmedStyle = lipgloss.NewStyle().
			Foreground(ColorDimmed)

	// Error style
	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError).
			Bold(true)

	// Success style
	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess).
			Bold(true)

	// Warning style
	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true)
)

// Box styles
var (
	// Box style for containers
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(0, 1).
			Margin(0, 0, 1, 0)

	// Box title style
	BoxTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary).
			Background(ColorBgDark).
			Padding(0, 1)

	// Card style for task items
	CardStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(ColorBorder).
			BorderLeft(true).
			BorderRight(false).
			BorderTop(false).
			BorderBottom(false).
			Padding(0, 1).
			Margin(0, 0, 0, 1)

	// Selected card style
	SelectedCardStyle = lipgloss.NewStyle().
				Border(lipgloss.NormalBorder()).
				BorderForeground(ColorSecondary).
				BorderLeft(true).
				BorderRight(false).
				BorderTop(false).
				BorderBottom(false).
				Background(ColorBgLight).
				Padding(0, 1).
				Margin(0, 0, 0, 1)
)

// Badge styles
var (
	// Score badge style
	ScoreStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorPrimary).
			Padding(0, 1).
			Margin(0, 0, 0, 1)

	// ID badge style
	IDStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorAccent).
			Padding(0, 1)

	// Priority badge styles
	P0Style = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(PriorityP0).
		Padding(0, 1)

	P1Style = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#000000")).
		Background(PriorityP1).
		Padding(0, 1)

	P2Style = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#000000")).
		Background(PriorityP2).
		Padding(0, 1)

	P3Style = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(PriorityP3).
		Padding(0, 1)

	P4Style = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(PriorityP4).
		Padding(0, 1)
)

// Status bar styles
var (
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorBgDark).
			Padding(0, 1)

	StatusBarKeyStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(ColorAccent).
				Background(ColorBgLight).
				Padding(0, 1)

	StatusBarDescStyle = lipgloss.NewStyle().
				Foreground(ColorMuted).
				Padding(0, 1)
)

// Logo style
var LogoStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorPrimary)

// Tagline style
var TaglineStyle = lipgloss.NewStyle().
		Italic(true).
		Foreground(ColorMuted)

// GetPriorityStyle returns the style for a given priority level
func GetPriorityStyle(priority int) lipgloss.Style {
	switch priority {
	case 0:
		return P0Style
	case 1:
		return P1Style
	case 2:
		return P2Style
	case 3:
		return P3Style
	default:
		return P4Style
	}
}

// GetPriorityText returns the text for a given priority level
func GetPriorityText(priority int) string {
	switch priority {
	case 0:
		return "P0"
	case 1:
		return "P1"
	case 2:
		return "P2"
	case 3:
		return "P3"
	default:
		return "P4"
	}
}

// GetPriorityLabel returns a descriptive label for a priority level
func GetPriorityLabel(priority int) string {
	switch priority {
	case 0:
		return "Critical"
	case 1:
		return "High"
	case 2:
		return "Medium"
	case 3:
		return "Low"
	default:
		return "Backlog"
	}
}

// GetTypeIcon returns an icon for an issue type
func GetTypeIcon(issueType string) string {
	switch issueType {
	case "feature":
		return "✨"
	case "bug":
		return "🐛"
	case "task":
		return "🔧"
	case "chore":
		return "📝"
	case "epic":
		return "🎯"
	default:
		return "📋"
	}
}

// GetStatusIcon returns an icon for a status
func GetStatusIcon(status string) string {
	switch status {
	case "open":
		return "○"
	case "in_progress":
		return "◐"
	case "closed":
		return "●"
	case "blocked":
		return "⊘"
	default:
		return "○"
	}
}
