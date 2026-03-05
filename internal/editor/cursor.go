package editor

// IsWordChar reports whether r is a word character.
// Word characters are letters, digits, and underscores.
func IsWordChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'
}

// MoveWordForward moves cursor to the start of the next word.
func (m *Model) MoveWordForward() {
	if m.Cursor.BlockIdx >= len(m.Blocks) {
		return
	}

	block := m.Blocks[m.Cursor.BlockIdx]
	if m.Cursor.LineIdx >= len(block.Lines) {
		return
	}

	line := block.Lines[m.Cursor.LineIdx]
	runes := []rune(line)
	col := m.Cursor.Col

	// Skip current word
	for col < len(runes) && IsWordChar(runes[col]) {
		col++
	}

	// Skip non-word, non-whitespace characters (punctuation, etc.)
	for col < len(runes) && !IsWordChar(runes[col]) && runes[col] != ' ' && runes[col] != '\t' {
		col++
	}

	// Skip whitespace
	for col < len(runes) && (runes[col] == ' ' || runes[col] == '\t') {
		col++
	}

	if col >= len(runes) {
		// Move to next line
		if m.Cursor.LineIdx+1 < len(block.Lines) {
			m.Cursor.LineIdx++
			m.Cursor.Col = 0
			// Skip leading whitespace on next line
			nextLine := []rune(block.Lines[m.Cursor.LineIdx])
			for m.Cursor.Col < len(nextLine) && (nextLine[m.Cursor.Col] == ' ' || nextLine[m.Cursor.Col] == '\t') {
				m.Cursor.Col++
			}
		}
	} else {
		m.Cursor.Col = col
	}

	m.ensureCursorInView()
}

// MoveWordBackward moves cursor to the start of the previous word.
func (m *Model) MoveWordBackward() {
	if m.Cursor.BlockIdx >= len(m.Blocks) {
		return
	}

	block := m.Blocks[m.Cursor.BlockIdx]
	if m.Cursor.LineIdx >= len(block.Lines) {
		return
	}

	line := block.Lines[m.Cursor.LineIdx]
	runes := []rune(line)
	col := m.Cursor.Col

	// Skip current position if we're in the middle of a word
	if col > 0 && col <= len(runes) && IsWordChar(runes[col-1]) {
		// We're at end of word, need to skip back past this word first
		for col > 0 && IsWordChar(runes[col-1]) {
			col--
		}
	}

	// Skip whitespace
	for col > 0 && (runes[col-1] == ' ' || runes[col-1] == '\t') {
		col--
	}

	// Now skip back to start of word
	for col > 0 && IsWordChar(runes[col-1]) {
		col--
	}

	if col == 0 && m.Cursor.LineIdx > 0 {
		// Move to previous line
		m.Cursor.LineIdx--
		prevLine := []rune(block.Lines[m.Cursor.LineIdx])
		m.Cursor.Col = len(prevLine)
		// Skip trailing whitespace
		for m.Cursor.Col > 0 && (prevLine[m.Cursor.Col-1] == ' ' || prevLine[m.Cursor.Col-1] == '\t') {
			m.Cursor.Col--
		}
		// Skip back to start of word on previous line
		for m.Cursor.Col > 0 && IsWordChar(prevLine[m.Cursor.Col-1]) {
			m.Cursor.Col--
		}
	} else {
		m.Cursor.Col = col
	}

	m.ensureCursorInView()
}

// MoveToEndOfWord moves cursor to the end of the current/next word.
func (m *Model) MoveToEndOfWord() {
	if m.Cursor.BlockIdx >= len(m.Blocks) {
		return
	}

	block := m.Blocks[m.Cursor.BlockIdx]
	if m.Cursor.LineIdx >= len(block.Lines) {
		return
	}

	line := block.Lines[m.Cursor.LineIdx]
	runes := []rune(line)
	col := m.Cursor.Col

	// Skip whitespace first
	for col < len(runes) && (runes[col] == ' ' || runes[col] == '\t') {
		col++
	}

	// Move to end of word
	for col < len(runes) && IsWordChar(runes[col]) {
		col++
	}

	if col >= len(runes) {
		// Move to next line
		if m.Cursor.LineIdx+1 < len(block.Lines) {
			m.Cursor.LineIdx++
			m.MoveToEndOfWord()
			return
		}
	} else {
		m.Cursor.Col = col
	}

	m.ensureCursorInView()
}

// MoveToStartOfLine moves cursor to the beginning of the line.
func (m *Model) MoveToStartOfLine() {
	m.Cursor.Col = 0
	m.ensureCursorInView()
}

// MoveToEndOfLine moves cursor to the end of the line.
func (m *Model) MoveToEndOfLine() {
	if m.Cursor.BlockIdx < len(m.Blocks) {
		block := m.Blocks[m.Cursor.BlockIdx]
		if m.Cursor.LineIdx < len(block.Lines) {
			m.Cursor.Col = len([]rune(block.Lines[m.Cursor.LineIdx]))
		}
	}
	m.ensureCursorInView()
}

// GoToFirstLine moves cursor to the first line of the document.
func (m *Model) GoToFirstLine() {
	if len(m.Blocks) > 0 {
		m.Cursor.BlockIdx = 0
		m.Cursor.LineIdx = 0
		m.Cursor.Col = 0
	}
	m.ensureCursorInView()
}

// GoToLastLine moves cursor to the last line of the document.
func (m *Model) GoToLastLine() {
	if len(m.Blocks) > 0 {
		lastBlockIdx := len(m.Blocks) - 1
		m.Cursor.BlockIdx = lastBlockIdx
		lines := m.Blocks[lastBlockIdx].Lines
		m.Cursor.LineIdx = max(len(lines)-1, 0)
		if len(lines) > 0 {
			m.Cursor.Col = len([]rune(lines[m.Cursor.LineIdx]))
		}
	}
	m.ensureCursorInView()
}
