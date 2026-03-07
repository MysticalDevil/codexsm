# codexsm Architecture Notes

## Decoupling Status

Current codebase is acceptable for the current release scope and does not require mandatory large refactor before shipping.

Known hot spot:

- `cli/delete.go` and `cli/restore.go` still contain command orchestration + output responsibilities and should continue converging to thinner wrappers.
- `session/scanner.go` has been split, but scan/parse/score tuning can still be optimized with benchmarks.

## Architecture Design

`codexsm` follows a layered approach:

1. Entry and command wiring:
- `main.go`
- `cli/root.go`

2. Command layer:
- `cli/list.go`
- `cli/list_columns.go`
- `cli/pager.go`
- `cli/ansi.go`
- `cli/group.go`
- `cli/delete.go`
- `cli/restore.go`
- `cli/tui.go`
- `cli/doctor.go`
- `cli/config.go`

3. TUI package:
- `tui/command.go`
- `tui/view.go`
- `tui/state.go`
- `tui/actions.go`
- `tui/preview.go`
- `tui/render.go`
- `tui/theme.go`
- `tui/text.go`
- `tui/helpers.go`

4. Domain and storage logic:
- `session/*` for scanning/filtering/delete operations
- `audit/*` for action logs
- `config/*` for path and app config resolution
- `internal/ops/*` for shared operation helpers (`preview mode`, interactive confirms)
- `internal/fileutil/*` for move/copy file helpers
- `internal/restoreexec/*` and `internal/deleteexec/*` for operation execution wrappers
- `internal/tui/layout/*` for TUI layout metrics

5. Test support:
- `internal/testsupport/*` fixture sandbox helpers

Rules:

- CLI and TUI reuse the same core session/audit logic.
- `cli/tui.go` is an entry bridge; TUI behavior is implemented in `tui/*`.
- destructive actions default to simulation (`dry-run`) paths.
- action logging stays centralized in `audit`.
- each batch operation is tagged with a `batch_id` for traceability and rollback.

Rollback flow:

1. `delete` (soft-delete) writes one `batch_id` into action logs.
2. `restore --batch-id <id>` resolves session ids from audit logs.
3. restore scans trash and restores matched sessions under normal safety guards.

## Performance Baselines

Current benchmark baselines are tracked in Go benchmark tests:

- `session/bench_test.go`
  - `BenchmarkScanSessions`
  - `BenchmarkFilterSessions`
- `cli/list_bench_test.go`
  - `BenchmarkRenderTable`

These baselines are intended to protect refactors in scan/filter/render hot paths.

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
- default file: `~/.config/codexsm/config.json`
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
