package ui

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/RNAV2019/quasar/internal/config"
	"github.com/RNAV2019/quasar/internal/editor"
	"github.com/RNAV2019/quasar/internal/latex"
	"github.com/RNAV2019/quasar/internal/notebook"
	"github.com/atotto/clipboard"
)

type Mode int

const (
	Normal Mode = iota
	Insert
	Select
	Command
	NewNote
	Help
	DeleteConfirm
)

type Model struct {
	mode               Mode
	width              int
	height             int
	Time               time.Time
	Editor             editor.Model
	Config             *config.Config
	InlineRenders      map[string]InlineMathRender
	PendingRenders     int
	CompiledMath       []string
	CmdInput           textinput.Model
	StatusMessage      string
	ParsedDoc          *editor.Document // Parsed document structure
	FileTree           *FileTree
	ShowFileTree       bool
	pendingSpace       bool // Track if space was pressed (for space+key combos)
	NewNoteDialog      NewNoteDialog
	HelpDialog         HelpDialog
	DeleteConfirmDialog ConfirmDialog
	NotebookName       string
	NotebookPath       string
	CurrentFile        string // Path to currently open file, empty if no file open
	Autocomplete       AutocompleteBox
	slashStartCol      int // Column where slash started for autocomplete
}

type TickMsg time.Time

type BlockProcessedMsg struct {
	BlockIdx    int
	ImageID     uint32
	ImageCols   int
	ImageHeight int
	Error       error
	Source      string
}

type InlineMathProcessedMsg struct {
	BlockIdx    int
	LineIdx     int
	StartCol    int
	EndCol      int
	ImageID     uint32
	ImageCols   int
	ImageHeight int
	Error       error
}

type InlineMathRender struct {
	ImageID     uint32
	ImageCols   int
	ImageHeight int
	Length      int
}

var inlineMathRe = regexp.MustCompile(`\$[^\$]*\$`)

func doTick() tea.Cmd {
	return tea.Tick(time.Millisecond*50, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m *Model) processDirtyBlocks() tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.Editor.Blocks {
		block := &m.Editor.Blocks[i]
		if !block.IsDirty {
			continue
		}

		switch block.Type {
		case editor.MathBlock:
			if m.Editor.Cursor.BlockIdx == i {
				continue
			}
			block.IsDirty = false
			m.PendingRenders++
			blockIdx := i
			lines := make([]string, len(block.Lines))
			copy(lines, block.Lines)
			numLines := len(block.Lines)
			cmds = append(cmds, func() tea.Msg {
				contentLines := lines
				if len(contentLines) >= 2 && contentLines[0] == "$$" && contentLines[len(contentLines)-1] == "$$" {
					contentLines = contentLines[1 : len(contentLines)-1]
				}
				start := 0
				for start < len(contentLines) && strings.TrimSpace(contentLines[start]) == "" {
					start++
				}
				content := strings.Join(contentLines[start:], "\n")
				if strings.TrimSpace(content) == "" {
					return BlockProcessedMsg{
						BlockIdx: blockIdx, ImageID: 0, ImageCols: 0,
						ImageHeight: 0, Error: nil, Source: "",
					}
				}
				path, err := latex.CompileToPNG(content, m.Config.CacheDir, false)
				var info latex.ImageInfo
				if err == nil {
					info, err = latex.TransmitImageForKitty(path, numLines, 0)
				}
				return BlockProcessedMsg{
					BlockIdx: blockIdx, ImageID: info.ImageID, ImageCols: info.Cols,
					ImageHeight: info.Rows, Error: err, Source: content,
				}
			})

		case editor.TextBlock:
			anyProcessed := false
			for lineIdx, line := range block.Lines {
				isCursorLine := m.Editor.Cursor.BlockIdx == i && m.Editor.Cursor.LineIdx == lineIdx && m.mode == Insert
				matches := inlineMathRe.FindAllStringIndex(line, -1)

				if isCursorLine {
					filtered := make([][]int, 0, len(matches))
					for _, match := range matches {
						if m.Editor.Cursor.Col < match[0] || m.Editor.Cursor.Col >= match[1] {
							filtered = append(filtered, match)
						}
					}
					matches = filtered
				}

				prefix := fmt.Sprintf("%d-%d-", i, lineIdx)
				for key, render := range m.InlineRenders {
					if strings.HasPrefix(key, prefix) {
						if render.ImageID != 0 {
							latex.DeleteImage(render.ImageID)
						}
						delete(m.InlineRenders, key)
					}
				}

				for _, match := range matches {
					m.PendingRenders++
					blockIdx := i
					lIdx := lineIdx
					start, end := match[0], match[1]
					content := line[start+1 : end-1]
					cmds = append(cmds, func() tea.Msg {
						path, err := latex.CompileToPNG(content, m.Config.CacheDir, true)
						var info latex.ImageInfo
						if err == nil {
							info, err = latex.TransmitImageForKitty(path, 1, end-start)
						}
						return InlineMathProcessedMsg{
							BlockIdx: blockIdx, LineIdx: lIdx, StartCol: start, EndCol: end,
							ImageID: info.ImageID, ImageCols: info.Cols,
							ImageHeight: info.Rows, Error: err,
						}
					})
					anyProcessed = true
				}
			}
			if m.Editor.Cursor.BlockIdx != i || anyProcessed {
				block.IsDirty = false
			}
		}
	}
	return tea.Batch(cmds...)
}

// updateParsedDoc re-parses the document when blocks change
func (m *Model) updateParsedDoc() {
	m.ParsedDoc = editor.ParseDocument(m.Editor.Blocks)
}

// updateEditorSize adjusts the editor size based on file tree visibility
func (m *Model) updateEditorSize() {
	widthAdjust := 0
	if m.ShowFileTree {
		widthAdjust = m.FileTree.Width + 1
	}
	m.Editor.SetSize(m.width-widthAdjust, m.height-1)
}

// loadFile loads a file into the editor
func (m *Model) loadFile(path string) error {
	// Clean up old images before loading new file
	for _, block := range m.Editor.Blocks {
		if block.ImageID != 0 {
			latex.DeleteImage(block.ImageID)
		}
	}
	for _, render := range m.InlineRenders {
		if render.ImageID != 0 {
			latex.DeleteImage(render.ImageID)
		}
	}
	// Clear inline renders
	m.InlineRenders = make(map[string]InlineMathRender)

	model, err := editor.LoadFromFile(path)
	if err != nil {
		return err
	}
	m.Editor = *model
	m.ParsedDoc = editor.ParseDocument(m.Editor.Blocks)
	// Mark all blocks as dirty to trigger math compilation
	for i := range m.Editor.Blocks {
		m.Editor.Blocks[i].IsDirty = true
	}
	// Update editor size to account for file tree
	m.updateEditorSize()
	m.CurrentFile = path
	return nil
}

// createNewNote creates a new note in the current notebook
func (m *Model) createNewNote(name string) error {
	spec := notebook.ParseNoteName(name, m.NotebookPath)
	if err := spec.Create(); err != nil {
		return err
	}
	// Refresh file tree
	m.FileTree.Refresh()
	// Load the new note
	if err := m.loadFile(spec.Path); err != nil {
		return err
	}
	return nil
}

// getSlashQuery extracts the current slash command query from the editor
func (m *Model) getSlashQuery() string {
	if m.Editor.Cursor.BlockIdx >= len(m.Editor.Blocks) {
		return ""
	}
	block := m.Editor.Blocks[m.Editor.Cursor.BlockIdx]
	if m.Editor.Cursor.LineIdx >= len(block.Lines) {
		return ""
	}
	line := block.Lines[m.Editor.Cursor.LineIdx]
	col := m.Editor.Cursor.Col

	// Check if we're after a slash
	if m.slashStartCol < 0 || col <= m.slashStartCol {
		return ""
	}

	// Extract text from slash to cursor
	runes := []rune(line)
	if m.slashStartCol >= len(runes) {
		return ""
	}
	if runes[m.slashStartCol] != '/' {
		return ""
	}

	return string(runes[m.slashStartCol:col])
}

// confirmAutocomplete inserts the selected slash command
func (m *Model) confirmAutocomplete() {
	cmd := m.Autocomplete.GetSelected()
	if cmd == nil {
		m.Autocomplete.Close()
		return
	}

	// Delete the slash and query text
	block := &m.Editor.Blocks[m.Editor.Cursor.BlockIdx]
	line := block.Lines[m.Editor.Cursor.LineIdx]
	runes := []rune(line)

	// Remove from slashStartCol to current cursor
	newLine := string(runes[:m.slashStartCol]) + string(runes[m.Editor.Cursor.Col:])
	block.Lines[m.Editor.Cursor.LineIdx] = newLine
	m.Editor.Cursor.Col = m.slashStartCol

	// Special handling for math block
	if cmd.Trigger == "math" {
		m.insertMathBlock()
		m.Autocomplete.Close()
		block.IsDirty = true
		return
	}

	// Insert the snippet, handling newlines properly
	// Track cursor position for CursorPos
	var cursorLine, cursorCol int
	for i, r := range cmd.Snippet {
		if r == '\n' {
			m.Editor.InsertNewLine()
		} else {
			m.Editor.InsertChar(r)
		}
		// Record position at CursorPos
		if i == cmd.CursorPos-1 {
			cursorLine = m.Editor.Cursor.LineIdx
			cursorCol = m.Editor.Cursor.Col
		}
	}

	// Move cursor to recorded position if CursorPos > 0
	if cmd.CursorPos > 0 {
		m.Editor.Cursor.LineIdx = cursorLine
		m.Editor.Cursor.Col = cursorCol
	}

	m.Autocomplete.Close()
	block.IsDirty = true
}

// insertMathBlock creates a properly structured math block at cursor position
func (m *Model) insertMathBlock() {
	blockIdx := m.Editor.Cursor.BlockIdx
	lineIdx := m.Editor.Cursor.LineIdx
	col := m.Editor.Cursor.Col

	block := &m.Editor.Blocks[blockIdx]
	line := block.Lines[lineIdx]
	runes := []rune(line)

	// Split the line at cursor
	leftPart := string(runes[:col])
	rightPart := string(runes[col:])

	// Split the current block
	var leftBlockLines []string
	var rightBlockLines []string

	// Lines before current line go to left block
	leftBlockLines = append(leftBlockLines, block.Lines[:lineIdx]...)
	// Current line split goes to both
	if leftPart != "" {
		leftBlockLines = append(leftBlockLines, leftPart)
	}
	if rightPart != "" {
		rightBlockLines = append(rightBlockLines, rightPart)
	}
	// Lines after current line go to right block
	rightBlockLines = append(rightBlockLines, block.Lines[lineIdx+1:]...)

	// Create math block
	mathBlock := editor.Block{
		Type:    editor.MathBlock,
		Lines:   []string{"$$", "", "$$"},
		IsDirty: true,
	}

	// Rebuild blocks slice
	var newBlocks []editor.Block

	// Add blocks before current
	for i := 0; i < blockIdx; i++ {
		newBlocks = append(newBlocks, m.Editor.Blocks[i])
	}

	// Add left text block (if not empty)
	if len(leftBlockLines) > 0 {
		newBlocks = append(newBlocks, editor.Block{
			Type:    editor.TextBlock,
			Lines:   leftBlockLines,
			IsDirty: true,
		})
	}

	// Add math block
	newBlocks = append(newBlocks, mathBlock)

	// Add right text block (if not empty)
	if len(rightBlockLines) > 0 {
		newBlocks = append(newBlocks, editor.Block{
			Type:    editor.TextBlock,
			Lines:   rightBlockLines,
			IsDirty: true,
		})
	}

	// Add blocks after current
	for i := blockIdx + 1; i < len(m.Editor.Blocks); i++ {
		newBlocks = append(newBlocks, m.Editor.Blocks[i])
	}

	// Handle edge case: if no blocks, add empty text block
	if len(newBlocks) == 0 {
		newBlocks = []editor.Block{
			{Type: editor.TextBlock, Lines: []string{""}},
		}
	}

	m.Editor.Blocks = newBlocks

	// Position cursor on the empty line inside math block (line 1, col 0)
	// Find math block index
	for i, b := range m.Editor.Blocks {
		if b.Type == editor.MathBlock && len(b.Lines) == 3 && b.Lines[0] == "$$" && b.Lines[2] == "$$" {
			m.Editor.Cursor.BlockIdx = i
			m.Editor.Cursor.LineIdx = 1
			m.Editor.Cursor.Col = 0
			break
		}
	}
}

// executeCommand handles command mode commands
// Returns true if the application should quit
func (m *Model) executeCommand() bool {
	cmd := strings.TrimSpace(m.CmdInput.Value())
	cmd = strings.TrimPrefix(cmd, ":")
	cmd = strings.TrimSpace(cmd)

	switch cmd {
	case "w", "write":
		if err := m.Editor.SaveToFile(m.Config.NotesDir, m.CurrentFile); err != nil {
			m.StatusMessage = fmt.Sprintf("Error: %v", err)
			return false
		}
		m.StatusMessage = "File saved successfully"
		return false
	case "wq":
		if err := m.Editor.SaveToFile(m.Config.NotesDir, m.CurrentFile); err != nil {
			m.StatusMessage = fmt.Sprintf("Error: %v", err)
			return false
		}
		return true
	case "q", "quit":
		return true
	case "new":
		if m.NotebookPath == "" {
			m.StatusMessage = "No notebook open"
			return false
		}
		m.mode = NewNote
		m.NewNoteDialog.Activate()
		return false
	case "h", "help":
		m.mode = Help
		m.HelpDialog.Activate()
		m.CmdInput.SetValue("")
		m.CmdInput.Blur()
		return false
	case "delete", "del":
		if m.CurrentFile == "" {
			m.StatusMessage = "No file open to delete"
			return false
		}
		m.mode = DeleteConfirm
		m.DeleteConfirmDialog.Activate()
		m.CmdInput.SetValue("")
		m.CmdInput.Blur()
		return false
	default:
		m.StatusMessage = fmt.Sprintf("unknown command: %s", cmd)
		return false
	}
}

func InitialModel(cfg *config.Config) Model {
	ti := textinput.New()
	ti.Prompt = "> "
	ti.Placeholder = ""
	ti.SetVirtualCursor(false)
	ti.CharLimit = 30

	m := Model{
		mode:               Normal,
		Time:               time.Now(),
		Editor:             editor.NewModel(),
		Config:             cfg,
		InlineRenders:      make(map[string]InlineMathRender),
		CmdInput:           ti,
		FileTree:           NewFileTree(cfg.NotesDir),
		ShowFileTree:       false,
		NewNoteDialog:      NewNewNoteDialog(),
		HelpDialog:         NewHelpDialog(),
		DeleteConfirmDialog: NewConfirmDialog("Delete Note", "Delete this note?"),
		Autocomplete:       NewAutocompleteBox(),
	}
	// Initialize parsed document
	m.ParsedDoc = editor.ParseDocument(m.Editor.Blocks)
	return m
}

func InitialModelWithNotebook(cfg *config.Config, notebookPath, notebookName string) Model {
	m := InitialModel(cfg)
	m.NotebookName = notebookName
	m.NotebookPath = notebookPath
	m.FileTree = NewFileTreeForNotebook(notebookPath, notebookName)
	m.ShowFileTree = true
	m.FileTree.Focused = true
	return m
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(doTick(), m.processDirtyBlocks())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		oldCursor := m.Editor.Cursor
		oldMode := m.mode

		if m.mode == Insert {
			switch msg.String() {
			case "left":
				m.Editor.MoveCursor(0, -1)
				m.Autocomplete.Close()
			case "down":
				m.Editor.MoveCursor(1, 0)
				m.Autocomplete.Close()
			case "up":
				m.Editor.MoveCursor(-1, 0)
				m.Autocomplete.Close()
			case "right":
				m.Editor.MoveCursor(0, 1)
				m.Autocomplete.Close()
			case "backspace", "delete":
				m.Editor.Backspace()
				// Update autocomplete query after backspace
				if m.Autocomplete.IsActive() {
					query := m.getSlashQuery()
					if query == "" {
						m.Autocomplete.Close()
					} else {
						m.Autocomplete.UpdateQuery(query)
					}
				}
			case "enter":
				if m.Autocomplete.IsActive() {
					m.confirmAutocomplete()
				} else {
					m.Editor.InsertNewLine()
				}
			case "tab":
				if m.Autocomplete.IsActive() {
					m.Autocomplete.MoveDown()
				} else {
					m.Editor.InsertChar('\t')
				}
			case "shift+tab":
				if m.Autocomplete.IsActive() {
					m.Autocomplete.MoveUp()
				}
			case "space":
				m.Editor.InsertChar(' ')
				m.Autocomplete.Close()
			case "esc":
				m.mode = Normal
				m.Autocomplete.Close()
				cmds = append(cmds, m.processDirtyBlocks())
			default:
				if msg.Text != "" {
					for _, r := range msg.Text {
						m.Editor.InsertChar(r)
					}
					// Check for slash command trigger
					if msg.Text == "/" {
						m.slashStartCol = m.Editor.Cursor.Col - 1
						m.Autocomplete.Start("/")
					} else if m.Autocomplete.IsActive() {
						// Update autocomplete query
						query := m.getSlashQuery()
						if query == "" {
							m.Autocomplete.Close()
						} else {
							m.Autocomplete.UpdateQuery(query)
						}
					}
				}
			}
		} else if m.mode == Command {
			switch msg.String() {
			case "ctrl+c", "esc":
				m.mode = Normal
				m.CmdInput.SetValue("")
				m.CmdInput.Blur()
				m.StatusMessage = ""
			case "enter":
				if m.executeCommand() {
					return m, tea.Quit
				}
				if m.mode != NewNote && m.mode != Help && m.mode != DeleteConfirm {
					m.mode = Normal
					m.CmdInput.SetValue("")
					m.CmdInput.Blur()
				}
			default:
				var cmd tea.Cmd
				m.CmdInput, cmd = m.CmdInput.Update(msg)
				cmds = append(cmds, cmd)
			}
		} else if m.mode == NewNote {
			switch msg.String() {
			case "ctrl+c", "esc":
				m.mode = Normal
				m.NewNoteDialog.Deactivate()
				m.StatusMessage = ""
			case "enter":
				noteName := m.NewNoteDialog.Value()
				if noteName != "" {
					if err := m.createNewNote(noteName); err != nil {
						m.StatusMessage = fmt.Sprintf("Error: %v", err)
					} else {
						m.StatusMessage = ""
					}
				}
				m.mode = Normal
				m.NewNoteDialog.Deactivate()
			default:
				m.NewNoteDialog.Update(msg)
			}
		} else if m.mode == Help {
			// Any key closes help dialog
			m.mode = Normal
			m.HelpDialog.Deactivate()
		} else if m.mode == DeleteConfirm {
			switch msg.String() {
			case "left", "h":
				m.DeleteConfirmDialog.SelectYes()
			case "right", "l":
				m.DeleteConfirmDialog.SelectNo()
			case "enter":
				if m.DeleteConfirmDialog.IsYesSelected() {
					// Delete the file
					if m.CurrentFile != "" {
						if err := os.Remove(m.CurrentFile); err != nil {
							m.StatusMessage = fmt.Sprintf("Error deleting file: %v", err)
						} else {
							m.StatusMessage = "Note deleted"
							m.CurrentFile = ""
							m.Editor = editor.NewModel()
							m.ParsedDoc = editor.ParseDocument(m.Editor.Blocks)
							m.FileTree.Refresh()
						}
					}
				}
				m.mode = Normal
				m.DeleteConfirmDialog.Deactivate()
			case "esc":
				m.mode = Normal
				m.DeleteConfirmDialog.Deactivate()
			}
		} else {
			// Handle space+key combinations first (works regardless of file tree focus)
			if m.pendingSpace {
				m.pendingSpace = false
				switch msg.String() {
				case "f":
					m.ShowFileTree = !m.ShowFileTree
					if m.ShowFileTree {
						m.FileTree.Refresh()
						m.FileTree.Focused = true
					} else {
						m.FileTree.Focused = false
					}
					m.updateEditorSize()
				case "/":
					if m.ShowFileTree {
						m.FileTree.Focused = !m.FileTree.Focused
					}
				}
			} else if m.ShowFileTree && m.FileTree.Focused {
				// Handle file tree navigation when focused
				switch msg.String() {
				case "j", "down":
					m.FileTree.MoveDown()
				case "k", "up":
					m.FileTree.MoveUp()
				case "enter":
					if m.FileTree.IsSelectedDir() {
						m.FileTree.ToggleExpand()
					} else {
						// Load file and close file tree
						path := m.FileTree.GetSelectedPath()
						if path != "" {
							if err := m.loadFile(path); err != nil {
								m.StatusMessage = fmt.Sprintf("Error: %v", err)
							} else {
								m.FileTree.Focused = false
								m.ShowFileTree = false
								m.updateEditorSize()
								m.StatusMessage = ""
								cmds = append(cmds, m.processDirtyBlocks())
							}
						}
					}
				case "esc":
					m.FileTree.Focused = false
				case ":":
					// Allow entering command mode from file tree
					m.mode = Command
					m.CmdInput.SetValue("")
					cmds = append(cmds, m.CmdInput.Focus())
				case "space":
					m.pendingSpace = true
				}
			} else {
				switch msg.String() {
				case "h", "left":
					m.Editor.MoveCursor(0, -1)
				case "j", "down":
					m.Editor.MoveCursor(1, 0)
				case "k", "up":
					m.Editor.MoveCursor(-1, 0)
				case "l", "right":
					m.Editor.MoveCursor(0, 1)
				case "d":
					m.Editor.DeleteChar()
				case "i":
					m.mode = Insert
				case "o":
					m.Editor.EndOfLine()
					m.Editor.InsertNewLine()
					m.mode = Insert
				case ":":
					m.mode = Command
					m.CmdInput.SetValue("")
					cmds = append(cmds, m.CmdInput.Focus())
				case "space":
					m.pendingSpace = true
				}
			}
		}

		modeChanged := oldMode != m.mode
		cursorMoved := oldCursor != m.Editor.Cursor

		if modeChanged || m.mode == Insert {
			if m.Editor.Cursor.BlockIdx < len(m.Editor.Blocks) {
				m.Editor.Blocks[m.Editor.Cursor.BlockIdx].IsDirty = true
			}
			if oldCursor.BlockIdx != m.Editor.Cursor.BlockIdx && oldCursor.BlockIdx < len(m.Editor.Blocks) {
				m.Editor.Blocks[oldCursor.BlockIdx].IsDirty = true
			}
		}

		if modeChanged && !cursorMoved {
			cmds = append(cmds, m.processDirtyBlocks())
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateEditorSize()

	case BlockProcessedMsg:
		m.PendingRenders--
		if msg.Source != "" {
			m.CompiledMath = append(m.CompiledMath, msg.Source)
		}
		if msg.BlockIdx < len(m.Editor.Blocks) {
			block := &m.Editor.Blocks[msg.BlockIdx]
			if block.ImageID != 0 && block.ImageID != msg.ImageID {
				latex.DeleteImage(block.ImageID)
			}
			block.ImageID = msg.ImageID
			block.ImageCols = msg.ImageCols
			block.ImageHeight = msg.ImageHeight
			block.HasError = msg.Error != nil
			if msg.Error != nil {
				block.ErrorMessage = msg.Error.Error()
			} else {
				block.ErrorMessage = ""
			}
		}

	case InlineMathProcessedMsg:
		m.PendingRenders--
		if msg.Error == nil && msg.BlockIdx < len(m.Editor.Blocks) {
			key := fmt.Sprintf("%d-%d-%d", msg.BlockIdx, msg.LineIdx, msg.StartCol)
			m.InlineRenders[key] = InlineMathRender{
				ImageID:     msg.ImageID,
				ImageCols:   msg.ImageCols,
				ImageHeight: msg.ImageHeight,
				Length:      msg.EndCol - msg.StartCol,
			}
		}

	case TickMsg:
		m.Time = time.Time(msg)
		if m.PendingRenders == 0 {
			for _, b := range m.Editor.Blocks {
				if b.IsDirty {
					// Re-parse document when blocks are dirty
					m.updateParsedDoc()
					cmds = append(cmds, m.processDirtyBlocks())
					break
				}
			}
		}
		return m, tea.Batch(doTick(), tea.Batch(cmds...))

	case tea.MouseClickMsg:
		// Handle link clicking
		if url := m.getLinkAtPosition(msg.X, msg.Y); url != "" {
			if err := clipboard.WriteAll(url); err == nil {
				m.StatusMessage = fmt.Sprintf("Copied: %s", url)
			} else {
				m.StatusMessage = fmt.Sprintf("Failed to copy: %v", err)
			}
		}
	}

	return m, tea.Batch(cmds...)
}

// linkRegex matches markdown links and images
var linkRegex = regexp.MustCompile(`!?\[([^\]]*)\]\(([^)]+)\)`)

// getLinkAtPosition returns the URL if a link is at the given screen position
func (m Model) getLinkAtPosition(x, y int) string {
	// Calculate layout offsets
	gutterWidth := 0
	totalLines := 0
	for _, block := range m.Editor.Blocks {
		totalLines += len(block.Lines)
	}
	if totalLines > 0 {
		gutterWidth = len(fmt.Sprint(totalLines))
	}

	// Calculate offset for file tree
	fileTreeOffset := 0
	if m.ShowFileTree {
		fileTreeOffset = 25
	}

	// Content area starts at x = 2 + gutterWidth + 3 + fileTreeOffset
	contentStartX := 2 + gutterWidth + 3 + fileTreeOffset
	// Content area starts at y = 0 (with scroll offset)
	contentStartY := 0

	// Check if click is in content area
	if x < contentStartX {
		return ""
	}

	// Calculate relative position in content
	relX := x - contentStartX
	relY := y - contentStartY

	// Calculate absolute line number including scroll offset
	offsetAbsLine := 0
	for i := 0; i < m.Editor.Offset.BlockIdx && i < len(m.Editor.Blocks); i++ {
		offsetAbsLine += len(m.Editor.Blocks[i].Lines)
	}
	offsetAbsLine += m.Editor.Offset.LineIdx

	absLine := relY + offsetAbsLine

	// Find block and line index for this absolute line
	currentAbsLine := 0
	for blockIdx, block := range m.Editor.Blocks {
		for lineIdx, line := range block.Lines {
			if currentAbsLine == absLine {
				// Found the line - check for links
				return m.extractLinkAtColumn(line, relX)
			}
			currentAbsLine++
			_ = lineIdx // not used
		}
		_ = blockIdx // not used
	}

	return ""
}

// extractLinkAtColumn finds a link at the given visual column and returns its URL
func (m Model) extractLinkAtColumn(line string, visualCol int) string {
	runes := []rune(line)
	
	// Find all links in the line
	matches := linkRegex.FindAllStringSubmatchIndex(string(runes), -1)
	
	for _, match := range matches {
		// match[0] is start of full match, match[1] is end
		// match[4] is start of URL (group 2), match[5] is end
		if len(match) >= 6 {
			linkStart := match[0]
			linkEnd := match[1]
			urlStart := match[4]
			urlEnd := match[5]
			
			// Convert to visual columns
			visualStart := editor.RuneColToVisualCol(line, linkStart)
			visualEnd := editor.RuneColToVisualCol(line, linkEnd)
			
			// Check if click is within this link
			if visualCol >= visualStart && visualCol < visualEnd {
				return string(runes[urlStart:urlEnd])
			}
		}
	}
	
	return ""
}
