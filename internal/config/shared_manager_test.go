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
	"time"

	"github.com/rizome-dev/opun/internal/utils"
	"github.com/rizome-dev/opun/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestNewSharedConfigManager(t *testing.T) {
	// Set up test home directory
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	t.Run("Creates Default Config", func(t *testing.T) {
		manager, err := NewSharedConfigManager()
		require.NoError(t, err)
		assert.NotNil(t, manager)

		// Check that config file was created
		configPath := filepath.Join(tempDir, ".opun", "shared-config.yaml")
		assert.FileExists(t, configPath)

		// Check default servers
		servers := manager.GetMCPServers()
		assert.Greater(t, len(servers), 0)

		// Check opun server exists
		hasOpun := false
		for _, server := range servers {
			if server.Name == "opun" {
				hasOpun = true
				assert.True(t, server.Required)
				assert.True(t, server.Installed)
				break
			}
		}
		assert.True(t, hasOpun)
	})

	t.Run("Loads Existing Config", func(t *testing.T) {
		// Create a config file
		configPath := filepath.Join(tempDir, ".opun", "shared-config.yaml")
		os.MkdirAll(filepath.Dir(configPath), 0755)

		config := &core.SharedConfig{
			Version: "1.0",
			MCPServers: []core.SharedMCPServer{
				{
					Name:    "test-server",
					Package: "test-package",
					Command: "test-cmd",
				},
			},
			LastUpdated: time.Now(),
		}

		data, err := yaml.Marshal(config)
		require.NoError(t, err)
		err = os.WriteFile(configPath, data, 0644)
		require.NoError(t, err)

		// Load it
		manager, err := NewSharedConfigManager()
		require.NoError(t, err)

		servers := manager.GetMCPServers()
		assert.Len(t, servers, 1)
		assert.Equal(t, "test-server", servers[0].Name)
	})
}

func TestSharedConfigManager_Save(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	manager, err := NewSharedConfigManager()
	require.NoError(t, err)

	// Add a new server
	err = manager.AddMCPServer(core.SharedMCPServer{
		Name:    "new-server",
		Package: "new-package",
		Command: "new-cmd",
	})
	require.NoError(t, err)

	// Reload and verify
	manager2, err := NewSharedConfigManager()
	require.NoError(t, err)

	found := false
	for _, server := range manager2.GetMCPServers() {
		if server.Name == "new-server" {
			found = true
			break
		}
	}
	assert.True(t, found)
}

func TestSharedConfigManager_UpdateMCPServerStatus(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	manager, err := NewSharedConfigManager()
	require.NoError(t, err)

	// Add a server
	err = manager.AddMCPServer(core.SharedMCPServer{
		Name:      "test-server",
		Package:   "test-package",
		Command:   "test-cmd",
		Installed: false,
	})
	require.NoError(t, err)

	// Update status
	err = manager.UpdateMCPServerStatus("test-server", true, "1.0.0")
	require.NoError(t, err)

	// Verify
	servers := manager.GetMCPServers()
	for _, server := range servers {
		if server.Name == "test-server" {
			assert.True(t, server.Installed)
			assert.Equal(t, "1.0.0", server.Version)
			break
		}
	}

	// Test updating non-existent server
	err = manager.UpdateMCPServerStatus("non-existent", true, "1.0.0")
	assert.Error(t, err)
}

func TestSharedConfigManager_AddSlashCommand(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	manager, err := NewSharedConfigManager()
	require.NoError(t, err)

	// Add a command
	err = manager.AddSlashCommand(core.SharedSlashCommand{
		Name:        "test-cmd",
		Description: "Test command",
		Type:        "prompt",
		Handler:     "test-handler",
	})
	require.NoError(t, err)

	// Verify
	commands := manager.GetSlashCommands()
	found := false
	for _, cmd := range commands {
		if cmd.Name == "test-cmd" {
			found = true
			assert.Equal(t, "Test command", cmd.Description)
			break
		}
	}
	assert.True(t, found)

	// Test updating existing command
	err = manager.AddSlashCommand(core.SharedSlashCommand{
		Name:        "test-cmd",
		Description: "Updated description",
		Type:        "prompt",
		Handler:     "test-handler",
	})
	require.NoError(t, err)

	// Verify update
	commands = manager.GetSlashCommands()
	for _, cmd := range commands {
		if cmd.Name == "test-cmd" {
			assert.Equal(t, "Updated description", cmd.Description)
			break
		}
	}
}

func TestSharedConfigManager_SyncToProvider(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	manager, err := NewSharedConfigManager()
	require.NoError(t, err)

	// Test syncing to Claude
	err = manager.SyncToProvider("claude")
	require.NoError(t, err)

	// Check that Claude config was created (could be in either location)
	claudeConfig1 := filepath.Join(tempDir, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	claudeConfig2 := filepath.Join(tempDir, ".config", "claude", "claude_desktop_config.json")

	// At least one should exist
	config1Exists := utils.FileExists(claudeConfig1)
	config2Exists := utils.FileExists(claudeConfig2)
	assert.True(t, config1Exists || config2Exists, "Claude config should exist in one of the expected locations")

	// Test syncing to Gemini
	err = manager.SyncToProvider("gemini")
	require.NoError(t, err)

	// Check that Gemini config was created
	geminiConfig := filepath.Join(tempDir, ".gemini", "settings.json")
	assert.FileExists(t, geminiConfig)

	// Test unsupported provider
	err = manager.SyncToProvider("unsupported")
	assert.Error(t, err)
}

func TestSharedConfigManager_CheckMCPServerInstalled(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Create a config with a properly configured opun server
	configPath := filepath.Join(tempDir, ".opun", "shared-config.yaml")
	os.MkdirAll(filepath.Dir(configPath), 0755)

	config := &core.SharedConfig{
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
				Version:     "1.0.0",
			},
		},
	}

	data, err := yaml.Marshal(config)
	require.NoError(t, err)
	err = os.WriteFile(configPath, data, 0644)
	require.NoError(t, err)

	manager, err := NewSharedConfigManager()
	require.NoError(t, err)

	// Check opun server - it's marked as installed with "builtin" path
	// The current implementation requires the file to exist, so it returns false
	installed, version, err := manager.CheckMCPServerInstalled("opun")
	require.NoError(t, err)
	assert.False(t, installed)
	assert.Empty(t, version)

	// Check non-existent server
	installed, version, err = manager.CheckMCPServerInstalled("non-existent")
	require.NoError(t, err)
	assert.False(t, installed)
	assert.Empty(t, version)
}

func TestSharedConfigManager_EnsureOpunServer(t *testing.T) {
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Create a config without opun server
	configPath := filepath.Join(tempDir, ".opun", "shared-config.yaml")
	os.MkdirAll(filepath.Dir(configPath), 0755)

	config := &core.SharedConfig{
		Version:    "1.0",
		MCPServers: []core.SharedMCPServer{},
	}

	data, err := yaml.Marshal(config)
	require.NoError(t, err)
	err = os.WriteFile(configPath, data, 0644)
	require.NoError(t, err)

	// Load and sync
	manager, err := NewSharedConfigManager()
	require.NoError(t, err)

	err = manager.SyncToProvider("claude")
	require.NoError(t, err)

	// Verify opun server was added
	hasOpun := false
	for _, server := range manager.GetMCPServers() {
		if server.Name == "opun" {
			hasOpun = true
			break
		}
	}
	assert.True(t, hasOpun)
}
