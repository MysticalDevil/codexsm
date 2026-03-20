# Release Checklist

This checklist is for preparing the next `codexsm` release.

## Preflight

1. Confirm branch is up to date and clean:
   - `git status --short`
   - `git fetch origin main`
2. Confirm release version and date:
   - choose next semver tag (example: `v0.3.6`)
   - record release date in `CHANGELOG.md`
3. Confirm Go environment:
   - `go version` (must be `1.26+`)
   - `echo $GOEXPERIMENT` (must include `jsonv2`)
4. Sync local toolchain via `mise`:
   - `mise install`

## Quality Gates

Run all required gates locally:

```bash
just check
just cover-gate
just ci-smoke
just bench-gate
just bench-session
just bench-cli
```

Expected thresholds:

- unit coverage: `>= 60%`
- integration coverage: `>= 72%`
- benchmark sort 3k: `<= 15,000,000 ns/op`
- benchmark sort 10k: `<= 50,000,000 ns/op`

## Risk Dataset Validation

Validate risk scanning output against fixture data:

```bash
codexsm doctor risk --sessions-root ./testdata/fixtures/risky-static/sessions --format json --sample-limit 5
```

Expected behavior:

- command exits with code `1` (risk detected)
- output is valid JSON
- JSON includes non-zero `risk_total`

Optional extreme-dataset smoke checks:

```bash
just gen-sessions-extreme
codexsm doctor risk --sessions-root ./testdata/fixtures/extreme-static/sessions --format json --sample-limit 4
```

Expected behavior:

- command completes without crashing on oversize/meta/no-newline samples
- output remains valid JSON
- risk report includes missing-meta/corrupted samples from the extreme corpus

For larger manual stress validation, use:

```bash
just gen-sessions-large
```

This target is intended for local benchmarking and memory checks, not the default release gate.

Additional lightweight benchmark sweeps:

```bash
just bench-session
just bench-cli
just bench-tui
```

These commands should run successfully in CI/local validation, but they do not yet enforce new numeric thresholds beyond the existing TUI sort gate.

CI orchestration note:

- pull requests run the lighter `lint` + `test` jobs
- tag pushes additionally run `smoke` + `bench`
- `just ci-smoke` is the shared local/CI entrypoint for rollback, doctor, and risk-fixture smoke checks

## Documentation Updates

Before tagging, update:

1. `CHANGELOG.md`:
   - move completed items from `[Unreleased]` into the new version section.
2. `README.md` and `docs/COMMANDS.md`:
   - update version examples (install and `just check-release <version>`).
   - update any TUI interaction notes if delete/restore safety prompts changed for the release.

## Tag And Release

1. Create release commit(s) with docs/changelog updates.
2. Tag and push:
   - `git tag vX.Y.Z`
   - `git push origin main --follow-tags`
3. Publish GitHub release from tag.
4. Verify CI workflow is green for:
   - `push` tag run
