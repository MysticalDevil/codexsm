# Command Guide

## Command Matrix

| Command | Purpose | Safe Default |
| --- | --- | --- |
| `list` | list sessions | read-only |
| `group` | aggregate sessions by key | read-only |
| `tui` | interactive browse/manage | dry-run actions |
| `delete` | delete or soft-delete sessions | `--dry-run=true` |
| `restore` | restore from trash | `--dry-run=true` |
| `doctor` | environment/config checks | read-only |
| `config` | inspect/init/validate config | read-only except `init` |

## Common Commands

```bash
codexsm help
codexsm help list
codexsm help group
codexsm help tui
codexsm help delete
codexsm help restore
codexsm help doctor
codexsm help config
```

## Shell Completion

```bash
# Bash
codexsm completion bash > ~/.local/share/bash-completion/completions/codexsm

# Zsh
codexsm completion zsh > "${fpath[1]}/_codexsm"

# Fish
codexsm completion fish > ~/.config/fish/completions/codexsm.fish

# PowerShell
codexsm completion powershell > codexsm.ps1
```

## Session Browsing

```bash
# Recent sessions
codexsm list

# Detailed table
codexsm list --detailed

# CSV output
codexsm list --format csv --column session_id,health,host_dir

# Group summary
codexsm group --by day
codexsm group --by health --sort count --order desc --limit 5
```

## TUI

```bash
codexsm tui
codexsm tui --group-by host
codexsm tui --source trash --theme gruvbox --theme-color border_focus=#fabd2f
```

TUI keys (default):

- `j/k`, `Down/Up`: move selection
- `g/G`: first/last item
- `Tab` / `h` / `l`: switch focus panes
- `Ctrl+d` / `Ctrl+u`: preview page scroll
- `d`: delete selected session
- `r`: restore selected session (trash source)
- `m`: migrate missing-host sessions to trash
- `y/n`: confirm/cancel pending action
- `q`: quit

> [!TIP]
> Use `t` / `p` (or `1` / `2`) to explicitly focus tree/preview panes.

## Delete And Restore

Selectors:

- `--id <session_id>`
- `--id-prefix <prefix>`
- `--host-contains <text>`
- `--path-contains <text>`
- `--head-contains <text>`
- `--older-than <duration>`
- `--health <ok|corrupted|missing-meta>`
- `--batch-id <id>` (restore only; cannot combine with selector flags)

Safety rules:

- `delete` and `restore` default to dry-run.
- Real execution requires `--dry-run=false --confirm`.
- Multi-target real execution requires approval (`--yes` or interactive confirmation).

> [!WARNING]
> Run one dry-run preview before real delete/restore whenever selector scope is broad.

Examples:

```bash
# Dry-run delete
codexsm delete --id-prefix 019ca9

# Real soft delete
codexsm delete --id-prefix 019ca9 --dry-run=false --confirm

# Real hard delete
codexsm delete --id 019ca9c1-3df3-7551-b04b-b2a91c486755 --dry-run=false --confirm --hard

# Dry-run restore
codexsm restore --id-prefix 019ca9

# Real restore
codexsm restore --id-prefix 019ca9 --dry-run=false --confirm

# Roll back one soft-delete batch
codexsm restore --batch-id b-20260305T120102Z-9f1a2b3c4d5e --dry-run=false --confirm
```

## Diagnostics And Config

```bash
codexsm doctor
codexsm doctor --strict
codexsm doctor risk
codexsm doctor risk --format json --integrity-check

codexsm config show
codexsm config show --resolved
codexsm config validate
codexsm config init
```

## Development And Release

```bash
just fmt
just lint
just test
just test-integration
just test-all
just cover
just cover-gate
just bench-tui
just bench-gate
just check
just check-release 0.2.1
```
