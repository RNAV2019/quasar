package cli

import (
	"fmt"
	"strings"

	"github.com/RNAV2019/quasar/internal/styles"
)

// PrintError prints a styled error message to stdout.
func PrintError(msg string) {
	fmt.Println(styles.ErrorStyle.Render(msg))
}

// PrintErrorWithHints prints a styled error message followed by hint lines.
func PrintErrorWithHints(msg string, hints []string) {
	var fullMsg strings.Builder
	fullMsg.WriteString(msg)
	fullMsg.WriteString("\n\n")
	for _, hint := range hints {
		fullMsg.WriteString("  ")
		fullMsg.WriteString(hint)
		fullMsg.WriteString("\n")
	}
	fmt.Println(styles.ErrorStyle.Render(fullMsg.String()))
}

// PrintSuccess prints a styled success message to stdout.
func PrintSuccess(msg string) {
	fmt.Println(styles.SuccessStyle.Render(msg))
}

// PrintInfo prints a styled informational message to stdout.
func PrintInfo(msg string) {
	fmt.Println(styles.InfoStyle.Render(msg))
}

// PrintNoDefaultNotebook prints guidance when no default notebook is configured.
func PrintNoDefaultNotebook() {
	msg := "No default notebook set.\n\n" +
		"Create one with: quasar nb new\n" +
		"Or set existing: quasar nb default <NAME>"
	PrintError(msg)
}

// PrintNoNotebooks prints guidance when no notebooks exist.
func PrintNoNotebooks() {
	msg := "No notebooks found.\n\n" +
		"Create one with: quasar nb new"
	PrintInfo(msg)
}

// PrintNotebookNotFound prints an error that the named notebook was not found.
func PrintNotebookNotFound(name string) {
	PrintError(fmt.Sprintf("Notebook '%s' not found.", name))
}

// PrintNotebookCreated prints a success message after creating a notebook.
func PrintNotebookCreated(name string) {
	PrintSuccess(fmt.Sprintf("Notebook '%s' created.", name))
}

// PrintNotebookDeleted prints a success message after deleting a notebook.
func PrintNotebookDeleted(name string) {
	PrintSuccess(fmt.Sprintf("Notebook '%s' deleted.", name))
}

// PrintNotebookRenamed prints a success message after renaming a notebook.
func PrintNotebookRenamed(oldName, newName string) {
	PrintSuccess(fmt.Sprintf("Notebook '%s' renamed to '%s'.", oldName, newName))
}

// PrintDefaultNotebookSet prints a success message after setting the default notebook.
func PrintDefaultNotebookSet(name string) {
	PrintSuccess(fmt.Sprintf("Default notebook set to '%s'.", name))
}
