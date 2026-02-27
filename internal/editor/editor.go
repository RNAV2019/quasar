package editor

// Represents a single line in the text editor
type Line struct {
	Raw         string // Raw string data
	Rendered    string // Either syntax highlighted or kitty graphics
	IsMath      bool   // Indicate whether the line is math or not
	IsDirty     bool   // Whether to async re-render line
	ImageHeight int    // Height of image in terminal cells
}

// Represents a coordinate in the text editor by way or row and column
type Position struct {
	Row int
	Col int
}

// Handles state of text editor
// Holds content buffer, cursor position, and the visible viewport
type Model struct {
	Lines  []Line
	Cursor Position
	Offset Position // Represents the viewports scroll position -> First visible character in viewport
	Width  int
	Height int
}

// Initialize the editor with default values
func NewModel() Model {
	return Model{
		Lines: []Line{
			{Raw: "Welcome to quasar notes", IsDirty: true},
			{Raw: "Type some math notes here...", IsDirty: true},
		},
		Cursor: Position{Row: 0, Col: 0},
		Offset: Position{Row: 0, Col: 0},
	}
}

// Update the viewports dimensions
func (m *Model) SetSize(width, height int) {
	m.Width = width
	m.Height = height
	m.ensureCursorInView() // Re-clamp if windows shrinks
}

// Move cursor safely within the bounds and updates the viewport
func (m *Model) MoveCursor(rowDelta, colDelta int) {
	// Calculate new row within bounds
	newRow := max(m.Cursor.Row+rowDelta, 0)
	if newRow >= len(m.Lines) {
		newRow = len(m.Lines) - 1
	}
	m.Cursor.Row = newRow

	// Calculate new col within bounds
	lineLen := len(m.Lines[m.Cursor.Row].Raw)
	newCol := min(max(m.Cursor.Col+colDelta, 0), lineLen)

	// Snap cursor if next line is shorter than current line
	m.Cursor.Col = min(newCol, len(m.Lines[m.Cursor.Row].Raw))

	m.ensureCursorInView()
}

func (m *Model) EndOfLine() {
	lineLen := len(m.Lines[m.Cursor.Row].Raw)
	m.Cursor.Col = lineLen
	m.ensureCursorInView()
}

// Adjust offset to keep cursor in view
func (m *Model) ensureCursorInView() {
	if m.Cursor.Row < m.Offset.Row {
		// Cursor is above viewport
		m.Offset.Row = m.Cursor.Row
	} else if m.Cursor.Row >= m.Offset.Row+m.Height {
		// Cursor is below viewport
		m.Offset.Row = m.Cursor.Row - m.Height + 1
	}

	if m.Cursor.Col < m.Offset.Col {
		// Cursor is left of viewport -> Scroll left
		m.Offset.Col = m.Cursor.Col
	} else if m.Cursor.Col >= m.Offset.Col+m.Width {
		// Cursor is right of viewport -> Scroll right
		m.Offset.Col = m.Cursor.Col - m.Width + 1
	}
}

func (m Model) ViewLines() []string {
	var visibleLines []string

	endRow := min(m.Offset.Row+m.Height, len(m.Lines))

	for i := m.Offset.Row; i < endRow; i++ {
		line := m.Lines[i]
		txt := line.Raw

		if len(txt) > m.Offset.Col {
			txt = txt[m.Offset.Col:]
		} else {
			txt = ""
		}

		if len(txt) > m.Width {
			txt = txt[:m.Width]
		}
		visibleLines = append(visibleLines, txt)
	}
	return visibleLines
}

func (m *Model) InsertChar(r rune) {
	line := &m.Lines[m.Cursor.Row]
	runes := []rune(line.Raw)

	if m.Cursor.Col >= len(line.Raw) {
		runes = append(runes, r)
	} else {
		runes = append(runes[:m.Cursor.Col], append([]rune{r}, runes[m.Cursor.Col:]...)...)
	}

	line.Raw = string(runes)
	line.IsDirty = true
	m.Cursor.Col++
}
func (m *Model) Backspace() {
	if m.Cursor.Col > 0 {
		line := &m.Lines[m.Cursor.Row]
		runes := []rune(line.Raw)

		runes = append(runes[:m.Cursor.Col-1], runes[m.Cursor.Col:]...)

		line.Raw = string(runes)
		line.IsDirty = true
		m.Cursor.Col--
	} else if m.Cursor.Row > 0 {
		currentLine := m.Lines[m.Cursor.Row]
		prevLineIdx := m.Cursor.Row - 1
		prevLine := &m.Lines[prevLineIdx]

		newCol := len([]rune(prevLine.Raw))

		prevLine.Raw += currentLine.Raw
		prevLine.IsDirty = true

		m.Lines = append(m.Lines[:m.Cursor.Row], m.Lines[m.Cursor.Row+1:]...)

		m.Cursor.Row = prevLineIdx
		m.Cursor.Col = newCol
	}
}

func (m *Model) InsertNewLine() {
	line := &m.Lines[m.Cursor.Row]
	runes := []rune(line.Raw)

	left := string(runes[:m.Cursor.Col])
	right := string(runes[m.Cursor.Col:])

	line.Raw = left
	line.IsDirty = true

	newLine := Line{Raw: right, IsDirty: true}

	newLines := make([]Line, 0)
	newLines = append(newLines, m.Lines[:m.Cursor.Row+1]...)
	newLines = append(newLines, newLine)
	newLines = append(newLines, m.Lines[m.Cursor.Row+1:]...)
	m.Lines = newLines

	m.Cursor.Row++
	m.Cursor.Col = 0
	m.ensureCursorInView()
}
