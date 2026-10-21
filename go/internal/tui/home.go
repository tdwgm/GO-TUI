package tui

// home.go — the demo first screen: an Overview section and an Items section
// (add, live search, delete). Replace it with your app's own content; keep the
// grammar: sections + rows on top, the one-line menu at the bottom.

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// homeButtons is the Home menu (Help last).
var homeButtons = []button{
	{"Delete", "delete", "x"}, {"Config", "config", "o"}, {"Log", "log", "l"},
	{"Quit", "back", "q"}, {"Help", "help", "h"},
}

type homeState struct {
	items  []string
	newIn  textinput.Model
	search textinput.Model
	focus  int // 0 = New item, 1 = Search, 2.. = list row (index-2 into visible)
}

func newHome() homeState {
	n := textinput.New()
	n.Prompt, n.Placeholder = "", "type and press Enter"
	n.Focus()
	s := textinput.New()
	s.Prompt, s.Placeholder = "", "filter"
	return homeState{newIn: n, search: s}
}

type itemsMsg struct {
	items []string
	err   error
}

func loadItemsCmd() tea.Cmd {
	return func() tea.Msg {
		items, err := loadItems()
		return itemsMsg{items, err}
	}
}

func (m model) onItems(msg itemsMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.statusMsg = "Could not load items: " + msg.err.Error()
		logf("error loading items: %v", msg.err)
		return m, nil
	}
	m.home.items = msg.items
	return m, nil
}

// visible returns the items matching the search field.
func (h homeState) visible() []string {
	q := strings.ToLower(strings.TrimSpace(h.search.Value()))
	if q == "" {
		return h.items
	}
	var out []string
	for _, it := range h.items {
		if strings.Contains(strings.ToLower(it), q) {
			out = append(out, it)
		}
	}
	return out
}

func (h *homeState) setFocus(i int) {
	h.newIn.Blur()
	h.search.Blur()
	h.focus = i
	switch i {
	case 0:
		h.newIn.Focus()
	case 1:
		h.search.Focus()
	}
}

func (h homeState) typing() bool { return h.focus < 2 }

func (m model) updateHome(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	h := &m.home
	last := 1 + len(h.visible()) // last focusable row index
	switch k.String() {
	case "tab", "shift+tab":
		if m.menuActive {
			m.menuActive = false
			h.setFocus(0)
		} else {
			m.menuActive, m.btn = true, 0
			h.setFocus(-1)
		}
		return m, textinput.Blink
	case "down":
		if m.menuActive {
			return m, nil
		}
		if h.focus >= last {
			m.menuActive, m.btn = true, 0
			h.setFocus(-1)
		} else {
			h.setFocus(h.focus + 1)
		}
		return m, textinput.Blink
	case "up":
		if m.menuActive {
			m.menuActive = false
			h.setFocus(last)
		} else if h.focus > 0 {
			h.setFocus(h.focus - 1)
		}
		return m, textinput.Blink
	case "left", "right":
		if m.menuActive {
			d := 1
			if k.String() == "left" {
				d = -1
			}
			m.btn = nextIdx(m.btn, d, len(homeButtons))
			return m, nil
		}
	case "enter":
		if m.menuActive {
			return m.run(homeButtons[m.btn].action)
		}
		if h.focus == 0 {
			return m.addItem()
		}
		return m, nil
	case "esc":
		return m.run("back")
	}
	if !h.typing() || m.menuActive {
		switch k.String() {
		case "q":
			return m.run("back")
		case "?":
			return m.run("help")
		}
		for _, b := range homeButtons {
			if b.key == k.String() {
				return m.run(b.action)
			}
		}
		return m, nil
	}
	var cmd tea.Cmd
	if h.focus == 0 {
		h.newIn, cmd = h.newIn.Update(k)
	} else {
		h.search, cmd = h.search.Update(k)
	}
	return m, cmd
}

// homeForward passes non-key messages (cursor blink) to the focused field.
func (m model) homeForward(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.home.focus {
	case 0:
		m.home.newIn, cmd = m.home.newIn.Update(msg)
	case 1:
		m.home.search, cmd = m.home.search.Update(msg)
	}
	return m, cmd
}

func (m model) addItem() (tea.Model, tea.Cmd) {
	v := strings.TrimSpace(m.home.newIn.Value())
	if v == "" {
		return m, nil
	}
	m.home.items = append(m.home.items, v)
	if err := saveItems(m.home.items); err != nil {
		m.statusMsg = "Save failed: " + err.Error()
		return m, nil
	}
	m.home.newIn.SetValue("")
	m.statusMsg = "Added " + v
	logf("added item %q", v)
	return m, nil
}

// selected is the item under the cursor, if the cursor is on the list.
func (h homeState) selected() (string, bool) {
	vis := h.visible()
	if i := h.focus - 2; i >= 0 && i < len(vis) {
		return vis[i], true
	}
	return "", false
}

// homeAction runs the Home-specific menu actions.
func (m model) homeAction(action string) (tea.Model, tea.Cmd) {
	if action != "delete" {
		return m, nil
	}
	it, ok := m.home.selected()
	if !ok {
		m.statusMsg = "Select an item first (↓ into the list)"
		return m, nil
	}
	if m.cfg.ConfirmDelete {
		return m.askConfirm("delete", fmt.Sprintf("Delete %q?", it)), nil
	}
	return m.deleteSelected()
}

func (m model) deleteSelected() (tea.Model, tea.Cmd) {
	it, ok := m.home.selected()
	m.cur = screenHome
	if !ok {
		return m, nil
	}
	var keep []string
	for _, x := range m.home.items {
		if x != it {
			keep = append(keep, x)
		}
	}
	m.home.items = keep
	if err := saveItems(keep); err != nil {
		m.statusMsg = "Delete failed: " + err.Error()
		return m, nil
	}
	m.statusMsg = "Deleted " + it
	logf("deleted item %q", it)
	if m.home.focus-2 >= len(m.home.visible()) {
		m.home.setFocus(max(m.home.focus-1, 0))
	}
	return m, nil
}

func (m model) bodyHome() string {
	h := m.home
	var b strings.Builder
	b.WriteString("\n" + sectionHeader("Overview") + "\n")
	b.WriteString(infoLine("Greeting", valueOrDash(m.cfg.Greeting)) + "\n")
	b.WriteString(infoLine("Items", fgStyle.Render(fmt.Sprintf("%d · %s", len(h.items), itemsPath()))) + "\n")
	b.WriteString(infoLine("Config", fgStyle.Render(ConfigPath())) + "\n")
	b.WriteString("\n" + sectionHeader("Items") + "\n")
	b.WriteString(rowLine(!m.menuActive && h.focus == 0, "New item", h.newIn.View()) + "\n")
	b.WriteString(rowLine(!m.menuActive && h.focus == 1, "Search", h.search.View()) + "\n")
	vis := h.visible()
	if len(vis) == 0 {
		b.WriteString(indent + "  " + mutedStyle.Render("(no items)") + "\n")
	}
	// window the list around the cursor (ITEMS_SHOWN rows)
	n := m.cfg.ItemsShownInt()
	start := 0
	if c := h.focus - 2; c >= n {
		start = c - n + 1
	}
	end := min(start+n, len(vis))
	if start > 0 {
		b.WriteString(indent + mutedStyle.Render(fmt.Sprintf("  ↑ %d more", start)) + "\n")
	}
	for i := start; i < end; i++ {
		focused := !m.menuActive && h.focus-2 == i
		marker, txt := "  ", fgStyle.Render(vis[i])
		if focused {
			marker, txt = keyStyle.Render("▶ "), keyStyle.Render(vis[i])
		}
		b.WriteString(fitLine(indent+marker+txt) + "\n")
	}
	if end < len(vis) {
		b.WriteString(indent + mutedStyle.Render(fmt.Sprintf("  ↓ %d more", len(vis)-end)) + "\n")
	}
	return b.String()
}

func (m model) viewHome() string {
	head := m.banner() + "\n\n" + screenTitle("Home") + "\n"
	return head + m.bodyHome() + m.tail(homeButtons, m.btn, m.menuActive)
}

// valueOrDash renders a value in the text colour, or a muted "—" when it is
// unset — never a mix of styles inside one value.
func valueOrDash(v string) string {
	if strings.TrimSpace(v) == "" {
		return mutedStyle.Render("—")
	}
	return fgStyle.Render(v)
}
