package editor

const maxUndoHistory = 100

// Snapshot captures the editor state for undo/redo.
type Snapshot struct {
	Blocks []Block
	Cursor Position
}

// UndoManager tracks undo/redo history.
type UndoManager struct {
	undoStack []Snapshot
	redoStack []Snapshot
}

// NewUndoManager creates a new UndoManager.
func NewUndoManager() *UndoManager {
	return &UndoManager{}
}

func cloneBlocks(blocks []Block) []Block {
	clone := make([]Block, len(blocks))
	for i, b := range blocks {
		clone[i] = Block{
			Type: b.Type,
		}
		clone[i].Lines = make([]string, len(b.Lines))
		copy(clone[i].Lines, b.Lines)
	}
	return clone
}

// Save pushes the current state onto the undo stack and clears the redo stack.
func (u *UndoManager) Save(m *Model) {
	u.undoStack = append(u.undoStack, Snapshot{
		Blocks: cloneBlocks(m.Blocks),
		Cursor: m.Cursor,
	})
	if len(u.undoStack) > maxUndoHistory {
		u.undoStack = u.undoStack[len(u.undoStack)-maxUndoHistory:]
	}
	u.redoStack = nil
}

// Undo restores the previous state. Returns true if state was restored.
func (u *UndoManager) Undo(m *Model) bool {
	if len(u.undoStack) == 0 {
		return false
	}
	// Push current state to redo stack
	u.redoStack = append(u.redoStack, Snapshot{
		Blocks: cloneBlocks(m.Blocks),
		Cursor: m.Cursor,
	})
	// Pop from undo stack
	snap := u.undoStack[len(u.undoStack)-1]
	u.undoStack = u.undoStack[:len(u.undoStack)-1]
	m.Blocks = snap.Blocks
	m.Cursor = snap.Cursor
	// Mark all blocks dirty so images recompile
	for i := range m.Blocks {
		m.Blocks[i].IsDirty = true
	}
	return true
}

// Redo restores the next state. Returns true if state was restored.
func (u *UndoManager) Redo(m *Model) bool {
	if len(u.redoStack) == 0 {
		return false
	}
	// Push current state to undo stack
	u.undoStack = append(u.undoStack, Snapshot{
		Blocks: cloneBlocks(m.Blocks),
		Cursor: m.Cursor,
	})
	// Pop from redo stack
	snap := u.redoStack[len(u.redoStack)-1]
	u.redoStack = u.redoStack[:len(u.redoStack)-1]
	m.Blocks = snap.Blocks
	m.Cursor = snap.Cursor
	for i := range m.Blocks {
		m.Blocks[i].IsDirty = true
	}
	return true
}
