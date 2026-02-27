package ui

import (
	"fmt"
	"strings"

	"github.com/RNAV2019/quasar/internal/editor"
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

	// Gutter indicators
	mathGutterIndicator  = lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Render("│")  // Blue
	textGutterIndicator  = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("│") // Gray
	errorGutterIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("│") // Red
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

		// For all blocks, active or inactive, rendered or not, we reserve space equal to the number of raw text lines.
		// This ensures the layout is always stable.
		height := len(block.Lines)

		if !isBlockActive && block.Type == editor.MathBlock && block.Rendered != "" {
			// Inactive, rendered math block. We draw empty lines to hold space for the image overlay.
			// The image will be drawn over this space. If the image is taller, it may be clipped.
			for i := 0; i < height; i++ {
				lineNum := globalLineIdx + 1 + i
				lineNumStr := fmt.Sprintf(" %*d ", gutterWidth, lineNum)
				contentBuilder.WriteString(indicator)
				contentBuilder.WriteString(gutterStyle.Render(lineNumStr))
				contentBuilder.WriteString("\n")
			}
		} else {
			// Active block or a text block. Render raw text.
			for lineIdx, lineStr := range block.Lines {
				lineNum := globalLineIdx + 1
				lineNumStr := fmt.Sprintf(" %*d ", gutterWidth, lineNum)

				isCursorLine := isBlockActive && lineIdx == m.Editor.Cursor.LineIdx
				var styledGutter string
				if isCursorLine {
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
		// Always advance the logical line index by the number of lines in the source.
		// The loop for rendered blocks does not increment globalLineIdx, so we do it here.
		if !isBlockActive && block.Type == editor.MathBlock && block.Rendered != "" {
			globalLineIdx += len(block.Lines)
		}
	}

	renderContentHeight := max(m.height-lipgloss.Height(statusLine), 0)
	renderContent := clearStyle.
		Height(renderContentHeight).
		PaddingLeft(2).
		Render(contentBuilder.String())

	view := lipgloss.JoinVertical(lipgloss.Top, renderContent, statusLine)

	// Move terminal cursor to the correct position.
	cursorX := 2 + gutterWidth + 3 + m.Editor.Cursor.Col

	cursorY := 0
	for i := 0; i < m.Editor.Cursor.BlockIdx; i++ {
		// The height of every block is its line count, ensuring a stable layout.
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
