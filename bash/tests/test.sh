#!/usr/bin/env bash
# tests/test.sh — headless tests of the TUI standard (same checks as the Go and
# Python templates). Drives key_in/view without a terminal. Run: bash tests/test.sh
set -uo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TMP=$(mktemp -d); trap 'rm -rf "$TMP"' EXIT
export HOME="$TMP" XDG_DATA_HOME="" XDG_STATE_HOME=""
unset XDG_DATA_HOME XDG_STATE_HOME
APP_NAME=demo APP_DESC="demo app" VERSION=0.1.0 BANNER_ART=""
. "$SCRIPT_DIR/lib/font.sh"; . "$SCRIPT_DIR/lib/standard.sh"; . "$SCRIPT_DIR/lib/config.sh"; . "$SCRIPT_DIR/lib/system.sh"; . "$SCRIPT_DIR/lib/app.sh"

fails=0
plain() { sed 's/\x1b\[[0-9;]*m//g'; }
check() { if eval "$2"; then printf '  ok    %s\n' "$1"; else printf '  FAIL  %s\n' "$1"; fails=$(( fails + 1 )); fi; }
fresh() { STATE=home STATUS="" MENU_ACTIVE=0 BTN=0 SCROLL=0 QUIT=0 NEW_TEXT="" SEARCH="" FOCUS=0; app_init; ITEMS=(alpha bravo charlie); COLS=100 LINES_=0; }
press() { local k; for k in "$@"; do key_in "$k"; done; }

# 1 text indented, exactly one rule (Home, Config, Help)
fresh
for scr in home config help; do
    case $scr in config) fresh; press down down o ;; help) fresh; press down down h ;; esac
    v=$(view | plain)
    rules=$(grep -c '^·' <<<"$v"); bad=$(grep -v '^$' <<<"$v" | grep -v '^·' | grep -vc '^  ')
    check "$scr: one rule, all text indented" "[[ $rules == 1 && $bad == 0 ]]"
done

# 1b no line wider than the terminal (long paths are cut with …)
fresh; COLS=60; v=$(view | plain); widest=$(awk '{ if (length($0) > m) m = length($0) } END { print m }' <<<"$v")
check "no line wider than 60 columns (widest $widest)" "(( widest <= 60 ))"

# 2 menu line + navigation
fresh; v=$(view | plain)
check "menu line format, nothing selected" "grep -q 'Delete x · Config o · Log l · Quit q · Help h' <<<\"\$v\" && ! grep -q '▶ Delete' <<<\"\$v\""
press tab; v=$(view | plain)
check "tab selects the first entry" "(( MENU_ACTIVE == 1 )) && grep -q '▶ Delete' <<<\"\$v\""
press right right; check "→→ moves to the third entry" "(( BTN == 2 ))"
press tab; check "tab leaves the menu, back to the top" "(( MENU_ACTIVE == 0 && FOCUS == 0 ))"

# 3 q ≡ Esc
fresh; press q; check "q types inside a field" "[[ \$NEW_TEXT == q && \$QUIT == 0 ]]"
press escape; check "Esc quits on Home" "(( QUIT == 1 ))"

# 4 Help returns, global shortcuts, Confirm
fresh; press down down o tab h; check "h opens Help from the Config menu" "[[ \$STATE == help ]]"
press q; check "q in Help returns to Config" "[[ \$STATE == config ]]"
press ctrl+u; check "^U asks first, No preselected" "[[ \$STATE == confirm && \$BTN == 1 ]]"
press n; check "n cancels back to Config" "[[ \$STATE == config ]]"

# 5 theme reset outside Config is saved
fresh; CFG[COLOR_HEADER]=red; press ctrl+t
check "^T resets and saves the theme" "[[ \${CFG[COLOR_HEADER]} == ocean ]] && grep -q 'COLOR_HEADER=\"ocean\"' \"\$(config_path)\""

# 6 every field in the config file
fresh; CFG[COLOR_TEXT]=cyan; config_save_fresh
for f in "${FIELDS[@]}"; do _field "$f"; _on_os "$F_OS" || continue
    check "file has $F_KEY" "grep -q '^$F_KEY=\"' \"\$(config_path)\""; done
config_load; check "round trip keeps COLOR_TEXT" "[[ \${CFG[COLOR_TEXT]} == cyan ]]"

# 7 Config fits the terminal
fresh; LINES_=30; press down down o; for i in $(seq 12); do press down; done
v=$(view | plain); check "Config view fits 30 lines" "(( \$(wc -l <<<\"\$v\") <= 30 )) && grep -q '▶' <<<\"\$v\""

# 8 add + delete
fresh; CFG[CONFIRM_DELETE]=false; press z e d enter
check "add an item" "(( \${#ITEMS[@]} == 4 ))"
press down down x; check "delete the selected item" "[[ \${ITEMS[0]} == bravo ]]"

# 9 Config shows where the alias is installed — exactly once
fresh; printf "alias demo='/x'\n" >"$HOME/.zshrc"; press down down o
v=$(view | plain); line=$(grep 'alias demo:' <<<"$v")
check "alias row: installed at ~/.zshrc:1, shown once" "[[ \"\$line\" == *'installed — ~/.zshrc:1' && \$(grep -o '~/.zshrc:1' <<<\"\$line\" | wc -l) == 1 ]]"
rm -f "$HOME/.zshrc"

# 10 banner font + live info modes (preview without saving)
fresh; press down down o
for i in $(seq 1 30); do press down; [[ "${F_KEYS[$F_FOCUS]}" == BANNER_FONT ]] && break; done
press right right; v=$(view | plain)
check "BANNER_FONT preview: half-block logo shown" "grep -q '▀' <<<\"\$v\""
fresh; CFG[LIVE_INFO]=off; theme_apply; v=$(view | plain)
check "LIVE_INFO=off hides the block" "! grep -q '󰉹' <<<\"\$(sed -n '1,12p' <<<\"\$v\")\""
fresh; v=$(live | plain); check "live block: icon + value, no words" "grep -q '󰉹 3' <<<\"\$v\" && ! grep -q items <<<\"\$v\""
fresh; ITEMS=(); check "no live data → no live block" "[[ -z \"\$(live)\" ]]"
fresh; v=$(view | plain); check "LIVE_INFO=full shows items + host IP icon" "grep -q '󰌗' <<<\"\$v\""

# 11 §17: generated Help, Banner sub-heading + text, Doctor, lowercase config menu, Clear, bold selection
fresh; press down down h; v=$(view | plain)
check "help: no per-screen list, no ?" "! grep -qE 'Other screens|Navigation|Everywhere|h, \?, F1' <<<\"\$v\""
check "help: Shortcuts from the menus + app text last" "grep -q Shortcuts <<<\"\$v\" && grep -q 'Doctor (Config)' <<<\"\$v\" && grep -q 'Refresh' <<<\"\$v\" && grep -q 'Clear (Log)' <<<\"\$v\" && grep -q '\^s' <<<\"\$v\" && grep -q Aliases <<<\"\$v\""
fresh; press down down o; v=$(view | plain)
check "config: Banner is an indented sub-heading with Text, Doctor section, lowercase menu" "grep -q '^   *Banner\$' <<<\"\$v\" && grep -q 'Text:' <<<\"\$v\" && grep -q 'Run doctor' <<<\"\$v\" && grep -q 'Save \^s' <<<\"\$v\" && ! grep -q '\^S' <<<\"\$v\""
press ctrl+d; check "ctrl+d from Config → Doctor screen with checks" "[[ \$STATE == doctor ]] && grep -q 'config' <<<\"\$(view | plain)\""
press escape; check "Back from Doctor returns to Config" "[[ \$STATE == config ]]"
fresh; press down down o; for i in $(seq 1 40); do press down; [[ "${F_KEYS[$F_FOCUS]}" == BANNER_TEXT ]] && break; done
press H I; v=$(view | plain); check "BANNER_TEXT previews with a font" "grep -q '█' <<<\"\$v\""
fresh; press down down l; v=$(view | plain); check "log menu says Clear" "grep -q 'Clear c' <<<\"\$v\" && ! grep -q 'Clear log' <<<\"\$v\""
fresh; press tab; v=$(view); check "selected menu entry is bold" "grep -q $'▶ .\\{0,12\\}\e\\[1m' <<<\"\$v\""

echo; if (( fails )); then echo "$fails FAILED"; exit 1; fi; echo "all passed"
