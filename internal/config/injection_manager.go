package config

// Copyright (C) 2025 Rizome Labs, Inc.
//
// This program is free software; you can redistribute it and/or
// modify it under the terms of the GNU General Public License
// as published by the Free Software Foundation; either version 2
// of the License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program; if not, write to the Free Software
// Foundation, Inc., 51 Franklin Street, Fifth Floor, Boston, MA  02110-1301, USA.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/rizome-dev/opun/internal/utils"
	"github.com/rizome-dev/opun/pkg/core"
)

// InjectionManager handles dynamic configuration injection for providers
type InjectionManager struct {
	sharedManager  *SharedConfigManager
	workspaceDir   string // Temporary workspace for provider configs
	actionRegistry core.ActionRegistry
}

// NewInjectionManager creates a new configuration injection manager
func NewInjectionManager(actionRegistry core.ActionRegistry) (*InjectionManager, error) {
	sharedManager, err := NewSharedConfigManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create shared config manager: %w", err)
	}

	// Create workspace directory for temporary configs
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	workspaceDir := filepath.Join(homeDir, ".opun", "workspace")
	if err := utils.EnsureDir(workspaceDir); err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}

	return &InjectionManager{
		sharedManager:  sharedManager,
		workspaceDir:   workspaceDir,
		actionRegistry: actionRegistry,
	}, nil
}

// PrepareProviderEnvironment prepares the environment for a provider launch
func (m *InjectionManager) PrepareProviderEnvironment(provider string) (*ProviderEnvironment, error) {
	// First, ensure prompt commands are up to date
	if err := UpdatePromptCommands(); err != nil {
		// Log but don't fail - prompts are nice to have
		fmt.Printf("Warning: failed to update prompt commands: %v\n", err)
	}

	// Get current working directory
	currentDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %w", err)
	}

	env := &ProviderEnvironment{
		Provider:    provider,
		Environment: make(map[string]string),
		WorkingDir:  currentDir, // Use actual working directory, not workspace
	}

	switch strings.ToLower(provider) {
	case "claude":
		if err := m.prepareClaudeEnvironment(env); err != nil {
			return nil, fmt.Errorf("failed to prepare Claude environment: %w", err)
		}
	case "gemini":
		if err := m.prepareGeminiEnvironment(env); err != nil {
			return nil, fmt.Errorf("failed to prepare Gemini environment: %w", err)
		}
	case "qwen":
		if err := m.prepareQwenEnvironment(env); err != nil {
			return nil, fmt.Errorf("failed to prepare Qwen environment: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	// Always sync MCP configuration
	if err := m.sharedManager.SyncToProvider(provider); err != nil {
		return nil, fmt.Errorf("failed to sync MCP config: %w", err)
	}

	return env, nil
}

// prepareClaudeEnvironment prepares Claude-specific environment
func (m *InjectionManager) prepareClaudeEnvironment(env *ProviderEnvironment) error {
	// Create .claude directory structure in current working directory
	currentDir := env.WorkingDir
	claudeDir := filepath.Join(currentDir, ".claude")
	commandsDir := filepath.Join(claudeDir, "commands")
	if err := utils.EnsureDir(commandsDir); err != nil {
		return err
	}

	// Generate slash commands from shared config
	if err := m.generateClaudeSlashCommands(commandsDir); err != nil {
		return err
	}

	// Generate prompt commands
	if err := m.generateClaudePromptCommands(commandsDir); err != nil {
		return err
	}

	// Generate action commands from action registry
	if err := m.generateClaudeActionCommands(commandsDir); err != nil {
		return err
	}

	// Create project-level MCP configuration in current directory
	mcpConfigPath := filepath.Join(currentDir, ".mcp.json")
	if err := m.generateProjectMCPConfig(mcpConfigPath); err != nil {
		return err
	}

	// Set environment to use current directory as project
	env.Environment["CLAUDE_PROJECT_DIR"] = currentDir

	return nil
}

// prepareGeminiEnvironment prepares Gemini-specific environment
func (m *InjectionManager) prepareGeminiEnvironment(env *ProviderEnvironment) error {
	// Since Gemini doesn't support custom slash commands,
	// we rely entirely on MCP servers for extensions

	// Create GEMINI.md for system prompt customization in workspace
	geminiMdPath := filepath.Join(m.workspaceDir, "GEMINI.md")
	if err := m.generateGeminiSystemPrompt(geminiMdPath); err != nil {
		return err
	}

	// Ensure MCP servers include our slash command server
	// This is already handled by SyncToProvider

	return nil
}

// prepareQwenEnvironment prepares Qwen-specific environment
func (m *InjectionManager) prepareQwenEnvironment(env *ProviderEnvironment) error {
	// Since Qwen is a fork of Gemini, it has similar limitations
	// We rely entirely on MCP servers for extensions

	// Create QWEN.md for system prompt customization in workspace
	qwenMdPath := filepath.Join(m.workspaceDir, "QWEN.md")
	if err := m.generateQwenSystemPrompt(qwenMdPath); err != nil {
		return err
	}

	// Ensure MCP servers include our slash command server
	// This is already handled by SyncToProvider

	return nil
}

// generateClaudeSlashCommands generates markdown files for Claude slash commands
func (m *InjectionManager) generateClaudeSlashCommands(commandsDir string) error {
	commands := m.sharedManager.GetSlashCommands()

	for _, cmd := range commands {
		// Skip hidden commands
		if cmd.Hidden {
			continue
		}

		// Create command file
		filename := fmt.Sprintf("%s.md", cmd.Name)
		cmdFilePath := filepath.Join(commandsDir, filename)

		content := m.generateCommandMarkdown(cmd)
		if err := utils.WriteFile(cmdFilePath, []byte(content)); err != nil {
			return fmt.Errorf("failed to write command %s: %w", cmd.Name, err)
		}

		// Also create files for aliases
		for _, alias := range cmd.Aliases {
			aliasFile := filepath.Join(commandsDir, fmt.Sprintf("%s.md", alias))
			if err := utils.WriteFile(aliasFile, []byte(content)); err != nil {
				return fmt.Errorf("failed to write alias %s: %w", alias, err)
			}
		}
	}

	return nil
}

// generateClaudePromptCommands generates commands for prompts
func (m *InjectionManager) generateClaudePromptCommands(commandsDir string) error {
	// Use the prompt command generator to create proper prompt files
	generator, err := NewPromptCommandGenerator()
	if err != nil {
		return fmt.Errorf("failed to create prompt generator: %w", err)
	}

	return generator.GenerateClaudePromptFiles(commandsDir)
}

// generateClaudeActionCommands generates commands from the action registry
func (m *InjectionManager) generateClaudeActionCommands(commandsDir string) error {
	if m.actionRegistry == nil {
		// No action registry, skip
		return nil
	}

	// Get actions for Claude provider
	actions := m.actionRegistry.List("claude")

	for _, action := range actions {
		// Skip if action has the same name as existing command
		// to avoid conflicts

		// Create markdown file for the action
		filename := fmt.Sprintf("%s.md", action.ID)
		cmdFilePath := filepath.Join(commandsDir, filename)

		content := m.generateActionMarkdown(action)
		if err := utils.WriteFile(cmdFilePath, []byte(content)); err != nil {
			return fmt.Errorf("failed to write action %s: %w", action.ID, err)
		}
	}

	return nil
}

// generateCommandMarkdown generates markdown content for a command
func (m *InjectionManager) generateCommandMarkdown(cmd core.SharedSlashCommand) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", cmd.Description))
	sb.WriteString(fmt.Sprintf("%s\n\n", cmd.Description))

	switch cmd.Type {
	case "workflow":
		sb.WriteString("Execute the following Opun workflow:\n")
		sb.WriteString(fmt.Sprintf("- Workflow: %s\n", cmd.Handler))
		sb.WriteString("- Arguments: $ARGUMENTS\n\n")
		sb.WriteString("Use the opun MCP server to execute this workflow.\n")

	case "prompt":
		sb.WriteString("Execute the following prompt:\n")
		sb.WriteString(fmt.Sprintf("- Prompt: %s\n", cmd.Handler))
		sb.WriteString("- Arguments: $ARGUMENTS\n\n")
		sb.WriteString("Use the opun MCP server to fetch and execute this prompt.\n")

	case "builtin":
		sb.WriteString("Execute the following built-in command:\n")
		sb.WriteString(fmt.Sprintf("- Command: %s\n", cmd.Handler))
		sb.WriteString("- Arguments: $ARGUMENTS\n\n")

	default:
		sb.WriteString("Execute this custom command with arguments: $ARGUMENTS\n")
	}

	return sb.String()
}

// generateActionMarkdown generates markdown content for a standardized action
func (m *InjectionManager) generateActionMarkdown(action core.StandardAction) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("# %s\n\n", action.Name))
	sb.WriteString(fmt.Sprintf("%s\n\n", action.Description))

	if action.Command != "" {
		sb.WriteString("## Command\n\n")
		sb.WriteString("```bash\n")
		sb.WriteString(fmt.Sprintf("%s $ARGUMENTS\n", action.Command))
		sb.WriteString("```\n\n")
		sb.WriteString("Execute this system command with the provided arguments.\n")
	} else if action.WorkflowRef != "" {
		sb.WriteString("## Workflow\n\n")
		sb.WriteString(fmt.Sprintf("Execute the Opun workflow: `%s`\n\n", action.WorkflowRef))
		sb.WriteString("Use the opun MCP server to execute this workflow with arguments: $ARGUMENTS\n")
	} else if action.PromptRef != "" {
		sb.WriteString("## Prompt\n\n")
		sb.WriteString(fmt.Sprintf("Execute the Opun prompt: `%s`\n\n", action.PromptRef))
		sb.WriteString("Use the opun MCP server to execute this prompt with arguments: $ARGUMENTS\n")
	}

	if action.Category != "" {
		sb.WriteString(fmt.Sprintf("\n## Category\n\nCategory: %s\n", action.Category))
	}

	return sb.String()
}

// generateProjectMCPConfig generates project-level MCP configuration
func (m *InjectionManager) generateProjectMCPConfig(configPath string) error {
	// Get MCP servers from shared config
	servers := m.sharedManager.GetMCPServers()

	// Create project MCP config
	config := make(map[string]interface{})
	mcpServers := make(map[string]interface{})

	for _, server := range servers {
		if !server.Installed && !server.Required {
			continue
		}

		serverConfig := map[string]interface{}{
			"type":    "stdio",
			"command": server.Command,
			"args":    server.Args,
		}

		if len(server.Env) > 0 {
			serverConfig["env"] = server.Env
		}

		mcpServers[server.Name] = serverConfig
	}

	config["mcpServers"] = mcpServers

	// Marshal and write
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return utils.WriteFile(configPath, data)
}

// generateGeminiSystemPrompt generates GEMINI.md for system customization
func (m *InjectionManager) generateGeminiSystemPrompt(mdPath string) error {
	tmpl := `# GEMINI.md

This file provides system-level guidance for Gemini CLI when working in this Opun session.

## Available Commands via MCP

Since Gemini doesn't support custom slash commands natively, use the MCP tools to access Opun functionality:

### Workflows
{{range .Commands}}{{if eq .Type "workflow"}}
- **{{.Name}}**: {{.Description}}
  - Handler: {{.Handler}}
{{end}}{{end}}

### Prompts
{{range .Commands}}{{if eq .Type "prompt"}}
- **{{.Name}}**: {{.Description}}
  - Handler: {{.Handler}}
{{end}}{{end}}

### Built-in Commands
{{range .Commands}}{{if eq .Type "builtin"}}
- **{{.Name}}**: {{.Description}}
{{end}}{{end}}

## Using Commands

To execute any of these commands, use the MCP tools:
1. List available tools with the MCP server
2. Execute the desired command through the opun tool
3. For prompts, use the opun tool

## Session Configuration

This is a managed Opun session with the following MCP servers available:
{{range .Servers}}{{if .Installed}}
- **{{.Name}}**: {{.Package}}
{{end}}{{end}}
`

	t, err := template.New("gemini").Parse(tmpl)
	if err != nil {
		return err
	}

	data := struct {
		Commands []core.SharedSlashCommand
		Servers  []core.SharedMCPServer
	}{
		Commands: m.sharedManager.GetSlashCommands(),
		Servers:  m.sharedManager.GetMCPServers(),
	}

	file, err := os.Create(mdPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return t.Execute(file, data)
}

// generateQwenSystemPrompt generates QWEN.md for system customization
func (m *InjectionManager) generateQwenSystemPrompt(mdPath string) error {
	tmpl := `# QWEN.md

This file provides system-level guidance for Qwen Code CLI when working in this Opun session.

## Available Commands via MCP

Since Qwen doesn't support custom slash commands natively, use the MCP tools to access Opun functionality:

### Workflows
{{range .Commands}}{{if eq .Type "workflow"}}
- **{{.Name}}**: {{.Description}}
  - Handler: {{.Handler}}
{{end}}{{end}}

### Prompts
{{range .Commands}}{{if eq .Type "prompt"}}
- **{{.Name}}**: {{.Description}}
  - Handler: {{.Handler}}
{{end}}{{end}}

### Built-in Commands
{{range .Commands}}{{if eq .Type "builtin"}}
- **{{.Name}}**: {{.Description}}
{{end}}{{end}}

## Using Commands

To execute any of these commands, use the MCP tools:
1. List available tools with the MCP server
2. Execute the desired command through the opun tool
3. For prompts, use the opun tool

## Session Configuration

This is a managed Opun session with the following MCP servers available:
{{range .Servers}}{{if .Installed}}
- **{{.Name}}**: {{.Package}}
{{end}}{{end}}
`

	t, err := template.New("qwen").Parse(tmpl)
	if err != nil {
		return err
	}

	data := struct {
		Commands []core.SharedSlashCommand
		Servers  []core.SharedMCPServer
	}{
		Commands: m.sharedManager.GetSlashCommands(),
		Servers:  m.sharedManager.GetMCPServers(),
	}

	file, err := os.Create(mdPath)
	if err != nil {
		return err
	}
	defer file.Close()

	return t.Execute(file, data)
}

// ProviderEnvironment contains the prepared environment for a provider
type ProviderEnvironment struct {
	Provider    string
	Environment map[string]string
	WorkingDir  string
	ConfigFiles []string // List of generated config files
}

// Cleanup removes temporary files created for the provider
func (env *ProviderEnvironment) Cleanup() error {
	// In production, we might want to keep these for debugging
	// For now, we'll keep them as they're in the workspace
	return nil
}

// CleanupWorkspace removes old workspace files
func (m *InjectionManager) CleanupWorkspace() error {
	// Remove files older than 24 hours
	// Implementation depends on requirements
	return nil
}
