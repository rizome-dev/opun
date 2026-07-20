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
			Description: "Persistent memory across sessions - enables the provider to remember context between conversations",
			Required:    true,
		},
		{
			Name:        "sequential-thinking",
			Package:     "@modelcontextprotocol/server-sequential-thinking",
			Description: "Enhanced reasoning capabilities - improves step-by-step problem solving",
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
				"OPENROUTER_DEFAULT_MODEL": "Default model to use (e.g., anthropic/claude-sonnet-5)",
			},
		},
	}
}
