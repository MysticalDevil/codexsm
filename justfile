set shell := ["bash", "-cu"]

go := "go"
gofumpt := if env_var_or_default("GOFUMPT", "") != "" {
  env_var("GOFUMPT")
} else {
  "gofumpt"
}
goexperiment := "GOEXPERIMENT=jsonv2"

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

build:
  {{goexperiment}} {{go}} build -o codex-sm .

check: fmt lint test-all build
