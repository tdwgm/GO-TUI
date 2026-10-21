# tui-example

> **Reference implementation of the TUI Scaffold v2 standard in Go (Bubbletea + Lipgloss)**

`tui-example` is a complete, production-ready terminal user interface application built using [Charmbracelet Bubbletea](https://github.com/charmbracelet/bubbletea) and [Lipgloss](https://github.com/charmbracelet/lipgloss). It demonstrates every rule, screen, and feature of the **TUI Scaffold v2 standard**.

---

## ⚡ Quick Start

### Build and Run

```bash
# Build the binary into bin/tui-example
make build

# Run interactively
./bin/tui-example

# Or during development:
make run

# Run headless tests:
make test
```

### CLI Subcommands

`tui-example` works both as an interactive TUI (default) and as a standard CLI tool:

```bash
./bin/tui-example tui        # Launch the interactive UI (default when no args)
./bin/tui-example doctor     # Check configuration, log paths, shell aliases, and binary installation
./bin/tui-example install    # Copy binary to ~/.local/bin, write aliases, update config
./bin/tui-example uninstall  # Remove binary and shell alias (keeps user config)
./bin/tui-example version    # Print banner and version details
./bin/tui-example help       # Show CLI help and documentation
```

---

## ✨ Features & Architecture

### 1. Banner & Typsnitt
* **ASCII Art & Fonts**: Renders the application banner with version number (`v0.1.0`) directly following the logo, description, and the distinctive Fraktur byline:
  ```
  𝖇𝖞 𝕿𝖉𝖜𝖌 𝕰𝖓𝖙𝖊𝖗𝖙𝖆𝖎𝖓𝖒𝖊𝖓𝖙 𝕾𝖔𝖑𝖚𝖙𝖎𝖔𝖓𝖘 ®
  ```
* **Custom Banner Fonts**: Supports selectable font formats (`default`, `block`, `half`, `slant`, `shade`) via `BANNER_FONT` and configurable custom banner text via `BANNER_TEXT`.
* **Single Rule**: Exactly one dotted rule (`······`) under the banner; starts at column 0 while all body text starts indented 2 spaces.

### 2. Live Info Block
* **Directly Under Banner**: Printed only when live data exists (in this demo: when items exist).
* **Nerd Font MDI Icons**: Uses clean icons without noisy labels (explained in **Help → Icons**):
  ```
  󰖂 127.0.0.1   󰄬 7 items   󰍉 Ready
  ```
* **Light Styling**: Values highlighted in `COLOR_LIVE`, secondary counters in `COLOR_SUBTITLE`, separated by three spaces.

### 3. One-Line Bottom Menu & Glitter Effect
* **Grammar**: Exactly one line at the bottom:
  ```
  ▶ Delete x · Config o · Log l · Quit q · Help h
  ```
* **Glitter Sweep (`MENU_EFFECT=glitter`)**: When the menu is focused, an elegant glint sweeps across the active label (~90 ms per step).
* **Navigation**: Press **Tab** or **↓** to enter the menu; **← / →** moves between buttons; **Tab** or **↑** returns to the content.
* **Symmetry**: `q` and `Esc` behave identically on every screen (back one level, quits at root).

### 4. Config Screen (1:1 with Config File)
* **Roundtrip Configuration**: Every setting on the Config screen maps 1:1 to `~/.config/tui-example/tui-example.conf`.
* **Comment Preservation**: Hand-written comments, blank lines, and ordering in the config file are preserved when saving from the TUI.
* **Element-Level Theme Customization**:
  * Accent (`THEME`: `blue`, `green`, `red`, `yellow`, `magenta`, `cyan`, `orange`, `purple`)
  * `COLOR_BANNER`, `COLOR_HEADER`, `COLOR_TEXT`, `COLOR_MUTED`
  * `COLOR_SHORTCUT`, `COLOR_SELECTED`, `COLOR_RULE`, `COLOR_SUBTITLE`, `COLOR_LIVE`
  * Banner settings (`BANNER_TEXT`, `BANNER_FONT`, `LIVE_INFO`)
  * **[Reset to default]**: Restores all theme and banner options to the standard palette.

### 5. Built-in Doctor & Log Viewer
* **Doctor (`^d` / `Run doctor`)**: Runs real-time diagnostics verifying configuration files, binary install status, log paths, and shell aliases.
* **Log Viewer (`^l` / `Log`)**: Integrated tail viewer showing the latest log entries (`~/.local/state/tui-example/tui-example.log`) with Refresh (`r`) and Clear (`c`).

### 6. Generated Help Screen (`h`)
* Generates all help sections automatically from button tables and config keys:
  * **About**: Version, config file path, log path
  * **Icons**: Detailed explanation of all Nerd Font MDI glyphs used in the Live Info block
  * **Shortcuts**: Unified list of all shortcuts merged across every screen
  * **Commands**: CLI subcommand usage
  * **Application info**: Shell aliases and integration guide

---

## ⌨️ Shortcuts Reference

| Key | Action | Scope |
| :--- | :--- | :--- |
| `Tab` / `Shift+Tab` | Toggle focus between content fields and bottom menu | Everywhere |
| `←` / `→` | Navigate buttons in bottom menu | Bottom menu |
| `↑` / `↓` | Navigate list items or fields | Screen body |
| `q` / `Esc` | Back / Cancel (Quits on Home screen) | Everywhere |
| `Ctrl+C` | Force quit immediately | Everywhere |
| `h` | Open Help screen | Everywhere |
| `o` | Open Config screen | Home |
| `l` / `^l` | Open Log viewer | Home / Global |
| `^d` | Run Doctor diagnostics | Config / Global |
| `^r` | Reload config from disk | Config |
| `^t` | Reset theme to default (with confirmation) | Config / Global |
| `^s` | Save config changes | Config |

---

## 📁 Project Layout

```
cmd/tui-example/
  main.go                 # Application entry point; sets AppName and AppDescription

internal/tui/
  app.go                  # Main Bubbletea model, update loop, screen routing, shortcuts
  app_test.go             # Comprehensive headless test suite (13 tests)
  banner.go               # Banner rendering, versioning, byline, and Live Info block
  cli.go                  # CLI subcommands: tui, install, doctor, uninstall, version, help
  config.go               # Config struct, canonical Fields table, load/save logic
  font.go                 # Multi-font ASCII generator (default, block, half, slant, shade)
  form.go                 # Interactive Config screen (Theme, Banner, Shell, Doctor, Log)
  home.go                 # Home screen: overview, live search, item store
  layout.go               # Screen grammar: indentation, single rule, tuta-style rows, bottom menu
  parse.go                # Atomic, comment-preserving KEY="value" config parser
  store.go                # JSON item storage (~/.local/share/tui-example/items.json)
  style.go                # Color themes, named Lipgloss styles, and glitter effects
  system.go               # Shell alias installer, doctor checks, log file management
  util.go                 # String utilities, truncation, and window scrolling

Makefile                  # Build, test, run, and clean targets
```

---

## 🧪 Testing

The Elm architecture allows full verification of UI state, formatting, scrolling, and theme transitions headlessly without launching a terminal window:

```bash
go test -v ./...
```

Tests include:
* Indentation (2 spaces) and single dotted rule verification
* Bottom menu navigation and glitter animation
* Symmetry between `q` and `Esc`
* Global `Ctrl` shortcut execution and confirmation screens
* 1:1 synchronization between `Config` struct, form fields, and config file
* Windowed scrolling when content exceeds terminal dimensions
* Banner font switching and Live Info block toggling
* Diagnostic doctor execution and log viewer clearing
