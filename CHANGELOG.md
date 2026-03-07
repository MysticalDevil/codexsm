# Changelog

All notable changes to this project are documented in this file.

The format is based on Keep a Changelog and this project follows Semantic Versioning.

## [Unreleased]

### Added

- Added `config` command group with `show`, `init`, and `validate`.
- Added benchmark baselines for `session.ScanSessions`, `session.FilterSessions`, and `cli` list table rendering.
- Added unit tests for `internal/ops`, `internal/fileutil`, and `internal/restoreexec`.
- Added CI smoke checks for `config` subcommands.
- Added `doctor` host-path diagnostics with cleanup strategy guidance for sessions whose `host_dir` no longer exists.
- Added TUI missing-host maintenance workflow:
  - tree marker for sessions with missing host paths
  - details panel host status marker
  - `m` key action to migrate matched host sessions to trash (dry-run/confirm safety preserved)
- Added TUI regression tests for host grouping stability and missing-host strategy paths.

### Changed

- Refactored TUI layout into `internal/tui/layout` and split TUI view rendering.
- Refactored list implementation into focused modules (`list`, `list_columns`, `pager`, `ansi`).
- Refactored restore/delete shared operation helpers into `internal` packages.
- Split session scanner internals into parsing and head-scoring modules.
- Fixed TUI host grouping to aggregate same-host sessions under a single group header even with time-sorted input.
- Moved TUI implementation package from `internal/tui/browser` to top-level `tui` while keeping `codexsm tui` command behavior unchanged.
- Updated README release/install examples toward `v0.1.9`.

## [v0.1.7] - 2026-03-04

### Added

- Added composable selectors for host/path/head filtering.
- Added short aliases for high-frequency CLI flags.
- Added `--preview` modes (`sample|full|none`) for delete/restore.
- Added `batch_id` audit logging and `restore --batch-id` rollback.
- Added tests for preview modes, selector combinations, and rollback flows.

### Changed

- Improved docs and release/CI readiness for rollback workflows.

## [v0.1.6] - 2026-03-04

### Changed

- Renamed project/module identity from `codex-sm` to `codexsm`.
- Updated docs and release examples to new module/binary naming.
- Improved CI docs consistency checks.
- Enhanced justfile maintainability and CLI list usability docs/tests.

## [v0.1.5] - 2026-03-05

### Added

- Added runtime config loading (`$CSM_CONFIG` / default config path).
- Added `doctor` command for local configuration and path sanity checks.
- Added major TUI modularization with themes and action system support.

### Changed

- Improved build/version behavior to resolve release tag metadata correctly.
- Added architecture/release prep documentation.
