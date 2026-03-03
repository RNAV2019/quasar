package tui

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type PromptModel struct {
	textInput   textinput.Model
	promptLabel string
	submitted   bool
	cancelled   bool
	width       int
}

func NewPromptModel(label string, placeholder string) PromptModel {
	ti := textinput.New()
	ti.Placeholder = placeholder
	ti.Focus()

	return PromptModel{
		textInput:   ti,
		promptLabel: label,
		width:       50,
	}
}

func (m PromptModel) Init() tea.Cmd {
	return nil
}

func (m PromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			m.submitted = true
			return m, tea.Quit
		case "esc", "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m PromptModel) View() tea.View {
	var b strings.Builder

	b.WriteString(TitleStyle.Render(m.promptLabel))
	b.WriteString("\n\n")

	inputView := m.textInput.View()
	b.WriteString(inputView)
	b.WriteString("\n\n")

	hintStyle := lipgloss.NewStyle().Foreground(DimColor)
	b.WriteString(hintStyle.Render("enter: confirm  |  esc: cancel"))

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

func (m PromptModel) Value() string {
	return m.textInput.Value()
}

func (m PromptModel) Submitted() bool {
	return m.submitted
}

func (m PromptModel) Cancelled() bool {
	return m.cancelled
}

func RunPrompt(label string, placeholder string) (string, error) {
	model := NewPromptModel(label, placeholder)
	p := tea.NewProgram(model, tea.WithoutSignalHandler())
	result, err := p.Run()
	if err != nil {
		return "", fmt.Errorf("prompt failed: %w", err)
	}

	m, ok := result.(PromptModel)
	if !ok {
		return "", fmt.Errorf("unexpected model type")
	}

	if m.Cancelled() || !m.Submitted() {
		return "", fmt.Errorf("cancelled")
	}

	return m.Value(), nil
}
