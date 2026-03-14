# codexsm Architecture Notes

## Decoupling Status

Current codebase is acceptable for the current release scope and does not require a mandatory large refactor before shipping.

Known hot spots:

- `cli` root still carries some shared orchestration/render helpers (`group.go`, `pager.go`, `agents.go`) and can continue converging toward thinner routing-only root files.
- `session/scanner/*` and `session/migrate/*` are split subpackages, but scan and migration hot paths still need benchmark-driven tuning.
- `tui/*` is more modular, but narrow-width behavior still depends on coordinated changes across layout metrics, keybar rendering, and info-row formatting.

## Architecture Design

`codexsm` follows a layered approach. The current dependency view is:

```text
External/runtime:
- Go std + encoding/json/v2
- cobra
- bubbletea / lipgloss / go-runewidth

                         +----------------------+
                         | config/*             |
                         | path/config resolve  |
                         +----------+-----------+
                                    |
                                    v
+------------------+      +----------------------+----------------------+
| main.go          | ---> | cli/root.go                                   |
|                  |      | registers command tree                        |
+------------------+      +----------------------+----------------------+
                                    |
                                    v
            +-----------------------+-----------------------+
            | cli root commands                             |
            | group/agents/completion/version/session       |
            +-----------------------+-----------------------+
                                    |
                  +-----------------+-----------------+
                  |                                   |
                  v                                   v
    +-------------------------------+    +-------------------------------+
    | cli subpackages               |    | tui/*                         |
    | cli/config                    |    | command/app/state/actions     |
    | cli/delete                    |    | layout/render/view/text/theme |
    | cli/doctor                    |    | + tui/preview/*               |
    | cli/list                      |    +-------------------------------+
    | cli/restore                   |                    |
    | cli/util                      |                    +----> bubbletea/lipgloss/runewidth
    +---------------+---------------+
                    |
                    v
          +---------+----------+
          | usecase/*          |
          | list/group/action  |
          | doctor/tui/preview |
          +----+-----------+---+
               |           |
               |           +----------------------+
               |                                  |
               v                                  v
   +---------------------------+      +------------------------+
   | session/*                 |      | audit/*                |
   | selector/delete/restore   |      | batch/action logs      |
   | risk/integrity            |      +------------------------+
   +-----------+---------------+
               |
      +--------+-----------------------+
      |                                |
      v                                v
+-------------+               +---------------------+
|scanner/*    |               | session/migrate/*   |
|scan/head/io |               | exec/batch/index/sql|
+-------------+               +----------+----------+
                                          |
                                          v
                                 +------------------+
                                 | util/file.go     |
                                 +------------------+

All layers may use:
- Go std + encoding/json/v2
- internal/ops/* (preview mode / interactive confirm helpers)
```

1. Entry and command wiring:
- `main.go`
- `cli/root.go`

2. Command layer:
- root command files under `cli/*.go` (`group`, `agents`, `completion`, `version`, `session`, shared pager/ansi/logging/runtime ports).
- subpackage command implementations:
  - `cli/config/command.go`
  - `cli/delete/command.go`
  - `cli/doctor/*.go`
  - `cli/list/*.go`
  - `cli/restore/command.go`
  - `cli/util/util.go`
- migration command:
  - `cli/migrate.go` (mounted under `session migrate`)

3. TUI package:
- `tui/command.go`
- `tui/app.go`
- `tui/state.go`
- `tui/actions.go`
- `tui/view.go`, `tui/render.go`, `tui/layout.go`, `tui/theme.go`, `tui/text.go`
- `tui/preview/*` (`build/index/model/service/types`)

4. Domain and storage logic:
- `session/*` for model/filter/risk/integrity/delete/restore operations
- `session/scanner/*` for scanning and head extraction
- `session/migrate/*` for migration batch/index/rollout execution
- `usecase/*` for command-level orchestration shared by CLI/TUI
- `audit/*` for action logs
- `config/*` for path and app config resolution
- `internal/ops/*` for shared operation helpers (`preview mode`, interactive confirms)
- `util/file.go` for move/copy file helpers

5. Test support:
- `internal/testsupport/*` fixture sandbox helpers

Rules:

- CLI and TUI reuse the same core session/audit logic.
- destructive actions default to simulation (`dry-run`) paths.
- action logging stays centralized in `audit`.
- each batch operation is tagged with a `batch_id` for traceability and rollback.

Boundary intent:

- `cli/*` should stay thin orchestration and output adaptation.
- `cli` subpackages should own command-specific argument validation and output formatting.
- `tui/*` should own interaction state, key handling, and rendering.
- `tui/preview/*` should stay preview-specific and avoid reverse dependencies to CLI.
- `session/*`, `audit/*`, and `config/*` should remain reusable by both CLI and TUI.

Shared session-processing boundaries:

- `session/scanner/head.go` and `session/scanner/parse.go` build conversation heads used by list/group/TUI flows.
- `usecase/preview.go` extracts normalized preview messages; `tui/preview/*` renders/stores preview lines and index records.
- `session/risk.go` and `session/integrity.go` separate base health risk detection from optional integrity verification.
- `usecase/list.go`, `usecase/group.go`, `usecase/delete.go`, and `usecase/restore.go` own most command-facing data preparation so CLI wrappers mainly handle flags and rendering.

Rollback flow:

1. `delete` (soft-delete) writes one `batch_id` into action logs.
2. `restore --batch-id <id>` resolves session ids from audit logs.
3. restore scans trash and restores matched sessions under normal safety guards.

## Performance Hot Paths

The current hot paths that deserve benchmark attention are:

- session tree scanning and selector filtering:
  - `session/bench_test.go`
  - `session/scanner/*.go`
- CLI table/JSON/risk rendering:
  - `cli/list/bench_test.go`
  - `cli/doctor/bench_test.go`
- TUI preview construction and preview index persistence:
  - `tui/bench_test.go`
  - `tui/preview/*.go`

Current baselines and rerun commands are tracked in [docs/BENCHMARKS.md](./BENCHMARKS.md).

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
- titles: `title_tree`, `title_preview`, `group`, `accent_group`
- selection: `selected_fg`, `selected_bg`, `cursor_active`, `cursor_inactive`
- keybar: `keys_label`, `keys_key`, `keys_sep`, `keys_text`
- info/status: `info_header`, `info_value`, `status`, `status_info`
- preview roles: `prefix_user`, `prefix_assistant`, `prefix_other`, `prefix_default`
- tag highlighting: `tag_default`, `tag_system`, `tag_lifecycle`, `tag_danger`, `tag_success`
- semantic state roles: `status_ok`, `status_warn`, `status_risk`

Semantic usage in TUI:

- tree/group accent uses `accent_group` (not warning colors)
- warning text and warning markers use `status_warn`
- risk/error text and critical markers use `status_risk`
- neutral risk/readout uses `status_info`

Rendering note:

- main panes inherit the terminal's default background instead of painting the theme `bg`
- theme `bg` is reserved for local contrast needs such as foreground-on-accent combinations
- grouped tree supports runtime fold state: `z` toggles the selected session's group, `Z` expands all groups
- widths below normal threshold automatically switch to compact mode (minimum runtime width: 81 columns), using stacked panes and compact key/tree/footer text

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
