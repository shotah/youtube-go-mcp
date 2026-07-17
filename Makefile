# youtube-go-mcp Makefile — same vibes as go-garmin
#
# Run `make` or `make help` to see everything.

.DEFAULT_GOAL := help

.PHONY: help fmt vet lint test test-short test-race coverage check \
	build cli install tidy deps clean docker self-test auth \
	install-hooks tools run release version

# Build-time version stamp (git describe). Release tags are tracked in ./VERSION.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION) -X github.com/shotah/youtube-go-mcp/internal/mcp.ServerVersion=$(VERSION)

# Release bump: patch (default), minor, or major. Or set TAG=v0.2.0 explicitly.
BUMP ?= patch

# Optional: `make test PKG=./internal/ytmusic/...`
PKG ?= ./...

BINARY ?= bin/youtube-go-mcp
ifeq ($(OS),Windows_NT)
BINARY := bin/youtube-go-mcp.exe
endif

##@ Getting oriented

help: ## Show this help
	@echo.
	@echo Usage:  make ^<target^>
	@echo.
	@echo Getting oriented
	@echo   help                   Show this help
	@echo.
	@echo Daily loop (format -^> lint -^> test)
	@echo   fmt                    Format imports/code (goimports-reviser)
	@echo   vet                    Static analysis (go vet)
	@echo   lint                   Full lint suite (golangci-lint)
	@echo   test                   Unit tests (PKG=./path/... for one package)
	@echo   test-short             Unit tests with -short (skips live InnerTube)
	@echo   test-race              Unit tests with the race detector
	@echo   coverage               Coverage report for internal packages
	@echo   check                  Autofix, lint, and short tests
	@echo.
	@echo Build ^& run
	@echo   build                  Compile all packages (sanity check)
	@echo   cli                    Build static MCP binary into ./bin/
	@echo   install                Install youtube-go-mcp into GOPATH/bin
	@echo   run                    go run MCP  (make run ARGS="--self-test")
	@echo   self-test              Build + run --self-test smoke checks
	@echo   docker                 Build distroless Docker image
	@echo.
	@echo Modules ^& cleanup
	@echo   tidy                   Sync go.mod / go.sum with imports
	@echo   deps                   Download module deps
	@echo   clean                  Remove binaries and coverage artifacts
	@echo.
	@echo Project-specific
	@echo   auth                   Interactive headers export -^> headers.json
	@echo   install-hooks          Install git pre-commit (autofix + lint + test)
	@echo   version                Show VERSION file + next patch (dry-run)
	@echo   release                Bump tag, update VERSION, push (BUMP=patch^|minor^|major)
	@echo.
	@echo Tooling
	@echo   tools                  Install goimports-reviser + golangci-lint v2
	@echo.

##@ Daily loop (format → lint → test)

fmt: ## Autofix imports/code (goimports-reviser + golangci-lint fmt/fix)
	goimports-reviser -format -recursive .
	-golangci-lint fmt ./...
	-golangci-lint run --fix ./...

vet: ## Static analysis (go vet)
	go vet ./...

lint: ## Full lint suite (golangci-lint; no write)
	golangci-lint run ./...

test: ## Unit tests (PKG=./path/... for one package)
	go test $(PKG)

test-short: ## Unit tests with -short (skips live InnerTube)
	go test -short $(PKG)

test-race: ## Unit tests with the race detector
	go test -race $(PKG)

COVERAGE_PKG ?= ./internal/...

coverage: ## Tests + coverage report for internal packages (writes coverage.out; -short skips live InnerTube)
	go test -short -cover "-coverprofile=coverage.out" $(COVERAGE_PKG)
	go tool cover "-func=coverage.out"

check: fmt lint test-short ## Autofix, lint, short tests (matches pre-commit)

##@ Build & run

build: ## Compile all packages (sanity check; no binary kept)
	go build ./...

cli: ## Build the static MCP binary into ./bin/
	mkdir -p bin
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/youtube-go-mcp

install: ## Install youtube-go-mcp into $$GOPATH/bin (or $$GOBIN)
	CGO_ENABLED=0 go install -trimpath -ldflags "$(LDFLAGS)" ./cmd/youtube-go-mcp

run: ## Build & run MCP — e.g. make run ARGS="--self-test"
	go run -ldflags "$(LDFLAGS)" ./cmd/youtube-go-mcp $(ARGS)

self-test: cli ## Build + run --self-test smoke checks
	./$(BINARY) --self-test

docker: ## Build distroless Docker image
	docker build -t youtube-go-mcp:$(VERSION) .

##@ Modules & cleanup

tidy: ## Sync go.mod / go.sum with imports
	go mod tidy

deps: ## Download module deps into the module cache
	go mod download

clean: ## Remove built binaries and coverage artifacts
	go clean ./...
ifeq ($(OS),Windows_NT)
	-cmd /C "rmdir /S /Q bin 2>NUL & del /Q coverage coverage.out coverage.txt youtube-go-mcp.exe 2>NUL"
else
	rm -rf bin
	rm -f coverage coverage.out coverage.txt youtube-go-mcp youtube-go-mcp.exe
endif

##@ Project-specific

auth: ## Interactive browser headers export → headers.json
	go run ./cmd/youtube-go-mcp auth --out headers.json

install-hooks: ## Install git pre-commit hook (autofix + lint + test)
ifeq ($(OS),Windows_NT)
	copy /Y scripts\pre-commit .git\hooks\pre-commit
else
	cp scripts/pre-commit .git/hooks/pre-commit
	chmod +x .git/hooks/pre-commit
endif
	@echo "Installed .git/hooks/pre-commit"

version: ## Show VERSION file and latest git tag / next patch
	@go run ./cmd/release -dry-run

# Bump semver, commit VERSION, annotated-tag, push HEAD + tag (triggers GoReleaser).
# Examples:
#   make release
#   make release BUMP=minor
#   make release TAG=v0.2.0
#   make release DRY_RUN=1
release: ## Bump version tag, update VERSION, push (BUMP=patch|minor|major)
	go run ./cmd/release \
		$(if $(TAG),-version=$(TAG),-bump=$(BUMP)) \
		$(if $(DRY_RUN),-dry-run,) \
		$(if $(SKIP_PUSH),-skip-push,) \
		$(if $(ALLOW_DIRTY),-allow-dirty,)

##@ Tooling

tools: ## Install goimports-reviser + golangci-lint v2 into $$GOBIN
	go install github.com/incu6us/goimports-reviser/v3@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest
	@echo Installed tools. Ensure GOPATH/bin is on PATH, then: golangci-lint version
