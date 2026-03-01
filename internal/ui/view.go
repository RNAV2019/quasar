package ui

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/RNAV2019/quasar/internal/editor"
	"github.com/RNAV2019/quasar/internal/latex"
	"github.com/charmbracelet/glamour"
)

var (
	modeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFF")).
			Background(lipgloss.Color("#FF5F87")).
			Padding(0, 1)

	clearStyle = lipgloss.NewStyle()

	gutterStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	currentLineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F87")).Bold(true)
	markdownRender   *glamour.TermRenderer

	mathGutterIndicator  = lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Render("│")
	textGutterIndicator  = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("│")
	errorGutterIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("│")
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
	renderedRight := modeStyle.Render(" " + m.Time.Format("15:04"))

	wLeft := lipgloss.Width(renderedLeft)
	wCenter := lipgloss.Width(renderedCenter)
	wRight := lipgloss.Width(renderedRight)

	gapWidth := max(m.width-(wLeft+wCenter+wRight), 0)

	gap1Width := gapWidth / 2
	gap2Width := gapWidth - gap1Width

	gap1 := clearStyle.Width(gap1Width).Render(strings.Repeat(" ", gap1Width))
	gap2 := clearStyle.Width(gap2Width).Render(strings.Repeat(" ", gap2Width))

	statusLine := lipgloss.JoinHorizontal(lipgloss.Top,
		renderedLeft,
		gap1,
		renderedCenter,
		gap2,
		renderedRight,
	)

	totalLines := 0
	for _, block := range m.Editor.Blocks {
		totalLines += len(block.Lines)
	}
	gutterWidth := len(fmt.Sprint(totalLines))

	var contentBuilder strings.Builder
	globalLineIdx := 0
	for blockIdx, block := range m.Editor.Blocks {
		isBlockActive := blockIdx == m.Editor.Cursor.BlockIdx

		var indicator string
		if block.HasError {
			indicator = errorGutterIndicator
		} else if block.Type == editor.MathBlock {
			indicator = mathGutterIndicator
		} else {
			indicator = textGutterIndicator
		}

		height := len(block.Lines)

		if !isBlockActive && block.Type == editor.MathBlock && block.ImageID != 0 {
			for i := range height {
				lineNum := globalLineIdx + 1 + i
				lineNumStr := fmt.Sprintf(" %*d ", gutterWidth, lineNum)
				contentBuilder.WriteString(indicator)
				contentBuilder.WriteString(gutterStyle.Render(lineNumStr))
				contentBuilder.WriteString(latex.PlaceholderRow(block.ImageID, uint16(i), block.ImageCols))
				contentBuilder.WriteString("\n")
			}
			globalLineIdx += height
		} else {
			for lineIdx, lineStr := range block.Lines {
				shouldBlank := !(m.mode == Insert && isBlockActive)

				if shouldBlank && block.Type == editor.TextBlock {
					lineStr = m.applyInlinePlaceholders(blockIdx, lineIdx, lineStr)
				}

				lineNum := globalLineIdx + 1
				lineNumStr := fmt.Sprintf(" %*d ", gutterWidth, lineNum)

				var styledGutter string
				if isBlockActive && lineIdx == m.Editor.Cursor.LineIdx {
					styledGutter = currentLineStyle.Render(lineNumStr)
				} else {
					styledGutter = gutterStyle.Render(lineNumStr)
				}
				contentBuilder.WriteString(indicator)
				contentBuilder.WriteString(styledGutter)
				contentBuilder.WriteString(lineStr)
				contentBuilder.WriteString("\n")
				globalLineIdx++
			}
		}
	}

	renderContentHeight := max(m.height-lipgloss.Height(statusLine), 0)
	renderContent := clearStyle.
		Height(renderContentHeight).
		PaddingLeft(2).
		Render(contentBuilder.String())

	view := lipgloss.JoinVertical(lipgloss.Top, renderContent, statusLine)

	cursorX := 2 + gutterWidth + 3 + m.Editor.Cursor.Col

	cursorY := 0
	for i := 0; i < m.Editor.Cursor.BlockIdx; i++ {
		cursorY += len(m.Editor.Blocks[i].Lines)
	}
	cursorY += m.Editor.Cursor.LineIdx

	var cursorConfig tea.Cursor
	if m.mode == Normal {
		cursorConfig = tea.Cursor{
			Position: tea.Position{X: cursorX, Y: cursorY},
			Shape:    tea.CursorBlock,
			Blink:    false,
			Color:    lipgloss.White,
		}
	} else {
		cursorConfig = tea.Cursor{
			Position: tea.Position{X: cursorX, Y: cursorY},
			Shape:    tea.CursorBar,
			Color:    lipgloss.White,
		}
	}

	v := tea.NewView(view)
	v.AltScreen = true
	v.Cursor = &cursorConfig
	return v
}

func (m Model) applyInlinePlaceholders(blockIdx, lineIdx int, lineStr string) string {
	type inlineMatch struct {
		startCol int
		render   InlineMathRender
		hovered  bool
	}

	var matches []inlineMatch
	isBlockActive := blockIdx == m.Editor.Cursor.BlockIdx

	for key, render := range m.InlineRenders {
		var bIdx, lIdx, startCol int
		fmt.Sscanf(key, "%d-%d-%d", &bIdx, &lIdx, &startCol)
		if bIdx != blockIdx || lIdx != lineIdx {
			continue
		}

		hovered := m.mode == Normal && isBlockActive && lineIdx == m.Editor.Cursor.LineIdx &&
			m.Editor.Cursor.Col >= startCol && m.Editor.Cursor.Col < startCol+render.Length

		matches = append(matches, inlineMatch{startCol: startCol, render: render, hovered: hovered})
	}

	if len(matches) == 0 {
		return lineStr
	}

	slices.SortFunc(matches, func(a, b inlineMatch) int {
		return a.startCol - b.startCol
	})

	runes := []rune(lineStr)
	var result strings.Builder
	pos := 0

	for _, match := range matches {
		if match.startCol > pos {
			result.WriteString(string(runes[pos:match.startCol]))
		}

		if match.hovered {
			end := match.startCol + match.render.Length
			if end > len(runes) {
				end = len(runes)
			}
			result.WriteString(string(runes[match.startCol:end]))
		} else {
			result.WriteString(latex.PlaceholderRow(match.render.ImageID, 0, match.render.Length))
		}
		pos = match.startCol + match.render.Length
	}

	if pos < len(runes) {
		result.WriteString(string(runes[pos:]))
	}

	return result.String()
}
