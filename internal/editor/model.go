// Package editor implements a block-based text editor with support for
// markdown content, math blocks, cursor movement, and selections.
package editor

// BlockType represents the type of content in a block.
type BlockType int

const (
	// TextBlock indicates a block of regular markdown text.
	TextBlock BlockType = iota
	// MathBlock indicates a block of LaTeX math delimited by $$.
	MathBlock
)

// Block represents a section of content in the editor.
type Block struct {
	Type         BlockType
	Lines        []string
	ImageID      uint32
	ImageCols    int
	ImageHeight  int
	IsDirty      bool
	IsLoading    bool
	HasError     bool
	ErrorMessage string
}

// Position represents a cursor position in the document.
type Position struct {
	BlockIdx int // Which block the cursor is in
	LineIdx  int // Which line within the block
	Col      int // Which character within the line
}

// Selection represents a text selection in the editor.
type Selection struct {
	Active      bool     // Whether selection is active
	Start       Position // Start of selection
	End         Position // End of selection
	WasLineWise bool     // Whether this was a line-wise selection (used for paste)
}

// Model holds the state of the text editor.
// It contains the content buffer, cursor position, and viewport state.
type Model struct {
	Blocks    []Block
	Cursor    Position
	Offset    Position  // Viewport scroll position
	Width     int
	Height    int
	Selection Selection // Current selection
}

// NewModel initializes the editor with default values and front matter.
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

// SetSize updates the viewport dimensions.
func (m *Model) SetSize(width, height int) {
	m.Width = width
	m.Height = height
	m.ensureCursorInView()
}
