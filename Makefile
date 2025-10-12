# GLF - GitLab Fuzzy Finder
# Makefile for cross-platform builds

# Binary name
BINARY=glf

# Version from git tag or default
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')

# Build flags
LDFLAGS=-s -w
LDFLAGS+=-X main.version=$(VERSION)
LDFLAGS+=-X main.commit=$(COMMIT)
LDFLAGS+=-X main.buildTime=$(BUILD_TIME)

# Go build flags
GOFLAGS=-trimpath
TAGS=

# Platforms
PLATFORMS=darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 windows/amd64

# Directories
BUILD_DIR=build
DIST_DIR=dist

.PHONY: all build clean install uninstall test lint fmt help \
	build-linux build-macos build-windows build-all release

# Default target
all: build

## build: Build for current platform
build:
	@echo "Building $(BINARY) $(VERSION)..."
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -tags "$(TAGS)" -o $(BINARY) ./cmd/glf
	@echo "✓ Build complete: ./$(BINARY)"

## build-linux: Build for Linux (amd64 and arm64)
build-linux:
	@echo "Building for Linux..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -tags "$(TAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-amd64 ./cmd/glf
	GOOS=linux GOARCH=arm64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -tags "$(TAGS)" -o $(BUILD_DIR)/$(BINARY)-linux-arm64 ./cmd/glf
	@echo "✓ Linux builds complete"

## build-macos: Build for macOS (amd64 and arm64)
build-macos:
	@echo "Building for macOS..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -tags "$(TAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-amd64 ./cmd/glf
	GOOS=darwin GOARCH=arm64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -tags "$(TAGS)" -o $(BUILD_DIR)/$(BINARY)-darwin-arm64 ./cmd/glf
	@echo "✓ macOS builds complete"

## build-windows: Build for Windows (amd64)
build-windows:
	@echo "Building for Windows..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -tags "$(TAGS)" -o $(BUILD_DIR)/$(BINARY)-windows-amd64.exe ./cmd/glf
	@echo "✓ Windows build complete"

## build-all: Build for all platforms
build-all: build-linux build-macos build-windows
	@echo "✓ All platform builds complete"
	@ls -lh $(BUILD_DIR)/

## release: Create release archives for all platforms
release: clean build-all
	@echo "Creating release archives..."
	@mkdir -p $(DIST_DIR)
	@cd $(BUILD_DIR) && \
	for file in *; do \
		if [ -f "$$file" ]; then \
			case "$$file" in \
				*.exe) \
					zip "../$(DIST_DIR)/$${file%.exe}.zip" "$$file" ../README.md ../LICENSE 2>/dev/null || zip "../$(DIST_DIR)/$${file%.exe}.zip" "$$file" ../README.md; \
					;; \
				*) \
					tar czf "../$(DIST_DIR)/$$file.tar.gz" "$$file" ../README.md ../LICENSE 2>/dev/null || tar czf "../$(DIST_DIR)/$$file.tar.gz" "$$file" ../README.md; \
					;; \
			esac; \
		fi; \
	done
	@echo "✓ Release archives created in $(DIST_DIR)/"
	@ls -lh $(DIST_DIR)/

## install: Install binary to /usr/local/bin
install: build
	@echo "Installing $(BINARY) to /usr/local/bin..."
	@install -d /usr/local/bin
	@install -m 755 $(BINARY) /usr/local/bin/$(BINARY)
	@echo "✓ Installed successfully"

## uninstall: Remove binary from /usr/local/bin
uninstall:
	@echo "Uninstalling $(BINARY)..."
	@rm -f /usr/local/bin/$(BINARY)
	@echo "✓ Uninstalled successfully"

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BINARY) $(BUILD_DIR) $(DIST_DIR)
	@go clean
	@echo "✓ Clean complete"

## test: Run tests
test:
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...
	@echo "✓ Tests complete"

## lint: Run linters
lint:
	@echo "Running linters..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed. Install with:"; \
		echo "  brew install golangci-lint (macOS)"; \
		echo "  or go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

## fmt: Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✓ Format complete"

## deps: Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "✓ Dependencies updated"

## help: Show this help message
help:
	@echo "GLF Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
