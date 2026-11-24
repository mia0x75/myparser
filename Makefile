.PHONY: all build parser generate-check fmt test race vet lint verify clean help
.DEFAULT_GOAL := help

GOYACC:=go run golang.org/x/tools/cmd/goyacc@v0.52.0
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

parser/parser_generated.go: parser/parser.y
	$(MAKE) parser

parser: ## Regenerate the parser and reject grammar conflicts
	@set -eu; \
	output=$$(mktemp); \
	report=$$(mktemp); \
	trap 'rm -f "$$output" "$$report"' EXIT HUP INT TERM; \
	if ! $(GOYACC) -l -o parser/parser_generated.go -v "$$report" parser/parser.y >"$$output" 2>&1; then \
		cat "$$output"; exit 1; \
	fi; \
	sed -i '1s# -v .*parser/parser.y# -v parser/parser.output parser/parser.y#' parser/parser_generated.go; \
	cat "$$output"; \
	if grep -Eq '(shift|reduce)/reduce' "$$output"; then \
		echo 'Grammar conflicts found in parser.y'; exit 1; \
	fi
	go run github.com/mia0x75/parserkit/cmd/yaccspan parser/parser_generated.go

generate-check: parser ## Verify the committed generated parser is up to date
	@set -eu; \
	dir=$$(mktemp -d); \
	trap 'rm -f "$$dir/committed" "$$dir/expected" "$$dir/actual"; rmdir "$$dir"' EXIT HUP INT TERM; \
	if git cat-file -e HEAD:parser/parser_generated.go 2>/dev/null; then \
		git show HEAD:parser/parser_generated.go >"$$dir/committed"; \
	else \
		git show HEAD:parser.go >"$$dir/committed"; \
	fi; \
	sed '1d' "$$dir/committed" >"$$dir/expected"; \
	sed -i 's/^package myparser$$/package parser/' "$$dir/expected"; \
	sed '1d' parser/parser_generated.go >"$$dir/actual"; \
	cmp "$$dir/expected" "$$dir/actual"

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
