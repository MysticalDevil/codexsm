set shell := ["bash", "-cu"]

go := "go"
gofumpt := if env_var_or_default("GOFUMPT", "") != "" {
  env_var("GOFUMPT")
} else {
  "gofumpt"
}
goexperiment := "GOEXPERIMENT=jsonv2"
go_with_experiment := goexperiment + " " + go
go_cache_dir := env_var_or_default("GO_CACHE_DIR", "/tmp/codexsm-go-cache")
go_with_experiment_cache := "env " + goexperiment + " GOCACHE=" + go_cache_dir + " " + go
version := env_var_or_default("VERSION", "dev")
bin := env_var_or_default("BIN", "codexsm")
integration_pkg := env_var_or_default("INTEGRATION_PKG", "./cli")
unit_cov_min := env_var_or_default("UNIT_COV_MIN", "50")
integration_cov_min := env_var_or_default("INTEGRATION_COV_MIN", "65")
bench_sort_3k_ns_max := env_var_or_default("BENCH_SORT_3K_NS_MAX", "15000000")
bench_sort_10k_ns_max := env_var_or_default("BENCH_SORT_10K_NS_MAX", "50000000")
gen_seed := env_var_or_default("GEN_SEED", "20260308")
gen_count := env_var_or_default("GEN_COUNT", "3000")
gen_min_turns := env_var_or_default("GEN_MIN_TURNS", "12")
gen_max_turns := env_var_or_default("GEN_MAX_TURNS", "48")
gen_risk_missing_meta_count := env_var_or_default("GEN_RISK_MISSING_META_COUNT", "3")
gen_risk_corrupted_count := env_var_or_default("GEN_RISK_CORRUPTED_COUNT", "3")
gen_time_range_start := env_var_or_default("GEN_TIME_RANGE_START", "2026-03-01T00:00:00Z")
gen_time_range_end := env_var_or_default("GEN_TIME_RANGE_END", "2026-03-31T23:59:59Z")
gen_output_root := env_var_or_default("GEN_OUTPUT_ROOT", "testdata/_generated/sessions")

# Show available targets
default:
  @just --list

# Install required dev tools
tools:
  {{go_with_experiment}} install mvdan.cc/gofumpt@latest

# Format Go sources
fmt:
  {{gofumpt}} -w .

# Run static checks
lint:
  {{go_with_experiment}} vet ./...

# Run unit test suite
test:
  {{go_with_experiment}} test ./...

# Run integration tests
test-integration:
  {{go_with_experiment}} test -tags=integration {{integration_pkg}}

# Run all tests
test-all: test test-integration

# Report unit test coverage
cover-unit:
  {{go_with_experiment}} test -count=1 ./... -coverprofile=coverage_unit.out
  {{go}} tool cover -func=coverage_unit.out

# Report integration test coverage
cover-integration:
  {{go_with_experiment}} test -count=1 -tags=integration {{integration_pkg}} -coverprofile=coverage_integration.out
  {{go}} tool cover -func=coverage_integration.out

# Report both coverage sets
cover: cover-unit cover-integration

# Enforce unit + integration coverage thresholds
cover-gate:
  {{go_with_experiment}} test -count=1 ./... -coverprofile=coverage_unit.out
  awk -v got="$({{go}} tool cover -func=coverage_unit.out | awk '/^total:/ {gsub("%","",$3); print $3}')" -v min="{{unit_cov_min}}" 'BEGIN { if (got+0 < min+0) { printf("unit coverage %.1f%% < %.1f%%\n", got, min); exit 1 } else { printf("unit coverage %.1f%% >= %.1f%%\n", got, min) } }'
  {{go_with_experiment}} test -count=1 -tags=integration {{integration_pkg}} -coverprofile=coverage_integration.out
  awk -v got="$({{go}} tool cover -func=coverage_integration.out | awk '/^total:/ {gsub("%","",$3); print $3}')" -v min="{{integration_cov_min}}" 'BEGIN { if (got+0 < min+0) { printf("integration coverage %.1f%% < %.1f%%\n", got, min); exit 1 } else { printf("integration coverage %.1f%% >= %.1f%%\n", got, min) } }'

check-tag version:
  [[ "$(git tag --points-at HEAD --list "v{{version}}" | wc -l | tr -d ' ')" == "1" ]] || { echo "expected current HEAD to have tag v{{version}}"; exit 1; }

check-readme-version version:
  rg -q "@v{{version}}" README.md || { echo "README.md does not contain install example with @v{{version}}"; exit 1; }

check-release version:
  just check
  just cover-gate
  just check-tag {{version}}
  just check-readme-version {{version}}

# Build local binary
build:
  {{go_with_experiment}} build -ldflags="-X main.version={{version}}" -o {{bin}} .

# Run TUI micro-benchmarks
bench-tui:
  {{go_with_experiment_cache}} test -run='^$' -bench='Benchmark(SortTUISessions_3k|SortTUISessions_10k|BuildPreviewLines_LargeSession)$' -benchmem ./tui

# Enforce TUI benchmark latency guardrails (ns/op)
bench-gate:
  if ! out="$({{go_with_experiment_cache}} test -run='^$' -bench='Benchmark(SortTUISessions_3k|SortTUISessions_10k)$' ./tui -count=1 2>&1)"; then \
    echo "$out"; \
    exit 1; \
  fi; \
  echo "$out"; \
  echo "$out" | awk -v max3="{{bench_sort_3k_ns_max}}" -v max10="{{bench_sort_10k_ns_max}}" '\
    /BenchmarkSortTUISessions_3k/ { for (i = 1; i <= NF; i++) if ($i == "ns/op") { got3 = $(i-1); break } } \
    /BenchmarkSortTUISessions_10k/ { for (i = 1; i <= NF; i++) if ($i == "ns/op") { got10 = $(i-1); break } } \
    END { \
      if (got3 == "" || got10 == "") { \
        print "benchmark output missing required rows"; \
        exit 1; \
      } \
      if (got3 + 0 > max3 + 0) { \
        printf("bench sort 3k %.0f ns/op > %.0f ns/op\n", got3 + 0, max3 + 0); \
        fail = 1; \
      } else { \
        printf("bench sort 3k %.0f ns/op <= %.0f ns/op\n", got3 + 0, max3 + 0); \
      } \
      if (got10 + 0 > max10 + 0) { \
        printf("bench sort 10k %.0f ns/op > %.0f ns/op\n", got10 + 0, max10 + 0); \
        fail = 1; \
      } else { \
        printf("bench sort 10k %.0f ns/op <= %.0f ns/op\n", got10 + 0, max10 + 0); \
      } \
      if (fail) exit 1; \
    }'

# Generate seeded random session dataset
gen-sessions:
  python3 scripts/gen_seeded_sessions.py \
    --seed {{gen_seed}} \
    --count {{gen_count}} \
    --min-turns {{gen_min_turns}} \
    --max-turns {{gen_max_turns}} \
    --risk-missing-meta-count {{gen_risk_missing_meta_count}} \
    --risk-corrupted-count {{gen_risk_corrupted_count}} \
    --time-range-start {{gen_time_range_start}} \
    --time-range-end {{gen_time_range_end}} \
    --output-root {{gen_output_root}}

# Remove generated coverage files
clean:
  rm -f coverage_unit.out coverage_integration.out

# Run the smallest full quality gate
check: fmt lint test-all build
