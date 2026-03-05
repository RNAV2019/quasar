// Package cli implements the cobra command tree for the quasar CLI.
package cli

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/RNAV2019/quasar/internal/config"
	"github.com/RNAV2019/quasar/internal/git"
	"github.com/RNAV2019/quasar/internal/notebook"
	"github.com/spf13/cobra"
)

// OpenNotebookFunc is the callback invoked to open a notebook in the TUI.
var OpenNotebookFunc func(name string)

// NewRootCmd builds and returns the top-level cobra command.
func NewRootCmd(config *config.Config) *cobra.Command {
	cfg = config

	cmd := &cobra.Command{
		Use:   "quasar",
		Short: "A TUI markdown editor with notebook support",
		Long:  "Quasar is a terminal-based markdown editor with LaTeX math support organized around notebooks.",
		Run: func(cmd *cobra.Command, args []string) {
			clearCache, _ := cmd.Flags().GetBool("clear-cache")
			if clearCache {
				if err := clearCacheDir(cfg); err != nil {
					PrintError(fmt.Sprintf("Failed to clear cache: %v", err))
					return
				}
				PrintSuccess("Cache cleared successfully.")
				return
			}

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

	cmd.Flags().BoolP("clear-cache", "", false, "Clear the cache directory")

	cmd.AddCommand(NewNotebookCmd(config))
	cmd.AddCommand(newCompletionCmd())
	cmd.AddCommand(newBackupCmd(config))
	cmd.AddCommand(newSyncCmd(config))

	return cmd
}

func openNotebook(name string) {
	if OpenNotebookFunc != nil {
		OpenNotebookFunc(name)
	}
}

// clearCacheDir recursively removes all files and directories from the cache directory.
func clearCacheDir(cfg *config.Config) error {
	return os.RemoveAll(cfg.CacheDir)
}

// newCompletionCmd returns the completion command for shell autocompletion.
func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for bash, zsh, fish, or PowerShell.

To load completions in your shell, run:

Bash:
  source <(quasar completion bash)

Zsh:
  source <(quasar completion zsh)
  compdef _quasar quasar

Fish:
  quasar completion fish | source

PowerShell:
  quasar completion powershell | Out-String | Invoke-Expression
`,
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletion(os.Stdout)
			}
			return nil
		},
	}
	return cmd
}

// notebookCompletionFunc provides dynamic completion for notebook names.
func notebookCompletionFunc(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if cfg == nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	notebooks, err := notebook.List(cfg.NotesDir)
	if err != nil {
		return []string{}, cobra.ShellCompDirectiveNoFileComp
	}

	var names []string
	for _, nb := range notebooks {
		names = append(names, nb.Name)
	}

	return names, cobra.ShellCompDirectiveNoFileComp
}

// newBackupCmd returns the backup command for committing and pushing notes.
func newBackupCmd(config *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Backup notes to remote repository",
		Long:  "Commit all changes and push them to the remote Git repository.",
		RunE: func(cmd *cobra.Command, args []string) error {
			setRemote, _ := cmd.Flags().GetString("set-remote")
			if setRemote != "" {
				return handleSetRemote(config.NotesDir, setRemote)
			}

			return handleBackup(config.NotesDir)
		},
	}

	cmd.Flags().StringP("set-remote", "r", "", "Set the remote repository URL")

	return cmd
}

// newSyncCmd returns the sync command for pulling the latest notes.
func newSyncCmd(config *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync notes from remote repository",
		Long:  "Pull the latest changes from the remote Git repository.",
		RunE: func(cmd *cobra.Command, args []string) error {
			return handleSync(config.NotesDir)
		},
	}

	return cmd
}

// handleSetRemote sets or updates the remote repository URL.
func handleSetRemote(notesDir string, remoteURL string) error {
	status, err := git.GetStatus(notesDir)
	if err != nil {
		return err
	}

	if !status.IsRepo {
		PrintError("Not a Git repository. Initialize with 'quasar backup' first.")
		return nil
	}

	if status.RemoteURL == "" {
		if err := git.AddRemote(notesDir, "origin", remoteURL); err != nil {
			PrintError(fmt.Sprintf("Failed to add remote: %v", err))
			return nil
		}
		PrintSuccess(fmt.Sprintf("Remote repository set to: %s", remoteURL))
	} else {
		if err := git.SetRemote(notesDir, "origin", remoteURL); err != nil {
			PrintError(fmt.Sprintf("Failed to update remote: %v", err))
			return nil
		}
		PrintSuccess(fmt.Sprintf("Remote repository updated to: %s", remoteURL))
	}

	return nil
}

// handleBackup commits all changes and pushes to the remote.
func handleBackup(notesDir string) error {
	status, err := git.GetStatus(notesDir)
	if err != nil {
		return err
	}

	if !status.IsRepo {
		PrintError("Not a Git repository.")
		return nil
	}

	if !status.HasChanges {
		PrintInfo("No changes to backup.")
		return nil
	}

	fmt.Print("\n📝 Backup Notes\n")
	fmt.Print("─────────────────────────────────────────────\n")

	if status.RemoteURL == "" {
		fmt.Print("No remote repository configured.\n")
		fmt.Print("Enter remote repository URL (press Enter to skip):\n> ")

		remoteURL := readLine()
		remoteURL = strings.TrimSpace(remoteURL)

		if remoteURL != "" {
			if err := git.AddRemote(notesDir, "origin", remoteURL); err != nil {
				PrintError(fmt.Sprintf("Failed to add remote: %v", err))
				return nil
			}
			status.RemoteURL = remoteURL
		} else {
			fmt.Print("\nSkipping push. You can set remote later with:\n")
			fmt.Print("  quasar backup --set-remote <url>\n\n")
		}
	}

	fmt.Print("\nEnter commit message (default: 'Update notes'):\n> ")

	message := readLine()
	message = strings.TrimSpace(message)

	if message == "" {
		message = "Update notes"
	}

	fmt.Print("\n⏳ Backing up notes...\n")

	if err := git.AddAll(notesDir); err != nil {
		PrintError(fmt.Sprintf("Failed to stage changes: %v", err))
		return nil
	}

	if _, err := git.Commit(notesDir, message); err != nil {
		PrintError(fmt.Sprintf("Failed to commit: %v", err))
		return nil
	}

	if status.RemoteURL != "" {
		if _, err := git.Push(notesDir); err != nil {
			PrintError(fmt.Sprintf("Failed to push: %v", err))
			return nil
		}
		PrintSuccess("Notes backed up successfully!")
	} else {
		PrintSuccess("Notes committed locally!")
	}

	return nil
}

// handleSync pulls the latest changes from the remote.
func handleSync(notesDir string) error {
	status, err := git.GetStatus(notesDir)
	if err != nil {
		return err
	}

	if !status.IsRepo {
		PrintError("Not a Git repository.")
		return nil
	}

	if status.RemoteURL == "" {
		PrintError("No remote repository configured.")
		return nil
	}

	fmt.Print("\n🔄 Syncing Notes\n")
	fmt.Print("─────────────────────────────────────────────\n")
	fmt.Print("⏳ Pulling latest changes...\n")

	output, err := git.Pull(notesDir)
	if err != nil {
		PrintError(fmt.Sprintf("Failed to sync: %v", err))
		return nil
	}

	if output != "" && output != "Already up to date.\n" {
		PrintSuccess("Notes synced successfully!")
		fmt.Printf("\n%s\n", output)
	} else {
		PrintInfo("Notes are already up to date.")
	}

	return nil
}

// readLine reads an entire line from stdin including spaces.
func readLine() string {
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSuffix(line, "\n")
}
