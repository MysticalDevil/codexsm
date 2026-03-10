# Repository Agent Notes

## Local Repo Layout

- The local worktree root is `~/Worktrees/codexsm`.
- The active main worktree is `~/Worktrees/codexsm/main`.
- The Zig experimental worktree is `~/Worktrees/codexsm/zig` and shares `~/Worktrees/codexsm/main/.git`.
- The C experimental worktree is `~/Worktrees/codexsm/c` as a separate local clone.
- These local repos should point directly at the GitHub remote for `MysticalDevil/codexsm`.
- When describing local paths in docs or notes, prefer `~`-based paths instead of `/home/...`.
