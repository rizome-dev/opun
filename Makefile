# Project variables
PROJECT_NAME := opun
BINARY_NAME := opun
GO_FILES := $(shell find . -name '*.go' -type f -not -path "./vendor/*" -not -path "./_archive/*")
MAIN_PACKAGE := ./cmd/opun

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_TIME := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X 'main.version=$(VERSION)' -X 'main.commit=$(COMMIT)' -X 'main.buildTime=$(BUILD_TIME)'"

# Go commands
GOCMD := go
GOBUILD := $(GOCMD) build
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod
GOFMT := $(GOCMD) fmt
GOVET := $(GOCMD) vet

# Build output
BUILD_DIR := build
COVERAGE_FILE := coverage.out

# Colors for terminal output
COLOR_RESET := \033[0m
COLOR_BOLD := \033[1m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE := \033[34m

# Default target
.DEFAULT_GOAL := help

.PHONY: all
all: clean lint test build ## Run all main targets (clean, lint, test, build)

.PHONY: help
help: ## Display this help message
	@echo "$(COLOR_BOLD)$(PROJECT_NAME) Makefile$(COLOR_RESET)"
	@echo "$(COLOR_BOLD)Usage:$(COLOR_RESET) make $(COLOR_GREEN)[target]$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Targets:$(COLOR_RESET)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(COLOR_GREEN)%-20s$(COLOR_RESET) %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build the binary
	@echo "$(COLOR_BLUE)Building $(BINARY_NAME)...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "$(COLOR_GREEN)✓ Binary built: $(BUILD_DIR)/$(BINARY_NAME)$(COLOR_RESET)"

.PHONY: build-cross
build-cross: ## Build binaries for multiple platforms
	@echo "$(COLOR_BLUE)Building cross-platform binaries...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	@GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	@GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PACKAGE)
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)
	@echo "$(COLOR_GREEN)✓ Cross-platform binaries built$(COLOR_RESET)"

.PHONY: install
install: build ## Build and install the binary to /usr/local/bin
	@echo "$(COLOR_BLUE)Installing $(BINARY_NAME) to /usr/local/bin...$(COLOR_RESET)"
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@# Clear ALL extended attributes and quarantine flags
	@sudo xattr -d com.apple.quarantine /usr/local/bin/$(BINARY_NAME) 2>/dev/null || true
	@sudo xattr -d com.apple.provenance /usr/local/bin/$(BINARY_NAME) 2>/dev/null || true
	@sudo xattr -cr /usr/local/bin/$(BINARY_NAME) 2>/dev/null || true
	@# Re-sign the binary
	@sudo codesign --force --deep --sign - /usr/local/bin/$(BINARY_NAME) 2>/dev/null || true
	@echo "$(COLOR_GREEN)✓ Installed to /usr/local/bin/$(BINARY_NAME)$(COLOR_RESET)"
	@$(MAKE) post-install

.PHONY: dev
dev: ## Quick development build (no optimization)
	@CGO_ENABLED=0 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "$(COLOR_GREEN)✓ Development build complete$(COLOR_RESET)"

.PHONY: run
run: ## Build and run the application
	@CGO_ENABLED=0 $(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@./$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

.PHONY: clean
clean: ## Remove build artifacts
	@echo "$(COLOR_YELLOW)Cleaning...$(COLOR_RESET)"
	@rm -rf $(BUILD_DIR)
	@rm -f $(COVERAGE_FILE)
	@rm -f *.test
	@rm -f *.out
	@echo "$(COLOR_GREEN)✓ Clean complete$(COLOR_RESET)"

.PHONY: test
test: ## Run unit tests
	@echo "$(COLOR_BLUE)Running tests...$(COLOR_RESET)"
	@$(GOTEST) -race -v ./...
	@echo "$(COLOR_GREEN)✓ Tests passed$(COLOR_RESET)"

.PHONY: test-short
test-short: ## Run unit tests (short mode)
	@echo "$(COLOR_BLUE)Running short tests...$(COLOR_RESET)"
	@$(GOTEST) -short -v ./...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "$(COLOR_BLUE)Running tests with coverage...$(COLOR_RESET)"
	@$(GOTEST) -race -coverprofile=$(COVERAGE_FILE) -covermode=atomic ./...
	@$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o coverage.html
	@echo "$(COLOR_GREEN)✓ Coverage report generated: coverage.html$(COLOR_RESET)"

.PHONY: test-integration
test-integration: ## Run integration tests
	@echo "$(COLOR_BLUE)Running integration tests...$(COLOR_RESET)"
	@$(GOTEST) -race -tags=integration -v ./test/integration/...

.PHONY: test-e2e
test-e2e: build ## Run end-to-end tests
	@echo "$(COLOR_BLUE)Running e2e tests...$(COLOR_RESET)"
	@$(GOTEST) -race -tags=e2e -v ./test/e2e/...

.PHONY: test-all
test-all: test test-integration test-e2e ## Run all tests

.PHONY: benchmark
benchmark: ## Run benchmarks
	@echo "$(COLOR_BLUE)Running benchmarks...$(COLOR_RESET)"
	@$(GOTEST) -bench=. -benchmem ./...

.PHONY: fmt
fmt: ## Format Go code
	@echo "$(COLOR_BLUE)Formatting code...$(COLOR_RESET)"
	@$(GOFMT) ./...
	@echo "$(COLOR_GREEN)✓ Code formatted$(COLOR_RESET)"

.PHONY: vet
vet: ## Run go vet
	@echo "$(COLOR_BLUE)Running go vet...$(COLOR_RESET)"
	@$(GOVET) ./...
	@echo "$(COLOR_GREEN)✓ Vet passed$(COLOR_RESET)"

.PHONY: lint
lint: ## Run golangci-lint
	@echo "$(COLOR_BLUE)Running linters...$(COLOR_RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
		echo "$(COLOR_GREEN)✓ Linting passed$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)⚠ golangci-lint not installed. Run 'make setup' to install.$(COLOR_RESET)"; \
	fi

.PHONY: check
check: fmt vet lint test ## Run all checks (fmt, vet, lint, test)
	@echo "$(COLOR_GREEN)✓ All checks passed$(COLOR_RESET)"

.PHONY: deps
deps: ## Download dependencies
	@echo "$(COLOR_BLUE)Downloading dependencies...$(COLOR_RESET)"
	@$(GOMOD) download
	@echo "$(COLOR_GREEN)✓ Dependencies downloaded$(COLOR_RESET)"

.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "$(COLOR_BLUE)Updating dependencies...$(COLOR_RESET)"
	@$(GOGET) -u ./...
	@$(GOMOD) tidy
	@echo "$(COLOR_GREEN)✓ Dependencies updated$(COLOR_RESET)"

.PHONY: vendor
vendor: ## Create vendor directory
	@echo "$(COLOR_BLUE)Vendoring dependencies...$(COLOR_RESET)"
	@$(GOMOD) vendor
	@echo "$(COLOR_GREEN)✓ Dependencies vendored$(COLOR_RESET)"

.PHONY: setup
setup: ## Install development tools
	@echo "$(COLOR_BLUE)Installing development tools...$(COLOR_RESET)"
	@$(GOCMD) install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@$(GOCMD) install github.com/securego/gosec/v2/cmd/gosec@latest
	@$(GOCMD) install github.com/goreleaser/goreleaser@latest
	@echo "$(COLOR_GREEN)✓ Development tools installed$(COLOR_RESET)"

.PHONY: generate
generate: ## Run go generate
	@echo "$(COLOR_BLUE)Running go generate...$(COLOR_RESET)"
	@$(GOCMD) generate ./...
	@echo "$(COLOR_GREEN)✓ Generation complete$(COLOR_RESET)"

.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "$(COLOR_BLUE)Building Docker image...$(COLOR_RESET)"
	@docker build -t $(PROJECT_NAME):$(VERSION) .
	@echo "$(COLOR_GREEN)✓ Docker image built: $(PROJECT_NAME):$(VERSION)$(COLOR_RESET)"

.PHONY: docker-run
docker-run: docker-build ## Build and run Docker container
	@docker run --rm -it $(PROJECT_NAME):$(VERSION)

.PHONY: release-dry-run
release-dry-run: ## Perform a dry run of goreleaser
	@echo "$(COLOR_BLUE)Running release dry run...$(COLOR_RESET)"
	@goreleaser release --snapshot --skip-publish --rm-dist

.PHONY: release
release: ## Create a new release (requires tag)
	@echo "$(COLOR_BLUE)Creating release...$(COLOR_RESET)"
	@goreleaser release --rm-dist

# Opun-specific targets
.PHONY: refactor-sessions-clean
refactor-sessions-clean: ## Clean up refactor sessions
	@echo "$(COLOR_YELLOW)Cleaning refactor sessions...$(COLOR_RESET)"
	@rm -rf refactor-sessions/
	@echo "$(COLOR_GREEN)✓ Refactor sessions cleaned$(COLOR_RESET)"

.PHONY: fix-permissions
fix-permissions: ## Fix ownership of ~/.opun directory
	@echo "$(COLOR_BLUE)Fixing ~/.opun directory ownership...$(COLOR_RESET)"
	@if [ -d ~/.opun ]; then \
		sudo chown -R $$(whoami) ~/.opun; \
		echo "$(COLOR_GREEN)✓ Fixed ownership of ~/.opun$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)~/.opun directory not found$(COLOR_RESET)"; \
	fi

.PHONY: init-config
init-config: ## Initialize default config file if it doesn't exist
	@echo "$(COLOR_BLUE)Initializing default config...$(COLOR_RESET)"
	@ACTUAL_USER=$$(if [ -n "$$SUDO_USER" ]; then echo "$$SUDO_USER"; else whoami; fi); \
	ACTUAL_HOME=$$(if [ -n "$$SUDO_USER" ]; then eval echo ~$$SUDO_USER; else echo ~; fi); \
	mkdir -p "$$ACTUAL_HOME/.opun" && \
	if [ -n "$$SUDO_USER" ]; then \
		chown "$$ACTUAL_USER:$$(id -gn $$ACTUAL_USER)" "$$ACTUAL_HOME/.opun"; \
	fi && \
	if [ ! -f "$$ACTUAL_HOME/.opun/config.yaml" ]; then \
		echo 'agent:\n  provider: claude\n  model: sonnet\nquality_mode: standard\nworkflows: []\npromptgarden:\n  prompts: []' > "$$ACTUAL_HOME/.opun/config.yaml" && \
		if [ -n "$$SUDO_USER" ]; then \
			chown "$$ACTUAL_USER:$$(id -gn $$ACTUAL_USER)" "$$ACTUAL_HOME/.opun/config.yaml"; \
		fi && \
		echo "$(COLOR_GREEN)✓ Created default config at ~/.opun/config.yaml$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)Config already exists at ~/.opun/config.yaml$(COLOR_RESET)"; \
	fi

.PHONY: post-install
post-install: ## Post-installation setup (run as user, not sudo)
	@echo ""
	@# Detect the actual user even when running with sudo
	@ACTUAL_USER=$$(if [ -n "$$SUDO_USER" ]; then echo "$$SUDO_USER"; else whoami; fi); \
	ACTUAL_HOME=$$(if [ -n "$$SUDO_USER" ]; then eval echo ~$$SUDO_USER; else echo ~; fi); \
	echo "$(COLOR_BLUE)Setting up Opun for user $$ACTUAL_USER...$(COLOR_RESET)"; \
	mkdir -p "$$ACTUAL_HOME/.opun/workflows" "$$ACTUAL_HOME/.opun/promptgarden" "$$ACTUAL_HOME/.opun/sessions" "$$ACTUAL_HOME/.opun/mcp" "$$ACTUAL_HOME/.opun/workspace" && \
	if [ -n "$$SUDO_USER" ]; then \
		chown -R "$$ACTUAL_USER:$$(id -gn $$ACTUAL_USER)" "$$ACTUAL_HOME/.opun"; \
	fi && \
	echo "$(COLOR_GREEN)✓ Created ~/.opun directory structure with correct ownership$(COLOR_RESET)"
	@echo ""
	@# Fix npm cache ownership if needed
	@ACTUAL_USER=$$(if [ -n "$$SUDO_USER" ]; then echo "$$SUDO_USER"; else whoami; fi); \
	ACTUAL_HOME=$$(if [ -n "$$SUDO_USER" ]; then eval echo ~$$SUDO_USER; else echo ~; fi); \
	if [ -d "$$ACTUAL_HOME/.npm" ] && [ -n "$$(find "$$ACTUAL_HOME/.npm" -user root 2>/dev/null | head -1)" ]; then \
		echo "$(COLOR_YELLOW)Fixing npm cache ownership...$(COLOR_RESET)"; \
		chown -R "$$ACTUAL_USER:$$(id -gn $$ACTUAL_USER)" "$$ACTUAL_HOME/.npm" && echo "$(COLOR_GREEN)✓ Fixed npm cache ownership$(COLOR_RESET)" || echo "$(COLOR_YELLOW)⚠️  Failed to fix npm cache ownership$(COLOR_RESET)"; \
	fi
	@echo ""
	@echo "$(COLOR_GREEN)✨ Opun installation complete!$(COLOR_RESET)"
	@echo ""
	@# Check if opun is already configured
	@if [ -f ~/.opun/config.yaml ] && \
		grep -q "default_provider:" ~/.opun/config.yaml 2>/dev/null && \
		[ -d ~/.opun/workflows ] && \
		[ -d ~/.opun/promptgarden ] && \
		[ -d ~/.opun/sessions ] && \
		[ -d ~/.opun/mcp ]; then \
		echo "$(COLOR_YELLOW)📋 Existing Opun configuration detected$(COLOR_RESET)"; \
		echo ""; \
		read -p "Do you want to run the setup wizard again? (y/N) " -n 1 -r; \
		echo ""; \
		if [[ ! $$REPLY =~ ^[Yy]$$ ]]; then \
			echo "$(COLOR_GREEN)✓ Skipping setup - Opun is ready to use!$(COLOR_RESET)"; \
			echo "$(COLOR_BLUE)Run 'opun chat' to start chatting or 'opun setup' to reconfigure$(COLOR_RESET)"; \
		else \
			echo "$(COLOR_BLUE)Starting interactive setup...$(COLOR_RESET)"; \
			opun setup || echo "$(COLOR_YELLOW)⚠ Setup can be run later with 'opun setup'$(COLOR_RESET)"; \
		fi \
	else \
		echo "$(COLOR_BLUE)Starting interactive setup...$(COLOR_RESET)"; \
		opun setup || echo "$(COLOR_YELLOW)⚠ Setup can be run later with 'opun setup'$(COLOR_RESET)"; \
	fi

.PHONY: install-claude
install-claude: ## Install Claude CLI (if not installed)
	@if ! command -v claude >/dev/null 2>&1; then \
		echo "$(COLOR_BLUE)Installing Claude CLI...$(COLOR_RESET)"; \
		npm install -g @anthropic-ai/claude-cli; \
	else \
		echo "$(COLOR_GREEN)✓ Claude CLI already installed$(COLOR_RESET)"; \
	fi

.PHONY: install-gemini
install-gemini: ## Install Gemini CLI (if not installed)
	@if ! command -v gemini >/dev/null 2>&1; then \
		echo "$(COLOR_YELLOW)⚠ Please install Gemini CLI manually from https://github.com/google/generative-ai-cli$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_GREEN)✓ Gemini CLI already installed$(COLOR_RESET)"; \
	fi

.PHONY: providers-check
providers-check: ## Check if AI providers are installed
	@echo "$(COLOR_BLUE)Checking AI providers...$(COLOR_RESET)"
	@if command -v claude >/dev/null 2>&1; then \
		echo "$(COLOR_GREEN)✓ Claude CLI installed$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)✗ Claude CLI not found$(COLOR_RESET)"; \
	fi
	@if command -v gemini >/dev/null 2>&1; then \
		echo "$(COLOR_GREEN)✓ Gemini CLI installed$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)✗ Gemini CLI not found$(COLOR_RESET)"; \
	fi

.PHONY: check-setup
check-setup: ## Check if Opun is properly configured
	@echo "$(COLOR_BLUE)Checking Opun configuration...$(COLOR_RESET)"
	@echo ""
	@# Check directory structure
	@echo "$(COLOR_BOLD)Directory Structure:$(COLOR_RESET)"
	@if [ -d ~/.opun ]; then \
		echo "$(COLOR_GREEN)✓ ~/.opun directory exists$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)✗ ~/.opun directory not found$(COLOR_RESET)"; \
	fi
	@if [ -d ~/.opun/workflows ]; then \
		echo "$(COLOR_GREEN)✓ ~/.opun/workflows directory exists$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)✗ ~/.opun/workflows directory not found$(COLOR_RESET)"; \
	fi
	@if [ -d ~/.opun/promptgarden ]; then \
		echo "$(COLOR_GREEN)✓ ~/.opun/promptgarden directory exists$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)✗ ~/.opun/promptgarden directory not found$(COLOR_RESET)"; \
	fi
	@if [ -d ~/.opun/sessions ]; then \
		echo "$(COLOR_GREEN)✓ ~/.opun/sessions directory exists$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)✗ ~/.opun/sessions directory not found$(COLOR_RESET)"; \
	fi
	@if [ -d ~/.opun/mcp ]; then \
		echo "$(COLOR_GREEN)✓ ~/.opun/mcp directory exists$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)✗ ~/.opun/mcp directory not found$(COLOR_RESET)"; \
	fi
	@echo ""
	@# Check configuration file
	@echo "$(COLOR_BOLD)Configuration:$(COLOR_RESET)"
	@if [ -f ~/.opun/config.yaml ]; then \
		echo "$(COLOR_GREEN)✓ Config file exists$(COLOR_RESET)"; \
		if grep -q "default_provider:" ~/.opun/config.yaml 2>/dev/null; then \
			provider=$$(grep "default_provider:" ~/.opun/config.yaml | awk '{print $$2}'); \
			echo "$(COLOR_GREEN)✓ Default provider configured: $$provider$(COLOR_RESET)"; \
		else \
			echo "$(COLOR_YELLOW)✗ No default provider configured$(COLOR_RESET)"; \
		fi; \
	else \
		echo "$(COLOR_YELLOW)✗ Config file not found$(COLOR_RESET)"; \
	fi
	@echo ""
	@# Check providers
	@echo "$(COLOR_BOLD)AI Providers:$(COLOR_RESET)"
	@if command -v claude >/dev/null 2>&1 || command -v npx >/dev/null 2>&1; then \
		echo "$(COLOR_GREEN)✓ Claude available$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)✗ Claude not available$(COLOR_RESET)"; \
	fi
	@if command -v gemini >/dev/null 2>&1; then \
		echo "$(COLOR_GREEN)✓ Gemini available$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)✗ Gemini not available$(COLOR_RESET)"; \
	fi
	@echo ""
	@# Summary
	@if [ -f ~/.opun/config.yaml ] && \
		grep -q "default_provider:" ~/.opun/config.yaml 2>/dev/null && \
		[ -d ~/.opun/workflows ] && \
		[ -d ~/.opun/promptgarden ] && \
		[ -d ~/.opun/sessions ] && \
		[ -d ~/.opun/mcp ]; then \
		echo "$(COLOR_GREEN)✨ Opun is properly configured and ready to use!$(COLOR_RESET)"; \
		echo "$(COLOR_BLUE)Run 'opun chat' to start chatting$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)⚠ Opun setup is incomplete$(COLOR_RESET)"; \
		echo "$(COLOR_BLUE)Run 'opun setup' to complete configuration$(COLOR_RESET)"; \
	fi
