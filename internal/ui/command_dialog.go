package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// RenderCommandBar renders the command line dialog and returns the updated view and cursor configuration.
// It takes the current view, model dimensions, and the command input model to render a centered dialog.
func RenderCommandBar(m Model, view string) (string, tea.Cursor) {
	dialogWidth := 40
	dialogX := (m.width - dialogWidth) / 2
	dialogY := 1

	// Build the command bar with title manually
	titleText := " CmdLine "
	titleLen := lipgloss.Width(titleText)
	borderWidth := dialogWidth
	contentWidth := borderWidth - 2

	leftPadding := max((contentWidth-titleLen)/2, 0)
	rightPadding := max(contentWidth-titleLen-leftPadding, 0)

	leftBorder := strings.Repeat("─", leftPadding)
	rightBorder := strings.Repeat("─", rightPadding)

	topBorder := lipgloss.NewStyle().Foreground(lipgloss.Color("#89B4FA")).Render("╭" + leftBorder + titleText + rightBorder + "╮")
	bottomBorder := lipgloss.NewStyle().Foreground(lipgloss.Color("#89B4FA")).Render("╰" + strings.Repeat("─", contentWidth) + "╯")

	// Render the textinput component
	m.CmdInput.SetWidth(contentWidth - 4)
	inputView := m.CmdInput.View()
	inputWidth := lipgloss.Width(inputView)
	padding := max(contentWidth-inputWidth-1, 0)
	middleLine := "│ " + inputView + strings.Repeat(" ", padding) + "│"
	middleLine = lipgloss.NewStyle().Foreground(lipgloss.Color("#89B4FA")).Render(middleLine)

	dialogBox := lipgloss.JoinVertical(lipgloss.Top, topBorder, middleLine, bottomBorder)

	bgLayer := lipgloss.NewLayer(view)
	dialogLayer := lipgloss.NewLayer(dialogBox).X(dialogX).Y(dialogY).Z(1)
	compositor := lipgloss.NewCompositor(bgLayer, dialogLayer)
	view = compositor.Render()

	// Get cursor from textinput component
	var cursorConfig tea.Cursor
	cmdCursor := m.CmdInput.Cursor()
	if cmdCursor != nil {
		// Position cursor accounting for the dialog position and left border/padding
		cursorConfig = tea.Cursor{
			Position: tea.Position{X: dialogX + 2 + cmdCursor.Position.X, Y: dialogY + 1 + cmdCursor.Position.Y},
			Shape:    tea.CursorBar,
			Color:    lipgloss.Color("#CDD6F4"),
		}
	} else {
		cursorConfig = tea.Cursor{
			Position: tea.Position{X: dialogX + 4, Y: dialogY + 1},
			Shape:    tea.CursorBar,
			Color:    lipgloss.Color("#CDD6F4"),
		}
	}

	return view, cursorConfig
}
