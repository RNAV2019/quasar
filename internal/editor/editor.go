package editor

import "strconv"

// MoveCursor moves the cursor by the specified delta.
func (m *Model) MoveCursor(lineDelta, colDelta int) {
	m.Cursor.Col += colDelta
	m.Cursor.LineIdx += lineDelta

	if m.Cursor.LineIdx < 0 {
		if m.Cursor.BlockIdx > 0 {
			m.Cursor.BlockIdx--
			m.Cursor.LineIdx = len(m.Blocks[m.Cursor.BlockIdx].Lines) - 1
		} else {
			m.Cursor.LineIdx = 0
		}
	} else if m.Cursor.LineIdx >= len(m.Blocks[m.Cursor.BlockIdx].Lines) {
		if m.Cursor.BlockIdx < len(m.Blocks)-1 {
			m.Cursor.BlockIdx++
			m.Cursor.LineIdx = 0
		} else {
			m.Cursor.LineIdx = len(m.Blocks[m.Cursor.BlockIdx].Lines) - 1
		}
	}

	if m.Cursor.LineIdx >= 0 && m.Cursor.LineIdx < len(m.Blocks[m.Cursor.BlockIdx].Lines) {
		lineLen := len(m.Blocks[m.Cursor.BlockIdx].Lines[m.Cursor.LineIdx])
		if m.Cursor.Col > lineLen {
			m.Cursor.Col = lineLen
		}
	}
	if m.Cursor.Col < 0 {
		m.Cursor.Col = 0
	}

	m.ensureCursorInView()
}

// EndOfLine moves the cursor to the end of the current line.
func (m *Model) EndOfLine() {
	lineLen := len(m.Blocks[m.Cursor.BlockIdx].Lines[m.Cursor.LineIdx])
	m.Cursor.Col = lineLen
	m.ensureCursorInView()
}

func (m *Model) ensureCursorInView() {
	if m.Height <= 0 {
		return
	}

	totalLines := 0
	for _, block := range m.Blocks {
		totalLines += len(block.Lines)
	}

	m.Offset.Col = 0

	// Reserve space at bottom so last line isn't right against statusline
	const bottomPadding = 3
	viewHeight := m.Height - bottomPadding
	if viewHeight < 1 {
		viewHeight = 1
	}

	if totalLines <= viewHeight {
		m.Offset.BlockIdx = 0
		m.Offset.LineIdx = 0
		return
	}

	cursorAbsLine := 0
	for i := 0; i < m.Cursor.BlockIdx && i < len(m.Blocks); i++ {
		cursorAbsLine += len(m.Blocks[i].Lines)
	}
	cursorAbsLine += m.Cursor.LineIdx

	offsetAbsLine := 0
	for i := 0; i < m.Offset.BlockIdx && i < len(m.Blocks); i++ {
		offsetAbsLine += len(m.Blocks[i].Lines)
	}
	offsetAbsLine += m.Offset.LineIdx

	if cursorAbsLine < offsetAbsLine {
		m.Offset.BlockIdx = m.Cursor.BlockIdx
		m.Offset.LineIdx = m.Cursor.LineIdx
	}
	if cursorAbsLine >= offsetAbsLine+viewHeight {
		targetAbsLine := cursorAbsLine - viewHeight + 1
		if targetAbsLine < 0 {
			targetAbsLine = 0
		}
		// Limit scrolling so we don't scroll past the point where bottom padding is visible
		maxScroll := totalLines - m.Height + bottomPadding
		if maxScroll < 0 {
			maxScroll = 0
		}
		if targetAbsLine > maxScroll {
			targetAbsLine = maxScroll
		}
		m.Offset.BlockIdx = 0
		m.Offset.LineIdx = 0
		currentLine := 0
		for i := range m.Blocks {
			for j := range m.Blocks[i].Lines {
				if currentLine >= targetAbsLine {
					m.Offset.BlockIdx = i
					m.Offset.LineIdx = j
					break
				}
				currentLine++
			}
			if currentLine >= targetAbsLine {
				break
			}
		}
	}
}

// ViewLines returns all visible lines from all blocks.
func (m *Model) ViewLines() []string {
	var visibleLines []string
	for _, block := range m.Blocks {
		visibleLines = append(visibleLines, block.Lines...)
	}
	return visibleLines
}

// MaxLineLength returns the maximum allowed line length based on viewport width.
func (m *Model) MaxLineLength() int {
	totalLines := 0
	for _, block := range m.Blocks {
		totalLines += len(block.Lines)
	}
	gutterWidth := len(strconv.Itoa(totalLines))
	maxLen := m.Width - 5 - gutterWidth - 2
	return max(maxLen, 20)
}

// GetLineCount returns the total number of lines across all blocks.
func (m *Model) GetLineCount() int {
	count := 0
	for _, block := range m.Blocks {
		count += len(block.Lines)
	}
	return count
}
