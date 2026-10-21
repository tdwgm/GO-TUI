package tui

import (
	"os"
	"regexp"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func plain(s string) string { return ansiRE.ReplaceAllString(s, "") }

func keyMsg(k string) tea.KeyMsg {
	switch k {
	case "tab":
		return tea.KeyMsg{Type: tea.KeyTab}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	case "f1":
		return tea.KeyMsg{Type: tea.KeyF1}
	case "ctrl+u":
		return tea.KeyMsg{Type: tea.KeyCtrlU}
	case "ctrl+t":
		return tea.KeyMsg{Type: tea.KeyCtrlT}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
}

func press(t *testing.T, m model, keys ...string) (model, tea.Cmd) {
	t.Helper()
	var cmd tea.Cmd
	for _, k := range keys {
		var mm tea.Model
		mm, cmd = m.Update(keyMsg(k))
		m = mm.(model)
	}
	return m, cmd
}

func isQuit(cmd tea.Cmd) bool {
	if cmd == nil {
		return false
	}
	_, ok := cmd().(tea.QuitMsg)
	return ok
}

// testModel runs with a temporary HOME so nothing real is read or written.
func testModel(t *testing.T) model {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_DATA_HOME", "")
	t.Setenv("XDG_STATE_HOME", "")
	AppName = "demo"
	m := initialModel()
	m.width = 100
	m.home.items = []string{"alpha", "bravo", "charlie"}
	return m
}

func TestTextIndentedOneRule(t *testing.T) {
	m := testModel(t)
	views := map[string]string{"home": m.View()}
	for _, k := range []string{"o", "h"} {
		mm, _ := press(t, m, "down", "down", k) // onto the list so letters are shortcuts
		views[k] = mm.View()
	}
	for name, v := range views {
		rules := 0
		for i, l := range strings.Split(plain(v), "\n") {
			switch {
			case l == "":
			case strings.HasPrefix(l, "·"):
				rules++
			case !strings.HasPrefix(l, indent):
				t.Errorf("%s line %d not indented: %q", name, i, l)
			}
		}
		if rules != 1 {
			t.Errorf("%s has %d rules, want exactly 1", name, rules)
		}
	}
}

func TestMenuLineAndNavigation(t *testing.T) {
	m := testModel(t)
	v := plain(m.View())
	if !strings.Contains(v, "Sample s · Delete x · Doctor d · Reload r · Config o · Log l · Quit q · Help h") || strings.Contains(v, "▶ Sample") {
		t.Fatalf("menu line wrong or selected too early:\n%s", v)
	}
	m, _ = press(t, m, "tab")
	if !m.menuActive || !strings.Contains(plain(m.View()), "▶ Sample") {
		t.Fatal("tab should select the first menu entry")
	}
	m, _ = press(t, m, "right", "right")
	if m.btn != 2 {
		t.Fatalf("→→: btn=%d", m.btn)
	}
	m, _ = press(t, m, "tab")
	if m.menuActive || m.home.focus != 0 {
		t.Fatal("tab in the menu returns to the top")
	}
}

func TestQAndEscSymmetric(t *testing.T) {
	m := testModel(t)
	m, _ = press(t, m, "q") // focus on "New item": q is typed
	if m.home.newIn.Value() != "q" {
		t.Fatalf("q should type into the field, got %q", m.home.newIn.Value())
	}
	if _, cmd := press(t, m, "esc"); !isQuit(cmd) {
		t.Fatal("esc on Home quits")
	}
	m = testModel(t)
	m, _ = press(t, m, "down", "down", "o") // list row → o = Config
	if m.cur != screenConfig {
		t.Fatalf("o: cur=%v", m.cur)
	}
	m, _ = press(t, m, "tab", "q") // menu focused: q = back
	if m.cur != screenHome {
		t.Fatalf("q on the Config menu: cur=%v", m.cur)
	}
	m, _ = press(t, m, "tab", "h", "q") // menu focused: h = Help, q returns
	if m.cur != screenHome {
		t.Fatalf("q in Help returns: cur=%v", m.cur)
	}
}

func TestGlobalShortcutsAndConfirm(t *testing.T) {
	m := testModel(t)
	m, _ = press(t, m, "ctrl+u")
	if m.cur != screenConfirm || m.btn != 1 {
		t.Fatalf("ctrl+u: cur=%v btn=%d (want Confirm, No selected)", m.cur, m.btn)
	}
	m, _ = press(t, m, "n")
	if m.cur != screenHome {
		t.Fatal("n cancels")
	}
	m.cfg.ColorHeader = "red"
	m, _ = press(t, m, "ctrl+t")
	if m.cfg.ColorHeader != "ocean" {
		t.Fatal("ctrl+t resets the theme")
	}
	if b, err := os.ReadFile(ConfigPath()); err != nil || !strings.Contains(string(b), `COLOR_HEADER="ocean"`) {
		t.Fatalf("theme reset not saved: %v", err)
	}
}

func TestEveryFieldInConfigFile(t *testing.T) {
	_ = testModel(t)
	cfg := Default()
	cfg.ColorText = "cyan"
	if err := SaveFresh(cfg); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(ConfigPath())
	for _, f := range Fields {
		if fieldOnThisOS(f) && !strings.Contains(string(b), "\n"+f.Key+"=") {
			t.Errorf("config file lacks %s", f.Key)
		}
	}
	got, found, err := Load()
	if err != nil || !found || got.ColorText != "cyan" {
		t.Fatalf("round trip: %+v found=%v err=%v", got, found, err)
	}
}

func TestConfigScrollsToFit(t *testing.T) {
	m := testModel(t)
	m.height = 30
	m, _ = press(t, m, "down", "down", "o")
	for i := 0; i < 12; i++ {
		m, _ = press(t, m, "down")
	}
	v := plain(m.View())
	if n := strings.Count(v, "\n"); n > m.height {
		t.Fatalf("%d lines in a %d-line terminal", n, m.height)
	}
	if !strings.Contains(v, "▶") {
		t.Fatal("focused row not visible")
	}
}

func TestAddAndDeleteItem(t *testing.T) {
	m := testModel(t)
	m.cfg.ConfirmDelete = false
	m, _ = press(t, m, "z", "e", "d", "enter")
	if len(m.home.items) != 4 {
		t.Fatalf("add: %v", m.home.items)
	}
	m, _ = press(t, m, "down", "down", "x") // first list row = alpha
	if len(m.home.items) != 3 || m.home.items[0] != "bravo" {
		t.Fatalf("delete: %v", m.home.items)
	}
}

// No line may be wider than the terminal: long values (paths) are cut with ….
func TestNoLineWiderThanTerminal(t *testing.T) {
	m := testModel(t)
	m.width, termWidth = 60, 60
	defer func() { termWidth = 0 }()
	for _, v := range []string{m.View(), func() string { mm, _ := press(t, m, "down", "down", "o"); return mm.View() }()} {
		for _, l := range strings.Split(plain(v), "\n") {
			if n := len([]rune(l)); n > 60 {
				t.Errorf("line %d wide: %q", n, l)
			}
		}
	}
}

// The Shell section shows where the alias is installed, exactly once.
func TestAliasRowShowsWhereOnce(t *testing.T) {
	m := testModel(t)
	home, _ := os.UserHomeDir()
	if err := os.WriteFile(home+"/.zshrc", []byte("alias demo='/x'\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	m, _ = press(t, m, "down", "down", "o")
	for _, l := range strings.Split(plain(m.View()), "\n") {
		if strings.Contains(l, "alias demo:") {
			if strings.Count(l, "~/.zshrc:1") != 1 || !strings.Contains(l, "installed —") {
				t.Fatalf("alias row: %q", l)
			}
			return
		}
	}
	t.Fatal("no alias row")
}

// BANNER_FONT previews live in Config; LIVE_INFO=off hides the block.
func TestBannerFontAndLiveInfo(t *testing.T) {
	m := testModel(t)
	m, _ = press(t, m, "down", "down", "o")
	for i := 0; i < 30 && m.form.rows[m.form.focus].key != "BANNER_FONT"; i++ {
		m, _ = press(t, m, "down")
	}
	m, _ = press(t, m, "right", "right") // default → block → half
	if !strings.Contains(plain(m.View()), "▀") {
		t.Fatal("BANNER_FONT=half not previewed")
	}
	m, _ = press(t, m, "esc")
	m.cfg.LiveInfo = "off"
	applyStyle(m.cfg)
	if strings.Contains(strings.SplitN(plain(m.View()), "› Home", 2)[0], "\U000F0279") {
		t.Fatal("LIVE_INFO=off still shows the block")
	}
	m.cfg.LiveInfo = "full"
	applyStyle(m.cfg)
	if l := plain(m.liveLine()); !strings.Contains(l, "\U000F0279 3") || strings.Contains(l, "items") {
		t.Fatalf("live block = icon + value, no words: %q", l)
	}
	m.home.items = nil
	if m.liveLine() != "" {
		t.Fatal("no live data → no live block")
	}
}

// Requirements 2026-10-09 (gu todo §17, generic form): Help is generated from
// the menus (one Shortcuts section, no per-screen list, no "?"), the app text
// comes last; Config has Banner as an indented sub-heading of Theme with a
// Text field, a Doctor section and lowercase shortcuts; the Log menu says
// Clear; the selected menu entry is bold.
func TestHelpFromMenusAndDoctor(t *testing.T) {
	m := testModel(t)
	m, _ = press(t, m, "down", "down", "h") // leave the text fields first
	v := plain(m.View())
	for _, bad := range []string{"Other screens", "Navigation", "Everywhere", "h, ?, F1", "Ctrl+S"} {
		if strings.Contains(v, bad) {
			t.Errorf("help must not contain %q", bad)
		}
	}
	for _, want := range []string{"Shortcuts", "Delete", "Doctor (Config)", "^s", "^d", "Refresh", "Clear (Log)", "h", "Commands", "Aliases", "Integrations"} {
		if !strings.Contains(v, want) {
			t.Errorf("help lacks %q", want)
		}
	}
	if strings.Index(v, "Aliases") < strings.Index(v, "Commands") {
		t.Error("application text must come after shortcuts and commands")
	}
	m, _ = press(t, m, "esc")
	for key, want := range map[string]screen{"o": screenConfig, "l": screenLog, "h": screenHelp} {
		mm, _ := press(t, m, "down", "down", key)
		if mm.cur != want {
			t.Errorf("key %q: screen %v, want %v", key, mm.cur, want)
		}
	}
	m, _ = press(t, m, "down", "down", "o", "ctrl+d")
	if m.cur != screenDoctor || !strings.Contains(m.doctorText, "config") {
		t.Fatalf("ctrl+d from Config: cur=%v text=%q", m.cur, m.doctorText)
	}
	if !strings.Contains(plain(m.View()), "Checks") {
		t.Error("doctor screen lacks the checklist")
	}
	m, _ = press(t, m, "esc")
	if m.cur != screenConfig || m.form == nil {
		t.Fatalf("Back from Doctor must return to Config with its form, got %v", m.cur)
	}
}

func TestConfigBannerSubheadingAndLogClear(t *testing.T) {
	m := testModel(t)
	m, _ = press(t, m, "down", "down", "o")
	v := plain(m.View())
	for _, want := range []string{"Save ^s", "Doctor ^d", "Help h", "\n" + indent + "  Banner\n", "Text:", "Font:", "Run doctor"} {
		if !strings.Contains(v, want) {
			t.Errorf("config lacks %q", want)
		}
	}
	if strings.Contains(v, "\n"+indent+"Banner\n") || strings.Contains(v, "^S") || strings.Contains(v, "F1") {
		t.Error("Banner must be an indented sub-heading and shortcuts lowercase")
	}
	for i := range m.form.rows {
		if m.form.rows[i].key == "BANNER_TEXT" {
			m.form.rows[i].input.SetValue("HELLO")
		}
	}
	cfg := m.form.toConfig()
	if cfg.BannerText != "HELLO" {
		t.Fatalf("BannerText = %q", cfg.BannerText)
	}
	applyStyle(cfg)
	if b := plain(renderBannerLive("")); !strings.Contains(b, "█") {
		t.Errorf("custom banner text must be drawn with a font:\n%s", b)
	}
	m.form.resetTheme()
	if m.form.toConfig().BannerText != "" {
		t.Error("reset must clear the banner text")
	}
	m, _ = press(t, m, "esc", "down", "down", "l")
	if lv := plain(m.View()); !strings.Contains(lv, "Clear c") || strings.Contains(lv, "Clear log") {
		t.Error("log menu must say Clear")
	}
}

// The selected menu entry is never bold or recoloured as a whole: with
// MENU_EFFECT=glitter a glint (accent, bold) sweeps across its label while the
// menu has focus; with "none" the label stays plain. The tick chain runs only
// while the menu has focus.
func TestMenuGlitter(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(termenv.Ascii)
	defer func() { menuFrame = 0 }()
	m := testModel(t)
	applyStyle(m.cfg)
	m, cmd := press(t, m, "tab")
	if cmd == nil || !m.menuTicking {
		t.Fatal("tab into the menu must arm the menu tick")
	}
	menuLine := func(mm model) string { v := mm.View(); return v[strings.Index(v, "▶ "):] }
	menuFrame = 0
	off := menuLine(m)
	menuFrame = 2
	on := menuLine(m)
	if off == on || !strings.Contains(on, "\x1b[1;") {
		t.Fatalf("frame 2 must bold-accent one character of the label: %q", on)
	}
	mm, cmd := m.Update(menuTickMsg{})
	m = mm.(model)
	if cmd == nil || menuFrame != 3 {
		t.Fatal("tick while focused must advance the frame and continue")
	}
	m, _ = press(t, m, "tab")
	mm, cmd = m.Update(menuTickMsg{})
	m = mm.(model)
	if cmd != nil || m.menuTicking {
		t.Fatal("tick after the menu lost focus must end the chain")
	}
	m.cfg.MenuEffect = "none"
	applyStyle(m.cfg)
	m, _ = press(t, m, "tab")
	menuFrame = 2
	v := menuLine(m)
	if lbl := v[strings.Index(v, "D"):]; strings.Contains(lbl[:40], "\x1b[1;") {
		t.Fatalf("MENU_EFFECT=none must not bold the label: %q", lbl[:40])
	}
}
