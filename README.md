# codexsm

`codexsm` is a safety-first local Codex session manager written in Go.

Project design notes:

- [Architecture Notes](./docs/ARCHITECTURE.md)

## Compatibility

- Go: `1.26+`
- Required experiment: `GOEXPERIMENT=jsonv2`
- JSON packages: `encoding/json/v2`, `encoding/json/jsontext`

> [!IMPORTANT]
> `GOEXPERIMENT=jsonv2` is currently required for both build and install.
> If this env is missing, `encoding/json/v2` and `encoding/json/jsontext` will fail to compile.
>
> Recommended:
>
> ```bash
> export GOEXPERIMENT=jsonv2
> ```

It provides:

- Session listing (`list`)
- Session grouping (`group`)
- Optional terminal UI (`tui`)
- Safe deletion (`delete`, dry-run by default)
- Session restore from trash (`restore`, dry-run by default)
- Environment diagnostics (`doctor`)

## Features

- Safe by default:
  - `delete` runs with `--dry-run=true`
  - `restore` runs with `--dry-run=true`
- Real operations require explicit intent:
  - `--dry-run=false --confirm`
- Default delete mode is soft delete (move to trash)
- Optional hard delete with `--hard`
- Interactive confirmation for real delete/restore (enabled by default)
- Readable CLI output:
  - compact list view by default
  - `HEAD` with noise filtering
  - `HOST` column from session cwd (`~` under home)
  - customizable head width (`--head-width`)
  - detailed mode
  - pager mode (Vim keys: `j/k/g/G`, plus `a/q`)
  - colored help/output
  - `json/table/csv/tsv` formats
- Optional TUI mode:
  - keyboard navigation (`j/k/g/G`)
  - tree grouping (`--group-by month|day|health|host|none`)
  - configurable source (`--source sessions|trash`)
  - selected session detail view
  - in-TUI delete/restore actions with safety guards
  - semantic preview highlighting (`U/A` role markers and `<...>` tag coloring)
  - built-in themes (`tokyonight`, `catppuccin`, `gruvbox`, `onedark`, `nord`, `dracula`)
  - custom theme overrides (`--theme-color key=value` or config file)

## TUI Status

Current TUI is usable for daily browsing and safe cleanup operations:

- stable session tree navigation and preview scrolling
- multiple grouping modes for tree browsing
- readable bottom info bar with host path preview
- dry-run and real delete from TUI source `sessions` (`d`)
- dry-run and real restore from TUI source `trash` (`r`)
- built-in theme presets and user custom color overrides

Current limitations:

- TUI still focuses on single-session actions per keypress (bulk actions stay in CLI)
- layout and color tuning are still being iterated

## Configuration

`codexsm` loads runtime config from:

- `$CSM_CONFIG` (if set)
- otherwise `~/.config/codexsm/config.json`

Example:

```json
{
  "sessions_root": "~/.codex/sessions",
  "trash_root": "~/.codex/trash",
  "log_file": "~/.codex/codexsm/logs/actions.log",
  "tui": {
    "group_by": "month",
    "source": "sessions",
    "theme": "catppuccin",
    "colors": {
      "keys_label": "#ffffff",
      "keys_key": "#89dceb",
      "border_focus": "#f38ba8"
    }
  }
}
```

Common color keys for `tui.colors` / `--theme-color`:

- `bg`, `fg`, `border`, `border_focus`
- `title_tree`, `title_preview`, `group`
- `selected_fg`, `selected_bg`, `cursor_active`, `cursor_inactive`
- `keys_label`, `keys_key`, `keys_sep`, `keys_text`
- `info_header`, `info_value`, `status`
- `prefix_user`, `prefix_assistant`, `prefix_other`, `prefix_default`
- `tag_default`, `tag_system`, `tag_lifecycle`, `tag_danger`, `tag_success`

## Build

When building/testing from source, enable:

```bash
export GOEXPERIMENT=jsonv2
```

```bash
just build
```

Or:

```bash
GOEXPERIMENT=jsonv2 go build -ldflags="-X main.version=0.1.6" -o codexsm .
```

Default local build version is `dev`. Set `VERSION` for release builds:

```bash
VERSION=0.1.6 just build
```

## Install

Preferred (Go):

```bash
GOEXPERIMENT=jsonv2 go install github.com/MysticalDevil/codexsm@v0.1.6
```

With `mise`:

```bash
GOEXPERIMENT=jsonv2 mise install go:github.com/MysticalDevil/codexsm@v0.1.6
```

Note:

- The installed binary name is `codexsm`.

## Quick Start

```bash
# List recent sessions (default limit: 10)
codexsm list

# Launch interactive TUI
codexsm tui

# Launch TUI grouped by host
codexsm tui --group-by host

# Launch TUI with a different source and theme
codexsm tui --source trash --theme gruvbox --theme-color border_focus=#fabd2f

# Detailed list view
codexsm list --detailed

# Custom columns, no header
codexsm list --format csv --no-header --column session_id,health

# Sort by size ascending
codexsm list --sort size --order asc --limit 20

# Show all with pager
codexsm list --limit 0 --pager

# Group by day
codexsm group --by day

# Group by health with sorting and limit
codexsm group --by health --sort count --order desc --limit 5

# Environment checks
codexsm doctor
codexsm doctor --strict

# Dry-run delete (default behavior)
codexsm delete --id-prefix 019ca9

# Real soft delete
codexsm delete --id-prefix 019ca9 --dry-run=false --confirm

# Real hard delete
codexsm delete --id 019ca9c1-3df3-7551-b04b-b2a91c486755 --dry-run=false --confirm --hard

# Dry-run restore from trash
codexsm restore --id-prefix 019ca9

# Real restore
codexsm restore --id-prefix 019ca9 --dry-run=false --confirm
```

## Delete/Restore Safety Model

`delete` and `restore` targets are selected by flags (not positional args):

- `--id <session_id>`
- `--id-prefix <prefix>`
- `--older-than <duration>` (for example `30d`, `12h`)
- `--health <ok|corrupted|missing-meta>`

Rules:

- At least one selector is required
- Dry-run is default
- Real operation requires `--confirm`
- Batch real operation requires approval (`--yes` or interactive confirm)

## Command Help

```bash
codexsm help
codexsm help list
codexsm help group
codexsm help delete
codexsm help restore
codexsm help doctor
codexsm help version
```

## Development

```bash
just fmt
just lint
just test            # unit tests
just test-integration
just test-all
just cover
just cover-unit
just cover-integration
just cover-gate
just check
just check-release 0.1.6

# Coverage requirements
# - unit >= 50%
# - integration >= 65%
just cover-gate
```

Tooling defaults:

- Formatter: `gofumpt`
- Lint: `go vet`

## License

Licensed under the BSD 3-Clause License. See [LICENSE](./LICENSE) for details.
