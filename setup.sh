#!/usr/bin/env bash
# setup.sh — verify host has tools needed by all three templates.
# Run once per machine. Doesn't install anything — reports + suggests.
set -euo pipefail

RESET='\033[0m'; BOLD='\033[1m'
ACCENT='\033[38;2;136;192;208m'; OK='\033[38;2;163;190;140m'; WARN='\033[38;2;235;203;139m'; ERR='\033[38;2;191;97;106m'

ok()   { printf "  ${OK}✓${RESET} %s\n" "$*"; }
warn() { printf "  ${WARN}!${RESET} %s\n" "$*"; }
fail() { printf "  ${ERR}✗${RESET} %s\n" "$*"; failed=1; }

failed=0

printf "${BOLD}${ACCENT}tui-cli-template setup — checking host${RESET}\n\n"

printf "${ACCENT}Go template${RESET}\n"
if command -v go >/dev/null 2>&1; then
    ok "go: $(go version | awk '{print $3}')"
else
    fail "go not installed — sudo dnf install golang"
fi

printf "\n${ACCENT}Python (Textual) template${RESET}\n"
if command -v python3 >/dev/null 2>&1; then
    pyver="$(python3 --version 2>&1 | awk '{print $2}')"
    ok "python3: $pyver"
else
    fail "python3 not installed — sudo dnf install python3"
fi
if command -v uv >/dev/null 2>&1; then
    ok "uv: $(uv --version 2>&1 | awk '{print $2}')"
else
    warn "uv not installed — install via: curl -LsSf https://astral.sh/uv/install.sh | sh"
    warn "  (the /tui-scaffold skill can install uv interactively when scaffolding python)"
fi

printf "\n${ACCENT}Bash template${RESET}\n"
ok "bash: $BASH_VERSION"
if command -v tput >/dev/null 2>&1; then
    ok "tput: present"
else
    fail "tput not installed — sudo dnf install ncurses"
fi
if command -v shellcheck >/dev/null 2>&1; then
    ok "shellcheck: $(shellcheck --version | awk '/^version/{print $2}')"
else
    warn "shellcheck not installed (optional but recommended) — sudo dnf install ShellCheck"
fi

printf "\n"
if (( failed )); then
    printf "${ERR}Setup incomplete — install missing tools above.${RESET}\n"
    exit 1
fi
printf "${OK}All required tools present.${RESET}\n"
