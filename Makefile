.PHONY: dev build clean deps generate run

# Development with hot reload
dev:
	@echo "Starting development server with hot reload..."
	@air

# Generate Templ files
generate:
	@templ generate

# Build production binary
build: generate
	@echo "Building production binary..."
	@CGO_ENABLED=0 go build -ldflags="-s -w" -o bin/server ./cmd/server
	@echo "Binary created: bin/server"
	@ls -lh bin/server

# Run production binary
run: build
	@./bin/server

# Install dependencies (one-time)
deps:
	@echo "Installing Go tools..."
	@go install github.com/a-h/templ/cmd/templ@latest
	@go install github.com/air-verse/air@latest
	@echo "Installing Go module dependencies..."
	@go mod tidy
	@echo "Done! Run 'make dev' to start development."

# Clean build artifacts
clean:
	@rm -rf bin/ tmp/
	@find . -name '*_templ.go' -delete
	@echo "Cleaned build artifacts."
