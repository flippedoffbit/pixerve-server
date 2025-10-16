.PHONY: build test clean run fmt lint dev

# Build the binary
build:
	go build -o pixerve -ldflags "-s -w -X pixerve/routes.version=$(shell git describe --tags --always --dirty) -X pixerve/routes.buildTime=$(shell date -u +%Y-%m-%dT%H:%M:%SZ) -X pixerve/routes.gitCommit=$(shell git rev-parse HEAD)" .

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	go clean
	rm -f pixerve
	rm -rf tests/test_*.db

# Run the server
run:
	go run .

# Format code
fmt:
	go fmt ./...

# Run linter (requires golangci-lint to be installed)
lint:
	golangci-lint run

# Development mode with live reload (requires air)
dev:
	air

# Install development dependencies
deps:
	go mod tidy
	go mod download