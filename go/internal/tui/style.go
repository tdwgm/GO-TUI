// style.go — the theme: palette, per-element colours and the shared styles
// (TUI Scaffold standard v2; same as gu-vpn-pid/style.go).
//
// Modelled on tuta-cli: body text has NO colour (the terminal's own foreground),
// secondary text is muted, section headers are ocean blue, the accent marks focus
// (▶, [ ]) and titles, errors are red. Nothing is hard-coded white.
//
// The theme is configurable per element (config keys, Settings → Theme):
//
//	THEME         accent colour (focus marker, brackets, titles, keys)
//	COLOR_BANNER  the ASCII banner             (default: accent)
//	COLOR_HEADER  section headers, ✓ messages  (default: ocean)
//	COLOR_TEXT    body text and values          (default: terminal default)
//	COLOR_MUTED   labels of info rows, hints, banner subtitle (default: grey)
//
// Each element takes a colour name from colorNames; "accent" follows THEME and
// "default" means no colour at all.
package tui

import "github.com/charmbracelet/lipgloss"

// Fixed palette (Tuta dark). DangerColor/OKColor are semantic and also offered
// as theme colours ("red", "green").
var (
	MutedColor  = lipgloss.Color("#888888")
	DangerColor = lipgloss.Color("#FF7A7A")
	OKColor     = lipgloss.Color("#88CC88")
	BgColor     = lipgloss.Color("#232323")
)

// accentNames are the THEME choices; colorNames the per-element choices.
var (
	accentNames = []string{"blue", "green", "red", "yellow", "magenta", "cyan", "orange", "purple"}
	colorNames  = []string{"accent", "default", "blue", "ocean", "green", "red", "yellow", "magenta", "cyan", "orange", "purple", "grey", "white"}
)

var namedColors = map[string]lipgloss.Color{
	"blue":    "#ACC7FF",
	"ocean":   "#7AA2F7", // deeper blue: the default section headers
	"green":   OKColor,
	"red":     DangerColor,
	"yellow":  "#F3F99D",
	"magenta": "#FF79C6",
	"cyan":    "#8BE9FD",
	"orange":  "#FFB86C",
	"purple":  "#BD93F9",
	"grey":    MutedColor,
	"white":   "#EEEEEE",
}

// Live theme colours (rebuilt by applyStyle).
var (
	AccentColor lipgloss.Color         = namedColors["blue"]
	BannerColor lipgloss.TerminalColor = namedColors["blue"]
	HeaderColor lipgloss.TerminalColor = namedColors["ocean"]
	TextColor   lipgloss.TerminalColor = lipgloss.NoColor{}
	HintColor   lipgloss.TerminalColor = MutedColor
)

// Element styles (rebuilt by applyStyle) and the banner/live choices.
var (
	shortcutStyle = lipgloss.NewStyle().Foreground(MutedColor)
	selectedStyle = lipgloss.NewStyle()
	subtitleStyle = lipgloss.NewStyle().Foreground(MutedColor)
	liveStyle     = lipgloss.NewStyle()
	uiBannerFont  = "default"
	uiBannerText  = ""
	uiLiveInfo    = "full"
)

// Shared styles, rebuilt by applyStyle.
var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(AccentColor)
	keyStyle    = lipgloss.NewStyle().Bold(true).Foreground(AccentColor)
	bannerStyle = lipgloss.NewStyle().Bold(true).Foreground(BannerColor)
	okStyle     = lipgloss.NewStyle().Foreground(HeaderColor)
	fgStyle     = lipgloss.NewStyle().Foreground(TextColor)
	mutedStyle  = lipgloss.NewStyle().Foreground(HintColor)
	errStyle    = lipgloss.NewStyle().Foreground(DangerColor)
	// warnStyle is the in-progress colour; it follows the accent (no orange).
	warnStyle = lipgloss.NewStyle().Foreground(AccentColor)
)

// colorFor resolves an element colour name. Unknown names fall back to def.
func colorFor(name string, def lipgloss.TerminalColor) lipgloss.TerminalColor {
	switch name {
	case "accent":
		return AccentColor
	case "default":
		return lipgloss.NoColor{}
	}
	if c, ok := namedColors[name]; ok {
		return c
	}
	return def
}

// applyStyle rebuilds every colour and style from cfg. Called at start, after a
// settings save/reload, and on every theme-choice change (live preview).
func applyStyle(cfg *Config) {
	if c, ok := namedColors[cfg.Theme]; ok && cfg.Theme != "grey" && cfg.Theme != "white" {
		AccentColor = c
	} else {
		AccentColor = namedColors["blue"]
	}
	BannerColor = colorFor(cfg.ColorBanner, AccentColor)
	HeaderColor = colorFor(cfg.ColorHeader, namedColors["ocean"])
	TextColor = colorFor(cfg.ColorText, lipgloss.NoColor{})
	HintColor = colorFor(cfg.ColorMuted, MutedColor)

	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(AccentColor)
	keyStyle = lipgloss.NewStyle().Bold(true).Foreground(AccentColor)
	warnStyle = lipgloss.NewStyle().Foreground(AccentColor)
	bannerStyle = lipgloss.NewStyle().Bold(true).Foreground(BannerColor)
	okStyle = lipgloss.NewStyle().Foreground(HeaderColor)
	fgStyle = lipgloss.NewStyle().Foreground(TextColor)
	mutedStyle = lipgloss.NewStyle().Foreground(HintColor)
	shortcutStyle = lipgloss.NewStyle().Foreground(colorFor(cfg.ColorShortcut, MutedColor))
	selectedStyle = lipgloss.NewStyle().Foreground(colorFor(cfg.ColorSelected, lipgloss.NoColor{}))
	subtitleStyle = lipgloss.NewStyle().Foreground(colorFor(cfg.ColorSubtitle, MutedColor))
	liveStyle = lipgloss.NewStyle().Foreground(colorFor(cfg.ColorLive, lipgloss.NoColor{}))
	bylineStyle = subtitleStyle
	bannerRuleStyle = lipgloss.NewStyle().Foreground(colorFor(cfg.ColorRule, MutedColor)).Faint(true)
	uiBannerFont, uiLiveInfo, uiBannerText = cfg.BannerFont, cfg.LiveInfo, cfg.BannerText
	uiMenuEffect = cfg.MenuEffect
}

// spinnerColor cycles the spinner through the theme's own colours (accent →
// header → banner) instead of a fixed rainbow, so it never clashes with THEME.
func spinnerColor(tick int) lipgloss.TerminalColor {
	cycle := []lipgloss.TerminalColor{AccentColor, AccentColor, HeaderColor, HeaderColor}
	if _, plain := BannerColor.(lipgloss.NoColor); !plain {
		cycle = append(cycle, BannerColor, BannerColor)
	}
	return cycle[(tick/2)%len(cycle)]
}
