# lib/config.sh — config file ~/.config/<app>/<app>.conf, KEY="value" + # comments.
# FIELDS is the single source for the file and the Config screen; every Config
# value is a key here. OS-specific keys (os=unix) only on that OS. config_save
# keeps comments; config_save_fresh (install) rewrites the file in the current
# format keeping the values.
[[ -n "${_CONFIG_SH:-}" ]] && return 0
_CONFIG_SH=1

THEME_SECTION='Theme (Config → Theme; "Reset to default" restores these)'
# key|default|description|section|os   (the description may contain " | ")
FIELDS=(
    'GREETING|hello|Line shown on the Home screen.|General|'
    'ITEMS_SHOWN|10|Max items listed before the list scrolls.|General|'
    'CONFIRM_DELETE|true|Ask before deleting an item.|Behaviour|'
    "THEME|blue|Accent: blue / green / red / yellow / magenta / cyan / orange / purple|$THEME_SECTION|"
    "COLOR_BANNER|accent|Element colours: accent / default / blue / ocean / green / red / yellow / magenta / cyan / orange / purple / grey / white|$THEME_SECTION|"
    "COLOR_HEADER|ocean||$THEME_SECTION|"
    "COLOR_TEXT|white||$THEME_SECTION|"
    "COLOR_MUTED|grey||$THEME_SECTION|"
    "COLOR_SHORTCUT|grey|Menu shortcuts, selected menu entry, the dotted rule, text under the logo, live info block|$THEME_SECTION|"
    "COLOR_SELECTED|default||$THEME_SECTION|"
    "COLOR_RULE|grey||$THEME_SECTION|"
    "COLOR_SUBTITLE|white||$THEME_SECTION|"
    "COLOR_LIVE|default||$THEME_SECTION|"
    "BANNER_TEXT||Banner (Config → Theme → Banner). Text: empty = the app name; any other text is drawn with the font (block when the font is default)|$THEME_SECTION|"
    "BANNER_FONT|default|Font: default / block / half / slant / shade|$THEME_SECTION|"
    "LIVE_INFO|full|Live info under the banner: full (items, search, host IP) / compact / off|$THEME_SECTION|"
    'SHELL_ALIASES|true|Keep the shell alias in ~/.zshrc + ~/.bashrc: true / false|Shell|unix'
)
THEME_KEYS=(THEME COLOR_BANNER COLOR_HEADER COLOR_TEXT COLOR_MUTED COLOR_SHORTCUT COLOR_SELECTED
    COLOR_RULE COLOR_SUBTITLE COLOR_LIVE BANNER_TEXT BANNER_FONT LIVE_INFO)   # what "Reset to default" restores
LIVE_INFO_MODES=(full compact off)
declare -A CFG=()

# _field LINE — sets F_KEY F_DEF F_DESC F_SEC F_OS.
_field() {
    local l="$1"
    F_KEY="${l%%|*}"; l="${l#*|}"
    F_DEF="${l%%|*}"; l="${l#*|}"
    F_OS="${l##*|}"; l="${l%|*}"
    F_SEC="${l##*|}"; F_DESC="${l%|*}"
}

_on_os() { [[ -z "$1" ]] || { [[ "$1" == unix ]] && [[ "$(uname -s)" != MINGW* ]]; }; }

config_path() { printf '%s/.config/%s/%s.conf' "$HOME" "$APP_NAME" "$APP_NAME"; }

config_defaults() {
    local f
    CFG=()
    for f in "${FIELDS[@]}"; do
        _field "$f"
        if _on_os "$F_OS"; then CFG[$F_KEY]="$F_DEF"; fi
    done
}

# _allowed KEY VALUE — theme values must be known names.
_allowed() {
    case "$1" in
        THEME) [[ " ${ACCENT_NAMES[*]} " == *" $2 "* ]] ;;
        COLOR_*) [[ " ${COLOR_NAMES[*]} " == *" $2 "* ]] ;;
        BANNER_FONT) [[ " ${BANNER_FONTS[*]} " == *" $2 "* ]] ;;
        LIVE_INFO) [[ " ${LIVE_INFO_MODES[*]} " == *" $2 "* ]] ;;
        *) return 0 ;;
    esac
}

# config_load — fills CFG (defaults + file); returns 1 when there is no file.
config_load() {
    config_defaults
    local p k v line
    p=$(config_path)
    [[ -f "$p" ]] || return 1
    while IFS= read -r line; do
        [[ "$line" =~ ^([A-Z][A-Z0-9_]*)=\"(.*)\"$ ]] || continue
        k="${BASH_REMATCH[1]}"; v="${BASH_REMATCH[2]}"
        if [[ -v "CFG[$k]" ]] && _allowed "$k" "$v"; then CFG[$k]="$v"; fi
    done <"$p"
    return 0
}

_template() {
    local f sec=""
    printf '# %s configuration (%s). Edit here or on the Config screen.\n' "$APP_NAME" "$(config_path)"
    printf '# Format: shell-style KEY="value" with # comments.\n'
    for f in "${FIELDS[@]}"; do
        _field "$f"
        _on_os "$F_OS" || continue
        if [[ "$F_SEC" != "$sec" ]]; then printf '\n# --- %s ---\n' "$F_SEC"; sec="$F_SEC"; fi
        if [[ -n "$F_DESC" ]]; then printf '# %s\n' "$F_DESC"; fi
        printf '%s="%s"\n' "$F_KEY" "$F_DEF"
    done
}

# _write — source text on stdin; writes it with CFG's values (missing keys appended).
_write() {
    local p line k
    local -A seen=()
    p=$(config_path)
    mkdir -p "$(dirname "$p")"
    {
        while IFS= read -r line; do
            if [[ "$line" =~ ^([A-Z][A-Z0-9_]*)= ]] && [[ -v "CFG[${BASH_REMATCH[1]}]" ]]; then
                k="${BASH_REMATCH[1]}"
                printf '%s="%s"\n' "$k" "${CFG[$k]}"
                seen[$k]=1
            else
                printf '%s\n' "$line"
            fi
        done
        for k in "${!CFG[@]}"; do
            [[ -v "seen[$k]" ]] || printf '%s="%s"\n' "$k" "${CFG[$k]}"
        done
    } >"$p.tmp"
    chmod 600 "$p.tmp" && mv "$p.tmp" "$p"
}

config_save() {
    local p
    p=$(config_path)
    if [[ -f "$p" ]]; then _write <"$p"; else _template | _write; fi
}

config_save_fresh() { _template | _write; }

truthy() { [[ "${1,,}" =~ ^(1|true|yes|on)$ ]]; }
