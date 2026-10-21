# lib/app.sh — the TUI as state + key + view (same behaviour as the Go and
# Python templates; testable headless with tests/test.sh).
#
# Screens: home, config, log, help, confirm. Keys (standard v2.1): Tab/↓ into
# the one-line menu (first entry), ←/→ in the menu, Tab/↑ out; q ≡ Esc = back
# (quit on Home; q types inside a text field); Config's Ctrl shortcuts
# (^S ^L ^E ^R ^T ^U) and F1 work everywhere; uninstall goes through Confirm.
[[ -n "${_APP_SH:-}" ]] && return 0
_APP_SH=1

HOME_MENU=("Delete|delete|x" "Config|config|o" "Log|log|l" "Quit|back|q" "Help|help|h")
CONFIG_MENU=("Save|save|^s" "Cancel|back|esc" "Doctor|doctor|^d" "Log|log|^l" "Edit|edit-config|^e" "Reload|reload|^r"
    "Reset theme|theme-reset|^t" "Uninstall|uninstall|^u" "Help|help|h")   # lowercase like the Home menu
declare -A GLOBAL=([ctrl+l]=log [ctrl+e]=edit-config [ctrl+r]=reload [ctrl+t]=theme-reset [ctrl+u]=uninstall [ctrl+d]=doctor)

STATE=home STATUS="" MENU_ACTIVE=0 BTN=0 SCROLL=0 QUIT=0 WANTS_EDITOR=0
NEW_TEXT="" SEARCH="" FOCUS=0
HELP_RETURN=home LOG_RETURN=home CONFIRM_RETURN=home CONFIRM_ACTION="" CONFIRM_TEXT=""
ITEMS=()
# Config form rows (parallel arrays): kind label key value choices idx action
F_KIND=() F_LABEL=() F_KEYS=() F_VAL=() F_CHOICES=() F_IDX=() F_ACT=()
F_FOCUS=0 F_INMENU=0 F_BTN=0

items_path() { printf '%s/%s/items.txt' "${XDG_DATA_HOME:-$HOME/.local/share}" "$APP_NAME"; }

items_load() {
    local p; p=$(items_path)
    if [[ -f "$p" ]]; then mapfile -t ITEMS <"$p"; else ITEMS=(alpha bravo charlie delta echo); fi
}

items_save() {
    local p; p=$(items_path)
    mkdir -p "$(dirname "$p")"
    printf '%s\n' "${ITEMS[@]}" >"$p"
}

DOCTOR_RETURN=home DOCTOR_TEXT=""
app_init() {
    config_load || true
    theme_apply
    items_load
}

visible() { # prints the items matching SEARCH, one per line
    local it q="${SEARCH,,}"
    for it in "${ITEMS[@]}"; do [[ -z "$q" || "${it,,}" == *"$q"* ]] && printf '%s\n' "$it"; done
    return 0
}

# ---------- Config form ----------

_frow() { F_KIND+=("$1"); F_LABEL+=("$2"); F_KEYS+=("$3"); F_VAL+=("$4"); F_CHOICES+=("$5"); F_IDX+=("$6"); F_ACT+=("$7"); }

_fchoice() { # label key "choices..."
    local v="${CFG[$2]}" i=0 c idx=0
    [[ "$v" == true ]] && v=yes
    [[ "$v" == false ]] && v=no
    for c in $3; do [[ "$c" == "$v" ]] && idx=$i; i=$(( i + 1 )); done
    _frow choice "$1" "$2" "" "$3" "$idx" ""
}

form_new() {
    F_KIND=() F_LABEL=() F_KEYS=() F_VAL=() F_CHOICES=() F_IDX=() F_ACT=()
    _frow info Config "" "$(config_path)" "" 0 ""
    _frow header General "" "" "" 0 ""
    _frow text Greeting GREETING "${CFG[GREETING]}" "" 0 ""
    _frow text "Items shown" ITEMS_SHOWN "${CFG[ITEMS_SHOWN]}" "" 0 ""
    _frow header Behaviour "" "" "" 0 ""
    _fchoice "Confirm delete" CONFIRM_DELETE "no yes"
    _frow header Theme "" "" "" 0 ""
    _fchoice Accent THEME "${ACCENT_NAMES[*]}"
    _fchoice Banner COLOR_BANNER "${COLOR_NAMES[*]}"
    _fchoice Headers COLOR_HEADER "${COLOR_NAMES[*]}"
    _fchoice Text COLOR_TEXT "${COLOR_NAMES[*]}"
    _fchoice "Muted text" COLOR_MUTED "${COLOR_NAMES[*]}"
    _fchoice "Selected text" COLOR_SELECTED "${COLOR_NAMES[*]}"
    _fchoice Shortcuts COLOR_SHORTCUT "${COLOR_NAMES[*]}"
    _fchoice Rule COLOR_RULE "${COLOR_NAMES[*]}"
    _fchoice "Under logo" COLOR_SUBTITLE "${COLOR_NAMES[*]}"
    _fchoice "Live info text" COLOR_LIVE "${COLOR_NAMES[*]}"
    _frow subheader Banner "" "" "" 0 ""
    _frow text Text BANNER_TEXT "${CFG[BANNER_TEXT]:-}" "" 0 ""
    _fchoice Font BANNER_FONT "${BANNER_FONTS[*]}"
    _fchoice "Live info" LIVE_INFO "${LIVE_INFO_MODES[*]}"
    _frow action "Reset to default" "" "" "" 0 theme-reset
    if aliases_supported; then # OS-specific: only shown where it exists
        local where state="not installed"
        where=$(alias_where | paste -sd' ' | sed 's/ / · /g')
        [[ -n "$where" ]] && state="installed — $where"
        _frow header Shell "" "" "" 0 ""
        _frow info "alias $APP_NAME" "" "$state" "" 0 ""
        _fchoice "Shell alias" SHELL_ALIASES "no yes"
    fi
    _frow header Doctor "" "" "" 0 ""
    _frow action "Run doctor" "" "" "" 0 doctor
    _frow header Log "" "" "" 0 ""
    _frow info "Log file" "" "$(log_path)" "" 0 ""
    F_INMENU=0 F_BTN=0
    F_FOCUS=$(_fsel | head -1)
}

_fsel() { local i; for i in "${!F_KIND[@]}"; do [[ "${F_KIND[$i]}" =~ ^(text|choice|action)$ ]] && echo "$i"; done; return 0; }

_fchoice_val() { local -a c; read -ra c <<<"${F_CHOICES[$1]}"; printf '%s' "${c[${F_IDX[$1]}]}"; }

form_to_cfg() { # copies the rows into CFG
    local i v
    for i in "${!F_KIND[@]}"; do
        case "${F_KIND[$i]}" in
            text) CFG[${F_KEYS[$i]}]="${F_VAL[$i]}" ;;
            choice) v=$(_fchoice_val "$i"); [[ "$v" == yes ]] && v=true; [[ "$v" == no ]] && v=false
                CFG[${F_KEYS[$i]}]="$v" ;;
        esac
    done
}

form_reset_theme() {
    local i k c n
    config_defaults_copy
    for i in "${!F_KIND[@]}"; do
        k="${F_KEYS[$i]}"
        [[ " ${THEME_KEYS[*]} " == *" $k "* ]] || continue
        [[ "${F_KIND[$i]}" == text ]] && { F_VAL[$i]="${DEF[$k]}"; continue; }
        n=0
        for c in ${F_CHOICES[$i]}; do [[ "$c" == "${DEF[$k]}" ]] && F_IDX[$i]=$n; n=$(( n + 1 )); done
    done
    _preview
}

config_defaults_copy() { # DEF = defaults, CFG untouched
    local -A save=(); local k
    for k in "${!CFG[@]}"; do save[$k]="${CFG[$k]}"; done
    config_defaults
    declare -gA DEF=()
    for k in "${!CFG[@]}"; do DEF[$k]="${CFG[$k]}"; done
    CFG=()
    for k in "${!save[@]}"; do CFG[$k]="${save[$k]}"; done
}

_preview() { # live theme preview from the form without committing CFG
    local -A save=(); local k
    for k in "${!CFG[@]}"; do save[$k]="${CFG[$k]}"; done
    form_to_cfg; theme_apply
    CFG=()
    for k in "${!save[@]}"; do CFG[$k]="${save[$k]}"; done
}

# ---------- keys ----------

key_in() { # dispatches one key name
    local k="$1"
    if [[ "$k" == ctrl+c ]]; then QUIT=1; return; fi
    if [[ -n "${GLOBAL[$k]:-}" && "$STATE" != confirm ]]; then run_action "${GLOBAL[$k]}"; return; fi
    case "$STATE" in
        home) _key_home "$k" ;;
        config) _key_config "$k" ;;
        *) _key_simple "$k" ;;
    esac
}

buttons() { # prints the current screen's menu entries
    case "$STATE" in
        home) printf '%s\n' "${HOME_MENU[@]}" ;;
        config) printf '%s\n' "${CONFIG_MENU[@]}" ;;
        log) printf '%s\n' "Refresh|log-refresh|r" "Clear|log-clear|c" "Back|back|esc" "Help|help|h" ;;
        doctor) printf '%s\n' "Refresh|doctor-refresh|r" "Back|back|esc" "Help|help|h" ;;
        help) printf '%s\n' "Back|back|esc" ;;
        confirm) printf '%s\n' "Yes|confirm-yes|y" "No|back|n" ;;
    esac
}

_menu_shortcut() { # runs the entry whose key is $1; returns 1 when none
    local it lab act k
    while IFS='|' read -r lab act k; do
        : "$lab"
        if [[ "$k" == "$1" ]]; then run_action "$act"; return 0; fi
    done < <(buttons)
    return 1
}

_menu_run_selected() { local -a b; mapfile -t b < <(buttons); IFS='|' read -r _ act _ <<<"${b[$BTN]}"; run_action "$act"; }

_key_simple() {
    local -a b; mapfile -t b < <(buttons)
    case "$1" in
        tab) MENU_ACTIVE=$(( 1 - MENU_ACTIVE )); BTN=0 ;;
        down) (( MENU_ACTIVE )) || { _scroll 1 || { MENU_ACTIVE=1; BTN=0; }; } ;;
        up|shift+tab) if (( MENU_ACTIVE )); then MENU_ACTIVE=0; else _scroll -1 || true; fi ;;
        left|right) if (( MENU_ACTIVE )); then BTN=$(( (BTN + $(_step "$1") + ${#b[@]}) % ${#b[@]} )); else MENU_ACTIVE=1; BTN=0; fi ;;
        enter) (( MENU_ACTIVE )) && _menu_run_selected ;;
        q|escape) run_action back ;;
        *) _menu_shortcut "$1" || { [[ "$1" == h ]] && run_action help; } ;;
    esac
    return 0
}

_step() { [[ "$1" == left ]] && echo -1 || echo 1; }

_key_home() {
    local -a vis; mapfile -t vis < <(visible)
    local last=$(( 1 + ${#vis[@]} ))
    case "$1" in
        tab|shift+tab) MENU_ACTIVE=$(( 1 - MENU_ACTIVE )); BTN=0; (( MENU_ACTIVE )) || FOCUS=0 ;;
        down) if (( ! MENU_ACTIVE )); then if (( FOCUS >= last )); then MENU_ACTIVE=1; BTN=0; else FOCUS=$(( FOCUS + 1 )); fi; fi ;;
        up) if (( MENU_ACTIVE )); then MENU_ACTIVE=0; FOCUS=$last; elif (( FOCUS > 0 )); then FOCUS=$(( FOCUS - 1 )); fi ;;
        left|right) if (( MENU_ACTIVE )); then BTN=$(( (BTN + $(_step "$1") + ${#HOME_MENU[@]}) % ${#HOME_MENU[@]} )); fi ;;
        enter) if (( MENU_ACTIVE )); then _menu_run_selected; elif (( FOCUS == 0 )); then _add_item; fi ;;
        escape) run_action back ;;
        *)
            if (( FOCUS < 2 && ! MENU_ACTIVE )); then # typing into New item / Search
                local ch=""
                case "$1" in space) ch=" " ;; ?) ch="$1" ;; esac
                if (( FOCUS == 0 )); then
                    if [[ "$1" == backspace ]]; then NEW_TEXT="${NEW_TEXT%?}"; else NEW_TEXT+="$ch"; fi
                else
                    if [[ "$1" == backspace ]]; then SEARCH="${SEARCH%?}"; else SEARCH+="$ch"; fi
                fi
            elif [[ "$1" == q ]]; then run_action back
            elif [[ "$1" == h ]]; then run_action help
            else _menu_shortcut "$1" || true; fi ;;
    esac
    return 0
}

_key_config() {
    local -a sel; mapfile -t sel < <(_fsel)
    local i k="$1"
    case "$k" in
        tab|shift+tab) F_INMENU=$(( 1 - F_INMENU )); F_BTN=0; (( F_INMENU )) || F_FOCUS=${sel[0]}; return 0 ;;
        down) (( F_INMENU )) && return 0
            for i in "${sel[@]}"; do (( i > F_FOCUS )) && { F_FOCUS=$i; return 0; }; done
            F_INMENU=1; F_BTN=0; return 0 ;;
        up) if (( F_INMENU )); then F_INMENU=0; F_FOCUS=${sel[-1]}; else
                for (( i = ${#sel[@]} - 1; i >= 0; i-- )); do (( sel[i] < F_FOCUS )) && { F_FOCUS=${sel[i]}; break; }; done; fi
            return 0 ;;
    esac
    if (( F_INMENU )) && [[ "$k" == left || "$k" == right ]]; then
        F_BTN=$(( (F_BTN + $(_step "$k") + ${#CONFIG_MENU[@]}) % ${#CONFIG_MENU[@]} )); return 0
    fi
    local kind="${F_KIND[$F_FOCUS]}"
    local typing=0; (( ! F_INMENU )) && [[ "$kind" == text ]] && typing=1
    if (( ! typing )) && [[ "$k" =~ ^(left|right|space)$ ]]; then
        if [[ "$kind" == choice ]]; then
            local -a c; read -ra c <<<"${F_CHOICES[$F_FOCUS]}"
            F_IDX[$F_FOCUS]=$(( (F_IDX[F_FOCUS] + $(_step "$k") + ${#c[@]}) % ${#c[@]} ))
            [[ " ${THEME_KEYS[*]} " == *" ${F_KEYS[$F_FOCUS]} "* ]] && _preview
        fi
        return 0
    fi
    case "$k" in
        escape) run_action back ;;
        q) if (( typing )); then F_VAL[$F_FOCUS]+=q; else run_action back; fi ;;
        h) if (( typing )); then F_VAL[$F_FOCUS]+=h; else run_action help; fi ;;
        ctrl+s) run_action save ;;
        enter) if (( F_INMENU )); then IFS='|' read -r _ act _ <<<"${CONFIG_MENU[$F_BTN]}"; run_action "$act"
            elif [[ "$kind" == action ]]; then run_action "${F_ACT[$F_FOCUS]}"; else run_action save; fi ;;
        backspace) (( typing )) && F_VAL[$F_FOCUS]="${F_VAL[$F_FOCUS]%?}" ;;
        space) (( typing )) && F_VAL[$F_FOCUS]+=" " ;;
        ?) (( typing )) && F_VAL[$F_FOCUS]+="$k" ;;
    esac
    (( typing )) && [[ "${F_KEYS[$F_FOCUS]}" == BANNER_TEXT ]] && _preview  # live preview
    return 0
}

# ---------- actions ----------

run_action() {
    MENU_ACTIVE=0 SCROLL=0
    case "$1" in
        back)
            case "$STATE" in
                home) QUIT=1 ;;
                config) theme_apply; STATE=home; STATUS="Config unchanged" ;;
                help) STATE=$HELP_RETURN ;;
                log) STATE=$LOG_RETURN ;;
                doctor) STATE=$DOCTOR_RETURN ;;
                confirm) STATE=$CONFIRM_RETURN; STATUS=Cancelled ;;
            esac ;;
        config) form_new; STATE=config; STATUS="" ;;
        save) form_to_cfg; config_save; theme_apply; STATE=home; STATUS="Config saved"
            local m; m=$(sync_aliases); [[ -n "$m" ]] && STATUS+=" · $m"; logf "config saved" ;;
        help) [[ "$STATE" != help ]] && HELP_RETURN=$STATE; STATE=help ;;
        log|log-refresh) [[ "$STATE" != log ]] && LOG_RETURN=$STATE; STATE=log ;;
        doctor|doctor-refresh) [[ "$STATE" != doctor ]] && DOCTOR_RETURN=$STATE; DOCTOR_TEXT=$(doctor 2>&1 || true); STATE=doctor; STATUS="" ;;
        log-clear) : >"$(log_path)" 2>/dev/null || true; STATUS="Log cleared" ;;
        edit-config) WANTS_EDITOR=1 ;;
        reload) config_load || true; theme_apply; [[ "$STATE" == config ]] && form_new; STATUS="Config reloaded from disk" ;;
        theme-reset)
            if [[ "$STATE" == config ]]; then form_reset_theme; STATUS="Theme reset to default — Save to keep it"
            else config_defaults_copy; local k; for k in "${THEME_KEYS[@]}"; do CFG[$k]="${DEF[$k]}"; done
                config_save; theme_apply; STATUS="Theme reset to default and saved"; fi ;;
        uninstall) _ask uninstall "Uninstall $APP_NAME? Removes the launcher and the shell alias (config is kept)." ;;
        confirm-yes) case "$CONFIRM_ACTION" in uninstall) do_uninstall; QUIT=1 ;; delete) _delete_selected ;; esac ;;
        delete)
            local it; it=$(_selected)
            if [[ -z "$it" ]]; then STATUS="Select an item first (↓ into the list)"
            elif truthy "${CFG[CONFIRM_DELETE]}"; then _ask delete "Delete \"$it\"?"
            else _delete_selected; fi ;;
    esac
    return 0
}

_ask() { [[ "$STATE" != confirm ]] && CONFIRM_RETURN=$STATE; STATE=confirm CONFIRM_ACTION=$1 CONFIRM_TEXT=$2 MENU_ACTIVE=1 BTN=1; }

_selected() { local -a vis; mapfile -t vis < <(visible); local i=$(( FOCUS - 2 )); (( i >= 0 && i < ${#vis[@]} )) && printf '%s' "${vis[$i]}"; return 0; }

_add_item() {
    local v="${NEW_TEXT#"${NEW_TEXT%%[![:space:]]*}"}"
    [[ -z "$v" ]] && return 0
    ITEMS+=("$v"); items_save; NEW_TEXT=""; STATUS="Added $v"; logf "added item $v"
}

_delete_selected() {
    local it k; it=$(_selected); STATE=home
    [[ -z "$it" ]] && return 0
    local -a keep=(); for k in "${ITEMS[@]}"; do [[ "$k" != "$it" ]] && keep+=("$k"); done
    ITEMS=("${keep[@]}"); items_save; STATUS="Deleted $it"; logf "deleted item $it"
}

# ---------- view ----------

# live — info block under the banner (LIVE_INFO): full = items · search · host IP,
# compact = items, off = nothing.
# Shown only while there is live data (the demo: items exist — gu: the tunnel
# is up). Icons, no words (Help → Icons explains them), no bold.
live() {
    local s mode="${UI_LIVE_INFO:-full}" sep='   ' host
    [[ "$mode" == off ]] && return 0
    (( ${#ITEMS[@]} )) || return 0
    s="$(accentc $'\U000F0279') $(livec "${#ITEMS[@]}")"
    if [[ "$mode" == full ]]; then
        [[ -n "$SEARCH" ]] && s+="$sep$(accentc $'\U000F0349') $(livec "$SEARCH")"
        host=$(ip -4 route get 192.0.2.1 2>/dev/null | sed -n 's/.* src \([0-9.]*\).*/\1/p')
        s+="$sep$(accentc $'\U000F0317') $(livec "${host:-—}")"
    fi
    printf '%s' "$s"
}

_tail() { # $1 sel $2 active, then entries
    local sel=$1 act=$2; shift 2
    [[ -n "$STATUS" ]] && printf '\n%s\n' "$(flash "$STATUS")"
    printf '\n%s\n' "$(menu "$sel" "$act" "$@")"
}

_room() { (( ${LINES_:-0} > 0 )) || { echo 0; return; }; echo $(( LINES_ - $(printf '%s' "$1" | wc -l) - $(printf '%s' "$2" | wc -l) - 1 )); }

_scroll() { # d → 0 when moved
    local title body head tail n room ms nxt
    body=$(_body)
    head="$(banner "$(live)")"$'\n\n'"$(title x)"$'\n'
    local -a b; mapfile -t b < <(buttons)
    tail=$(_tail "$BTN" "$MENU_ACTIVE" "${b[@]}")
    n=$(printf '%s\n' "$body" | wc -l); room=$(_room "$head" "$tail")
    ms=0; (( room > 0 && n > room )) && ms=$(( n - ( (room < 5 ? 5 : room) - 2) ))
    if (( SCROLL == 0 && $1 > 0 )); then nxt=2; elif (( SCROLL == 2 && $1 < 0 )); then nxt=0; else nxt=$(( SCROLL + $1 )); fi
    (( nxt > ms )) && nxt=$ms; (( nxt < 0 )) && nxt=0
    (( nxt == SCROLL )) && return 1
    SCROLL=$nxt; return 0
}

# _shortcut_rows — appends to `rows` ONE truthful list keyed by shortcut, merged
# over every screen's menu: a key that means the same everywhere is listed once
# ("h: Help"); one that differs lists every meaning with its screens
# ("r: Refresh (Log, Doctor)"); one that exists on a single screen other than
# the first names it ("^s: Save (Config)"). q/Esc are left to the navigation rows.
_shortcut_rows() {
    local -a order=(); local -A labs=() scrs=()
    local scr name st lab act k first="" key l n parts uniq
    for scr in Home:home Log:log Doctor:doctor Confirm:confirm Config:config; do
        name="${scr%%:*}" st="${scr#*:}"; [[ -z "$first" ]] && first=$name
        while IFS='|' read -r lab act k; do
            : "$act"
            [[ -z "$k" || "$k" == q || "$k" == esc ]] && continue
            [[ -v "labs[$k]" ]] || { order+=("$k"); labs[$k]=""; scrs[$k]=""; }
            labs[$k]+="$lab"$'\n'; scrs[$k]+="$name"$'\n'
        done < <(STATE=$st buttons)
    done
    for k in "${order[@]}"; do
        key="${k//\// / }"
        uniq=$(printf '%s' "${labs[$k]}" | awk '!s[$0]++')
        n=$(printf '%s' "${labs[$k]}" | grep -c .)
        if (( $(wc -l <<<"$uniq") == 1 )); then
            if (( n == 1 )) && [[ "${scrs[$k]%$'\n'}" != "$first" ]]; then rows+=("$key" "$uniq (${scrs[$k]%$'\n'})"); else rows+=("$key" "$uniq"); fi
        else
            parts=""
            while IFS= read -r l; do
                parts+="${parts:+ · }$l ($(paste -d'\n' <(printf '%s' "${labs[$k]}") <(printf '%s' "${scrs[$k]}") | awk -F'\t' -v L="$l" '$1==L{print $2}' | paste -sd, | sed 's/,/, /g'))"
            done <<<"$uniq"
            rows+=("$key" "$parts")
        fi
    done
}

# _help_body — ONE general Shortcuts section generated from HOME_MENU and
# CONFIG_MENU (nothing hand-written that could drift from the keys), the
# commands, then the application text LAST (aliases, integrations).
_help_body() {
    _sec() { printf '\n%s\n' "$(header "$1")"; shift; while (( $# )); do printf '%s\n' "$(info "$1" "$2")"; shift 2; done; }
    _para() { printf '\n%s\n' "$(header "$1")"; printf '%s\n' "$2" | fold -s -w $(( ${COLS:-100} - ${#INDENT} - 2 )) | while IFS= read -r l; do printf '%s%s\n' "$INDENT" "$(fg "$l")"; done; }
    _sec About Version "$APP_NAME v$VERSION   bash $BASH_VERSION" Config "$(config_path)" Log "$(log_path)"
    _sec "Icons (under the banner, while there are items)" $'\U000F0279  Items' "number of items" \
        $'\U000F0349  Search' "the current search" $'\U000F0317  Host' "this machine's LAN address"
    local -a rows=("Tab, ↓" "into the menu (first entry selected)" "← / →" "between menu entries · change a choice in Config" \
        "Tab, ↑" "out of the menu, back to the top" "↑ / ↓" "rows · scroll · fields in Config" \
        Enter "run the selected entry · add an item · save in Config" "q, Esc" "back — quit on Home (q types inside a text field)" Ctrl+C quit)
    local it lab act k
    _shortcut_rows
    rows+=("^l ^e ^r ^t ^u ^d" "also from Home: log · edit config · reload · reset theme · uninstall · doctor")
    _sec Shortcuts "${rows[@]}"
    _sec Commands "$APP_NAME tui" "this interface (default)" "$APP_NAME install" "launcher, shell alias, config file in the current format" \
        "$APP_NAME doctor" "diagnostics (also Config → Doctor)" "$APP_NAME uninstall" "remove launcher + alias" "$APP_NAME version" version "$APP_NAME help" "CLI help"
    _para Aliases "\`$APP_NAME\` is a shell alias for the launcher in ~/.local/bin (Config → Shell shows where it is defined; SHELL_ALIASES=false removes it)."
    _para Integrations "TODO($APP_NAME): describe each integration and backend here — what it does, where its files live and which config key turns it on (gu: Remmina GU, Bitwarden, KeePass)."
}

_body() {
    case "$STATE" in
        log) printf '\n%s  %s\n' "$(header Log)" "$(muted "$(log_path)")"
            if [[ -s "$(log_path)" ]]; then tail -n 200 "$(log_path)" | while IFS= read -r l; do printf '%s  %s\n' "$INDENT" "$(fg "$l")"; done
            else printf '%s  %s\n' "$INDENT" "$(muted '(empty)')"; fi ;;
        help) _help_body ;;
        doctor) printf '\n%s\n' "$(header Checks)"
            while IFS= read -r l; do
                if [[ "$l" == *✗* ]]; then printf '%s%s\n' "$INDENT" "$(errc "$l")"; else printf '%s%s\n' "$INDENT" "$(fg "$l")"; fi
            done <<<"$DOCTOR_TEXT" ;;
        confirm) printf '\n%s%s\n' "$INDENT" "$(fg "$CONFIRM_TEXT")" ;;
        *) _home_body ;;
    esac
}

_home_body() {
    local -a vis; mapfile -t vis < <(visible)
    local n="${CFG[ITEMS_SHOWN]}" cur=$(( FOCUS - 2 )) start=0 i f nt st
    [[ "$n" =~ ^[0-9]+$ ]] || n=10
    printf '\n%s\n' "$(header Overview)"
    if [[ -n "${CFG[GREETING]}" ]]; then printf '%s\n' "$(info Greeting "${CFG[GREETING]}")"; else printf '%s  %s %s\n' "$INDENT" "$(fg "$(printf '%-*s' "$LABEL_WIDTH" 'Greeting:')")" "$(muted —)"; fi
    printf '%s\n' "$(info Items "${#ITEMS[@]} · $(items_path)")"
    printf '%s\n' "$(info Config "$(config_path)")"
    printf '\n%s\n' "$(header Items)"
    nt=$(fg "$NEW_TEXT"); [[ -z "$NEW_TEXT" ]] && nt=$(muted "type and press Enter")
    st=$(fg "$SEARCH"); [[ -z "$SEARCH" ]] && st=$(muted filter)
    f=0; (( ! MENU_ACTIVE && FOCUS == 0 )) && f=1; printf '%s\n' "$(row "$f" "New item" "$nt")"
    f=0; (( ! MENU_ACTIVE && FOCUS == 1 )) && f=1; printf '%s\n' "$(row "$f" Search "$st")"
    (( ${#vis[@]} == 0 )) && printf '%s  %s\n' "$INDENT" "$(muted '(no items)')"
    (( cur >= n )) && start=$(( cur - n + 1 ))
    (( start > 0 )) && printf '%s%s\n' "$INDENT" "$(muted "  ↑ $start more")"
    for (( i = start; i < ${#vis[@]} && i < start + n; i++ )); do
        if (( ! MENU_ACTIVE && i == cur )); then printf '%s%s%s\n' "$INDENT" "$(key '▶ ')" "$(key "${vis[$i]}")"
        else printf '%s  %s\n' "$INDENT" "$(fg "${vis[$i]}")"; fi
    done
    (( start + n < ${#vis[@]} )) && printf '%s%s\n' "$INDENT" "$(muted "  ↓ $(( ${#vis[@]} - start - n )) more")"
    return 0
}

_config_view() {
    local i lines=() focus_line=-1 focused v arr
    for i in "${!F_KIND[@]}"; do
        focused=0; (( ! F_INMENU && i == F_FOCUS )) && focused=1
        [[ "${F_KIND[$i]}" == header ]] && (( i > 0 )) && lines+=("")
        (( focused )) && focus_line=${#lines[@]}
        case "${F_KIND[$i]}" in
            header) lines+=("$(header "${F_LABEL[$i]}")") ;;
            subheader) lines+=("" "$(subheader "${F_LABEL[$i]}")") ;;
            info) lines+=("$(info "${F_LABEL[$i]}" "${F_VAL[$i]}")") ;;
            text) v=$(fg "${F_VAL[$i]}"); (( focused )) && v+=$(key '▏'); lines+=("$(row "$focused" "${F_LABEL[$i]}" "$v")") ;;
            choice) if (( focused )); then arr=key; else arr=muted; fi
                lines+=("$(row "$focused" "${F_LABEL[$i]}" "$($arr '‹ ')$(fg "$(_fchoice_val "$i")")$($arr ' ›')")") ;;
            action) if (( focused )); then lines+=("$INDENT$(key '▶ ')$(key '[')$(key "${F_LABEL[$i]}")$(key ']')")
                else lines+=("$INDENT  $(key '[')$(fg "${F_LABEL[$i]}")$(key ']')"); fi ;;
        esac
    done
    local head tail room
    head="$(banner "$(live)")"$'\n\n'"$(title Config)"$'\n'
    tail=$(_tail "$F_BTN" "$F_INMENU" "${CONFIG_MENU[@]}")
    room=$(_room "$head"$'\n' "$tail")
    printf '%s\n' "$head"
    if (( room > 0 && ${#lines[@]} > room )); then
        (( focus_line < 0 )) && focus_line=$(( ${#lines[@]} - 1 ))
        printf '%s\n' "${lines[@]}" | window_at $(( focus_line - ((room < 5 ? 5 : room) - 2) / 2 )) "$room"
    else
        printf '%s\n' "${lines[@]}"
    fi
    printf '%s\n' "$tail"
}

view() {
    if [[ "$STATE" == config ]]; then _config_view; return; fi
    local t head tail body room
    case "$STATE" in log) t=Log ;; help) t=Help ;; doctor) t=Doctor ;; confirm) t=Confirm ;; *) t=Home ;; esac
    head="$(banner "$(live)")"$'\n\n'"$(title "$t")"
    body=$(_body)
    local -a b; mapfile -t b < <(buttons)
    tail=$(_tail "$BTN" "$MENU_ACTIVE" "${b[@]}")
    room=$(_room "$head"$'\n' "$tail")
    printf '%s\n' "$head"
    if [[ "$STATE" == home ]]; then printf '%s\n' "$body"; else printf '%s\n' "$body" | window_at "$SCROLL" "$room"; fi
    printf '%s\n' "$tail"
}

# ---------- terminal ----------

read_key() { # sets KEY from the TTY (raw: arrows, F1, Ctrl keys)
    local c rest="" r
    IFS= read -rsn1 c </dev/tty || { KEY=ctrl+c; return; }
    case "$c" in
        $'\e') while IFS= read -rsn1 -t 0.02 r </dev/tty; do rest+="$r"; done
            case "$rest" in "[A") KEY=up ;; "[B") KEY=down ;; "[C") KEY=right ;; "[D") KEY=left ;;
                "[Z") KEY=shift+tab ;; OP|"[11~") KEY=f1 ;; "") KEY=escape ;; *) KEY=unknown ;; esac ;;
        $'\t') KEY=tab ;; "") KEY=enter ;; $'\x7f'|$'\b') KEY=backspace ;; " ") KEY=space ;;
        $'\x13') KEY=ctrl+s ;; $'\x0c') KEY=ctrl+l ;; $'\x05') KEY=ctrl+e ;; $'\x12') KEY=ctrl+r ;;
        $'\x14') KEY=ctrl+t ;; $'\x15') KEY=ctrl+u ;; $'\x03') KEY=ctrl+c ;;
        *) KEY="$c" ;;
    esac
}

tui_run() {
    [[ -t 0 && -t 1 ]] || { echo "$APP_NAME: the TUI needs an interactive terminal" >&2; return 1; }
    app_init
    logf "start $APP_NAME v$VERSION"
    local old; old=$(stty -g </dev/tty)
    stty -ixon -isig -echo </dev/tty
    printf '\e[?1049h\e[?25l'
    trap 'stty "$old" </dev/tty; printf "\e[?25h\e[?1049l"' EXIT
    while (( ! QUIT )); do
        COLS=$(tput cols) LINES_=$(tput lines)
        printf '\e[H\e[2J%s' "$(view)"
        read_key
        key_in "$KEY"
        if (( WANTS_EDITOR )); then
            WANTS_EDITOR=0
            stty "$old" </dev/tty; printf '\e[?25h\e[?1049l'
            ${VISUAL:-${EDITOR:-vi}} "$(config_path)" </dev/tty >/dev/tty
            stty -ixon -isig -echo </dev/tty; printf '\e[?1049h\e[?25l'
            run_action reload
        fi
    done
}
