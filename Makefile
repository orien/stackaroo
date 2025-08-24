# Stackaroo Makefile
# Copyright © 2025 Stackaroo Contributors
# SPDX-License-Identifier: BSD-3-Clause

.PHONY: build test test-unit test-aws clean help run install lint fmt vet

# Variables
BINARY_NAME := stackaroo
BUILD_DIR := bin
CMD_DIR := ./cmd
INTERNAL_DIR := ./internal

# Default target
help: ## Show this help message
	@echo "Available targets:"
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Build

build: ## Build the main stackaroo binary
	@echo "🔨 Building stackaroo..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) .
	@echo "✅ Binary built: $(BUILD_DIR)/$(BINARY_NAME)"

build-test-aws: ## Build the AWS module test program
	@echo "🔨 Building AWS test program..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/test-aws $(CMD_DIR)/test-aws
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

##@ AWS Testing Shortcuts

aws-test-us-east-1: build-test-aws ## Test AWS module in us-east-1
	@$(BUILD_DIR)/test-aws -region=us-east-1 -dry-run=true -verbose=true

aws-test-us-west-2: build-test-aws ## Test AWS module in us-west-2
	@$(BUILD_DIR)/test-aws -region=us-west-2 -dry-run=true -verbose=true

aws-test-profile: build-test-aws ## Test AWS module with specific profile (set PROFILE env var)
	@$(BUILD_DIR)/test-aws -profile=$(PROFILE) -dry-run=true -verbose=true

##@ Release

version: ## Show version information
	@echo "Stackaroo version information:"
	@echo "Go version: $(shell go version)"
	@echo "Git commit: $(shell git rev-parse --short HEAD 2>/dev/null || echo 'unknown')"
	@echo "Build time: $(shell date)"

release-build: clean ## Build release binaries for multiple platforms
	@echo "🚀 Building release binaries..."
	@mkdir -p $(BUILD_DIR)/release
	@GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-amd64 .
	@GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/release/$(BINARY_NAME)-linux-arm64 .
	@GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-amd64 .
	@GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/release/$(BINARY_NAME)-darwin-arm64 .
	@GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/release/$(BINARY_NAME)-windows-amd64.exe .
	@echo "✅ Release binaries built in $(BUILD_DIR)/release/"
	@ls -la $(BUILD_DIR)/release/

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

deps: ## Show module dependencies
	@echo "📦 Module dependencies:"
	@go list -m all

env: ## Show Go environment
	@echo "🌍 Go environment:"
	@go env

doctor: ## Run diagnostics
	@echo "🏥 Running diagnostics..."
	@echo "Go version: $(shell go version)"
	@echo "Module: $(shell go list -m)"
	@echo "GOPATH: $(shell go env GOPATH)"
	@echo "GOROOT: $(shell go env GOROOT)"
	@echo "Build cache: $(shell go env GOCACHE)"
	@echo "Module cache: $(shell go env GOMODCACHE)"
	@echo ""
	@echo "Available AWS regions for testing:"
	@echo "  - us-east-1 (N. Virginia)"
	@echo "  - us-west-2 (Oregon)"
	@echo "  - eu-west-1 (Ireland)"
	@echo ""
	@echo "To test with specific region: make aws-test-us-east-1"
	@echo "To test with specific profile: PROFILE=myprofile make aws-test-profile"
