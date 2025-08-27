# Variables
BINARY_NAME := orchcli
PACKAGE := github.com/kubeorchestra/cli
VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X '$(PACKAGE)/cmd.version=$(VERSION)' \
           -X '$(PACKAGE)/cmd.commit=$(COMMIT)' \
           -X '$(PACKAGE)/cmd.buildDate=$(BUILD_DATE)'

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
NC := \033[0m # No Color

.PHONY: all build clean test install run help version

## help: Display this help message
help:
	@echo "OrchCLI Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make <target>"
	@echo ""
	@echo "Targets:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  ${GREEN}%-15s${NC} %s\n", $$1, $$2}' $(MAKEFILE_LIST)

## all: Build the binary
all: build

## build: Build the binary with version information
build:
	@echo "$(GREEN)Building $(BINARY_NAME)...$(NC)"
	@go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) main.go
	@echo "$(GREEN)Build complete!$(NC)"

## install: Install the binary to GOPATH/bin
install:
	@echo "$(GREEN)Installing $(BINARY_NAME)...$(NC)"
	@go install -ldflags "$(LDFLAGS)" .
	@echo "$(GREEN)Installed to $$(go env GOPATH)/bin/$(BINARY_NAME)$(NC)"

## clean: Remove build artifacts
clean:
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@rm -f $(BINARY_NAME)
	@echo "$(GREEN)Clean complete!$(NC)"

## test: Run all tests with coverage
test:
	@echo "$(GREEN)Running tests with coverage...$(NC)"
	@go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	@echo "$(GREEN)Coverage report:$(NC)"
	@go tool cover -func=coverage.out

## test-unit: Run unit tests only
test-unit:
	@echo "$(GREEN)Running unit tests...$(NC)"
	@go test -v -short ./tests/unit/...

## test-integration: Run integration tests
test-integration:
	@echo "$(GREEN)Running integration tests...$(NC)"
	@RUN_INTEGRATION_TESTS=true go test -v -tags=integration ./tests/integration/...

## test-coverage: Generate HTML coverage report
test-coverage: test
	@echo "$(GREEN)Generating HTML coverage report...$(NC)"
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generated: coverage.html$(NC)"

## test-benchmark: Run benchmarks
test-benchmark:
	@echo "$(GREEN)Running benchmarks...$(NC)"
	@go test -bench=. -benchmem ./...

## run: Run the CLI directly
run:
	@go run -ldflags "$(LDFLAGS)" main.go

## version: Display version information
version:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT)"
	@echo "Build Date: $(BUILD_DATE)"

## mod-tidy: Tidy and verify go modules
mod-tidy:
	@echo "$(GREEN)Tidying modules...$(NC)"
	@go mod tidy
	@go mod verify

## fmt: Format Go code
fmt:
	@echo "$(GREEN)Formatting code...$(NC)"
	@go fmt ./...

## vet: Run go vet
vet:
	@echo "$(GREEN)Running go vet...$(NC)"
	@go vet ./...

## lint: Run golangci-lint (requires golangci-lint to be installed)
lint:
	@echo "$(GREEN)Running linters...$(NC)"
	@which golangci-lint > /dev/null || (echo "$(RED)golangci-lint not installed$(NC)" && exit 1)
	@golangci-lint run --config=.golangci.yml

## lint-fix: Run golangci-lint with auto-fix
lint-fix:
	@echo "$(GREEN)Running linters with auto-fix...$(NC)"
	@which golangci-lint > /dev/null || (echo "$(RED)golangci-lint not installed$(NC)" && exit 1)
	@golangci-lint run --fix --config=.golangci.yml

## install-tools: Install development tools
install-tools:
	@echo "$(GREEN)Installing development tools...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "$(GREEN)Tools installed!$(NC)"

## build-all: Build for multiple platforms
build-all:
	@echo "$(GREEN)Building for multiple platforms...$(NC)"
	@mkdir -p dist
	@GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME)_darwin_amd64 main.go
	@GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME)_darwin_arm64 main.go
	@GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME)_linux_amd64 main.go
	@GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME)_linux_arm64 main.go
	@GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME)_windows_amd64.exe main.go
	@echo "$(GREEN)Multi-platform build complete! Binaries in dist/$(NC)"

## npm-build: Build binary for npm package
npm-build:
	@echo "$(GREEN)Building binary for npm package...$(NC)"
	@VERSION=$$(node -p "require('./package.json').version") && \
		go build -ldflags "-X 'github.com/kubeorchestra/cli/cmd.version=$$VERSION' -X 'github.com/kubeorchestra/cli/cmd.commit=$$(git rev-parse --short HEAD 2>/dev/null || echo 'release')' -X 'github.com/kubeorchestra/cli/cmd.buildDate=$$(date -u +"%Y-%m-%dT%H:%M:%SZ")'" -o npm/bin/$(BINARY_NAME)-bin main.go
	@chmod +x npm/bin/$(BINARY_NAME)-bin
	@echo "$(GREEN)NPM binary built!$(NC)"

## npm-test: Test npm package locally
npm-test: npm-build
	@echo "$(GREEN)Testing npm package locally...$(NC)"
	@npm pack
	@echo "$(GREEN)Package created! Test with: npm install -g kubeorchestra-orchcli-*.tgz$(NC)"

## npm-publish: Publish to npm registry
npm-publish: npm-build
	@echo "$(GREEN)Publishing to npm registry...$(NC)"
	@npm publish --access public
	@echo "$(GREEN)Published to npm!$(NC)"