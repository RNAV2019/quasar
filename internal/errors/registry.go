// Package errors provides a thread-safe global error registry for collecting
// and displaying application errors.
package errors

import (
	"sync"
	"time"
)

// Error represents a single error in the application.
type Error struct {
	Message   string
	Source    string    // e.g., "latex", "file", "editor"
	Timestamp time.Time
}

// Registry is a thread-safe error registry.
type Registry struct {
	mu     sync.RWMutex
	errors []Error
}

// Global is the default application-wide error registry.
var Global = &Registry{}

// Add adds an error to the registry.
func (r *Registry) Add(message, source string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errors = append(r.errors, Error{
		Message:   message,
		Source:    source,
		Timestamp: time.Now(),
	})
}

// Clear removes all errors.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.errors = nil
}

// Count returns the number of errors.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.errors)
}

// GetAll returns a copy of all errors.
func (r *Registry) GetAll() []Error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]Error, len(r.errors))
	copy(result, r.errors)
	return result
}

// AddError is a convenience function to add to the global registry.
func AddError(message, source string) {
	Global.Add(message, source)
}

// ClearErrors clears the global registry.
func ClearErrors() {
	Global.Clear()
}

// ErrorCount returns the count from the global registry.
func ErrorCount() int {
	return Global.Count()
}

// GetErrors returns all errors from the global registry.
func GetErrors() []Error {
	return Global.GetAll()
}
