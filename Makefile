.PHONY: build run clean test fmt lint help build-ui run-web

# Variables
BINARY_NAME=go-word-create
MAIN_PACKAGE=./cmd/server
MONTH_CMD=./cmd/get-month-issues-from-jira
SPRINT_CMD=./cmd/get-sprint-issues-from-jira
WEB_CMD=./cmd/web
OUTPUT_DIR=./bin
UI_DIR=./cmd/web/ui

# Default target
help:
	@echo "Available targets:"
	@echo "  make build              - Build all binaries"
	@echo "  make build-server       - Build server binary"
	@echo "  make build-month        - Build month issues fetcher"
	@echo "  make build-sprint       - Build sprint issues fetcher"
	@echo "  make build-ui           - Build React UI (requires npm)"
	@echo "  make build-web          - Build web server binary"
	@echo "  make run                - Run the server"
	@echo "  make run-web            - Build UI and run web server"
	@echo "  make run-month MONTH=2025.10 - Run month issues with date parameter"
	@echo "  make dev-ui             - Start React dev server"
	@echo "  make clean              - Remove build artifacts"
	@echo "  make test               - Run tests"
	@echo "  make fmt                - Format code"
	@echo "  make lint               - Run linter"
	@echo "  make help               - Show this help message"

# Build all binaries
build: build-server build-month build-sprint build-web
	@echo "✓ All binaries built in $(OUTPUT_DIR)/"

# Build server binary
build-server:
	@mkdir -p $(OUTPUT_DIR)
	go build -o $(OUTPUT_DIR)/server $(MAIN_PACKAGE)
	@echo "✓ Server built: $(OUTPUT_DIR)/server"

# Build month issues fetcher
build-month:
	@mkdir -p $(OUTPUT_DIR)
	go build -o $(OUTPUT_DIR)/get-month-issues $(MONTH_CMD)
	@echo "✓ Month fetcher built: $(OUTPUT_DIR)/get-month-issues"

# Build sprint issues fetcher
build-sprint:
	@mkdir -p $(OUTPUT_DIR)
	go build -o $(OUTPUT_DIR)/get-sprint-issues $(SPRINT_CMD)
	@echo "✓ Sprint fetcher built: $(OUTPUT_DIR)/get-sprint-issues"

# Build React UI
build-ui:
	@cd $(UI_DIR) && npm run build
	@echo "✓ React UI built"

# Build web static server
build-web:
	@mkdir -p $(OUTPUT_DIR)
	go build -o $(OUTPUT_DIR)/web $(WEB_CMD)
	@echo "✓ Web server built: $(OUTPUT_DIR)/web"

# Run the server
run: build-server
	$(OUTPUT_DIR)/server

# Run web server with built UI
run-web: build-ui build-web
	@cd $(OUTPUT_DIR) && ./web

# Dev mode: start React dev server (requires npm)
dev-ui:
	@cd $(UI_DIR) && npm run dev

# Run month issues fetcher
run-month: build-month
	@if [ -z "$(MONTH)" ]; then \
		echo "Error: MONTH parameter required (format: YYYY.MM)"; \
		echo "Usage: make run-month MONTH=2025.10"; \
		exit 1; \
	fi
	$(OUTPUT_DIR)/get-month-issues -month="$(MONTH)" -debug

# Clean build artifacts
clean:
	@rm -rf $(OUTPUT_DIR)
	@go clean
	@cd $(UI_DIR) && npm run clean 2>/dev/null || true
	@echo "✓ Cleaned build artifacts"

# Run tests
test:
	@go test -v ./...

# Format code
fmt:
	@go fmt ./...
	@echo "✓ Code formatted"

# Run linter
lint:
	@which golangci-lint > /dev/null || (echo "Installing golangci-lint..."; go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	@golangci-lint run ./...
