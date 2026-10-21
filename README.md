<p align="center">
  <img src="assets/go-tui.png" alt="GO-TUI" width="800">
</p>

> **Modern, standardized Terminal User Interface (TUI) starter templates and reference implementation in Go ([Bubbletea](https://github.com/charmbracelet/bubbletea) + [Lipgloss](https://github.com/charmbracelet/lipgloss)).**

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-blue?logo=go)](https://golang.org)
[![Bubbletea](https://img.shields.io/badge/TUI-Bubbletea-teal)](https://github.com/charmbracelet/bubbletea)
[![Lipgloss](https://img.shields.io/badge/Styling-Lipgloss-pink)](https://github.com/charmbracelet/lipgloss)
[![License](https://img.shields.io/badge/License-MIT-green)](LICENSE)

GO-TUI provides a production-grade, architectural foundation for terminal tools. It ships with a complete, fully functional demo application (`tui-example`), a reproducible starter template, and a strict terminal design grammar.

---

## Quick Start: Try the Included Demo

A complete, runnable reference implementation is included out-of-the-box in `tui-example/`:

```bash
# Clone the repository
git clone https://github.com/tdwgm/GO-TUI.git
cd GO-TUI

# Build and run the example TUI
make build
make run

# Run the unit test suite headlessly:
make test
```

### CLI Subcommands Out-of-the-Box

The binary operates as both an interactive TUI and a scriptable CLI:

```bash
./tui-example/bin/tui-example            # Launch the interactive TUI
./tui-example/bin/tui-example doctor     # Self-diagnostic checklist (config, logs, install, aliases)
./tui-example/bin/tui-example install    # Install to ~/.local/bin + configure shell aliases
./tui-example/bin/tui-example uninstall  # Cleanly remove binary and aliases
./tui-example/bin/tui-example version    # Show banner and version information
./tui-example/bin/tui-example help       # CLI help and options
```

---

## The TUI Standard v2

GO-TUI implements the **TUI Scaffold v2 standard**:

### 1. Distinctive Header & Live Info Block
* **Banner Art & Fonts**: Multi-style ASCII fonts (`default`, `block`, `half`, `slant`, `shade`) rendered above a Fraktur byline.
* **Single Rule Grammar**: Exactly one dotted line (`······`) spans the terminal; all body text is neatly indented by 2 spaces.
* **Live Info Block**: Positioned immediately below the banner rule. It renders **only when live data exists** using Nerd Font MDI icons without clutter:
  ```
  󰖂 127.0.0.1   󰄬 7 items   󰍉 Ready
  ```

### 2. Streamlined One-Line Bottom Menu & Glitter Sweep
* **Menu Bar**: Clean, unified bottom navigation:
  ```
  ▶ Delete x · Config o · Log l · Quit q · Help h
  ```
* **Glitter Animation (`MENU_EFFECT=glitter`)**: A subtle glint animation sweeps across the active button label when the menu is focused.
* **Predictable Navigation**:
  * `Tab` / `↓` moves focus into the bottom menu.
  * `←` / `→` cycles through actions.
  * `Tab` / `↑` returns to the content body.
  * `q` and `Esc` are strictly symmetric: navigate back one screen, or quit from the Home screen.

### 3. Config Screen (1:1 with File Synchronization)
* Every field visible on the Config screen is a real key in `~/.config/<app>/<app>.conf`.
* Saving from the TUI preserves hand-written comments, blank lines, and custom ordering.
* **Theme per Element**:
  * Accent colour (`THEME`: `blue`, `ocean`, `green`, `red`, `yellow`, `magenta`, `cyan`, `orange`, `purple`)
  * Banner, Header, Text, Muted, Shortcuts, Rule, Subtitle, and Live Info colours
  * Banner typography & live info toggles
  * **[Reset to default]** row to quickly restore the standard palette.

### 4. Built-in Observability & Help
* **Doctor (`^d` / `Run doctor`)**: Interactive diagnostic screen verifying config file validity, installation paths, aliases, and dependencies.
* **Log Viewer (`^l` / `Log`)**: Integrated tail viewer with instant Refresh (`r`) and Clear (`c`).
* **Contextual Help (`h`)**: Automatically generated index merging all keyboard shortcuts across every screen, plus Nerd Font icon explanations.

---

## Global Shortcuts

| Key | Action | Scope |
| :--- | :--- | :--- |
| `Tab` / `Shift+Tab` | Toggle focus between content fields and bottom menu | Everywhere |
| `←` / `→` | Navigate menu buttons | Bottom menu |
| `↑` / `↓` | Navigate list items or fields | Body |
| `q` / `Esc` | Back / Cancel (Quits at root) | Everywhere |
| `Ctrl+C` | Force quit | Everywhere |
| `h` | Contextual Help screen | Everywhere |
| `o` | Configuration screen | Home |
| `^d` | Run Doctor diagnostics | Config / Global |
| `^l` | Open Log viewer | Home / Global |
| `^r` | Reload config from disk | Config |
| `^s` | Save config changes | Config |
| `^t` | Reset theme to default | Config / Global |

---

## Repository Structure

```
├── tui-example/              # Full runnable reference implementation (Go)
│   ├── cmd/tui-example/      # Application entrypoint
│   └── internal/tui/         # Pure model, layout, style, banner, config, and views
├── go/                       # Starter template files for new Go projects
│   ├── cmd/{{PROJECT_NAME}}/ # Templated main.go
│   ├── internal/tui/         # Core TUI framework modules
│   └── go.mod.tmpl           # Go module template
├── python/                   # Companion Python starter template (Textual + uv)
├── bash/                     # Companion Bash 4+ starter template
├── Makefile                  # Build, test, and run helpers
├── setup.sh                  # Host dependency validator
└── README.md
```

---

## Headless Testing

GO-TUI follows the **Elm architecture** (`Init`, `Update`, `View`). All screens, navigation transitions, theme switches, and layout constraints are tested without launching a virtual terminal:

```bash
cd tui-example
go test -v ./...
```

---

## License

MIT License. See [LICENSE](LICENSE) for details.
