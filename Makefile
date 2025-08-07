.PHONY: all build build-core install-deps dev dev-admin dev-student clean setup check-deps

all: build

# Check prerequisites
check-deps:
	@command -v pnpm >/dev/null 2>&1 || { echo "pnpm is not installed. Please install it first: npm install -g pnpm"; exit 1; }
	@command -v go >/dev/null 2>&1 || { echo "Go is not installed. Please install it first."; exit 1; }

# Install frontend dependencies
install-deps: check-deps
	cd view && pnpm install

# Setup development environment
setup: install-deps build
	@echo "Setup complete!"

# Build Go backend
build:
	cd plugins/core && go build -o ../../bin/core

# Alias for build (for package.json compatibility)
build-core: build

# Start all portals
dev:
	cd view && pnpm dev"

# Clean build artifacts
clean:
	rm -rf bin/
