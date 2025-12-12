.PHONY: build install clean test fmt vet lint run help release version

BINARY_NAME := glovebox
BINARY_PATH := bin/$(BINARY_NAME)
INSTALL_DIR := /usr/local/bin
INSTALL_PATH := $(INSTALL_DIR)/$(BINARY_NAME)
SYMLINK_PATH := $(INSTALL_DIR)/gb

# Use mise if available, otherwise assume go is in PATH
GO := $(shell command -v mise >/dev/null 2>&1 && echo "mise exec -- go" || echo "go")

# Version from git tags
VERSION := $(shell git describe --tags --always --dirty)
LDFLAGS := -ldflags "-X github.com/joelhelbling/glovebox/cmd.Version=$(VERSION)"

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary to bin/glovebox
	$(GO) build $(LDFLAGS) -o $(BINARY_PATH) .

version: ## Show current version
	@echo $(VERSION)

install: build ## Install to /usr/local/bin
	cp $(BINARY_PATH) $(INSTALL_PATH)
	ln -sf $(BINARY_NAME) $(SYMLINK_PATH)

uninstall: ## Remove from /usr/local/bin
	rm -f $(INSTALL_PATH) $(SYMLINK_PATH)

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

# Release target - usage: make release V=v0.3.0
release: ## Create a release (usage: make release V=v0.3.0)
ifndef V
	$(error Version required. Usage: make release V=v0.3.0)
endif
	@if [ -n "$$(git status --porcelain)" ]; then \
		echo "Error: Working directory not clean. Commit changes first."; \
		exit 1; \
	fi
	git tag -a $(V) -m "Release $(V)"
	$(MAKE) build
	@echo "Built $(BINARY_PATH) with version $(V)"
	@echo "To push: git push origin main && git push origin $(V)"
