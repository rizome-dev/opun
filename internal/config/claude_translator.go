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
	"os"
	"path/filepath"
	"runtime"

	"github.com/rizome-dev/opun/pkg/core"
)

// ClaudeConfigTranslator translates shared config to Claude Desktop format
type ClaudeConfigTranslator struct{}

// NewClaudeConfigTranslator creates a new Claude config translator
func NewClaudeConfigTranslator() *ClaudeConfigTranslator {
	return &ClaudeConfigTranslator{}
}

// ClaudeDesktopConfig represents Claude Desktop's configuration format
type ClaudeDesktopConfig struct {
	MCPServers map[string]ClaudeMCPServer `json:"mcpServers,omitempty"`
}

// ClaudeMCPServer represents an MCP server in Claude's format
type ClaudeMCPServer struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// TranslateMCPConfig translates shared MCP config to Claude format
func (c *ClaudeConfigTranslator) TranslateMCPConfig(servers []core.SharedMCPServer) (interface{}, error) {
	claudeConfig := ClaudeDesktopConfig{
		MCPServers: make(map[string]ClaudeMCPServer),
	}

	// Add all configured MCP servers
	for _, server := range servers {
		// Skip if not installed (unless it's required)
		if !server.Installed && !server.Required {
			continue
		}

		claudeServer := ClaudeMCPServer{
			Command: server.Command,
			Args:    server.Args,
		}

		// Add environment variables if present and not empty
		if len(server.Env) > 0 {
			envVars := make(map[string]string)
			hasAnyValue := false

			for k, v := range server.Env {
				if v != "" {
					envVars[k] = v
					hasAnyValue = true
				}
			}

			// Only set env if we have actual values
			if hasAnyValue {
				claudeServer.Env = envVars
			}
		}

		claudeConfig.MCPServers[server.Name] = claudeServer
	}

	return claudeConfig, nil
}

// TranslateSlashCommands translates shared commands to Claude format
// Claude doesn't currently support custom slash commands via config
func (c *ClaudeConfigTranslator) TranslateSlashCommands(commands []core.SharedSlashCommand) (interface{}, error) {
	// Claude doesn't support custom slash commands via configuration yet
	// In the future, this could generate a Claude extension or MCP server
	// that provides these commands
	return nil, nil
}

// GetConfigPath returns Claude Desktop's config file path
func (c *ClaudeConfigTranslator) GetConfigPath() string {
	homeDir, _ := os.UserHomeDir()

	// Try different locations based on OS and Claude version
	switch runtime.GOOS {
	case "darwin":
		// Primary location on macOS
		primary := filepath.Join(homeDir, "Library", "Application Support", "Claude", "claude_desktop_config.json")
		if _, err := os.Stat(filepath.Dir(primary)); err == nil {
			return primary
		}
		// Fallback to XDG config
		return filepath.Join(homeDir, ".config", "claude", "claude_desktop_config.json")
	case "linux":
		// XDG config directory
		if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
			return filepath.Join(xdgConfig, "claude", "claude_desktop_config.json")
		}
		return filepath.Join(homeDir, ".config", "claude", "claude_desktop_config.json")
	case "windows":
		// Windows location
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "Claude", "claude_desktop_config.json")
		}
		return filepath.Join(homeDir, "AppData", "Roaming", "Claude", "claude_desktop_config.json")
	default:
		// Fallback
		return filepath.Join(homeDir, ".claude", "claude_desktop_config.json")
	}
}

// SupportsSymlinks returns whether Claude config can be symlinked
func (c *ClaudeConfigTranslator) SupportsSymlinks() bool {
	// Claude Desktop reads the config file directly, so symlinking could work
	// but it's safer to write the actual file
	return false
}
