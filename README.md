# codexsm

`codexsm` is a safety-first local Codex session manager written in Go.

## Quick Links

| Topic | Link |
| --- | --- |
| Architecture | [docs/ARCHITECTURE.md](./docs/ARCHITECTURE.md) |
| Command Guide | [docs/COMMANDS.md](./docs/COMMANDS.md) |
| Release Checklist | [docs/RELEASE.md](./docs/RELEASE.md) |
| Docs Index | [docs/INDEX.md](./docs/INDEX.md) |
| Changelog | [CHANGELOG.md](./CHANGELOG.md) |

## Compatibility

- Go: `1.26+`
- Required experiment: `GOEXPERIMENT=jsonv2`
- JSON packages: `encoding/json/v2`, `encoding/json/jsontext`

> [!IMPORTANT]
> `GOEXPERIMENT=jsonv2` is required for build, install, and test.

```bash
export GOEXPERIMENT=jsonv2
```

## Install

```bash
GOEXPERIMENT=jsonv2 go install github.com/MysticalDevil/codexsm@v0.2.4
```

Or with `mise`:

```bash
GOEXPERIMENT=jsonv2 mise install go:github.com/MysticalDevil/codexsm@v0.2.4
```

## Quick Start

```bash
# List sessions
codexsm list

# Open TUI
codexsm tui

# Grouped TUI
codexsm tui --group-by host

# Run health checks
codexsm doctor

# Dry-run delete
codexsm delete --id-prefix 019ca9

# Dry-run restore from trash
codexsm restore --id-prefix 019ca9
```

> [!TIP]
> For complete examples and command flags, use [docs/COMMANDS.md](./docs/COMMANDS.md).

## At A Glance

| Area | Summary |
| --- | --- |
| Browse | `list`, `group`, and `tui` for session discovery |
| Safety | `dry-run` by default, explicit `--confirm` for real actions |
| Recovery | `batch_id`-based rollback with `restore --batch-id` |
| Diagnostics | `doctor` and `config` validation tooling |

## Core Features

- Session listing and grouping (`list`, `group`)
- Interactive browser (`tui`) with theme support
- Safe delete/restore workflow (`dry-run` by default)
- TUI pending-action confirmation shown in bottom keybar (`Y/N`) with stronger visibility
- TUI delete keeps navigation continuity by advancing selection to the next session
- Batch rollback via `restore --batch-id`
- Diagnostics and configuration (`doctor`, `config`)

## Safety Model

- Destructive actions default to simulation (`--dry-run=true`).
- Real execution requires explicit opt-in (`--dry-run=false --confirm`).
- Multi-target real execution requires additional approval (`--yes` or interactive confirmation).
- Soft-delete is default; hard delete is explicit (`--hard`).
- Operation logs include `batch_id` for audit and rollback.

> [!NOTE]
> Recommended flow: preview first, then real execution with explicit confirmation.

## Configuration

Config path resolution:

- `$CSM_CONFIG` when set
- otherwise `~/.config/codexsm/config.json`

Example:

```json
{
  "sessions_root": "~/.codex/sessions",
  "trash_root": "~/.codex/trash",
  "log_file": "~/.codex/codexsm/logs/actions.log",
  "tui": {
    "group_by": "host",
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

TUI note:

- main panes follow the terminal's default background
- theme colors still control borders, titles, selection, keybar, and preview roles
- `bg` remains available for local emphasis, such as highlighted action prompts

## Build And Dev

```bash
just build
just check
just cover-gate
just bench-gate
codexsm doctor risk --sessions-root ./testdata/fixtures/risky-static/sessions --format json --sample-limit 5
just check-release 0.2.4
```

Release build example:

```bash
GOEXPERIMENT=jsonv2 go build -ldflags="-X main.version=0.2.4" -o codexsm .
```

## License

BSD 3-Clause. See [LICENSE](./LICENSE).
