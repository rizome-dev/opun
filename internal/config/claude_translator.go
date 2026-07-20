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

	"github.com/rizome-dev/opun/pkg/core"
)

// ClaudeConfigTranslator translates shared config to Claude Code CLI format
type ClaudeConfigTranslator struct{}

// NewClaudeConfigTranslator creates a new Claude config translator
func NewClaudeConfigTranslator() *ClaudeConfigTranslator {
	return &ClaudeConfigTranslator{}
}

// ClaudeConfig represents the mcpServers section of Claude Code's user config
type ClaudeConfig struct {
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
	claudeConfig := ClaudeConfig{
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

// GetConfigPath returns the Claude Code CLI's user-scope config file path
func (c *ClaudeConfigTranslator) GetConfigPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".claude.json")
}

// SupportsSymlinks returns whether Claude config can be symlinked
func (c *ClaudeConfigTranslator) SupportsSymlinks() bool {
	return false
}
