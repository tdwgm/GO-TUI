package tui

// form.go — the Config screen: a list of rows bound to config KEYS plus the
// bottom menu. Rows edit a key/value map that round-trips through
// fromMap/toMap, so a new setting needs only a Fields entry and a row here.
//
// Keys (standard v2): ↑/↓ rows (past the last → menu) · ←/→/Space change a
// choice · Tab ⇄ menu (back to the top) · in the menu ←/→ · Enter save / run ·
// Ctrl+S save · Esc (q outside text fields) back · F1 help.

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type rowKind int

const (
	rowText rowKind = iota
	rowChoice
	rowAction
	rowHeader
	rowInfo
	rowSubheader // an indented heading inside a section (Theme → Banner)
)

type formRow struct {
	kind    rowKind
	label   string
	key     string // config key (rowText / rowChoice)
	input   textinput.Model
	choices []string
	choice  int
	action  string // rowAction
	info    string // rowInfo value
}

func (r formRow) selectable() bool {
	return r.kind == rowText || r.kind == rowChoice || r.kind == rowAction
}

func (r formRow) value() string {
	if r.kind == rowChoice {
		return r.choices[r.choice]
	}
	return strings.TrimSpace(r.input.Value())
}

type form struct {
	rows      []formRow
	focus     int
	buttons   []button
	inButtons bool
	btn       int
	base      *Config
}

// yesNo choices map to the "true"/"false" stored in the file.
var yesNo = []string{"no", "yes"}

func newForm(cfg *Config) *form {
	m := toMap(cfg)
	text := func(label, key string) formRow {
		ti := textinput.New()
		ti.Prompt = ""
		ti.SetValue(m[key])
		return formRow{kind: rowText, label: label, key: key, input: ti}
	}
	choice := func(label, key string, opts []string) formRow {
		cur := m[key]
		if cur == "true" || cur == "false" {
			cur = map[string]string{"true": "yes", "false": "no"}[cur]
		}
		r := formRow{kind: rowChoice, label: label, key: key, choices: opts}
		for i, o := range opts {
			if o == cur {
				r.choice = i
			}
		}
		return r
	}
	header := func(l string) formRow { return formRow{kind: rowHeader, label: l} }
	info := func(l, v string) formRow { return formRow{kind: rowInfo, label: l, info: v} }

	f := &form{base: cfg}
	f.rows = []formRow{
		info("Config", displayPath(ConfigPath())),
		header("General"),
		text("Greeting", "GREETING"),
		text("Items shown", "ITEMS_SHOWN"),
		header("Behaviour"),
		choice("Confirm delete", "CONFIRM_DELETE", yesNo),
		header("Theme"),
		choice("Accent", "THEME", accentNames),
		choice("Banner", "COLOR_BANNER", colorNames),
		choice("Headers", "COLOR_HEADER", colorNames),
		choice("Text", "COLOR_TEXT", colorNames),
		choice("Muted text", "COLOR_MUTED", colorNames),
		choice("Selected text", "COLOR_SELECTED", colorNames),
		choice("Shortcuts", "COLOR_SHORTCUT", colorNames),
		choice("Rule", "COLOR_RULE", colorNames),
		choice("Footer rule", "COLOR_RULE_FOOTER", colorNames),
		choice("Under logo", "COLOR_SUBTITLE", colorNames),
		choice("Live info text", "COLOR_LIVE", colorNames),
		choice("Menu effect", "MENU_EFFECT", menuEffects),
		{kind: rowSubheader, label: "Banner"},
		text("Text", "BANNER_TEXT"),
		choice("Font", "BANNER_FONT", bannerFonts),
		choice("Live info", "LIVE_INFO", liveInfoModes),
		{kind: rowAction, label: "Reset to default", action: "theme-reset"},
	}
	for i := range f.rows {
		if f.rows[i].key == "BANNER_TEXT" {
			f.rows[i].input.Placeholder = AppName
		}
	}
	if aliasesSupported() { // OS-specific: only shown where it exists
		where := "not installed"
		if w := aliasWhere(); len(w) > 0 {
			where = "installed — " + strings.Join(w, " · ")
		}
		f.rows = append(f.rows,
			header("Shell"),
			info("alias "+AppName, where),
			choice("Shell alias", "SHELL_ALIASES", yesNo),
		)
	}
	f.rows = append(f.rows,
		header("Doctor"), formRow{kind: rowAction, label: "Run doctor", action: "doctor"},
		header("Log"), info("Log file", displayPath(logPath())))
	f.buttons = configMenu
	f.focusRow(f.firstSelectable())
	return f
}

// configMenu is the Config screen's bottom menu — lowercase shortcuts like the
// Home menu. Help's shortcut list is generated from it (and homeButtons).
var configMenu = []button{
	{"Save", "save", "^s"}, {"Cancel", "back", "esc"}, {"Doctor", "doctor", "^d"}, {"Log", "log", "^l"},
	{"Edit", "edit-config", "^e"}, {"Reload", "reload", "^r"},
	{"Reset theme", "theme-reset", "^t"}, {"Uninstall", "uninstall", "^u"},
	{"Help", "help", "h"},
}

// toConfig folds the rows onto a copy of base.
func (f *form) toConfig() *Config {
	m := toMap(f.base)
	for _, r := range f.rows {
		switch r.kind {
		case rowText:
			m[r.key] = r.value()
		case rowChoice:
			v := r.value()
			if v == "yes" || v == "no" {
				v = boolStr(v == "yes")
			}
			m[r.key] = v
		}
	}
	c := &Config{}
	fromMap(c, m)
	return c
}

// setChoice selects value v on the row bound to key.
func (f *form) setChoice(key, v string) {
	for i := range f.rows {
		if f.rows[i].key != key {
			continue
		}
		for j, c := range f.rows[i].choices {
			if c == v {
				f.rows[i].choice = j
			}
		}
	}
}

// resetTheme puts every theme row back to its default (live preview).
func (f *form) resetTheme() {
	d := toMap(Default())
	for _, k := range themeKeys {
		v := d[k]
		f.setChoice(k, v)
		for i := range f.rows {
			if f.rows[i].key == k && f.rows[i].kind == rowText {
				f.rows[i].input.SetValue(v)
			}
		}
	}
	applyStyle(f.toConfig())
}

func (f *form) firstSelectable() int {
	for i, r := range f.rows {
		if r.selectable() {
			return i
		}
	}
	return 0
}

func (f *form) lastSelectable() int {
	for i := len(f.rows) - 1; i >= 0; i-- {
		if f.rows[i].selectable() {
			return i
		}
	}
	return 0
}

func (f *form) focusRow(i int) {
	f.rows[f.focus].input.Blur()
	f.inButtons, f.focus = false, i
	if f.rows[i].kind == rowText {
		f.rows[i].input.Focus()
	}
}

func (f *form) focusMenu() {
	f.rows[f.focus].input.Blur()
	f.inButtons, f.btn = true, 0
}

// move steps between selectable rows; past the last row it enters the menu,
// and ↑ from the menu returns to the last row.
func (f *form) move(d int) {
	if f.inButtons {
		if d < 0 {
			f.focusRow(f.lastSelectable())
		}
		return
	}
	for i := f.focus + d; i >= 0 && i < len(f.rows); i += d {
		if f.rows[i].selectable() {
			f.focusRow(i)
			return
		}
	}
	if d > 0 {
		f.focusMenu()
	}
}

func (f *form) cycle(d int) {
	r := &f.rows[f.focus]
	if f.inButtons || r.kind != rowChoice {
		return
	}
	r.choice = nextIdx(r.choice, d, len(r.choices))
	if strings.HasPrefix(r.key, "COLOR_") || r.key == "THEME" || r.key == "BANNER_FONT" || r.key == "LIVE_INFO" || r.key == "MENU_EFFECT" {
		applyStyle(f.toConfig()) // live preview
	}
}

func (f *form) typing() bool { return !f.inButtons && f.rows[f.focus].kind == rowText }

// ---- model glue ----

func (m model) updateConfig(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	f := m.form
	switch k.String() {
	case "tab", "shift+tab":
		if f.inButtons {
			f.focusRow(f.firstSelectable())
		} else {
			f.focusMenu()
		}
		return m, textinput.Blink
	case "down":
		f.move(1)
		return m, textinput.Blink
	case "up":
		f.move(-1)
		return m, textinput.Blink
	case "left", "right":
		d := 1
		if k.String() == "left" {
			d = -1
		}
		if f.inButtons {
			f.btn = nextIdx(f.btn, d, len(f.buttons))
			return m, nil
		}
		if !f.typing() {
			f.cycle(d)
			return m, nil
		}
	case " ":
		if !f.typing() {
			f.cycle(1)
			return m, nil
		}
	case "esc":
		return m.run("back")
	case "q":
		if !f.typing() {
			return m.run("back")
		}
	case "h":
		if !f.typing() {
			return m.run("help")
		}
	case "ctrl+s":
		return m.run("save")
	case "enter":
		if f.inButtons {
			return m.run(f.buttons[f.btn].action)
		}
		if r := f.rows[f.focus]; r.kind == rowAction {
			return m.run(r.action)
		}
		return m.run("save")
	}
	if f.typing() {
		var cmd tea.Cmd
		f.rows[f.focus].input, cmd = f.rows[f.focus].input.Update(k)
		if f.rows[f.focus].key == "BANNER_TEXT" {
			applyStyle(f.toConfig()) // live preview
		}
		return m, cmd
	}
	return m, nil
}

func (m model) viewConfig() string {
	f := m.form
	var lines []string
	focusLine := -1
	for i, r := range f.rows {
		focused := !f.inButtons && i == f.focus
		if r.kind == rowHeader && i > 0 {
			lines = append(lines, "")
		}
		if focused {
			focusLine = len(lines)
		}
		switch r.kind {
		case rowHeader:
			lines = append(lines, sectionHeader(r.label))
		case rowSubheader:
			lines = append(lines, "", indent+"  "+okStyle.Render(r.label))
		case rowInfo:
			lines = append(lines, infoLine(r.label, fgStyle.Render(r.info)))
		case rowText:
			lines = append(lines, rowLine(focused, r.label, r.input.View()))
		case rowChoice:
			l, rr := mutedStyle.Render("‹ "), mutedStyle.Render(" ›")
			if focused {
				l, rr = keyStyle.Render("‹ "), keyStyle.Render(" ›")
			}
			lines = append(lines, rowLine(focused, r.label, l+fgStyle.Render(r.value())+rr))
		case rowAction:
			marker, lbl := "  ", fgStyle.Render(r.label)
			if focused {
				marker, lbl = keyStyle.Render("▶ "), keyStyle.Render(r.label)
			}
			lines = append(lines, indent+marker+keyStyle.Render("[")+lbl+keyStyle.Render("]"))
		}
	}
	head := m.banner() + "\n\n" + screenTitle("Config") + "\n\n"
	tail := m.tail(f.buttons, f.btn, f.inButtons)
	return head + strings.Join(scrollWindow(lines, focusLine, m.room(head, tail)), "\n") + "\n" + tail
}
