# Makefile for hist_scanner

# Binary name
BINARY_NAME := hist_scanner

# Version (can be overridden: make VERSION=1.0.0)
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Go parameters
GOCMD := $(shell which go 2>/dev/null || echo "/usr/local/go/bin/go")
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOMOD := $(GOCMD) mod
GOFMT := $(GOCMD) fmt

# Build flags for static, self-contained binaries
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION) -X main.buildTime=$(BUILD_TIME) -X main.commit=$(COMMIT)"

# Output directory
DIST_DIR := dist

# Platforms for cross-compilation
PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64

# Default target: build for current platform
.PHONY: build
build: tidy
	CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) ./cmd/hist_scanner/

# Build for all platforms
.PHONY: all
all: clean tidy $(PLATFORMS)

# Cross-compile for each platform
.PHONY: $(PLATFORMS)
$(PLATFORMS):
	$(eval GOOS := $(word 1,$(subst /, ,$@)))
	$(eval GOARCH := $(word 2,$(subst /, ,$@)))
	$(eval EXT := $(if $(filter windows,$(GOOS)),.exe,))
	$(eval OUTPUT := $(DIST_DIR)/$(BINARY_NAME)-$(GOOS)-$(GOARCH)$(EXT))
	@echo "Building $(OUTPUT)..."
	@mkdir -p $(DIST_DIR)
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) $(GOBUILD) $(LDFLAGS) -o $(OUTPUT) ./cmd/hist_scanner/

# Run tests
.PHONY: test
test:
	$(GOTEST) -v ./...

# Format code
.PHONY: fmt
fmt:
	$(GOFMT) ./...

# Tidy dependencies
.PHONY: tidy
tidy:
	$(GOMOD) tidy

# Clean build artifacts
.PHONY: clean
clean:
	rm -f $(BINARY_NAME)
	rm -rf $(DIST_DIR)

# Install to system (requires root on Linux/macOS)
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME)..."
ifeq ($(shell uname),Linux)
	sudo cp $(BINARY_NAME) /usr/local/bin/
else ifeq ($(shell uname),Darwin)
	sudo cp $(BINARY_NAME) /usr/local/bin/
else
	@echo "Please manually copy $(BINARY_NAME) to your PATH"
endif

# Uninstall from system
.PHONY: uninstall
uninstall:
ifeq ($(shell uname),Linux)
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
else ifeq ($(shell uname),Darwin)
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
else
	@echo "Please manually remove $(BINARY_NAME) from your PATH"
endif

# Show version info that will be embedded
.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Build Time: $(BUILD_TIME)"
	@echo "Commit: $(COMMIT)"

# Create release archives
.PHONY: release
release: all
	@echo "Creating release archives..."
	@cd $(DIST_DIR) && \
	for f in $(BINARY_NAME)-*; do \
		if [ -f "$$f" ]; then \
			if echo "$$f" | grep -q "windows"; then \
				zip "$${f%.exe}.zip" "$$f"; \
			else \
				tar -czf "$$f.tar.gz" "$$f"; \
			fi; \
		fi; \
	done
	@echo "Release archives created in $(DIST_DIR)/"

# List all targets
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build     - Build for current platform (default)"
	@echo "  all       - Build for all platforms"
	@echo "  test      - Run tests"
	@echo "  fmt       - Format code"
	@echo "  tidy      - Tidy go.mod"
	@echo "  clean     - Remove build artifacts"
	@echo "  install   - Install to /usr/local/bin (requires sudo)"
	@echo "  uninstall - Remove from /usr/local/bin (requires sudo)"
	@echo "  version   - Show version info"
	@echo "  release   - Build all platforms and create archives"
	@echo "  help      - Show this help"
	@echo ""
	@echo "Platforms: $(PLATFORMS)"
