package cli

import (
	"fmt"

	"github.com/RNAV2019/quasar/internal/cli/tui"
	"github.com/RNAV2019/quasar/internal/config"
	"github.com/RNAV2019/quasar/internal/notebook"
	"github.com/RNAV2019/quasar/internal/styles"
	"github.com/spf13/cobra"
)

var cfg *config.Config

// NewNotebookCmd builds the "nb" subcommand and its children.
func NewNotebookCmd(config *config.Config) *cobra.Command {
	cfg = config

	cmd := &cobra.Command{
		Use:   "nb [name]",
		Short: "Manage notebooks",
		Long:  "Create, open, delete, and manage notebooks for organizing notes.",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				name := args[0]
				if !notebook.Exists(cfg.NotesDir, name) {
					PrintNotebookNotFound(name)
					return
				}
				openNotebook(name)
				return
			}
			cmd.Help()
		},
		ValidArgsFunction: notebookCompletionFunc,
	}

	cmd.AddCommand(newNbNewCmd())
	cmd.AddCommand(newNbDefaultCmd())
	cmd.AddCommand(newNbDeleteCmd())
	cmd.AddCommand(newNbRenameCmd())
	cmd.AddCommand(newNbListCmd())

	return cmd
}

func newNbNewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "new",
		Short: "Create a new notebook",
		Long:  "Create a new notebook with the specified name. Prompts for name if not provided.",
		Run: func(cmd *cobra.Command, args []string) {
			var name string
			var err error

			if len(args) > 0 {
				name = args[0]
			} else {
				name, err = tui.RunPrompt("Notebook Name:", "my-notebook")
				if err != nil {
					return
				}
			}

			if name == "" {
				PrintError("Notebook name cannot be empty.")
				return
			}

			if err := notebook.Create(cfg.NotesDir, name); err != nil {
				PrintError(err.Error())
				return
			}

			PrintNotebookCreated(name)
		},
	}
}

func newNbDefaultCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "default <name>",
		Short: "Set the default notebook",
		Long:  "Set the specified notebook as the default notebook to open when running 'quasar' without arguments.",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: notebookCompletionFunc,
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			if !notebook.Exists(cfg.NotesDir, name) {
				PrintNotebookNotFound(name)
				return
			}

			if err := cfg.SetDefaultNotebook(name); err != nil {
				PrintError(fmt.Sprintf("Failed to set default notebook: %v", err))
				return
			}

			PrintDefaultNotebookSet(name)
		},
	}
}

func newNbDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a notebook",
		Long:  "Delete the specified notebook after confirmation.",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: notebookCompletionFunc,
		Run: func(cmd *cobra.Command, args []string) {
			name := args[0]

			if !notebook.Exists(cfg.NotesDir, name) {
				PrintNotebookNotFound(name)
				return
			}

			confirmed, err := tui.RunConfirm(fmt.Sprintf("Delete notebook '%s'?", name))
			if err != nil || !confirmed {
				PrintInfo("Cancelled.")
				return
			}

			if err := notebook.Delete(cfg.NotesDir, name); err != nil {
				PrintError(err.Error())
				return
			}

			PrintNotebookDeleted(name)
		},
	}
}

func newNbRenameCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rename <name>",
		Short: "Rename a notebook",
		Long:  "Rename the specified notebook. Prompts for the new name.",
		Args:  cobra.ExactArgs(1),
		ValidArgsFunction: notebookCompletionFunc,
		Run: func(cmd *cobra.Command, args []string) {
			oldName := args[0]

			if !notebook.Exists(cfg.NotesDir, oldName) {
				PrintNotebookNotFound(oldName)
				return
			}

			newName, err := tui.RunPrompt("New Notebook Name:", oldName)
			if err != nil {
				return
			}

			if newName == "" {
				PrintError("Notebook name cannot be empty.")
				return
			}

			if err := notebook.Rename(cfg.NotesDir, oldName, newName); err != nil {
				PrintError(err.Error())
				return
			}

			PrintNotebookRenamed(oldName, newName)
		},
	}
}

func newNbListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all notebooks",
		Long:  "List all notebooks in the notes directory.",
		Run: func(cmd *cobra.Command, args []string) {
			notebooks, err := notebook.List(cfg.NotesDir)
			if err != nil {
				PrintError(err.Error())
				return
			}

			if len(notebooks) == 0 {
				PrintNoNotebooks()
				return
			}

			defaultNb, _ := cfg.GetDefaultNotebook()

			fmt.Println()
			for _, nb := range notebooks {
				if nb.Name == defaultNb {
					fmt.Printf("  %s %s\n", styles.BoldStyle.Render("*"), nb.Name)
				} else {
					fmt.Printf("    %s\n", nb.Name)
				}
			}
			fmt.Println()
		},
	}
}
