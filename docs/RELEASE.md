# Release Checklist

This checklist is for preparing the next `codexsm` release.

## Preflight

1. Confirm branch is up to date and clean:
   - `git status --short`
   - `git fetch origin main`
2. Confirm release version and date:
   - choose next semver tag (example: `v0.2.2`)
   - record release date in `CHANGELOG.md`
3. Confirm Go environment:
   - `go version` (must be `1.26+`)
   - `echo $GOEXPERIMENT` (must include `jsonv2`)

## Quality Gates

Run all required gates locally:

```bash
just check
just cover-gate
just bench-gate
```

Expected thresholds:

- unit coverage: `>= 50%`
- integration coverage: `>= 65%`
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

## Documentation Updates

Before tagging, update:

1. `CHANGELOG.md`:
   - move completed items from `[Unreleased]` into the new version section.
2. `README.md` and `docs/COMMANDS.md`:
   - update version examples (install and `just check-release <version>`).
3. `docs/INDEX.md`:
   - verify links include this checklist and remain valid.

## Tag And Release

1. Create release commit(s) with docs/changelog updates.
2. Tag and push:
   - `git tag vX.Y.Z`
   - `git push origin main --follow-tags`
3. Publish GitHub release from tag.
4. Verify CI workflow is green for:
   - `push` tag run
   - `release published` run
