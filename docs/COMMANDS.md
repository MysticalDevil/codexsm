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
| `agents explain` | show AGENTS.md source chain and effective rules | read-only |
| `session migrate` | copy sessions to a new cwd/path | dry-run |

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
codexsm help agents
codexsm help session migrate
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
- `d`: delete selected session, or delete the selected group from a group header
- `r`: restore selected session (trash source)
- `m`: migrate missing-host sessions to trash
- `y/n`: confirm/cancel pending action
- `q`: quit

> [!TIP]
> Use `t` / `p` (or `1` / `2`) to explicitly focus tree/preview panes.
>
> When a destructive action is pending, the bottom `KEYS` row is replaced by a high-visibility one-line confirm prompt (`Y` continue / `N` cancel).
> Real TUI delete on a group header requires three confirms before execution.
> After a delete succeeds, selection advances to the next available session instead of jumping back to the top.
> Width is adaptive by tier: `full` (`>=118`), `medium` (`96-117`), `compact` (`80-95`), `ultra` (`65-79`, single active pane with `Tab`/`1`/`2`).

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
codexsm doctor --compact-home=false
codexsm doctor risk
codexsm doctor risk --format json --integrity-check

codexsm config show
codexsm config show --resolved
codexsm config validate
codexsm config init

codexsm agents explain
codexsm agents explain --show-shadowed
codexsm agents explain --format json
```

## Session Migration

```bash
# Dry-run one source -> target migration
codexsm session migrate --from /old/path --to /new/path

# Real one-shot migration
codexsm session migrate --from /old/path --to /new/path --dry-run=false --confirm

# Dry-run batch migration from TOML mappings
codexsm session migrate --file ./migrate.toml
```

Batch file shape:

```toml
[[mapping]]
from = "/path/to/source/project"
to = "/path/to/new/main"
branch = "main"

[[mapping]]
from = "/path/to/source/project"
to = "/path/to/new/feature"
branch = "feature-branch"
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
just check-release 0.3.5
```
