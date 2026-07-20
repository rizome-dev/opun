# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Overview

Opun is a Go-based Terminal Development Kit (TDK) that provides a standardized framework for managing AI code agents. It orchestrates Claude Code, Gemini CLI, and Qwen Code through their headless CLI modes (with optional interactive PTY sessions), offering workflow automation, cross-provider subagent orchestration, multi-agent coordination, and extensibility through plugins and MCP servers.

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
The system drives AI providers via their headless CLI modes:
- `internal/headless/` - Headless runners (`claude -p`, `gemini -p`, `qwen -p` with `--output-format json`) and per-provider JSON result parsers
- `pkg/core/provider.go` - Core Provider interface
- `internal/providers/` - Claude, Gemini, and Qwen implementations
- Provider creation: `internal/providers/factory.go`

### Key Architecture Patterns
- **Headless Execution**: Workflow agents and subagents run through provider print modes with JSON output; no terminal scraping
- **Interactive PTY (opt-in)**: `opun chat` is a transparent PTY passthrough; workflow agents with `interactive: true` run in a visible PTY session (unix only)
- **Workflow Engine**: YAML-based sequential/parallel agent orchestration
- **Subagent System**: Cross-provider task delegation and specialization
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
  # Using subagents for specialized tasks
  - name: security
    subagent: security-auditor
    prompt: "Audit for vulnerabilities"
    depends_on: [analyze]
```

Key features:
- Agent dependencies and output chaining
- Variable substitution and file inclusion
- Subagent delegation for specialized tasks
- Cross-provider orchestration
- Retry logic with exponential backoff
- Conditional execution based on success/failure

### Subagent System

The subagent system enables cross-provider task delegation:

```go
// Core subagent interfaces in pkg/core/subagent.go
type SubAgent interface {
    GetName() string
    GetCapabilities() []string
    CanHandle(task SubAgentTask) bool
    Execute(ctx context.Context, task SubAgentTask) (*SubAgentResult, error)
}

type SubAgentManager interface {
    Register(agent SubAgent) error
    Execute(ctx context.Context, task SubAgentTask, agentName string) (*SubAgentResult, error)
    FindBestAgent(task SubAgentTask) (SubAgent, error)
}
```

**Key Components**:
- `internal/subagent/` - Subagent implementation and management
- `internal/subagent/factory.go` - Creates subagents from configurations
- `internal/subagent/manager.go` - Manages subagent registry and execution
- `internal/cli/subagent.go` - CLI commands for subagent operations

**Subagent Commands**:
```bash
opun subagent list                    # List all registered subagents
opun subagent create <config>         # Create from configuration
opun subagent delete <name>           # Remove a subagent
opun subagent execute <name> <task>   # Execute task on specific agent
opun subagent info <name>             # Show agent details
```

**Integration with Workflows**:
Subagents can be used in workflows by specifying the `subagent` field:
```yaml
agents:
  - id: reviewer
    subagent: code-reviewer  # Use named subagent
    prompt: "Review this code"
```

### Important Implementation Details

#### Provider Detection
- Providers resolve to their binary on PATH (`claude`, `gemini`, `qwen`); a `Command` override in provider config is honored
- Headless runs pipe the prompt via stdin and parse JSON output (schemas differ per provider: Claude/Qwen emit result objects/arrays, Gemini emits `{response, stats}`)
- Interactive ready detection: Claude uses the `│ >` input-box family; Gemini/Qwen match the "Type your message" placeholder after ANSI stripping

#### Execution Modes
- **Headless (default)**: `claude -p` / `gemini -p` / `qwen -p` with `--output-format json`; the executor writes agent output files itself
- **Interactive** (`interactive: true` per agent, unix only): PTY session with typed prompt injection
- Permission modes map per provider: `auto-edit` → Claude `acceptEdits` / Gemini `auto_edit` / Qwen `auto-edit`; `yolo` → Claude `bypassPermissions`

#### Error Handling
- Workflows support `continue_on_error` and `stop_on_error`
- Headless runs enforce a per-step timeout (default 10m, `settings.timeout` per agent)

### Project Structure
```
cmd/opun/          # CLI entry point (uses Charm's fang)
internal/
  cli/             # Cobra commands (add, update, delete, list, chat, run, refactor, capability, subagent, setup, mcp, action, completion)
  command/         # Command execution and registry
  headless/        # Headless provider runners and JSON result parsing
  mcp/             # MCP (Model Context Protocol) integration
  plugin/          # Plugin loading and management
  promptgarden/    # Prompt storage and templating
  providers/       # AI provider implementations (claude, gemini, qwen)
  subagent/        # Subagent implementation and management
  workflow/        # YAML-based workflow engine
  utils/           # Utilities (clipboard, opun dir resolution)
pkg/               # Public interfaces
  command/         # Command types
  core/            # Core interfaces (Provider, Tool, MCP, Prompt, SubAgent)
  plugin/          # Plugin types
  subagent/        # Subagent types and interfaces
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
├── subagents/          # Subagent configurations
├── actions/            # Action definitions (simple command wrappers)
├── plugins/           # Installed plugins
├── mcp/              # MCP server installations
└── sessions/         # Session data
```

### Key Implementation Notes
- **NEVER create duplicate files with suffixes like `_enhanced`, `_new`, `_v2`** - Always modify original files
- Workflow agents and subagents execute through `internal/headless` (provider print modes); PTY is reserved for `opun chat` and opt-in interactive agents
- `OPUN_HOME` overrides the data directory (default `~/.opun`); all path resolution flows through `utils.OpunDir()`
- MCP server config is synced to each CLI's real config file: `~/.claude.json`, `~/.gemini/settings.json`, `~/.qwen/settings.json` (merged, preserving unmanaged entries); MCP servers run on demand via `npx -y`
- Provider configurations support MCP servers and tools
- Subagents enable cross-provider orchestration with capability-based routing (`core.MatchesCapabilities`)
- The opun MCP stdio server exposes `subagent_execute`/`subagent_list` tools for cross-provider delegation

## Subagent Development

### Creating a Subagent

Subagents are created from configuration files or programmatically:

```go
// Using the factory
import "github.com/rizome-dev/opun/internal/subagent"

config := core.SubAgentConfig{
    Name:         "my-agent",
    Provider:     core.ProviderTypeClaude,
    Model:        "sonnet",
    Capabilities: []string{"code-review", "security"},
    SystemPrompt: "You are an expert reviewer...",
}

agent, err := subagent.CreateSubAgent(config)
```

### Registering with Manager

```go
manager := subagent.NewManager()
err := manager.Register(agent)
```

### Executing Tasks

```go
task := core.SubAgentTask{
    Name:        "Review Code",
    Description: "Review the authentication module",
    Context:     map[string]interface{}{"file": "auth.go"},
    Requirements: []string{"code-review", "security"},
}

result, err := manager.Execute(ctx, task, "my-agent")
// Or let the manager find the best agent
result, err = manager.ExecuteBest(ctx, task)
```

### Workflow Integration

In workflows, subagents are referenced by name:

```yaml
agents:
  - id: step1
    subagent: my-agent
    prompt: "Perform the task"
```

The workflow engine automatically:
1. Looks up the subagent by name
2. Creates a provider session if needed
3. Executes the task with the subagent's configuration
4. Captures output for downstream agents

### Testing Subagents

```go
// Create a mock subagent for testing
mockAgent := &MockSubAgent{
    name: "test-agent",
    capabilities: []string{"test"},
}

// Test capability matching
assert.True(t, mockAgent.CanHandle(task))

// Test execution
result, err := mockAgent.Execute(ctx, task)
assert.NoError(t, err)
assert.NotNil(t, result)
```