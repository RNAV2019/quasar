// Package notebook provides CRUD operations for notebook directories and notes.
package notebook

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Notebook represents a named notebook directory.
type Notebook struct {
	Name string
	Path string
}

// List returns all notebooks found in notesDir.
func List(notesDir string) ([]Notebook, error) {
	entries, err := os.ReadDir(notesDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read notes directory: %w", err)
	}

	var notebooks []Notebook
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			notebooks = append(notebooks, Notebook{
				Name: entry.Name(),
				Path: filepath.Join(notesDir, entry.Name()),
			})
		}
	}

	return notebooks, nil
}

// Create creates a new notebook directory with the given name.
func Create(notesDir, name string) error {
	if name == "" {
		return fmt.Errorf("notebook name cannot be empty")
	}

	notebookPath := filepath.Join(notesDir, name)

	if info, err := os.Stat(notebookPath); err == nil {
		if info.IsDir() {
			return fmt.Errorf("notebook '%s' already exists", name)
		}
		return fmt.Errorf("a file named '%s' already exists", name)
	}

	if err := os.MkdirAll(notebookPath, 0755); err != nil {
		return fmt.Errorf("failed to create notebook: %w", err)
	}

	return nil
}

// Delete removes the named notebook directory and all its contents.
func Delete(notesDir, name string) error {
	if name == "" {
		return fmt.Errorf("notebook name cannot be empty")
	}

	notebookPath := filepath.Join(notesDir, name)

	if info, err := os.Stat(notebookPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("notebook '%s' does not exist", name)
		}
		return fmt.Errorf("failed to access notebook: %w", err)
	} else if !info.IsDir() {
		return fmt.Errorf("'%s' is not a directory", name)
	}

	if err := os.RemoveAll(notebookPath); err != nil {
		return fmt.Errorf("failed to delete notebook: %w", err)
	}

	return nil
}

// Rename renames a notebook directory from oldName to newName.
func Rename(notesDir, oldName, newName string) error {
	if oldName == "" || newName == "" {
		return fmt.Errorf("notebook names cannot be empty")
	}

	oldPath := filepath.Join(notesDir, oldName)
	newPath := filepath.Join(notesDir, newName)

	if info, err := os.Stat(oldPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("notebook '%s' does not exist", oldName)
		}
		return fmt.Errorf("failed to access notebook: %w", err)
	} else if !info.IsDir() {
		return fmt.Errorf("'%s' is not a directory", oldName)
	}

	if info, err := os.Stat(newPath); err == nil {
		if info.IsDir() {
			return fmt.Errorf("notebook '%s' already exists", newName)
		}
		return fmt.Errorf("a file named '%s' already exists", newName)
	}

	if err := os.Rename(oldPath, newPath); err != nil {
		return fmt.Errorf("failed to rename notebook: %w", err)
	}

	return nil
}

// Exists reports whether a notebook with the given name exists.
func Exists(notesDir, name string) bool {
	info, err := os.Stat(filepath.Join(notesDir, name))
	if err != nil {
		return false
	}
	return info.IsDir()
}

// Path returns the full filesystem path for the named notebook.
func Path(notesDir, name string) string {
	return filepath.Join(notesDir, name)
}
