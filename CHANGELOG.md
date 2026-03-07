# Changelog

All notable changes to this project are documented in this file.

The format is based on Keep a Changelog and this project follows Semantic Versioning.

## [Unreleased]

## [v0.2.1] - 2026-03-08

### Added

- Added shell completion usage examples to command documentation.
- Added TUI tree health markers for faster anomaly scanning.

### Changed

- Improved `doctor session_host_paths` readability with compact sample paths and clearer action layout.
- Updated README and command guide release examples toward `v0.2.1`.
- Refined TUI tree status semantics: `OK` stays green `•`, host-missing / `MISSING-META` show orange `!`, `CORRUPTED` shows red `✖`, and non-healthy session names are colorized.

## [v0.2.0] - 2026-03-08

### Added

- Added shell completion command: `codexsm completion [bash|zsh|fish|powershell]`.
- Added completion command tests for valid and invalid shell cases.

### Changed

- Improved `doctor` output readability:
  - uppercased status values (`PASS/WARN/FAIL`)
  - colored table headers
  - aligned multi-line detail rendering
  - clearer `session_host_paths` action block
- Updated display-layer health statuses to uppercase in CLI list and TUI (`OK/CORRUPTED/MISSING-META`).
- Polished README and docs command/architecture presentation.
- Updated release examples and checks toward `v0.2.0`.

## [v0.1.9] - 2026-03-08

### Added

- Added `doctor` host-path diagnostics with cleanup strategy guidance for sessions whose `host_dir` no longer exists.
- Added TUI missing-host maintenance workflow with tree marker, detail host status marker, and `m` key migrate-to-trash action (dry-run/confirm safety preserved).
- Added TUI regression tests for host grouping stability and missing-host strategy paths.

### Changed

- Fixed TUI host grouping to aggregate same-host sessions under a single group header even with time-sorted input.
- Moved TUI implementation package from `internal/tui/browser` to top-level `tui` while keeping `codexsm tui` command behavior unchanged.
- Updated README release/install examples toward `v0.1.9`.

## [v0.1.8] - 2026-03-05

### Added

- Added `config` command group with `show`, `init`, and `validate`.
- Added benchmark baselines for `session.ScanSessions`, `session.FilterSessions`, and `cli` list table rendering.
- Added unit tests for `internal/ops`, `internal/fileutil`, and `internal/restoreexec`.
- Added CI smoke checks for `config` subcommands.

### Changed

- Refactored TUI layout into `internal/tui/layout` and split TUI view rendering.
- Refactored list implementation into focused modules (`list`, `list_columns`, `pager`, `ansi`).
- Refactored restore/delete shared operation helpers into `internal` packages.
- Split session scanner internals into parsing and head-scoring modules.
- Updated README release/install examples toward `v0.1.8`.

## [v0.1.7] - 2026-03-05

### Added

- Added composable selectors for host/path/head filtering.
- Added short aliases for high-frequency CLI flags.
- Added `--preview` modes (`sample|full|none`) for delete/restore.
- Added `batch_id` audit logging and `restore --batch-id` rollback.
- Added tests for preview modes, selector combinations, and rollback flows.

### Changed

- Improved docs and release/CI readiness for rollback workflows.

## [v0.1.6] - 2026-03-05

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
