package ui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type HelpDialog struct {
	active bool
}

func NewHelpDialog() HelpDialog {
	return HelpDialog{active: false}
}

func (d *HelpDialog) Activate() {
	d.active = true
}

func (d *HelpDialog) Deactivate() {
	d.active = false
}

func (d HelpDialog) Active() bool {
	return d.active
}

func (d HelpDialog) Render(view string, m Model) (string, tea.Cursor) {
	if !d.active {
		return view, tea.Cursor{}
	}

	dialogWidth := 50
	dialogX := (m.width - dialogWidth) / 2
	dialogY := 2

	borderColor := lipgloss.Color("#89B4FA")
	titleColor := lipgloss.Color("#F5C2E7")
	textColor := lipgloss.Color("#CDD6F4")
	keyColor := lipgloss.Color("#A6E3A1")
	dimColor := lipgloss.Color("#6C7086")

	contentWidth := dialogWidth - 2

	// Title
	titleText := " Help "
	titleLen := lipgloss.Width(titleText)
	leftPadding := max((contentWidth-titleLen)/2, 0)
	rightPadding := max(contentWidth-titleLen-leftPadding, 0)

	leftBorder := strings.Repeat("─", leftPadding)
	rightBorder := strings.Repeat("─", rightPadding)

	topBorder := lipgloss.NewStyle().Foreground(borderColor).Render("╭" + leftBorder + titleText + rightBorder + "╮")
	bottomBorder := lipgloss.NewStyle().Foreground(borderColor).Render("╰" + strings.Repeat("─", contentWidth) + "╯")

	// Build content lines
	keyStyle := lipgloss.NewStyle().Foreground(keyColor)
	descStyle := lipgloss.NewStyle().Foreground(textColor)
	sectionStyle := lipgloss.NewStyle().Foreground(titleColor).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(dimColor)

	makeLine := func(key, desc string) string {
		keyPart := keyStyle.Render(key)
		descPart := descStyle.Render(desc)
		usedWidth := lipgloss.Width(keyPart) + lipgloss.Width(descPart) + 2
		padding := contentWidth - usedWidth
		if padding < 0 {
			padding = 0
		}
		leftBorder := lipgloss.NewStyle().Foreground(borderColor).Render("│")
		rightBorder := lipgloss.NewStyle().Foreground(borderColor).Render("│")
		return leftBorder + " " + keyPart + strings.Repeat(" ", padding) + descPart + " " + rightBorder
	}

	makeSection := func(name string) string {
		nameStyled := sectionStyle.Render(name)
		usedWidth := lipgloss.Width(nameStyled) + 2
		padding := contentWidth - usedWidth
		if padding < 0 {
			padding = 0
		}
		leftBorder := lipgloss.NewStyle().Foreground(borderColor).Render("│")
		rightBorder := lipgloss.NewStyle().Foreground(borderColor).Render("│")
		return leftBorder + " " + nameStyled + strings.Repeat(" ", padding) + " " + rightBorder
	}

	makeEmptyLine := func() string {
		return lipgloss.NewStyle().Foreground(borderColor).Render("│" + strings.Repeat(" ", contentWidth) + "│")
	}

	var lines []string
	lines = append(lines, topBorder)
	lines = append(lines, makeSection("Navigation"))
	lines = append(lines, makeLine("h/j/k/l", "move left/down/up/right"))
	lines = append(lines, makeLine("space+f", "toggle file tree"))
	lines = append(lines, makeLine("space+/", "focus file tree"))
	lines = append(lines, makeEmptyLine())
	lines = append(lines, makeSection("Editing"))
	lines = append(lines, makeLine("i", "insert mode"))
	lines = append(lines, makeLine("o", "insert line below"))
	lines = append(lines, makeLine("d", "delete character"))
	lines = append(lines, makeLine("esc", "normal mode"))
	lines = append(lines, makeEmptyLine())
	lines = append(lines, makeSection("Commands"))
	lines = append(lines, makeLine(":w", "save file"))
	lines = append(lines, makeLine(":q", "quit"))
	lines = append(lines, makeLine(":new", "create new note"))
	lines = append(lines, makeLine(":help", "show this help"))
	lines = append(lines, makeEmptyLine())

	// Hint line - simple centered text
	hintText := "press any key to close"
	hintStyled := dimStyle.Render(hintText)
	hintWidth := lipgloss.Width(hintStyled)
	hintLeftPad := (contentWidth - hintWidth) / 2
	hintRightPad := contentWidth - hintWidth - hintLeftPad
	borderLeft := lipgloss.NewStyle().Foreground(borderColor).Render("│")
	borderRight := lipgloss.NewStyle().Foreground(borderColor).Render("│")
	hintLine := borderLeft + strings.Repeat(" ", hintLeftPad) + hintStyled + strings.Repeat(" ", hintRightPad) + borderRight
	lines = append(lines, hintLine)
	lines = append(lines, bottomBorder)

	dialogBox := lipgloss.JoinVertical(lipgloss.Top, lines...)

	bgLayer := lipgloss.NewLayer(view)
	dialogLayer := lipgloss.NewLayer(dialogBox).X(dialogX).Y(dialogY).Z(1)
	compositor := lipgloss.NewCompositor(bgLayer, dialogLayer)
	view = compositor.Render()

	return view, tea.Cursor{}
}
