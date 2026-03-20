set shell := ["bash", "-cu"]

go := "go"
goexperiment := "GOEXPERIMENT=jsonv2"
go_with_experiment := goexperiment + " " + go
version := env_var_or_default("VERSION", "dev")
bin := env_var_or_default("BIN", "codexsm")
integration_pkg := env_var_or_default("INTEGRATION_PKG", "./cli/...")
unit_cov_min := env_var_or_default("UNIT_COV_MIN", "60")
integration_cov_min := env_var_or_default("INTEGRATION_COV_MIN", "72")
bench_sort_3k_ns_max := env_var_or_default("BENCH_SORT_3K_NS_MAX", "15000000")
bench_sort_10k_ns_max := env_var_or_default("BENCH_SORT_10K_NS_MAX", "50000000")
gen_seed := env_var_or_default("GEN_SEED", "20260308")
gen_count := env_var_or_default("GEN_COUNT", "3000")
gen_output_root := env_var_or_default("GEN_OUTPUT_ROOT", "testdata/_generated/sessions")

# Show available targets
default:
  @just --list

# Format Go sources
fmt:
  mise exec -- gofumpt -w .

# Run static checks
lint:
  mise exec -- golangci-lint run

# Validate documentation links and release-version examples
docs-check:
  test -f README.md
  test -f CHANGELOG.md
  test -f docs/ARCHITECTURE.md
  test -f docs/COMMANDS.md
  test -f docs/RELEASE.md
  rg -q "docs/ARCHITECTURE.md" README.md
  rg -q "docs/COMMANDS.md" README.md
  rg -q "docs/RELEASE.md" README.md
  rg -q "CHANGELOG.md" README.md
  if rg -n "internal/tui/layout|internal/fileutil|internal/restoreexec|internal/deleteexec|session/scanner_head.go|session/migrate.go" docs README.md; then \
    echo "docs still reference removed/legacy paths"; \
    exit 1; \
  fi

# Run unit test suite
test:
  {{go_with_experiment}} test ./...

# Run integration tests
test-integration:
  {{go_with_experiment}} test -tags=integration {{integration_pkg}}

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
  {{go_with_experiment}} test -run='^$' -bench='Benchmark(SortTUISessions_3k|SortTUISessions_10k|BuildPreviewLines|PreviewIndex)' -benchmem ./tui

# Run session scan/filter micro-benchmarks
bench-session:
  {{go_with_experiment}} test -run='^$' -bench='Benchmark(ScanSessions|FilterSessions)' -benchmem ./session

# Run CLI rendering and doctor-risk micro-benchmarks
bench-cli:
  {{go_with_experiment}} test -run='^$' -bench='Benchmark(RenderTable|RenderJSON|DoctorRiskJSON)' -benchmem ./cli

# Run CI smoke checks against built binary and risk fixture dataset
ci-smoke:
  set -e; \
  ci_config_dir="${XDG_CONFIG_HOME:-$HOME/.config}/codexsm"; \
  ci_state_dir="${XDG_STATE_HOME:-$HOME/.local/state}/codexsm/ci"; \
  mkdir -p "$ci_config_dir" "$ci_state_dir"; \
  {{go_with_experiment}} build -ldflags="-X main.version={{version}}" -o {{bin}} .; \
  ./{{bin}} restore --help | grep -q -- "--batch-id"; \
  ./{{bin}} delete --help | grep -q -- "--preview"; \
  ./{{bin}} config --help | grep -q -- "show"; \
  ./{{bin}} config --help | grep -q -- "validate"; \
  CSM_CONFIG="$ci_config_dir/config.json" ./{{bin}} config init --dry-run | grep -q -- "\"sessions_root\""; \
  if ./{{bin}} restore --batch-id b-test --id deadbeef >"$ci_state_dir/restore-conflict.log" 2>&1; then \
    echo "expected restore conflict to fail"; \
    exit 1; \
  fi; \
  grep -q "cannot be combined" "$ci_state_dir/restore-conflict.log"; \
  ./{{bin}} doctor >"$ci_state_dir/doctor.txt"; \
  rc=0; \
  ./{{bin}} doctor risk --sessions-root ./testdata/fixtures/risky-static/sessions --format json --sample-limit 3 >"$ci_state_dir/risk.json" || rc=$?; \
  if [ "$rc" -ne 1 ]; then \
    echo "expected doctor risk fixture check to exit 1, got $rc"; \
    exit 1; \
  fi; \
  python3 scripts/check_doctor_risk_json.py "$ci_state_dir/risk.json" --sample-limit 3

# Enforce TUI benchmark latency guardrails (ns/op)
bench-gate:
  if ! out="$({{go_with_experiment}} test -run='^$' -bench='Benchmark(SortTUISessions_3k|SortTUISessions_10k)$' ./tui -count=1 2>&1)"; then \
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
    --output-root {{gen_output_root}}

# Generate a compact extreme dataset for local regression checks
gen-sessions-extreme:
  python3 scripts/gen_seeded_sessions.py \
    --seed {{gen_seed}} \
    --count 0 \
    --risk-missing-meta-count 1 \
    --risk-corrupted-count 1 \
    --large-file-count 1 \
    --oversize-meta-count 1 \
    --oversize-user-count 1 \
    --oversize-assistant-count 1 \
    --no-newline-count 1 \
    --mixed-corrupt-huge-count 1 \
    --unicode-wide-count 1 \
    --long-message-bytes 65536 \
    --meta-line-bytes 32768 \
    --large-file-target-bytes 262144 \
    --payload-shape mixed \
    --output-root {{gen_output_root}}

# Generate a larger stress dataset for manual benchmarking and memory checks
gen-sessions-large:
  python3 scripts/gen_seeded_sessions.py \
    --seed {{gen_seed}} \
    --count 200 \
    --risk-missing-meta-count 3 \
    --risk-corrupted-count 3 \
    --large-file-count 6 \
    --oversize-meta-count 4 \
    --oversize-user-count 6 \
    --oversize-assistant-count 6 \
    --no-newline-count 4 \
    --mixed-corrupt-huge-count 4 \
    --unicode-wide-count 4 \
    --long-message-bytes 262144 \
    --meta-line-bytes 131072 \
    --large-file-target-bytes 2097152 \
    --payload-shape log-heavy \
    --output-root {{gen_output_root}}

# Generate a large dataset and run list/doctor risk smoke checks locally
stress-cli:
  set -e; \
  ci_state_dir="${XDG_STATE_HOME:-$HOME/.local/state}/codexsm/ci"; \
  mkdir -p "$ci_state_dir"; \
  tmpdir="$(mktemp -d "${TMPDIR:-/tmp}/codexsm-stress-cli.XXXXXX")"; \
  trap 'rm -rf "$tmpdir"' EXIT; \
  {{go_with_experiment}} build -ldflags="-X main.version={{version}}" -o {{bin}} .; \
  python3 scripts/gen_seeded_sessions.py \
    --seed {{gen_seed}} \
    --count 200 \
    --risk-missing-meta-count 3 \
    --risk-corrupted-count 3 \
    --large-file-count 6 \
    --oversize-meta-count 4 \
    --oversize-user-count 6 \
    --oversize-assistant-count 6 \
    --no-newline-count 4 \
    --mixed-corrupt-huge-count 4 \
    --unicode-wide-count 4 \
    --long-message-bytes 262144 \
    --meta-line-bytes 131072 \
    --large-file-target-bytes 2097152 \
    --payload-shape log-heavy \
    --output-root "$tmpdir/sessions"; \
  ./codexsm list --sessions-root "$tmpdir/sessions" --limit 50 --format json >"$ci_state_dir/stress-list.json"; \
  rc=0; \
  ./codexsm doctor risk --sessions-root "$tmpdir/sessions" --format json --sample-limit 5 >"$ci_state_dir/stress-risk.json" || rc=$?; \
  if [ "$rc" -ne 0 ] && [ "$rc" -ne 1 ]; then \
    echo "doctor risk stress smoke failed with rc=$rc"; \
    exit 1; \
  fi; \
  test -s "$ci_state_dir/stress-list.json"; \
  test -s "$ci_state_dir/stress-risk.json"

# Remove generated coverage files
clean:
  rm -f coverage_unit.out coverage_integration.out

# Run the smallest full quality gate
check: fmt lint docs-check test test-integration build
