package dialog

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/RNAV2019/quasar/internal/errors"
	"github.com/RNAV2019/quasar/internal/styles"
)

// ErrorDialog displays a list of accumulated errors.
type ErrorDialog struct {
	BaseDialog
}

// NewErrorDialog creates a new error dialog.
func NewErrorDialog() ErrorDialog {
	return ErrorDialog{
		BaseDialog: NewBaseDialog(80),
	}
}

// Render renders the error dialog centered on the view.
func (d ErrorDialog) Render(view string, dim Dimensions) (string, tea.Cursor) {
	if !d.Active {
		return view, tea.Cursor{}
	}

	style := styles.DefaultDialogStyle()

	titleStyle := lipgloss.NewStyle().Foreground(style.TitleColor).Bold(true)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6b6b"))
	sourceStyle := lipgloss.NewStyle().Foreground(style.DimColor)
	dimStyle := lipgloss.NewStyle().Foreground(style.DimColor)

	errs := errors.GetErrors()

	var lines []string
	lines = append(lines, titleStyle.Render("Errors"))
	lines = append(lines, "")

	if len(errs) == 0 {
		lines = append(lines, dimStyle.Render("No errors"))
	} else {
		maxWidth := 76
		for i, err := range errs {
			if i > 0 {
				lines = append(lines, "")
			}
			sourceLabel := sourceStyle.Render("[" + err.Source + "]")
			lines = append(lines, sourceLabel)

			msgLines := WrapText(err.Message, maxWidth)
			for _, msgLine := range msgLines {
				lines = append(lines, errorStyle.Render(msgLine))
			}
		}
	}

	lines = append(lines, "")
	if len(errs) > 0 {
		lines = append(lines, dimStyle.Render("press , to clear all errors | y to copy | esc to close"))
	} else {
		lines = append(lines, dimStyle.Render("press any key to close"))
	}

	content := strings.Join(lines, "\n")

	dialogBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(style.BorderColor).
		Padding(0, 2).
		Render(content)

	return centerDialog(view, dialogBox, dim), tea.Cursor{}
}

// WrapText wraps text to fit within the given width.
func WrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{text}
	}

	currentLine := words[0]
	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}
