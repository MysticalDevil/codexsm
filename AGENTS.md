# Repository Agent Notes

## Local Repo Layout

- The local worktree root is `~/Worktrees/codexsm`.
- The active main worktree is `~/Worktrees/codexsm/main`.
- The Zig experimental worktree is `~/Worktrees/codexsm/zig` and shares `~/Worktrees/codexsm/main/.git`.
- The C experimental worktree is `~/Worktrees/codexsm/c` as a separate local clone.
- These local repos should point directly at the GitHub remote for `MysticalDevil/codexsm`.
- When describing local paths in docs or notes, prefer `~`-based paths instead of `/home/...`.

## Querying Rules

- For structural code queries (e.g. pass-through wrappers, thin adapters, duplicated call patterns), prefer `ast-grep` (`sg`) first.
- Use `rg` as a supplement for plain text lookups, file discovery, and quick keyword filtering.
