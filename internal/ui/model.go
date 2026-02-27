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
}

type TickMsg time.Time

type LineProcessedMsg struct {
	Row         int
	Rendered    string
	ImageHeight int
}

func redrawImages(m Model) tea.Cmd {
	return func() tea.Msg {
		var seq strings.Builder

		// 1. Clear all existing Kitty images (Action: Delete, Delete: All)
		seq.WriteString("\x1b_Ga=d,d=A;\x1b\\")

		editorLines := m.Editor.ViewLines()
		gutterWidth := len(fmt.Sprint(len(m.Editor.Lines)))

		// Keep a running tally of physical Y-rows.
		// If you have a top status bar, change this to 2.
		currentY := 1

		// 2. Loop through visible lines and explicitly draw virtual images
		for i := range editorLines {
			actualRowIdx := m.Editor.Offset.Row + i
			if actualRowIdx >= len(m.Editor.Lines) {
				break
			}

			line := m.Editor.Lines[actualRowIdx]

			if actualRowIdx != m.Editor.Cursor.Row && line.IsMath && line.Rendered != "" {
				// Calculate absolute ANSI coordinates (1-based)
				imageX := 3 + gutterWidth + 2 + 1
				imageY := currentY

				// Save cursor (\033[s), move to coordinate, print image, restore cursor (\033[u)
				seq.WriteString("\033[s")
				fmt.Fprintf(&seq, "\033[%d;%dH", imageY, imageX)
				seq.WriteString(line.Rendered)
				seq.WriteString("\033[u")

				// Advance Y by the image's height plus padding
				currentY += max(line.ImageHeight, 1)
			} else {
				// Standard text lines take exactly 1 row
				currentY += 1
			}
		}

		// 3. Write directly to stdout, bypassing Bubble Tea entirely
		os.Stdout.WriteString(seq.String())

		return nil
	}
}

func doTick() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m Model) processDirtyLines() tea.Cmd {
	for rowIdx, line := range m.Editor.Lines {
		if line.IsDirty && rowIdx != m.Editor.Cursor.Row {
			// Check if it's math
			isMath := latex.IsMath(line.Raw)

			return func() tea.Msg {
				var rendered string
				var height int
				if isMath {
					path, err := latex.CompileToPNG(line.Raw, m.Config.CacheDir)
					if err == nil {
						rendered, height, err = latex.EncodeImageForKitty(path)
						if err != nil {
							fmt.Printf("kitty encoded err: %v\n", err)
						}
					}
				}
				return LineProcessedMsg{
					Row:         rowIdx,
					Rendered:    rendered,
					ImageHeight: height,
				}
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
		oldOffset := m.Editor.Offset.Row
		oldCursorRow := m.Editor.Cursor.Row
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
				return m, nil
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
				m.Editor.Backspace()
			case "i":
				m.mode = Insert
				return m, nil
			case "o":
				m.Editor.EndOfLine()
				m.Editor.InsertNewLine()
				m.mode = Insert
				return m, nil
			}
		}
		newOffset := m.Editor.Offset.Row
		newCursorRow := m.Editor.Cursor.Row

		wasMath := false
		if oldCursorRow < len(m.Editor.Lines) {
			wasMath = m.Editor.Lines[oldCursorRow].IsMath
		}

		isMath := false
		if newCursorRow < len(m.Editor.Lines) {
			isMath = m.Editor.Lines[newCursorRow].IsMath
		}

		if oldOffset != newOffset || wasMath || isMath {
			cmds = append(cmds, redrawImages(m))
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.Editor.SetSize(m.width, m.height-1)
		cmds = append(cmds, redrawImages(m))
	case LineProcessedMsg:
		if msg.Row < len(m.Editor.Lines) {
			m.Editor.Lines[msg.Row].Rendered = msg.Rendered
			m.Editor.Lines[msg.Row].IsDirty = false
			m.Editor.Lines[msg.Row].IsMath = msg.Rendered != ""
			m.Editor.Lines[msg.Row].ImageHeight = msg.ImageHeight
		}
		cmds = append(cmds, m.processDirtyLines(), redrawImages(m))
	case TickMsg:
		m.Time = time.Time(msg)
		return m, tea.Batch(doTick(), m.processDirtyLines())
	}
	return m, tea.Batch(cmds...)
}
