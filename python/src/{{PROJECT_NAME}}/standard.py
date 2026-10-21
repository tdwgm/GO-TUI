"""TUI Scaffold standard v2.1 — theme + layout grammar (Python port of the Go
template's style.go/layout.go/banner.go; same look as gu-vpn-pid).

All functions return Rich markup. Rules: every text line is indented, ONE dotted
rule (under the banner, column 0), no footer, tuta-cli rows, one-line bottom
menu "▶ Label key · … · Help h", colours only from the theme.
"""

from __future__ import annotations

import textwrap

from rich.markup import escape
from rich.text import Text

from . import font, meta

INDENT = "  "
LABEL_WIDTH = 18
RULE_WIDTH = 44

ACCENT_NAMES = ["blue", "green", "red", "yellow", "magenta", "cyan", "orange", "purple"]
COLOR_NAMES = ["accent", "default", "blue", "ocean", "green", "red", "yellow", "magenta", "cyan",
               "orange", "purple", "grey", "white"]
NAMED = {
    "blue": "#ACC7FF", "ocean": "#7AA2F7", "green": "#88CC88", "red": "#FF7A7A", "yellow": "#F3F99D",
    "magenta": "#FF79C6", "cyan": "#8BE9FD", "orange": "#FFB86C", "purple": "#BD93F9",
    "grey": "#888888", "white": "#EEEEEE",
}


class Theme:
    """Live colours; rebuilt by apply() from the config (live preview in Config)."""

    accent = NAMED["blue"]
    banner = NAMED["blue"]
    header = NAMED["ocean"]
    text = NAMED["white"]
    muted = NAMED["grey"]
    danger = NAMED["red"]
    shortcut = NAMED["grey"]
    selected = ""
    rule = NAMED["grey"]
    subtitle = NAMED["white"]
    live = ""
    banner_font = "default"
    banner_text = ""
    live_info = "full"
    menu_effect = "glitter"
    frame = 0  # advanced by the app's menu tick while the menu has focus

    @classmethod
    def apply(cls, cfg: dict) -> None:
        cls.accent = NAMED.get(cfg.get("THEME", "blue"), NAMED["blue"])

        def pick(name: str, default: str) -> str:
            if name == "accent":
                return cls.accent
            if name == "default":
                return ""
            return NAMED.get(name, default)

        cls.banner = pick(cfg.get("COLOR_BANNER", "accent"), cls.accent)
        cls.header = pick(cfg.get("COLOR_HEADER", "ocean"), NAMED["ocean"])
        cls.text = pick(cfg.get("COLOR_TEXT", "white"), NAMED["white"])
        cls.muted = pick(cfg.get("COLOR_MUTED", "grey"), NAMED["grey"])
        cls.shortcut = pick(cfg.get("COLOR_SHORTCUT", "grey"), NAMED["grey"])
        cls.selected = pick(cfg.get("COLOR_SELECTED", "default"), "")
        cls.rule = pick(cfg.get("COLOR_RULE", "grey"), NAMED["grey"])
        cls.subtitle = pick(cfg.get("COLOR_SUBTITLE", "white"), NAMED["white"])
        cls.live = pick(cfg.get("COLOR_LIVE", "default"), "")
        cls.banner_font = cfg.get("BANNER_FONT", "default")
        cls.banner_text = cfg.get("BANNER_TEXT", "")
        cls.live_info = cfg.get("LIVE_INFO", "full")
        cls.menu_effect = cfg.get("MENU_EFFECT", "glitter")


def st(style: str, s: str) -> str:
    """Wrap escaped text s in a style ("" = no style)."""
    s = escape(s)
    return f"[{style}]{s}[/]" if style else s


def fg(s: str) -> str: return st(Theme.text, s)
def muted(s: str) -> str: return st(Theme.muted, s)
def key(s: str) -> str: return st(f"bold {Theme.accent}", s)
def ok(s: str) -> str: return st(Theme.header, s)
def err(s: str) -> str: return st(Theme.danger, s)
def sub(s: str) -> str: return st(Theme.subtitle, s)
def live_val(s: str) -> str: return st(Theme.live, s)
def sep_rule(s: str) -> str: return st(Theme.rule, s)


def pad(s: str, n: int) -> str:
    return s + " " * max(0, n - len(s))


def fit(line: str, width: int) -> str:
    """Cut a markup line to the terminal width with … (long paths)."""
    if width <= 0:
        return line
    t = Text.from_markup(line)
    if t.cell_len <= width - 2:
        return line
    t.truncate(width - 2, overflow="ellipsis")
    return t.markup


def rule() -> str:
    return st(f"dim {Theme.rule}", "·" * RULE_WIDTH)


def title(name: str) -> str:
    return INDENT + st(f"bold {Theme.accent}", f"{meta.APP_NAME} › {name}")


def header(label: str) -> str:
    return INDENT + ok(label)


def subheader(label: str) -> str:
    """An indented heading inside a section (Theme → Banner)."""
    return INDENT + "  " + ok(label)


def row(focused: bool, label: str, value: str, width: int) -> str:
    if focused:
        return fit(INDENT + key("▶ ") + key(pad(label + ":", LABEL_WIDTH)) + " " + value, width)
    return fit(INDENT + "  " + fg(pad(label + ":", LABEL_WIDTH)) + " " + value, width)


def info(label: str, value: str, width: int) -> str:
    """A read-only row: label and value plain text; an unused or unset value is
    a muted "—" (value_or_dash) — never a mix of styles inside one value."""
    return fit(INDENT + "  " + fg(pad(label + ":", LABEL_WIDTH)) + " " + value, width)


def value_or_dash(v: str) -> str:
    return fg(v) if v.strip() else muted("—")


def wrap(text: str, width: int, style=muted) -> str:
    w = max(40, (width or 100) - len(INDENT) - 2)
    return "\n".join(INDENT + style(l) for l in textwrap.wrap(text, w))


def glitter_label(label: str) -> str:
    """The selected menu label (MENU_EFFECT): with "glitter" a glint — one character
    in the accent, bold, with a soft accent edge on each side — sweeps across it
    while the menu has focus (Theme.frame advances every ~90 ms); with "none" the
    label is plain. Never bold as a whole, never a colour change."""
    if Theme.menu_effect != "glitter" or not label:
        return st(Theme.selected, label)
    pos = Theme.frame % (len(label) + 4) - 2
    out = []
    for i, ch in enumerate(label):
        if i == pos:
            out.append(st(f"bold {Theme.accent}", ch))
        elif abs(i - pos) == 1:
            out.append(st(Theme.accent, ch))
        else:
            out.append(st(Theme.selected, ch))
    return "".join(out)


def menu(buttons: list[tuple[str, str, str]], sel: int, active: bool, width: int) -> str:
    """ONE line: '▶ Label key · Label key …' (label, action, key) — labels plain,
    the selected one with the glitter sweep (glitter_label), shortcuts in COLOR_SHORTCUT."""
    limit = (width or 100) - len(INDENT) - 1
    lines, cur, w = [], "", 0
    for i, (label, _action, k) in enumerate(buttons):
        selected = active and i == sel
        cell = (glitter_label(label) if selected else fg(label)) + (" " + st(Theme.shortcut, k) if k else "")
        cw = len(label) + (1 + len(k) if k else 0)
        if selected:
            cell, cw = key("▶ ") + cell, cw + 2
        elif w == 0:
            cell, cw = "  " + cell, cw + 2
        if w and w + 3 + cw > limit:
            lines.append(INDENT + cur)
            cur, w = "", 0
            if not selected:
                cell, cw = "  " + cell, cw + 2
        if w:
            cur += st(Theme.shortcut, " · ")
            w += 3
        cur += cell
        w += cw
    if cur:
        lines.append(INDENT + cur)
    return "\n".join(lines)


def flash(msg: str, width: int) -> str:
    l = msg.lower()
    if any(w in l for w in ("failed", "could not", "error")):
        return wrap("✗ " + msg, width, err)
    if any(w in l for w in ("saved", "reloaded", "cleared", "added", "deleted", "installed", "removed", "reset")):
        return wrap("✓ " + msg, width, ok)
    return wrap(msg, width, muted)


def window_at(lines: list[str], start: int, room: int) -> list[str]:
    """Show lines from start in room rows with muted ↑/↓ n more markers."""
    if room <= 0 or len(lines) <= room:
        return lines
    room = max(room, 5)
    inner = room - 2
    start = min(max(start, 0), len(lines) - inner)
    end = start + inner
    out = []
    if start > 0:
        out.append(INDENT + muted(f"  ↑ {start} more"))
    else:
        end += 1
    end = min(end, len(lines))
    out += lines[start:end]
    if end < len(lines):
        out.append(INDENT + muted(f"  ↓ {len(lines) - end} more"))
    return out


def scroll_window(lines: list[str], focus: int, room: int) -> list[str]:
    if room <= 0 or len(lines) <= room:
        return lines
    if focus < 0:
        focus = len(lines) - 1
    return window_at(lines, focus - (max(room, 5) - 2) // 2, room)


def frakturify(s: str) -> str:
    out = []
    for c in s:
        if "a" <= c <= "z":
            out.append(chr(0x1D586 + ord(c) - ord("a")))
        elif "A" <= c <= "Z":
            out.append(chr(0x1D56C + ord(c) - ord("A")))
        else:
            out.append(c)
    return "".join(out)


def banner(live: str = "") -> str:
    """Logo with v<version> after its last line, description, byline, the rule,
    and (TUI) the live info block directly under the rule."""
    # BANNER_TEXT / BANNER_FONT: the default logo only shows the app name, so any
    # other text is drawn with the font (block when the font is "default").
    text, style = Theme.banner_text or meta.APP_NAME, Theme.banner_font
    if style == "default" and text != meta.APP_NAME:
        style = "block"
    art = font.render(text, style) or (meta.BANNER_ART or meta.APP_NAME).strip("\n").split("\n")
    lines = [INDENT + st(f"bold {Theme.banner}", l) for l in art]
    lines[-1] += "  " + sub("v" + meta.VERSION)
    out = "\n".join(lines) + "\n" + INDENT + sub(meta.APP_DESCRIPTION) + "\n\n"
    out += INDENT + sub(frakturify("by Tdwg Entertainment Solutions") + " ®") + "\n\n" + rule()
    if live:  # the live info block sits directly UNDER the rule
        out += "\n\n" + "\n".join(INDENT + l for l in live.split("\n"))
    return out
