.PHONY: all build generate parser generate-check fmt test race vet lint verify clean help
.DEFAULT_GOAL := help

GOYACC ?= go run golang.org/x/tools/cmd/goyacc@v0.52.0
GOLANGCI_LINT_VERSION := v2.15.0

all: verify

build: ## Build all packages
	go build ./...

test: ## Run unit tests
	go test ./...

race: ## Run tests with race detection and randomized order
	go test -race -shuffle=on ./...

vet: ## Run Go static analysis
	go vet ./...

lint: ## Run the pinned golangci-lint version
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run ./...

verify: build vet race lint ## Run all quality gates

parser/parser_generated.go: parser/grammar.y
	$(MAKE) generate

parser: generate ## Alias for generate

generate: ## Regenerate the parser and reject grammar conflicts
	@set -eu; \
	output=$$(mktemp); \
	report=$$(mktemp); \
	trap 'rm -f "$$output" "$$report"' EXIT HUP INT TERM; \
	if ! $(GOYACC) -l -o parser/parser_generated.go -v "$$report" parser/grammar.y >"$$output" 2>&1; then \
		cat "$$output"; exit 1; \
	fi; \
	sed -i '1s# -v .*parser/grammar.y# -v parser/parser.output parser/grammar.y#' parser/parser_generated.go; \
	cat "$$output"; \
	if grep -Eq '(shift|reduce)/reduce' "$$output"; then \
		echo 'Grammar conflicts found in grammar.y'; exit 1; \
	fi
	go run github.com/mia0x75/parserkit/cmd/yaccspan parser/parser_generated.go

generate-check: ## Verify generated files match the grammar
	@set -eu; \
	before=$$(mktemp); after=$$(mktemp); \
	trap 'rm -f "$$before" "$$after"' EXIT HUP INT TERM; \
	sha256sum parser/parser_generated.go >"$$before"; \
	$(MAKE) generate; \
	sha256sum parser/parser_generated.go >"$$after"; \
	cmp "$$before" "$$after"

fmt: ## Format Go source files
	@echo "gofmt (simplify)"
	@gofmt -s -w .
	@if [ -n "$$(gofmt -s -l .)" ]; then \
		echo "gofmt: unformatted files remain:"; \
		gofmt -s -l .; \
		exit 1; \
	fi

clean: ## Clear Go build artifacts
	go clean ./...

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "%-20s %s\n", $$1, $$2}'
