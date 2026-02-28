package ui

import (
	"fmt"
	"os"
	"regexp"
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
	mode              Mode
	width             int
	height            int
	Time              time.Time
	Editor            editor.Model
	Config            *config.Config
	InlineRenders     map[string]InlineMathRender
	PendingRenders    int
	CompiledMath      []string
	needsImageRedraw  bool
}

type TickMsg time.Time

type BlockProcessedMsg struct {
	BlockIdx    int
	Rendered    string
	ImageHeight int
	Error       error
	Source      string
}

type InlineMathProcessedMsg struct {
	BlockIdx    int
	LineIdx     int
	StartCol    int
	EndCol      int
	Rendered    string
	ImageHeight int
	Error       error
}

type InlineMathRender struct {
	Rendered    string
	ImageHeight int
	Length      int
}

var inlineMathRe = regexp.MustCompile(`\$[^\$]*\$`)

func (m *Model) cursorInBlock(blockIdx int) bool {
	return m.Editor.Cursor.BlockIdx == blockIdx
}

func (m *Model) isBlockEditable(blockIdx int) bool {
	return m.cursorInBlock(blockIdx) && m.mode == Insert
}

func (m *Model) shouldShowRawMathBlock(blockIdx int) bool {
	return m.cursorInBlock(blockIdx)
}

func (m *Model) getHoveredMathIndex(blockIdx, lineIdx, col int) int {
	if blockIdx >= len(m.Editor.Blocks) {
		return -1
	}
	block := m.Editor.Blocks[blockIdx]
	if block.Type != editor.TextBlock || lineIdx >= len(block.Lines) {
		return -1
	}
	for i, match := range inlineMathRe.FindAllStringIndex(block.Lines[lineIdx], -1) {
		if col >= match[0] && col < match[1] {
			return i
		}
	}
	return -1
}

func (m *Model) gutterWidth() int {
	totalLines := 0
	for _, block := range m.Editor.Blocks {
		totalLines += len(block.Lines)
	}
	return len(fmt.Sprint(totalLines))
}

func redrawImages(m Model) tea.Cmd {
	return func() tea.Msg {
		var seq strings.Builder
		seq.WriteString("\x1b_Ga=d,d=A;\x1b\\")

		gutterWidth := m.gutterWidth()
		currentY := 1

		for blockIdx, block := range m.Editor.Blocks {
			if m.shouldShowRawMathBlock(blockIdx) {
				currentY += len(block.Lines)
				continue
			}
			if block.Type == editor.MathBlock && block.Rendered != "" {
				imageX := 2 + gutterWidth + 3 + 1
				seq.WriteString("\033[s")
				fmt.Fprintf(&seq, "\033[%d;%dH", currentY, imageX)
				seq.WriteString(block.Rendered)
				seq.WriteString("\033[u")
			}
			currentY += len(block.Lines)
		}

		for key, render := range m.InlineRenders {
			var blockIdx, lineIdx, startCol int
			fmt.Sscanf(key, "%d-%d-%d", &blockIdx, &lineIdx, &startCol)

			if m.isBlockEditable(blockIdx) {
				continue
			}
			if m.mode == Normal && m.cursorInBlock(blockIdx) && m.Editor.Cursor.LineIdx == lineIdx {
				if m.Editor.Cursor.Col >= startCol && m.Editor.Cursor.Col < startCol+render.Length {
					continue
				}
			}

			imageY := 1
			for i := 0; i < blockIdx; i++ {
				imageY += len(m.Editor.Blocks[i].Lines)
			}
			imageY += lineIdx
			imageX := 2 + gutterWidth + 3 + startCol + 1

			seq.WriteString("\033[s")
			fmt.Fprintf(&seq, "\033[%d;%dH", imageY, imageX)
			seq.WriteString(render.Rendered)
			seq.WriteString("\033[u")
		}

		os.Stdout.WriteString(seq.String())
		return nil
	}
}

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
			cmds = append(cmds, func() tea.Msg {
				contentLines := block.Lines
				if len(contentLines) >= 2 && contentLines[0] == "$$" && contentLines[len(contentLines)-1] == "$$" {
					contentLines = contentLines[1 : len(contentLines)-1]
				}
				start := 0
				for start < len(contentLines) && strings.TrimSpace(contentLines[start]) == "" {
					start++
				}
				content := strings.Join(contentLines[start:], "\n")
				path, err := latex.CompileToPNG(content, m.Config.CacheDir, false)
				var rendered string
				var height int
				if err == nil {
					rendered, height, _ = latex.EncodeImageForKitty(path, len(block.Lines))
				}
				return BlockProcessedMsg{
					BlockIdx: i, Rendered: rendered, ImageHeight: height, Error: err, Source: content,
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
				for key := range m.InlineRenders {
					if strings.HasPrefix(key, prefix) {
						delete(m.InlineRenders, key)
					}
				}

				for _, match := range matches {
					m.PendingRenders++
					start, end := match[0], match[1]
					content := line[start+1 : end-1]
					cmds = append(cmds, func() tea.Msg {
						path, err := latex.CompileToPNG(content, m.Config.CacheDir, true)
						var rendered string
						var height int
						if err == nil {
							rendered, height, _ = latex.EncodeImageForKitty(path, 1)
						}
						return InlineMathProcessedMsg{
							BlockIdx: i, LineIdx: lineIdx, StartCol: start, EndCol: end,
							Rendered: rendered, ImageHeight: height, Error: err,
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

func InitialModel(cfg *config.Config) Model {
	return Model{
		mode:          Normal,
		Time:          time.Now(),
		Editor:        editor.NewModel(),
		Config:        cfg,
		InlineRenders: make(map[string]InlineMathRender),
	}
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
		contentChanged := false

		if m.mode == Insert {
			switch msg.String() {
			case "ctrl+c":
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
				cmds = append(cmds, m.processDirtyBlocks())
			default:
				if msg.Text != "" {
					for _, r := range msg.Text {
						m.Editor.InsertChar(r)
					}
				}
			}
		} else {
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
				m.Editor.DeleteChar()
				contentChanged = true
			case "i":
				m.mode = Insert
			case "o":
				m.Editor.EndOfLine()
				m.Editor.InsertNewLine()
				m.mode = Insert
			}
		}

		cursorMoved := oldCursor != m.Editor.Cursor
		modeChanged := oldMode != m.mode
		if !cursorMoved && !modeChanged && !contentChanged {
			break
		}

		if modeChanged || m.mode == Insert {
			m.Editor.Blocks[m.Editor.Cursor.BlockIdx].IsDirty = true
			if oldCursor.BlockIdx != m.Editor.Cursor.BlockIdx {
				m.Editor.Blocks[oldCursor.BlockIdx].IsDirty = true
			}
			cmds = append(cmds, redrawImages(m))
		} else if m.mode == Normal {
			needsRedraw := false

			if contentChanged {
				cmds = append(cmds, m.processDirtyBlocks())
				needsRedraw = true
			}

			if oldCursor.BlockIdx != m.Editor.Cursor.BlockIdx {
				oldBlock := m.Editor.Blocks[oldCursor.BlockIdx]
				newBlock := m.Editor.Blocks[m.Editor.Cursor.BlockIdx]
				if oldBlock.Type == editor.MathBlock || newBlock.Type == editor.MathBlock {
					needsRedraw = true
				}
			}

			oldHover := m.getHoveredMathIndex(oldCursor.BlockIdx, oldCursor.LineIdx, oldCursor.Col)
			newHover := m.getHoveredMathIndex(m.Editor.Cursor.BlockIdx, m.Editor.Cursor.LineIdx, m.Editor.Cursor.Col)
			if oldHover != newHover {
				needsRedraw = true
			}

			if needsRedraw {
				cmds = append(cmds, redrawImages(m))
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.Editor.SetSize(m.width, m.height-1)
		cmds = append(cmds, redrawImages(m))

	case BlockProcessedMsg:
		m.PendingRenders--
		if msg.Source != "" {
			m.CompiledMath = append(m.CompiledMath, msg.Source)
		}
		if msg.BlockIdx < len(m.Editor.Blocks) {
			block := &m.Editor.Blocks[msg.BlockIdx]
			block.Rendered = msg.Rendered
			block.ImageHeight = msg.ImageHeight
			block.HasError = msg.Error != nil
		}
		if m.PendingRenders == 0 {
			m.needsImageRedraw = true
		}

	case InlineMathProcessedMsg:
		m.PendingRenders--
		if msg.Error == nil {
			key := fmt.Sprintf("%d-%d-%d", msg.BlockIdx, msg.LineIdx, msg.StartCol)
			m.InlineRenders[key] = InlineMathRender{
				Rendered:    msg.Rendered,
				ImageHeight: msg.ImageHeight,
				Length:      msg.EndCol - msg.StartCol,
			}
		}
		if m.PendingRenders == 0 {
			m.needsImageRedraw = true
		}

	case TickMsg:
		m.Time = time.Time(msg)
		if m.PendingRenders == 0 {
			for _, b := range m.Editor.Blocks {
				if b.IsDirty {
					cmds = append(cmds, m.processDirtyBlocks())
					break
				}
			}
			if m.needsImageRedraw {
				m.needsImageRedraw = false
				cmds = append(cmds, redrawImages(m))
			}
		}
		return m, tea.Batch(doTick(), tea.Batch(cmds...))
	}

	return m, tea.Batch(cmds...)
}
