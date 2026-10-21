"""Config file ~/.config/<app>/<app>.conf — KEY="value" with # comments.

FIELDS is the single source for the file and the Config screen. Every Config
value is a key here (standard v2.1). OS-specific keys (os="unix") are only read,
written and shown on that OS. save() keeps comments; save_fresh() (install)
rewrites the file in the current format keeping the values.
"""

from __future__ import annotations

import os
import re
import sys
from pathlib import Path

from . import meta
from .font import BANNER_FONTS
from .standard import ACCENT_NAMES, COLOR_NAMES

THEME_SECTION = 'Theme (Config → Theme; "Reset to default" restores these)'

# (key, default, description, section, os)
FIELDS = [
    ("GREETING", "hello", "Line shown on the Home screen.", "General", ""),
    ("ITEMS_SHOWN", "10", "Max items listed before the list scrolls.", "General", ""),
    ("CONFIRM_DELETE", "true", "Ask before deleting an item.", "Behaviour", ""),
    ("THEME", "blue", "Accent: " + " | ".join(ACCENT_NAMES), THEME_SECTION, ""),
    ("COLOR_BANNER", "accent", "Element colours: " + " | ".join(COLOR_NAMES), THEME_SECTION, ""),
    ("COLOR_HEADER", "ocean", "", THEME_SECTION, ""),
    ("COLOR_TEXT", "white", "", THEME_SECTION, ""),
    ("COLOR_MUTED", "grey", "", THEME_SECTION, ""),
    ("COLOR_SHORTCUT", "grey", "Menu shortcuts, selected menu entry, the dotted rule, text under the logo, live info block", THEME_SECTION, ""),
    ("COLOR_SELECTED", "default", "", THEME_SECTION, ""),
    ("COLOR_RULE", "grey", "", THEME_SECTION, ""),
    ("COLOR_SUBTITLE", "white", "", THEME_SECTION, ""),
    ("COLOR_LIVE", "default", "", THEME_SECTION, ""),
    ("BANNER_TEXT", "", "Banner (Config → Theme → Banner). Text: empty = the app name; any other text is drawn with the font (block when the font is default)", THEME_SECTION, ""),
    ("BANNER_FONT", "default", "Font: " + " | ".join(BANNER_FONTS), THEME_SECTION, ""),
    ("LIVE_INFO", "full", "Live info under the banner: full (items, search, host IP) | compact | off", THEME_SECTION, ""),
    ("MENU_EFFECT", "glitter", "Selected menu entry: glitter (a glint sweeps across the text) | none", THEME_SECTION, ""),
    ("SHELL_ALIASES", "true", "Keep the shell alias in ~/.zshrc + ~/.bashrc: true | false", "Shell", "unix"),
]
LIVE_INFO_MODES = ["full", "compact", "off"]
MENU_EFFECTS = ["glitter", "none"]
COLOR_KEYS = ["COLOR_BANNER", "COLOR_HEADER", "COLOR_TEXT", "COLOR_MUTED", "COLOR_SHORTCUT",
              "COLOR_SELECTED", "COLOR_RULE", "COLOR_SUBTITLE", "COLOR_LIVE"]
THEME_KEYS = ["THEME", *COLOR_KEYS, "BANNER_TEXT", "BANNER_FONT", "LIVE_INFO", "MENU_EFFECT"]  # what "Reset to default" restores
_ALLOWED = {"THEME": ACCENT_NAMES, "BANNER_FONT": BANNER_FONTS, "LIVE_INFO": LIVE_INFO_MODES, "MENU_EFFECT": MENU_EFFECTS,
            **{k: COLOR_NAMES for k in COLOR_KEYS}}
_LINE = re.compile(r'^\s*([A-Z][A-Z0-9_]*)\s*=\s*"?(.*?)"?\s*$')


def on_this_os(os_tag: str) -> bool:
    return os_tag == "" or (os_tag == "unix" and sys.platform != "win32") or os_tag == sys.platform


def config_path() -> Path:
    return Path.home() / ".config" / meta.APP_NAME / f"{meta.APP_NAME}.conf"


def defaults() -> dict:
    return {k: d for k, d, _, _, o in FIELDS if on_this_os(o)}


def sanitize(cfg: dict) -> dict:
    d = defaults()
    out = dict(d)
    for k, v in cfg.items():
        if k in d:
            out[k] = v if k not in _ALLOWED or v in _ALLOWED[k] else d[k]
    return out


def load() -> tuple[dict, bool]:
    p = config_path()
    if not p.exists():
        return defaults(), False
    vals = {}
    for line in p.read_text().splitlines():
        m = _LINE.match(line)
        if m and not line.lstrip().startswith("#"):
            vals[m.group(1)] = m.group(2)
    return sanitize(vals), True


def _template() -> list[str]:
    lines = [f"# {meta.APP_NAME} configuration ({config_path()}). Edit here or on the Config screen.",
             '# Format: shell-style KEY="value" with # comments.']
    section = None
    for k, d, desc, sec, o in FIELDS:
        if not on_this_os(o):
            continue
        if sec != section:
            lines += ["", f"# --- {sec} ---"]
            section = sec
        if desc:
            lines.append(f"# {desc}")
        lines.append(f'{k}="{d}"')
    return lines


def _write(lines: list[str], cfg: dict) -> None:
    cfg = sanitize(cfg)
    seen = set()
    out = []
    for line in lines:
        m = _LINE.match(line)
        if m and not line.lstrip().startswith("#") and m.group(1) in cfg:
            out.append(f'{m.group(1)}="{cfg[m.group(1)]}"')
            seen.add(m.group(1))
        else:
            out.append(line)
    out += [f'{k}="{v}"' for k, v in cfg.items() if k not in seen]
    p = config_path()
    p.parent.mkdir(parents=True, exist_ok=True)
    tmp = p.with_suffix(".tmp")
    tmp.write_text("\n".join(out) + "\n")
    os.chmod(tmp, 0o600)
    tmp.replace(p)


def save(cfg: dict) -> None:
    """Write values into the existing file, keeping its comments."""
    p = config_path()
    _write(p.read_text().splitlines() if p.exists() else _template(), cfg)


def save_fresh(cfg: dict) -> None:
    """Rewrite the file in the current format (install), keeping the values."""
    _write(_template(), cfg)


def truthy(v: str) -> bool:
    return v.strip().lower() in ("1", "true", "yes", "on")
