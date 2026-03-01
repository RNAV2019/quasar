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
	errorStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

	mathGutterIndicator  = lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Render("│")
	textGutterIndicator  = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("│")
	errorGutterIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("│")
)

type renderedBlock struct {
	lines           []string
	contentStartIdx int // Index in block.Lines where actual content starts (after front matter)
}

var modeName = map[Mode]string{
	Normal:  "NORMAL",
	Insert:  "INSERT",
	Select:  "SELECT",
	Command: "COMMAND",
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

// stripFrontMatter removes YAML front matter from content.
// If the content starts with "---" followed by a newline, it removes everything
// up to and including the next "---" line.
// Returns the stripped content and the line index where content starts.
func stripFrontMatter(content string) (string, int) {
	lines := strings.Split(content, "\n")
	if len(lines) < 3 {
		return content, 0
	}

	// Check if first line is "---"
	if strings.TrimSpace(lines[0]) != "---" {
		return content, 0
	}

	// Find the closing "---"
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			// Return everything after the closing "---" and the index where it starts
			return strings.Join(lines[i+1:], "\n"), i + 1
		}
	}

	return content, 0
}

// renderTextBlockWithGlamour renders a text block's content using glamour.
// It returns rendered markdown lines. Lines with inline math or that are cursor line
// will return empty string in the lines slice (to be rendered as raw text).
func (m Model) renderTextBlockWithGlamour(block editor.Block) renderedBlock {
	if len(block.Lines) == 0 {
		return renderedBlock{lines: []string{""}, contentStartIdx: 0}
	}

	// Join lines with newlines for multi-line markdown
	content := strings.Join(block.Lines, "\n")

	// Strip YAML front matter (between --- delimiters) before rendering
	content, contentStartIdx := stripFrontMatter(content)

	rendered, err := markdownRender.Render(content)
	if err != nil {
		return renderedBlock{lines: block.Lines}
	}

	// Split into lines
	lines := strings.Split(rendered, "\n")
	// Remove trailing empty line that glamour often adds
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return renderedBlock{lines: lines, contentStartIdx: contentStartIdx}
}

func (m Model) View() tea.View {
	if m.width == 0 {
		return tea.NewView("loading...")
	}

	renderedLeft := modeStyle.Render(modeName[m.mode])
	
	var renderedCenter string
	if m.mode == Command {
		renderedCenter = clearStyle.Render(":" + m.CommandBuffer)
	} else if m.StatusMessage != "" {
		renderedCenter = clearStyle.Render(m.StatusMessage)
	} else {
		renderedCenter = clearStyle.Render("[FILENAME]")
	}

	// Find any error message to display in statusline
	var statusError string
	for _, block := range m.Editor.Blocks {
		if block.HasError && block.ErrorMessage != "" {
			statusError = block.ErrorMessage
			break
		}
	}

	var renderedRight string
	if statusError != "" {
		renderedRight = errorStyle.Render(" "+statusError+" ")
	} else {
		renderedRight = modeStyle.Render(" " + m.Time.Format("15:04"))
	}

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

	// Track visual line mapping for cursor positioning
	// visualLineMap[blockIdx] = list of visual line counts per logical line
	visualLineMap := make(map[int][]int)

	for blockIdx, block := range m.Editor.Blocks {
		isBlockActive := blockIdx == m.Editor.Cursor.BlockIdx
		useMarkdown := m.mode == Normal && block.Type == editor.TextBlock

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
			visualLineMap[blockIdx] = make([]int, height)
			for i := range height {
				visualLineMap[blockIdx][i] = 1
			}
			for i := range height {
				lineNum := globalLineIdx + 1 + i
				lineNumStr := fmt.Sprintf(" %*d ", gutterWidth, lineNum)
				contentBuilder.WriteString(indicator)
				contentBuilder.WriteString(gutterStyle.Render(lineNumStr))
				contentBuilder.WriteString(latex.PlaceholderRow(block.ImageID, uint16(i), block.ImageCols))
				contentBuilder.WriteString("\n")
			}
			globalLineIdx += height
		} else if useMarkdown {
			// Render TextBlock with glamour in Normal mode
			// Only render markdown for non-cursor lines and lines without inline math
			rendered := m.renderTextBlockWithGlamour(block)
			visualLineMap[blockIdx] = make([]int, len(block.Lines))
			for i := range block.Lines {
				visualLineMap[blockIdx][i] = 1
			}

			for lineIdx, lineStr := range block.Lines {
				isCursorLine := isBlockActive && lineIdx == m.Editor.Cursor.LineIdx
				hasInlineMath := len(inlineMathRe.FindStringIndex(lineStr)) > 0

				var displayLine string
				// Only render markdown for non-cursor lines without inline math
				// Also need to ensure lineIdx is after front matter (contentStartIdx)
				renderedIdx := lineIdx - rendered.contentStartIdx
				if !isCursorLine && !hasInlineMath && renderedIdx >= 0 && renderedIdx < len(rendered.lines) {
					displayLine = rendered.lines[renderedIdx]
				} else {
					// Show raw text with inline math placeholders
					displayLine = m.applyInlinePlaceholders(blockIdx, lineIdx, lineStr)
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
				contentBuilder.WriteString(displayLine)
				contentBuilder.WriteString("\n")
				globalLineIdx++
			}
		} else {
			visualLineMap[blockIdx] = make([]int, height)
			for i := range height {
				visualLineMap[blockIdx][i] = 1
			}
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

	// Calculate cursor position
	cursorBlockIdx := m.Editor.Cursor.BlockIdx
	cursorLineIdx := m.Editor.Cursor.LineIdx
	cursorCol := m.Editor.Cursor.Col

	// Bounds check: ensure cursor is within valid block range
	if cursorBlockIdx >= len(m.Editor.Blocks) {
		cursorBlockIdx = len(m.Editor.Blocks) - 1
	}
	if cursorBlockIdx < 0 {
		cursorBlockIdx = 0
	}

	// Calculate visual Y position using the mapping
	cursorY := 0
	for i := 0; i < cursorBlockIdx && i < len(m.Editor.Blocks); i++ {
		if lines, ok := visualLineMap[i]; ok {
			for _, v := range lines {
				cursorY += v
			}
		} else {
			cursorY += len(m.Editor.Blocks[i].Lines)
		}
	}
	if lines, ok := visualLineMap[cursorBlockIdx]; ok {
		maxLineIdx := len(lines) - 1
		if cursorLineIdx > maxLineIdx {
			cursorLineIdx = maxLineIdx
		}
		if cursorLineIdx < 0 {
			cursorLineIdx = 0
		}
		for i := 0; i < cursorLineIdx && i < len(lines); i++ {
			cursorY += lines[i]
		}
	} else {
		if cursorLineIdx > 0 && cursorBlockIdx < len(m.Editor.Blocks) {
			maxLineIdx := len(m.Editor.Blocks[cursorBlockIdx].Lines) - 1
			if cursorLineIdx > maxLineIdx {
				cursorLineIdx = maxLineIdx
			}
		}
		if cursorLineIdx < 0 {
			cursorLineIdx = 0
		}
		cursorY += cursorLineIdx
	}

	// Cursor X is simple since cursor line shows raw text
	cursorX := 2 + gutterWidth + 3
	if cursorBlockIdx < len(m.Editor.Blocks) && cursorLineIdx < len(m.Editor.Blocks[cursorBlockIdx].Lines) {
		lineLen := len(m.Editor.Blocks[cursorBlockIdx].Lines[cursorLineIdx])
		if cursorCol > lineLen {
			cursorCol = lineLen
		}
	}
	cursorX += cursorCol

	var cursorConfig tea.Cursor
	if m.mode == Command {
		// Position cursor at end of command buffer
		cmdCursorX := 2 + len(":" + m.CommandBuffer)
		cmdCursorY := m.height - 1
		cursorConfig = tea.Cursor{
			Position: tea.Position{X: cmdCursorX, Y: cmdCursorY},
			Shape:    tea.CursorBar,
			Color:    lipgloss.White,
		}
	} else if m.mode == Normal {
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
	// Bounds check: if block no longer exists, return raw text
	if blockIdx >= len(m.Editor.Blocks) {
		return lineStr
	}

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
