package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

var (
	modeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF")).
			Background(lipgloss.Color("#FF5F87")). // Pink
			Padding(0, 1)

	clearStyle = lipgloss.NewStyle()

	gutterStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	currentLineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F87")).Bold(true)
)

var modeName = map[Mode]string{
	Normal: "NORMAL",
	Insert: "INSERT",
	Select: "SELECT",
}

func (m Model) View() tea.View {
	if m.width == 0 {
		return tea.NewView("loading...")
	}

	renderedLeft := modeStyle.Render(modeName[m.mode])
	renderedCenter := clearStyle.Render("[FILENAME]")
	renderedRight := modeStyle.Render(" " + m.Time.Format("15:04"))

	wLeft := lipgloss.Width(renderedLeft)
	wCenter := lipgloss.Width(renderedCenter)
	wRight := lipgloss.Width(renderedRight)

	gapWidth := max(m.width-(wLeft+wCenter+wRight), 0)

	gap1Width := gapWidth / 2
	gap2Width := gapWidth - gap1Width // Ensures exact total width even with odd numbers

	gap1 := clearStyle.Width(gap1Width).Render(strings.Repeat(" ", gap1Width))
	gap2 := clearStyle.Width(gap2Width).Render(strings.Repeat(" ", gap2Width))

	statusLine := lipgloss.JoinHorizontal(lipgloss.Top,
		renderedLeft,
		gap1,
		renderedCenter,
		gap2,
		renderedRight,
	)

	editorLines := m.Editor.ViewLines()
	maxLineNum := len(m.Editor.Lines)
	gutterWidth := len(fmt.Sprint(maxLineNum))

	var contentBuilder strings.Builder
	for i, lineStr := range editorLines {
		actualRowIdx := m.Editor.Offset.Row + i
		lineNum := actualRowIdx + 1
		lineNumStr := fmt.Sprintf("%*d  ", gutterWidth, lineNum)

		var styledGutter string
		if actualRowIdx == m.Editor.Cursor.Row {
			styledGutter = currentLineStyle.Render(lineNumStr)
		} else {
			styledGutter = gutterStyle.Render(lineNumStr)
		}

		contentBuilder.WriteString(styledGutter)

		line := m.Editor.Lines[actualRowIdx]
		if actualRowIdx != m.Editor.Cursor.Row && line.IsMath && line.Rendered != "" {
			h := max(line.ImageHeight, 1)
			contentBuilder.WriteString(strings.Repeat("\n", h))
		} else {
			contentBuilder.WriteString(lineStr)
		}
		if !(actualRowIdx != m.Editor.Cursor.Row && line.IsMath && line.Rendered != "") {
			contentBuilder.WriteString("\n")
		}
	}

	renderContentHeight := max(m.height-lipgloss.Height(statusLine), 0)
	renderContent := clearStyle.
		Height(renderContentHeight).
		PaddingLeft(3).
		Render(contentBuilder.String())

	view := lipgloss.JoinVertical(lipgloss.Top, renderContent, statusLine)

	// Move terminal cursor to the correct position
	// X: 3 (PaddingLeft) + gutterWidth + 2 (spaces) + (CursorCol - OffsetCol)
	// Y: CursorRow - OffsetRow
	cursorX := 3 + gutterWidth + 2 + (m.Editor.Cursor.Col - m.Editor.Offset.Col)
	cursorY := (m.Editor.Cursor.Row - m.Editor.Offset.Row)

	var cursorConfig tea.Cursor
	if m.mode == Normal {
		cursorConfig = tea.Cursor{
			Position: tea.Position{
				X: cursorX,
				Y: cursorY,
			},
			Shape: tea.CursorBlock,
			Blink: false,
			Color: lipgloss.White,
		}
	} else {
		cursorConfig = tea.Cursor{
			Position: tea.Position{
				X: cursorX,
				Y: cursorY,
			},
			Shape: tea.CursorBar,
			Color: lipgloss.White,
		}

	}

	v := tea.NewView(view)
	v.AltScreen = true
	v.Cursor = &cursorConfig
	return v
}
