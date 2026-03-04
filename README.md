# codex-sm

`codex-sm` is a safety-first local Codex session manager written in Go.

It provides:

- Session listing (`list`)
- Session grouping (`group`)
- Safe deletion (`delete`, dry-run by default)
- Session restore from trash (`restore`, dry-run by default)

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

## Build

This project uses Go JSON v2 packages (`encoding/json/v2`, `encoding/json/jsontext`).
When building/testing from source, enable:

```bash
export GOEXPERIMENT=jsonv2
```

```bash
just build
```

Or:

```bash
go build -o codex-sm .
```

## Install

Preferred (Go):

```bash
GOEXPERIMENT=jsonv2 go install github.com/MysticalDevil/codex-sm@v0.1.1
```

With `mise`:

```bash
GOEXPERIMENT=jsonv2 mise install go:github.com/MysticalDevil/codex-sm@v0.1.1
```

Note:

- The installed binary name is `codex-sm` (module root package name).
- If you prefer `csm`, add an alias, for example:

```bash
alias csm='codex-sm'
```

## Quick Start

```bash
# List recent sessions (default limit: 10)
codex-sm list

# Detailed list view
codex-sm list --detailed

# Custom columns, no header
codex-sm list --format csv --no-header --column session_id,health

# Show all with pager
codex-sm list --limit 0 --pager

# Group by day
codex-sm group --by day

# Group by health with sorting and limit
codex-sm group --by health --sort count --order desc --limit 5

# Dry-run delete (default behavior)
codex-sm delete --id-prefix 019ca9

# Real soft delete
codex-sm delete --id-prefix 019ca9 --dry-run=false --confirm

# Real hard delete
codex-sm delete --id 019ca9c1-3df3-7551-b04b-b2a91c486755 --dry-run=false --confirm --hard

# Dry-run restore from trash
codex-sm restore --id-prefix 019ca9

# Real restore
codex-sm restore --id-prefix 019ca9 --dry-run=false --confirm
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
codex-sm help
codex-sm help list
codex-sm help group
codex-sm help delete
codex-sm help restore
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
just check
```

Tooling defaults:

- Formatter: `gofumpt`
- Lint: `go vet`

## License

Licensed under the BSD 3-Clause License. See [LICENSE](./LICENSE) for details.
