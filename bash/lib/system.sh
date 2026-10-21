# lib/system.sh — what the app installs around itself: the launcher symlink in
# ~/.local/bin, the shell alias, the log file. All visible on the Config screen.
[[ -n "${_SYSTEM_SH:-}" ]] && return 0
_SYSTEM_SH=1

RC_FILES=(.zshrc .bashrc)

aliases_supported() { [[ "$(uname -s)" != MINGW* ]]; }

launcher_path() { printf '%s/.local/bin/%s' "$HOME" "$APP_NAME"; }

# alias_where — prints "~/.zshrc:12" for every alias definition.
alias_where() {
    local rc n line
    for rc in "${RC_FILES[@]}"; do
        [[ -f "$HOME/$rc" ]] || continue
        n=0
        while IFS= read -r line; do
            n=$(( n + 1 ))
            if [[ "$line" == "alias $APP_NAME="* ]]; then printf '~/%s:%d\n' "$rc" "$n"; fi
        done <"$HOME/$rc"
    done
    return 0
}

# sync_aliases — applies SHELL_ALIASES (only existing rc files); prints a summary.
sync_aliases() {
    aliases_supported || return 0
    local have=0 rc
    [[ -n "$(alias_where)" ]] && have=1
    if truthy "${CFG[SHELL_ALIASES]:-false}" && (( ! have )); then
        for rc in "${RC_FILES[@]}"; do
            if [[ -f "$HOME/$rc" ]]; then printf "\nalias %s='%s'\n" "$APP_NAME" "$(launcher_path)" >>"$HOME/$rc"; fi
        done
        printf 'Shell alias installed (%s)' "$APP_NAME"
    elif ! truthy "${CFG[SHELL_ALIASES]:-false}" && (( have )); then
        for rc in "${RC_FILES[@]}"; do
            [[ -f "$HOME/$rc" ]] || continue
            grep -v "^alias $APP_NAME=" "$HOME/$rc" >"$HOME/$rc.tmp" || true
            cat "$HOME/$rc.tmp" >"$HOME/$rc" && rm -f "$HOME/$rc.tmp"
        done
        printf 'Shell alias removed'
    fi
    return 0
}

log_path() { printf '%s/%s/%s.log' "${XDG_STATE_HOME:-$HOME/.local/state}" "$APP_NAME" "$APP_NAME"; }

logf() {
    local p
    p=$(log_path)
    mkdir -p "$(dirname "$p")"
    printf '%s\n' "$*" >>"$p"
}

# do_install — launcher symlink, alias per config, config file in the current format.
do_install() {
    local lp msg
    lp=$(launcher_path)
    mkdir -p "$(dirname "$lp")"
    ln -sfn "$SCRIPT_DIR/main.sh" "$lp" && echo "Installed $lp -> $SCRIPT_DIR/main.sh"
    config_load || true
    msg=$(sync_aliases)
    if [[ -n "$msg" ]]; then echo "$msg"; fi
    config_save_fresh && echo "Config file written in the current format: $(display_path "$(config_path)")"
}

do_uninstall() {
    rm -f "$(launcher_path)"
    CFG[SHELL_ALIASES]=false
    sync_aliases >/dev/null
}

# doctor — the checklist (`<app> doctor` and Config → Doctor). Exit 1 when a check fails.
doctor() {
    local bad=0 found=true where
    config_load || found=false
    _row() { if [[ "$1" == 1 ]]; then printf '  ✓  %-14s %s\n' "$2" "$3"; else printf '  ✗  %-14s %s\n' "$2" "$3"; bad=1; fi; }
    _row 1 config "$(display_path "$(config_path)") (found=$found)"
    if [[ -e "$(launcher_path)" ]]; then _row 1 installed "$(display_path "$(launcher_path)")"; else _row 0 installed "$(display_path "$(launcher_path)")"; fi
    if aliases_supported; then
        where=$(alias_where | paste -sd' ')
        if { truthy "${CFG[SHELL_ALIASES]}" && [[ -n "$where" ]]; } || { ! truthy "${CFG[SHELL_ALIASES]}" && [[ -z "$where" ]]; }; then
            _row 1 "shell alias" "at ${where:-—}"
        else
            _row 0 "shell alias" "SHELL_ALIASES=${CFG[SHELL_ALIASES]} but at ${where:-—}"
        fi
    fi
    _row 1 log "$(display_path "$(log_path)")"
    return $bad
}
