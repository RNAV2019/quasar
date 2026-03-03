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
	style     DialogStyle
}

func NewNewNoteDialog() NewNoteDialog {
	ti := textinput.New()
	ti.Placeholder = "NoteName or NoteName:Tag"
	ti.Focus()

	return NewNoteDialog{
		textInput: ti,
		active:    false,
		width:     60,
		style:     DefaultDialogStyle(),
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

	// Build content
	titleStyled := lipgloss.NewStyle().
		Foreground(lipgloss.Color(d.style.TitleColor)).
		Bold(true).
		Render("New Note")

	inputView := d.textInput.View()

	hintStyled := lipgloss.NewStyle().
		Foreground(lipgloss.Color(d.style.DimColor)).
		Render("format: Name or Name:Tag")

	lines := []string{
		titleStyled,
		"",
		inputView,
		"",
		hintStyled,
	}

	content := strings.Join(lines, "\n")

	dialogBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(d.style.BorderColor)).
		Padding(0, 2).
		Render(content)

	dialogWidth := lipgloss.Width(dialogBox)
	dialogX := (m.width - dialogWidth) / 2
	dialogY := 1

	bgLayer := lipgloss.NewLayer(view)
	dialogLayer := lipgloss.NewLayer(dialogBox).X(dialogX).Y(dialogY).Z(1)
	compositor := lipgloss.NewCompositor(bgLayer, dialogLayer)
	view = compositor.Render()

	var cursorConfig tea.Cursor
	cmdCursor := d.textInput.Cursor()
	textColor := lipgloss.Color(d.style.TextColor)
	if cmdCursor != nil {
		cursorConfig = tea.Cursor{
			Position: tea.Position{X: dialogX + 3 + cmdCursor.Position.X, Y: dialogY + 3 + cmdCursor.Position.Y},
			Shape:    tea.CursorBar,
			Color:    textColor,
		}
	} else {
		cursorConfig = tea.Cursor{
			Position: tea.Position{X: dialogX + 5, Y: dialogY + 3},
			Shape:    tea.CursorBar,
			Color:    textColor,
		}
	}

	return view, cursorConfig
}
