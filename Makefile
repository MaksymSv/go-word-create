.PHONY: build clean test fmt lint help

# Variables
MONTH_CMD=./cmd/get-month-issues-from-jira
SPRINT_CMD=./cmd/get-sprint-issues-from-jira
OUTPUT_DIR=./bin

# Default target
help:
	@echo "Available targets:"
	@echo "  make build              - Build all binaries"
	@echo "  make build-month        - Build month issues fetcher"
	@echo "  make build-sprint       - Build sprint issues fetcher"
	@echo "  make run-month MONTH=2025.10 [LOGONLY=1] - Run collecting month issues (LOGONLY optional)"
	@echo "  make run-sprint BOARD=\"My Board\" SPRINT=\"Sprint 16\" [LOGONLY=1] - Run collecting sprint issues (LOGONLY optional)"
	@echo "  make clean              - Remove build artifacts"
	@echo "  make test               - Run tests"
	@echo "  make fmt                - Format code"
	@echo "  make lint               - Run linter"
	@echo "  make help               - Show this help message"

# Build all binaries
build: build-month build-sprint
	@echo "✓ All binaries built in $(OUTPUT_DIR)/"

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

# Run month issues fetcher
run-month: build-month
	@if [ -z "$(MONTH)" ]; then \
		echo "Error: MONTH parameter required (format: YYYY.MM)"; \
		echo "Usage: make run-month MONTH=2025.10"; \
		exit 1; \
	fi
	$(OUTPUT_DIR)/get-month-issues -month="$(MONTH)"$(if $(LOGONLY), -debug)

# Run sprint issues fetcher
run-sprint: build-sprint
	@if [ -z "$(BOARD)" ]; then \
		echo "Error: BOARD parameter required"; \
		echo "Usage: make run-sprint BOARD=\"My Board\" SPRINT=\"Sprint 16\""; \
		exit 1; \
	fi
	@if [ -z "$(SPRINT)" ]; then \
		echo "Error: SPRINT parameter required"; \
		echo "Usage: make run-sprint BOARD=\"My Board\" SPRINT=\"Sprint 16\""; \
		exit 1; \
	fi
	$(OUTPUT_DIR)/get-sprint-issues -board="$(BOARD)" -sprint="$(SPRINT)"$(if $(LOGONLY), -debug)

# Clean build artifacts
clean:
	@rm -rf $(OUTPUT_DIR)
	@go clean
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
