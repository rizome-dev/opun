package core

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
	"time"
)

// SharedConfig represents the unified configuration for all providers
type SharedConfig struct {
	Version       string               `yaml:"version"`
	MCPServers    []SharedMCPServer    `yaml:"mcp_servers"`
	SlashCommands []SharedSlashCommand `yaml:"slash_commands"`
	LastUpdated   time.Time            `yaml:"last_updated"`
}

// SharedMCPServer represents an MCP server in the shared configuration
type SharedMCPServer struct {
	Name        string            `yaml:"name"`
	Package     string            `yaml:"package"`
	Command     string            `yaml:"command"`
	Args        []string          `yaml:"args"`
	Required    bool              `yaml:"required"`
	Installed   bool              `yaml:"installed"`
	InstallPath string            `yaml:"install_path,omitempty"`
	Version     string            `yaml:"version,omitempty"`
	Env         map[string]string `yaml:"env,omitempty"`
}

// SharedSlashCommand represents a slash command in the shared configuration
type SharedSlashCommand struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Type        string   `yaml:"type"` // workflow, prompt, plugin, builtin
	Handler     string   `yaml:"handler"`
	Aliases     []string `yaml:"aliases,omitempty"`
	Arguments   []string `yaml:"arguments,omitempty"`
	Hidden      bool     `yaml:"hidden,omitempty"`
}

// ProviderConfigTranslator translates shared config to provider-specific formats
type ProviderConfigTranslator interface {
	// TranslateMCPConfig translates shared MCP config to provider format
	TranslateMCPConfig(servers []SharedMCPServer) (interface{}, error)

	// TranslateSlashCommands translates shared commands to provider format
	TranslateSlashCommands(commands []SharedSlashCommand) (interface{}, error)

	// GetConfigPath returns the provider's config file path
	GetConfigPath() string

	// SupportsSymlinks returns true if provider config can be symlinked
	SupportsSymlinks() bool
}
