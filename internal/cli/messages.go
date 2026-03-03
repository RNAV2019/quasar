package cli

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F38BA8")).
			Padding(1, 2)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#A6E3A1")).
			Padding(1, 2)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#89B4FA")).
			Padding(1, 2)

	boldStyle = lipgloss.NewStyle().Bold(true)
)

func PrintError(msg string) {
	fmt.Println(errorStyle.Render(msg))
}

func PrintErrorWithHints(msg string, hints []string) {
	var fullMsg strings.Builder
	fullMsg.WriteString(msg)
	fullMsg.WriteString("\n\n")
	for _, hint := range hints {
		fullMsg.WriteString("  ")
		fullMsg.WriteString(hint)
		fullMsg.WriteString("\n")
	}
	fmt.Println(errorStyle.Render(fullMsg.String()))
}

func PrintSuccess(msg string) {
	fmt.Println(successStyle.Render(msg))
}

func PrintInfo(msg string) {
	fmt.Println(infoStyle.Render(msg))
}

func PrintNoDefaultNotebook() {
	msg := "No default notebook set.\n\n" +
		"Create one with: quasar nb new\n" +
		"Or set existing: quasar nb default <NAME>"
	PrintError(msg)
}

func PrintNoNotebooks() {
	msg := "No notebooks found.\n\n" +
		"Create one with: quasar nb new"
	PrintInfo(msg)
}

func PrintNotebookNotFound(name string) {
	PrintError(fmt.Sprintf("Notebook '%s' not found.", name))
}

func PrintNotebookCreated(name string) {
	PrintSuccess(fmt.Sprintf("Notebook '%s' created.", name))
}

func PrintNotebookDeleted(name string) {
	PrintSuccess(fmt.Sprintf("Notebook '%s' deleted.", name))
}

func PrintNotebookRenamed(oldName, newName string) {
	PrintSuccess(fmt.Sprintf("Notebook '%s' renamed to '%s'.", oldName, newName))
}

func PrintDefaultNotebookSet(name string) {
	PrintSuccess(fmt.Sprintf("Default notebook set to '%s'.", name))
}
