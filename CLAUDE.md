# GO-TUI — CLAUDE.md

Standardized TUI CLI starter templates (Go, Python, Bash) and runnable reference implementation (`tui-example`).

## Guidelines

- **Go**: Charmbracelet Bubbletea + Lipgloss. Pure model (`Update` / `View`), headless tests in `app_test.go`.
- **Grammar**:
  - All text indented by 2 spaces.
  - Exactly one dotted rule under the banner.
  - Live info block directly under the banner rule with Nerd Font MDI icons, active only when live data exists.
  - One-line bottom menu `▶ Label key · … · Help h` with glitter effect on focus.
  - Symmetry: `q` and `Esc` behave identically.
  - Config: All fields map 1:1 to `~/.config/<AppName>/<AppName>.conf` preserving comments on save.
- **Verification**:
  - `make build && make test`
