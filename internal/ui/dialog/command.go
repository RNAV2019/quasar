package dialog

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/RNAV2019/quasar/internal/styles"
)

// RenderCommandBar renders the command line dialog and returns the updated
// view and cursor configuration.
func RenderCommandBar(cmdInput textinput.Model, view string, dim Dimensions) (string, tea.Cursor) {
	dialogWidth := 40
	dialogX := (dim.Width - dialogWidth) / 2
	dialogY := 1

	titleText := " CmdLine "
	titleLen := lipgloss.Width(titleText)
	borderWidth := dialogWidth
	contentWidth := borderWidth - 2

	leftPadding := max((contentWidth-titleLen)/2, 0)
	rightPadding := max(contentWidth-titleLen-leftPadding, 0)

	leftBorder := strings.Repeat("─", leftPadding)
	rightBorder := strings.Repeat("─", rightPadding)

	topBorder := lipgloss.NewStyle().Foreground(styles.ColorBlue).Render("╭" + leftBorder + titleText + rightBorder + "╮")
	bottomBorder := lipgloss.NewStyle().Foreground(styles.ColorBlue).Render("╰" + strings.Repeat("─", contentWidth) + "╯")

	cmdInput.SetWidth(contentWidth - 4)
	inputView := cmdInput.View()
	inputWidth := lipgloss.Width(inputView)
	padding := max(contentWidth-inputWidth-1, 0)
	middleLine := "│ " + inputView + strings.Repeat(" ", padding) + "│"
	middleLine = lipgloss.NewStyle().Foreground(styles.ColorBlue).Render(middleLine)

	dialogBox := lipgloss.JoinVertical(lipgloss.Top, topBorder, middleLine, bottomBorder)

	bgLayer := lipgloss.NewLayer(view)
	dialogLayer := lipgloss.NewLayer(dialogBox).X(dialogX).Y(dialogY).Z(1)
	compositor := lipgloss.NewCompositor(bgLayer, dialogLayer)
	view = compositor.Render()

	var cursorConfig tea.Cursor
	cmdCursor := cmdInput.Cursor()
	if cmdCursor != nil {
		cursorConfig = tea.Cursor{
			Position: tea.Position{X: dialogX + 2 + cmdCursor.Position.X, Y: dialogY + 1 + cmdCursor.Position.Y},
			Shape:    tea.CursorBar,
			Color:    styles.ColorText,
		}
	} else {
		cursorConfig = tea.Cursor{
			Position: tea.Position{X: dialogX + 4, Y: dialogY + 1},
			Shape:    tea.CursorBar,
			Color:    styles.ColorText,
		}
	}

	return view, cursorConfig
}
