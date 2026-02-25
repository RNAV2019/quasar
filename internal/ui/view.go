package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	modeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF")).
			Background(lipgloss.Color("#FF5F87")). // Pink
			Padding(0, 1)

	clearStyle = lipgloss.NewStyle()

	gutterStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	currentLineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F87")).Bold(true)
	cursorStyle      = lipgloss.NewStyle().Background(lipgloss.Color("#3C4250"))
)

func (m Model) View() string {
	if m.width == 0 {
		return "loading..."
	}

	// 2. Render the text segments to get their fixed widths
	renderedLeft := modeStyle.Render("NORMAL")
	renderedCenter := clearStyle.Render("[FILENAME]")
	renderedRight := modeStyle.Render(" " + m.Time.Format("15:04"))

	wLeft := lipgloss.Width(renderedLeft)
	wCenter := lipgloss.Width(renderedCenter)
	wRight := lipgloss.Width(renderedRight)

	// 3. Calculate remaining space for the gaps
	gapWidth := max(m.width-(wLeft+wCenter+wRight), 0)

	// 4. Distribute the gap equally between the two spaces
	gap1Width := gapWidth / 2
	gap2Width := gapWidth - gap1Width // Ensures exact total width even with odd numbers

	// 5. Render the gaps with the dark grey background
	gap1 := clearStyle.Width(gap1Width).Render(strings.Repeat(" ", gap1Width))
	gap2 := clearStyle.Width(gap2Width).Render(strings.Repeat(" ", gap2Width))

	// 6. Assemble the status line
	statusLine := lipgloss.JoinHorizontal(lipgloss.Top,
		renderedLeft,
		gap1,
		renderedCenter,
		gap2,
		renderedRight,
	)

	// Main content area
	editorLines := m.Editor.ViewLines()
	maxLineNum := len(m.Editor.Lines)
	gutterWidth := len(fmt.Sprint(maxLineNum))

	var contentBuilder strings.Builder

	for i, line := range editorLines {
		actualRowIdx := m.Editor.Offset.Row + i
		isCursorLine := actualRowIdx == m.Editor.Cursor.Row
		lineNum := actualRowIdx + 1
		lineNumStr := fmt.Sprintf("%*d  ", gutterWidth, lineNum)

		var styledGutter string
		if actualRowIdx == m.Editor.Cursor.Row {
			styledGutter = currentLineStyle.Render(lineNumStr)
		} else {
			styledGutter = gutterStyle.Render(lineNumStr)
		}

		var renderedLine string
		if isCursorLine {
			runes := []rune(line)
			cursorCol := m.Editor.Cursor.Col

			if cursorCol >= len(runes) {
				renderedLine = string(runes) + cursorStyle.Render(" ")
			} else {
				left := string(runes[:cursorCol])
				char := string(runes[cursorCol])
				right := string(runes[cursorCol+1:])

				renderedLine = left + cursorStyle.Render(char) + right
			}
		} else {
			renderedLine = line
		}

		contentBuilder.WriteString(styledGutter)
		contentBuilder.WriteString(renderedLine)
		contentBuilder.WriteString("\n")
	}

	renderContentHeight := max(m.height-lipgloss.Height(statusLine), 0)
	renderContent := clearStyle.
		Height(renderContentHeight).
		PaddingLeft(3).
		Render(contentBuilder.String())

	return lipgloss.JoinVertical(lipgloss.Top, renderContent, statusLine)
}
