.PHONY: help build run fmt vet test lint clean

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | sed -E 's/Makefile://; s/:.*##/: /'

build: ## Build the blt binary
	go build -o blt ./cmd/blt

run: ## Run the app without building a binary
	go run ./cmd/blt

fmt: ## Format Go code
	go fmt ./...

vet: ## Run go vet
	go vet ./...

test: ## Run tests
	go test ./...

lint: ## Run golangci-lint if installed
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed; skipping"

clean: ## Remove built artifacts
	rm -f blt

