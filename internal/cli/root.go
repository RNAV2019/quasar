package cli

import (
	"fmt"

	"github.com/RNAV2019/quasar/internal/config"
	"github.com/RNAV2019/quasar/internal/notebook"
	"github.com/spf13/cobra"
)

var OpenNotebookFunc func(name string)

func NewRootCmd(config *config.Config) *cobra.Command {
	cfg = config

	cmd := &cobra.Command{
		Use:   "quasar",
		Short: "A TUI markdown editor with notebook support",
		Long:  "Quasar is a terminal-based markdown editor with LaTeX math support organized around notebooks.",
		Run: func(cmd *cobra.Command, args []string) {
			defaultNb, err := cfg.GetDefaultNotebook()
			if err != nil {
				PrintError(fmt.Sprintf("Failed to get default notebook: %v", err))
				return
			}

			if defaultNb == "" {
				PrintNoDefaultNotebook()
				return
			}

			if !notebook.Exists(cfg.NotesDir, defaultNb) {
				PrintError(fmt.Sprintf("Default notebook '%s' does not exist.", defaultNb))
				return
			}

			openNotebook(defaultNb)
		},
	}

	cmd.AddCommand(NewNotebookCmd(config))

	return cmd
}

func openNotebook(name string) {
	if OpenNotebookFunc != nil {
		OpenNotebookFunc(name)
	}
}
