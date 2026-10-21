"""The TUI as a pure model (Elm-style, like the Go template): key(k) changes
state, view() returns Rich markup. app.py only bridges it to Textual, so the
whole behaviour is testable without a terminal (tests/test_model.py).

Screens: home, config, log, help, confirm. Keys (standard v2.1): Tab/↓ into
the one-line menu (first entry), ←/→ in the menu, Tab/↑ out; q ≡ Esc = back
(quit on Home; q types inside a text field); Config's Ctrl shortcuts
(^S ^L ^E ^R ^T ^U) and F1 work everywhere; uninstall goes through Confirm.
"""

from __future__ import annotations

import json
import os
import platform
from pathlib import Path

from . import config as cfgmod
from . import meta, system
from .font import BANNER_FONTS
from .standard import (ACCENT_NAMES, COLOR_NAMES, INDENT, Theme, banner, err, fg, flash, header,
                       info, key, live_val, menu, muted, row, scroll_window, st, subheader,
                       title, value_or_dash, window_at, wrap)

HOME_MENU = [("Delete", "delete", "x"), ("Config", "config", "o"), ("Log", "log", "l"),
             ("Quit", "back", "q"), ("Help", "help", "h")]
# Lowercase shortcuts like the Home menu. Help's shortcut list is generated
# from HOME_MENU and CONFIG_MENU, so it never drifts from the keys.
CONFIG_MENU = [("Save", "save", "^s"), ("Cancel", "back", "esc"), ("Doctor", "doctor", "^d"),
               ("Log", "log", "^l"), ("Edit", "edit-config", "^e"), ("Reload", "reload", "^r"),
               ("Reset theme", "theme-reset", "^t"), ("Uninstall", "uninstall", "^u"),
               ("Help", "help", "h")]
GLOBAL = {"ctrl+l": "log", "ctrl+e": "edit-config", "ctrl+r": "reload", "ctrl+d": "doctor",
          "ctrl+t": "theme-reset", "ctrl+u": "uninstall"}
YES_NO = ["no", "yes"]


def host_ipv4() -> str:
    """The machine's outgoing IPv4 (no packet is sent: UDP connect only picks a route)."""
    import socket
    try:
        with socket.socket(socket.AF_INET, socket.SOCK_DGRAM) as s:
            s.connect(("192.0.2.1", 9))
            return s.getsockname()[0]
    except OSError:
        return ""


def shortcut_rows(screens: list[tuple[str, list]]) -> list[tuple[str, str]]:
    """ONE truthful Help list keyed by shortcut, merged over the screens' menus: a key
    that means the same everywhere is listed once ("h: Help"); one that differs lists
    every meaning with its screens ("r: Refresh (Log, Doctor)"); one that exists on a
    single screen other than the first names it ("^s: Save (Config)"). q/Esc are left
    to the navigation rows."""
    order, seen = [], {}
    for name, btns in screens:
        for label, _a, k in btns:
            if k in ("", "q", "esc"):
                continue
            if k not in seen:
                order.append(k)
                seen[k] = []
            seen[k].append((label, name))
    rows = []
    for k in order:
        ms, key = seen[k], k.replace("/", " / ")
        labels = list(dict.fromkeys(l for l, _n in ms))
        if len(labels) == 1:
            if len(ms) == 1 and ms[0][1] != screens[0][0]:
                rows.append((key, f"{labels[0]} ({ms[0][1]})"))
            else:
                rows.append((key, labels[0]))
        else:
            rows.append((key, " · ".join(f"{l} ({', '.join(n for ll, n in ms if ll == l)})" for l in labels)))
    return rows


def items_path() -> Path:
    base = os.environ.get("XDG_DATA_HOME") or str(Path.home() / ".local" / "share")
    return Path(base) / meta.APP_NAME / "items.json"


def load_items() -> list[str]:
    p = items_path()
    return json.loads(p.read_text()) if p.exists() else ["alpha", "bravo", "charlie", "delta", "echo"]


def save_items(items: list[str]) -> None:
    p = items_path()
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(json.dumps(items, indent=2) + "\n")


class Form:
    """Config rows bound to config keys: (kind, label, key, value/choices)."""

    def __init__(self, cfg: dict):
        self.base = dict(cfg)
        self.rows: list[dict] = []
        add = self.rows.append
        add({"kind": "info", "label": "Config", "value": cfgmod.display_path(cfgmod.config_path())})
        add({"kind": "header", "label": "General"})
        add({"kind": "text", "label": "Greeting", "key": "GREETING", "value": cfg["GREETING"]})
        add({"kind": "text", "label": "Items shown", "key": "ITEMS_SHOWN", "value": cfg["ITEMS_SHOWN"]})
        add({"kind": "header", "label": "Behaviour"})
        add(self._choice("Confirm delete", "CONFIRM_DELETE", YES_NO, cfg))
        add({"kind": "header", "label": "Theme"})
        add(self._choice("Accent", "THEME", ACCENT_NAMES, cfg))
        for label, k in (("Banner", "COLOR_BANNER"), ("Headers", "COLOR_HEADER"), ("Text", "COLOR_TEXT"),
                         ("Muted text", "COLOR_MUTED"), ("Selected text", "COLOR_SELECTED"),
                         ("Shortcuts", "COLOR_SHORTCUT"), ("Rule", "COLOR_RULE"),
                         ("Under logo", "COLOR_SUBTITLE"), ("Live info text", "COLOR_LIVE")):
            add(self._choice(label, k, COLOR_NAMES, cfg))
        add(self._choice("Menu effect", "MENU_EFFECT", cfgmod.MENU_EFFECTS, cfg))
        add({"kind": "subheader", "label": "Banner"})
        add({"kind": "text", "label": "Text", "key": "BANNER_TEXT", "value": cfg.get("BANNER_TEXT", "")})
        add(self._choice("Font", "BANNER_FONT", BANNER_FONTS, cfg))
        add(self._choice("Live info", "LIVE_INFO", cfgmod.LIVE_INFO_MODES, cfg))
        add({"kind": "action", "label": "Reset to default", "action": "theme-reset"})
        if system.aliases_supported():  # OS-specific: only shown where it exists
            where = system.alias_where()
            add({"kind": "header", "label": "Shell"})
            add({"kind": "info", "label": f"alias {meta.APP_NAME}",
                 "value": "installed — " + " · ".join(where) if where else "not installed"})
            add(self._choice("Shell alias", "SHELL_ALIASES", YES_NO, cfg))
        add({"kind": "header", "label": "Doctor"})
        add({"kind": "action", "label": "Run doctor", "action": "doctor"})
        add({"kind": "header", "label": "Log"})
        add({"kind": "info", "label": "Log file", "value": cfgmod.display_path(system.log_path())})
        self.focus = self.selectable()[0]
        self.in_menu = False
        self.btn = 0

    @staticmethod
    def _choice(label, k, choices, cfg):
        v = cfg[k]
        if v in ("true", "false"):
            v = "yes" if v == "true" else "no"
        return {"kind": "choice", "label": label, "key": k, "choices": choices,
                "idx": choices.index(v) if v in choices else 0}

    def selectable(self) -> list[int]:
        return [i for i, r in enumerate(self.rows) if r["kind"] in ("text", "choice", "action")]

    def to_config(self) -> dict:
        out = dict(self.base)
        for r in self.rows:
            if r["kind"] == "text":
                out[r["key"]] = r["value"].strip()
            elif r["kind"] == "choice":
                v = r["choices"][r["idx"]]
                out[r["key"]] = {"yes": "true", "no": "false"}.get(v, v)
        return cfgmod.sanitize(out)

    def reset_theme(self) -> None:
        d = cfgmod.defaults()
        for r in self.rows:
            if r.get("key") in cfgmod.THEME_KEYS:
                if r["kind"] == "text":
                    r["value"] = d[r["key"]]
                else:
                    r["idx"] = r["choices"].index(d[r["key"]])
        Theme.apply(self.to_config())

    def typing(self) -> bool:
        return not self.in_menu and self.rows[self.focus]["kind"] == "text"


class Model:
    def __init__(self):
        self.cfg, _ = cfgmod.load()
        Theme.apply(self.cfg)
        self.state = "home"
        self.width = self.height = 0
        self.status = ""
        self.menu_active = False
        self.btn = 0
        self.scroll = 0
        self.items = load_items()
        self.new_text = ""
        self.search = ""
        self.focus = 0  # 0 = New item, 1 = Search, 2.. = list row
        self.form: Form | None = None
        self.help_return = self.log_return = self.confirm_return = self.doctor_return = "home"
        self.doctor_lines: list[str] = []
        self.confirm_action = self.confirm_text = ""
        self.log_lines: list[str] = []
        self.quit = False
        self.wants_editor = False  # app.py opens $EDITOR, then calls reload()

    # ---------- keys ----------

    def key(self, k: str) -> None:
        if k == "ctrl+c":
            self.quit = True
            return
        if k in GLOBAL and self.state != "confirm":
            self.run(GLOBAL[k])
            return
        {"home": self._key_home, "config": self._key_config}.get(self.state, self._key_simple)(k)

    def menu_has_focus(self) -> bool:
        """True while the bottom menu has focus (the app's glitter tick runs then)."""
        if self.state == "config" and self.form is not None:
            return self.form.in_menu
        return self.menu_active

    def _menu_nav(self, k: str, buttons) -> bool:
        if k in ("left", "right") and self.menu_active:
            self.btn = (self.btn + (1 if k == "right" else -1)) % len(buttons)
            return True
        return False

    def _key_simple(self, k: str) -> None:
        buttons = self.buttons()
        if self._menu_nav(k, buttons):
            return
        if k == "tab":
            self.menu_active, self.btn = not self.menu_active, 0
        elif k == "down":
            if not self.menu_active and not self._scroll_by(1):
                self.menu_active, self.btn = True, 0
        elif k in ("up", "shift+tab"):
            if self.menu_active:
                self.menu_active = False
            else:
                self._scroll_by(-1)
        elif k in ("left", "right"):
            self.menu_active, self.btn = True, 0
        elif k == "enter" and self.menu_active:
            self.run(buttons[self.btn][1])
        elif k in ("q", "escape"):
            self.run("back")
        else:
            for _l, action, sk in buttons:
                if sk == k:
                    self.run(action)
                    return
            if k == "h":
                self.run("help")

    def visible(self) -> list[str]:
        q = self.search.strip().lower()
        return [i for i in self.items if q in i.lower()] if q else list(self.items)

    def _key_home(self, k: str) -> None:
        last = 1 + len(self.visible())
        if self._menu_nav(k, HOME_MENU):
            return
        if k in ("tab", "shift+tab"):
            self.menu_active, self.btn = not self.menu_active, 0
            if not self.menu_active:
                self.focus = 0
        elif k == "down" and not self.menu_active:
            if self.focus >= last:
                self.menu_active, self.btn = True, 0
            else:
                self.focus += 1
        elif k == "up":
            if self.menu_active:
                self.menu_active, self.focus = False, last
            elif self.focus > 0:
                self.focus -= 1
        elif k == "enter":
            if self.menu_active:
                self.run(HOME_MENU[self.btn][1])
            elif self.focus == 0:
                self._add_item()
        elif k == "escape":
            self.run("back")
        elif self.focus < 2 and not self.menu_active:  # typing into New item / Search
            attr = "new_text" if self.focus == 0 else "search"
            cur = getattr(self, attr)
            if k == "backspace":
                setattr(self, attr, cur[:-1])
            elif k == "space":
                setattr(self, attr, cur + " ")
            elif len(k) == 1:
                setattr(self, attr, cur + k)
        else:
            if k == "q":
                self.run("back")
            elif k == "h":
                self.run("help")
            else:
                for _l, action, sk in HOME_MENU:
                    if sk == k:
                        self.run(action)

    def _key_config(self, k: str) -> None:
        f = self.form
        if k in ("tab", "shift+tab"):
            f.in_menu, f.btn = not f.in_menu, 0
            if not f.in_menu:
                f.focus = f.selectable()[0]
            return
        if f.in_menu and k in ("left", "right"):
            f.btn = (f.btn + (1 if k == "right" else -1)) % len(CONFIG_MENU)
            return
        sel = f.selectable()
        if k == "down":
            if f.in_menu:
                return
            nxt = [i for i in sel if i > f.focus]
            if nxt:
                f.focus = nxt[0]
            else:
                f.in_menu, f.btn = True, 0
            return
        if k == "up":
            if f.in_menu:
                f.in_menu, f.focus = False, sel[-1]
            else:
                prv = [i for i in sel if i < f.focus]
                if prv:
                    f.focus = prv[-1]
            return
        r = f.rows[f.focus]
        if k in ("left", "right", "space") and not f.typing():
            if r["kind"] == "choice":
                r["idx"] = (r["idx"] + (-1 if k == "left" else 1)) % len(r["choices"])
                if r["key"] in cfgmod.THEME_KEYS:
                    Theme.apply(f.to_config())  # live preview
            return
        if k == "escape" or (k == "q" and not f.typing()):
            self.run("back")
        elif k == "h" and not f.typing():
            self.run("help")
        elif k == "ctrl+s":
            self.run("save")
        elif k == "enter":
            if f.in_menu:
                self.run(CONFIG_MENU[f.btn][1])
            elif r["kind"] == "action":
                self.run(r["action"])
            else:
                self.run("save")
        elif f.typing():
            if k == "backspace":
                r["value"] = r["value"][:-1]
            elif k == "space":
                r["value"] += " "
            elif len(k) == 1:
                r["value"] += k
            if r["key"] == "BANNER_TEXT":
                Theme.apply(f.to_config())  # live preview

    # ---------- actions ----------

    def buttons(self, state: str | None = None):
        return {
            "home": HOME_MENU,
            "config": CONFIG_MENU,
            "log": [("Refresh", "log-refresh", "r"), ("Clear", "log-clear", "c"),
                    ("Back", "back", "esc"), ("Help", "help", "h")],
            "doctor": [("Refresh", "doctor-refresh", "r"), ("Back", "back", "esc"), ("Help", "help", "h")],
            "help": [("Back", "back", "esc")],
            "confirm": [("Yes", "confirm-yes", "y"), ("No", "back", "n")],
        }[state or self.state]

    def run(self, action: str) -> None:
        self.menu_active, self.scroll = False, 0
        if action == "back":
            if self.state == "home":
                self.quit = True
            elif self.state == "config":
                Theme.apply(self.cfg)  # undo a live preview
                self.state, self.form, self.status = "home", None, "Config unchanged"
            elif self.state == "help":
                self.state = self.help_return
            elif self.state == "log":
                self.state = self.log_return
            elif self.state == "doctor":
                self.state = self.doctor_return
            elif self.state == "confirm":
                self.state, self.status = self.confirm_return, "Cancelled"
        elif action == "config":
            self.form, self.state, self.status = Form(self.cfg), "config", ""
        elif action == "save":
            self.cfg = self.form.to_config()
            cfgmod.save(self.cfg)
            Theme.apply(self.cfg)
            self.state, self.form = "home", None
            self.status = "Config saved"
            if msg := system.sync_aliases(self.cfg):
                self.status += " · " + msg
            system.logf("config saved")
        elif action == "help":
            if self.state != "help":
                self.help_return = self.state
            self.state = "help"
        elif action in ("log", "log-refresh"):
            if self.state != "log":
                self.log_return = self.state
            self.state, self.log_lines = "log", system.tail()
        elif action in ("doctor", "doctor-refresh"):
            if self.state != "doctor":
                self.doctor_return = self.state
            self.doctor_lines, _bad = doctor_lines()
            self.state, self.status = "doctor", ""
        elif action == "log-clear":
            p = system.log_path()
            if p.exists():
                p.write_text("")
            self.log_lines, self.status = [], "Log cleared"
        elif action == "edit-config":
            self.wants_editor = True
        elif action == "reload":
            self.reload()
        elif action == "theme-reset":
            if self.state == "config":
                self.form.reset_theme()
                self.status = "Theme reset to default — Save to keep it"
            else:
                d = cfgmod.defaults()
                self.cfg.update({k: d[k] for k in cfgmod.THEME_KEYS})
                cfgmod.save(self.cfg)
                Theme.apply(self.cfg)
                self.status = "Theme reset to default and saved"
        elif action == "uninstall":
            self._ask("uninstall", f"Uninstall {meta.APP_NAME}? Removes the launcher and the shell alias (config is kept).")
        elif action == "confirm-yes":
            if self.confirm_action == "uninstall":
                system.uninstall()
                self.quit = True
            elif self.confirm_action == "delete":
                self._delete_selected()
        elif action == "delete":
            it = self._selected()
            if it is None:
                self.status = "Select an item first (↓ into the list)"
            elif cfgmod.truthy(self.cfg["CONFIRM_DELETE"]):
                self._ask("delete", f'Delete "{it}"?')
            else:
                self._delete_selected()

    def reload(self) -> None:
        self.cfg, _ = cfgmod.load()
        Theme.apply(self.cfg)
        if self.state == "config":
            self.form = Form(self.cfg)
        self.status = "Config reloaded from disk"

    def _ask(self, action: str, text: str) -> None:
        if self.state != "confirm":
            self.confirm_return = self.state
        self.state, self.confirm_action, self.confirm_text = "confirm", action, text
        self.menu_active, self.btn = True, 1  # "No" preselected

    def _selected(self):
        vis = self.visible()
        i = self.focus - 2
        return vis[i] if 0 <= i < len(vis) else None

    def _add_item(self) -> None:
        v = self.new_text.strip()
        if v:
            self.items.append(v)
            save_items(self.items)
            self.new_text, self.status = "", f"Added {v}"
            system.logf(f"added item {v!r}")

    def _delete_selected(self) -> None:
        it = self._selected()
        self.state = "home"
        if it is None:
            return
        self.items = [x for x in self.items if x != it]
        save_items(self.items)
        self.status = f"Deleted {it}"
        system.logf(f"deleted item {it!r}")
        self.focus = min(self.focus, 1 + len(self.visible()))

    # ---------- view ----------

    def live(self) -> str:
        """Info block under the banner (LIVE_INFO): full = items · search · host IP,
        compact = items, off = nothing. Shown only while there is live data (the demo:
        items exist — gu: the tunnel is up). Icons, no words (Help → Icons), no bold."""
        if Theme.live_info == "off" or not self.items:
            return ""
        def icon(ic: str) -> str: return st(Theme.accent, ic)
        parts = [icon("\U000F0279") + " " + live_val(str(len(self.items)))]  # md-format_list_bulleted
        if Theme.live_info == "full":
            if self.search:
                parts.append(icon("\U000F0349") + " " + live_val(self.search))  # md-magnify
            parts.append(icon("\U000F0317") + " " + live_val(host_ipv4() or "—"))  # md-lan
        return "   ".join(parts)

    def _tail(self, buttons, sel, active) -> str:
        t = ""
        if self.status:
            t += "\n" + flash(self.status, self.width) + "\n"
        return t + "\n" + menu(buttons, sel, active, self.width) + "\n"

    def _room(self, head: str, tail: str) -> int:
        return self.height - head.count("\n") - tail.count("\n") - 1 if self.height > 0 else 0

    def _scroll_by(self, d: int) -> bool:
        ttl, body = self._body()
        head = banner(self.live()) + "\n\n" + title(ttl) + "\n"
        tail = self._tail(self.buttons(), self.btn, self.menu_active)
        n = len(body.rstrip("\n").split("\n"))
        room = self._room(head, tail)
        ms = 0 if room <= 0 or n <= room else n - (max(room, 5) - 2)
        nxt = 2 if self.scroll == 0 and d > 0 else 0 if self.scroll == 2 and d < 0 else self.scroll + d
        nxt = min(max(nxt, 0), ms)
        if nxt == self.scroll:
            return False
        self.scroll = nxt
        return True

    def _body(self) -> tuple[str, str]:
        if self.state == "log":
            lines = ["", header("Log") + "  " + muted(cfgmod.display_path(system.log_path()))]
            lines += [INDENT + "  " + (muted("(empty)"))] if not self.log_lines else [
                INDENT + "  " + fg(l) for l in self.log_lines]
            return "Log", "\n".join(lines) + "\n"
        if self.state == "help":
            return "Help", self._help()
        if self.state == "doctor":
            lines = ["", header("Checks")] + [INDENT + (err(l) if "✗" in l else fg(l)) for l in self.doctor_lines]
            return "Doctor", "\n".join(lines) + "\n"
        if self.state == "confirm":
            return "Confirm", "\n" + wrap(self.confirm_text, self.width, fg) + "\n"
        return "Home", self._home()

    def _home(self) -> str:
        w = self.width
        out = ["", header("Overview"), info("Greeting", value_or_dash(self.cfg["GREETING"]), w),
               info("Items", fg(f"{len(self.items)} · {cfgmod.display_path(items_path())}"), w),
               info("Config", fg(cfgmod.display_path(cfgmod.config_path())), w), "", header("Items"),
               row(not self.menu_active and self.focus == 0, "New item",
                   fg(self.new_text) or muted("type and press Enter"), w),
               row(not self.menu_active and self.focus == 1, "Search", fg(self.search) or muted("filter"), w)]
        vis = self.visible()
        if not vis:
            out.append(INDENT + "  " + muted("(no items)"))
        n = int(self.cfg["ITEMS_SHOWN"]) if self.cfg["ITEMS_SHOWN"].isdigit() else 10
        cur = self.focus - 2
        start = cur - n + 1 if cur >= n else 0
        if start:
            out.append(INDENT + muted(f"  ↑ {start} more"))
        for i, it in enumerate(vis[start:start + n], start):
            f = not self.menu_active and i == cur
            out.append(INDENT + (key("▶ ") + key(it) if f else "  " + fg(it)))
        if start + n < len(vis):
            out.append(INDENT + muted(f"  ↓ {len(vis) - start - n} more"))
        return "\n".join(out) + "\n"

    def help_shortcut_rows(self) -> list[tuple[str, str]]:
        """ONE general shortcut list: the navigation keys plus HOME_MENU and
        CONFIG_MENU exactly as they are defined — nothing hand-written that
        could drift from the keys."""
        rows = [("Tab, ↓", "into the menu (first entry selected)"),
                ("← / →", "between menu entries · change a choice in Config"),
                ("Tab, ↑", "out of the menu, back to the top"),
                ("↑ / ↓", "rows · scroll · fields in Config"),
                ("Enter", "run the selected entry · add an item · save in Config"),
                ("q, Esc", "back — quit on Home (q types inside a text field)"),
                ("Ctrl+C", "quit")]
        rows += shortcut_rows([("Home", HOME_MENU), ("Log", self.buttons("log")), ("Doctor", self.buttons("doctor")),
                               ("Confirm", self.buttons("confirm")), ("Config", CONFIG_MENU)])
        rows.append(("^l ^e ^r ^t ^u ^d", "also from Home: log · edit config · reload · reset theme · uninstall · doctor"))
        return rows

    def _help(self) -> str:
        out = []

        def sec(name, rows):
            out.append("")
            out.append(header(name))
            out.extend(info(a, fg(b), self.width) for a, b in rows)

        def para(name, text):
            out.append("")
            out.append(header(name))
            out.append(wrap(text, self.width, fg))

        sec("About", [("Version", f"{meta.APP_NAME} v{meta.VERSION}   Python {platform.python_version()}"),
                      ("Config", cfgmod.display_path(cfgmod.config_path())), ("Log", cfgmod.display_path(system.log_path()))])
        sec("Icons (under the banner, while there are items)",
            [("\U000F0279  Items", "number of items"), ("\U000F0349  Search", "the current search"),
             ("\U000F0317  Host", "this machine's LAN address")])
        sec("Shortcuts", self.help_shortcut_rows())
        sec("Commands", [(f"{meta.APP_NAME} tui", "this interface (default)"),
                         (f"{meta.APP_NAME} install", "launcher, shell alias, config file in the current format"),
                         (f"{meta.APP_NAME} doctor", "diagnostics (also Config → Doctor)"),
                         (f"{meta.APP_NAME} uninstall", "remove launcher + alias"),
                         (f"{meta.APP_NAME} version", "version"), (f"{meta.APP_NAME} help", "CLI help")])
        # The application text comes LAST: aliases, integrations, backends — every sentence true to the code.
        para("Aliases", f"`{meta.APP_NAME}` is a shell alias for the launcher in ~/.local/bin (Config → Shell shows "
             "where it is defined; SHELL_ALIASES=false removes it).")
        para("Integrations", f"TODO({meta.APP_NAME}): describe each integration and backend here — what it does, "
             "where its files live and which config key turns it on (gu: Remmina GU, Bitwarden, KeePass).")
        return "\n".join(out) + "\n"

    def _config_view(self) -> str:
        f = self.form
        lines, focus_line = [], -1
        for i, r in enumerate(f.rows):
            focused = not f.in_menu and i == f.focus
            if r["kind"] == "header" and i > 0:
                lines.append("")
            if focused:
                focus_line = len(lines)
            if r["kind"] == "header":
                lines.append(header(r["label"]))
            elif r["kind"] == "subheader":
                lines.extend(["", subheader(r["label"])])
            elif r["kind"] == "info":
                lines.append(info(r["label"], fg(r["value"]), self.width))
            elif r["kind"] == "text":
                lines.append(row(focused, r["label"], fg(r["value"]) + (key("▏") if focused else ""), self.width))
            elif r["kind"] == "choice":
                arr = key if focused else muted
                lines.append(row(focused, r["label"], arr("‹ ") + fg(r["choices"][r["idx"]]) + arr(" ›"), self.width))
            elif r["kind"] == "action":
                lines.append(INDENT + (key("▶ ") + key("[") + key(r["label"]) if focused
                                       else "  " + key("[") + fg(r["label"])) + key("]"))
        head = banner(self.live()) + "\n\n" + title("Config") + "\n\n"
        tail = self._tail(CONFIG_MENU, f.btn, f.in_menu)
        return head + "\n".join(scroll_window(lines, focus_line, self._room(head, tail))) + "\n" + tail

    def view(self) -> str:
        if self.state == "config":
            return self._config_view()
        ttl, body = self._body()
        head = banner(self.live()) + "\n\n" + title(ttl) + "\n"
        tail = self._tail(self.buttons(), self.btn, self.menu_active)
        lines = body.rstrip("\n").split("\n")
        if self.state == "home":
            return head + "\n".join(lines) + "\n" + tail
        return head + "\n".join(window_at(lines, self.scroll, self._room(head, tail))) + "\n" + tail


def doctor_lines() -> tuple[list[str], int]:
    """The checklist as lines (Config → Doctor) and the number of failed checks."""
    cfg, found = cfgmod.load()
    lines, bad = [], 0

    def rowp(okv, what, detail):
        nonlocal bad
        bad += 0 if okv else 1
        lines.append(f"  {'✓' if okv else '✗'}  {what:<14} {detail}")

    rowp(True, "config", f"{cfgmod.display_path(cfgmod.config_path())} (found={found})")
    rowp(system.launcher_path().exists(), "installed", cfgmod.display_path(system.launcher_path()))
    if system.aliases_supported():
        where = system.alias_where()
        rowp(cfgmod.truthy(cfg.get("SHELL_ALIASES", "false")) == bool(where), "shell alias", f"at {where}")
    rowp(True, "log", cfgmod.display_path(system.log_path()))
    return lines, bad


def doctor() -> int:
    lines, bad = doctor_lines()
    print("\n".join(lines))
    return 1 if bad else 0

