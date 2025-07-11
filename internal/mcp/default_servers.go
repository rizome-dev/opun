package mcp

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
)

// DefaultMCPServer represents a default MCP server configuration
type DefaultMCPServer struct {
	Name         string            `json:"name"`
	Package      string            `json:"package"`
	Description  string            `json:"description"`
	Required     bool              `json:"required"`
	EnvVars      map[string]string `json:"env_vars,omitempty"`
	Dependencies []string          `json:"dependencies,omitempty"`
}

// GetDefaultMCPServers returns the list of default MCP servers
func GetDefaultMCPServers() []DefaultMCPServer {
	return []DefaultMCPServer{
		{
			Name:        "memory",
			Package:     "@modelcontextprotocol/server-memory",
			Description: "Persistent memory across sessions - enables Claude to remember context between conversations",
			Required:    true,
		},
		{
			Name:        "sequential-thinking",
			Package:     "@modelcontextprotocol/server-sequential-thinking",
			Description: "Enhanced reasoning capabilities - improves Claude's step-by-step problem solving",
			Required:    true,
		},
		{
			Name:        "context7",
			Package:     "@upstash/context7-mcp",
			Description: "Advanced context management - sophisticated context handling and retrieval",
			Required:    false,
		},
		{
			Name:        "openrouterai",
			Package:     "@mcpservers/openrouterai",
			Description: "OpenRouter AI integration - access to multiple AI models through OpenRouter",
			Required:    false,
			EnvVars: map[string]string{
				"OPENROUTER_API_KEY":       "Your OpenRouter API key",
				"OPENROUTER_DEFAULT_MODEL": "Default model to use (e.g., anthropic/claude-3.5-sonnet)",
			},
		},
	}
}

// MCPInstaller handles installation of MCP servers
type MCPInstaller struct {
	servers []DefaultMCPServer
}

// NewMCPInstaller creates a new MCP installer
func NewMCPInstaller() *MCPInstaller {
	return &MCPInstaller{
		servers: GetDefaultMCPServers(),
	}
}

// InstallServer installs a single MCP server
func (i *MCPInstaller) InstallServer(ctx context.Context, server DefaultMCPServer) error {
	fmt.Printf("📦 Installing MCP server: %s\n", server.Name)

	// Check if npm is available
	if !i.hasNPM() {
		return fmt.Errorf("npm is required to install MCP servers. Please install Node.js and npm")
	}

	// Install the package globally
	cmd := exec.CommandContext(ctx, "npm", "install", "-g", server.Package)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to install %s: %w", server.Package, err)
	}

	fmt.Printf("✅ Successfully installed: %s\n", server.Name)
	return nil
}

// InstallServers installs multiple MCP servers
func (i *MCPInstaller) InstallServers(ctx context.Context, serverNames []string) error {
	for _, name := range serverNames {
		server := i.findServer(name)
		if server == nil {
			fmt.Printf("⚠️  Server '%s' not found in default list\n", name)
			continue
		}

		if err := i.InstallServer(ctx, *server); err != nil {
			fmt.Printf("❌ Failed to install %s: %v\n", name, err)
			continue
		}
	}

	return nil
}

// GenerateClaudeConfig generates Claude Desktop configuration for installed servers
func (i *MCPInstaller) GenerateClaudeConfig(serverNames []string) (map[string]interface{}, error) {
	config := make(map[string]interface{})
	mcpServers := make(map[string]interface{})

	for _, name := range serverNames {
		server := i.findServer(name)
		if server == nil {
			continue
		}

		serverConfig := map[string]interface{}{
			"command": "npx",
			"args":    []string{server.Package},
		}

		// Add environment variables if required
		if len(server.EnvVars) > 0 {
			env := make(map[string]string)
			for key := range server.EnvVars {
				// Check if env var exists
				if value := os.Getenv(key); value != "" {
					env[key] = value
				} else {
					// Use placeholder for missing env vars
					env[key] = fmt.Sprintf("${%s}", key)
				}
			}
			serverConfig["env"] = env
		}

		mcpServers[name] = serverConfig
	}

	config["mcpServers"] = mcpServers
	return config, nil
}

// WriteClaudeConfig writes the Claude Desktop configuration file
func (i *MCPInstaller) WriteClaudeConfig(serverNames []string) error {
	config, err := i.GenerateClaudeConfig(serverNames)
	if err != nil {
		return err
	}

	// Determine Claude Desktop config path
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Try common Claude Desktop config locations
	configPaths := []string{
		filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"),
		filepath.Join(home, ".config", "claude", "claude_desktop_config.json"),
		filepath.Join(home, ".claude", "claude_desktop_config.json"),
	}

	var configPath string
	for _, path := range configPaths {
		if dir := filepath.Dir(path); dir != "" {
			if err := os.MkdirAll(dir, 0755); err == nil {
				configPath = path
				break
			}
		}
	}

	if configPath == "" {
		return fmt.Errorf("could not determine Claude Desktop config path")
	}

	// Write config file
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return err
	}

	fmt.Printf("📝 Claude Desktop config written to: %s\n", configPath)
	return nil
}

// CheckServerInstalled checks if an MCP server is installed
func (i *MCPInstaller) CheckServerInstalled(server DefaultMCPServer) bool {
	cmd := exec.Command("npm", "list", "-g", server.Package)
	return cmd.Run() == nil
}

// ListInstalledServers returns a list of installed MCP servers
func (i *MCPInstaller) ListInstalledServers() []string {
	var installed []string
	for _, server := range i.servers {
		if i.CheckServerInstalled(server) {
			installed = append(installed, server.Name)
		}
	}
	return installed
}

// hasNPM checks if npm is available
func (i *MCPInstaller) hasNPM() bool {
	cmd := exec.Command("npm", "--version")
	return cmd.Run() == nil
}

// findServer finds a server by name
func (i *MCPInstaller) findServer(name string) *DefaultMCPServer {
	for _, server := range i.servers {
		if server.Name == name {
			return &server
		}
	}
	return nil
}

// GetRequiredEnvVars returns required environment variables for a server
func (i *MCPInstaller) GetRequiredEnvVars(serverName string) map[string]string {
	server := i.findServer(serverName)
	if server == nil {
		return nil
	}
	return server.EnvVars
}

// ValidateEnvVars validates that required environment variables are set
func (i *MCPInstaller) ValidateEnvVars(serverName string) []string {
	server := i.findServer(serverName)
	if server == nil {
		return nil
	}

	var missing []string
	for key := range server.EnvVars {
		if os.Getenv(key) == "" {
			missing = append(missing, key)
		}
	}

	return missing
}
