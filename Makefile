# Variables
BINARY_NAME=opsvault
BUILD_DIR=bin

.PHONY: all build build-linux build-darwin-amd64 build-darwin-arm64 build-windows clean fmt vet run help

all: build

# Help information
help:
	@echo "Usage:"
	@echo "  make build                  - Build for current OS"
	@echo "  make build-linux            - Build for Linux (amd64)"
	@echo "  make build-darwin-amd64     - Build for macOS (amd64)"
	@echo "  make build-darwin-arm64     - Build for macOS (arm64)"
	@echo "  make build-windows          - Build for Windows (amd64)"
	@echo "  make build-all              - Build for all platforms"
	@echo "  make clean                  - Clean build directory"
	@echo "  make fmt                    - Format Go source code"
	@echo "  make vet                    - Run Go vet check"
	@echo "  make run                    - Build and run the app locally"

# Local Build
build:
	@echo "Building for current OS..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) main.go

# Linux Build
build-linux:
	@echo "Building for Linux (amd64)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 main.go

# macOS Build (amd64)
build-darwin-amd64:
	@echo "Building for macOS (amd64)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 main.go

# macOS Build (arm64 / Apple Silicon)
build-darwin-arm64:
	@echo "Building for macOS (arm64)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 main.go

# Windows Build
build-windows:
	@echo "Building for Windows (amd64)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe main.go

# Build all platforms
build-all: build-linux build-darwin-amd64 build-darwin-arm64 build-windows

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
