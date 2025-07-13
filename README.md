# Opun - TDK (Terminal Development Kit)

At Rizome Labs, we are using CLI Code Agents (TDEs) a LOT. Therefore, we wanted a way to standardize their configuration & behavior across environments, as well as standardize a system for multi-agent & multi-provider TDE workflows (inspired by [willer/claude-fsd](https://github.com/willer/claude-fsd) & [sdi2200262/agentic-project-management](https://github.com/sdi2200262/agentic-project-management)).

Thus, Opun was born.

Opun is a Terminal Development Kit - a wrapper for Terminal Development Environments (TDEs), providing Sequential Workflow Automation, a standardized configuration system for Prompts, Workflows, Actions & Tools, remote distribution via manifests, MCP Client & Server, Slash Commands, a Prompt Garden, and much more.

built by: [rizome labs](https://rizome.dev)

reach out: [hi (at) rizome.dev](mailto://hi@rizome.dev)

**If you want an Agentic Swarm setup on-prem, reach out!**

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap rizome-dev/brews && brew install opun
```

### From Source (Windows)

```bash
git clone https://github.com/rizome-dev/opun && cd opun && sudo make install
```

### Binary Download

Download the latest binary for your platform from the [releases page](https://github.com/rizome-dev/opun/releases).

## Quick Start

```bash

# Initialize a chat session with the default provider -- or, specify the provider (chat {gemini,claude})
opun chat

# Add a workflow, tool or prompt -- this is interactive, no need for flags (--{prompt,workflow} --path --name)
opun add

# Run a workflow -- this is interactive, no need for options (run <NAME.md>)
opun run

# Manipulate the registry
opun {update,delete}
```

## Configuration

Opun uses a modular configuration system with different config files for various components. All configuration files are stored in `~/.opun/`.

### Workflows (`~/.opun/workflows/*.yaml`)

Workflows orchestrate multiple AI agents in sequence:

```yaml
name: code-review-workflow
command: review         # Slash command to trigger this workflow
description: Comprehensive code review with multiple perspectives
version: 1.0.0
author: "Your Name"

# Workflow variables
variables:
  - name: file_path
    description: "Path to the file to review"
    type: string
    required: true
    
  - name: focus_areas
    description: "Specific areas to focus on"
    type: string
    required: false
    default: "security,performance,maintainability"
    
  - name: severity_threshold
    description: "Minimum severity level to report"
    type: string
    required: false
    default: "medium"
    enum: ["low", "medium", "high", "critical"]

# Global workflow settings
settings:
  output_dir: "./review-outputs/{{timestamp}}"
  log_level: "info"
  stop_on_error: false
  timeout: 300          # Global timeout in seconds

# Agent definitions
agents:
  # First agent: Initial code analysis
  - id: analyzer
    name: "Code Analyzer"
    provider: claude
    model: sonnet
    prompt: |
      Analyze the code in {{file_path}} and create a structured report covering:
      1. Code structure and organization
      2. Potential issues and code smells
      3. Complexity metrics
      Focus areas: {{focus_areas}}
    output: analysis-report.md
    settings:
      timeout: 60
      retry_count: 2
      temperature: 0.2
      quality_mode: deep-think
      
  # Second agent: Security review (depends on analyzer)
  - id: security-reviewer
    name: "Security Auditor"
    provider: gemini
    model: pro
    depends_on: ["analyzer"]
    condition: "analyzer.success && '{{focus_areas}}'.includes('security')"
    prompt: |
      Based on the initial analysis:
      {{file:./review-outputs/{{timestamp}}/analysis-report.md}}
      
      Perform a security audit of {{file_path}}:
      - Identify vulnerabilities (OWASP Top 10)
      - Check for hardcoded secrets
      - Review authentication/authorization
      - Assess input validation
    output: security-audit.md
    settings:
      timeout: 90
      mcp_servers: ["sequential-thinking"]
      tools: ["search-code", "analyze-security"]
      
  # Third agent: Performance review (parallel with security)
  - id: performance-reviewer
    name: "Performance Analyzer"
    provider: claude
    model: sonnet
    depends_on: ["analyzer"]
    condition: "'{{focus_areas}}'.includes('performance')"
    prompt: |
      Review {{file_path}} for performance:
      - Algorithm complexity
      - Database query optimization
      - Memory usage patterns
      - Caching opportunities
    output: performance-review.md
    settings:
      timeout: 60
      temperature: 0.3
      continue_on_error: true
    on_failure:
      - type: log
        message: "Performance review failed, continuing with other reviews"
        
  # Final agent: Consolidate all reviews
  - id: consolidator
    name: "Review Consolidator"
    provider: claude
    model: opus
    depends_on: ["security-reviewer", "performance-reviewer"]
    prompt: |
      Consolidate all reviews into a final report:
      
      Initial Analysis: {{analyzer.output}}
      Security Review: {{security-reviewer.output}}
      Performance Review: {{performance-reviewer.output}}
      
      Create a prioritized action list with severity levels.
      Only include items with severity >= {{severity_threshold}}.
    output: final-review.md
    settings:
      quality_mode: deep-think
      wait_for_file: "./review-outputs/{{timestamp}}/performance-review.md"
      interactive: true    # Allow user interaction during consolidation
    on_success:
      - type: log
        message: "Code review completed successfully!"
      - type: execute
        data:
          command: "open ./review-outputs/{{timestamp}}/final-review.md"

# Workflow metadata
metadata:
  tags: ["code-review", "quality", "automation"]
  estimated_duration: "5-10 minutes"
  requirements:
    - "File must exist at specified path"
    - "Appropriate read permissions"
```

### Remote Manifests

Opun supports installing collections of prompts, workflows, actions, and tools from remote URLs:

```yaml
# Example manifest file (hosted at https://example.com/my-toolkit.yaml)
name: my-toolkit
version: 1.0.0
description: A collection of useful Opun configurations
author: "Your Name"
repository: https://github.com/example/my-toolkit

imports:
  prompts:
    - name: code-review
      description: "Comprehensive code review prompt"
      content: |
        Perform a thorough code review focusing on:
        - Code quality and best practices
        - Potential bugs and edge cases
        - Performance considerations
        - Security vulnerabilities
        
        File: {{file_path}}
        Focus areas: {{focus_areas | default: "all"}}
      tags: ["review", "quality"]
    
    - name: refactor-suggestions
      description: "Suggest refactoring improvements"
      content: |
        Analyze the code and suggest refactoring improvements.
        Consider design patterns, code duplication, and complexity.
      tags: ["refactoring", "improvement"]

  workflows:
    - name: full-analysis
      description: "Complete code analysis workflow"
      agents:
        - name: reviewer
          prompt: "@code-review"
          provider: claude
          model: sonnet
        - name: refactorer
          prompt: "@refactor-suggestions"
          provider: claude
          model: sonnet
          depends_on: [reviewer]

  actions:
    - name: format-all
      description: "Format all code files in directory"
      type: script
      script: |
        #!/bin/bash
        find . -name "*.go" -exec gofmt -w {} \;
        find . -name "*.js" -exec prettier --write {} \;
        find . -name "*.py" -exec black {} \;
```

To install from a manifest:

```bash
# Interactive mode
opun add
# Choose: Remote → Any type → Enter URL

# The manifest will be downloaded and all items installed
```

### MCP Tools (`~/.opun/mcp/tools/*.yaml`)

Tools extend the capabilities of AI agents through the MCP protocol:

```yaml
name: web-search
description: Search the web for information

input_schema:
  type: object
  properties:
    query:
      type: string
      description: Search query
    max_results:
      type: integer
      description: Maximum number of results
      default: 5
  required: ["query"]

output_schema:
  type: object
  properties:
    results:
      type: array
      items:
        type: object
        properties:
          title:
            type: string
          url:
            type: string
          snippet:
            type: string
```

### Tools (`~/.opun/tools/*.yaml`)

Tools provide quick access to common commands:

```yaml
# Search code tool
id: search-code
name: Search Code
description: Search for patterns in code files using ripgrep
category: development

# Direct command execution
command: "rg --type-add 'code:*.{js,ts,go,py,java,rs,cpp,c,h}' -t code"

# Optional: Limit to specific providers
providers:
  - claude
  - gemini

---

# Workflow-based tool
id: analyze-security
name: Security Analysis
description: Run security analysis workflow
category: security

# Reference to a workflow
workflow_ref: "security-audit-workflow"

# Tool-specific configuration
config:
  depth: "comprehensive"
  report_format: "markdown"

---

# Prompt-based tool
id: explain-error
name: Explain Error
description: Explain error messages and suggest fixes
category: debugging

# Reference to a prompt template
prompt_ref: "explain-error-template"

# Additional context to include
context:
  include_stack_trace: true
  include_recent_changes: true
```

### Prompt Garden Templates (`~/.opun/promptgarden/*.md`)

Prompt templates with metadata and variable substitution:

```markdown
---
name: code-explanation
description: Explain complex code with examples
category: documentation
tags: 
  - explanation
  - documentation
  - teaching
version: 1.0.0
variables:
  - name: code_snippet
    description: The code to explain
    required: true
  - name: audience_level
    description: Target audience expertise level
    required: false
    default: "intermediate"
  - name: focus_areas
    description: Specific aspects to focus on
    required: false
---

# Code Explanation Template

Please explain the following code for a {{audience_level}} developer:

```
{{code_snippet}}
```

## Requirements:
1. Start with a high-level overview
2. Explain the purpose and main functionality
3. Break down complex parts step-by-step
4. Use analogies where helpful
5. {{#if focus_areas}}Focus especially on: {{focus_areas}}{{/if}}
6. Provide a practical usage example
7. Mention any potential gotchas or edge cases

## Format:
- Use clear headings
- Include code comments where helpful
- Provide a summary at the end
- Keep explanations concise but thorough
```

### Environment Variables

Opun supports environment variable substitution in configurations:

```bash
# Set environment variables for MCP servers
export OPENROUTER_API_KEY="your-api-key"
export UPSTASH_REDIS_REST_URL="your-redis-url"
export UPSTASH_REDIS_REST_TOKEN="your-redis-token"

# Custom variables for workflows
export PROJECT_ROOT="/path/to/project"
export REVIEW_TEAM_SLACK="@code-review-team"
```

