# Guide for New Agents

This document provides essential information for new agents to effectively work in the quasar repository.

## Project Overview

Quasar is a Go project that processes mathematical expressions and renders them as text. The main entry point is `cmd/main.go`, while core functionality resides in the `internal` package.

## Code Organization

- **main.go**: Command-line interface
- **internal/config**: Application configuration management
- **internal/latex**: LaTeX parsing and processing logic
- **internal/editor**: Document editing functions
- **internal/ui**: Text formatting and rendering operations

## Directory Structure

```
.
├── cmd/main.go              # Command line entry point
├── internal/
│   ├── config/setup.go     # Configuration initialization
│   ├── latex/               # Equation processing
│   │   ├── compiler.go      # Translate LaTeX to text
│   │   └── kitty.go         # Kitty font fallbacks?
│   ├── editor/editor.go    # Document editing logic
│   └── ui/
│       ├── model.go        # Rendering data structures
│       ├── styles.go       # Styling information
│       └── view.go         # Output generation and formatting
```

## Build & Run Steps

To build the project:
```bash
go build ./...
```

### Testing Guidelines
- Currently no automated test suite set up for testing.

## Code Conventions

All file and code should follow standard Go naming conventions.

This guide will be updated as more information becomes available.