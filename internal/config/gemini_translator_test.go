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
	"testing"

	"github.com/rizome-dev/opun/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiConfigTranslator_TranslateMCPConfig(t *testing.T) {
	// Set test home directory
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	translator := NewGeminiConfigTranslator()

	servers := []core.SharedMCPServer{
		{
			Name:      "opun",
			Command:   "opun",
			Args:      []string{"mcp", "stdio"},
			Installed: true,
			Required:  true,
		},
		{
			Name:      "filesystem",
			Command:   "npx",
			Args:      []string{"@modelcontextprotocol/server-filesystem", "/path"},
			Installed: false,
		},
		{
			Name:     "context7",
			Command:  "npx",
			Args:     []string{"@upstash/context7-mcp"},
			Env:      map[string]string{"API_KEY": "secret"},
			Required: true,
		},
	}

	config, err := translator.TranslateMCPConfig(servers)
	require.NoError(t, err)

	geminiConfig, ok := config.(GeminiConfig)
	require.True(t, ok)

	// Check mcpServers - only installed or required servers are included
	assert.Len(t, geminiConfig.MCPServers, 2)

	// Check opun server
	opunServer, exists := geminiConfig.MCPServers["opun"]
	require.True(t, exists)
	assert.Equal(t, "opun", opunServer.Command)
	assert.Equal(t, []string{"mcp", "stdio"}, opunServer.Args)

	// Check filesystem server is not included (not installed and not required)
	_, exists = geminiConfig.MCPServers["filesystem"]
	require.False(t, exists)

	// Check context7 server with environment
	ctx7Server, exists := geminiConfig.MCPServers["context7"]
	require.True(t, exists)
	assert.Equal(t, "secret", ctx7Server.Env["API_KEY"])
}

func TestGeminiConfigTranslator_GetConfigPath(t *testing.T) {
	translator := NewGeminiConfigTranslator()
	path := translator.GetConfigPath()
	assert.Equal(t, filepath.Join(os.Getenv("HOME"), ".gemini", "settings.json"), path)
}

func TestGeminiConfigTranslator_TranslateSlashCommands(t *testing.T) {
	translator := NewGeminiConfigTranslator()

	// Gemini doesn't support slash commands directly
	config, err := translator.TranslateSlashCommands([]core.SharedSlashCommand{
		{Name: "test", Description: "Test command"},
	})

	require.NoError(t, err)
	assert.Nil(t, config)
}
