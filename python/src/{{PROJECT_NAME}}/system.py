"""What the app installs around itself: a launcher in ~/.local/bin, the shell
alias, the log file. Each is visible on the Config screen and driven by a
config key (standard v2.1)."""

from __future__ import annotations

import os
import shutil
import sys
from pathlib import Path

from . import meta
from .config import load, save_fresh, truthy

RC_FILES = [".zshrc", ".bashrc"]


def aliases_supported() -> bool:
    return sys.platform != "win32"


def launcher_path() -> Path:
    return Path.home() / ".local" / "bin" / meta.APP_NAME


def alias_where() -> list[str]:
    where = []
    for rc in RC_FILES:
        p = Path.home() / rc
        if p.exists():
            for i, line in enumerate(p.read_text().splitlines(), 1):
                if line.startswith(f"alias {meta.APP_NAME}="):
                    where.append(f"~/{rc}:{i}")
    return where


def sync_aliases(cfg: dict) -> str:
    """Apply SHELL_ALIASES; only existing rc files are touched."""
    if not aliases_supported():
        return ""
    want, have = truthy(cfg.get("SHELL_ALIASES", "false")), bool(alias_where())
    for rc in RC_FILES:
        p = Path.home() / rc
        if not p.exists():
            continue
        if want and not have:
            with p.open("a") as f:
                f.write(f"\nalias {meta.APP_NAME}='{launcher_path()}'\n")
        elif have and not want:
            keep = [l for l in p.read_text().splitlines() if not l.startswith(f"alias {meta.APP_NAME}=")]
            p.write_text("\n".join(keep) + "\n")
    if want and not have:
        return f"Shell alias installed ({meta.APP_NAME})"
    if have and not want:
        return "Shell alias removed"
    return ""


def log_path() -> Path:
    base = os.environ.get("XDG_STATE_HOME") or str(Path.home() / ".local" / "state")
    return Path(base) / meta.APP_NAME / f"{meta.APP_NAME}.log"


def logf(msg: str) -> None:
    p = log_path()
    p.parent.mkdir(parents=True, exist_ok=True)
    with p.open("a") as f:
        f.write(msg + "\n")


def tail(n: int = 200) -> list[str]:
    p = log_path()
    return p.read_text().splitlines()[-n:] if p.exists() else []


def install(project_dir: Path) -> list[str]:
    """Launcher script → ~/.local/bin, alias per config, config file rewritten."""
    out = []
    lp = launcher_path()
    lp.parent.mkdir(parents=True, exist_ok=True)
    uv = shutil.which("uv") or "uv"
    lp.write_text(f'#!/bin/sh\nexec "{uv}" run --project "{project_dir}" {meta.APP_NAME} "$@"\n')
    lp.chmod(0o755)
    out.append(f"Installed {lp}")
    cfg, _ = load()
    if msg := sync_aliases(cfg):
        out.append(msg)
    save_fresh(cfg)
    from .config import config_path
    out.append(f"Config file written in the current format: {config_path()}")
    return out


def uninstall() -> None:
    launcher_path().unlink(missing_ok=True)
    sync_aliases({"SHELL_ALIASES": "false"})
