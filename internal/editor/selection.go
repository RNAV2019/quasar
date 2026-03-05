package editor

import "strings"

// SelectLine selects the current line.
func (m *Model) SelectLine() {
	m.Selection.Active = true
	m.Selection.Start = Position{BlockIdx: m.Cursor.BlockIdx, LineIdx: m.Cursor.LineIdx, Col: 0}
	m.Selection.End = Position{BlockIdx: m.Cursor.BlockIdx, LineIdx: m.Cursor.LineIdx, Col: len(m.Blocks[m.Cursor.BlockIdx].Lines[m.Cursor.LineIdx])}
	m.Selection.WasLineWise = true
}

// ClearSelection clears the current selection.
func (m *Model) ClearSelection() {
	m.Selection.Active = false
	m.Selection.WasLineWise = false
}

// HasSelection reports whether there is an active selection.
func (m *Model) HasSelection() bool {
	return m.Selection.Active && (m.Selection.Start != m.Selection.End)
}

// GetSelectedText returns the currently selected text.
func (m *Model) GetSelectedText() string {
	if !m.HasSelection() {
		return ""
	}

	start, end := m.Selection.Start, m.Selection.End
	if start.BlockIdx > end.BlockIdx || (start.BlockIdx == end.BlockIdx && m.posGreater(start, end)) {
		start, end = end, start
	}

	var result strings.Builder

	if start.BlockIdx == end.BlockIdx {
		block := m.Blocks[start.BlockIdx]
		if start.LineIdx == end.LineIdx {
			line := block.Lines[start.LineIdx]
			runes := []rune(line)
			startCol := min(start.Col, len(runes))
			endCol := min(end.Col, len(runes))
			if startCol > endCol {
				startCol, endCol = endCol, startCol
			}
			result.WriteString(string(runes[startCol:endCol]))
		} else {
			for i := start.LineIdx; i <= end.LineIdx && i < len(block.Lines); i++ {
				if i == start.LineIdx {
					line := block.Lines[i]
					runes := []rune(line)
					startCol := min(start.Col, len(runes))
					result.WriteString(string(runes[startCol:]))
				} else if i == end.LineIdx {
					line := block.Lines[i]
					runes := []rune(line)
					endCol := min(end.Col, len(runes))
					result.WriteString(string(runes[:endCol]))
				} else {
					result.WriteString(block.Lines[i])
				}
				if i < end.LineIdx {
					result.WriteString("\n")
				}
			}
		}
	} else {
		for i := start.BlockIdx; i <= end.BlockIdx && i < len(m.Blocks); i++ {
			block := m.Blocks[i]
			startLine, endLine := 0, len(block.Lines)-1
			if i == start.BlockIdx {
				startLine = start.LineIdx
			}
			if i == end.BlockIdx {
				endLine = end.LineIdx
			}
			for j := startLine; j <= endLine && j < len(block.Lines); j++ {
				if i == start.BlockIdx && j == start.LineIdx {
					line := block.Lines[j]
					runes := []rune(line)
					startCol := min(start.Col, len(runes))
					result.WriteString(string(runes[startCol:]))
				} else if i == end.BlockIdx && j == end.LineIdx {
					line := block.Lines[j]
					runes := []rune(line)
					endCol := min(end.Col, len(runes))
					result.WriteString(string(runes[:endCol]))
				} else {
					result.WriteString(block.Lines[j])
				}
				if !(i == end.BlockIdx && j == end.LineIdx) {
					result.WriteString("\n")
				}
			}
		}
	}

	return result.String()
}

func (m *Model) posGreater(a, b Position) bool {
	if a.BlockIdx != b.BlockIdx {
		return a.BlockIdx > b.BlockIdx
	}
	if a.LineIdx != b.LineIdx {
		return a.LineIdx > b.LineIdx
	}
	return a.Col > b.Col
}

// YankLine copies the current line to the clipboard.
func (m *Model) YankLine() (string, bool) {
	if m.Cursor.BlockIdx >= len(m.Blocks) {
		return "", false
	}
	block := m.Blocks[m.Cursor.BlockIdx]
	if m.Cursor.LineIdx >= len(block.Lines) {
		return "", false
	}
	return block.Lines[m.Cursor.LineIdx], true
}

// YankSelection returns the selected text and whether it was line-wise.
func (m *Model) YankSelection() (string, bool, bool) {
	if !m.HasSelection() {
		return "", false, false
	}
	text := m.GetSelectedText()
	isLineWise := m.Selection.WasLineWise
	return text, isLineWise, true
}

// SelectWord selects the current word under the cursor.
func (m *Model) SelectWord() {
	if m.Cursor.BlockIdx >= len(m.Blocks) {
		return
	}
	block := m.Blocks[m.Cursor.BlockIdx]
	if m.Cursor.LineIdx >= len(block.Lines) {
		return
	}

	line := block.Lines[m.Cursor.LineIdx]
	runes := []rune(line)

	if len(runes) == 0 {
		return
	}

	col := m.Cursor.Col
	if col >= len(runes) {
		col = len(runes) - 1
	}
	if col < 0 {
		col = 0
	}

	startCol := col
	endCol := col

	for startCol > 0 && IsWordChar(runes[startCol-1]) {
		startCol--
	}

	for endCol < len(runes) && IsWordChar(runes[endCol]) {
		endCol++
	}

	if startCol < endCol {
		m.Selection.Active = true
		m.Selection.Start = Position{BlockIdx: m.Cursor.BlockIdx, LineIdx: m.Cursor.LineIdx, Col: startCol}
		m.Selection.End = Position{BlockIdx: m.Cursor.BlockIdx, LineIdx: m.Cursor.LineIdx, Col: endCol}
		m.Selection.WasLineWise = false
	}
}

// ExtendSelection extends the selection to include the current cursor position.
func (m *Model) ExtendSelection() {
	if !m.Selection.Active {
		m.Selection.Start = m.Cursor
		m.Selection.Active = true
	}
	m.Selection.End = m.Cursor
	m.Selection.WasLineWise = false
}

// DeleteSelection deletes the selected text and returns it.
func (m *Model) DeleteSelection() (string, bool) {
	if !m.HasSelection() {
		return "", false
	}

	deleted := m.GetSelectedText()

	start, end := m.Selection.Start, m.Selection.End
	if start.BlockIdx > end.BlockIdx || (start.BlockIdx == end.BlockIdx && m.posGreater(start, end)) {
		start, end = end, start
	}

	if start.BlockIdx == end.BlockIdx {
		block := &m.Blocks[start.BlockIdx]
		if start.LineIdx == end.LineIdx {
			line := &block.Lines[start.LineIdx]
			runes := []rune(*line)
			startCol := min(start.Col, len(runes))
			endCol := min(end.Col, len(runes))
			if startCol > endCol {
				startCol, endCol = endCol, startCol
			}
			*line = string(runes[:startCol]) + string(runes[endCol:])
			m.Cursor.Col = startCol
		} else {
			newLines := []string{}
			for i, line := range block.Lines {
				if i < start.LineIdx {
					newLines = append(newLines, line)
				} else if i == start.LineIdx {
					runes := []rune(line)
					startCol := min(start.Col, len(runes))
					newLines = append(newLines, string(runes[:startCol]))
				} else if i > start.LineIdx && i < end.LineIdx {
					// Lines between start and end are part of selection - skip them
					continue
				} else if i == end.LineIdx {
					runes := []rune(line)
					endCol := min(end.Col, len(runes))
					newLines[len(newLines)-1] += string(runes[endCol:])
					m.Cursor.Col = len([]rune(newLines[len(newLines)-1])) - (len(runes) - endCol)
				} else if i > end.LineIdx {
					// Lines after the selection - keep them
					newLines = append(newLines, line)
				}
			}
			block.Lines = newLines
			m.Cursor.LineIdx = start.LineIdx
		}
		block.IsDirty = true
	}

	m.ClearSelection()
	m.ensureCursorInView()
	return deleted, true
}
