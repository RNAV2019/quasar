package editor

import "strings"

// isValidMathBlock checks if a block is a valid math block (starts and ends with $$)
func isValidMathBlock(block Block) bool {
	if block.Type != MathBlock {
		return false
	}
	if len(block.Lines) < 2 {
		return false
	}
	return block.Lines[0] == "$$" && block.Lines[len(block.Lines)-1] == "$$"
}

// canMergeBlocks checks if two blocks can be safely merged without corrupting math blocks
func canMergeBlocks(prevBlock, currentBlock Block) bool {
	// Never merge with or into a math block - math blocks must maintain their $$ boundaries
	if prevBlock.Type == MathBlock || currentBlock.Type == MathBlock {
		return false
	}
	return true
}

// ValidateBlocks checks and repairs block integrity after editing operations.
// It returns the repaired block slice and true if any repairs were made.
func ValidateBlocks(blocks []Block) ([]Block, bool) {
	needsRepair := false
	
	// Check each math block has proper $$ delimiters
	for i := range blocks {
		if blocks[i].Type == MathBlock {
			if len(blocks[i].Lines) < 2 {
				needsRepair = true
				break
			}
			if blocks[i].Lines[0] != "$$" || blocks[i].Lines[len(blocks[i].Lines)-1] != "$$" {
				needsRepair = true
				break
			}
		}
	}
	
	if !needsRepair {
		return blocks, false
	}
	
	// Re-parse all blocks from lines to fix integrity
	var allLines []string
	for _, block := range blocks {
		allLines = append(allLines, block.Lines...)
	}
	
	return CreateModelFromLines(allLines).Blocks, true
}

// InsertChar inserts a character at the cursor position.
func (m *Model) InsertChar(r rune) {
	block := &m.Blocks[m.Cursor.BlockIdx]
	block.HasError = false

	line := &block.Lines[m.Cursor.LineIdx]

	maxLen := m.MaxLineLength()
	if len([]rune(*line)) >= maxLen {
		return
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

// Backspace deletes the character before the cursor.
func (m *Model) Backspace() {
	block := &m.Blocks[m.Cursor.BlockIdx]
	block.HasError = false

	isAtStartOfLine := m.Cursor.Col == 0
	isLastLine := m.Cursor.LineIdx == len(block.Lines)-1

	if block.Type == MathBlock && isAtStartOfLine && isLastLine && block.Lines[m.Cursor.LineIdx] == "" && len(block.Lines) > 2 {
		currentBlockIdx := m.Cursor.BlockIdx
		if currentBlockIdx > 0 {
			prevBlockIndex := currentBlockIdx - 1
			prevBlock := &m.Blocks[prevBlockIndex]

			m.Cursor.BlockIdx = prevBlockIndex
			m.Cursor.LineIdx = max(len(prevBlock.Lines)-1, 0)
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
		// At start of block (line 0, col 0)
		prevBlockIndex := m.Cursor.BlockIdx - 1
		prevBlock := &m.Blocks[prevBlockIndex]
		currentBlock := m.Blocks[m.Cursor.BlockIdx]

		// If current line is empty and there are more lines, just remove the empty line
		// instead of trying to merge blocks
		if block.Type == TextBlock && len(currentBlock.Lines) > 1 && block.Lines[0] == "" {
			block.Lines = block.Lines[1:]
			block.IsDirty = true
			// Cursor stays at line 0, col 0
			return
		}

		// Check if we can safely merge these blocks
		// Math blocks should never be merged - they need proper $$ boundaries
		if !canMergeBlocks(*prevBlock, currentBlock) {
			// Can't merge - just move cursor to end of previous block
			m.Cursor.BlockIdx = prevBlockIndex
			m.Cursor.LineIdx = max(len(prevBlock.Lines)-1, 0)
			m.Cursor.Col = 0
			if len(prevBlock.Lines) > 0 && m.Cursor.LineIdx < len(prevBlock.Lines) {
				m.Cursor.Col = len([]rune(prevBlock.Lines[m.Cursor.LineIdx]))
			}
			return
		}

		newCursorBlockIdx := prevBlockIndex
		newCursorLineIdx := max(len(prevBlock.Lines)-1, 0)
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

// DeleteChar deletes the character under the cursor.
// On empty lines, it deletes the entire line and moves cursor appropriately.
func (m *Model) DeleteChar() {
	block := &m.Blocks[m.Cursor.BlockIdx]
	block.HasError = false
	line := &block.Lines[m.Cursor.LineIdx]
	runes := []rune(*line)

	// If the line is empty, delete the entire line
	if len(runes) == 0 {
		// If this is the only line in the block, don't delete
		if len(block.Lines) == 1 {
			return
		}

		// Determine where to move cursor after deletion
		isLastLine := m.Cursor.LineIdx == len(block.Lines)-1
		
		// Remove the line
		block.Lines = append(block.Lines[:m.Cursor.LineIdx], block.Lines[m.Cursor.LineIdx+1:]...)
		block.IsDirty = true

		// Move cursor appropriately
		if isLastLine {
			// Move to previous line
			m.Cursor.LineIdx--
			m.Cursor.Col = len([]rune(block.Lines[m.Cursor.LineIdx]))
		} else {
			// Stay at current line index (now pointing to next line), col 0
			m.Cursor.Col = 0
		}

		m.ensureCursorInView()
		return
	}

	// Normal case: delete character under cursor
	if m.Cursor.Col < len(runes) {
		runes = append(runes[:m.Cursor.Col], runes[m.Cursor.Col+1:]...)
		*line = string(runes)
		block.IsDirty = true
	}
}

// InsertNewLine inserts a newline at the cursor position.
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

// PasteLine inserts a line below the current line with the given content.
func (m *Model) PasteLine(content string) {
	if m.Cursor.BlockIdx >= len(m.Blocks) {
		return
	}
	block := &m.Blocks[m.Cursor.BlockIdx]
	lineIdx := m.Cursor.LineIdx + 1

	lineIdx = min(lineIdx, len(block.Lines))

	block.Lines = append(block.Lines[:lineIdx], append([]string{content}, block.Lines[lineIdx:]...)...)
	block.IsDirty = true
	m.Cursor.LineIdx = lineIdx
	m.Cursor.Col = 0
	m.ensureCursorInView()
}

// PasteText inserts text at the current cursor position.
func (m *Model) PasteText(text string) {
	if text == "" {
		return
	}

	if m.Cursor.BlockIdx >= len(m.Blocks) {
		return
	}
	block := &m.Blocks[m.Cursor.BlockIdx]
	if m.Cursor.LineIdx >= len(block.Lines) {
		return
	}

	lines := strings.Split(text, "\n")
	currentLine := block.Lines[m.Cursor.LineIdx]
	currentRunes := []rune(currentLine)
	prefix := string(currentRunes[:m.Cursor.Col])
	suffix := string(currentRunes[m.Cursor.Col:])

	if len(lines) == 1 {
		block.Lines[m.Cursor.LineIdx] = prefix + lines[0] + suffix
		m.Cursor.Col += len([]rune(lines[0]))
	} else {
		block.Lines[m.Cursor.LineIdx] = prefix + lines[0]
		newLines := make([]string, len(lines)-1)
		for i := 1; i < len(lines); i++ {
			newLines[i-1] = lines[i]
		}
		newLines[len(newLines)-1] += suffix
		block.Lines = append(block.Lines[:m.Cursor.LineIdx+1], append(newLines, block.Lines[m.Cursor.LineIdx+1:]...)...)
		m.Cursor.LineIdx += len(lines) - 1
		m.Cursor.Col = len([]rune(newLines[len(newLines)-1])) - len([]rune(suffix))
	}

	block.IsDirty = true
	m.ensureCursorInView()
}
