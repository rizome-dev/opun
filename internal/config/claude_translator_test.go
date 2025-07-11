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
	"strings"
	"testing"

	"github.com/rizome-dev/opun/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaudeConfigTranslator_TranslateMCPConfig(t *testing.T) {
	// Set test home directory
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	translator := NewClaudeConfigTranslator()

	servers := []core.SharedMCPServer{
		{
			Name:        "opun",
			Command:     "opun",
			Args:        []string{"mcp", "stdio"},
			Installed:   true,
			Required:    true,
			InstallPath: "builtin",
		},
		{
			Name:      "filesystem",
			Command:   "npx",
			Args:      []string{"@modelcontextprotocol/server-filesystem", "~/Documents"},
			Installed: false,
		},
		{
			Name:     "memory",
			Command:  "npx",
			Args:     []string{"@modelcontextprotocol/server-memory"},
			Env:      map[string]string{"KEY": "value"},
			Required: true,
		},
	}

	config, err := translator.TranslateMCPConfig(servers)
	require.NoError(t, err)

	claudeConfig, ok := config.(ClaudeDesktopConfig)
	require.True(t, ok)

	// Check mcpServers - only installed or required servers are included
	assert.Len(t, claudeConfig.MCPServers, 2)

	// Check opun server
	opunServer, exists := claudeConfig.MCPServers["opun"]
	require.True(t, exists)
	assert.Equal(t, "opun", opunServer.Command)
	assert.Equal(t, []string{"mcp", "stdio"}, opunServer.Args)

	// Check filesystem server is not included (not installed and not required)
	_, exists = claudeConfig.MCPServers["filesystem"]
	require.False(t, exists)

	// Check memory server with environment
	memServer, exists := claudeConfig.MCPServers["memory"]
	require.True(t, exists)
	assert.Equal(t, "value", memServer.Env["KEY"])
}

func TestClaudeConfigTranslator_GetConfigPath(t *testing.T) {
	translator := NewClaudeConfigTranslator()
	path := translator.GetConfigPath()
	// Path will be expanded, so check it contains the expected path components
	// On macOS, it could be either the primary location or the fallback XDG location
	validPaths := []string{
		"Library/Application Support/Claude/claude_desktop_config.json",
		".config/claude/claude_desktop_config.json",
	}
	
	validPath := false
	for _, expectedPath := range validPaths {
		if strings.Contains(path, expectedPath) {
			validPath = true
			break
		}
	}
	assert.True(t, validPath, "Path should contain one of the valid Claude config paths, got: %s", path)
}

func TestClaudeConfigTranslator_TranslateSlashCommands(t *testing.T) {
	translator := NewClaudeConfigTranslator()

	// Claude doesn't support slash commands directly
	config, err := translator.TranslateSlashCommands([]core.SharedSlashCommand{
		{Name: "test", Description: "Test command"},
	})

	require.NoError(t, err)
	assert.Nil(t, config)
}
