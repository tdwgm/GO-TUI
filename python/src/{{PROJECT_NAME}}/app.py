"""Entry point: the CLI and the Textual bridge for the pure Model (model.py).

`<app>` / `<app> tui` opens the TUI; `install | doctor | uninstall | version |
help` are the standard commands (TUI Scaffold standard v2.1).
"""

from __future__ import annotations

import os
import platform
import shlex
import subprocess
import sys
from pathlib import Path

from rich.text import Text
from textual.app import App, ComposeResult
from textual.widgets import Static

from . import config as cfgmod
from . import meta, system
from .model import Model, doctor
from .standard import Theme, banner


class TuiApp(App):
    """Renders Model.view() into one Static and feeds it every key."""

    CSS = """
    Screen { background: ansi_default; }
    #view { color: ansi_default; background: ansi_default; }
    """

    def __init__(self) -> None:
        super().__init__(ansi_color=True)
        self.model = Model()

    def compose(self) -> ComposeResult:
        yield Static(id="view")

    def on_mount(self) -> None:
        system.logf(f"start {meta.APP_NAME} v{meta.VERSION}")
        self.set_interval(0.09, self._menu_tick)  # the glitter sweep (MENU_EFFECT)
        self.refresh_view()

    def _menu_tick(self) -> None:
        if Theme.menu_effect == "glitter" and self.model.menu_has_focus():
            Theme.frame += 1
            self.refresh_view()

    def on_resize(self, event) -> None:
        self.model.width, self.model.height = event.size.width, event.size.height
        self.refresh_view()

    def on_key(self, event) -> None:
        event.stop()
        event.prevent_default()
        k = event.key
        if len(k) > 1 and event.character and event.is_printable and k not in ("space",):
            k = event.character  # e.g. "question_mark" → "?"
        self.model.key(k)
        if self.model.wants_editor:
            self.model.wants_editor = False
            editor = os.environ.get("VISUAL") or os.environ.get("EDITOR") or "vi"
            with self.suspend():
                subprocess.run([*shlex.split(editor), str(cfgmod.config_path())])
            self.model.reload()
        if self.model.quit:
            self.exit()
            return
        self.refresh_view()

    def refresh_view(self) -> None:
        self.query_one("#view", Static).update(Text.from_markup(self.model.view()))


def usage() -> None:
    print(Text.from_markup(banner()).plain + "\n")
    print(f"Usage: {meta.APP_NAME} [command]   (no command opens the TUI)\n")
    for c, d in [("tui", "the interactive UI (default)"),
                 ("install", "launcher in ~/.local/bin, shell alias, config file in the current format"),
                 ("doctor", "check config, launcher, alias, log"),
                 ("uninstall", "remove the launcher and the alias (config kept)"),
                 ("version", "print the version"), ("help", "this text")]:
        print(f"  {c:<10} {d}")
    print(f"\nConfig: {cfgmod.config_path()}\nLog:    {system.log_path()}")


def main() -> None:
    cmd = sys.argv[1] if len(sys.argv) > 1 else ""
    if cmd in ("", "tui"):
        TuiApp().run()
    elif cmd in ("help", "info", "-h", "--help"):
        usage()
    elif cmd in ("version", "-v", "--version"):
        if sys.stdout.isatty():
            print(Text.from_markup(banner()).plain + "\n")
        print(f"{meta.APP_NAME} v{meta.VERSION} (Python {platform.python_version()})")
    elif cmd == "install":
        print(Text.from_markup(banner()).plain + "\n")
        for line in system.install(Path(__file__).resolve().parents[2]):
            print(line)
    elif cmd == "uninstall":
        system.uninstall()
        print(f"Removed {system.launcher_path()} and the shell alias (config kept).")
    elif cmd == "doctor":
        sys.exit(doctor())
    else:
        print(f"unknown command: {cmd}\n", file=sys.stderr)
        usage()
        sys.exit(1)


if __name__ == "__main__":
    main()
