# lib/standard.sh — TUI Scaffold standard v2.1: theme + layout grammar (Bash port
# of the Go template's style.go/layout.go/banner.go — same look as gu-vpn-pid).
# Rules: every text line indented, ONE dotted rule (under the banner, column 0),
# no footer, tuta-cli rows, one-line menu "▶ Label key · … · Help h", colours
# only from the theme. Functions print to stdout (callers capture with $(...)).
[[ -n "${_STANDARD_SH:-}" ]] && return 0
_STANDARD_SH=1

INDENT="  "
LABEL_WIDTH=18
ACCENT_NAMES=(blue green red yellow magenta cyan orange purple)
COLOR_NAMES=(accent default blue ocean green red yellow magenta cyan orange purple grey white)
declare -A NAMED=([blue]="172;199;255" [ocean]="122;162;247" [green]="136;204;136" [red]="255;122;122" [yellow]="243;249;157"
    [magenta]="255;121;198" [cyan]="139;233;253" [orange]="255;184;108" [purple]="189;147;249"
    [grey]="136;136;136" [white]="238;238;238")
R=$'\e[0m'
C_ACCENT="" C_BANNER="" C_HEADER="" C_TEXT="" C_MUTED="" C_ERR=$'\e[38;2;255;122;122m'
C_SHORTCUT="" C_SELECTED="" C_RULE="" C_SUBTITLE="" C_LIVE=""

# _pick NAME FALLBACK — escape sequence for a theme colour name.
_pick() {
    case "$1" in
        accent) printf '%s' "$C_ACCENT" ;;
        default) printf '' ;;
        *) if [[ -n "${NAMED[$1]:-}" ]]; then printf '\e[38;2;%sm' "${NAMED[$1]}"; else printf '%s' "$2"; fi ;;
    esac
}

# theme_apply — rebuild colours from CFG (THEME + COLOR_*); also the live preview.
theme_apply() {
    C_ACCENT=$'\e[38;2;'"${NAMED[${CFG[THEME]:-blue}]:-${NAMED[blue]}}"'m'
    C_BANNER=$(_pick "${CFG[COLOR_BANNER]:-accent}" "$C_ACCENT")
    C_HEADER=$(_pick "${CFG[COLOR_HEADER]:-ocean}" $'\e[38;2;122;162;247m')
    C_TEXT=$(_pick "${CFG[COLOR_TEXT]:-white}" $'\e[38;2;238;238;238m')
    C_MUTED=$(_pick "${CFG[COLOR_MUTED]:-grey}" $'\e[38;2;136;136;136m')
    C_SHORTCUT=$(_pick "${CFG[COLOR_SHORTCUT]:-grey}" $'\e[38;2;136;136;136m')
    C_SELECTED=$(_pick "${CFG[COLOR_SELECTED]:-default}" "")
    C_RULE=$(_pick "${CFG[COLOR_RULE]:-grey}" $'\e[38;2;136;136;136m')
    C_SUBTITLE=$(_pick "${CFG[COLOR_SUBTITLE]:-white}" $'\e[38;2;238;238;238m')
    C_LIVE=$(_pick "${CFG[COLOR_LIVE]:-default}" "")
    UI_BANNER_FONT="${CFG[BANNER_FONT]:-default}" UI_LIVE_INFO="${CFG[LIVE_INFO]:-full}" UI_BANNER_TEXT="${CFG[BANNER_TEXT]:-}"  # live preview reads these
}

fg()    { printf '%s%s%s' "$C_TEXT" "$1" "$R"; }
muted() { printf '%s%s%s' "$C_MUTED" "$1" "$R"; }
key()   { printf '\e[1m%s%s%s' "$C_ACCENT" "$1" "$R"; }
okc()   { printf '%s%s%s' "$C_HEADER" "$1" "$R"; }
errc()  { printf '%s%s%s' "$C_ERR" "$1" "$R"; }
shortc() { printf '%s%s%s' "$C_SHORTCUT" "$1" "$R"; }
selc()  { printf '%s%s%s' "$C_SELECTED" "$1" "$R"; }
subc()  { printf '%s%s%s' "$C_SUBTITLE" "$1" "$R"; }
livec() { printf '%s%s%s' "$C_LIVE" "$1" "$R"; }
accentc() { printf '%s%s%s' "$C_ACCENT" "$1" "$R"; }  # accent, not bold (icons)
rulec() { printf '%s%s%s' "$C_RULE" "$1" "$R"; }

# cutp TEXT MAX — plain text cut to MAX chars with … (long paths).
cutp() {
    local s="$1" n="$2"
    if (( n > 1 && ${#s} > n )); then s="${s:0:n-1}…"; fi
    printf '%s' "$s"
}

# vmax — room for a row value at the current width.
vmax() { echo $(( ${COLS:-100} - ${#INDENT} - LABEL_WIDTH - 5 )); }

rule()   { printf '\e[2m%s%s%s' "$C_RULE" "············································" "$R"; }
title()  { printf '%s\e[1m%s%s › %s%s' "$INDENT" "$C_ACCENT" "$APP_NAME" "$1" "$R"; }
header() { printf '%s%s' "$INDENT" "$(okc "$1")"; }

# row FOCUSED LABEL VALUE_STYLED — tuta-cli row: "▶ " + accent label when focused.
row() {
    local lbl
    printf -v lbl '%-*s' "$LABEL_WIDTH" "$2:"
    if [[ "$1" == 1 ]]; then
        printf '%s%s%s %s' "$INDENT" "$(key '▶ ')" "$(key "$lbl")" "$3"
    else
        printf '%s  %s %s' "$INDENT" "$(fg "$lbl")" "$3"
    fi
}

# info LABEL VALUE_PLAIN [MUTED_SUFFIX] — read-only row; value + suffix are cut
# together to the terminal width with … (long paths never wrap).
info() {
    local lbl v="$2" sfx="${3:-}" max
    printf -v lbl '%-*s' "$LABEL_WIDTH" "$1:"
    max=$(vmax)
    if (( ${#v} >= max )); then v=$(cutp "$v" "$max"); sfx=""
    elif (( ${#v} + ${#sfx} > max )); then sfx=$(cutp "$sfx" $(( max - ${#v} ))); fi
    printf '%s  %s %s' "$INDENT" "$(fg "$lbl")" "$(fg "$v")"
    if [[ -n "$sfx" ]]; then muted "$sfx"; fi
}

# subheader TEXT — an indented heading inside a section (Theme → Banner).
subheader() { printf '%s  %s%s%s' "$INDENT" "$C_HEADER" "$1" "$R"; }

# menu SEL ACTIVE "Label|action|key" ... — ONE line, labels plain, the selected
# one bold, shortcuts in COLOR_SHORTCUT, Help last.
menu() {
    local sel=$1 active=$2
    shift 2
    local i=0 out="" w=0 lab act k cell cw it
    local limit=$(( ${COLS:-100} - ${#INDENT} - 1 ))
    for it in "$@"; do
        IFS='|' read -r lab act k <<<"$it"
        : "$act"
        if [[ $active == 1 && $i == "$sel" ]]; then
            cell="$(key '▶ ')$(printf '\e[1m%s%s%s' "$C_SELECTED" "$lab" "$R")"; cw=$(( ${#lab} + 2 ))  # bold = selected
        else
            cell=$(fg "$lab"); cw=${#lab}
            if (( w == 0 )); then cell="  $cell"; cw=$(( cw + 2 )); fi
        fi
        if [[ -n "$k" ]]; then cell+=" $(shortc "$k")"; cw=$(( cw + 1 + ${#k} )); fi
        if (( w > 0 && w + 3 + cw > limit )); then out+=$'\n'; w=0; fi
        if (( w > 0 )); then out+=$(shortc ' · '); w=$(( w + 3 )); else out+="$INDENT"; fi
        out+="$cell"; w=$(( w + cw )); i=$(( i + 1 ))
    done
    printf '%s' "$out"
}

frakturify() {
    local s="$1" out="" c o i
    for (( i = 0; i < ${#s}; i++ )); do
        c="${s:i:1}"
        printf -v o '%d' "'$c"
        if [[ $c == [a-z] ]]; then
            printf -v c "\\U$(printf '%08x' $(( 0x1D586 + o - 97 )))"
        elif [[ $c == [A-Z] ]]; then
            printf -v c "\\U$(printf '%08x' $(( 0x1D56C + o - 65 )))"
        fi
        out+="$c"
    done
    printf '%s' "$out"
}

# banner [LIVE] — logo with v<version> after its last line, description, byline,
# the one rule, and (TUI) the live info block directly under the rule.
banner() {
    local art="${BANNER_ART:-$APP_NAME}" line
    local -a lines=()
    # BANNER_TEXT / BANNER_FONT: the default logo only shows the app name, so any
    # other text is drawn with the font (block when the font is "default").
    local text="${UI_BANNER_TEXT:-$APP_NAME}" font="${UI_BANNER_FONT:-default}" f
    [[ "$font" == default && "$text" != "$APP_NAME" ]] && font=block
    f=$(font_render "$text" "$font")
    [[ -n "$f" ]] && art="$f"
    while IFS= read -r line; do
        lines+=("$INDENT$(printf '\e[1m%s%s%s' "$C_BANNER" "$line" "$R")")
    done <<<"$art"
    lines[-1]+="  $(subc "v$VERSION")"
    printf '%s\n' "${lines[@]}"
    printf '%s%s\n' "$INDENT" "$(subc "$APP_DESC")"
    printf '\n%s%s\n\n' "$INDENT" "$(subc "$(frakturify 'by Tdwg Entertainment Solutions') ®")"
    rule
    # the live info block sits directly UNDER the rule
    if [[ -n "${1:-}" ]]; then printf '\n\n%s%s' "$INDENT" "$1"; fi
}

# flash MSG — ✓ (header colour) / ✗ red / neutral muted.
flash() {
    local l="${1,,}"
    case "$l" in
        *failed*|*"could not"*|*error*) printf '%s%s' "$INDENT" "$(errc "✗ $1")" ;;
        *saved*|*reloaded*|*cleared*|*added*|*deleted*|*installed*|*removed*|*reset*)
            printf '%s%s' "$INDENT" "$(okc "✓ $1")" ;;
        *) printf '%s%s' "$INDENT" "$(muted "$1")" ;;
    esac
}

# window_at START ROOM — lines on stdin; prints the window with ↑/↓ n more.
window_at() {
    local start=$1 room=$2 n end inner
    local -a L
    mapfile -t L
    n=${#L[@]}
    if (( room <= 0 || n <= room )); then printf '%s\n' "${L[@]}"; return 0; fi
    (( room < 5 )) && room=5
    inner=$(( room - 2 ))
    (( start > n - inner )) && start=$(( n - inner ))
    (( start < 0 )) && start=0
    end=$(( start + inner ))
    if (( start > 0 )); then printf '%s%s\n' "$INDENT" "$(muted "  ↑ $start more")"; else end=$(( end + 1 )); fi
    (( end > n )) && end=$n
    printf '%s\n' "${L[@]:start:end-start}"
    if (( end < n )); then printf '%s%s\n' "$INDENT" "$(muted "  ↓ $(( n - end )) more")"; fi
    return 0
}
