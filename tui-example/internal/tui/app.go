// Package tui is the bubbletea front-end, built on the TUI Scaffold standard v2
// (same grammar as gu-vpn-pid): one Elm-style model with a screen enum. app.go
// owns the model, key dispatch, the shared frame (banner · title · body ·
// status line · one-line menu) and the generic screens Log, Help and Confirm;
// home.go and form.go (Config) are the app-specific ones.
package tui

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type screen int

const (
	screenHome screen = iota
	screenConfig
	screenLog
	screenHelp
	screenConfirm
	screenDoctor // the `doctor` checklist, run from Config
)

type model struct {
	cur    screen
	cfg    *Config
	width  int
	height int

	statusMsg string // one-line message above the menu (✓ / ✗ / neutral)

	// bottom menu of Home/Log/Help/Confirm (Config has its own in form)
	menuActive bool
	btn        int
	scroll     int // first body line of a screen taller than the terminal

	home homeState
	form *form

	// back-navigation targets
	helpReturn, logReturn, confirmReturn screen
	confirmAction, confirmText           string
	logLines                             []string
	doctorReturn                         screen
	doctorText                           string
	menuTicking                          bool // a menuTick chain is running
}

func initialModel() model {
	cfg, _, err := Load()
	if cfg == nil {
		cfg = Default()
	}
	applyStyle(cfg)
	m := model{cfg: cfg, cur: screenHome}
	m.home = newHome()
	if err != nil {
		m.statusMsg = "Config could not be read: " + err.Error()
	}
	return m
}

func (m model) Init() tea.Cmd { return tea.Batch(loadItemsCmd(), textinput.Blink) }

// --- messages ---

type editorDoneMsg struct{ err error }

// menuTickMsg advances the glitter sweep over the selected menu entry while
// the menu has focus (MENU_EFFECT=glitter); the chain ends itself when focus
// leaves the menu.
type menuTickMsg struct{}

func menuTickCmd() tea.Cmd {
	return tea.Tick(90*time.Millisecond, func(time.Time) tea.Msg { return menuTickMsg{} })
}

// menuHasFocus reports whether the bottom menu of the current screen has focus.
func (m model) menuHasFocus() bool {
	if m.cur == screenConfig && m.form != nil {
		return m.form.inButtons
	}
	return m.menuActive
}

// armMenuTick starts the glitter tick when the menu just got focus.
func (m model) armMenuTick(cmd tea.Cmd) (tea.Model, tea.Cmd) {
	if uiMenuEffect == "glitter" && m.menuHasFocus() && !m.menuTicking {
		m.menuTicking = true
		return m, tea.Batch(cmd, menuTickCmd())
	}
	return m, cmd
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		termWidth = msg.Width
		return m, nil
	case itemsMsg:
		return m.onItems(msg)
	case editorDoneMsg:
		m = m.reload()
		if msg.err != nil {
			m.statusMsg = "Editor failed: " + msg.err.Error()
		}
		return m, nil
	case tea.KeyMsg:
		mm, cmd := m.onKey(msg)
		if t, ok := mm.(model); ok {
			return t.armMenuTick(cmd)
		}
		return mm, cmd
	case menuTickMsg:
		if uiMenuEffect == "glitter" && m.menuHasFocus() {
			menuFrame++
			return m, menuTickCmd()
		}
		m.menuTicking = false
		return m, nil
	}
	if m.cur == screenHome {
		return m.homeForward(msg)
	}
	return m, nil
}

// onKey: Ctrl+C always quits; the Config-screen shortcuts work everywhere;
// then the screen's own handler.
func (m model) onKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "ctrl+c":
		return m, tea.Quit
	case "ctrl+l":
		return m.run("log")
	case "ctrl+e":
		return m.run("edit-config")
	case "ctrl+r":
		return m.run("reload")
	case "ctrl+t":
		return m.run("theme-reset")
	case "ctrl+u":
		return m.run("uninstall")
	case "ctrl+d":
		return m.run("doctor")
	}
	switch m.cur {
	case screenConfig:
		return m.updateConfig(k)
	case screenHome:
		return m.updateHome(k)
	}
	return m.updateSimple(k)
}

// updateSimple handles Log, Help and Confirm: menu navigation + shortcuts.
func (m model) updateSimple(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	btns := m.screenButtons()
	switch k.String() {
	case "tab":
		m.menuActive = !m.menuActive
		m.btn = 0
		if !m.menuActive {
			m.scroll = 0
		}
		return m, nil
	case "down":
		if !m.menuActive {
			if mm, moved := m.scrollBy(1); moved {
				return mm, nil
			}
			m.menuActive, m.btn = true, 0
		}
		return m, nil
	case "up", "shift+tab":
		if m.menuActive {
			m.menuActive = false
		} else {
			m, _ = m.scrollBy(-1)
		}
		return m, nil
	case "left", "right":
		if !m.menuActive {
			m.menuActive, m.btn = true, 0
			return m, nil
		}
		d := 1
		if k.String() == "left" {
			d = -1
		}
		m.btn = nextIdx(m.btn, d, len(btns))
		return m, nil
	case "enter":
		if m.menuActive {
			return m.run(btns[m.btn].action)
		}
		return m, nil
	case "q", "esc":
		return m.run("back")
	}
	for _, b := range btns {
		if b.key == k.String() {
			return m.run(b.action)
		}
	}
	if k.String() == "h" {
		return m.run("help")
	}
	return m, nil
}

// screenButtons is the bottom menu of Log/Help/Confirm (Help is always last).
func (m model) screenButtons() []button {
	switch m.cur {
	case screenLog:
		return []button{{"Refresh", "log-refresh", "r"}, {"Clear", "log-clear", "c"}, {"Back", "back", "esc"}, {"Help", "help", "h"}}
	case screenDoctor:
		return []button{{"Refresh", "doctor-refresh", "r"}, {"Back", "back", "esc"}, {"Help", "help", "h"}}
	case screenHelp:
		return []button{{"Back", "back", "esc"}}
	case screenConfirm:
		return []button{{"Yes", "confirm-yes", "y"}, {"No", "back", "n"}}
	}
	return homeButtons
}

// run performs an action; "back" is what q/Esc do on every screen.
func (m model) run(action string) (tea.Model, tea.Cmd) {
	m.menuActive, m.scroll = false, 0
	switch action {
	case "back":
		switch m.cur {
		case screenHome:
			return m, tea.Quit
		case screenConfig:
			applyStyle(m.cfg) // undo a live theme preview
			m.form = nil
			m.cur = screenHome
			m.statusMsg = "Config unchanged"
		case screenHelp:
			m.cur = m.helpReturn
		case screenLog:
			m.cur = m.logReturn
		case screenDoctor:
			m.cur = m.doctorReturn
		case screenConfirm:
			m.cur = m.confirmReturn
			m.statusMsg = "Cancelled"
		}
		return m, textinput.Blink
	case "config":
		m.form = newForm(m.cfg)
		m.cur = screenConfig
		m.statusMsg = ""
		return m, textinput.Blink
	case "save":
		cfg := m.form.toConfig()
		if err := Save(cfg); err != nil {
			m.statusMsg = "Save failed: " + err.Error()
			return m, nil
		}
		m.cfg = cfg
		applyStyle(cfg)
		m.form, m.cur = nil, screenHome
		m.statusMsg = "Config saved"
		if s := syncAliases(cfg); s != "" {
			m.statusMsg += " · " + s
		}
		logf("config saved")
		return m, nil
	case "help":
		if m.cur != screenHelp {
			m.helpReturn = m.cur
		}
		m.cur = screenHelp
		return m, nil
	case "log", "log-refresh":
		if m.cur != screenLog {
			m.logReturn = m.cur
		}
		m.cur = screenLog
		m.logLines = tailLines(logPath(), 200)
		return m, nil
	case "doctor", "doctor-refresh":
		if m.cur != screenDoctor {
			m.doctorReturn = m.cur
		}
		var b bytes.Buffer
		doctor(&b)
		m.doctorText, m.cur, m.statusMsg = b.String(), screenDoctor, ""
		return m, nil
	case "log-clear":
		_ = atomicWrite(logPath(), nil)
		m.logLines = nil
		m.statusMsg = "Log cleared"
		return m, nil
	case "edit-config":
		c, err := editorCmd()
		if err != nil {
			m.statusMsg = "Edit failed: " + err.Error()
			return m, nil
		}
		return m, tea.ExecProcess(c, func(err error) tea.Msg { return editorDoneMsg{err} })
	case "reload":
		return m.reload(), textinput.Blink
	case "theme-reset":
		if m.cur == screenConfig { // preview; Save keeps it
			m.form.resetTheme()
			m.statusMsg = "Theme reset to default — Save to keep it"
			return m, nil
		}
		m.cfg.resetTheme()
		applyStyle(m.cfg)
		if err := Save(m.cfg); err != nil {
			m.statusMsg = "Theme reset failed: " + err.Error()
		} else {
			m.statusMsg = "Theme reset to default and saved"
		}
		return m, nil
	case "uninstall":
		return m.askConfirm("uninstall", "Uninstall "+AppName+"? Removes the binary and the shell alias (config is kept)."), nil
	case "confirm-yes":
		switch m.confirmAction {
		case "uninstall":
			uninstall()
			return m, tea.Quit
		case "delete":
			return m.deleteSelected()
		}
	}
	return m.homeAction(action)
}

func (m model) reload() model {
	cfg, _, err := Load()
	if err != nil {
		m.statusMsg = "Reload failed: " + err.Error()
		return m
	}
	m.cfg = cfg
	applyStyle(cfg)
	if m.cur == screenConfig {
		m.form = newForm(cfg)
	}
	m.statusMsg = "Config reloaded from disk"
	return m
}

func (m model) askConfirm(action, text string) model {
	if m.cur != screenConfirm {
		m.confirmReturn = m.cur
	}
	m.cur, m.confirmAction, m.confirmText = screenConfirm, action, text
	m.menuActive, m.btn = true, 1 // "No" preselected
	return m
}

// --- frame ---

// liveLine is the info block directly under the banner (LIVE_INFO):
// full = items · search · host IP, compact = items, off = nothing. Icons in
// the accent, values in COLOR_LIVE, separators in COLOR_RULE.
func (m model) liveLine() string {
	const icList, icSearch, icHost = "\U000F0279", "\U000F0349", "\U000F0317" // md-format_list_bulleted, md-magnify, md-lan
	// Shown only while there is live data (the demo: items exist — gu: the
	// tunnel is up). Icons, no words (Help → Icons explains them), no bold.
	if uiLiveInfo == "off" || len(m.home.items) == 0 {
		return ""
	}
	icon := func(ic string) string { return lipgloss.NewStyle().Foreground(AccentColor).Render(ic) }
	parts := []string{icon(icList) + " " + liveStyle.Render(fmt.Sprintf("%d", len(m.home.items)))}
	if uiLiveInfo == "full" {
		if q := m.home.search.Value(); q != "" {
			parts = append(parts, icon(icSearch)+" "+liveStyle.Render(q))
		}
		parts = append(parts, icon(icHost)+" "+liveStyle.Render(firstNonEmpty(hostIPv4(), "—")))
	}
	return fitLine(indent + strings.Join(parts, "   "))
}

// hostIPv4 is the machine's outgoing IPv4: the source address the kernel picks
// for a route to the internet (a UDP "connect" sends no packet). Asking the
// routing table — not "the first interface that is up" — avoids picking
// podman0/docker0/tailscale0. Cached for 30 s (the view renders every second).
func hostIPv4() string {
	hostIPMu.Lock()
	defer hostIPMu.Unlock()
	if time.Since(hostIPAt) < 30*time.Second {
		return hostIPVal
	}
	hostIPVal, hostIPAt = "", time.Now()
	if c, err := net.Dial("udp4", "192.0.2.1:9"); err == nil {
		if a, ok := c.LocalAddr().(*net.UDPAddr); ok {
			hostIPVal = a.IP.String()
		}
		c.Close()
	}
	return hostIPVal
}

var (
	hostIPMu  sync.Mutex
	hostIPAt  time.Time
	hostIPVal string
)

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

func (m model) banner() string { return renderBannerLive(m.liveLine()) }

// tail is the status message + the one-line menu under every screen.
func (m model) tail(btns []button, sel int, active bool) string {
	var t strings.Builder
	if m.statusMsg != "" {
		t.WriteString("\n" + flashLine(m.width, classifyMsg(m.statusMsg), m.statusMsg) + "\n")
	}
	t.WriteString("\n" + footerRule() + "\n" + renderMenu(btns, sel, active, m.width) + "\n")
	return t.String()
}

// room is how many body lines fit between head and tail (0 = no limit).
func (m model) room(head, tail string) int {
	if m.height <= 0 {
		return 0
	}
	return m.height - strings.Count(head, "\n") - strings.Count(tail, "\n") - 1
}

// frame renders a scrollable screen.
func (m model) frame(title, body string) string {
	head := m.banner() + "\n\n" + screenTitle(title) + "\n"
	tail := m.tail(m.screenButtons(), m.btn, m.menuActive)
	lines := strings.Split(strings.TrimSuffix(body, "\n"), "\n")
	return head + strings.Join(windowAt(lines, m.scroll, m.room(head, tail)), "\n") + "\n" + tail
}

func (m model) body() (string, string) {
	switch m.cur {
	case screenLog:
		return "Log", m.bodyLog()
	case screenHelp:
		return "Help", m.bodyHelp()
	case screenDoctor:
		return "Doctor", m.bodyDoctor()
	case screenConfirm:
		return "Confirm", "\n" + wrapText(m.width, fgStyle, m.confirmText) + "\n"
	}
	return "Home", m.bodyHome()
}

// scrollBy moves the body window; false at the end (↓ then enters the menu).
func (m model) scrollBy(d int) (model, bool) {
	title, body := m.body()
	head := m.banner() + "\n\n" + screenTitle(title) + "\n"
	tail := m.tail(m.screenButtons(), m.btn, m.menuActive)
	n := strings.Count(strings.TrimSuffix(body, "\n"), "\n") + 1
	next := m.scroll + d
	if m.scroll == 0 && d > 0 {
		next = 2 // skip the step that only swaps line 0 for the "↑ 1 more" marker
	} else if m.scroll == 2 && d < 0 {
		next = 0
	}
	next = min(max(next, 0), maxScroll(n, m.room(head, tail)))
	if next == m.scroll {
		return m, false
	}
	m.scroll = next
	return m, true
}

func (m model) View() string {
	if m.cur == screenConfig {
		return m.viewConfig()
	}
	if m.cur == screenHome {
		return m.viewHome()
	}
	return m.frame(m.body())
}

func classifyMsg(msg string) msgKind {
	l := strings.ToLower(msg)
	for _, w := range []string{"failed", "could not", "error"} {
		if strings.Contains(l, w) {
			return msgErr
		}
	}
	for _, w := range []string{"saved", "reloaded", "cleared", "added", "deleted", "installed", "removed", "reset"} {
		if strings.Contains(l, w) {
			return msgOK
		}
	}
	return msgInfo
}

func (m model) bodyLog() string {
	var b strings.Builder
	b.WriteString("\n" + fitLine(sectionHeader("Log")+"  "+mutedStyle.Render(displayPath(logPath()))) + "\n")
	if len(m.logLines) == 0 {
		b.WriteString(indent + "  " + mutedStyle.Render("(empty)") + "\n")
	}
	for _, l := range m.logLines {
		st := fgStyle
		if ll := strings.ToLower(l); strings.Contains(ll, "error") || strings.Contains(ll, "fail") {
			st = errStyle
		}
		b.WriteString(fitLine(indent+"  "+st.Render(l)) + "\n")
	}
	return b.String()
}

// helpShortcutRows is ONE general shortcut list: the navigation keys plus the
// Home and Config menus exactly as they are defined (homeButtons / configMenu)
// — nothing hand-written that could drift from the keys.
func helpShortcutRows() [][2]string {
	rows := [][2]string{
		{"Tab, ↓", "into the menu (first entry selected)"},
		{"← / →", "between menu entries · change a choice in Config"},
		{"Tab, ↑", "out of the menu, back to the top"},
		{"↑ / ↓", "rows · scroll · fields in Config"},
		{"Enter", "run the selected entry · add an item · save in Config"},
		{"q, Esc", "back — quit on Home (q types inside a text field)"},
		{"Ctrl+C", "quit"},
	}
	rows = append(rows, shortcutRows([]screenMenu{
		{"Home", homeButtons},
		{"Log", (model{cur: screenLog}).screenButtons()},
		{"Doctor", (model{cur: screenDoctor}).screenButtons()},
		{"Confirm", (model{cur: screenConfirm}).screenButtons()},
		{"Config", configMenu},
	})...)
	return append(rows, [2]string{"^l ^e ^r ^t ^u ^d", "also from Home: log · edit config · reload · reset theme · uninstall · doctor"})
}

func (m model) bodyHelp() string {
	var b strings.Builder
	sec := func(title string, rows [][2]string) {
		b.WriteString("\n" + sectionHeader(title) + "\n")
		for _, r := range rows {
			b.WriteString(infoLine(r[0], fgStyle.Render(r[1])) + "\n")
		}
	}
	para := func(title, text string) {
		b.WriteString("\n" + sectionHeader(title) + "\n")
		b.WriteString(ind(wrapText(m.width-len(indent)-2, fgStyle, text)) + "\n")
	}
	bin := binName()
	sec("About", [][2]string{
		{"Version", AppName + " v" + Version + "   " + runtime.Version()},
		{"Repo", RepoURL},
		{"Config", displayPath(ConfigPath())},
		{"Log", displayPath(logPath())},
	})
	sec("Icons (under the banner, while there are items)", [][2]string{
		{"\U000F0279  Items", "number of items"},
		{"\U000F0349  Search", "the current search"},
		{"\U000F0317  Host", "this machine's LAN address"},
	})
	sec("Shortcuts", helpShortcutRows())
	sec("Commands", [][2]string{
		{bin + " tui", "this interface (default)"},
		{bin + " install", "launcher, shell alias, config file in the current format"},
		{bin + " doctor", "diagnostics (also Config → Doctor)"},
		{bin + " uninstall", "remove launcher + alias"},
		{bin + " version", "version"},
		{bin + " help", "CLI help"},
	})
	// The application text comes LAST: what the tool's aliases, integrations and
	// backends are and how they are used. Keep every sentence true to the code.
	para("Aliases", "`"+bin+"` is a shell alias for the launcher in ~/.local/bin (Config → Shell shows where it is "+
		"defined; SHELL_ALIASES=false removes it).")
	para("Integrations", "TODO("+AppName+"): describe each integration and backend here — what it does, where its files live "+
		"and which config key turns it on (gu: Remmina GU, Bitwarden, KeePass).")
	return b.String()
}

// bodyDoctor shows the captured `doctor` checklist.
func (m model) bodyDoctor() string {
	var b strings.Builder
	b.WriteString("\n" + sectionHeader("Checks") + "\n")
	for _, l := range strings.Split(strings.TrimRight(m.doctorText, "\n"), "\n") {
		st := fgStyle
		if strings.Contains(l, "✗") {
			st = errStyle
		}
		b.WriteString(fitLine(indent+st.Render(strings.TrimRight(l, " "))) + "\n")
	}
	return b.String()
}

// Run starts the TUI. Entry point from Main().
func Run(stdout, stderr io.Writer) int {
	logf("start %s v%s", AppName, Version)
	if _, err := tea.NewProgram(initialModel(), tea.WithOutput(stdout), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	return 0
}
