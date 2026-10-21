package tui

// A tiny local JSON store for the demo "items" — a stand-in for whatever your
// app actually persists. Lives at ~/.local/share/<AppName>/items.json and is
// written atomically. Swap []string for your own type as the app grows.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// itemsPath returns ~/.local/share/<AppName>/items.json (XDG data dir).
func itemsPath() string {
	dir := os.Getenv("XDG_DATA_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dir, AppName, "items.json")
}

// loadItems reads the item list. A missing/empty file yields an empty slice.
func loadItems() ([]string, error) {
	p := itemsPath()
	raw, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return seedItems(), nil
		}
		return nil, fmt.Errorf("read %s: %w", p, err)
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		return []string{}, nil
	}
	var items []string
	if err := json.Unmarshal(raw, &items); err != nil {
		return nil, fmt.Errorf("parse %s: %w", p, err)
	}
	return items, nil
}

// saveItems writes the list atomically, creating the data dir if needed.
func saveItems(items []string) error {
	p := itemsPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(p), err)
	}
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return atomicWrite(p, data)
}

// seedItems is the first-run demo content, so a fresh scaffold isn't empty.
func seedItems() []string {
	return []string{
		"Charmbracelet Bubbletea + Lipgloss TUI standard",
		"Per-element theme customization (accent, banner, headers, text, shortcuts)",
		"Live info block directly under the banner rule with Nerd Font MDI icons",
		"Glitter sweep effect across active menu selection",
		"Config screen mirroring 1:1 with ~/.config/GO-TUI/GO-TUI.conf",
		"Built-in doctor diagnostic checklist (^d) and log viewer (^l)",
		"Contextual Help screen (h) with unified shortcuts and CLI command guide",
	}
}
