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
	"os/exec"
	"path/filepath"
	"strings"

	"time"

	"github.com/rizome-dev/opun/internal/utils"
	"github.com/rizome-dev/opun/pkg/core"
	"gopkg.in/yaml.v3"
)

// SharedConfigManager manages the unified configuration for all providers
type SharedConfigManager struct {
	configPath string
	config     *core.SharedConfig
}

// NewSharedConfigManager creates a new shared configuration manager
func NewSharedConfigManager() (*SharedConfigManager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".opun", "shared-config.yaml")

	manager := &SharedConfigManager{
		configPath: configPath,
	}

	// Load existing config or create default
	if err := manager.Load(); err != nil {
		if os.IsNotExist(err) {
			manager.config = manager.getDefaultConfig()
			if err := manager.Save(); err != nil {
				return nil, fmt.Errorf("failed to save default config: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
	}

	return manager, nil
}

// Load loads the shared configuration from disk
func (m *SharedConfigManager) Load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		return err
	}

	var config core.SharedConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	m.config = &config
	return nil
}

// Save saves the shared configuration to disk
func (m *SharedConfigManager) Save() error {
	// Update timestamp
	m.config.LastUpdated = time.Now()

	data, err := yaml.Marshal(m.config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Use utils.WriteFile which handles directory creation and permissions
	if err := utils.WriteFile(m.configPath, data); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// CheckMCPServerInstalled checks if an MCP server is already installed
func (m *SharedConfigManager) CheckMCPServerInstalled(serverName string) (bool, string, error) {
	// First check our tracked status
	for _, server := range m.config.MCPServers {
		if server.Name == serverName && server.Installed {
			// Verify it's actually installed
			if server.InstallPath != "" {
				if _, err := os.Stat(server.InstallPath); err == nil {
					return true, server.Version, nil
				}
			}

			// Check npm global install
			cmd := exec.Command("npm", "list", "-g", server.Package, "--json")
			output, err := cmd.Output()
			if err == nil {
				var result map[string]interface{}
				if err := json.Unmarshal(output, &result); err == nil {
					if deps, ok := result["dependencies"].(map[string]interface{}); ok {
						if pkg, ok := deps[server.Package].(map[string]interface{}); ok {
							if version, ok := pkg["version"].(string); ok {
								return true, version, nil
							}
						}
					}
				}
			}
		}
	}

	// Direct npm check
	for _, server := range m.config.MCPServers {
		if server.Name == serverName {
			cmd := exec.Command("npm", "list", "-g", server.Package, "--json")
			output, err := cmd.Output()
			if err == nil {
				var result map[string]interface{}
				if err := json.Unmarshal(output, &result); err == nil {
					if deps, ok := result["dependencies"].(map[string]interface{}); ok {
						if pkg, ok := deps[server.Package].(map[string]interface{}); ok {
							if version, ok := pkg["version"].(string); ok {
								// Update our tracking
								m.UpdateMCPServerStatus(serverName, true, version)
								return true, version, nil
							}
						}
					}
				}
			}
			break
		}
	}

	return false, "", nil
}

// UpdateMCPServerStatus updates the installation status of an MCP server
func (m *SharedConfigManager) UpdateMCPServerStatus(serverName string, installed bool, version string) error {
	for i, server := range m.config.MCPServers {
		if server.Name == serverName {
			m.config.MCPServers[i].Installed = installed
			m.config.MCPServers[i].Version = version
			return m.Save()
		}
	}
	return fmt.Errorf("server %s not found", serverName)
}

// GetMCPServers returns all configured MCP servers
func (m *SharedConfigManager) GetMCPServers() []core.SharedMCPServer {
	return m.config.MCPServers
}

// GetSlashCommands returns all configured slash commands
func (m *SharedConfigManager) GetSlashCommands() []core.SharedSlashCommand {
	return m.config.SlashCommands
}

// AddMCPServer adds a new MCP server to the configuration
func (m *SharedConfigManager) AddMCPServer(server core.SharedMCPServer) error {
	// Check if already exists
	for i, existing := range m.config.MCPServers {
		if existing.Name == server.Name {
			m.config.MCPServers[i] = server
			return m.Save()
		}
	}

	m.config.MCPServers = append(m.config.MCPServers, server)
	return m.Save()
}

// AddSlashCommand adds a new slash command to the configuration
func (m *SharedConfigManager) AddSlashCommand(command core.SharedSlashCommand) error {
	// Check if already exists
	for i, existing := range m.config.SlashCommands {
		if existing.Name == command.Name {
			m.config.SlashCommands[i] = command
			return m.Save()
		}
	}

	m.config.SlashCommands = append(m.config.SlashCommands, command)
	return m.Save()
}

// SyncToProvider syncs the shared configuration to a specific provider
func (m *SharedConfigManager) SyncToProvider(providerName string) error {
	var translator core.ProviderConfigTranslator

	switch strings.ToLower(providerName) {
	case "claude":
		translator = NewClaudeConfigTranslator()
	case "gemini":
		translator = NewGeminiConfigTranslator()
	default:
		return fmt.Errorf("unsupported provider: %s", providerName)
	}

	// Ensure the opun MCP server is available
	m.ensureOpunServer()

	// Translate and write MCP config
	if mcpConfig, err := translator.TranslateMCPConfig(m.config.MCPServers); err != nil {
		return fmt.Errorf("failed to translate MCP config: %w", err)
	} else if err := m.writeProviderConfig(translator, mcpConfig); err != nil {
		return fmt.Errorf("failed to write provider config: %w", err)
	}

	// Slash commands are now exposed through the MCP server

	return nil
}

// writeProviderConfig writes the translated config to the provider's config file
func (m *SharedConfigManager) writeProviderConfig(translator core.ProviderConfigTranslator, config interface{}) error {
	configPath := translator.GetConfigPath()

	// Expand ~ to home directory
	if strings.HasPrefix(configPath, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		configPath = filepath.Join(homeDir, configPath[2:])
	}

	// Marshal config
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Use utils.WriteFile which handles directory creation and permissions
	if err := utils.WriteFile(configPath, data); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// getDefaultConfig returns the default shared configuration
func (m *SharedConfigManager) getDefaultConfig() *core.SharedConfig {
	return &core.SharedConfig{
		Version: "1.0",
		MCPServers: []core.SharedMCPServer{
			{
				Name:        "opun",
				Package:     "opun",
				Command:     "opun",
				Args:        []string{"mcp", "stdio"},
				Required:    true,
				Installed:   true,
				InstallPath: "builtin",
			},
			{
				Name:     "memory",
				Package:  "@modelcontextprotocol/server-memory",
				Command:  "npx",
				Args:     []string{"@modelcontextprotocol/server-memory"},
				Required: true,
			},
			{
				Name:     "sequential-thinking",
				Package:  "@modelcontextprotocol/server-sequential-thinking",
				Command:  "npx",
				Args:     []string{"@modelcontextprotocol/server-sequential-thinking"},
				Required: true,
			},
			{
				Name:    "filesystem",
				Package: "@modelcontextprotocol/server-filesystem",
				Command: "npx",
				Args:    []string{"@modelcontextprotocol/server-filesystem", "~/Documents"},
			},
			{
				Name:    "context7",
				Package: "@upstash/context7-mcp",
				Command: "npx",
				Args:    []string{"@upstash/context7-mcp"},
				// Context7 is free and doesn't require an API key
			},
		},
		SlashCommands: []core.SharedSlashCommand{
			{
				Name:        "refactor",
				Description: "Run code refactoring workflow",
				Type:        "workflow",
				Handler:     "refactor-code",
				Aliases:     []string{"ref"},
			},
			{
				Name:        "analyze",
				Description: "Analyze codebase structure",
				Type:        "prompt",
				Handler:     "promptgarden://analyze-code",
			},
			{
				Name:        "mcp",
				Description: "Manage MCP servers",
				Type:        "builtin",
				Handler:     "mcp_manager",
			},
		},
	}
}

// ensureOpunServer ensures the unified Opun MCP server is in the configuration
func (m *SharedConfigManager) ensureOpunServer() {
	// Check if already exists
	for _, server := range m.config.MCPServers {
		if server.Name == "opun" {
			return
		}
	}

	// Add the unified Opun server with stdio support
	opunServer := core.SharedMCPServer{
		Name:        "opun",
		Package:     "opun",
		Command:     "opun",
		Args:        []string{"mcp", "stdio"},
		Required:    true,
		Installed:   true,
		InstallPath: "builtin",
	}

	m.config.MCPServers = append(m.config.MCPServers, opunServer)
}
