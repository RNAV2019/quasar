package ui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/RNAV2019/quasar/internal/config"
	"github.com/RNAV2019/quasar/internal/editor"
	"github.com/RNAV2019/quasar/internal/latex"
)

type Mode int

const (
	Normal Mode = iota
	Insert
	Select
	Command
)

type Model struct {
	mode           Mode
	width          int
	height         int
	Time           time.Time
	Editor         editor.Model
	Config         *config.Config
	InlineRenders  map[string]InlineMathRender
	PendingRenders int
	CompiledMath   []string
	CmdInput       textinput.Model
	StatusMessage  string
	ParsedDoc      *editor.Document // Parsed document structure
	FileTree       *FileTree
	ShowFileTree   bool
	pendingSpace   bool // Track if space was pressed (for space+key combos)
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
	return nil
}

// executeCommand handles command mode commands
// Returns true if the application should quit
func (m *Model) executeCommand() bool {
	cmd := strings.TrimSpace(m.CmdInput.Value())
	cmd = strings.TrimPrefix(cmd, ":")
	cmd = strings.TrimSpace(cmd)

	switch cmd {
	case "w", "write":
		if err := m.Editor.SaveToFile(m.Config.NotesDir); err != nil {
			m.StatusMessage = fmt.Sprintf("Error: %v", err)
			return false
		}
		m.StatusMessage = "File saved successfully"
		return false
	case "q", "quit":
		return true
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
		mode:          Normal,
		Time:          time.Now(),
		Editor:        editor.NewModel(),
		Config:        cfg,
		InlineRenders: make(map[string]InlineMathRender),
		CmdInput:      ti,
		FileTree:      NewFileTree(cfg.NotesDir),
		ShowFileTree:  false,
	}
	// Initialize parsed document
	m.ParsedDoc = editor.ParseDocument(m.Editor.Blocks)
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
			case "down":
				m.Editor.MoveCursor(1, 0)
			case "up":
				m.Editor.MoveCursor(-1, 0)
			case "right":
				m.Editor.MoveCursor(0, 1)
			case "backspace", "delete":
				m.Editor.Backspace()
			case "enter":
				m.Editor.InsertNewLine()
			case "tab":
				m.Editor.InsertChar('\t')
			case "space":
				m.Editor.InsertChar(' ')
			case "esc":
				m.mode = Normal
				cmds = append(cmds, m.processDirtyBlocks())
			default:
				if msg.Text != "" {
					for _, r := range msg.Text {
						m.Editor.InsertChar(r)
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
				m.mode = Normal
				m.CmdInput.SetValue("")
				m.CmdInput.Blur()
			default:
				var cmd tea.Cmd
				m.CmdInput, cmd = m.CmdInput.Update(msg)
				cmds = append(cmds, cmd)
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
	}

	return m, tea.Batch(cmds...)
}
