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

// QwenConfigTranslator translates shared config to Qwen format
type QwenConfigTranslator struct{}

// NewQwenConfigTranslator creates a new Qwen config translator
func NewQwenConfigTranslator() *QwenConfigTranslator {
	return &QwenConfigTranslator{}
}

// QwenConfig represents Qwen's configuration format
// Since Qwen is a fork of Gemini, it uses the same format
type QwenConfig struct {
	MCPServers map[string]QwenMCPServer `json:"mcpServers,omitempty"`
}

// QwenMCPServer represents an MCP server in Qwen's format
type QwenMCPServer struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	URL     string            `json:"url,omitempty"`
	HTTPURL string            `json:"httpUrl,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
	CWD     string            `json:"cwd,omitempty"`
	Timeout int               `json:"timeout,omitempty"`
	Trust   bool              `json:"trust,omitempty"`
}

// TranslateMCPConfig translates shared MCP config to Qwen format
func (q *QwenConfigTranslator) TranslateMCPConfig(servers []core.SharedMCPServer) (interface{}, error) {
	qwenConfig := QwenConfig{
		MCPServers: make(map[string]QwenMCPServer),
	}

	// Convert MCP servers to Qwen format
	for _, server := range servers {
		// Skip if not installed (unless required)
		if !server.Installed && !server.Required {
			continue
		}

		qwenServer := QwenMCPServer{
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
				qwenServer.Env = envVars
			}
		}

		qwenConfig.MCPServers[server.Name] = qwenServer
	}

	return qwenConfig, nil
}

// TranslateSlashCommands is not supported by Qwen
func (q *QwenConfigTranslator) TranslateSlashCommands(commands []core.SharedSlashCommand) (interface{}, error) {
	// Qwen doesn't support custom slash commands
	// It only has built-in commands like /mcp, /chat, etc.
	// Extensions should be done via MCP servers
	return nil, nil
}

// GetConfigPath returns Qwen's config file path
func (q *QwenConfigTranslator) GetConfigPath() string {
	homeDir, _ := os.UserHomeDir()

	// Since Qwen is a fork of Gemini, it likely uses ~/.qwen/settings.json
	return filepath.Join(homeDir, ".qwen", "settings.json")
}

// SupportsSymlinks returns whether Qwen config can be symlinked
func (q *QwenConfigTranslator) SupportsSymlinks() bool {
	// Assume Qwen can handle symlinks for now
	return true
}

