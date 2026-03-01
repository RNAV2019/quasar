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
	lines             [][]string
	contentStartIdx   int
	inlineMathChecker func(lineIdx int) bool
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
		for j := 0; j < inputCount; j++ {
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
		for j := 0; j < inputCount; j++ {
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
			rendered := m.renderTextBlockWithGlamour(blockIdx, block)
			visualLineMap[blockIdx] = make([]int, len(block.Lines))

			for lineIdx, lineStr := range block.Lines {
				isCursorLine := isBlockActive && lineIdx == m.Editor.Cursor.LineIdx
				hasInlineMath := rendered.inlineMathChecker != nil && rendered.inlineMathChecker(lineIdx)

				var visualLines []string
				if isCursorLine || hasInlineMath {
					visualLines = []string{m.applyInlinePlaceholders(blockIdx, lineIdx, lineStr)}
				} else if lineIdx >= rendered.contentStartIdx && rendered.lines[lineIdx] != nil && len(rendered.lines[lineIdx]) > 0 {
					visualLines = rendered.lines[lineIdx]
				} else {
					visualLines = []string{lineStr}
				}
				if len(visualLines) == 0 {
					visualLines = []string{""}
				}

				visualLineMap[blockIdx][lineIdx] = len(visualLines)

				for vIdx, vLine := range visualLines {
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
					contentBuilder.WriteString(vLine)
					contentBuilder.WriteString("\n")
					globalLineIdx++
				}
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
