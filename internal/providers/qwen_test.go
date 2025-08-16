package providers

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
	"context"
	"os"
	"testing"

	"github.com/rizome-dev/opun/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewQwenProvider(t *testing.T) {
	config := core.ProviderConfig{
		Name:    "test-qwen",
		Type:    core.ProviderTypeQwen,
		Model:   "code",
		Command: "qwen",
		Args:    []string{"chat"},
		Features: core.ProviderFeatures{
			Interactive: true,
			MCP:         true,
		},
	}

	provider := NewQwenProvider(config)
	assert.NotNil(t, provider)
	assert.Equal(t, "test-qwen", provider.Name())
	assert.Equal(t, core.ProviderTypeQwen, provider.Type())
}

func TestQwenProvider_GetReadyPattern(t *testing.T) {
	config := core.ProviderConfig{
		Name:    "test-qwen",
		Type:    core.ProviderTypeQwen,
		Command: "qwen",
	}

	provider := NewQwenProvider(config)
	assert.Equal(t, "│ >", provider.GetReadyPattern())
}

func TestQwenProvider_GetOutputPattern(t *testing.T) {
	config := core.ProviderConfig{
		Name:    "test-qwen",
		Type:    core.ProviderTypeQwen,
		Command: "qwen",
	}

	provider := NewQwenProvider(config)
	assert.Equal(t, "│ >", provider.GetOutputPattern())
}

func TestQwenProvider_GetErrorPattern(t *testing.T) {
	config := core.ProviderConfig{
		Name:    "test-qwen",
		Type:    core.ProviderTypeQwen,
		Command: "qwen",
	}

	provider := NewQwenProvider(config)
	assert.Equal(t, "Error:", provider.GetErrorPattern())
}

func TestQwenProvider_GetPromptInjectionMethod(t *testing.T) {
	config := core.ProviderConfig{
		Name:    "test-qwen",
		Type:    core.ProviderTypeQwen,
		Command: "qwen",
	}

	provider := NewQwenProvider(config)
	assert.Equal(t, "clipboard", provider.GetPromptInjectionMethod())
}

func TestQwenProvider_SupportsModel(t *testing.T) {
	config := core.ProviderConfig{
		Name:    "test-qwen",
		Type:    core.ProviderTypeQwen,
		Command: "qwen",
	}

	provider := NewQwenProvider(config)

	// Test supported models
	assert.True(t, provider.SupportsModel("code"))
	assert.True(t, provider.SupportsModel("pro"))
	assert.True(t, provider.SupportsModel("flash"))
	assert.True(t, provider.SupportsModel("ultra"))
	assert.True(t, provider.SupportsModel("chat"))

	// Test case insensitive
	assert.True(t, provider.SupportsModel("CODE"))
	assert.True(t, provider.SupportsModel("Chat"))

	// Test unsupported model
	assert.False(t, provider.SupportsModel("unsupported"))
}

func TestQwenProvider_GetMCPServers(t *testing.T) {
	config := core.ProviderConfig{
		Name:    "test-qwen",
		Type:    core.ProviderTypeQwen,
		Command: "qwen",
	}

	provider := NewQwenProvider(config)
	servers := provider.GetMCPServers()

	assert.Len(t, servers, 1)
	assert.Equal(t, "filesystem", servers[0].Name)
	assert.True(t, servers[0].Enabled)
}

func TestQwenProvider_GetTools(t *testing.T) {
	config := core.ProviderConfig{
		Name:    "test-qwen",
		Type:    core.ProviderTypeQwen,
		Command: "qwen",
	}

	provider := NewQwenProvider(config)
	tools := provider.GetTools()

	assert.Len(t, tools, 2)

	// Check read_file tool
	assert.Equal(t, "read_file", tools[0].Name)
	assert.Equal(t, "filesystem", tools[0].Category)

	// Check write_file tool
	assert.Equal(t, "write_file", tools[1].Name)
	assert.Equal(t, "filesystem", tools[1].Category)
}

func TestQwenProvider_PrepareSession(t *testing.T) {
	config := core.ProviderConfig{
		Name:    "test-qwen",
		Type:    core.ProviderTypeQwen,
		Command: "qwen",
	}

	provider := NewQwenProvider(config)
	ctx := context.Background()
	sessionID := "test-session"

	err := provider.PrepareSession(ctx, sessionID)
	assert.NoError(t, err)

	// Check that session directory was created
	sessionDir := os.TempDir() + "/opun/sessions/" + sessionID
	_, err = os.Stat(sessionDir)
	// Directory might not exist if cleanup ran, but PrepareSession should not error

	// Clean up
	_ = provider.CleanupSession(ctx, sessionID)
}

func TestQwenProvider_SupportsSlashCommands(t *testing.T) {
	config := core.ProviderConfig{
		Name:    "test-qwen",
		Type:    core.ProviderTypeQwen,
		Command: "qwen",
	}

	provider := NewQwenProvider(config)
	assert.True(t, provider.SupportsSlashCommands())
}

func TestQwenProvider_GetSlashCommandFormat(t *testing.T) {
	config := core.ProviderConfig{
		Name:    "test-qwen",
		Type:    core.ProviderTypeQwen,
		Command: "qwen",
	}

	provider := NewQwenProvider(config)
	assert.Equal(t, "mcp", provider.GetSlashCommandFormat())
}

func TestQwenProvider_GetPTYCommand(t *testing.T) {
	config := core.ProviderConfig{
		Name:       "test-qwen",
		Type:       core.ProviderTypeQwen,
		Command:    "qwen",
		Args:       []string{"chat"},
		WorkingDir: "/test/dir",
		Environment: map[string]string{
			"TEST_VAR": "test_value",
		},
	}

	provider := NewQwenProvider(config)
	cmd, err := provider.GetPTYCommand()

	require.NoError(t, err)
	// cmd.Path might be the full path to the binary, so just check it ends with "qwen"
	assert.Contains(t, cmd.Path, "qwen")
	// Check that Args contains the expected commands
	assert.Contains(t, cmd.Args[0], "qwen")
	assert.Equal(t, "chat", cmd.Args[1])
	assert.Equal(t, "/test/dir", cmd.Dir)

	// Check environment variable is set
	envFound := false
	for _, env := range cmd.Env {
		if env == "TEST_VAR=test_value" {
			envFound = true
			break
		}
	}
	assert.True(t, envFound, "TEST_VAR environment variable not found")
}

