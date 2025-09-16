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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

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
	subAgents        map[string]core.SubAgent
	generatedAgents  []string // Track generated agent files for cleanup
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
		subAgents:        make(map[string]core.SubAgent),
		generatedAgents:  []string{},
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

	// Prepare subagents for Claude Code
	if err := p.prepareSubAgents(); err != nil {
		fmt.Printf("Warning: failed to prepare subagents: %v\n", err)
		// Don't fail session preparation if subagent bridge fails
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

	// Clean up generated subagent files
	if err := p.cleanupSubAgents(); err != nil {
		fmt.Printf("Warning: failed to cleanup subagents: %v\n", err)
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

// SubAgentCapable implementation

// SupportsSubAgents returns true as Claude supports subagents via Task tool
func (p *ClaudeProvider) SupportsSubAgents() bool {
	return true
}

// GetSubAgentType returns the declarative type for Claude
func (p *ClaudeProvider) GetSubAgentType() core.SubAgentType {
	return core.SubAgentTypeDeclarative
}

// CreateSubAgent creates a new subagent with the given configuration
func (p *ClaudeProvider) CreateSubAgent(config core.SubAgentConfig) (core.SubAgent, error) {
	// Import the claude adapter package
	// This would be done at the top of the file in production
	// For now, we'll create a simple wrapper
	
	if config.Provider != core.ProviderTypeClaude {
		return nil, fmt.Errorf("invalid provider type for Claude: %s", config.Provider)
	}
	
	// In production, this would use the claude.NewClaudeAdapter
	// For now, return an error indicating the feature needs the adapter
	return nil, fmt.Errorf("subagent creation requires claude adapter implementation")
}

// ListSubAgents returns available subagents
func (p *ClaudeProvider) ListSubAgents() ([]core.SubAgentConfig, error) {
	// Check .claude/agents directory for agent definitions
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}
	
	agentsDir := filepath.Join(homeDir, ".claude", "agents")
	if _, err := os.Stat(agentsDir); os.IsNotExist(err) {
		return []core.SubAgentConfig{}, nil
	}
	
	var configs []core.SubAgentConfig
	
	// Read agent definition files
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read agents directory: %w", err)
	}
	
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}
		
		// Parse agent definition (simplified for now)
		name := strings.TrimSuffix(entry.Name(), ".md")
		configs = append(configs, core.SubAgentConfig{
			Name:        name,
			Type:        core.SubAgentTypeDeclarative,
			Provider:    core.ProviderTypeClaude,
			Strategy:    core.DelegationAutomatic,
		})
	}
	
	// Also return registered subagents
	for _, agent := range p.subAgents {
		configs = append(configs, agent.Config())
	}
	
	return configs, nil
}

// GetSubAgent retrieves a subagent by name
func (p *ClaudeProvider) GetSubAgent(name string) (core.SubAgent, error) {
	if agent, exists := p.subAgents[name]; exists {
		return agent, nil
	}
	
	return nil, fmt.Errorf("subagent %s not found", name)
}

// Delegate delegates a task to an appropriate subagent
func (p *ClaudeProvider) Delegate(ctx context.Context, task core.SubAgentTask) (*core.SubAgentResult, error) {
	// Find best matching agent
	var bestAgent core.SubAgent
	for _, agent := range p.subAgents {
		if agent.CanHandle(task) {
			bestAgent = agent
			break
		}
	}
	
	if bestAgent == nil {
		return nil, fmt.Errorf("no suitable agent found for task %s", task.Name)
	}
	
	return bestAgent.Execute(ctx, task)
}

// RegisterSubAgent registers a new subagent
func (p *ClaudeProvider) RegisterSubAgent(agent core.SubAgent) error {
	if agent == nil {
		return fmt.Errorf("agent cannot be nil")
	}
	
	p.subAgents[agent.Name()] = agent
	return nil
}

// UnregisterSubAgent unregisters a subagent
func (p *ClaudeProvider) UnregisterSubAgent(name string) error {
	if _, exists := p.subAgents[name]; !exists {
		return fmt.Errorf("agent %s not found", name)
	}
	
	delete(p.subAgents, name)
	return nil
}

// SyncSubAgentsToClaude synchronizes Opun subagents to Claude Code format
// This creates persistent markdown files in ~/.claude/agents/ from ~/.opun/subagents/ configs
func SyncSubAgentsToClaude() error {
	p := &ClaudeProvider{generatedAgents: []string{}}
	return p.syncSubAgents(false)
}

// prepareSubAgents creates markdown files for Claude Code from Opun subagent configs
func (p *ClaudeProvider) prepareSubAgents() error {
	return p.syncSubAgents(true)
}

// syncSubAgents performs the actual sync, with option for temporary vs persistent
func (p *ClaudeProvider) syncSubAgents(temporary bool) error {
	// Load subagent configurations from ~/.opun/subagents/
	loader, err := config.NewSubAgentConfigLoader()
	if err != nil {
		return fmt.Errorf("failed to create config loader: %w", err)
	}

	configs, err := loader.LoadAll()
	if err != nil {
		return fmt.Errorf("failed to load subagent configs: %w", err)
	}

	if len(configs) == 0 {
		return nil // No subagents to prepare
	}

	// Get home directory for .claude path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Create ~/.claude/agents/ directory
	agentsDir := filepath.Join(homeDir, ".claude", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create agents directory: %w", err)
	}

	var syncedAgents []string

	// Generate markdown files for each subagent
	for _, cfg := range configs {
		// Only process agents that support Claude
		if cfg.Provider != core.ProviderTypeClaude && cfg.Provider != "" {
			continue
		}

		markdown := p.generateAgentMarkdown(cfg)
		filename := fmt.Sprintf("%s.md", cfg.Name)
		filepath := filepath.Join(agentsDir, filename)

		if err := os.WriteFile(filepath, []byte(markdown), 0644); err != nil {
			return fmt.Errorf("failed to write agent file %s: %w", filename, err)
		}

		// Track generated files
		if temporary && p != nil {
			p.generatedAgents = append(p.generatedAgents, filepath)
		}
		syncedAgents = append(syncedAgents, cfg.Name)
	}

	// Create a manifest file to track Opun-managed agents (for persistent sync)
	if !temporary {
		manifest := struct {
			Version string   `json:"version"`
			Agents  []string `json:"agents"`
			Updated string   `json:"updated"`
		}{
			Version: "1.0",
			Agents:  syncedAgents,
			Updated: time.Now().Format(time.RFC3339),
		}

		manifestPath := filepath.Join(agentsDir, ".opun-manifest.json")
		data, _ := json.MarshalIndent(manifest, "", "  ")
		if err := os.WriteFile(manifestPath, data, 0644); err != nil {
			fmt.Printf("Warning: failed to write manifest: %v\n", err)
		}

		fmt.Printf("✅ Synced %d subagents to Claude Code\n", len(syncedAgents))
		for _, name := range syncedAgents {
			fmt.Printf("  - %s\n", name)
		}
	}

	return nil
}

// generateAgentMarkdown converts a SubAgentConfig to Claude Task tool markdown format
func (p *ClaudeProvider) generateAgentMarkdown(cfg core.SubAgentConfig) string {
	var sb strings.Builder

	// Start with YAML frontmatter (required for Claude Code)
	sb.WriteString("---\n")
	sb.WriteString(fmt.Sprintf("name: %s\n", cfg.Name))
	
	// Description - extract first line if multi-line
	description := cfg.Description
	if description == "" {
		description = fmt.Sprintf("Specialized agent for %s tasks", cfg.Name)
	}
	// Take only the first line for the frontmatter description
	if idx := strings.Index(description, "\n"); idx > 0 {
		description = description[:idx]
	}
	sb.WriteString(fmt.Sprintf("description: %s\n", description))
	
	// Model preference
	model := cfg.Model
	if model == "" {
		model = "sonnet" // Default model
	}
	sb.WriteString(fmt.Sprintf("model: %s\n", model))
	
	// Capabilities
	sb.WriteString("capabilities:\n")
	if len(cfg.Capabilities) > 0 {
		for _, cap := range cfg.Capabilities {
			sb.WriteString(fmt.Sprintf("  - %s\n", cap))
		}
	} else {
		sb.WriteString("  []\n")
	}
	
	// Context patterns
	sb.WriteString("context:\n")
	if len(cfg.Context) > 0 {
		for _, ctx := range cfg.Context {
			sb.WriteString(fmt.Sprintf("  - %s\n", ctx))
		}
	} else {
		sb.WriteString("  []\n")
	}
	
	// Instructions (system prompt)
	sb.WriteString("instructions: |\n")
	if cfg.SystemPrompt != "" {
		// Indent each line of the system prompt
		lines := strings.Split(cfg.SystemPrompt, "\n")
		for _, line := range lines {
			sb.WriteString(fmt.Sprintf("  %s\n", line))
		}
	} else if cfg.Description != "" {
		// Use full description as instructions if no system prompt
		lines := strings.Split(cfg.Description, "\n")
		for _, line := range lines {
			sb.WriteString(fmt.Sprintf("  %s\n", line))
		}
	} else {
		sb.WriteString(fmt.Sprintf("  You are %s, a specialized Opun agent.\n", cfg.Name))
	}
	
	// End frontmatter
	sb.WriteString("---\n\n")
	
	// Now add the markdown content (for readability in the UI)
	sb.WriteString(fmt.Sprintf("# %s\n\n", cfg.Name))
	
	if cfg.Description != "" {
		sb.WriteString(fmt.Sprintf("%s\n\n", cfg.Description))
	}

	// Add capabilities section if present
	if len(cfg.Capabilities) > 0 {
		sb.WriteString("## Capabilities\n\n")
		for _, cap := range cfg.Capabilities {
			sb.WriteString(fmt.Sprintf("- %s\n", cap))
		}
		sb.WriteString("\n")
	}

	// Add context patterns if available
	if len(cfg.Context) > 0 {
		sb.WriteString("## Context Patterns\n\n")
		sb.WriteString("This agent should be used when:\n")
		for _, ctx := range cfg.Context {
			sb.WriteString(fmt.Sprintf("- %s\n", ctx))
		}
		sb.WriteString("\n")
	}

	// Add the system prompt as instructions if present
	if cfg.SystemPrompt != "" {
		sb.WriteString("## Instructions\n\n")
		sb.WriteString(cfg.SystemPrompt)
		sb.WriteString("\n")
	}

	return sb.String()
}

// cleanupSubAgents removes generated markdown files from ~/.claude/agents/
func (p *ClaudeProvider) cleanupSubAgents() error {
	var lastErr error
	
	for _, filepath := range p.generatedAgents {
		if err := os.Remove(filepath); err != nil {
			if !os.IsNotExist(err) {
				lastErr = err
				fmt.Printf("Warning: failed to remove %s: %v\n", filepath, err)
			}
		}
	}

	// Clear the tracked files
	p.generatedAgents = []string{}
	
	return lastErr
}
