set shell := ["bash", "-cu"]

go := "go"
gofumpt := if env_var_or_default("GOFUMPT", "") != "" {
  env_var("GOFUMPT")
} else {
  "gofumpt"
}
goexperiment := "GOEXPERIMENT=jsonv2"
version := env_var_or_default("VERSION", "dev")
unit_cov_min := env_var_or_default("UNIT_COV_MIN", "50")
integration_cov_min := env_var_or_default("INTEGRATION_COV_MIN", "65")

default:
  @just --list

tools:
  {{goexperiment}} {{go}} install mvdan.cc/gofumpt@latest

fmt:
  {{gofumpt}} -w .

lint:
  {{goexperiment}} {{go}} vet ./...

test:
  {{goexperiment}} {{go}} test ./...

test-integration:
  {{goexperiment}} {{go}} test -tags=integration ./cli

test-all: test test-integration

cover-unit:
  {{goexperiment}} {{go}} test -count=1 ./... -coverprofile=coverage_unit.out
  {{go}} tool cover -func=coverage_unit.out

cover-integration:
  {{goexperiment}} {{go}} test -count=1 -tags=integration ./cli -coverprofile=coverage_integration.out
  {{go}} tool cover -func=coverage_integration.out

cover: cover-unit cover-integration

cover-gate:
  {{goexperiment}} {{go}} test -count=1 ./... -coverprofile=coverage_unit.out
  awk -v got="$({{go}} tool cover -func=coverage_unit.out | awk '/^total:/ {gsub("%","",$3); print $3}')" -v min="{{unit_cov_min}}" 'BEGIN { if (got+0 < min+0) { printf("unit coverage %.1f%% < %.1f%%\n", got, min); exit 1 } else { printf("unit coverage %.1f%% >= %.1f%%\n", got, min) } }'
  {{goexperiment}} {{go}} test -count=1 -tags=integration ./cli -coverprofile=coverage_integration.out
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

build:
  {{goexperiment}} {{go}} build -ldflags="-X main.version={{version}}" -o codex-sm .

check: fmt lint test-all build
