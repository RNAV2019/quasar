package ui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type NewNoteDialog struct {
	textInput textinput.Model
	active    bool
	width     int
	height    int
}

func NewNewNoteDialog() NewNoteDialog {
	ti := textinput.New()
	ti.Placeholder = "NoteName or NoteName:Tag"
	ti.Focus()

	return NewNoteDialog{
		textInput: ti,
		active:    false,
		width:     50,
	}
}

func (d *NewNoteDialog) Activate() {
	d.active = true
	d.textInput.SetValue("")
	d.textInput.Focus()
}

func (d *NewNoteDialog) Deactivate() {
	d.active = false
	d.textInput.Blur()
}

func (d *NewNoteDialog) Active() bool {
	return d.active
}

func (d *NewNoteDialog) Value() string {
	return d.textInput.Value()
}

func (d *NewNoteDialog) SetWidth(w int) {
	d.width = w
	d.textInput.SetWidth(w - 8)
}

func (d *NewNoteDialog) Update(msg tea.Msg) {
	d.textInput, _ = d.textInput.Update(msg)
}

func (d NewNoteDialog) Render(view string, m Model) (string, tea.Cursor) {
	if !d.active {
		return view, tea.Cursor{}
	}

	dialogWidth := d.width
	dialogX := (m.width - dialogWidth) / 2
	dialogY := 1

	titleText := " New Note "
	titleLen := lipgloss.Width(titleText)
	contentWidth := dialogWidth - 2

	leftPadding := max((contentWidth-titleLen)/2, 0)
	rightPadding := max(contentWidth-titleLen-leftPadding, 0)

	leftBorder := strings.Repeat("─", leftPadding)
	rightBorder := strings.Repeat("─", rightPadding)

	borderColor := lipgloss.Color("#89B4FA")
	textColor := lipgloss.Color("#CDD6F4")
	dimColor := lipgloss.Color("#6C7086")

	topBorder := lipgloss.NewStyle().Foreground(borderColor).Render("╭" + leftBorder + titleText + rightBorder + "╮")
	bottomBorder := lipgloss.NewStyle().Foreground(borderColor).Render("╰" + strings.Repeat("─", contentWidth) + "╯")

	inputView := d.textInput.View()
	inputWidth := lipgloss.Width(inputView)
	padding := max(contentWidth-inputWidth-1, 0)
	middleLine := "│ " + inputView + strings.Repeat(" ", padding) + "│"
	middleLine = lipgloss.NewStyle().Foreground(borderColor).Render(middleLine)

	hint := dimColor
	hintText := " format: Name or Name:Tag "
	hintPadding := max(contentWidth-lipgloss.Width(hintText)-2, 0)
	hintLine := lipgloss.NewStyle().Foreground(hint).Render("│ " + hintText + strings.Repeat(" ", hintPadding) + "│")

	dialogBox := lipgloss.JoinVertical(lipgloss.Top, topBorder, middleLine, hintLine, bottomBorder)

	bgLayer := lipgloss.NewLayer(view)
	dialogLayer := lipgloss.NewLayer(dialogBox).X(dialogX).Y(dialogY).Z(1)
	compositor := lipgloss.NewCompositor(bgLayer, dialogLayer)
	view = compositor.Render()

	var cursorConfig tea.Cursor
	cmdCursor := d.textInput.Cursor()
	if cmdCursor != nil {
		cursorConfig = tea.Cursor{
			Position: tea.Position{X: dialogX + 2 + cmdCursor.Position.X, Y: dialogY + 1 + cmdCursor.Position.Y},
			Shape:    tea.CursorBar,
			Color:    textColor,
		}
	} else {
		cursorConfig = tea.Cursor{
			Position: tea.Position{X: dialogX + 4, Y: dialogY + 1},
			Shape:    tea.CursorBar,
			Color:    textColor,
		}
	}

	return view, cursorConfig
}
