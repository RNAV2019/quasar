package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type ConfirmModel struct {
	message    string
	confirmed  *bool
	selected   int // 0 = yes, 1 = no
	width      int
}

func NewConfirmModel(message string) ConfirmModel {
	return ConfirmModel{
		message:  message,
		confirmed: new(bool),
		selected:  1, // Default to "no" for safety
		width:     50,
	}
}

func (m ConfirmModel) Init() tea.Cmd {
	return nil
}

func (m ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "left", "h":
			if m.selected > 0 {
				m.selected--
			}
		case "right", "l":
			if m.selected < 1 {
				m.selected++
			}
		case "enter":
			*m.confirmed = m.selected == 0
			return m, tea.Quit
		case "esc", "ctrl+c":
			*m.confirmed = false
			return m, tea.Quit
		case "y", "Y":
			*m.confirmed = true
			return m, tea.Quit
		case "n", "N":
			*m.confirmed = false
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m ConfirmModel) View() tea.View {
	var b strings.Builder

	b.WriteString(TitleStyle.Render(m.message))
	b.WriteString("\n\n")

	yesText := "  Yes  "
	noText := "  No   "

	yesStyle := lipgloss.NewStyle()
	noStyle := lipgloss.NewStyle()

	if m.selected == 0 {
		yesStyle = yesStyle.Foreground(ConfirmColor).Bold(true)
		noStyle = noStyle.Foreground(DimColor)
	} else {
		yesStyle = yesStyle.Foreground(DimColor)
		noStyle = noStyle.Foreground(CancelColor).Bold(true)
	}

	b.WriteString("  ")
	b.WriteString(yesStyle.Render(yesText))
	b.WriteString("    ")
	b.WriteString(noStyle.Render(noText))
	b.WriteString("\n\n")

	hintStyle := lipgloss.NewStyle().Foreground(DimColor)
	b.WriteString(hintStyle.Render("←/→: select  |  enter: confirm  |  esc: cancel"))

	dialogContent := b.String()
	dialogWidth := m.width
	contentWidth := dialogWidth - 2

	lines := strings.Split(dialogContent, "\n")
	var paddedLines []string
	for _, line := range lines {
		lineWidth := lipgloss.Width(line)
		if lineWidth < contentWidth {
			line += strings.Repeat(" ", contentWidth-lineWidth)
		}
		paddedLines = append(paddedLines, line)
	}

	innerContent := strings.Join(paddedLines, "\n")
	view := DialogStyle.Width(dialogWidth).Render(innerContent)

	return tea.NewView(view)
}

func (m ConfirmModel) Confirmed() bool {
	if m.confirmed == nil {
		return false
	}
	return *m.confirmed
}

func RunConfirm(message string) (bool, error) {
	model := NewConfirmModel(message)
	p := tea.NewProgram(model, tea.WithoutSignalHandler())
	result, err := p.Run()
	if err != nil {
		return false, fmt.Errorf("confirm failed: %w", err)
	}

	m, ok := result.(ConfirmModel)
	if !ok {
		return false, fmt.Errorf("unexpected model type")
	}

	return m.Confirmed(), nil
}
