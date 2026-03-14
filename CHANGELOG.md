# Changelog

All notable changes to this project are documented in this file.

The format is based on Keep a Changelog and this project follows Semantic Versioning.

## [Unreleased]

## [v0.3.3] - 2026-03-15

### Added

- Added adaptive TUI width tiers (`full`, `compact`, `ultra`) with mode-aware rendering and key hints.
- Added ultra single-pane navigation so narrow terminals can switch between tree and preview with shared state.
- Added focused TUI regression tests covering ultra tier behavior and preview request/render key consistency.

### Changed

- Refined compact tree presentation toward lower visual density (reduced connector noise, color-first emphasis).
- Updated TUI footer status summary format to `index/total | WARN: X RISK: X` and aligned semantic token usage.
- Updated ultra keybar copy to keep consistent short labels across width variants (`[U-TREE]`, `[U-PREVIEW]`).

### Fixed

- Fixed ultra mode preview loading flow so pane switching preserves selected-session preview continuity instead of falling into stale `preview not ready` states.

## [v0.3.2] - 2026-03-15

### Added

- Added `agents explain` command for AGENTS.md rule-source visibility (table/json, effective vs shadowed views).
- Added broader test coverage for TUI preview build/index/service flow and CLI command/util/pager paths.
- Added dedicated usecase tests after delete/restore orchestration split (`usecase/delete*`, `usecase/restore*`, `usecase/batch_policy*`).

### Changed

- Refactored TUI architecture to a clearer layered layout (`app/state/actions/layout/render/theme/tree/preview`) and moved preview orchestration/index handling into `tui/preview/*`.
- Consolidated TUI runtime flow into `tui/app.go`, removed legacy local preview/theme passthrough helpers, and flattened small subpackage file layouts.
- Refactored CLI command layout to subpackages (`cli/config`, `cli/delete`, `cli/doctor`, `cli/list`, `cli/restore`, `cli/util`), and relocated related tests into those package boundaries.
- Renamed session migration command implementation files from `session_migrate*` to `migrate*` while preserving `codexsm session migrate` behavior and flags.
- Split usecase delete/restore orchestration into focused files (`usecase/delete.go`, `usecase/restore.go`, `usecase/action_exec.go`, `usecase/batch_policy.go`) and removed the prior mixed file.
- Renamed delete/restore action input naming from `Candidates` to `Sessions` across usecase, CLI, and TUI call paths for clearer semantics.
- Reused shared bounded-line reader logic in session scanner and migration paths to reduce duplicate parsing limits code.
- Applied repo-wide lint-driven cleanups and tightened agent/lint conventions against thin pass-through wrappers.
- Refreshed architecture and release docs (including topology/diagram updates and `v0.3.2` examples) for release prep consistency.

### Fixed

- Fixed TUI lint compliance for exhaustive switch handling and related regressions caught during refactor/lint passes.

## [v0.3.1] - 2026-03-14

### Changed

- Refactored session internals into explicit subpackages: `session/scanner/*` (scan/head/parse/io) and `session/migrate/*` (exec/batch/index/rollout/sql), and updated CLI/core call sites accordingly.
- Moved TUI preview core logic into `tui/preview/*` and kept thin adapters in `tui/preview.go` and `tui/preview_index.go` for compatibility with existing TUI flow.
- Consolidated file move/copy helpers under `util/file.go` and removed now-empty legacy internal bridge package paths.
- Moved restore execution into `session` and removed the extra restore execution bridge layer so usecase/action flow is more direct.
- Updated architecture, command, release, and README documentation to match the current module layout and the `v0.3.1` release examples.
- Reworked the architecture ASCII dependency diagram to reflect the current layered topology (`cli -> usecase -> session`, plus `session/scanner`, `session/migrate`, and `tui/preview` submodules).
- Tightened local/CI docs checks (`just docs-check`, CI docs consistency step) to reject stale references to removed internal paths.

## [v0.3.0] - 2026-03-10

### Added

- Added `session migrate` for copying Codex sessions from one `cwd` to another while keeping Resume compatibility by cloning both rollout files and matching `threads` rows from Codex's local SQLite state.
- Added batch migration support via `codexsm session migrate --file <migrate.toml>`, with ordered `[[mapping]]` entries, dry-run aggregation, and stop-on-first-failure real execution.

### Changed

- Refactored migration SQLite statements into dedicated constants to keep migration index logic smaller and easier to review.
- Expanded mainline architecture and benchmark documentation to reflect current module boundaries and refreshed baseline measurements.
- Removed the redundant docs index page and kept the README as the primary documentation entrypoint.

## [v0.2.7] - 2026-03-10

### Changed

- Improved narrow-width TUI layout behavior by shrinking the tree pane, compacting the bottom keybar into adaptive variants, and compacting the top info pane before falling back to the min-size warning.
- Reduced the effective TUI minimum width requirement after decoupling keybar and info-panel width needs from the main split-pane layout.
- Updated the main README install examples and experimental-branch notes to point at `v0.2.6` and the current `zsession` backend scope.

### Fixed

- Fixed TUI keybar wrapping in borderline terminal widths by switching the `KEYS` row to width-aware compact variants instead of relying on a single long line.
- Fixed narrow-width TUI layout waste by dropping the inter-pane gap at smaller widths and rebalancing the left tree pane width.
- Fixed the preview oversize warning text to use a clearer user-facing message instead of exposing the raw `bufio.Scanner` `token too long` error.
- Fixed CI duplication after releases by stopping the main CI workflow from re-triggering on GitHub release publication when tag CI has already run.

## [v0.2.6] - 2026-03-10

### Fixed

- Fixed session scanning against real Codex CLI session files whose `session_meta` first line exceeds the default `bufio.Reader` chunk size but remains below the configured 1 MiB metadata limit. This restores correct `ID`, `HOST`, `HEALTH`, and TUI grouping for large sessions instead of incorrectly marking them as `CORRUPTED`.
- Fixed CI smoke and benchmark jobs to install `just` before invoking shared `justfile` entrypoints on GitHub-hosted Ubuntu runners.

## [v0.2.5] - 2026-03-09

### Added

- Added an `extreme-static` fixture corpus covering long single-message sessions, oversize meta payloads, mixed corruption, Unicode-heavy previews, and no-final-newline files.
- Added `gen-sessions-extreme` and `gen-sessions-large` workflows plus generator knobs for large files, oversize lines/messages, Unicode-heavy content, and no-final-newline outputs.
- Added local `just` workflows for `bench-session`, `bench-cli`, `bench-all`, and `stress-cli` to exercise lightweight benchmark sweeps and generated large-dataset smoke checks.
- Added `docs/BENCHMARKS.md` to record the first lightweight benchmark baseline snapshot and rerun commands.

### Changed

- Expanded fixture/schema and `session`/`tui`/`cli doctor risk` tests to consume the new extreme dataset directly.
- Expanded benchmark coverage across session scanning, TUI preview/index paths, and CLI table/JSON/risk rendering, while keeping new benchmark runs threshold-free for now.
- Reworked CI orchestration into separate lint, test, smoke, and bench jobs so pull requests stay lighter while tag/release runs keep the full quality gate.
- Improved CI/local smoke tooling by moving doctor-risk JSON validation into a dedicated Python script and shifting cache/state/runtime paths toward XDG-standard locations.

### Fixed

- Fixed session scanning against malformed inputs by bounding scanner line reads for oversized session metadata and conversation head lines.
- Fixed TUI preview index persistence and loading to enforce a byte budget, reducing large transient memory spikes from oversized cached previews.
- Fixed `tui --limit` behavior so the limit is applied during session scanning rather than only after fully loading all sessions.

## [v0.2.4] - 2026-03-09

### Changed

- Refined TUI rendering to follow the terminal default background while preserving themed borders, selection, and local highlight styles.
- Improved TUI layout behavior in narrower terminals by centering the main layout, reserving a terminal-edge safety column, and raising the minimum supported width to avoid partial border/keybar rendering.
- Simplified the TUI min-size fallback view so constrained terminals show a centered warning panel instead of attempting to render the normal keybar path.
- Raised coverage gates to `>= 60%` unit and `>= 72%` integration, and aligned CI coverage thresholds with the local release gate.

### Added

- Added TUI regression coverage for restore guard paths, pending-action cancellation paths, and min-size warning rendering constraints.
- Added CLI coverage for preview helper edge cases and restore selector/batch guard paths.

## [v0.2.3] - 2026-03-09

### Changed

- Improved TUI preview layout stability by correcting preview content height budgeting for large sessions, reducing visual drift/misalignment.
- Fixed TUI keybar width alignment so bottom `KEYS` container width matches the main area consistently.
- Improved TUI delete flow: after deleting a session, selection now advances to the next item (or nearest valid neighbor) instead of jumping back to the head.
- Improved TUI delete confirmation visibility by replacing the bottom `KEYS` row content with a prominent in-place `Y/N` confirm prompt while an action is pending.

## [v0.2.2] - 2026-03-08

### Added

- Added seeded session dataset generator with configurable seed, session count, turn range, and datetime range controls.
- Added richer multilingual prompt pools for synthetic conversations (Chinese, English, Spanish, Latin, Japanese, Korean, Arabic), including mixed-language and emoji patterns.
- Added larger default generated dataset volume for TUI stress testing.
- Added `just gen-sessions` workflow for reproducible dataset generation under `testdata/_generated/`.
- Added `doctor risk` command path with risk prioritization and structured JSON output (`--format json`).
- Added SHA256 sidecar integrity verification support for risk evaluation.
- Added static and generated risky fixture datasets under `testdata/fixtures/risky-static/` and generator risk injection options.
- Added asynchronous TUI preview loading path with disk-backed preview index support.
- Added TUI benchmark suite and guardrails (`bench-tui`, `bench-gate`) for sort latency.
- Added `docs/RELEASE.md` with preflight, quality-gate, risk-fixture validation, and tag/release checklist.

### Changed

- Changed TUI default grouping from month to host, while preserving `day` grouping support.
- Trimmed less meaningful TUI grouping modes to reduce navigation noise.
- Changed preview cache policy from entry-count cap to byte-budget LRU cap.
- Changed preview cache keying to include width, reducing stale-layout reuse after resize.
- Hardened preview index persistence with lock file coordination and corruption-tolerant recovery rewrite.
- Updated TUI large-session preview behavior to background loading with placeholder-first rendering.
- Expanded CI quality gates to include TUI benchmark thresholds and `doctor risk --format json` fixture validation.
- Updated CI trigger policy to run on pull requests to `main` and release events (`v*` tags and published releases).
- Updated README, docs index, and command guide with release-checklist navigation and release-prep commands.
- Updated release examples and checks toward `v0.2.2`.

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
