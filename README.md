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
  - detailed mode
  - pager mode
  - colored help/output
  - `json/table/csv/tsv` formats

## Build

```bash
make build
```

Or:

```bash
go build ./cmd/csm
```

## Quick Start

```bash
# List recent sessions (default limit: 10)
csm list

# Detailed list view
csm list --detailed

# Custom columns, no header
csm list --format csv --no-header --column session_id,health

# Show all with pager
csm list --limit 0 --pager

# Group by day
csm group --by day

# Group by health with sorting and limit
csm group --by health --sort count --order desc --limit 5

# Dry-run delete (default behavior)
csm delete --id-prefix 019ca9

# Real soft delete
csm delete --id-prefix 019ca9 --dry-run=false --confirm

# Real hard delete
csm delete --id 019ca9c1-3df3-7551-b04b-b2a91c486755 --dry-run=false --confirm --hard

# Dry-run restore from trash
csm restore --id-prefix 019ca9

# Real restore
csm restore --id-prefix 019ca9 --dry-run=false --confirm
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
csm help
csm help list
csm help group
csm help delete
csm help restore
```

## Development

```bash
make fmt
make lint
make test
make check
```

Tooling defaults:

- Formatter: `gofumpt`
- Lint: `go vet`

## License

Licensed under the BSD 3-Clause License. See [LICENSE](./LICENSE) for details.
