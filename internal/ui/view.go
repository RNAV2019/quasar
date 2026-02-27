package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/glamour"
)

var (
	modeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF")).
			Background(lipgloss.Color("#FF5F87")). // Pink
			Padding(0, 1)

	clearStyle = lipgloss.NewStyle()

	gutterStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	currentLineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F87")).Bold(true)
	markdownRender   *glamour.TermRenderer
)

var modeName = map[Mode]string{
	Normal: "NORMAL",
	Insert: "INSERT",
	Select: "SELECT",
}

func init() {
	customStyle := createStyle()
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(customStyle),
		glamour.WithWordWrap(0),
	)
	if err != nil {
		panic(err)
	}
	markdownRender = renderer
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

	rawDoc := strings.Join(editorLines, "\n")
	fullRendered, _ := markdownRender.Render(rawDoc)

	fullRendered = strings.TrimRight(fullRendered, "\r\n")
	renderedLines := strings.Split(fullRendered, "\n")

	safeToMap := len(renderedLines) == len(editorLines)

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

		line := m.Editor.Lines[actualRowIdx]

		contentBuilder.WriteString(styledGutter)
		if actualRowIdx == m.Editor.Cursor.Row {
			contentBuilder.WriteString(lineStr)
		} else if line.IsMath && line.Rendered != "" {
			h := max(line.ImageHeight, 1)
			contentBuilder.WriteString(strings.Repeat("\n", h-1))
		} else if safeToMap {
			contentBuilder.WriteString(renderedLines[i])
		} else {
			fallback, _ := markdownRender.Render(lineStr)
			fallback = strings.Trim(fallback, "\r\n")
			fallback = strings.TrimPrefix(fallback, "  ")
			contentBuilder.WriteString(fallback)

		}
		contentBuilder.WriteString("\n")

	}

	renderContentHeight := max(m.height-lipgloss.Height(statusLine), 0)
	renderContent := clearStyle.
		Height(renderContentHeight).
		PaddingLeft(3).
		Render(contentBuilder.String())

	view := lipgloss.JoinVertical(lipgloss.Top, renderContent, statusLine)

	// Move terminal cursor to the correct position
	// X: 3 (PaddingLeft) + gutterWidth + 2 (spaces) + (CursorCol - OffsetCol)
	cursorX := 3 + gutterWidth + 2 + (m.Editor.Cursor.Col - m.Editor.Offset.Col)

	// Y must account for the actual height of each line rendered above the cursor
	cursorY := 0
	for i := m.Editor.Offset.Row; i < m.Editor.Cursor.Row; i++ {
		if i >= len(m.Editor.Lines) {
			break
		}
		line := m.Editor.Lines[i]
		if line.IsMath && line.Rendered != "" {
			cursorY += max(line.ImageHeight, 1)
		} else {
			cursorY += 1
		}
	}

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
