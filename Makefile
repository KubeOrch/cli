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

## test: Run tests
test:
	@echo "$(GREEN)Running tests...$(NC)"
	@go test -v ./...

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
	@golangci-lint run

## build-all: Build for multiple platforms
build-all:
	@echo "$(GREEN)Building for multiple platforms...$(NC)"
	@mkdir -p dist
	@GOOS=darwin GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME)-darwin-amd64 main.go
	@GOOS=darwin GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME)-darwin-arm64 main.go
	@GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME)-linux-amd64 main.go
	@GOOS=linux GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME)-linux-arm64 main.go
	@GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY_NAME)-windows-amd64.exe main.go
	@echo "$(GREEN)Multi-platform build complete! Binaries in dist/$(NC)"