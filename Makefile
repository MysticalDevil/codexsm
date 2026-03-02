GO ?= go
GOFUMPT ?= $(shell command -v gofumpt 2>/dev/null || echo "$(HOME)/.local/lib/go/bin/gofumpt")

.PHONY: tools fmt lint test build check

tools:
	$(GO) install mvdan.cc/gofumpt@latest

fmt:
	$(GOFUMPT) -w .

lint:
	$(GO) vet ./...

test:
	$(GO) test ./...

build:
	$(GO) build .

check: fmt lint test build
