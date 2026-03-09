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
GOEXPERIMENT=jsonv2 go install github.com/MysticalDevil/codexsm@v0.2.5
```

Or with `mise`:

```bash
GOEXPERIMENT=jsonv2 mise install go:github.com/MysticalDevil/codexsm@v0.2.5
```

Experimental branch for performance-oriented users:

- branch: `exp/zig-incremental`
- scope: opt-in `zsession` scan/risk/preview pipeline for users comfortable with experimental native integration
- guide: <https://github.com/MysticalDevil/codexsm/tree/exp/zig-incremental/docs/INSTALL_ZIG_EXPERIMENT.md>

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
just bench-session
just bench-cli
just bench-gate
just bench-tui
just bench-all
just stress-cli
codexsm doctor risk --sessions-root ./testdata/fixtures/risky-static/sessions --format json --sample-limit 5
just gen-sessions-extreme
just gen-sessions-large
just check-release 0.2.5
```

Fixture note:

- `testdata/fixtures/rich/` keeps the general regression corpus.
- `testdata/fixtures/risky-static/` keeps deterministic risk-oriented samples for `doctor risk`.
- `testdata/fixtures/extreme-static/` keeps a small extreme corpus for oversized meta lines, long single messages, no-final-newline files, mixed corruption, and Unicode-heavy previews.
- Larger stress files are intentionally generated on demand via `just gen-sessions-extreme` or `just gen-sessions-large` instead of being committed as multi-megabyte fixtures.
- Lightweight benchmark suites are available through `just bench-session`, `just bench-cli`, and `just bench-tui`; `just stress-cli` is the heavier local-only smoke path for generated large datasets.

Release build example:

```bash
GOEXPERIMENT=jsonv2 go build -ldflags="-X main.version=0.2.5" -o codexsm .
```

## License

BSD 3-Clause. See [LICENSE](./LICENSE).
