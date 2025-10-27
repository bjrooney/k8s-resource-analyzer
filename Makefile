.PHONY: build run clean install test help

# Binary name
BINARY_NAME=k8s-analyzer

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_NAME) .
	@echo "Build complete: ./$(BINARY_NAME)"

# Build for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	GOOS=linux GOARCH=amd64 go build -o $(BINARY_NAME)-linux-amd64 .
	GOOS=darwin GOARCH=amd64 go build -o $(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build -o $(BINARY_NAME)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build -o $(BINARY_NAME)-windows-amd64.exe .
	@echo "Multi-platform build complete"

# Install dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies installed"

# Run the application
run: build
	@echo "Running analysis..."
	./$(BINARY_NAME)

# Run with AI
run-ai: build
	@echo "Running analysis with AI..."
	@if [ -z "$$OPENAI_API_KEY" ] && [ -z "$$AZURE_OPENAI_API_KEY" ]; then \
		echo "Error: No API key found. Set OPENAI_API_KEY or AZURE_OPENAI_API_KEY"; \
		exit 1; \
	fi
	./$(BINARY_NAME)

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*
	rm -f cluster-analysis-report.md
	@echo "Clean complete"

# Install the binary to PATH
install: build
	@echo "Installing $(BINARY_NAME) to /usr/local/bin..."
	sudo cp $(BINARY_NAME) /usr/local/bin/
	@echo "Installation complete"

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run

# Run tests (if any)
test:
	@echo "Running tests..."
	go test -v ./...

# Show help
help:
	@echo "Kubernetes Resource Analyzer - Makefile commands:"
	@echo ""
	@echo "  make build      - Build the application"
	@echo "  make build-all  - Build for multiple platforms"
	@echo "  make deps       - Download and tidy dependencies"
	@echo "  make run        - Build and run the analyzer"
	@echo "  make run-ai     - Build and run with AI (requires API key)"
	@echo "  make clean      - Remove build artifacts and reports"
	@echo "  make install    - Install binary to /usr/local/bin"
	@echo "  make fmt        - Format Go code"
	@echo "  make lint       - Run linter (requires golangci-lint)"
	@echo "  make test       - Run tests"
	@echo "  make help       - Show this help message"
	@echo ""
	@echo "Environment variables:"
	@echo "  OPENAI_API_KEY           - OpenAI API key"
	@echo "  AZURE_OPENAI_API_KEY     - Azure OpenAI API key"
	@echo "  AZURE_OPENAI_ENDPOINT    - Azure OpenAI endpoint"
