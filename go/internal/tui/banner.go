package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// App metadata, set from main() so this package stays project-agnostic.
var (
	AppName        = "app"
	AppDescription = "a bubbletea TUI"
	// Version is overridable at build time via -ldflags "-X ...tui.Version=x".
	Version = "0.1.0"
	// BannerArt is the block logo. Replace it with your own (keep it ≤ 4 lines);
	// empty = the app name in bold.
	BannerArt = ""
)

var (
	bylineStyle     = lipgloss.NewStyle().Foreground(MutedColor)
	bannerRuleStyle = lipgloss.NewStyle().Foreground(MutedColor).Faint(true)
)

// frakturify maps ASCII letters to Unicode Mathematical Bold Fraktur glyphs
// (𝖆–𝖟 / 𝕬–𝖅) — a gothic look without a special font. Non-letters pass through.
func frakturify(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(0x1D586 + (r - 'a'))
		case r >= 'A' && r <= 'Z':
			b.WriteRune(0x1D56C + (r - 'A'))
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// renderBanner is the banner without a live line (CLI commands).
func renderBanner() string { return renderBannerLive("") }

// renderBannerLive returns the banner — the logo with v<Version> directly after
// its last line, the description, the byline and the ONE dotted rule — and, in
// the TUI, the live info block (MDI icons) directly UNDER the rule. Text is
// indented; the rule is at column 0.
func renderBannerLive(live string) string {
	art := BannerArt
	if art == "" {
		art = AppName
	}
	lines := strings.Split(strings.Trim(art, "\n"), "\n")
	// BANNER_TEXT / BANNER_FONT: the default logo only shows the app name, so any
	// other text is drawn with the font (block when the font is "default").
	text, font := firstNonEmpty(uiBannerText, AppName), uiBannerFont
	if font == "default" && text != AppName {
		font = "block"
	}
	if f := renderFont(text, font); f != nil {
		lines = f
	}
	for i, l := range lines {
		lines[i] = indent + bannerStyle.Render(l)
	}
	lines[len(lines)-1] += "  " + subtitleStyle.Render("v"+Version)

	var b strings.Builder
	b.WriteString(strings.Join(lines, "\n") + "\n")
	b.WriteString(indent + subtitleStyle.Render(AppDescription) + "\n\n")
	b.WriteString(indent + bylineStyle.Render(frakturify("by Tdwg Entertainment Solutions")+" ®") + "\n\n")
	b.WriteString(rule())
	if live != "" {
		b.WriteString("\n\n" + live)
	}
	return b.String()
}
