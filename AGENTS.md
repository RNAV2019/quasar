# AGENTS.md

> **Quasar** is a small note-taking and math rendering tool built in Go using the Bubble Tea TUI framework.  The repository contains a single executable in `cmd/main.go` that pulls in a handful of internal packages.

## How to run locally

1.  Ensure Go 1.26+ is installed and `go` is on your `$PATH`.
2.  From the repository root run:

```
go run ./cmd/main.go
```

This will build the binary in‑memory and launch the interactive TUI.

## Build for distribution

```bash
# build a binary for the current OS/arch
go build -o quasar ./cmd/main.go
```

The produced binary can be distributed or installed with `go install ./cmd/...`.

## Test / Lint

There are no unit tests bundled with the repo, but static analysis can be run via:

```bash
# lint the whole project
golangci-lint run ./...
```

(If `golangci-lint` is not installed, you can just `go vet ./...` or `go test ./...` which will surface any compile‑time issues.)

## Directory layout

```
internal/
├── config/      # SetupEnvironment helper creating user‑specific cache/notes dirs
├── editor/      # TUI editor model – see editor/editor.go
├── latex/       # helpers for rendering LaTeX expressions via charm.land/bubbletea
└── ui/          # Bubble Tea view logic

cmd/
└── main.go      # entry point

go.mod

```

The `internal` packages are not intended for external use.

## Key files

- **cmd/main.go** – bootstraps config, creates a tea program with `ui.InitialModel` and runs it.
- **internal/editor/editor.go** – contains the `Model` struct that represents the current document in terms of *blocks* (text or math).  The model tracks cursor position, dirty flags, and rendering state.
- **internal/ui/view.go** – Bubble Tea view rendering logic (not shown here).
- **internal/config/setup.go** – helper that creates a `.cache/quasar` and `Documents/quasar` folder in the current user’s home.

## Common gotchas

- The project uses the **charm.land/bubbletea** library for terminal graphics.  It expects the host terminal to support Kitty's graphics protocol for rendering math.  If you run the binary in a plain `xterm` or a Windows console you will *not* see the math blocks.
- The editor model splits a line2011memory and launch the interactive TUI.

## Build for distribution

```bash
# build a binary for the current OS/arch
go build -o quasar ./cmd/main.go
```

The produced binary can be distributed or installed with `go install ./cmd/...`.

## Test / Lint

There are no unit tests bundled with the repo, but static analysis can be run via:

```bash
# lint the whole project
golangci-lint run ./...
```

(If `golangci-lint` is not installed, you can just `go vet ./...` or `go test ./...` which will surface any compile‑time issues.)

## Directory layout

```
internal/
├── config/      # SetupEnvironment helper creating user‑specific cache/notes dirs
├── editor/      # TUI editor model – see editor/editor.go
├── latex/       # helpers for rendering LaTeX expressions via charm.land/bubbletea
└── ui/          # Bubble Tea view logic

cmd/
└── main.go      # entry point

go.mod

```

The `internal` packages are not intended for external use.

## Key files

- **cmd/main.go** – bootstraps config, creates a tea program with `ui.InitialModel` and runs it.
- **internal/editor/editor.go** – contains the `Model` struct that represents the current document in terms of *blocks* (text or math).  The model tracks cursor position, dirty flags, and rendering state.
- **internal/ui/view.go** – Bubble Tea view rendering logic (not shown here).
- **internal/config/setup.go** – helper that creates a `.cache/quasar` and `Documents/quasar` folder in the current user’s home.

## Common gotchas

- The project uses the **charm.land/bubbletea** library for terminal graphics.  It expects the host terminal to support Kitty's graphics protocol for rendering math.  If you run the binary in a plain `xterm` or a Windows console you will *not* see the math blocks.
- The editor model splits a line into *text* and *math* blocks when the user types `$` inside a text block.  The logic is in `editor/editor.go:splitBlockForMath`.  It expects the cursor to be positioned on the second `$` of a `$$…$$` pair;  otherwise the split will be a no‑op.
- There are currently no tests;  if you add tests keep them in a `*_test.go` file under the same package.

## Useful commands for developers

- `go run ./cmd/main.go` – run the app.
- `go build -o quasar ./cmd/main.go` – build a static binary.
- `go test ./...` – run any future tests.
- `golangci-lint run ./...` – lint.

---

> This file is generated to aid future agents working in this repository.  It covers the essential commands, layout, and idiosyncratic patterns used by the code.
