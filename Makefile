# Variables
BINARY_NAME=opsvault
BUILD_DIR=bin
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags="-s -w -X OpsVault/cmd.Version=$(VERSION)"

.PHONY: all build build-linux build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows build-all copy-configs clean fmt vet run help release\:patch release\:minor release\:major

all: build build-linux

# Help information
help:
	@echo "Usage:"
	@echo "  make build                  - Build for current OS (with configs)"
	@echo "  make build-linux            - Build for Linux (amd64)"
	@echo "  make build-linux-arm64      - Build for Linux (arm64)"
	@echo "  make build-darwin-amd64     - Build for macOS (amd64)"
	@echo "  make build-darwin-arm64     - Build for macOS (arm64)"
	@echo "  make build-windows          - Build for Windows (amd64)"
	@echo "  make build-all              - Build for all platforms"
	@echo "  make copy-configs           - Copy default.yaml & ansible.yaml into $(BUILD_DIR)/configs"
	@echo "  make release:patch          - Bump patch version (v0.0.X) & trigger GitHub Release"
	@echo "  make release:minor          - Bump minor version (v0.X.0) & trigger GitHub Release"
	@echo "  make release:major          - Bump major version (vX.0.0) & trigger GitHub Release"
	@echo "  make clean                  - Clean build directory"
	@echo "  make fmt                    - Format Go source code"
	@echo "  make vet                    - Run Go vet check"
	@echo "  make run                    - Build and run the app locally"

# Copy clean config templates to build output directory without overwriting existing user configs
copy-configs:
	@mkdir -p $(BUILD_DIR)/configs
	@[ -f $(BUILD_DIR)/configs/default.yaml ] || cp configs/default.example.yaml $(BUILD_DIR)/configs/default.yaml
	@[ -f $(BUILD_DIR)/configs/ansible.yaml ] || cp configs/ansible.example.yaml $(BUILD_DIR)/configs/ansible.yaml
	@cp configs/default.example.yaml $(BUILD_DIR)/configs/default.example.yaml
	@cp configs/ansible.example.yaml $(BUILD_DIR)/configs/ansible.example.yaml
	@echo "Synced config files to $(BUILD_DIR)/configs/ (default.yaml, ansible.yaml)"

# Local Build
build: copy-configs
	@echo "Building for current OS ($(VERSION))..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) main.go

# Linux Build (amd64)
build-linux: copy-configs
	@echo "Building for Linux (amd64) ($(VERSION))..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 main.go

# Linux Build (arm64)
build-linux-arm64: copy-configs
	@echo "Building for Linux (arm64) ($(VERSION))..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 main.go

# macOS Build (amd64)
build-darwin-amd64: copy-configs
	@echo "Building for macOS (amd64) ($(VERSION))..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 main.go

# macOS Build (arm64 / Apple Silicon)
build-darwin-arm64: copy-configs
	@echo "Building for macOS (arm64) ($(VERSION))..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 main.go

# Windows Build
build-windows: copy-configs
	@echo "Building for Windows (amd64) ($(VERSION))..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe main.go

# Build all platforms
build-all: build-linux build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows

# Semantic version release targets via GitHub CLI & GitHub Actions
release\:patch:
	@bash scripts/release.sh patch

release\:minor:
	@bash scripts/release.sh minor

release\:major:
	@bash scripts/release.sh major

# Run locally
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Clean build directory
clean:
	@echo "Cleaning build directory..."
	rm -rf $(BUILD_DIR)

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Static check
vet:
	@echo "Running static checks..."
	go vet ./...
