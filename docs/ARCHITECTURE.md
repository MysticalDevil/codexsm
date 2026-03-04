# codex-sm Architecture Notes

## Decoupling Status

Current codebase is acceptable for the current release scope and does not require mandatory large refactor before shipping.

Known hot spot:

- `cli/tui.go` is feature-rich and relatively large; it should be split incrementally (renderer, keymap/action handler, and preview parser) in future iterations.

## Architecture Design

`codex-sm` follows a layered approach:

1. Entry and command wiring:
- `main.go`
- `cli/root.go`

2. Command layer:
- `cli/list.go`
- `cli/group.go`
- `cli/delete.go`
- `cli/restore.go`
- `cli/tui.go`
- `cli/doctor.go`

3. Domain and storage logic:
- `session/*` for scanning/filtering/delete operations
- `audit/*` for action logs
- `config/*` for path and app config resolution

4. Test support:
- `internal/testsupport/*` fixture sandbox helpers

Rules:

- CLI and TUI reuse the same core session/audit logic.
- destructive actions default to simulation (`dry-run`) paths.
- action logging stays centralized in `audit`.

## Theme And Color Conventions

Built-in themes:

- `tokyonight` (default)
- `catppuccin`
- `gruvbox`
- `onedark`
- `nord`
- `dracula`

Theme resolution order:

1. Built-in palette by `--theme` (or `tui.theme` from config)
2. Config overrides (`tui.colors`)
3. CLI overrides (`--theme-color key=value`, highest priority)

Recommended semantic keys:

- base: `bg`, `fg`, `border`, `border_focus`
- titles: `title_tree`, `title_preview`, `group`
- selection: `selected_fg`, `selected_bg`, `cursor_active`, `cursor_inactive`
- keybar: `keys_label`, `keys_key`, `keys_sep`, `keys_text`
- info/status: `info_header`, `info_value`, `status`
- preview roles: `prefix_user`, `prefix_assistant`, `prefix_other`, `prefix_default`
- tag highlighting: `tag_default`, `tag_system`, `tag_lifecycle`, `tag_danger`, `tag_success`

## Third-Party Packages

Core CLI:

- `github.com/spf13/cobra`: command tree and help UX

TUI:

- `github.com/charmbracelet/bubbletea`: event loop/model-update-view architecture
- `github.com/charmbracelet/lipgloss`: styles/layout/borders/colors
- `github.com/mattn/go-runewidth`: width-safe CJK and mixed text rendering

Rationale:

- mature ecosystem
- predictable cross-terminal behavior
- strong fit for keyboard-first interfaces

## Config Usage

Config model:

- `config.AppConfig`
- default file: `~/.config/codex-sm/config.json`
- override path: `$CSM_CONFIG`

Main keys:

- `sessions_root`
- `trash_root`
- `log_file`
- `tui.group_by`
- `tui.source`
- `tui.theme`
- `tui.colors`

Resolution and precedence:

1. command flags
2. config file (`$CSM_CONFIG` or default path)
3. built-in defaults

Path behavior:

- `~` is expanded via `config.ResolvePath`
- missing config file is non-fatal (zero-value config)

## JSON Runtime Requirement

This project uses:

- `encoding/json/v2`
- `encoding/json/jsontext`

Build/install/test must enable:

- `GOEXPERIMENT=jsonv2`
