.PHONY: all build parser generate-check fmt test race vet lint verify clean help
.DEFAULT_GOAL := help

GOYACC:=go run golang.org/x/tools/cmd/goyacc@v0.38.0
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

parser.go: parser.y
	$(MAKE) parser

parser: ## Regenerate the parser and reject grammar conflicts
	@set -eu; \
	output=$$(mktemp); \
	trap 'rm -f "$$output" y.output' EXIT HUP INT TERM; \
	if ! $(GOYACC) -l -o parser.go parser.y >"$$output" 2>&1; then \
		cat "$$output"; exit 1; \
	fi; \
	cat "$$output"; \
	if grep -Eq '(shift|reduce)/reduce' "$$output"; then \
		echo 'Grammar conflicts found in parser.y'; exit 1; \
	fi

generate-check: parser ## Verify the committed generated parser is up to date
	git diff --exit-code -- parser.go

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
