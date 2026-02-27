package ui

import (
	"fmt"
	"os"
	"strings"
	"time"

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
)

type Model struct {
	mode   Mode
	width  int
	height int
	Time   time.Time
	Editor editor.Model
	Config *config.Config
	// For debugging LaTeX compilation
	CompiledMath []string
}

type TickMsg time.Time

type BlockProcessedMsg struct {
	BlockIdx    int
	Rendered    string
	ImageHeight int
	Error       error
	Source      string // The raw latex string that was compiled
}

func redrawImages(m Model) tea.Cmd {
	return func() tea.Msg {
		var seq strings.Builder
		// 1. Clear all existing Kitty images (Action: Delete, Delete: All)
		seq.WriteString("\x1b_Ga=d,d=A;\x1b\\")

		totalLines := 0
		for _, block := range m.Editor.Blocks {
			totalLines += len(block.Lines)
		}
		gutterWidth := len(fmt.Sprint(totalLines))

		currentY := 1 // 1-based index for terminal rows

		// Loop through blocks and draw images for inactive math blocks
		for blockIdx, block := range m.Editor.Blocks {
			if block.Type == editor.MathBlock && block.Rendered != "" && m.Editor.Cursor.BlockIdx != blockIdx {
				// This is an inactive, rendered math block. Draw the image.
				imageX := 2 + gutterWidth + 3 + 1 // padding + gutter + spaces
				imageY := currentY

				seq.WriteString("\033[s")                               // Save cursor
				fmt.Fprintf(&seq, "\033[%d;%dH", imageY, imageX) // Move to position
				seq.WriteString(block.Rendered)                         // Print image
				seq.WriteString("\033[u")                               // Restore cursor
			}
			// Always advance Y by the number of raw text lines to ensure stable layout.
			currentY += len(block.Lines)
		}

		// 3. Write directly to stdout
		os.Stdout.WriteString(seq.String())

		return nil
	}
}

func doTick() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m *Model) processDirtyBlocks() tea.Cmd {
	for i := range m.Editor.Blocks {
		block := &m.Editor.Blocks[i]
		if block.IsDirty && m.Editor.Cursor.BlockIdx != i {
			if block.Type == editor.MathBlock {
				// This block is dirty and not active, so we process it.
				return func() tea.Msg {
					var rendered string
					var height int

					contentLines := block.Lines
					// Strip surrounding '$$' lines if they exist
					if len(contentLines) >= 2 && contentLines[0] == "$$" && contentLines[len(contentLines)-1] == "$$" {
						contentLines = contentLines[1 : len(contentLines)-1]
					}

					// Aggressively trim blank lines from the start of the content.
					start := 0
					for start < len(contentLines) && strings.TrimSpace(contentLines[start]) == "" {
						start++
					}
					content := strings.Join(contentLines[start:], "\n")

					path, err := latex.CompileToPNG(content, m.Config.CacheDir)
					if err == nil {
						// Force the image height to be the same as the raw text line count
						targetHeight := len(block.Lines)
						rendered, height, _ = latex.EncodeImageForKitty(path, targetHeight)
					}
					return BlockProcessedMsg{
						BlockIdx:    i,
						Rendered:    rendered,
						ImageHeight: height,
						Error:       err,
						Source:      content, // Add the source for debugging
					}
				}
			} else if block.Type == editor.TextBlock {
				// For text blocks, we just mark them as not dirty.
				block.IsDirty = false
			}
		}
	}
	return nil
}

func InitialModel(cfg *config.Config) Model {
	return Model{
		mode:   Normal,
		Time:   time.Now(),
		Editor: editor.NewModel(),
		Config: cfg,
	}
}

func (m Model) Init() tea.Cmd {
	return doTick()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyPressMsg:
		oldBlockIdx := m.Editor.Cursor.BlockIdx
		oldBlockWasMath := oldBlockIdx < len(m.Editor.Blocks) && m.Editor.Blocks[oldBlockIdx].Type == editor.MathBlock

		if m.mode == Insert {
			switch msg.String() {
			case "ctrl+c", "q":
				return m, tea.Quit
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
			case "space":
				m.Editor.InsertChar(' ')
			case "esc":
				m.mode = Normal
				// When switching mode, we might need to render an image
				if oldBlockWasMath {
					return m, redrawImages(m)
				}
				return m, nil
			default:
				if msg.Text != "" {
					for _, r := range msg.Text {
						m.Editor.InsertChar(r)
					}
				}
			}
		} else { // NORMAL mode
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "h", "left":
				m.Editor.MoveCursor(0, -1)
			case "j", "down":
				m.Editor.MoveCursor(1, 0)
			case "k", "up":
				m.Editor.MoveCursor(-1, 0)
			case "l", "right":
				m.Editor.MoveCursor(0, 1)
			case "d":
				m.Editor.DeleteChar() // Use new forward-delete function
			case "i":
				m.mode = Insert
				// When switching mode, we might need to hide an image
				if oldBlockWasMath {
					return m, redrawImages(m)
				}
				return m, nil
			case "o":
				m.Editor.EndOfLine()
				m.Editor.InsertNewLine()
				m.mode = Insert
				// After 'o', we have likely moved blocks, so redraw
				return m, redrawImages(m)
			}
		}

		newBlockIdx := m.Editor.Cursor.BlockIdx
		newBlockIsMath := newBlockIdx < len(m.Editor.Blocks) && m.Editor.Blocks[newBlockIdx].Type == editor.MathBlock

		// If we moved between blocks and either was a math block, we need to redraw images.
		if oldBlockIdx != newBlockIdx && (oldBlockWasMath || newBlockIsMath) {
			cmds = append(cmds, redrawImages(m))
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.Editor.SetSize(m.width, m.height-1)
		cmds = append(cmds, redrawImages(m))
	case BlockProcessedMsg:
		if msg.Source != "" {
			m.CompiledMath = append(m.CompiledMath, msg.Source)
		}
		if msg.BlockIdx < len(m.Editor.Blocks) {
			block := &m.Editor.Blocks[msg.BlockIdx]
			block.Rendered = msg.Rendered
			block.IsDirty = false
			block.ImageHeight = msg.ImageHeight
			block.HasError = msg.Error != nil // Set the error flag
		}
		cmds = append(cmds, m.processDirtyBlocks(), redrawImages(m))
	case TickMsg:
		m.Time = time.Time(msg)
		return m, tea.Batch(doTick(), m.processDirtyBlocks())
	}
	return m, tea.Batch(cmds...)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
