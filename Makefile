# Stackaroo Makefile
# Copyright © 2025 Stackaroo Contributors
# SPDX-License-Identifier: BSD-3-Clause

.PHONY: build test test-unit test-aws clean help run install lint fmt vet

# Variables
BINARY_NAME := stackaroo
BUILD_DIR := bin
CMD_DIR := ./cmd
INTERNAL_DIR := ./internal

# Version variables
BASE_VERSION := $(shell cat VERSION 2>/dev/null || echo "0.0.0")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')
BUILD_DATE := $(shell date -u '+%Y-%m-%d %H:%M:%S UTC')

# Determine if we're on a clean release tag
GIT_TAG := $(shell git describe --exact-match --tags 2>/dev/null || echo "")
GIT_DIRTY := $(shell test -z "$$(git status --porcelain 2>/dev/null)" || echo "-dirty")

# Version logic:
# - If on exact tag matching VERSION file: use clean version (v1.0.0)
# - Otherwise: append git info (1.0.0+a1b2c3d or 1.0.0+a1b2c3d-dirty)
VERSION := $(if $(and $(GIT_TAG),$(filter v$(BASE_VERSION),$(GIT_TAG))),v$(BASE_VERSION),$(BASE_VERSION)+$(GIT_COMMIT)$(GIT_DIRTY))

# Build flags
LDFLAGS := -ldflags="-w -s \
	-X 'github.com/orien/stackaroo/internal/version.Version=$(VERSION)' \
	-X 'github.com/orien/stackaroo/internal/version.GitCommit=$(GIT_COMMIT)' \
	-X 'github.com/orien/stackaroo/internal/version.BuildDate=$(BUILD_DATE)'"

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build

build: ## Build the main stackaroo binary
	@echo "🔨 Building stackaroo $(VERSION)..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "✅ Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

build-test-aws: ## Build the AWS module test program
	@echo "🔨 Building AWS test program..."
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/test-aws $(CMD_DIR)/test-aws
	@echo "✅ AWS test program built: $(BUILD_DIR)/test-aws"

build-all: build build-test-aws ## Build all binaries

##@ Test

test: test-unit ## Run all tests

test-unit: ## Run unit tests
	@echo "🧪 Running unit tests..."
	@go test -v ./internal/...

test-aws: build-test-aws ## Run AWS module test program (dry-run)
	@echo "🔍 Testing AWS module..."
	@$(BUILD_DIR)/test-aws -dry-run=true -verbose=true

test-aws-live: build-test-aws ## Run AWS module test program against real AWS (BE CAREFUL!)
	@echo "⚠️  Running AWS module test against REAL AWS..."
	@echo "⚠️  This will create real resources. Press Ctrl+C to cancel."
	@sleep 3
	@$(BUILD_DIR)/test-aws -dry-run=false -verbose=true

##@ Development

run: build ## Build and run stackaroo
	@echo "🚀 Running stackaroo..."
	@$(BUILD_DIR)/$(BINARY_NAME)

install: ## Install stackaroo to GOPATH/bin
	@echo "📦 Installing stackaroo..."
	@go install .
	@echo "✅ Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

fmt: ## Format Go code
	@echo "🎨 Formatting code..."
	@go fmt ./...

vet: ## Run go vet
	@echo "🔍 Running go vet..."
	@go vet ./...

golangci-lint: ## Run golangci-lint
	@echo "🔍 Running golangci-lint..."
	@golangci-lint run

install-golangci-lint: ## Install golangci-lint
	@echo "📦 Installing golangci-lint..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "✅ golangci-lint installed"

lint: fmt vet golangci-lint ## Run all linting tools

tidy: ## Tidy and verify module dependencies
	@echo "🧹 Tidying module dependencies..."
	@go mod tidy
	@go mod verify

##@ Clean

clean: ## Clean build artifacts
	@echo "🧹 Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@go clean

clean-all: clean ## Clean all artifacts including module cache
	@echo "🧹 Cleaning all artifacts..."
	@go clean -modcache




##@ Release

version: ## Show version information that would be embedded in binary
	@echo "Stackaroo version information:"
	@echo "Version: $(VERSION)"
	@echo "Base version: $(BASE_VERSION)"
	@echo "Git commit: $(GIT_COMMIT)"
	@echo "Build date: $(BUILD_DATE)"
	@echo "Git tag: $(GIT_TAG)"
	@echo "Git dirty: $(GIT_DIRTY)"
	@echo "Go version: $(shell go version)"


install-goreleaser: ## Install GoReleaser
	@echo "📦 Installing GoReleaser..."
	@go install github.com/goreleaser/goreleaser@latest
	@echo "✅ GoReleaser installed"

goreleaser-check: ## Check GoReleaser configuration
	@echo "🔍 Checking GoReleaser configuration..."
	@goreleaser check

goreleaser-snapshot: clean ## Build snapshot release with GoReleaser (no git tag required)
	@echo "📸 Building snapshot release with GoReleaser..."
	@goreleaser release --snapshot --clean
	@echo "✅ Snapshot release built in dist/"
	@ls -la dist/

goreleaser-dry-run: ## Dry run GoReleaser release process
	@echo "🧪 Running GoReleaser dry run..."
	@goreleaser release --skip=publish --clean
	@echo "✅ Dry run completed"

release-prepare: ## Prepare for release (run checks, validate config)
	@echo "🚀 Preparing for release..."
	@./scripts/release.sh --dry-run $(shell cat VERSION)

release: ## Create and push release tag (requires version argument: make release VERSION=1.2.3)
ifdef VERSION
	@echo "🚀 Creating release $(VERSION)..."
	@./scripts/release.sh $(VERSION)
else
	@echo "❌ VERSION argument required. Usage: make release VERSION=1.2.3"
	@exit 1
endif

##@ Git

git-check: ## Check git status and requirements
	@echo "📋 Git status:"
	@git status --porcelain
	@echo "📋 Recent commits:"
	@git --no-pager log --oneline -5

lint-fix: ## Run golangci-lint with auto-fix
	@echo "🔧 Running golangci-lint with auto-fix..."
	@golangci-lint run --fix

commit-check: lint test git-check ## Run pre-commit checks
	@echo "✅ All checks passed!"

##@ Info
