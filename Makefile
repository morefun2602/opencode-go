# opencode-go — 常用开发命令（与 .github/workflows/ci.yml 对齐：vet + test）
SHELL := /bin/bash

GO ?= go
BIN_DIR ?= bin
BINARY ?= opencode
OUT ?= $(BIN_DIR)/$(BINARY)

.PHONY: help build run test test-race vet check ci fmt tidy clean

.DEFAULT_GOAL := help

help:
	@echo "Targets:"
	@echo "  make build      - build binary to $(OUT)"
	@echo "  make run        - go run ./cmd/opencode"
	@echo "  make test       - go test ./..."
	@echo "  make test-race  - go test -race ./..."
	@echo "  make vet        - go vet ./..."
	@echo "  make check      - vet + test (same as ci)"
	@echo "  make ci         - alias for check"
	@echo "  make fmt        - go fmt ./..."
	@echo "  make tidy       - go mod tidy"
	@echo "  make clean      - remove $(BIN_DIR)/"
	@echo "Variables: GO=$(GO) OUT=$(OUT)"

build:
	@mkdir -p $(BIN_DIR)
	$(GO) build -o $(OUT) ./cmd/opencode

run:
	$(GO) run ./cmd/opencode

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

vet:
	$(GO) vet ./...

check: vet test
ci: check

fmt:
	$(GO) fmt ./...

tidy:
	$(GO) mod tidy

clean:
	rm -rf $(BIN_DIR)
