package providers

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
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rizome-dev/opun/internal/config"
	"github.com/rizome-dev/opun/internal/io"
	"github.com/rizome-dev/opun/internal/utils"
	"github.com/rizome-dev/opun/pkg/core"
)

// ClaudeProvider implements the Provider interface for Claude CLI
type ClaudeProvider struct {
	*core.BaseProvider
	session          *io.TransparentSession
	clipboard        utils.Clipboard
	injectionManager *config.InjectionManager
	environment      *config.ProviderEnvironment
}

// NewClaudeProvider creates a new Claude provider
func NewClaudeProvider(providerConfig core.ProviderConfig) *ClaudeProvider {
	baseProvider := core.NewBaseProvider(providerConfig.Name, core.ProviderTypeClaude)
	baseProvider.Initialize(providerConfig)

	// Create injection manager (optional)
	injectionManager, _ := config.NewInjectionManager(nil)

	return &ClaudeProvider{
		BaseProvider:     baseProvider,
		clipboard:        utils.NewClipboard(),
		injectionManager: injectionManager,
	}
}

// Validate validates the provider configuration
func (p *ClaudeProvider) Validate() error {
	if err := p.BaseProvider.Validate(); err != nil {
		return err
	}

	// Check if claude CLI is available
	if err := p.checkClaudeCLI(); err != nil {
		return fmt.Errorf("claude CLI not available: %w", err)
	}

	return nil
}

// GetPTYCommand returns the command to start Claude
func (p *ClaudeProvider) GetPTYCommand() (*exec.Cmd, error) {
	config := p.Config()
	// #nosec G204 -- executing configured provider command
	cmd := exec.Command(config.Command, config.Args...)

	// Apply injected environment if available
	if p.environment != nil {
		if p.environment.WorkingDir != "" {
			cmd.Dir = p.environment.WorkingDir
		}
		// Add injected environment variables
		cmd.Env = append(os.Environ(), "") // Start with system env
		for k, v := range p.environment.Environment {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	} else {
		// Fall back to config settings
		if config.WorkingDir != "" {
			cmd.Dir = config.WorkingDir
		}
		// Set environment variables from config
		for k, v := range config.Environment {
			cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%s", k, v))
		}
	}

	return cmd, nil
}

// GetPTYCommandWithPrompt returns the command with an initial prompt
func (p *ClaudeProvider) GetPTYCommandWithPrompt(prompt string) (*exec.Cmd, error) {
	// Claude doesn't support initial prompts via command line
	// We'll handle this via clipboard injection
	return p.GetPTYCommand()
}

// SupportsModel checks if Claude supports the given model
func (p *ClaudeProvider) SupportsModel(model string) bool {
	supportedModels := []string{"opus", "sonnet", "haiku"}
	for _, m := range supportedModels {
		if strings.EqualFold(m, model) {
			return true
		}
	}
	return false
}

// PrepareSession prepares a Claude session
func (p *ClaudeProvider) PrepareSession(ctx context.Context, sessionID string) error {
	// Create session directory if needed
	sessionDir := filepath.Join(os.TempDir(), "opun", "sessions", sessionID)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return err
	}

	// Prepare provider environment if injection manager is available
	if p.injectionManager != nil {
		env, err := p.injectionManager.PrepareProviderEnvironment(string(p.Type()))
		if err != nil {
			return fmt.Errorf("failed to prepare provider environment: %w", err)
		}
		p.environment = env
	}

	return nil
}

// CleanupSession cleans up a Claude session
func (p *ClaudeProvider) CleanupSession(ctx context.Context, sessionID string) error {
	// Clean up injected environment
	if p.environment != nil {
		if err := p.environment.Cleanup(); err != nil {
			// Log but don't fail on cleanup errors
			fmt.Printf("Warning: failed to cleanup environment: %v\n", err)
		}
		p.environment = nil
	}

	// Clean up session directory
	sessionDir := filepath.Join(os.TempDir(), "opun", "sessions", sessionID)
	return os.RemoveAll(sessionDir)
}

// GetReadyPattern returns the pattern indicating Claude is ready
func (p *ClaudeProvider) GetReadyPattern() string {
	// Claude Code uses different patterns
	return "> Try"
}

// GetOutputPattern returns the pattern indicating output completion
func (p *ClaudeProvider) GetOutputPattern() string {
	return "Human:"
}

// GetErrorPattern returns the pattern indicating an error
func (p *ClaudeProvider) GetErrorPattern() string {
	return "Error:"
}

// GetPromptInjectionMethod returns how to inject prompts
func (p *ClaudeProvider) GetPromptInjectionMethod() string {
	return "clipboard"
}

// InjectPrompt injects a prompt into Claude
func (p *ClaudeProvider) InjectPrompt(prompt string) error {
	return p.clipboard.Copy(prompt)
}

// GetMCPServers returns MCP servers for Claude
func (p *ClaudeProvider) GetMCPServers() []core.MCPServer {
	// Claude supports various MCP servers
	return []core.MCPServer{
		{
			Name:        "filesystem",
			Description: "File system operations",
			Enabled:     true,
		},
		{
			Name:        "web-browser",
			Description: "Web browsing capabilities",
			Enabled:     false,
		},
		{
			Name:        "code-analysis",
			Description: "Code analysis tools",
			Enabled:     true,
		},
	}
}

// GetTools returns available tools
func (p *ClaudeProvider) GetTools() []core.Tool {
	return []core.Tool{
		{
			Name:        "read_file",
			Description: "Read contents of a file",
			Category:    "filesystem",
		},
		{
			Name:        "write_file",
			Description: "Write contents to a file",
			Category:    "filesystem",
		},
		{
			Name:        "run_command",
			Description: "Execute a shell command",
			Category:    "system",
		},
	}
}

// GetSlashCommands returns slash commands supported by Claude
func (p *ClaudeProvider) GetSlashCommands() []core.SharedSlashCommand {
	// Claude supports slash commands via .claude/commands/ directory
	// These will be populated from the shared config
	return []core.SharedSlashCommand{}
}

// GetPlugins returns plugins used by Claude
func (p *ClaudeProvider) GetPlugins() []core.PluginReference {
	// Return empty list - plugins are handled via MCP servers
	return []core.PluginReference{}
}

// SupportsSlashCommands returns true as Claude supports slash commands
func (p *ClaudeProvider) SupportsSlashCommands() bool {
	return true
}

// GetSlashCommandDirectory returns the directory for Claude slash commands
func (p *ClaudeProvider) GetSlashCommandDirectory() string {
	return ".claude/commands"
}

// GetSlashCommandFormat returns markdown as the format for Claude commands
func (p *ClaudeProvider) GetSlashCommandFormat() string {
	return "markdown"
}

// PrepareSlashCommands creates markdown files for Claude slash commands
func (p *ClaudeProvider) PrepareSlashCommands(commands []core.SharedSlashCommand, targetDir string) error {
	commandsDir := filepath.Join(targetDir, p.GetSlashCommandDirectory())
	if err := os.MkdirAll(commandsDir, 0755); err != nil {
		return fmt.Errorf("failed to create commands directory: %w", err)
	}

	// Generate markdown files for each command
	for _, cmd := range commands {
		if cmd.Hidden {
			continue
		}

		content := p.generateCommandMarkdown(cmd)
		filename := fmt.Sprintf("%s.md", cmd.Name)
		cmdPath := filepath.Join(commandsDir, filename)

		if err := os.WriteFile(cmdPath, []byte(content), 0644); err != nil {
			return fmt.Errorf("failed to write command %s: %w", cmd.Name, err)
		}

		// Also create files for aliases
		for _, alias := range cmd.Aliases {
			aliasPath := filepath.Join(commandsDir, fmt.Sprintf("%s.md", alias))
			if err := os.WriteFile(aliasPath, []byte(content), 0644); err != nil {
				return fmt.Errorf("failed to write alias %s: %w", alias, err)
			}
		}
	}

	return nil
}

// generateCommandMarkdown generates markdown content for a Claude command
func (p *ClaudeProvider) generateCommandMarkdown(cmd core.SharedSlashCommand) string {
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

// StartSession starts an interactive session
func (p *ClaudeProvider) StartSession(ctx context.Context, workDir string) (*io.TransparentSession, error) {
	cmd, args := p.getCommand()

	config := io.TransparentSessionConfig{
		Provider: p.Name(),
		Command:  cmd,
		Args:     args,
	}

	session, err := io.NewTransparentSession(config)
	if err != nil {
		return nil, err
	}

	p.session = session
	return session, nil
}

// SendPrompt sends a prompt to the session
func (p *ClaudeProvider) SendPrompt(prompt string) error {
	if p.session == nil {
		return fmt.Errorf("no active session")
	}
	return p.session.SendInput([]byte(prompt + "\n"))
}

// CloseSession closes the current session
func (p *ClaudeProvider) CloseSession() error {
	if p.session == nil {
		return nil
	}
	err := p.session.Close()
	p.session = nil
	return err
}

// GetReadyPatterns returns patterns that indicate Claude is ready
func (p *ClaudeProvider) GetReadyPatterns() []string {
	return []string{
		"Human:",
		"Human Assistant Chat",
		">",
	}
}

// getCommand returns the command and args to run Claude
func (p *ClaudeProvider) getCommand() (string, []string) {
	config := p.Config()
	args := []string{}

	// Add model if specified
	if config.Model != "" {
		args = append(args, "--model", config.Model)
	}

	// Add any additional args from config
	args = append(args, config.Args...)

	// Get the claude command
	cmd := p.getClaudeCommand()

	// If using npx, add claude-code
	if cmd == "npx" {
		args = append([]string{"claude-code"}, args...)
	}

	return cmd, args
}

// checkClaudeCLI checks if the Claude CLI is available
func (p *ClaudeProvider) checkClaudeCLI() error {
	// Try 'claude' command first
	if _, err := exec.LookPath("claude"); err == nil {
		return nil
	}

	// Try 'npx claude-code' as fallback
	if _, err := exec.LookPath("npx"); err == nil {
		// Check if claude-code package is available
		cmd := exec.Command("npx", "--no-install", "claude-code", "--version")
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	return fmt.Errorf("claude CLI not found in PATH")
}

// getClaudeCommand returns the command to run Claude
func (p *ClaudeProvider) getClaudeCommand() string {
	config := p.Config()
	// Check for override in config
	if cmd, ok := config.Settings["command"].(string); ok && cmd != "" {
		return cmd
	}

	// Try 'claude' first
	if _, err := exec.LookPath("claude"); err == nil {
		return "claude"
	}

	// Fallback to npx
	return "npx"
}

// isInteractiveMode checks if we're in interactive mode
func (p *ClaudeProvider) isInteractiveMode() bool {
	config := p.Config()
	if interactive, ok := config.Settings["interactive"].(bool); ok {
		return interactive
	}
	return true // Default to interactive
}
