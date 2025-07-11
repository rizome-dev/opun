# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Opun is a Go-based Terminal Development Kit (TDK) that provides a standardized framework for managing AI code agents. It orchestrates Claude Code and Gemini CLI through PTY automation, offering workflow automation, multi-agent coordination, and extensibility through plugins and MCP servers.

## Essential Commands

### Development
```bash
make build              # Build the binary
make install           # Build and install to /usr/local/bin  
make dev               # Quick development rebuild (no optimization)
make run ARGS="--help" # Run the application with arguments
```

### Testing
```bash
make test              # Run unit tests
make test-coverage     # Run tests with coverage report
make test-integration  # Run integration tests  
make test-e2e         # Run end-to-end tests
make test-all         # Run all tests
make test-short        # Run unit tests in short mode
make benchmark         # Run benchmarks

# Run a single test
go test -v -run TestName ./path/to/package
```

### Code Quality
```bash
make fmt              # Format code
make vet              # Vet code
make lint             # Run golangci-lint
make check            # Run all checks (fmt, vet, lint, test)
```

### Additional Commands
```bash
make deps              # Download dependencies
make setup             # Install development tools (golangci-lint, gosec, goreleaser)
make generate          # Run go generate
make clean             # Remove build artifacts
make providers-check   # Check if AI providers are installed
make fix-permissions   # Fix ownership of ~/.opun directory
```

## Architecture Overview

### Provider Abstraction
The system uses PTY automation to control AI providers:
- `pkg/core/provider.go` - Core Provider interface
- `internal/providers/` - Claude and Gemini implementations
- Provider creation: `internal/providers/factory.go`
- PTY management: `internal/pty/` (session control and automation)

### Key Architecture Patterns
- **PTY Automation**: Terminal session management for provider interaction
- **Session Isolation**: Each workflow gets its own session directory
- **Workflow Engine**: YAML-based sequential/parallel agent orchestration
- **Plugin System**: Extensible via Go plugins, scripts, or JSON definitions
- **PromptGarden**: Centralized prompt management with templating
- **Event-Driven**: Components communicate via channels and callbacks

### Workflow System
Workflows are YAML files that orchestrate multiple agents:
```yaml
agents:
  - name: analyze
    prompt: "Analyze this code"
    provider: claude
    model: sonnet
  - name: refactor
    prompt: "Refactor based on {{analyze.output}}"
    depends_on: [analyze]
```

Key features:
- Agent dependencies and output chaining
- Variable substitution and file inclusion
- Retry logic with exponential backoff
- Conditional execution based on success/failure

### Important Implementation Details

#### Provider Detection
- Claude: Tries `claude` command, falls back to `npx claude-code`
- Gemini: Direct command execution
- Ready patterns are provider-specific (e.g., Claude: "Type /help")

#### Prompt Injection Methods
- **Clipboard**: Used by Claude (simulates Cmd+V)
- **File**: Write to temp file and reference
- **Stdin**: Direct pipe to provider

#### Error Handling
- `InteractiveModeRequiredError` - Triggers interactive mode for auth
- Workflows support `continue_on_error` and `stop_on_error`
- PTY sessions handle provider crashes gracefully

### Project Structure
```
cmd/opun/          # CLI entry point (uses Charm's fang)
internal/
  cli/             # Cobra commands (add, chat, list, prompt, refactor, run, setup)
  command/         # Command execution and registry
  io/              # Session management (managed and transparent)
  mcp/             # MCP (Model Context Protocol) integration  
  plugin/          # Plugin loading and management
  promptgarden/    # Prompt storage and templating
  providers/       # AI provider implementations
  pty/             # PTY automation and provider-specific handling
  workflow/        # YAML-based workflow engine
  utils/           # Utilities (clipboard operations)
pkg/               # Public interfaces
  command/         # Command types
  core/            # Core interfaces (Provider, Tool, MCP, Prompt)
  plugin/          # Plugin types
  workflow/        # Workflow types
```

### Testing Approach
- Unit tests alongside code files (`*_test.go`)
- Integration tests in `test/integration/`
- E2E tests in `test/e2e/`
- Mock providers available for testing
- Use testify for assertions
- Test fixtures in `internal/workflow/test-prompts/`

### Configuration and Storage
```
~/.opun/
├── config.yaml         # Main configuration
├── promptgarden/       # Prompt storage
├── workflows/          # Workflow definitions
├── actions/            # Action definitions (simple command wrappers)
├── plugins/           # Installed plugins
├── mcp/              # MCP server installations
└── sessions/         # Session data
```

### Key Implementation Notes
- **NEVER create duplicate files with suffixes like `_enhanced`, `_new`, `_v2`** - Always modify original files
- The architecture is moving from API-based to PTY-based automation
- Each workflow execution creates an isolated session
- PTY sessions are reused within a workflow for efficiency
- Clipboard integration is critical for Claude prompt injection
- Provider configurations support MCP servers and tools