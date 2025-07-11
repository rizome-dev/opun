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

// GeminiConfigTranslator translates shared config to Gemini format
type GeminiConfigTranslator struct{}

// NewGeminiConfigTranslator creates a new Gemini config translator
func NewGeminiConfigTranslator() *GeminiConfigTranslator {
	return &GeminiConfigTranslator{}
}

// GeminiConfig represents Gemini's configuration format
type GeminiConfig struct {
	MCPServers map[string]GeminiMCPServer `json:"mcpServers,omitempty"`
}

// GeminiMCPServer represents an MCP server in Gemini's format
type GeminiMCPServer struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	URL     string            `json:"url,omitempty"`
	HTTPURL string            `json:"httpUrl,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	CWD     string            `json:"cwd,omitempty"`
	Timeout int               `json:"timeout,omitempty"`
	Trust   bool              `json:"trust,omitempty"`
}

// TranslateMCPConfig translates shared MCP config to Gemini format
func (g *GeminiConfigTranslator) TranslateMCPConfig(servers []core.SharedMCPServer) (interface{}, error) {
	geminiConfig := GeminiConfig{
		MCPServers: make(map[string]GeminiMCPServer),
	}

	// Convert MCP servers to Gemini format
	for _, server := range servers {
		// Skip if not installed (unless required)
		if !server.Installed && !server.Required {
			continue
		}

		geminiServer := GeminiMCPServer{
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
				geminiServer.Env = envVars
			}
		}

		geminiConfig.MCPServers[server.Name] = geminiServer
	}

	return geminiConfig, nil
}

// TranslateSlashCommands is not supported by Gemini
func (g *GeminiConfigTranslator) TranslateSlashCommands(commands []core.SharedSlashCommand) (interface{}, error) {
	// Gemini doesn't support custom slash commands
	// It only has built-in commands like /mcp, /chat, etc.
	// Extensions should be done via MCP servers
	return nil, nil
}

// GetConfigPath returns Gemini's config file path
func (g *GeminiConfigTranslator) GetConfigPath() string {
	homeDir, _ := os.UserHomeDir()

	// According to Gemini CLI documentation, settings.json is located at ~/.gemini/settings.json
	return filepath.Join(homeDir, ".gemini", "settings.json")
}

// SupportsSymlinks returns whether Gemini config can be symlinked
func (g *GeminiConfigTranslator) SupportsSymlinks() bool {
	// Assume Gemini can handle symlinks for now
	return true
}
