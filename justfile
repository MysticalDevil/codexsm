set shell := ["bash", "-cu"]

go := "go"
home_dir := env_var_or_default("HOME", ".")
xdg_cache_home := env_var_or_default("XDG_CACHE_HOME", home_dir + "/.cache")
xdg_config_home := env_var_or_default("XDG_CONFIG_HOME", home_dir + "/.config")
xdg_state_home := env_var_or_default("XDG_STATE_HOME", home_dir + "/.local/state")
xdg_runtime_home := env_var_or_default("XDG_RUNTIME_DIR", xdg_cache_home + "/codexsm/runtime")
gofumpt := if env_var_or_default("GOFUMPT", "") != "" {
  env_var("GOFUMPT")
} else {
  "gofumpt"
}
goexperiment := "GOEXPERIMENT=jsonv2"
go_with_experiment := goexperiment + " " + go
go_cache_dir := env_var_or_default("GO_CACHE_DIR", xdg_cache_home + "/codexsm/go-cache")
go_with_experiment_cache := "env " + goexperiment + " GOCACHE=" + go_cache_dir + " " + go
version := env_var_or_default("VERSION", "dev")
bin := env_var_or_default("BIN", "codexsm")
integration_pkg := env_var_or_default("INTEGRATION_PKG", "./cli")
unit_cov_min := env_var_or_default("UNIT_COV_MIN", "60")
integration_cov_min := env_var_or_default("INTEGRATION_COV_MIN", "72")
bench_sort_3k_ns_max := env_var_or_default("BENCH_SORT_3K_NS_MAX", "15000000")
bench_sort_10k_ns_max := env_var_or_default("BENCH_SORT_10K_NS_MAX", "50000000")
gen_seed := env_var_or_default("GEN_SEED", "20260308")
gen_count := env_var_or_default("GEN_COUNT", "3000")
gen_min_turns := env_var_or_default("GEN_MIN_TURNS", "12")
gen_max_turns := env_var_or_default("GEN_MAX_TURNS", "48")
gen_risk_missing_meta_count := env_var_or_default("GEN_RISK_MISSING_META_COUNT", "3")
gen_risk_corrupted_count := env_var_or_default("GEN_RISK_CORRUPTED_COUNT", "3")
gen_large_file_count := env_var_or_default("GEN_LARGE_FILE_COUNT", "0")
gen_oversize_meta_count := env_var_or_default("GEN_OVERSIZE_META_COUNT", "0")
gen_oversize_user_count := env_var_or_default("GEN_OVERSIZE_USER_COUNT", "0")
gen_oversize_assistant_count := env_var_or_default("GEN_OVERSIZE_ASSISTANT_COUNT", "0")
gen_no_newline_count := env_var_or_default("GEN_NO_NEWLINE_COUNT", "0")
gen_mixed_corrupt_huge_count := env_var_or_default("GEN_MIXED_CORRUPT_HUGE_COUNT", "0")
gen_unicode_wide_count := env_var_or_default("GEN_UNICODE_WIDE_COUNT", "0")
gen_long_message_bytes := env_var_or_default("GEN_LONG_MESSAGE_BYTES", "131072")
gen_meta_line_bytes := env_var_or_default("GEN_META_LINE_BYTES", "98304")
gen_large_file_target_bytes := env_var_or_default("GEN_LARGE_FILE_TARGET_BYTES", "1048576")
gen_payload_shape := env_var_or_default("GEN_PAYLOAD_SHAPE", "mixed")
gen_time_range_start := env_var_or_default("GEN_TIME_RANGE_START", "2026-03-01T00:00:00Z")
gen_time_range_end := env_var_or_default("GEN_TIME_RANGE_END", "2026-03-31T23:59:59Z")
gen_output_root := env_var_or_default("GEN_OUTPUT_ROOT", "testdata/_generated/sessions")
ci_cache_dir := xdg_cache_home + "/codexsm/ci"
ci_config_dir := xdg_config_home + "/codexsm"
ci_state_dir := xdg_state_home + "/codexsm/ci"
ci_runtime_dir := xdg_runtime_home + "/codexsm"

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
  {{go_with_experiment_cache}} test -run='^$' -bench='Benchmark(SortTUISessions_3k|SortTUISessions_10k|BuildPreviewLines|PreviewIndex)' -benchmem ./tui

# Run session scan/filter micro-benchmarks
bench-session:
  {{go_with_experiment_cache}} test -run='^$' -bench='Benchmark(ScanSessions|FilterSessions)' -benchmem ./session

# Run CLI rendering and doctor-risk micro-benchmarks
bench-cli:
  {{go_with_experiment_cache}} test -run='^$' -bench='Benchmark(RenderTable|RenderJSON|DoctorRiskJSON)' -benchmem ./cli

# Run all lightweight benchmark suites
bench-all: bench-session bench-cli bench-tui

# Run CI smoke checks against built binary and risk fixture dataset
ci-smoke:
  set -e; \
  mkdir -p "{{go_cache_dir}}" "{{ci_cache_dir}}" "{{ci_config_dir}}" "{{ci_state_dir}}" "{{ci_runtime_dir}}"; \
  {{go_with_experiment_cache}} build -ldflags="-X main.version={{version}}" -o {{bin}} .; \
  ./{{bin}} restore --help | grep -q -- "--batch-id"; \
  ./{{bin}} delete --help | grep -q -- "--preview"; \
  ./{{bin}} config --help | grep -q -- "show"; \
  ./{{bin}} config --help | grep -q -- "validate"; \
  CSM_CONFIG="{{ci_config_dir}}/config.json" ./{{bin}} config init --dry-run | grep -q -- "\"sessions_root\""; \
  if ./{{bin}} restore --batch-id b-test --id deadbeef >"{{ci_state_dir}}/restore-conflict.log" 2>&1; then \
    echo "expected restore conflict to fail"; \
    exit 1; \
  fi; \
  grep -q "cannot be combined" "{{ci_state_dir}}/restore-conflict.log"; \
  ./{{bin}} doctor >"{{ci_state_dir}}/doctor.txt"; \
  rc=0; \
  ./{{bin}} doctor risk --sessions-root ./testdata/fixtures/risky-static/sessions --format json --sample-limit 3 >"{{ci_state_dir}}/risk.json" || rc=$?; \
  if [ "$rc" -ne 1 ]; then \
    echo "expected doctor risk fixture check to exit 1, got $rc"; \
    exit 1; \
  fi; \
  python3 scripts/check_doctor_risk_json.py "{{ci_state_dir}}/risk.json" --sample-limit 3

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
    --large-file-count {{gen_large_file_count}} \
    --oversize-meta-count {{gen_oversize_meta_count}} \
    --oversize-user-count {{gen_oversize_user_count}} \
    --oversize-assistant-count {{gen_oversize_assistant_count}} \
    --no-newline-count {{gen_no_newline_count}} \
    --mixed-corrupt-huge-count {{gen_mixed_corrupt_huge_count}} \
    --unicode-wide-count {{gen_unicode_wide_count}} \
    --long-message-bytes {{gen_long_message_bytes}} \
    --meta-line-bytes {{gen_meta_line_bytes}} \
    --large-file-target-bytes {{gen_large_file_target_bytes}} \
    --payload-shape {{gen_payload_shape}} \
    --time-range-start {{gen_time_range_start}} \
    --time-range-end {{gen_time_range_end}} \
    --output-root {{gen_output_root}}

# Generate a compact extreme dataset for local regression checks
gen-sessions-extreme:
  GEN_COUNT=0 \
  GEN_RISK_MISSING_META_COUNT=1 \
  GEN_RISK_CORRUPTED_COUNT=1 \
  GEN_LARGE_FILE_COUNT=1 \
  GEN_OVERSIZE_META_COUNT=1 \
  GEN_OVERSIZE_USER_COUNT=1 \
  GEN_OVERSIZE_ASSISTANT_COUNT=1 \
  GEN_NO_NEWLINE_COUNT=1 \
  GEN_MIXED_CORRUPT_HUGE_COUNT=1 \
  GEN_UNICODE_WIDE_COUNT=1 \
  GEN_LONG_MESSAGE_BYTES=65536 \
  GEN_META_LINE_BYTES=32768 \
  GEN_LARGE_FILE_TARGET_BYTES=262144 \
  GEN_PAYLOAD_SHAPE=mixed \
  just gen-sessions

# Generate a larger stress dataset for manual benchmarking and memory checks
gen-sessions-large:
  GEN_COUNT=200 \
  GEN_RISK_MISSING_META_COUNT=3 \
  GEN_RISK_CORRUPTED_COUNT=3 \
  GEN_LARGE_FILE_COUNT=6 \
  GEN_OVERSIZE_META_COUNT=4 \
  GEN_OVERSIZE_USER_COUNT=6 \
  GEN_OVERSIZE_ASSISTANT_COUNT=6 \
  GEN_NO_NEWLINE_COUNT=4 \
  GEN_MIXED_CORRUPT_HUGE_COUNT=4 \
  GEN_UNICODE_WIDE_COUNT=4 \
  GEN_LONG_MESSAGE_BYTES=262144 \
  GEN_META_LINE_BYTES=131072 \
  GEN_LARGE_FILE_TARGET_BYTES=2097152 \
  GEN_PAYLOAD_SHAPE=log-heavy \
  just gen-sessions

# Generate a large dataset and run list/doctor risk smoke checks locally
stress-cli:
  set -e; \
  mkdir -p "{{go_cache_dir}}" "{{ci_runtime_dir}}" "{{ci_state_dir}}"; \
  tmpdir="$(mktemp -d "{{ci_runtime_dir}}/stress-cli.XXXXXX")"; \
  trap 'rm -rf "$tmpdir"' EXIT; \
  {{go_with_experiment_cache}} build -ldflags="-X main.version={{version}}" -o {{bin}} .; \
  GEN_OUTPUT_ROOT="$tmpdir/sessions" just gen-sessions-large; \
  ./codexsm list --sessions-root "$tmpdir/sessions" --limit 50 --format json >"{{ci_state_dir}}/stress-list.json"; \
  rc=0; \
  ./codexsm doctor risk --sessions-root "$tmpdir/sessions" --format json --sample-limit 5 >"{{ci_state_dir}}/stress-risk.json" || rc=$?; \
  if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ]; then \
    echo "doctor risk stress smoke failed with rc=$rc"; \
    exit 1; \
  fi; \
  test -s "{{ci_state_dir}}/stress-list.json"; \
  test -s "{{ci_state_dir}}/stress-risk.json"

# Remove generated coverage files
clean:
  rm -f coverage_unit.out coverage_integration.out

# Run the smallest full quality gate
check: fmt lint test-all build
