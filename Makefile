.PHONY: dev build clean deps generate run install deploy redeploy stop status logs

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

# --- Deployment (smdctl) ---

# Install/deploy service for the first time
install: build
	@echo "Installing scratchpad service..."
	@smdctl run -f smdctl.yml
	@echo "Service installed. Check status with: make status"

# Deploy (alias for install)
deploy: install

# Redeploy service (stop, rebuild, start)
redeploy: build
	@echo "Redeploying scratchpad service..."
	@smdctl stop scratchpad 2>/dev/null || true
	@smdctl rm scratchpad 2>/dev/null || true
	@smdctl run -f smdctl.yml
	@echo "Service redeployed. Check status with: make status"

# Stop service
stop:
	@echo "Stopping scratchpad service..."
	@smdctl stop scratchpad

# Start service (if already installed)
start:
	@echo "Starting scratchpad service..."
	@smdctl start scratchpad

# Restart service
restart:
	@echo "Restarting scratchpad service..."
	@smdctl restart scratchpad

# Check service status
status:
	@smdctl status scratchpad

# View service logs
logs:
	@smdctl logs -f scratchpad

# Remove service completely
uninstall:
	@echo "Removing scratchpad service..."
	@smdctl stop scratchpad 2>/dev/null || true
	@smdctl rm scratchpad
	@echo "Service removed."
