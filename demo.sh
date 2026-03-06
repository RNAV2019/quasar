#!/usr/bin/env bash
# ╔══════════════════════════════════════════════════════════════╗
# ║  Quasar TUI Markdown Editor — Demo Recording Script         ║
# ║  Records a Ghostty terminal window with gpu-screen-recorder ║
# ║  and simulates keystrokes with wtype.                       ║
# ║                                                             ║
# ║  Dependencies: ghostty, gpu-screen-recorder, wtype,         ║
# ║                hyprctl, jq, ffmpeg                          ║
# ║  Install: sudo pacman -S wtype jq ffmpeg                   ║
# ╚══════════════════════════════════════════════════════════════╝
set -euo pipefail

# ── Configuration ─────────────────────────────────────────────
OUTPUT="demo.mp4"
COLS=100                     # Terminal columns (~1400px at 18pt)
ROWS=35                      # Terminal rows   (~800px at 18pt)
FONT_SIZE=18
FPS=60

# Typing speeds (ms between keystrokes)
SPEED_NORMAL=50
SPEED_SLOW=100
SPEED_FAST=35
SPEED_MEDIUM=40

# ── Helpers ───────────────────────────────────────────────────
type_text()  { wtype -d "${2:-$SPEED_NORMAL}" "$1"; }
type_slow()  { wtype -d "$SPEED_SLOW" "$1"; }
type_fast()  { wtype -d "$SPEED_FAST" "$1"; }
type_med()   { wtype -d "$SPEED_MEDIUM" "$1"; }
press_enter()   { wtype -k Return; }
press_escape()  { wtype -k Escape; }
press_space()   { wtype -k space; }
press_right()   { wtype -k Right; }
press_colon()   { wtype -k colon; }
press_tab()     { wtype -k Tab; }
pause()         { sleep "${1:-0.5}"; }

get_window_address_by_pid() {
    hyprctl clients -j | jq -r "
        .[] | select(.pid == $1) | .address
    " | head -1
}

cleanup() {
    echo "Cleaning up..."
    [[ -n "${RECORDER_PID:-}" ]] && kill -SIGINT "$RECORDER_PID" 2>/dev/null && wait "$RECORDER_PID" 2>/dev/null
    [[ -n "${GHOSTTY_PID:-}" ]] && kill "$GHOSTTY_PID" 2>/dev/null
    exit
}
trap cleanup EXIT INT TERM

# ── Preflight checks ─────────────────────────────────────────
for cmd in ghostty gpu-screen-recorder wtype hyprctl jq ffmpeg; do
    if ! command -v "$cmd" &>/dev/null; then
        echo "Error: $cmd not found. Install it first."
        exit 1
    fi
done

# ── Hidden setup (clean state) ────────────────────────────────
echo "Setting up clean state..."
mkdir -p ~/Documents/quasar
rm -rf ~/Documents/quasar/demo

# Remove old output
rm -f "$OUTPUT"

# ── Launch Ghostty on workspace 5 ─────────────────────────────
echo "Launching Ghostty..."
hyprctl dispatch workspace 5
sleep 0.3
ghostty \
    --window-width="$COLS" \
    --window-height="$ROWS" \
    --font-size="$FONT_SIZE" \
    --font-family="JetBrainsMono Nerd Font" \
    --title="Quasar Demo" \
    --theme="catppuccin-mocha" \
    --window-padding-x=10 \
    --window-padding-y=10 &
GHOSTTY_PID=$!
sleep 2

# ── Focus window & start recording ────────────────────────────
FULL_RAW="demo_full.mp4"
rm -f "$FULL_RAW"

ADDR=$(get_window_address_by_pid "$GHOSTTY_PID")
if [[ -z "$ADDR" ]]; then
    echo "Error: Could not find Ghostty window with PID $GHOSTTY_PID"
    echo "Available windows:"
    hyprctl clients -j | jq '.[] | {class, pid, title}'
    exit 1
fi

# Get window geometry (logical pixels) and monitor scale for post-crop
read WIN_X WIN_Y WIN_W WIN_H < <(hyprctl clients -j | jq -r "
    .[] | select(.pid == $GHOSTTY_PID) |
    \"\(.at[0]) \(.at[1]) \(.size[0]) \(.size[1])\"
" | head -1)

# hyprctl reports logical pixels; the recording is in native pixels.
# Multiply by the monitor scale factor for correct ffmpeg crop.
SCALE=$(hyprctl monitors -j | jq -r '.[0].scale')
CROP_X=$(jq -n "$WIN_X * $SCALE | round")
CROP_Y=$(jq -n "$WIN_Y * $SCALE | round")
CROP_W=$(jq -n "$WIN_W * $SCALE | round")
CROP_H=$(jq -n "$WIN_H * $SCALE | round")

# Detect monitor
MONITOR=$(gpu-screen-recorder --list-monitors 2>&1 | head -1 | cut -d'|' -f1)
echo "Recording monitor: $MONITOR (scale=${SCALE}, crop ${CROP_W}x${CROP_H} at ${CROP_X},${CROP_Y})"

# Focus the Ghostty window
hyprctl dispatch focuswindow "address:${ADDR}"
sleep 0.5

# Record the full monitor (GPU-accelerated), crop in post
gpu-screen-recorder -w "$MONITOR" -f "$FPS" -k h264 -q very_high -fm cfr -cursor no -o "$FULL_RAW" &
RECORDER_PID=$!
sleep 0.5

# ══════════════════════════════════════════════════════════════
#  DEMO SEQUENCE
# ══════════════════════════════════════════════════════════════

# ── 1. Create notebook via CLI ────────────────────────────────
pause 1
type_text "quasar nb new demo"
pause 0.5
press_enter
pause 2

# ── 2. Open notebook (launches TUI) ──────────────────────────
type_text "quasar nb demo"
pause 0.5
press_enter
pause 2

# ── 3. Create a new note via command mode ─────────────────────
press_colon
pause 0.5
type_text "new"
press_enter
pause 1

# Type note title in dialog
type_med "Automata Theory"
pause 0.3
press_enter
pause 2

# ── 4. Hide file tree for full-width editing ──────────────────
press_space
wtype "f"
pause 1

# ── 5. Navigate past YAML front matter (4 lines: 0-3) ────────
wtype "jjjjo"
pause 0.5
press_enter
pause 0.5

# ── 6. Heading 1 via slash command ────────────────────────────
type_slow "/h1"
pause 0.5
press_enter
pause 0.3
type_med "Deterministic Finite Automata"
pause 0.3
press_enter
press_enter

# ── 7. Body paragraph ────────────────────────────────────────
type_fast "A DFA is a mathematical model of computation"
press_enter
type_fast "that accepts or rejects strings of symbols"
press_enter
type_fast "and processes input one symbol at a time."
press_enter
press_enter

# ── 8. Heading 2 via slash command ────────────────────────────
type_slow "/h2"
pause 0.5
press_enter
pause 0.3
type_med "Formal Definition"
pause 0.3
press_enter
press_enter

# ── 9. Inline math with slash command ─────────────────────────
type_med "A DFA is the 5-tuple "
type_slow "/inlinemath"
pause 0.5
press_enter
pause 0.3
type_med 'M = (Q, \Sigma, \delta, q_0, F)'
press_right
type_med " where"
press_enter
type_fast "each component defines part of the machine."
press_enter
press_enter

# ── 10. Heading 2 ────────────────────────────────────────────
type_slow "/h2"
pause 0.5
press_enter
pause 0.3
type_med "State Diagram"
pause 0.3
press_enter
press_enter

# ── 11. Math block via slash command ──────────────────────────
type_slow "/math"
pause 0.5
press_tab
pause 0.3
press_enter
pause 0.5

# ── 12. Insert automata diagram via snippet (MathOnly) ───────
type_slow "/automata"
pause 0.5
press_enter
pause 0.5

# ── 13. Exit insert mode and move up to trigger render ───────
press_escape
pause 0.5
type_med "kkkkkkkkk"
pause 3

# ── 14. Save the file ────────────────────────────────────────
press_colon
pause 0.3
wtype "w"
press_enter
pause 4

# ══════════════════════════════════════════════════════════════
#  DONE — stop recording
# ══════════════════════════════════════════════════════════════
echo ""
echo "Stopping recording..."
kill -SIGINT "$RECORDER_PID"
wait "$RECORDER_PID" 2>/dev/null
RECORDER_PID=""

echo "Closing Ghostty..."
kill "$GHOSTTY_PID" 2>/dev/null
GHOSTTY_PID=""

# ── Crop full-screen recording to the window region ───────────
echo "Cropping to window region (${CROP_W}x${CROP_H} at ${CROP_X},${CROP_Y})..."
ffmpeg -y -i "$FULL_RAW" \
    -vf "crop=${CROP_W}:${CROP_H}:${CROP_X}:${CROP_Y}" \
    -c:v libx264 -preset fast -crf 18 \
    "$OUTPUT"
rm -f "$FULL_RAW"

echo "Done: $OUTPUT"
