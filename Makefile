.PHONY: build clean install uninstall test run help

# Binary name
BINARY_NAME=docgen

# Build flags for smaller binaries
LDFLAGS=-ldflags="-s -w"

# Default target
all: build

# Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	go build $(LDFLAGS) -o $(BINARY_NAME) .
	@echo "Built $(BINARY_NAME) successfully!"
	@ls -lh $(BINARY_NAME)

# Build and run
run: build
	./$(BINARY_NAME)

# Install to /usr/local/bin (requires sudo on most systems)
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BINARY_NAME) /usr/local/bin/
	@echo "$(BINARY_NAME) installed successfully!"

# Uninstall from /usr/local/bin
uninstall:
	@echo "Removing $(BINARY_NAME) from /usr/local/bin..."
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "$(BINARY_NAME) uninstalled successfully!"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*
	@echo "Clean complete!"

# Cross-compile for multiple platforms
cross-compile:
	@echo "Cross-compiling for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe .
	@echo "Cross-compilation complete!"
	@ls -lh $(BINARY_NAME)-*

# Create and push a new annotated tag, then run GoReleaser locally
release:
    @echo "Enter version (e.g., v0.1.1):"; \
    read VERSION; \
    git tag -a $$VERSION -m "Release $$VERSION"; \
    git push origin $$VERSION; \
    @echo "Running GoReleaser for $$VERSION..."; \
    GORELEASER_CURRENT_TAG=$$VERSION goreleaser release --clean

release-dry:
    @echo "Running GoReleaser dry run..."
    goreleaser release --clean --skip-publish --snapshot

# Run tests (if you add tests later)
test:
	go test -v ./...

# Show help
help:
	@echo "Available targets:"
	@echo "  make build          - Build the binary (default)"
	@echo "  make run            - Build and run the program"
	@echo "  make install        - Install to /usr/local/bin (requires sudo)"
	@echo "  make uninstall      - Remove from /usr/local/bin (requires sudo)"
	@echo "  make clean          - Remove build artifacts"
	@echo "  make cross-compile  - Build for Linux, macOS, and Windows"
	@echo "  make test           - Run tests"
	@echo "  make help           - Show this help message"
	@echo "  make release        - Tag and release with GoReleaser"
	@echo "  make release-dry    - Dry run GoReleaser without publishing"
