# gh-custom-properties
#
# Standard entry points: deps, build, install, test, lint, default, all.
# Run `make help` (or just `make`) to list every target.

BINARY_NAME ?= gh-custom-properties
GOFLAGS ?=
ARGS ?=

.PHONY: default all deps build install run test test/cover vet fmt fmt/check lint docs/gen clean help

default: help

## Run the full local pipeline (mirrors CI): format check, vet, lint, test, build
all: fmt/check vet lint test build

## Download and verify module dependencies
deps:
	go mod download
	go mod verify

## Build the gh-custom-properties binary
build: deps
	go build $(GOFLAGS) -o $(BINARY_NAME) .

## Install gh-custom-properties into $GOBIN (or $GOPATH/bin)
install: deps
	go install $(GOFLAGS) .

## Run the TUI (use ARGS="..." to pass flags, e.g. make run ARGS="--file repos.txt")
run: build
	./$(BINARY_NAME) $(ARGS)

## Run the test suite
test: deps
	go test ./...

## Run tests with coverage and print a per-function summary
test/cover: deps
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1

## Run go vet
vet:
	go vet ./...

## Format all Go source files in place
fmt:
	gofmt -w .

## Fail if any file isn't gofmt-formatted (used in CI instead of fmt, which mutates)
fmt/check:
	@test -z "$$(gofmt -l .)" || { gofmt -l .; echo "run 'make fmt' to fix" >&2; exit 1; }

## Run golangci-lint (see https://golangci-lint.run/welcome/install/ if it's not installed)
lint:
	golangci-lint run ./...

## Regenerate docs/gh-custom-properties.1 and .html from internal/clidoc
docs/gen:
	go generate ./...

## Remove build artifacts
clean:
	rm -f $(BINARY_NAME) coverage.out

## Show this help
help:
	@printf "Usage: make <target>\n\nAvailable targets:\n\n"
	@awk '/^[a-zA-Z0-9\-_%:\\\/]+:/ { \
		helpMessage = match(lastLine, /^## (.*)/); \
		if (helpMessage) { \
			helpCommand = $$1; \
			helpMessage = substr(lastLine, RSTART + 3, RLENGTH); \
			gsub("\\\\", "", helpCommand); \
			gsub(":+$$", "", helpCommand); \
			printf "  \033[36m%-14s\033[0m %s\n", helpCommand, helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST) | sort -u
	@printf "\n"
