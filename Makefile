.PHONY: build install clean test fmt vet lint run help

BINARY_NAME := glovebox
BINARY_PATH := bin/$(BINARY_NAME)
INSTALL_PATH := /usr/local/bin/$(BINARY_NAME)

# Use mise if available, otherwise assume go is in PATH
GO := $(shell command -v mise >/dev/null 2>&1 && echo "mise exec -- go" || echo "go")

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary to bin/glovebox
	$(GO) build -o $(BINARY_PATH) .

install: build ## Install to /usr/local/bin
	cp $(BINARY_PATH) $(INSTALL_PATH)

uninstall: ## Remove from /usr/local/bin
	rm -f $(INSTALL_PATH)

clean: ## Remove build artifacts
	rm -f $(BINARY_PATH)
	$(GO) clean

test: ## Run tests
	$(GO) test ./...

fmt: ## Format code
	$(GO) fmt ./...

vet: ## Run go vet
	$(GO) vet ./...

lint: fmt vet ## Run fmt and vet

run: build ## Build and run glovebox
	./$(BINARY_PATH)

all: lint test build ## Run lint, test, and build
