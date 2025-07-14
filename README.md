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
# If a fresh installation (configures default provider, default MCP servers, etc)
opun setup

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

Opun's configuration system is designed to be modular, declarative, and human-readable. Each type of configuration serves a specific purpose in the ecosystem, allowing you to build complex automation workflows while maintaining simplicity and reusability.

### Configuration Overview

All configuration files are stored in `~/.opun/` and organized by type:

```
~/.opun/
├── config.yaml            # Main configuration
├── workflows/            # Multi-agent workflow definitions
├── promptgarden/         # Reusable prompt templates
├── actions/              # Simple command shortcuts
├── tools/                # Provider-specific tools
├── mcp/                  # MCP server configurations
│   ├── servers/         # MCP server definitions
│   └── tools/           # MCP tool definitions
└── plugins/              # Installed plugins
```

### Workflows (`~/.opun/workflows/*.yaml`)

**Purpose**: Workflows are the heart of Opun's multi-agent orchestration. They allow you to chain multiple AI agents together, passing context between them to accomplish complex tasks that would be difficult for a single agent.

**Key Concepts**:
- **Sequential Execution**: Agents run one after another, with each agent able to access outputs from previous agents
- **Dependency Management**: Agents can depend on the success of previous agents using `depends_on`
- **Conditional Execution**: Use JavaScript-like expressions in `condition` to control when agents run
- **Context Passing**: Agents automatically save their outputs to files that subsequent agents can read using the `@` syntax
- **Variable Substitution**: Use `{{variable}}` syntax to inject workflow variables, agent outputs, or file contents

**Structure**:

```yaml
# Workflow Metadata
name: code-review-workflow          # Unique identifier for the workflow
command: review                     # Slash command to trigger: /review file.go
description: Comprehensive code review with multiple perspectives
version: 1.0.0                      # Semantic versioning for tracking changes
author: "Your Name"                 # Optional: workflow author

# Workflow Variables - Define inputs that users can provide
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
    enum: ["low", "medium", "high", "critical"]  # Restrict to specific values

# Global Workflow Settings - Apply to all agents unless overridden
settings:
  output_dir: "./review-outputs/{{timestamp}}"
  log_level: "info"
  stop_on_error: false
  timeout: 300          # Global timeout in seconds for entire workflow

# Agent Definitions - The core of your workflow
agents:
  # First agent: Initial code analysis
  - id: analyzer                    # Unique ID for referencing this agent
    name: "Code Analyzer"           # Human-readable name displayed during execution
    provider: claude                # AI provider: claude or gemini
    model: sonnet                   # Model variant (provider-specific)
    
    # The prompt is the instruction sent to the AI agent
    # Supports variable substitution and file references
    prompt: |
      Analyze the code in {{file_path}} and create a structured report covering:
      1. Code structure and organization
      2. Potential issues and code smells
      3. Complexity metrics
      Focus areas: {{focus_areas}}
      
      IMPORTANT: Save your complete analysis to the output file.
      
    # Output file where the agent should save results (relative to output_dir)
    # This file can be referenced by subsequent agents using {{analyzer.output}}
    output: analysis-report.md
    settings:
      timeout: 60
      retry_count: 2
      temperature: 0.2
      quality_mode: deep-think
      
  # Second agent: Security review (demonstrates context passing)
  - id: security-reviewer
    name: "Security Auditor"
    provider: gemini
    model: pro
    
    # Dependencies control execution order
    depends_on: ["analyzer"]        # Only run after analyzer completes
    
    # Conditional execution using JavaScript-like expressions
    # Access agent success status and workflow variables
    condition: "analyzer.success && '{{focus_areas}}'.includes('security')"
    
    prompt: |
      # Reading output from previous agent
      # The {{analyzer.output}} reference is automatically converted to @filepath
      Based on the initial analysis from the previous agent:
      {{analyzer.output}}
      
      Perform a security audit of {{file_path}}:
      - Identify vulnerabilities (OWASP Top 10)
      - Check for hardcoded secrets
      - Review authentication/authorization
      - Assess input validation
      
      Save your findings to the output file.
      
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
    
    # Can depend on multiple agents - runs after all complete
    depends_on: ["security-reviewer", "performance-reviewer"]
    
    # Accessing outputs from multiple previous agents
    # Each reference is converted to @/path/to/output/file.md
    prompt: |
      Consolidate all reviews into a final report.
      
      Read and synthesize the following analyses:
      - Initial Analysis: {{analyzer.output}}
      - Security Review: {{security-reviewer.output}}
      - Performance Review: {{performance-reviewer.output}}
      
      Create a prioritized action list with severity levels.
      Only include items with severity >= {{severity_threshold}}.
      
      Save the consolidated report to the output file.
      
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

**How Context Passing Works**:

1. **Output Files**: Each agent saves its results to a file specified in the `output` field
2. **Automatic References**: Use `{{agent-id.output}}` in prompts to reference previous outputs
3. **File Translation**: References are automatically converted to `@filepath` syntax that AI providers understand
4. **Timestamped Directories**: All outputs are saved in timestamped directories to prevent conflicts

**Running Workflows**:

```bash
# Using the slash command
opun chat
> /review path/to/file.go

# Direct execution with variables
opun run review --file_path=main.go --focus_areas=security,performance

# Interactive mode with variable prompts
opun run review
# You'll be prompted for any required variables
```

**Best Practices**:

- **Modular Design**: Keep each agent focused on a specific task
- **Clear Dependencies**: Use `depends_on` to ensure proper execution order
- **Conditional Logic**: Use `condition` to skip unnecessary agents
- **Error Handling**: Set `continue_on_error: true` for non-critical agents
- **Output Instructions**: Always remind agents to save their output to files

### Remote Manifests

**Purpose**: Remote manifests allow you to share and distribute collections of Opun configurations. Think of them as "packages" that bundle related prompts, workflows, actions, and tools together.

**Use Cases**:
- Share team-specific workflows and standards
- Distribute domain-specific toolkits (e.g., security auditing, performance testing)
- Create reusable configuration libraries
- Maintain consistency across projects

**Structure**:

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

**Installing from Manifests**:

```bash
# Interactive installation
opun add
# Select: Remote → Any type → Enter manifest URL

# Direct installation
opun add --url https://example.com/my-toolkit.yaml

# The manifest will be downloaded and all items installed to appropriate directories
```

**Creating Manifests**:

1. Bundle related configurations together
2. Host on any accessible URL (GitHub, GitLab, S3, etc.)
3. Use semantic versioning for updates
4. Include clear descriptions and documentation

### MCP Tools (`~/.opun/mcp/tools/*.yaml`)

**Purpose**: MCP (Model Context Protocol) tools extend AI agents with specific capabilities like web search, database queries, or API interactions. These tools are available to agents during execution.

**Key Concepts**:
- **Schema-Driven**: Define input and output schemas for type safety
- **Provider Integration**: Tools are exposed to AI providers through MCP servers
- **Composability**: Tools can be combined in workflows for complex operations

**Structure**:

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

**Usage in Workflows**:

```yaml
agents:
  - id: researcher
    provider: claude
    settings:
      mcp_servers: ["web-tools"]     # Enable MCP server with web-search tool
      tools: ["web-search"]           # Specific tool to make available
    prompt: |
      Research the latest security vulnerabilities for Node.js.
      Use web search to find recent CVEs and patches.
```

### Tools (`~/.opun/tools/*.yaml`)

**Purpose**: Tools are provider-specific shortcuts that make common operations available to AI agents. Unlike MCP tools, these are simpler and can directly execute commands, reference workflows, or use prompt templates.

**Types of Tools**:

1. **Command Tools**: Execute shell commands directly
2. **Workflow Tools**: Trigger existing workflows
3. **Prompt Tools**: Use prompt templates with specific context

**Structure Examples**:

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

**Using Tools**:

```bash
# Tools are automatically available to agents based on provider
# Configure in workflow agent settings:
settings:
  tools: ["search-code", "analyze-security", "explain-error"]
```

### Prompt Garden Templates (`~/.opun/promptgarden/*.md`)

**Purpose**: The Prompt Garden is your centralized repository of reusable prompt templates. It promotes consistency, best practices, and knowledge sharing across your team.

**Key Features**:
- **Metadata-Driven**: Each prompt includes metadata for organization and discovery
- **Variable Substitution**: Use `{{variable}}` placeholders for dynamic content
- **Conditional Logic**: Use `{{#if}}` blocks for conditional prompt sections
- **Version Control**: Track prompt evolution with semantic versioning
- **Categorization**: Organize prompts by category and tags

**Structure**:

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

**Using Prompt Garden Templates**:

```bash
# List available prompts
opun list prompts

# Use in chat with variables
opun prompt code-explanation --code_snippet="function fibonacci(n) { ... }"

# Reference in workflows
agents:
  - id: explainer
    prompt: "@code-explanation"    # Reference by prompt name
    variables:
      code_snippet: "{{file_content}}"
      audience_level: "beginner"
```

**Best Practices**:
- **Consistent Structure**: Use similar formats across related prompts
- **Clear Variables**: Document all variables with descriptions and defaults
- **Version Control**: Update version numbers when making significant changes
- **Effective Categorization**: Use categories and tags for easy discovery

### Environment Variables

**Purpose**: Environment variables provide a secure way to manage sensitive data and environment-specific configurations without hardcoding them in your configuration files.

**Use Cases**:
- API keys and tokens for MCP servers
- Project-specific paths and settings
- Team notifications and integrations
- Environment-specific configurations (dev/staging/prod)

**Setting Variables**:

```bash
# For MCP servers (common examples)
export OPENROUTER_API_KEY="your-api-key"
export UPSTASH_REDIS_REST_URL="your-redis-url"
export UPSTASH_REDIS_REST_TOKEN="your-redis-token"
export POSTGRES_CONNECTION_STRING="postgresql://..."

# Custom variables for workflows
export PROJECT_ROOT="/path/to/project"
export REVIEW_TEAM_SLACK="@code-review-team"
export CODE_STANDARDS_URL="https://company.com/standards"
export MAX_FILE_SIZE="1000000"

# Provider-specific settings
export CLAUDE_MODEL="sonnet"
export GEMINI_TEMPERATURE="0.7"
```

**Using in Configurations**:

```yaml
# In workflows
agents:
  - id: analyzer
    prompt: |
      Analyze code in {{env.PROJECT_ROOT}}
      Follow standards at {{env.CODE_STANDARDS_URL}}
      
# In MCP server configs
mcp_servers:
  - name: database
    env:
      CONNECTION_STRING: "${POSTGRES_CONNECTION_STRING}"
      
# In tool definitions
tools:
  - id: notify-team
    command: "slack-cli send --channel {{env.REVIEW_TEAM_SLACK}}"
```

**Best Practices**:
- **Security**: Never commit sensitive environment variables to version control
- **Documentation**: Maintain a `.env.example` file with dummy values
- **Validation**: Check for required variables at workflow start
- **Defaults**: Provide sensible defaults where appropriate
- **Naming**: Use UPPERCASE_WITH_UNDERSCORES for consistency

### Configuration Summary

Opun's configuration system is designed to scale from simple single-agent tasks to complex multi-agent workflows:

1. **Start Simple**: Begin with basic prompts in the Prompt Garden
2. **Build Workflows**: Combine prompts into multi-agent workflows
3. **Add Tools**: Extend capabilities with MCP and provider-specific tools
4. **Share Knowledge**: Create manifests to distribute your configurations
5. **Iterate and Improve**: Use version control and metadata to evolve your toolkit

The modular design ensures that each component can be developed, tested, and shared independently while working together seamlessly in production workflows.

