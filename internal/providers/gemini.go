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

// GeminiProvider implements the Provider interface for Gemini CLI
type GeminiProvider struct {
	*core.BaseProvider
	session          *io.TransparentSession
	clipboard        utils.Clipboard
	injectionManager *config.InjectionManager
	environment      *config.ProviderEnvironment
}

// NewGeminiProvider creates a new Gemini provider
func NewGeminiProvider(providerConfig core.ProviderConfig) *GeminiProvider {
	baseProvider := core.NewBaseProvider(providerConfig.Name, core.ProviderTypeGemini)
	baseProvider.Initialize(providerConfig)

	// Create injection manager (optional)
	injectionManager, _ := config.NewInjectionManager(nil)

	return &GeminiProvider{
		BaseProvider:     baseProvider,
		clipboard:        utils.NewClipboard(),
		injectionManager: injectionManager,
	}
}

// Validate validates the provider configuration
func (p *GeminiProvider) Validate() error {
	if err := p.BaseProvider.Validate(); err != nil {
		return err
	}

	// Check if gemini CLI is available
	if err := p.checkGeminiCLI(); err != nil {
		return fmt.Errorf("gemini CLI not available: %w", err)
	}

	return nil
}

// GetPTYCommand returns the command to start Gemini
func (p *GeminiProvider) GetPTYCommand() (*exec.Cmd, error) {
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
func (p *GeminiProvider) GetPTYCommandWithPrompt(prompt string) (*exec.Cmd, error) {
	// Gemini doesn't support initial prompts via command line
	// We'll handle this via clipboard injection
	return p.GetPTYCommand()
}

// SupportsModel checks if Gemini supports the given model
func (p *GeminiProvider) SupportsModel(model string) bool {
	supportedModels := []string{"pro", "flash", "ultra"}
	for _, m := range supportedModels {
		if strings.EqualFold(m, model) {
			return true
		}
	}
	return false
}

// PrepareSession prepares a Gemini session
func (p *GeminiProvider) PrepareSession(ctx context.Context, sessionID string) error {
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

// CleanupSession cleans up a Gemini session
func (p *GeminiProvider) CleanupSession(ctx context.Context, sessionID string) error {
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

// GetReadyPattern returns the pattern indicating Gemini is ready
func (p *GeminiProvider) GetReadyPattern() string {
	return "│ >"
}

// GetOutputPattern returns the pattern indicating output completion
func (p *GeminiProvider) GetOutputPattern() string {
	return "│ >"
}

// GetErrorPattern returns the pattern indicating an error
func (p *GeminiProvider) GetErrorPattern() string {
	return "Error:"
}

// GetPromptInjectionMethod returns how to inject prompts
func (p *GeminiProvider) GetPromptInjectionMethod() string {
	return "clipboard"
}

// InjectPrompt injects a prompt into Gemini
func (p *GeminiProvider) InjectPrompt(prompt string) error {
	return p.clipboard.Copy(prompt)
}

// GetMCPServers returns MCP servers for Gemini
func (p *GeminiProvider) GetMCPServers() []core.MCPServer {
	// Gemini has limited MCP support currently
	return []core.MCPServer{
		{
			Name:        "filesystem",
			Description: "File system operations",
			Enabled:     true,
		},
	}
}

// GetTools returns available tools
func (p *GeminiProvider) GetTools() []core.Tool {
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
	}
}

// GetSlashCommands returns slash commands supported by Gemini
func (p *GeminiProvider) GetSlashCommands() []core.SharedSlashCommand {
	// Gemini supports slash commands via MCP integration
	// These will be populated from the shared config
	return []core.SharedSlashCommand{}
}

// GetPlugins returns plugins used by Gemini
func (p *GeminiProvider) GetPlugins() []core.PluginReference {
	// Return empty list - plugins are handled via MCP servers
	return []core.PluginReference{}
}

// SupportsSlashCommands returns true as Gemini supports MCP-based slash commands
func (p *GeminiProvider) SupportsSlashCommands() bool {
	return true // Via MCP servers
}

// GetSlashCommandDirectory returns empty as Gemini uses MCP not directories
func (p *GeminiProvider) GetSlashCommandDirectory() string {
	return ""
}

// GetSlashCommandFormat returns "mcp" as Gemini uses MCP servers
func (p *GeminiProvider) GetSlashCommandFormat() string {
	return "mcp"
}

// PrepareSlashCommands ensures MCP servers are configured for Gemini
func (p *GeminiProvider) PrepareSlashCommands(commands []core.SharedSlashCommand, targetDir string) error {
	// Gemini uses MCP servers to expose commands
	// The actual configuration is handled by the MCP sync process
	// Nothing to do here as MCP servers are configured in settings.json
	return nil
}

// StartSession starts an interactive session
func (p *GeminiProvider) StartSession(ctx context.Context, workDir string) (*io.TransparentSession, error) {
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
func (p *GeminiProvider) SendPrompt(prompt string) error {
	if p.session == nil {
		return fmt.Errorf("no active session")
	}
	return p.session.SendInput([]byte(prompt + "\n"))
}

// CloseSession closes the current session
func (p *GeminiProvider) CloseSession() error {
	if p.session == nil {
		return nil
	}
	err := p.session.Close()
	p.session = nil
	return err
}

// GetReadyPatterns returns patterns that indicate Gemini is ready
func (p *GeminiProvider) GetReadyPatterns() []string {
	return []string{
		"│ >",
		"Type your message",
	}
}

// getCommand returns the command and args to run Gemini
func (p *GeminiProvider) getCommand() (string, []string) {
	config := p.Config()
	args := []string{"chat"}

	// Add model if specified
	if config.Model != "" {
		args = append(args, "--model", config.Model)
	} else {
		// Default to gemini-pro
		args = append(args, "--model", "gemini-pro")
	}

	// Add temperature if specified
	if temp, ok := config.Settings["temperature"].(float64); ok {
		args = append(args, "--temperature", fmt.Sprintf("%.2f", temp))
	}

	// Add any additional args from config
	args = append(args, config.Args...)

	// Get the gemini command
	cmd := p.getGeminiCommand()

	return cmd, args
}

// checkGeminiCLI checks if the Gemini CLI is available
func (p *GeminiProvider) checkGeminiCLI() error {
	// Try 'gemini' command
	if _, err := exec.LookPath("gemini"); err != nil {
		return fmt.Errorf("gemini CLI not found in PATH")
	}

	// Verify it works
	cmd := exec.Command("gemini", "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gemini CLI found but not working: %w", err)
	}

	return nil
}

// getGeminiCommand returns the command to run Gemini
func (p *GeminiProvider) getGeminiCommand() string {
	config := p.Config()
	// Check for override in config
	if cmd, ok := config.Settings["command"].(string); ok && cmd != "" {
		return cmd
	}

	return "gemini"
}
