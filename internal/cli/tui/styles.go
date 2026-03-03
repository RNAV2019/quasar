package tui

import "charm.land/lipgloss/v2"

var (
	BorderColor   = lipgloss.Color("#89B4FA")
	TextColor     = lipgloss.Color("#CDD6F4")
	AccentColor   = lipgloss.Color("#F5C2E7")
	ConfirmColor  = lipgloss.Color("#A6E3A1")
	CancelColor   = lipgloss.Color("#F38BA8")
	DimColor      = lipgloss.Color("#6C7086")

	DialogStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(BorderColor).
			Padding(0, 1)

	TitleStyle = lipgloss.NewStyle().
			Foreground(AccentColor).
			Bold(true)

	PromptStyle = lipgloss.NewStyle().
			Foreground(TextColor)

	ConfirmButtonStyle = lipgloss.NewStyle().
				Foreground(ConfirmColor).
				Bold(true)

	CancelButtonStyle = lipgloss.NewStyle().
				Foreground(CancelColor)
)
