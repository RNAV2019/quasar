package editor

import "strconv"

type BlockType int

const (
	TextBlock BlockType = iota
	MathBlock
)

type Block struct {
	Type         BlockType
	Lines        []string
	ImageID      uint32
	ImageCols    int
	ImageHeight  int
	IsDirty      bool
	HasError     bool
	ErrorMessage string
}

// Represents a coordinate in the text editor by way of row and column
type Position struct {
	BlockIdx int // Which block the cursor is in
	LineIdx  int // Which line *within* the block
	Col      int // Which character *within* the line
}

// Handles state of text editor
// Holds content buffer, cursor position, and the visible viewport
type Model struct {
	Blocks []Block
	Cursor Position
	Offset Position // Represents the viewports scroll position
	Width  int
	Height int
}

// Initialize the editor with default values and default front matter
func NewModel() Model {
	return Model{
		Blocks: []Block{
			{
				Type: TextBlock,
				Lines: []string{
					"---",
					"title: ",
					"tags: []",
					"---",
					"",
					"Welcome to quasar notes",
					"Type your notes and maths here",
				},
				IsDirty: true,
			},
		},
		Cursor: Position{BlockIdx: 0, LineIdx: 0, Col: 0},
		Offset: Position{BlockIdx: 0, LineIdx: 0, Col: 0},
	}
}

// Update the viewports dimensions
func (m *Model) SetSize(width, height int) {
	m.Width = width
	m.Height = height
	m.ensureCursorInView() // Re-clamp if windows shrinks
}

func (m *Model) MoveCursor(lineDelta, colDelta int) {
	m.Cursor.Col += colDelta
	m.Cursor.LineIdx += lineDelta

	// Clamp vertically
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

	// Clamp horizontally
	// Ensure LineIdx is valid before accessing Lines
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

func (m *Model) EndOfLine() {
	lineLen := len(m.Blocks[m.Cursor.BlockIdx].Lines[m.Cursor.LineIdx])
	m.Cursor.Col = lineLen
	m.ensureCursorInView()
}

func (m *Model) ensureCursorInView() {
	// Don't scroll if height not properly set yet
	if m.Height <= 0 {
		return
	}

	// Calculate total lines
	totalLines := 0
	for _, block := range m.Blocks {
		totalLines += len(block.Lines)
	}

	// Reset horizontal offset (no horizontal scrolling)
	m.Offset.Col = 0

	// Vertical scrolling: only scroll when cursor reaches bottom of viewport
	viewHeight := m.Height

	// If content fits entirely in viewport, reset offset to top
	if totalLines <= viewHeight {
		m.Offset.BlockIdx = 0
		m.Offset.LineIdx = 0
		return
	}

	// Calculate absolute line number for cursor
	cursorAbsLine := 0
	for i := 0; i < m.Cursor.BlockIdx && i < len(m.Blocks); i++ {
		cursorAbsLine += len(m.Blocks[i].Lines)
	}
	cursorAbsLine += m.Cursor.LineIdx

	// Calculate absolute line number for offset
	offsetAbsLine := 0
	for i := 0; i < m.Offset.BlockIdx && i < len(m.Blocks); i++ {
		offsetAbsLine += len(m.Blocks[i].Lines)
	}
	offsetAbsLine += m.Offset.LineIdx

	// Scroll up if cursor is above viewport
	if cursorAbsLine < offsetAbsLine {
		m.Offset.BlockIdx = m.Cursor.BlockIdx
		m.Offset.LineIdx = m.Cursor.LineIdx
	}
	// Scroll down only when cursor reaches/passes the bottom of viewport
	if cursorAbsLine >= offsetAbsLine+viewHeight {
		// Find new offset to place cursor at bottom
		targetAbsLine := cursorAbsLine - viewHeight + 1
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

func (m *Model) ViewLines() []string {
	var visibleLines []string
	// This is a simplified version. A real implementation would consider the offset.
	for _, block := range m.Blocks {
		visibleLines = append(visibleLines, block.Lines...)
	}
	return visibleLines
}

func (m *Model) MaxLineLength() int {
	totalLines := 0
	for _, block := range m.Blocks {
		totalLines += len(block.Lines)
	}
	gutterWidth := len(strconv.Itoa(totalLines))
	// Layout: [padding 2][indicator 1][line numbers gutterWidth+2][content]
	// Subtract extra 2 for some right margin
	maxLen := m.Width - 5 - gutterWidth - 2
	if maxLen < 20 {
		maxLen = 20
	}
	return maxLen
}

func (m *Model) InsertChar(r rune) {
	block := &m.Blocks[m.Cursor.BlockIdx]
	block.HasError = false // Reset error state on edit

	if block.Type == MathBlock {
		isDelimiterLine := (m.Cursor.LineIdx == 0 || m.Cursor.LineIdx == len(block.Lines)-1) && block.Lines[m.Cursor.LineIdx] == "$$"
		if isDelimiterLine {
			return
		}
	}

	line := &block.Lines[m.Cursor.LineIdx]

	// Enforce max line length (except for math blocks and $ handling)
	maxLen := m.MaxLineLength()
	if block.Type == TextBlock && r != '$' && len([]rune(*line)) >= maxLen {
		return // Don't insert if at max length
	}

	if r == '$' && block.Type == TextBlock {
		runes := []rune(*line)
		// Check if part of $$ block
		if m.Cursor.Col > 0 && runes[m.Cursor.Col-1] == '$' {
			// User typed '$$'. Let's convert the first '$' to a multiline math block.
			// The `splitBlockForMath` function expects the cursor to be after '$$'.
			if m.Cursor.Col < len(runes) && runes[m.Cursor.Col] == '$' {
				// If the next character is the auto-inserted '$', swallow it.
				*line = string(runes[:m.Cursor.Col-1]) + "$$" + string(runes[m.Cursor.Col+1:])
			} else {
				*line = string(runes[:m.Cursor.Col-1]) + "$$" + string(runes[m.Cursor.Col:])
			}
			m.Cursor.Col++ // move cursor to be after '$$'
			m.splitBlockForMath()
			return
		} else {
			// This is the first '$'. Insert '$$' and place cursor in the middle.
			if m.Cursor.Col >= len(runes) {
				*line += "$$"
			} else {
				*line = string(runes[:m.Cursor.Col]) + "$$" + string(runes[m.Cursor.Col:])
			}
			m.Cursor.Col++
			block.IsDirty = true
			return
		}
	}

	runes := []rune(*line)

	if m.Cursor.Col >= len(runes) {
		runes = append(runes, r)
	} else {
		runes = append(runes[:m.Cursor.Col], append([]rune{r}, runes[m.Cursor.Col:]...)...)
	}
	*line = string(runes)
	m.Cursor.Col++

	block.IsDirty = true
}

func (m *Model) splitBlockForMath() {
	currentBlockIdx := m.Cursor.BlockIdx
	currentLineIdx := m.Cursor.LineIdx
	currentCol := m.Cursor.Col

	originalBlock := m.Blocks[currentBlockIdx]
	lineWithDollars := originalBlock.Lines[currentLineIdx]
	runes := []rune(lineWithDollars)

	// Text on the line before the "$$"
	beforeText := string(runes[:currentCol-2])
	// Text on the line after the "$$"
	afterText := string(runes[currentCol:])

	// --- Block Assembly ---

	// Part 1: The TextBlock before the math.
	block1Lines := originalBlock.Lines[:currentLineIdx]
	if beforeText != "" {
		block1Lines = append(block1Lines, beforeText)
	}

	// Part 2: The new MathBlock.
	block2 := Block{Type: MathBlock, Lines: []string{"$$", "", "$$"}, IsDirty: true}

	// Part 3: The TextBlock after the math.
	block3Lines := []string{}
	if afterText != "" {
		block3Lines = append(block3Lines, afterText)
	}
	if currentLineIdx+1 < len(originalBlock.Lines) {
		block3Lines = append(block3Lines, originalBlock.Lines[currentLineIdx+1:]...)
	}

	// --- Final Slice Assembly ---
	finalBlocks := []Block{}
	finalBlocks = append(finalBlocks, m.Blocks[:currentBlockIdx]...) // Blocks before the one we split.

	cursorBlockOffset := 0
	if len(block1Lines) > 0 {
		finalBlocks = append(finalBlocks, Block{Type: TextBlock, Lines: block1Lines, IsDirty: true})
	} else {
		cursorBlockOffset = -1 // The original block was removed
	}

	finalBlocks = append(finalBlocks, block2) // Add the math block

	if len(block3Lines) > 0 {
		finalBlocks = append(finalBlocks, Block{Type: TextBlock, Lines: block3Lines, IsDirty: true})
	}

	if currentBlockIdx+1 < len(m.Blocks) {
		finalBlocks = append(finalBlocks, m.Blocks[currentBlockIdx+1:]...) // Blocks after the one we split.
	}

	m.Blocks = finalBlocks

	// Update cursor position to be in the new math block
	m.Cursor.BlockIdx = currentBlockIdx + cursorBlockOffset + 1
	m.Cursor.LineIdx = 1 // Place cursor on the empty line between the '$$'
	m.Cursor.Col = 0
}

func (m *Model) Backspace() {
	block := &m.Blocks[m.Cursor.BlockIdx]
	block.HasError = false

	// --- Special Case: Delete MathBlock on backspace on empty last line ---
	isAtStartOfLine := m.Cursor.Col == 0
	isLastLine := m.Cursor.LineIdx == len(block.Lines)-1

	if block.Type == MathBlock && isAtStartOfLine && isLastLine && block.Lines[m.Cursor.LineIdx] == "" && len(block.Lines) > 2 {
		currentBlockIdx := m.Cursor.BlockIdx
		if currentBlockIdx > 0 {
			prevBlockIndex := currentBlockIdx - 1
			prevBlock := &m.Blocks[prevBlockIndex]

			m.Cursor.BlockIdx = prevBlockIndex
			m.Cursor.LineIdx = len(prevBlock.Lines) - 1
			if m.Cursor.LineIdx < 0 {
				m.Cursor.LineIdx = 0
			}
			m.Cursor.Col = 0
			if len(prevBlock.Lines) > 0 {
				m.Cursor.Col = len(prevBlock.Lines[m.Cursor.LineIdx])
			}

			m.Blocks = append(m.Blocks[:currentBlockIdx], m.Blocks[currentBlockIdx+1:]...)
		} else {
			*block = Block{Type: TextBlock, Lines: []string{""}}
			m.Cursor.BlockIdx = 0
			m.Cursor.LineIdx = 0
			m.Cursor.Col = 0
		}
		return
	}

	// --- Existing Backspace Logic ---
	if m.Cursor.Col > 0 {
		line := &block.Lines[m.Cursor.LineIdx]
		runes := []rune(*line)
		runes = append(runes[:m.Cursor.Col-1], runes[m.Cursor.Col:]...)
		*line = string(runes)
		block.IsDirty = true
		m.Cursor.Col--
	} else if m.Cursor.LineIdx > 0 {
		currentLine := block.Lines[m.Cursor.LineIdx]
		prevLineIdx := m.Cursor.LineIdx - 1
		prevLine := &block.Lines[prevLineIdx]
		newCol := len(*prevLine)
		*prevLine += currentLine
		block.Lines = append(block.Lines[:m.Cursor.LineIdx], block.Lines[m.Cursor.LineIdx+1:]...)
		block.IsDirty = true
		m.Cursor.LineIdx--
		m.Cursor.Col = newCol
	} else if m.Cursor.BlockIdx > 0 {
		currentBlock := m.Blocks[m.Cursor.BlockIdx]
		prevBlockIndex := m.Cursor.BlockIdx - 1
		prevBlock := &m.Blocks[prevBlockIndex]

		newCursorBlockIdx := prevBlockIndex
		newCursorLineIdx := len(prevBlock.Lines) - 1
		if newCursorLineIdx < 0 {
			newCursorLineIdx = 0
		}
		newCursorCol := 0
		if len(prevBlock.Lines) > 0 {
			newCursorCol = len(prevBlock.Lines[newCursorLineIdx])
		}

		if len(prevBlock.Lines) > 0 && len(currentBlock.Lines) > 0 {
			prevBlock.Lines[len(prevBlock.Lines)-1] += currentBlock.Lines[0]
			prevBlock.Lines = append(prevBlock.Lines, currentBlock.Lines[1:]...)
		} else {
			prevBlock.Lines = append(prevBlock.Lines, currentBlock.Lines...)
		}
		prevBlock.IsDirty = true
		m.Blocks = append(m.Blocks[:m.Cursor.BlockIdx], m.Blocks[m.Cursor.BlockIdx+1:]...)

		m.Cursor.BlockIdx = newCursorBlockIdx
		m.Cursor.LineIdx = newCursorLineIdx
		m.Cursor.Col = newCursorCol
	}

	m.ensureCursorInView()
}

// Deletes the character under the cursor
func (m *Model) DeleteChar() {
	block := &m.Blocks[m.Cursor.BlockIdx]
	block.HasError = false
	line := &block.Lines[m.Cursor.LineIdx]
	runes := []rune(*line)

	if m.Cursor.Col < len(runes) {
		runes = append(runes[:m.Cursor.Col], runes[m.Cursor.Col+1:]...)
		*line = string(runes)
		block.IsDirty = true
	}
}

func (m *Model) InsertNewLine() {
	block := &m.Blocks[m.Cursor.BlockIdx]

	if block.Type == MathBlock {
		lineContent := block.Lines[m.Cursor.LineIdx]
		isFirstLine := m.Cursor.LineIdx == 0
		isLastLine := m.Cursor.LineIdx == len(block.Lines)-1

		if isLastLine && lineContent == "$$" {
			newTextBlock := Block{Type: TextBlock, Lines: []string{""}}

			insertIndex := m.Cursor.BlockIdx + 1
			if insertIndex < len(m.Blocks) {
				m.Blocks = append(m.Blocks[:insertIndex], append([]Block{newTextBlock}, m.Blocks[insertIndex:]...)...)
			} else {
				m.Blocks = append(m.Blocks, newTextBlock)
			}

			m.Cursor.BlockIdx = insertIndex
			m.Cursor.LineIdx = 0
			m.Cursor.Col = 0
			return
		}

		if isFirstLine && lineContent == "$$" {
			block.Lines = append(block.Lines[:1], append([]string{""}, block.Lines[1:]...)...)
			block.IsDirty = true
			block.HasError = false
			m.Cursor.LineIdx = 1
			m.Cursor.Col = 0
			return
		}
	}

	line := &block.Lines[m.Cursor.LineIdx]
	runes := []rune(*line)

	left := string(runes[:m.Cursor.Col])
	right := string(runes[m.Cursor.Col:])

	*line = left
	block.IsDirty = true
	block.HasError = false

	nextLineIdx := m.Cursor.LineIdx + 1
	block.Lines = append(block.Lines[:nextLineIdx], append([]string{right}, block.Lines[nextLineIdx:]...)...)

	m.Cursor.LineIdx++
	m.Cursor.Col = 0
	m.ensureCursorInView()
}

