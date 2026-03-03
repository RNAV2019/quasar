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
	"github.com/charmbracelet/x/ansi"
)

var (
	gutterStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	currentLineStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5F87")).Bold(true)
	tabHighlightStyle = lipgloss.NewStyle().Background(lipgloss.Color("#313244"))
	markdownRender   *glamour.TermRenderer

	mathGutterIndicator  = lipgloss.NewStyle().Foreground(lipgloss.Color("69")).Render("│")
	textGutterIndicator  = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("│")
	errorGutterIndicator = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("│")
)

type renderedBlock struct {
	lines             [][]string
	contentStartIdx   int
	inlineMathChecker func(lineIdx int) bool
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

func trimEmptyLines(lines []string) []string {
	start := 0
	for start < len(lines) && strings.TrimSpace(lines[start]) == "" {
		start++
	}
	end := len(lines)
	for end > start && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	return lines[start:end]
}

func renderGlamourLines(content string) []string {
	out, err := markdownRender.Render(content)
	if err != nil {
		return nil
	}
	return trimEmptyLines(strings.Split(strings.TrimRight(out, "\n"), "\n"))
}

func distributeLines(rendered [][]string, start int, inputCount int, outLines []string) {
	if len(outLines) <= inputCount {
		for j := range inputCount {
			if j < len(outLines) {
				rendered[start+j] = []string{outLines[j]}
			} else {
				rendered[start+j] = []string{""}
			}
		}
	} else {
		linesPerInput := len(outLines) / inputCount
		extra := len(outLines) % inputCount
		outIdx := 0
		for j := range inputCount {
			count := linesPerInput
			if j < extra {
				count++
			}
			rendered[start+j] = outLines[outIdx : outIdx+count]
			outIdx += count
		}
	}
}

// renderTextBlockWithGlamour renders a text block using glamour.
// Single lines are rendered individually. Code blocks and tables are
// rendered as groups so glamour can produce syntax highlighting and borders.
func (m Model) renderTextBlockWithGlamour(blockIdx int, block editor.Block) renderedBlock {
	if len(block.Lines) == 0 {
		return renderedBlock{lines: [][]string{{""}}}
	}

	var contentStartIdx int
	var hasInlineMath func(lineIdx int) bool

	if m.ParsedDoc != nil && blockIdx < len(m.ParsedDoc.Blocks) {
		parsedBlock := m.ParsedDoc.Blocks[blockIdx]
		contentStartIdx = parsedBlock.FrontMatterEnd
		hasInlineMath = parsedBlock.HasInlineMath
	} else {
		if len(block.Lines) >= 3 && strings.TrimSpace(block.Lines[0]) == "---" {
			for i := 1; i < len(block.Lines); i++ {
				if strings.TrimSpace(block.Lines[i]) == "---" {
					contentStartIdx = i + 1
					for contentStartIdx < len(block.Lines) && strings.TrimSpace(block.Lines[contentStartIdx]) == "" {
						contentStartIdx++
					}
					break
				}
			}
		}
		hasInlineMath = func(lineIdx int) bool {
			if lineIdx >= len(block.Lines) {
				return false
			}
			return len(inlineMathRe.FindStringIndex(block.Lines[lineIdx])) > 0
		}
	}

	rendered := make([][]string, len(block.Lines))
	for i := 0; i < contentStartIdx && i < len(block.Lines); i++ {
		rendered[i] = nil
	}

	i := contentStartIdx
	for i < len(block.Lines) {
		line := block.Lines[i]
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			rendered[i] = []string{""}
			i++
			continue
		}

		// Fenced code block
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			fenceChar := string(rune(trimmed[0]))
			end := i + 1
			closed := false
			for end < len(block.Lines) {
				et := strings.TrimSpace(block.Lines[end])
				if len(et) >= 3 && strings.TrimLeft(et, fenceChar) == "" {
					closed = true
					end++
					break
				}
				end++
			}

			regionLines := make([]string, end-i)
			copy(regionLines, block.Lines[i:end])
			if !closed {
				regionLines = append(regionLines, fenceChar+fenceChar+fenceChar)
			}

			outLines := renderGlamourLines(strings.Join(regionLines, "\n"))
			if outLines == nil {
				for j := i; j < end; j++ {
					rendered[j] = []string{block.Lines[j]}
				}
			} else {
				rendered[i] = []string{block.Lines[i]}
				closingIdx := end - 1
				if closed && closingIdx > i {
					rendered[closingIdx] = []string{block.Lines[closingIdx]}
				}
				contentStart := i + 1
				contentEnd := end
				if closed {
					contentEnd = end - 1
				}
				for j := contentStart; j < contentEnd; j++ {
					outIdx := j - contentStart
					if outIdx < len(outLines) {
						rendered[j] = []string{outLines[outIdx]}
					} else {
						rendered[j] = []string{block.Lines[j]}
					}
				}
			}
			i = end
			continue
		}

		// Table (contiguous lines starting with |)
		if strings.HasPrefix(trimmed, "|") {
			end := i + 1
			for end < len(block.Lines) && strings.HasPrefix(strings.TrimSpace(block.Lines[end]), "|") {
				end++
			}
			if end-i >= 2 {
				outLines := renderGlamourLines(strings.Join(block.Lines[i:end], "\n"))
				if outLines == nil {
					for j := i; j < end; j++ {
						rendered[j] = []string{block.Lines[j]}
					}
				} else {
					distributeLines(rendered, i, end-i, outLines)
				}
				i = end
				continue
			}
		}

		// Single line
		out, err := markdownRender.Render(line)
		if err != nil {
			rendered[i] = []string{line}
			i++
			continue
		}
		out = strings.TrimRight(out, "\n")
		parts := strings.Split(out, "\n")
		rendered[i] = []string{""}
		for _, p := range parts {
			if strings.TrimSpace(p) != "" {
				rendered[i] = []string{p}
				break
			}
		}
		i++
	}

	return renderedBlock{
		lines:             rendered,
		contentStartIdx:   contentStartIdx,
		inlineMathChecker: hasInlineMath,
	}
}

func (m Model) View() tea.View {
	if m.width == 0 {
		return tea.NewView("loading...")
	}

	statusLine := m.RenderStatusline()
	// Statusline is always 1 line tall
	renderContentHeight := max(m.height-1, 0)

	// Calculate file tree width offset
	fileTreeOffset := 0
	if m.ShowFileTree {
		fileTreeOffset = m.FileTree.Width + 1 // +1 for separator
	}

	// Handle empty editor state (no file open)
	if m.CurrentFile == "" {
		var contentBuilder strings.Builder
		for i := 0; i < renderContentHeight; i++ {
			contentBuilder.WriteString(gutterStyle.Render(" ~"))
			contentBuilder.WriteString("\n")
		}
		contentStr := strings.TrimSuffix(contentBuilder.String(), "\n")

		var view string
		if m.ShowFileTree {
			fileTreeContent := m.FileTree.Render(renderContentHeight)
			fileTreeRender := clearStyle.
				Width(m.FileTree.Width).
				Render(fileTreeContent)

			separator := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#45475a")).
				Render("│")

			editorContent := clearStyle.
				MaxWidth(m.width - fileTreeOffset).
				PaddingLeft(2).
				Render(contentStr)

			mainContent := lipgloss.JoinHorizontal(lipgloss.Top, fileTreeRender, separator, editorContent)
			view = lipgloss.JoinVertical(lipgloss.Top, mainContent, statusLine)
		} else {
			renderContent := clearStyle.
				MaxWidth(m.width).
				PaddingLeft(2).
				Render(contentStr)
			view = lipgloss.JoinVertical(lipgloss.Top, renderContent, statusLine)
		}

		// Render dialogs on top of the view
		var cursorConfig tea.Cursor
		if m.mode == Command {
			view, cursorConfig = RenderCommandBar(m, view)
		} else if m.mode == Help {
			view, cursorConfig = m.HelpDialog.Render(view, m)
		} else if m.mode == NewNote {
			view, cursorConfig = m.NewNoteDialog.Render(view, m)
		} else if m.mode == DeleteConfirm {
			view, cursorConfig = m.DeleteConfirmDialog.Render(view, m)
		}

		// Render autocomplete if active
		if m.Autocomplete.IsActive() {
			m.Autocomplete.SetPosition(2, 3)
			view = m.Autocomplete.Render(view, m)
		}

		v := tea.NewView(view)
		v.AltScreen = true
		// In empty state, only show cursor for Command/NewNote modes
		// Help mode and Normal mode (no content) have no cursor
		if (m.mode == Command || m.mode == NewNote) && cursorConfig.Shape != 0 {
			v.Cursor = &cursorConfig
		} else {
			v.Cursor = nil
		}
		return v
	}

	totalLines := 0
	for _, block := range m.Editor.Blocks {
		totalLines += len(block.Lines)
	}
	gutterWidth := len(fmt.Sprint(totalLines))

	// Calculate available content width after accounting for padding, gutter, and file tree
	// Layout: [padding 2][indicator 1][line numbers gutterWidth+2][content]
	contentWidth := m.width - 5 - gutterWidth - fileTreeOffset
	if contentWidth < 1 {
		contentWidth = 1
	}

	// Calculate absolute offset line number
	offsetAbsLine := 0
	for i := 0; i < m.Editor.Offset.BlockIdx && i < len(m.Editor.Blocks); i++ {
		offsetAbsLine += len(m.Editor.Blocks[i].Lines)
	}
	offsetAbsLine += m.Editor.Offset.LineIdx

	var contentBuilder strings.Builder
	globalLineIdx := 0
	visualLinesRendered := 0

	// Track visual line mapping for cursor positioning
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
				// Skip lines before offset
				if globalLineIdx < offsetAbsLine {
					globalLineIdx++
					continue
				}
				// Stop if viewport is full
				if visualLinesRendered >= renderContentHeight {
					globalLineIdx++
					continue
				}

				lineNum := globalLineIdx + 1
				lineNumStr := fmt.Sprintf(" %*d ", gutterWidth, lineNum)
				contentBuilder.WriteString(indicator)
				contentBuilder.WriteString(gutterStyle.Render(lineNumStr))
				// For math blocks, we can't horizontally scroll the image, so skip horizontal offset
				contentBuilder.WriteString(latex.PlaceholderRow(block.ImageID, uint16(i), block.ImageCols))
				contentBuilder.WriteString("\n")
				globalLineIdx++
				visualLinesRendered++
			}
		} else if useMarkdown {
			rendered := m.renderTextBlockWithGlamour(blockIdx, block)
			visualLineMap[blockIdx] = make([]int, len(block.Lines))

			for lineIdx, lineStr := range block.Lines {
				isCursorLine := isBlockActive && lineIdx == m.Editor.Cursor.LineIdx
				hasInlineMath := rendered.inlineMathChecker != nil && rendered.inlineMathChecker(lineIdx)

				var visualLines []string
				if isCursorLine || hasInlineMath {
					visualLines = []string{editor.ExpandTabs(m.applyInlinePlaceholders(blockIdx, lineIdx, lineStr))}
				} else if lineIdx >= rendered.contentStartIdx && rendered.lines[lineIdx] != nil && len(rendered.lines[lineIdx]) > 0 {
					visualLines = rendered.lines[lineIdx]
				} else {
					visualLines = []string{editor.ExpandTabs(lineStr)}
				}
				if len(visualLines) == 0 {
					visualLines = []string{""}
				}

				visualLineMap[blockIdx][lineIdx] = len(visualLines)

				for vIdx, vLine := range visualLines {
					// Skip lines before offset
					if globalLineIdx < offsetAbsLine {
						globalLineIdx++
						continue
					}
					// Stop if viewport is full
					if visualLinesRendered >= renderContentHeight {
						globalLineIdx++
						continue
					}

					lineNum := globalLineIdx + 1
					lineNumStr := fmt.Sprintf(" %*d ", gutterWidth, lineNum)

					var styledGutter string
					if vIdx == 0 {
						if isBlockActive && lineIdx == m.Editor.Cursor.LineIdx {
							styledGutter = currentLineStyle.Render(lineNumStr)
						} else {
							styledGutter = gutterStyle.Render(lineNumStr)
						}
					} else {
						styledGutter = gutterStyle.Render(strings.Repeat(" ", gutterWidth+2))
					}
					contentBuilder.WriteString(indicator)
					contentBuilder.WriteString(styledGutter)
					// Apply horizontal scroll offset and truncate to content width
					contentBuilder.WriteString(truncateLine(vLine, contentWidth))
					contentBuilder.WriteString("\n")
					globalLineIdx++
					visualLinesRendered++
				}
			}
		} else {
			visualLineMap[blockIdx] = make([]int, height)
			for i := range height {
				visualLineMap[blockIdx][i] = 1
			}
			for lineIdx, lineStr := range block.Lines {
				// Skip lines before offset
				if globalLineIdx < offsetAbsLine {
					globalLineIdx++
					continue
				}
				// Stop if viewport is full
				if visualLinesRendered >= renderContentHeight {
					globalLineIdx++
					continue
				}

			shouldBlank := !(m.mode == Insert && isBlockActive)

			// Check if cursor is on a tab in Normal mode
			isOnTab := m.mode == Normal && isBlockActive && lineIdx == m.Editor.Cursor.LineIdx &&
				m.Editor.Cursor.Col < len([]rune(lineStr)) && len([]rune(lineStr)) > 0 &&
				[]rune(lineStr)[m.Editor.Cursor.Col] == '\t'

			if shouldBlank && block.Type == editor.TextBlock {
				lineStr = m.applyInlinePlaceholders(blockIdx, lineIdx, lineStr)
			}

			// Expand tabs with highlighting if cursor is on a tab
			if isOnTab {
				lineStr = renderLineWithTabHighlight(lineStr, m.Editor.Cursor.Col, tabHighlightStyle)
			} else {
				lineStr = editor.ExpandTabs(lineStr)
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
				// Apply horizontal scroll offset and truncate to content width
				contentBuilder.WriteString(truncateLine(lineStr, contentWidth))
				contentBuilder.WriteString("\n")
				globalLineIdx++
				visualLinesRendered++
			}
		}
	}

	// Remove trailing newline if present
	contentStr := strings.TrimSuffix(contentBuilder.String(), "\n")

	// Pad content to fill viewport height using the tracked count
	for visualLinesRendered < renderContentHeight {
		contentStr += "\n"
		visualLinesRendered++
	}

	// Render file tree if visible
	var view string
	if m.ShowFileTree {
		fileTreeContent := m.FileTree.Render(renderContentHeight)
		fileTreeRender := clearStyle.
			Width(m.FileTree.Width).
			Render(fileTreeContent)

		separator := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#45475a")).
			Render("│")

		editorContent := clearStyle.
			MaxWidth(m.width - fileTreeOffset).
			PaddingLeft(2).
			Render(contentStr)

		mainContent := lipgloss.JoinHorizontal(lipgloss.Top, fileTreeRender, separator, editorContent)
		view = lipgloss.JoinVertical(lipgloss.Top, mainContent, statusLine)
	} else {
		renderContent := clearStyle.
			MaxWidth(m.width).
			PaddingLeft(2).
			Render(contentStr)
		view = lipgloss.JoinVertical(lipgloss.Top, renderContent, statusLine)
	}

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

	// Adjust cursor Y for vertical scroll offset
	cursorY -= offsetAbsLine
	if cursorY < 0 {
		cursorY = 0
	}

	// Cursor X calculation
	cursorX := 2 + gutterWidth + 3
	if m.ShowFileTree {
		cursorX += fileTreeOffset
	}
	if cursorBlockIdx < len(m.Editor.Blocks) && cursorLineIdx < len(m.Editor.Blocks[cursorBlockIdx].Lines) {
		lineLen := len(m.Editor.Blocks[cursorBlockIdx].Lines[cursorLineIdx])
		if cursorCol > lineLen {
			cursorCol = lineLen
		}
		// Convert rune column to visual column (tabs expand to 4 spaces)
		line := m.Editor.Blocks[cursorBlockIdx].Lines[cursorLineIdx]
		visualCol := editor.RuneColToVisualCol(line, cursorCol)
		cursorX += visualCol
	}

	var cursorConfig tea.Cursor
	// When file tree is focused, cursor is hidden (selection shown via highlight)
	if m.mode == Command {
		view, cursorConfig = RenderCommandBar(m, view)
	} else if m.mode == NewNote {
		view, cursorConfig = m.NewNoteDialog.Render(view, m)
	} else if m.mode == Help {
		view, cursorConfig = m.HelpDialog.Render(view, m)
	} else if m.mode == DeleteConfirm {
		view, cursorConfig = m.DeleteConfirmDialog.Render(view, m)
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

	// Render autocomplete if active (in Insert mode)
	if m.Autocomplete.IsActive() {
		m.Autocomplete.SetPosition(cursorX, cursorY+1)
		view = m.Autocomplete.Render(view, m)
	}

	v := tea.NewView(view)
	v.AltScreen = true
	// Cursor visibility logic:
	// - Help mode: no cursor
	// - DeleteConfirm mode: no cursor
	// - File tree focused in Normal mode: no cursor (selection shown via highlight)
	// - Otherwise: show cursor
	if m.mode == Help || m.mode == DeleteConfirm {
		v.Cursor = nil
	} else if m.ShowFileTree && m.FileTree.Focused && m.mode == Normal {
		v.Cursor = nil
	} else {
		v.Cursor = &cursorConfig
	}
	return v
}

// truncateLine truncates a line to fit within maxChars, preserving ANSI escape codes.
func truncateLine(line string, maxChars int) string {
	if maxChars <= 0 {
		return ""
	}
	return ansi.Truncate(line, maxChars, "")
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

// renderLineWithTabHighlight expands tabs and highlights the tab at cursor position
func renderLineWithTabHighlight(line string, cursorCol int, highlightStyle lipgloss.Style) string {
	runes := []rune(line)
	var result strings.Builder

	for i, r := range runes {
		if r == '\t' {
			if i == cursorCol {
				result.WriteString(highlightStyle.Render("    "))
			} else {
				result.WriteString("    ")
			}
		} else {
			result.WriteString(string(r))
		}
	}

	return result.String()
}
