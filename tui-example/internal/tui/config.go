package tui

// Config + the config file ~/.config/<AppName>/<AppName>.conf.
//
// Every setting on the Config screen is a key in this file (standard: "all
// reflected in the config file"). To add a setting: a row in Fields, a field in
// Config, and a line in fromMap/toMap — then a row in the Config form (form.go).
// Fields may be OS-specific (OS != ""): they are only read, written and shown
// on that OS.

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Config holds every user-configurable setting.
type Config struct {
	Greeting      string
	ItemsShown    string // max list rows before the list scrolls
	ConfirmDelete bool

	// Theme per element (style.go). Theme is the accent; the others take a
	// colour name, "accent" or "default".
	Theme       string
	ColorBanner string
	ColorHeader string
	ColorText   string
	ColorMuted  string
	// More theme elements: menu shortcuts, selected menu entry, the dotted
	// rule, text under the logo, the live info block.
	ColorShortcut   string
	ColorSelected   string
	ColorRule       string
	ColorRuleFooter string
	ColorSubtitle   string
	ColorLive       string
	// BannerText: the logo text; empty = the app name. Any other text is drawn
	// with the font (block when the font is "default").
	BannerText string
	// BannerFont: default (the name in bold) | block | half | slant | shade.
	BannerFont string
	// LiveInfo: what the block under the banner shows — full | compact | off.
	LiveInfo string
	// MenuEffect: glitter (a glint sweeps across the selected menu entry) | none.
	MenuEffect string

	// ShellAliases keeps `alias <AppName>=…` in ~/.zshrc and ~/.bashrc.
	ShellAliases bool
}

// Field describes one config key for the file and the Config screen.
type Field struct {
	Key         string
	Default     string
	Description string
	Section     string
	OS          string // "" = everywhere; "unix" = linux+darwin; or a GOOS
}

// Fields is the canonical table. Order = order in the config file.
var Fields = []Field{
	{"GREETING", "Welcome to GO-TUI — reference implementation of TUI Scaffold v2", "Line shown on the Home screen.", "General", ""},
	{"ITEMS_SHOWN", "10", "Max items listed before the list scrolls.", "General", ""},
	{"CONFIRM_DELETE", "true", "Ask before deleting an item.", "Behaviour", ""},
	{"THEME", "blue", "Accent: blue | green | red | yellow | magenta | cyan | orange | purple", "Theme (Config → Theme; \"Reset to default\" restores these)", ""},
	{"COLOR_BANNER", "accent", "Element colours: accent | default | blue | ocean | green | red | yellow | magenta | cyan | orange | purple | grey | white", "Theme (Config → Theme; \"Reset to default\" restores these)", ""},
	{"COLOR_HEADER", "ocean", "", "Theme (Config → Theme; \"Reset to default\" restores these)", ""},
	{"COLOR_TEXT", "white", "", "Theme (Config → Theme; \"Reset to default\" restores these)", ""},
	{"COLOR_MUTED", "accent", "", "Theme (Config → Theme; \"Reset to default\" restores these)", ""},
	{"COLOR_SHORTCUT", "accent", "Menu shortcuts, selected menu entry, the dotted rule, text under the logo, live info block", "Theme (Config → Theme; \"Reset to default\" restores these)", ""},
	{"COLOR_SELECTED", "white", "", "Theme (Config → Theme; \"Reset to default\" restores these)", ""},
	{"COLOR_RULE", "grey", "", "Theme (Config → Theme; \"Reset to default\" restores these)", ""},
	{"COLOR_RULE_FOOTER", "grey", "Divider above bottom menu", "Theme (Config → Theme; \"Reset to default\" restores these)", ""},
	{"COLOR_SUBTITLE", "white", "", "Theme (Config → Theme; \"Reset to default\" restores these)", ""},
	{"COLOR_LIVE", "blue", "", "Theme (Config → Theme; \"Reset to default\" restores these)", ""},
	{"BANNER_TEXT", "", "Banner (Config → Theme → Banner). Text: empty = the app name; any other text is drawn with the font (block when the font is default)", "Theme (Config → Theme; \"Reset to default\" restores these)", ""},
	{"BANNER_FONT", "default", "Font: default (the name) | block | half | slant | shade", "Theme (Config → Theme; \"Reset to default\" restores these)", ""},
	{"LIVE_INFO", "full", "Live info under the banner: full (items, search, host IP) | compact | off", "Theme (Config → Theme; \"Reset to default\" restores these)", ""},
	{"MENU_EFFECT", "glitter", "Selected menu entry: glitter (a glint sweeps across the text) | none", "Theme (Config → Theme; \"Reset to default\" restores these)", ""},
	{"SHELL_ALIASES", "true", "Keep the shell alias in ~/.zshrc + ~/.bashrc: true | false", "Shell", "unix"},
}

// fieldOnThisOS reports whether f applies to the running OS.
func fieldOnThisOS(f Field) bool {
	switch f.OS {
	case "":
		return true
	case "unix":
		return runtime.GOOS != "windows"
	}
	return f.OS == runtime.GOOS
}

// ConfigPath returns ~/.config/<AppName>/<AppName>.conf.
func ConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", AppName, AppName+".conf")
}

// displayPath shortens paths starting with the user's home directory to ~/....
func displayPath(p string) string {
	if p == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err == nil && home != "" && strings.HasPrefix(p, home) {
		return "~" + strings.TrimPrefix(p, home)
	}
	return p
}

// Default returns a Config populated with the defaults declared in Fields.
func Default() *Config {
	c := &Config{}
	fromMap(c, map[string]string{})
	return c
}

// Load reads the config file. A missing file yields Default() and found=false.
func Load() (*Config, bool, error) {
	p := ConfigPath()
	raw, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			old := filepath.Join(filepath.Dir(filepath.Dir(p)), strings.ToLower(AppName), strings.ToLower(AppName)+".conf")
			if oldRaw, oldErr := os.ReadFile(old); oldErr == nil {
				raw = oldRaw
			} else {
				return Default(), false, nil
			}
		} else {
			return nil, false, fmt.Errorf("read %s: %w", p, err)
		}
	}
	doc, err := Parse(raw)
	if err != nil {
		return nil, true, err
	}
	values := map[string]string{}
	for _, f := range Fields {
		if v, ok := doc.Get(f.Key); ok && fieldOnThisOS(f) {
			values[f.Key] = v
		}
	}
	c := &Config{}
	fromMap(c, values)
	return c, true, nil
}

// Save writes cfg into the existing file, keeping its comments and unknown keys.
func Save(cfg *Config) error {
	var doc *Doc
	if raw, err := os.ReadFile(ConfigPath()); err == nil {
		doc, _ = Parse(raw)
	}
	if doc == nil || len(doc.entries) == 0 {
		doc = configTemplate()
	}
	return writeDoc(doc, cfg)
}

// SaveFresh rewrites the file in the CURRENT format (every key with its
// comment, in order) keeping cfg's values. `install` uses it so a config file
// from an older version gains the new keys.
func SaveFresh(cfg *Config) error { return writeDoc(configTemplate(), cfg) }

func writeDoc(doc *Doc, cfg *Config) error {
	p := ConfigPath()
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return fmt.Errorf("mkdir %s: %w", filepath.Dir(p), err)
	}
	m := toMap(cfg)
	for _, f := range Fields {
		if fieldOnThisOS(f) {
			doc.Set(f.Key, m[f.Key])
		}
	}
	return atomicWrite(p, doc.Bytes())
}

func fromMap(cfg *Config, values map[string]string) {
	pick := func(key string) string {
		if v, ok := values[key]; ok {
			return v
		}
		for _, f := range Fields {
			if f.Key == key {
				return f.Default
			}
		}
		return ""
	}
	oneOf := func(v, def string, allowed []string) string {
		for _, a := range allowed {
			if v == a {
				return v
			}
		}
		return def
	}
	cfg.Greeting = pick("GREETING")
	cfg.ItemsShown = pick("ITEMS_SHOWN")
	cfg.ConfirmDelete = parseBool(pick("CONFIRM_DELETE"))
	cfg.Theme = oneOf(pick("THEME"), "blue", accentNames)
	cfg.ColorBanner = oneOf(pick("COLOR_BANNER"), "accent", colorNames)
	cfg.ColorHeader = oneOf(pick("COLOR_HEADER"), "ocean", colorNames)
	cfg.ColorText = oneOf(pick("COLOR_TEXT"), "white", colorNames)
	cfg.ColorMuted = oneOf(pick("COLOR_MUTED"), "accent", colorNames)
	cfg.ColorShortcut = oneOf(pick("COLOR_SHORTCUT"), "accent", colorNames)
	cfg.ColorSelected = oneOf(pick("COLOR_SELECTED"), "white", colorNames)
	cfg.ColorRule = oneOf(pick("COLOR_RULE"), "grey", colorNames)
	cfg.ColorRuleFooter = oneOf(pick("COLOR_RULE_FOOTER"), cfg.ColorRule, colorNames)
	cfg.ColorSubtitle = oneOf(pick("COLOR_SUBTITLE"), "white", colorNames)
	cfg.ColorLive = oneOf(pick("COLOR_LIVE"), "blue", colorNames)
	cfg.BannerText = pick("BANNER_TEXT")
	cfg.BannerFont = oneOf(pick("BANNER_FONT"), "default", bannerFonts)
	cfg.LiveInfo = oneOf(pick("LIVE_INFO"), "full", liveInfoModes)
	cfg.MenuEffect = oneOf(pick("MENU_EFFECT"), "glitter", menuEffects)
	cfg.ShellAliases = parseBool(pick("SHELL_ALIASES")) && runtime.GOOS != "windows"
}

func toMap(cfg *Config) map[string]string {
	return map[string]string{
		"GREETING":          cfg.Greeting,
		"ITEMS_SHOWN":       cfg.ItemsShown,
		"CONFIRM_DELETE":    boolStr(cfg.ConfirmDelete),
		"THEME":             cfg.Theme,
		"COLOR_BANNER":      cfg.ColorBanner,
		"COLOR_HEADER":      cfg.ColorHeader,
		"COLOR_TEXT":        cfg.ColorText,
		"COLOR_MUTED":       cfg.ColorMuted,
		"COLOR_SHORTCUT":    cfg.ColorShortcut,
		"COLOR_SELECTED":    cfg.ColorSelected,
		"COLOR_RULE":        cfg.ColorRule,
		"COLOR_RULE_FOOTER": cfg.ColorRuleFooter,
		"COLOR_SUBTITLE":    cfg.ColorSubtitle,
		"COLOR_LIVE":        cfg.ColorLive,
		"BANNER_TEXT":       cfg.BannerText,
		"BANNER_FONT":       cfg.BannerFont,
		"LIVE_INFO":         cfg.LiveInfo,
		"MENU_EFFECT":       cfg.MenuEffect,
		"SHELL_ALIASES":     boolStr(cfg.ShellAliases),
	}
}

// themeKeys are the keys "Reset to default" restores (theme + banner choices).
var themeKeys = []string{"THEME", "COLOR_BANNER", "COLOR_HEADER", "COLOR_TEXT", "COLOR_MUTED", "COLOR_SHORTCUT",
	"COLOR_SELECTED", "COLOR_RULE", "COLOR_RULE_FOOTER", "COLOR_SUBTITLE", "COLOR_LIVE", "BANNER_TEXT", "BANNER_FONT", "LIVE_INFO", "MENU_EFFECT"}

// liveInfoModes are the LIVE_INFO choices.
var liveInfoModes = []string{"full", "compact", "off"}

// resetTheme puts every theme key back to its default.
func (c *Config) resetTheme() {
	m, d := toMap(c), toMap(Default())
	for _, k := range themeKeys {
		m[k] = d[k]
	}
	fromMap(c, m)
}

func parseBool(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "yes", "on":
		return true
	}
	return false
}

func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// configTemplate returns a fresh Doc: header, sections, each key with its
// description at its default value.
func configTemplate() *Doc {
	d := NewDoc()
	d.AppendComment(AppName + " configuration (" + displayPath(ConfigPath()) + "). Edit here or on the Config screen.")
	d.AppendComment(`Format: shell-style KEY="value" with # comments.`)
	var lastSection string
	for _, f := range Fields {
		if !fieldOnThisOS(f) {
			continue
		}
		if f.Section != lastSection {
			d.AppendBlank()
			d.AppendComment("--- " + f.Section + " ---")
			lastSection = f.Section
		}
		if f.Description != "" {
			d.AppendComment(f.Description)
		}
		d.Set(f.Key, f.Default)
	}
	return d
}

// ItemsShownInt parses ITEMS_SHOWN, flooring to 1 and falling back to 10.
func (c *Config) ItemsShownInt() int {
	n, err := strconv.Atoi(strings.TrimSpace(c.ItemsShown))
	if err != nil || n < 1 {
		return 10
	}
	return n
}

// editorCmd opens the config file in $VISUAL / $EDITOR (vi fallback).
func editorCmd() (*exec.Cmd, error) {
	editor := os.Getenv("VISUAL")
	if editor == "" {
		editor = os.Getenv("EDITOR")
	}
	if editor == "" {
		editor = "vi"
	}
	bin, err := exec.LookPath(editor)
	if err != nil {
		return nil, fmt.Errorf("editor %q not found in PATH", editor)
	}
	return exec.Command(bin, ConfigPath()), nil
}
