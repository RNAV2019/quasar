# Quasar ✦

A terminal-based markdown editor with live LaTeX rendering and notebook organization.

Quasar combines helix/vim-style editing with inline math rendering powered by the Kitty graphics protocol. Organize your notes into notebooks, render LaTeX and TikZ diagrams directly in your terminal, and sync everything with Git.

## Demo

<video src="demo.mp4" controls></video>

## Features

**Editor**
- Helix/Vim style keybindings — normal, insert, visual, and command modes
- Undo/redo, system clipboard integration, and slash commands
- Markdown rendering with syntax highlighting (Catppuccin Mocha theme)
- File tree sidebar for navigating notes

**Math**
- LaTeX block (`$$...$$`) and inline (`$...$`) math rendering
- TikZ diagrams and pgfplots support
- Rendered inline via the Kitty graphics protocol
- Compiled images cached by content hash for instant re-renders

**Notebooks**
- Create, delete, rename, and list notebooks from the CLI
- Tag-based subdirectory organization
- YAML front matter (title, tag, date) on every note

**Git Integration**
- `quasar backup` — commit and push all notes to a remote
- `quasar sync` — pull latest changes from remote

**Customization**
- User-defined math snippets via `~/.config/quasar/snippets.yaml`

## Installation

**Prerequisites:** Go 1.22+, a TeX distribution (`texlive`), a Kitty-protocol compatible terminal (Kitty, WezTerm, Ghostty, etc.)

```bash
# Build from source
git clone https://github.com/RNAV2019/quasar.git
cd quasar
go build -o quasar ./cmd/quasar

# Move to PATH (optional)
sudo mv quasar /usr/local/bin/
```

## Quick Start

```bash
# Create a notebook
quasar nb new physics

# Open it
quasar physics

# Inside the editor, create a note
:new Electromagnetism:Maxwell
```

On first run, Quasar will initialize a Git repo at `~/Documents/quasar/` and generate LaTeX format files in `~/.cache/quasar/`.

## Keybindings

### Normal Mode

| Key | Action |
|-----|--------|
| `h` `j` `k` `l` | Move left/down/up/right |
| `[count]` + motion | Repeat motion N times (e.g., `3j`, `5l`, `2w`) |
| `w` / `b` | Next / previous word |
| `gh` / `gl` | Start / end of line |
| `i` | Enter insert mode |
| `o` | New line below + insert mode |
| `v` | Enter visual mode |
| `d` | Delete character |
| `y` | Yank (copy) line or selection |
| `p` | Paste |
| `u` / `U` | Undo / redo |
| `x` | Select entire line |
| `e` | Select current word |
| `space+f` | Toggle file tree |
| `space+/` | Focus file tree |
| `:` | Enter command mode |

### Insert Mode

| Key | Action |
|-----|--------|
| `/` | Open slash command menu |
| `Tab` / `Shift+Tab` | Navigate autocomplete |
| `Esc` | Return to normal mode |

### Command Mode

| Command | Action |
|---------|--------|
| `:w` | Save |
| `:q` | Quit |
| `:wq` | Save and quit |
| `:q!` | Force quit |
| `:new Name:Tag` | Create note (tag optional) |
| `:<number>` | Go to line (e.g., `:42`) |
| `:delete` | Delete current note |
| `:h` | Show help |

See `:h` inside the editor for the full list.

## Configuration

```
~/.config/quasar/
└── snippets.yaml      # User-defined math snippets

~/.cache/quasar/
├── notebooks.yaml     # Notebook registry
├── *.png              # Cached rendered math
└── *.fmt              # Precompiled LaTeX formats

~/Documents/quasar/    # All notebooks (Git repo)
```

### Custom Snippets

Add math snippets that appear in the `/` autocomplete menu:

```yaml
snippets:
  - trigger: matrix
    label: "Matrix"
    body: |
      \begin{bmatrix}
        $0
      \end{bmatrix}
    cursor: "$0"
```

## Tech Stack

[Bubble Tea](https://github.com/charmbracelet/bubbletea) · [Lipgloss](https://github.com/charmbracelet/lipgloss) · [Glamour](https://github.com/charmbracelet/glamour) · [Cobra](https://github.com/spf13/cobra) · [Kitty Graphics Protocol](https://sw.kovidgoyal.net/kitty/graphics-protocol/)
