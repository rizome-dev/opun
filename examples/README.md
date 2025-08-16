# Opun Examples

This directory contains simple, clear examples of each Opun component type.

## Structure

```
examples/
├── prompt/          # Prompt templates for AI agents
├── workflow/        # Multi-agent orchestration workflows
├── subagents/       # Cross-provider subagent configurations
├── action/          # Commands that AI agents can execute
├── tool/            # MCP tools for Opun's MCP server
└── manifest/        # Example manifest for remote distribution
```

## Core Components

### 1. Prompt Example (`prompt/`)

**File**: `code-review.md` - Template for comprehensive code reviews

This example shows:
- Prompt metadata (variables, tags)
- Conditional templating
- Variable substitution
- Structured output format

To add locally:
```bash
opun add
# Choose: Local → Prompt → Select file
```

### 2. Workflow Example (`workflow/`)

**File**: `test-and-fix.yaml` - Runs tests, analyzes failures, suggests fixes

This example demonstrates:
- Sequential agent execution
- Output chaining between agents
- Conditional execution
- Mixed provider usage (Claude + Gemini)
- Variable substitution

To add locally:
```bash
opun add
# Choose: Local → Workflow → Select file
```

### 3. Action Example (`action/`)

**File**: `find-todos.yaml` - Searches for TODO comments in code

This example shows:
- Simple command wrapping (uses `ripgrep`)
- Provider compatibility
- Basic configuration

To add locally:
```bash
opun add
# Choose: Local → Action → Select file
```

### 4. Subagent Example (`subagents/`)

**Files**: 
- `claude-reviewer.yaml` - Claude-based code review specialist
- `gemini-analyzer.yaml` - Gemini-based architecture analyzer
- `qwen-refactorer.yaml` - Qwen-based code refactoring expert
- `cross-provider-workflow.yaml` - Workflow using multiple subagents

These examples demonstrate:
- Provider-specific configurations
- Capability definitions
- System prompts for specialization
- Cross-provider orchestration in workflows

To create a subagent:
```bash
opun subagent create examples/subagents/claude-reviewer.yaml
```

To use in a workflow:
```bash
opun run examples/subagents/cross-provider-workflow.yaml
```

### 5. Tool Example (`tool/`)

**File**: `calculator.yaml` - Basic arithmetic calculations for MCP

This example demonstrates:
- Input/output schema definitions
- Tool metadata
- MCP-compatible structure

To add locally:
```bash
opun add
# Choose: Local → Tool → Select file
```

## Remote Installation

### Remote Manifest (`manifest/`)

**File**: `example-manifest.yaml` - Manifest for bulk importing configurations

This shows how to package multiple items for remote distribution:
- Multiple prompts
- Workflows
- Actions

To install from URL:
```bash
opun add
# Choose: Remote → Any type → Enter URL
```

Example URL format:
```
https://raw.githubusercontent.com/user/repo/main/manifest.yaml
```

## Usage Examples

### Adding Items Locally

```bash
# Interactive mode
opun add
# Then choose: Local → [Type] → Select file

# Or use command line flags (legacy)
opun add --workflow --path examples/workflow/test-and-fix.yaml --name my-workflow
opun add --prompt --path examples/prompt/code-review.md --name review
opun add --action --path examples/action/find-todos.yaml --name todos
```

### Adding Items from URL

```bash
# Interactive mode
opun add
# Then choose: Remote → [Type] → Enter URL

# Direct URL installation
opun add
# Remote → Any → https://example.com/manifest.yaml
```

### Using Added Items

**Prompts:**
```bash
opun prompt code-review --file-path main.go
```

**Workflows:**
```bash
opun run test-and-fix
```

**Subagents:**
```bash
# List available subagents
opun subagent list

# Execute task with specific subagent
opun subagent execute claude-reviewer "Review this code: func main() {...}"

# Get subagent information
opun subagent info claude-reviewer
```

**Actions (in chat):**
```bash
opun chat
/find-todos
```

**Tools (via MCP):**
Tools are automatically available to AI agents through the MCP protocol.

## Key Concepts

### Component Types

- **Prompts**: Reusable templates with variables and logic
- **Workflows**: Multi-agent orchestration with dependencies
- **Subagents**: Specialized AI agents for cross-provider delegation
- **Actions**: Simple commands that agents can execute
- **Tools**: MCP-compatible tools for advanced agent capabilities

### Local vs Remote

- **Local**: Add files from your filesystem
- **Remote**: Install from URLs (manifest files)

### Storage Locations

```
~/.opun/
├── promptgarden/    # Prompts
├── workflows/       # Workflows
├── subagents/       # Subagent configurations
├── actions/         # Actions
├── mcp/
│   └── tools/      # MCP Tools
└── installs/       # Remote installation records
```

## Creating Your Own

1. **Start with examples**: Copy and modify the examples
2. **Test locally**: Add and test your configurations
3. **Package for sharing**: Create a manifest file
4. **Share via URL**: Host on GitHub or any web server

## Manifest Format

For remote distribution, create a manifest file:

```yaml
name: my-toolkit
version: 1.0.0
description: My custom Opun toolkit
author: Your Name

imports:
  prompts:
    - name: my-prompt
      description: "Does something useful"
      content: |
        Your prompt template here
      tags: ["custom", "toolkit"]
  
  workflows:
    - name: my-workflow
      description: "Multi-step process"
      agents:
        - name: step1
          prompt: "First step"
          provider: claude
        - name: step2
          subagent: code-reviewer  # Using a subagent
          prompt: "Review the output"
  
  subagents:
    - name: my-specialist
      provider: gemini
      model: gemini-pro
      capabilities:
        - analysis
        - research
      description: "Specialized research agent"
      system_prompt: |
        You are a research specialist...
  
  actions:
    - name: my-action
      description: "Runs a command"
      type: script
      script: |
        #!/bin/bash
        echo "Hello from action"
```