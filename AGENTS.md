# Repository Agent Notes

## TUI Compile And Visual Verification Flow

When validating TUI layout/interaction changes, run this exact sequence:

1. Compile with `just build`.
2. Launch TUI in Alacritty (new window), for example:
   - `alacritty -t codex-sm-tui-check -e sh -lc 'cd /home/omega/ai-workspace/codex-session-manager; ./codex-sm tui --limit 80; exec sh'`
3. Locate the TUI window via `hyprctl -j clients` and record:
   - `address`
   - geometry (`at` + `size`)
4. Capture screenshot with `grim` using that geometry.
5. Inspect the screenshot and confirm:
   - top `KEYS` panel visible
   - left tree panel and right preview/bar panel alignment
   - focused pane highlight is clear
   - preview and tree scrolling behavior is correct
6. Close the temporary verification terminal window when done.

