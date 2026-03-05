package tui

import (
	"fmt"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type setupDoneMsg struct{ err error }

type SetupModel struct {
	spinner spinner.Model
	message string
	done    bool
	err     error
	setupFn func() error
	width   int
	height  int
}

func NewSetupModel(setupFn func() error) SetupModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#89B4FA"))
	return SetupModel{
		spinner: s,
		message: "Preparing quasar for note-taking",
		setupFn: setupFn,
	}
}

func (m SetupModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, func() tea.Msg {
		err := m.setupFn()
		return setupDoneMsg{err: err}
	})
}

func (m SetupModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case setupDoneMsg:
		m.done = true
		m.err = msg.err
		return m, tea.Quit
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m SetupModel) View() tea.View {
	if m.width == 0 {
		return tea.NewView("")
	}

	var content string
	if m.done {
		if m.err != nil {
			content = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F38BA8")).
				Render(fmt.Sprintf("  Setup failed: %v", m.err))
		} else {
			content = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A6E3A1")).
				Render("  Ready")
		}
	} else {
		content = fmt.Sprintf("  %s %s", m.spinner.View(),
			lipgloss.NewStyle().
				Foreground(lipgloss.Color("#CDD6F4")).
				Render(m.message))
	}

	styled := lipgloss.NewStyle().
		PaddingTop(1).
		PaddingLeft(1).
		Render(content)

	return tea.NewView(styled)
}

func (m SetupModel) Err() error {
	return m.err
}

func RunSetup(setupFn func() error) error {
	model := NewSetupModel(setupFn)
	p := tea.NewProgram(model)
	final, err := p.Run()
	if err != nil {
		return fmt.Errorf("setup TUI error: %w", err)
	}
	if m, ok := final.(SetupModel); ok && m.err != nil {
		return m.err
	}
	return nil
}
