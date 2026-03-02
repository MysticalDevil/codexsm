# codex-sm

`codex-sm` is a safety-first local Codex session manager written in Go.

It provides:

- Session listing (`list`)
- Session grouping (`group`)
- Safe deletion (`delete`, dry-run by default)

## Features

- Safe by default: `delete` runs with `--dry-run=true`
- Real deletion requires explicit intent: `--dry-run=false --confirm`
- Default real deletion mode is soft delete (move to trash)
- Optional hard delete with `--hard`
- Interactive confirmation for real delete (enabled by default)
- Readable CLI output:
  - compact list view by default
  - detailed mode
  - pager mode
  - colored help/output

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

# Show all with pager
csm list --limit 0 --pager

# Group by day
csm group --by day

# Group by health
csm group --by health

# Dry-run delete (default behavior)
csm delete --id-prefix 019ca9

# Real soft delete
csm delete --id-prefix 019ca9 --dry-run=false --confirm

# Real hard delete
csm delete --id 019ca9c1-3df3-7551-b04b-b2a91c486755 --dry-run=false --confirm --hard
```

## Delete Safety Model

`delete` targets are selected by flags (not positional args):

- `--id <session_id>`
- `--id-prefix <prefix>`
- `--older-than <duration>` (for example `30d`, `12h`)
- `--health <ok|corrupted|missing-meta>`

Rules:

- At least one selector is required
- Dry-run is default
- Real delete requires `--confirm`
- Batch real delete requires approval (`--yes` or interactive confirm)

## Command Help

```bash
csm help
csm help list
csm help group
csm help delete
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
