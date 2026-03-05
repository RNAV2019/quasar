package ui

import (
	"fmt"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/RNAV2019/quasar/internal/editor"
	"github.com/RNAV2019/quasar/internal/latex"
	"github.com/RNAV2019/quasar/internal/styles"
	"github.com/RNAV2019/quasar/internal/ui/dialog"
	"github.com/RNAV2019/quasar/internal/ui/layout"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/x/ansi"
)

var (
	markdownRender *glamour.TermRenderer
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

// layoutParams builds layout.Params from the current model state.
func (m Model) layoutParams(contentHeight int, contentView, statusLine string) layout.Params {
	p := layout.Params{
		Width:         m.width,
		ContentHeight: contentHeight,
		ShowFileTree:  m.ShowFileTree,
		ContentView:   contentView,
		StatusLine:    statusLine,
	}
	if m.ShowFileTree {
		p.FileTreeWidth = m.FileTree.Width
		p.FileTreeView = m.FileTree.Render(contentHeight)
	}
	return p
}

// renderEmptyState renders the view when no file is open.
func (m Model) renderEmptyState(contentHeight int, statusLine string) string {
	var contentBuilder strings.Builder
	for range contentHeight {
		contentBuilder.WriteString(styles.GutterStyle.Render(" ~"))
		contentBuilder.WriteString("\n")
	}
	contentStr := strings.TrimSuffix(contentBuilder.String(), "\n")
	return layout.Render(m.layoutParams(contentHeight, contentStr, statusLine))
}

// renderLoadingState renders the view while compiling images.
func (m Model) renderLoadingState(contentHeight int, statusLine string) string {
	fileTreeOffset := 0
	if m.ShowFileTree {
		fileTreeOffset = m.FileTree.Width + 1
	}

	loadingStyle := lipgloss.NewStyle().
		Foreground(styles.ColorBlue).
		Bold(true)

	contentWidth := m.width - fileTreeOffset - 5
	loadingText := loadingStyle.Render("Compiling images...")
	centeredText := lipgloss.NewStyle().
		Width(contentWidth).
		Align(lipgloss.Center).
		Render(loadingText)

	var contentBuilder strings.Builder
	for i := range contentHeight {
		if i == contentHeight/2 {
			contentBuilder.WriteString(styles.GutterStyle.Render("   "))
			contentBuilder.WriteString(centeredText)
		}
		contentBuilder.WriteString("\n")
	}
	contentStr := strings.TrimSuffix(contentBuilder.String(), "\n")
	return layout.Render(m.layoutParams(contentHeight, contentStr, statusLine))
}

// renderDialogOverlay renders whichever dialog is active on top of the view.
func (m Model) renderDialogOverlay(view string) (string, tea.Cursor) {
	dim := dialog.Dimensions{Width: m.width, Height: m.height}
	switch m.mode {
	case Command:
		return dialog.RenderCommandBar(m.CmdInput, view, dim)
	case Help:
		return m.HelpDialog.Render(view, dim)
	case Error:
		return m.ErrorDialog.Render(view, dim)
	case NewNote:
		return m.NewNoteDialog.Render(view, dim)
	case DeleteConfirm:
		return m.DeleteConfirmDialog.Render(view, dim)
	case QuitConfirm:
		return m.QuitConfirmDialog.Render(view, dim)
	case FileTreeDelete:
		return m.FileTreeDeleteDialog.Render(view, dim)
	case FileTreeRename:
		return m.RenameDialog.Render(view, dim)
	default:
		return view, tea.Cursor{}
	}
}

// calculateCursor computes the cursor position from the visual line map.
func (m Model) calculateCursor(visualLineMap map[int][]int, offsetAbsLine, gutterWidth, fileTreeOffset int) (int, int) {
	cursorBlockIdx := m.Editor.Cursor.BlockIdx
	cursorLineIdx := m.Editor.Cursor.LineIdx
	cursorCol := m.Editor.Cursor.Col

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
		line := m.Editor.Blocks[cursorBlockIdx].Lines[cursorLineIdx]
		visualCol := editor.RuneColToVisualCol(line, cursorCol)
		cursorX += visualCol
	}

	return cursorX, cursorY
}

// View renders the complete TUI frame including content, dialogs, and cursor.
func (m Model) View() tea.View {
	if m.width == 0 {
		return tea.NewView("loading...")
	}

	statusLine := m.RenderStatusline()
	renderContentHeight := max(m.height-1, 0)

	// Handle empty editor state (no file open)
	if m.CurrentFile == "" {
		view := m.renderEmptyState(renderContentHeight, statusLine)

		var cursorConfig tea.Cursor
		view, cursorConfig = m.renderDialogOverlay(view)

		if m.Autocomplete.IsActive() {
			m.Autocomplete.SetPosition(2, 3)
			view = m.Autocomplete.Render(view)
		}

		v := tea.NewView(view)
		v.AltScreen = true
		if (m.mode == Command || m.mode == NewNote || m.mode == FileTreeRename) && cursorConfig.Shape != 0 {
			v.Cursor = &cursorConfig
		} else {
			v.Cursor = nil
		}
		return v
	}

	// Handle document loading state
	if m.DocumentLoading {
		view := m.renderLoadingState(renderContentHeight, statusLine)
		v := tea.NewView(view)
		v.AltScreen = true
		v.Cursor = nil
		return v
	}

	fileTreeOffset := 0
	if m.ShowFileTree {
		fileTreeOffset = m.FileTree.Width + 1
	}

	totalLines := 0
	for _, block := range m.Editor.Blocks {
		totalLines += len(block.Lines)
	}
	gutterWidth := len(fmt.Sprint(totalLines))
	contentWidth := max(m.width-5-gutterWidth-fileTreeOffset, 1)

	// Calculate absolute offset line number
	offsetAbsLine := 0
	for i := 0; i < m.Editor.Offset.BlockIdx && i < len(m.Editor.Blocks); i++ {
		offsetAbsLine += len(m.Editor.Blocks[i].Lines)
	}
	offsetAbsLine += m.Editor.Offset.LineIdx

	var contentBuilder strings.Builder
	globalLineIdx := 0
	visualLinesRendered := 0
	visualLineMap := make(map[int][]int)

	for blockIdx, block := range m.Editor.Blocks {
		isBlockActive := blockIdx == m.Editor.Cursor.BlockIdx
		useMarkdown := m.mode == Normal && block.Type == editor.TextBlock

		var indicator string
		if block.HasError {
			indicator = styles.ErrorGutterIndicator
		} else if block.Type == editor.MathBlock {
			indicator = styles.MathGutterIndicator
		} else {
			indicator = styles.TextGutterIndicator
		}

		height := len(block.Lines)

		if !isBlockActive && block.Type == editor.MathBlock && block.ImageID != 0 {
			displayHeight := block.ImageHeight
			if displayHeight < height {
				displayHeight = height
			}
			visualLineMap[blockIdx] = make([]int, displayHeight)
			for i := range displayHeight {
				visualLineMap[blockIdx][i] = 1
			}
			for i := range displayHeight {
				if globalLineIdx < offsetAbsLine {
					globalLineIdx++
					continue
				}
				if visualLinesRendered >= renderContentHeight {
					globalLineIdx++
					continue
				}

				lineNum := globalLineIdx + 1
				lineNumStr := fmt.Sprintf(" %*d ", gutterWidth, lineNum)
				contentBuilder.WriteString(indicator)
				contentBuilder.WriteString(styles.GutterStyle.Render(lineNumStr))
				contentBuilder.WriteString(latex.PlaceholderRow(block.ImageID, uint16(i), block.ImageCols))
				contentBuilder.WriteString("\n")
				globalLineIdx++
				visualLinesRendered++
			}
		} else if !isBlockActive && block.Type == editor.MathBlock && block.IsLoading {
			visualLineMap[blockIdx] = make([]int, height)
			for i := range height {
				visualLineMap[blockIdx][i] = 1
			}
			for range height {
				if globalLineIdx < offsetAbsLine {
					globalLineIdx++
					continue
				}
				if visualLinesRendered >= renderContentHeight {
					globalLineIdx++
					continue
				}

				lineNum := globalLineIdx + 1
				lineNumStr := fmt.Sprintf(" %*d ", gutterWidth, lineNum)
				contentBuilder.WriteString(indicator)
				contentBuilder.WriteString(styles.GutterStyle.Render(lineNumStr))
				contentBuilder.WriteString(styles.DimStyle.Render("⋯"))
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
					if globalLineIdx < offsetAbsLine {
						globalLineIdx++
						continue
					}
					if visualLinesRendered >= renderContentHeight {
						globalLineIdx++
						continue
					}

					lineNum := globalLineIdx + 1
					lineNumStr := fmt.Sprintf(" %*d ", gutterWidth, lineNum)

					var styledGutter string
					if vIdx == 0 {
						if isBlockActive && lineIdx == m.Editor.Cursor.LineIdx {
							styledGutter = styles.CurrentLineStyle.Render(lineNumStr)
						} else {
							styledGutter = styles.GutterStyle.Render(lineNumStr)
						}
					} else {
						styledGutter = styles.GutterStyle.Render(strings.Repeat(" ", gutterWidth+2))
					}
					contentBuilder.WriteString(indicator)
					contentBuilder.WriteString(styledGutter)
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
				if globalLineIdx < offsetAbsLine {
					globalLineIdx++
					continue
				}
				if visualLinesRendered >= renderContentHeight {
					globalLineIdx++
					continue
				}

				shouldBlank := !(m.mode == Insert && isBlockActive)

				isOnTab := m.mode == Normal && isBlockActive && lineIdx == m.Editor.Cursor.LineIdx &&
					m.Editor.Cursor.Col < len([]rune(lineStr)) && len([]rune(lineStr)) > 0 &&
					[]rune(lineStr)[m.Editor.Cursor.Col] == '\t'

				if shouldBlank && block.Type == editor.TextBlock {
					lineStr = m.applyInlinePlaceholders(blockIdx, lineIdx, lineStr)
				}

				if isOnTab {
					lineStr = renderLineWithTabHighlight(lineStr, m.Editor.Cursor.Col, styles.TabHighlightStyle)
				} else {
					lineStr = editor.ExpandTabs(lineStr)
				}

				lineStr = m.applySelectionHighlighting(lineStr, blockIdx, lineIdx)

				lineNum := globalLineIdx + 1
				lineNumStr := fmt.Sprintf(" %*d ", gutterWidth, lineNum)

				var styledGutter string
				if isBlockActive && lineIdx == m.Editor.Cursor.LineIdx {
					styledGutter = styles.CurrentLineStyle.Render(lineNumStr)
				} else {
					styledGutter = styles.GutterStyle.Render(lineNumStr)
				}
				contentBuilder.WriteString(indicator)
				contentBuilder.WriteString(styledGutter)
				contentBuilder.WriteString(truncateLine(lineStr, contentWidth))
				contentBuilder.WriteString("\n")
				globalLineIdx++
				visualLinesRendered++
			}
		}
	}

	contentStr := strings.TrimSuffix(contentBuilder.String(), "\n")

	for visualLinesRendered < renderContentHeight {
		contentStr += "\n"
		visualLinesRendered++
	}

	view := layout.Render(m.layoutParams(renderContentHeight, contentStr, statusLine))

	cursorX, cursorY := m.calculateCursor(visualLineMap, offsetAbsLine, gutterWidth, fileTreeOffset)

	var cursorConfig tea.Cursor
	view, cursorConfig = m.renderDialogOverlay(view)

	// For non-dialog modes, set editor cursor
	if cursorConfig.Shape == 0 {
		if m.mode == Insert {
			cursorConfig = tea.Cursor{
				Position: tea.Position{X: cursorX, Y: cursorY},
				Shape:    tea.CursorBar,
				Color:    lipgloss.White,
			}
		} else {
			cursorConfig = tea.Cursor{
				Position: tea.Position{X: cursorX, Y: cursorY},
				Shape:    tea.CursorBlock,
				Blink:    false,
				Color:    lipgloss.White,
			}
		}
	}

	if m.Autocomplete.IsActive() {
		m.Autocomplete.SetPosition(cursorX, cursorY+1)
		view = m.Autocomplete.Render(view)
	}

	v := tea.NewView(view)
	v.AltScreen = true
	if m.mode == Help || m.mode == Error || m.mode == DeleteConfirm || m.mode == QuitConfirm || m.mode == FileTreeDelete {
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
			m.Editor.Cursor.Col >= startCol && m.Editor.Cursor.Col < startCol+render.TextLength

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
			end := min(match.startCol+match.render.TextLength, len(runes))
			result.WriteString(string(runes[match.startCol:end]))
		} else {
			result.WriteString(latex.PlaceholderRow(match.render.ImageID, 0, match.render.Length))
		}
		pos = match.startCol + match.render.TextLength
	}

	if pos < len(runes) {
		result.WriteString(string(runes[pos:]))
	}

	return result.String()
}

// renderLineWithTabHighlight expands tabs and highlights the tab at cursor position.
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

// isPositionSelected checks if a given block/line/col is within the current selection.
func (m Model) isPositionSelected(blockIdx, lineIdx, col int) bool {
	if !m.Editor.Selection.Active {
		return false
	}

	start := m.Editor.Selection.Start
	end := m.Editor.Selection.End

	if start.BlockIdx > end.BlockIdx ||
		(start.BlockIdx == end.BlockIdx && start.LineIdx > end.LineIdx) ||
		(start.BlockIdx == end.BlockIdx && start.LineIdx == end.LineIdx && start.Col > end.Col) {
		start, end = end, start
	}

	pos := editor.Position{BlockIdx: blockIdx, LineIdx: lineIdx, Col: col}

	if pos.BlockIdx < start.BlockIdx || pos.BlockIdx > end.BlockIdx {
		return false
	}
	if pos.BlockIdx == start.BlockIdx && pos.BlockIdx == end.BlockIdx {
		if pos.LineIdx < start.LineIdx || pos.LineIdx > end.LineIdx {
			return false
		}
		if pos.LineIdx == start.LineIdx && pos.LineIdx == end.LineIdx {
			return pos.Col >= start.Col && pos.Col < end.Col
		}
		if pos.LineIdx == start.LineIdx {
			return pos.Col >= start.Col
		}
		if pos.LineIdx == end.LineIdx {
			return pos.Col < end.Col
		}
		return true
	}
	if pos.BlockIdx == start.BlockIdx {
		if pos.LineIdx < start.LineIdx {
			return false
		}
		if pos.LineIdx == start.LineIdx {
			return pos.Col >= start.Col
		}
		return true
	}
	if pos.BlockIdx == end.BlockIdx {
		if pos.LineIdx > end.LineIdx {
			return false
		}
		if pos.LineIdx == end.LineIdx {
			return pos.Col < end.Col
		}
		return true
	}
	return true
}

// applySelectionHighlighting applies selection highlighting to selected portions of the line.
func (m Model) applySelectionHighlighting(line string, blockIdx, lineIdx int) string {
	if !m.Editor.Selection.Active {
		return line
	}

	runes := []rune(line)
	var result strings.Builder
	inSelection := false
	selectionStart := 0

	for col := 0; col <= len(runes); col++ {
		selected := col < len(runes) && m.isPositionSelected(blockIdx, lineIdx, col)

		if selected && !inSelection {
			if col > selectionStart {
				result.WriteString(string(runes[selectionStart:col]))
			}
			inSelection = true
			selectionStart = col
		} else if !selected && inSelection {
			selectedText := string(runes[selectionStart:col])
			result.WriteString(styles.SelectionStyle.Render(selectedText))
			inSelection = false
			selectionStart = col
		}
	}

	if inSelection && selectionStart < len(runes) {
		selectedText := string(runes[selectionStart:])
		result.WriteString(styles.SelectionStyle.Render(selectedText))
	} else if selectionStart < len(runes) {
		result.WriteString(string(runes[selectionStart:]))
	}

	return result.String()
}
