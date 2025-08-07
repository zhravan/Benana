.PHONY: all build install-deps dev clean

all: build

# Build Go backend
build:
	cd plugins/core && go build -o ../../bin/core

install-deps: check-deps
	@echo "Installing frontend dependencies..."
	cd view && pnpm install

# Start all portals (requires terminal multiplexer or separate terminals)
dev:
	@echo "Starting portals"
	@echo "Run 'make dev-admin' in one terminal and 'make dev-student' in another"
	@echo "Or use: cd view && pnpm dev"

# Clean build artifacts
clean:
	rm -rf bin/
