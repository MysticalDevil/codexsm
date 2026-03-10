# Benchmark Baselines

This document records lightweight benchmark baselines for `codexsm`.

## Baseline Snapshot

- Date: `2026-03-10`
- Commit: `01c80e2`
- Go: `go1.26.1 linux/amd64`
- CPU: `AMD Ryzen 7 4800H with Radeon Graphics`
- Runs: `go test -run '^$' -bench ... -benchmem -count=3`
- Interpretation: use these numbers as a first local baseline, not as hard gates.

## Commands

```bash
go test -run '^$' -bench 'Benchmark(ScanSessions|FilterSessions)' ./session -benchmem -count=3
go test -run '^$' -bench 'Benchmark(RenderTable|RenderJSON|DoctorRiskJSON)' ./cli -benchmem -count=3
go test -run '^$' -bench 'Benchmark(SortTUISessions_3k|SortTUISessions_10k|BuildPreviewLines|PreviewIndex)' ./tui -benchmem -count=3
```

## Session

| Benchmark | Median ns/op | Median B/op | Median allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkScanSessions` | `872,434` | `197,588` | `682` |
| `BenchmarkFilterSessions/all` | `3,272` | `5,080` | `4` |
| `BenchmarkFilterSessions/host_head_health` | `4,844` | `5,080` | `4` |
| `BenchmarkFilterSessions/older_than` | `2,205` | `4,888` | `2` |
| `BenchmarkScanSessionsLimited_3k` | `83,763,440` | `17,501,494` | `60,274` |
| `BenchmarkScanSessions_AllVsLimited_3k/all` | `80,239,823` | `19,181,414` | `60,343` |
| `BenchmarkScanSessions_AllVsLimited_3k/limited_100` | `81,762,990` | `17,492,142` | `60,278` |
| `BenchmarkScanSessions_ExtremeMix` | `34,510,266` | `9,151,241` | `24,286` |

Observation:

- `ScanSessionsLimited_3k` reduces retained-result memory versus the full-scan path, but current scan cost still dominates because all files are parsed.

## CLI

| Benchmark | Median ns/op | Median B/op | Median allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkDoctorRiskJSON` | `17,970,034` | `4,923,236` | `17,073` |
| `BenchmarkRenderTable` | `3,313,728` | `2,161,020` | `15,673` |
| `BenchmarkRenderTable_LargeColumns` | `5,773,960` | `4,184,011` | `18,082` |
| `BenchmarkRenderJSON` | `2,066,419` | `1,202,056` | `10` |

Observation:

- `doctor risk --format json` is the heaviest CLI benchmark in this set and is the best early candidate for later optimization work.

## TUI

| Benchmark | Median ns/op | Median B/op | Median allocs/op |
| --- | ---: | ---: | ---: |
| `BenchmarkSortTUISessions_3k` | `2,274,792` | `409,602` | `1` |
| `BenchmarkSortTUISessions_10k` | `8,376,444` | `1,368,064` | `1` |
| `BenchmarkBuildPreviewLines_LargeSession` | `33,620,444` | `909,091` | `13,220` |
| `BenchmarkBuildPreviewLines_OversizeUser` | `134,269,530` | `3,229,860` | `18,064` |
| `BenchmarkBuildPreviewLines_OversizeAssistant` | `209,001,010` | `4,945,912` | `30,107` |
| `BenchmarkBuildPreviewLines_UnicodeWide` | `143,551,470` | `7,421,062` | `15,129` |
| `BenchmarkPreviewIndexLoad_1k` | `1,954,543` | `731,155` | `6,048` |
| `BenchmarkPreviewIndexUpsert_1k` | `6,132,681` | `1,231,516` | `9,127` |
| `BenchmarkPreviewIndexUpsert_Trimmed` | `46,960,555` | `28,850,249` | `6,474` |

Observation:

- Oversize preview inputs and byte-budget trimming are the most expensive TUI paths in the current lightweight benchmark set.

## Next Use

- Re-run this file's command set after meaningful scanner, preview, or CLI rendering changes.
- Only add new CI thresholds after at least one more baseline pass on a second machine or runner class.
