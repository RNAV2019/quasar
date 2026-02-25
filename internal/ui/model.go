package ui

import (
	"time"

	"github.com/RNAV2019/quasar/internal/editor"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	width  int
	height int
	Time   time.Time
	Editor editor.Model
}

type TickMsg time.Time

func doTick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func InitialModel() Model {
	return Model{
		Time:   time.Now(),
		Editor: editor.NewModel(),
	}
}

func (m Model) Init() tea.Cmd {
	return doTick()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC:
			return m, tea.Quit
		case tea.KeyLeft:
			m.Editor.MoveCursor(0, -1)
		case tea.KeyDown:
			m.Editor.MoveCursor(1, 0)
		case tea.KeyUp:
			m.Editor.MoveCursor(-1, 0)
		case tea.KeyRight:
			m.Editor.MoveCursor(0, 1)
		case tea.KeyBackspace, tea.KeyDelete:
			m.Editor.Backspace()
		case tea.KeyEnter:
			m.Editor.InsertNewLine()
		case tea.KeySpace:
			m.Editor.InsertChar(' ')
		case tea.KeyRunes:
			for _, r := range msg.Runes {
				m.Editor.InsertChar(r)
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		m.Editor.SetSize(m.width, m.height-1)
	case TickMsg:
		m.Time = time.Time(msg)
		return m, doTick()
	}
	return m, nil
}
