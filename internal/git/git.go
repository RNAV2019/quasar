// Package git provides Git operations for backing up and syncing notes.
package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Status checks the current Git status.
type Status struct {
	IsRepo        bool
	HasChanges    bool
	IsDirty       bool
	CurrentBranch string
	RemoteURL     string
}

// GetStatus returns the current Git status of a directory.
func GetStatus(dir string) (*Status, error) {
	status := &Status{}

	// Check if it's a git repo
	if _, err := os.Stat(fmt.Sprintf("%s/.git", dir)); err != nil {
		return status, nil
	}

	status.IsRepo = true

	// Get current branch
	branch, err := runGit(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err == nil {
		status.CurrentBranch = strings.TrimSpace(branch)
	}

	// Get remote URL
	remoteURL, err := runGit(dir, "config", "--get", "remote.origin.url")
	if err == nil {
		status.RemoteURL = strings.TrimSpace(remoteURL)
	}

	// Check for changes
	output, _ := runGit(dir, "status", "--porcelain")
	status.HasChanges = output != ""
	status.IsDirty = status.HasChanges

	return status, nil
}

// Init initializes a new Git repository.
func Init(dir string) error {
	_, err := runGit(dir, "init")
	if err != nil {
		return fmt.Errorf("failed to initialize git repo: %w", err)
	}
	return nil
}

// AddRemote adds a remote repository.
func AddRemote(dir, name, url string) error {
	_, err := runGit(dir, "remote", "add", name, url)
	if err != nil {
		return fmt.Errorf("failed to add remote: %w", err)
	}
	return nil
}

// SetRemote updates a remote repository URL.
func SetRemote(dir, name, url string) error {
	_, err := runGit(dir, "remote", "set-url", name, url)
	if err != nil {
		return fmt.Errorf("failed to set remote: %w", err)
	}
	return nil
}

// Clone clones a repository into the directory.
func Clone(repoURL, targetDir string) error {
	cmd := exec.Command("git", "clone", repoURL, targetDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to clone repository: %w\n%s", err, string(output))
	}
	return nil
}

// Pull pulls latest changes from the remote.
func Pull(dir string) (string, error) {
	output, err := runGit(dir, "pull", "origin", "main")
	if err != nil {
		// Try master if main doesn't exist
		output, err = runGit(dir, "pull", "origin", "master")
		if err != nil {
			return "", fmt.Errorf("failed to pull: %w", err)
		}
	}
	return output, nil
}

// AddAll stages all changes.
func AddAll(dir string) error {
	_, err := runGit(dir, "add", "-A")
	if err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}
	return nil
}

// Commit commits staged changes with a message.
func Commit(dir, message string) (string, error) {
	output, err := runGit(dir, "commit", "-m", message)
	if err != nil {
		return "", fmt.Errorf("failed to commit: %w", err)
	}
	return output, nil
}

// Push pushes commits to the remote.
func Push(dir string) (string, error) {
	// Determine the default branch
	branch := "main"
	currentBranch, err := runGit(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if err == nil {
		branch = strings.TrimSpace(currentBranch)
	}

	output, err := runGit(dir, "push", "-u", "origin", branch)
	if err != nil {
		return "", fmt.Errorf("failed to push: %w", err)
	}
	return output, nil
}

// CommitAndPush commits all changes and pushes to remote.
func CommitAndPush(dir, message string) error {
	if err := AddAll(dir); err != nil {
		return err
	}

	_, err := Commit(dir, message)
	if err != nil {
		return err
	}

	_, err = Push(dir)
	if err != nil {
		return err
	}

	return nil
}

// runGit executes a git command in the specified directory.
func runGit(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}

	return string(output), nil
}
