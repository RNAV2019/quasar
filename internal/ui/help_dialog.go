package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type HelpDialog struct {
	BaseDialog
}

func NewHelpDialog() HelpDialog {
	return HelpDialog{
		BaseDialog: NewBaseDialog(60),
	}
}

func (d HelpDialog) Render(view string, m Model) (string, tea.Cursor) {
	if !d.active {
		return view, tea.Cursor{}
	}

	style := d.style
	contentWidth := 54

	keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(style.KeyColor))
	descStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(style.TextColor))
	sectionStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(style.TitleColor)).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(style.DimColor))

	makeLine := func(key, desc string) string {
		keyPart := keyStyle.Render(key)
		descPart := descStyle.Render(desc)
		keyWidth := lipgloss.Width(keyPart)
		descWidth := lipgloss.Width(descPart)
		padding := contentWidth - keyWidth - descWidth
		if padding < 2 {
			padding = 2
		}
		return keyPart + strings.Repeat(" ", padding) + descPart
	}

	var lines []string

	// Title
	lines = append(lines, sectionStyle.Render("Help"))
	lines = append(lines, "")

	// Navigation section
	lines = append(lines, sectionStyle.Render("Navigation"))
	lines = append(lines, makeLine("h/j/k/l", "move left/down/up/right"))
	lines = append(lines, makeLine("space+f", "toggle file tree"))
	lines = append(lines, makeLine("space+/", "focus file tree"))
	lines = append(lines, "")

	// Editing section
	lines = append(lines, sectionStyle.Render("Editing"))
	lines = append(lines, makeLine("i", "insert mode"))
	lines = append(lines, makeLine("o", "insert line below"))
	lines = append(lines, makeLine("d", "delete character"))
	lines = append(lines, makeLine("esc", "normal mode"))
	lines = append(lines, "")

	// Slash Commands section
	lines = append(lines, sectionStyle.Render("Slash Commands (Insert Mode)"))
	lines = append(lines, makeLine("/h1-/h6", "heading 1-6"))
	lines = append(lines, makeLine("/code", "code block"))
	lines = append(lines, makeLine("/inlinemath", "inline math"))
	lines = append(lines, makeLine("/math", "math block"))
	lines = append(lines, makeLine("/bold", "bold text"))
	lines = append(lines, makeLine("/link", "link"))
	lines = append(lines, makeLine("tab/shift+tab", "navigate autocomplete"))
	lines = append(lines, "")

	// Commands section
	lines = append(lines, sectionStyle.Render("Commands"))
	lines = append(lines, makeLine(":w", "save file"))
	lines = append(lines, makeLine(":q", "quit"))
	lines = append(lines, makeLine(":wq", "save and quit"))
	lines = append(lines, makeLine(":new", "create new note"))
	lines = append(lines, makeLine(":delete", "delete current note"))
	lines = append(lines, makeLine(":help", "show this help"))
	lines = append(lines, "")

	// Hint
	lines = append(lines, dimStyle.Render("press any key to close"))

	content := strings.Join(lines, "\n")

	dialogBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(style.BorderColor)).
		Padding(0, 2).
		Render(content)

	return centerDialog(view, dialogBox, m), tea.Cursor{}
}