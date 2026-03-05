package dialog

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/RNAV2019/quasar/internal/styles"
)

// HelpDialog displays keybinding help in a two-column layout.
type HelpDialog struct {
	BaseDialog
}

// NewHelpDialog creates a new help dialog.
func NewHelpDialog() HelpDialog {
	return HelpDialog{
		BaseDialog: NewBaseDialog(80),
	}
}

// Render renders the help dialog centered on the view.
func (d HelpDialog) Render(view string, dim Dimensions) (string, tea.Cursor) {
	if !d.Active {
		return view, tea.Cursor{}
	}

	style := styles.DefaultDialogStyle()
	colWidth := 36

	keyStyle := lipgloss.NewStyle().Foreground(style.KeyColor)
	descStyle := lipgloss.NewStyle().Foreground(style.TextColor)
	sectionStyle := lipgloss.NewStyle().Foreground(style.TitleColor).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(style.DimColor)

	makeLine := func(key, desc string) string {
		keyPart := keyStyle.Render(key)
		descPart := descStyle.Render(desc)
		keyWidth := lipgloss.Width(keyPart)
		descWidth := lipgloss.Width(descPart)
		padding := max(colWidth-keyWidth-descWidth, 2)
		return keyPart + strings.Repeat(" ", padding) + descPart
	}

	padLine := func(line string, width int) string {
		return line + strings.Repeat(" ", max(width-lipgloss.Width(line), 0))
	}

	// Left column: Navigation, Motion, Selection, Editing
	var leftLines []string
	leftLines = append(leftLines, sectionStyle.Render("Navigation"))
	leftLines = append(leftLines, makeLine("h/j/k/l", "move left/down/up/right"))
	leftLines = append(leftLines, makeLine("gh", "go to start of line"))
	leftLines = append(leftLines, makeLine("gl", "go to end of line"))
	leftLines = append(leftLines, makeLine("space+f", "toggle file tree"))
	leftLines = append(leftLines, makeLine("space+/", "focus file tree"))
	leftLines = append(leftLines, "")
	leftLines = append(leftLines, sectionStyle.Render("Motion"))
	leftLines = append(leftLines, makeLine("w", "next word"))
	leftLines = append(leftLines, makeLine("b", "previous word"))
	leftLines = append(leftLines, "")
	leftLines = append(leftLines, sectionStyle.Render("Selection"))
	leftLines = append(leftLines, makeLine("v", "enter select mode"))
	leftLines = append(leftLines, makeLine("e", "select word"))
	leftLines = append(leftLines, makeLine("x", "select line"))
	leftLines = append(leftLines, makeLine("esc", "clear selection"))
	leftLines = append(leftLines, "")
	leftLines = append(leftLines, sectionStyle.Render("Editing"))
	leftLines = append(leftLines, makeLine("i", "insert mode"))
	leftLines = append(leftLines, makeLine("o", "insert line below"))
	leftLines = append(leftLines, makeLine("d", "delete char/line"))
	leftLines = append(leftLines, makeLine("y", "yank (copy)"))
	leftLines = append(leftLines, makeLine("p", "paste after"))
	leftLines = append(leftLines, makeLine("esc", "normal mode"))

	// Right column: Select Mode, Insert Mode, Commands, File Tree
	var rightLines []string
	rightLines = append(rightLines, sectionStyle.Render("Select Mode"))
	rightLines = append(rightLines, makeLine("h/j/k/l", "extend selection"))
	rightLines = append(rightLines, makeLine("w/b/e", "extend by word"))
	rightLines = append(rightLines, makeLine("gh/gl", "extend to line ends"))
	rightLines = append(rightLines, makeLine("x", "extend to full line"))
	rightLines = append(rightLines, makeLine("y", "yank selection"))
	rightLines = append(rightLines, makeLine("d", "delete selection"))
	rightLines = append(rightLines, makeLine("esc", "cancel selection"))
	rightLines = append(rightLines, "")
	rightLines = append(rightLines, sectionStyle.Render("Insert Mode"))
	rightLines = append(rightLines, makeLine("arrows", "move cursor"))
	rightLines = append(rightLines, makeLine("backspace", "delete char"))
	rightLines = append(rightLines, makeLine("enter", "new line"))
	rightLines = append(rightLines, makeLine("tab", "next autocomplete"))
	rightLines = append(rightLines, makeLine("/", "slash commands"))
	rightLines = append(rightLines, makeLine("esc", "normal mode"))
	rightLines = append(rightLines, "")
	rightLines = append(rightLines, sectionStyle.Render("Commands"))
	rightLines = append(rightLines, makeLine(":w", "save file"))
	rightLines = append(rightLines, makeLine(":q", "quit"))
	rightLines = append(rightLines, makeLine(":wq", "save and quit"))
	rightLines = append(rightLines, makeLine(":new", "create new note"))
	rightLines = append(rightLines, makeLine(":delete", "delete current note"))
	rightLines = append(rightLines, makeLine(":help", "show this help"))

	// Ensure both columns have same number of lines
	maxLines := max(len(leftLines), len(rightLines))
	for len(leftLines) < maxLines {
		leftLines = append(leftLines, "")
	}
	for len(rightLines) < maxLines {
		rightLines = append(rightLines, "")
	}

	var lines []string
	lines = append(lines, sectionStyle.Render("Help"))
	lines = append(lines, "")

	for i := range maxLines {
		leftPad := padLine(leftLines[i], colWidth)
		lines = append(lines, leftPad+"  "+rightLines[i])
	}

	// Slash commands section (full width)
	lines = append(lines, "")
	lines = append(lines, sectionStyle.Render("Slash Commands (Insert Mode)"))
	slashLine1 := makeLine("/h1-/h6", "heading 1-6") + "  " + makeLine("/bold", "bold text")
	slashLine2 := makeLine("/code", "code block") + "  " + makeLine("/link", "link")
	slashLine3 := makeLine("/math", "math block") + "  " + makeLine("/inlinemath", "inline math")
	lines = append(lines, slashLine1)
	lines = append(lines, slashLine2)
	lines = append(lines, slashLine3)

	// File Tree section
	lines = append(lines, "")
	lines = append(lines, sectionStyle.Render("File Tree"))
	ftLine1 := makeLine("j/k", "navigate") + "  " + makeLine("enter", "open/expand")
	ftLine2 := makeLine("x", "delete") + "  " + makeLine("r", "rename")
	ftLine3 := makeLine("esc", "close file tree") + "  " + makeLine(":", "command mode")
	lines = append(lines, ftLine1)
	lines = append(lines, ftLine2)
	lines = append(lines, ftLine3)

	// Hint
	lines = append(lines, "")
	lines = append(lines, dimStyle.Render("press any key to close"))

	content := strings.Join(lines, "\n")

	dialogBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(style.BorderColor).
		Padding(0, 2).
		Render(content)

	return centerDialog(view, dialogBox, dim), tea.Cursor{}
}
