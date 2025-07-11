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
	"context"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProviderRegistry(t *testing.T) {
	registry := NewProviderRegistry()

	t.Run("Register and Get Provider", func(t *testing.T) {
		// Create a mock provider
		mockProvider := &MockProvider{
			name: "TestProvider",
			typ:  ProviderTypeClaude,
		}

		// Register provider
		err := registry.Register(mockProvider.Type(), mockProvider)
		require.NoError(t, err)

		// Get provider
		provider, exists := registry.Get(ProviderTypeClaude)
		assert.True(t, exists)
		assert.Equal(t, "TestProvider", provider.Name())
		assert.Equal(t, ProviderTypeClaude, provider.Type())
	})

	t.Run("Get Non-existent Provider", func(t *testing.T) {
		provider, exists := registry.Get(ProviderTypeGemini)
		assert.False(t, exists)
		assert.Nil(t, provider)
	})

	t.Run("List Providers", func(t *testing.T) {
		types := registry.List()
		assert.Contains(t, types, ProviderTypeClaude)
	})
}

func TestBaseProvider(t *testing.T) {
	t.Run("Initialize", func(t *testing.T) {
		provider := NewBaseProvider("TestProvider", ProviderTypeClaude)

		config := ProviderConfig{
			Type:    ProviderTypeClaude,
			Command: "claude",
			Features: ProviderFeatures{
				Interactive: true,
				Batch:       true,
			},
		}

		err := provider.Initialize(config)
		assert.NoError(t, err)

		assert.Equal(t, "TestProvider", provider.Name())
		assert.Equal(t, ProviderTypeClaude, provider.Type())
		assert.True(t, provider.Features().Interactive)
		assert.True(t, provider.Features().Batch)
	})

	t.Run("Validate", func(t *testing.T) {
		provider := NewBaseProvider("TestProvider", ProviderTypeClaude)

		// Should fail without initialization
		err := provider.Validate()
		assert.Error(t, err)
		assert.Equal(t, ErrInvalidConfig, err)

		// Initialize with valid config
		provider.Initialize(ProviderConfig{
			Command: "claude",
		})

		err = provider.Validate()
		assert.NoError(t, err)
	})
}

// MockProvider implements Provider interface for testing
type MockProvider struct {
	name string
	typ  ProviderType
	BaseProvider
}

func (m *MockProvider) Name() string {
	return m.name
}

func (m *MockProvider) Type() ProviderType {
	return m.typ
}

func (m *MockProvider) GetPTYCommand() (*exec.Cmd, error) {
	return exec.Command("echo", "test"), nil
}

func (m *MockProvider) GetPTYCommandWithPrompt(prompt string) (*exec.Cmd, error) {
	return exec.Command("echo", prompt), nil
}

func (m *MockProvider) PrepareSession(ctx context.Context, sessionID string) error {
	return nil
}

func (m *MockProvider) CleanupSession(ctx context.Context, sessionID string) error {
	return nil
}

func (m *MockProvider) GetReadyPattern() string {
	return "ready"
}

func (m *MockProvider) GetOutputPattern() string {
	return "done"
}

func (m *MockProvider) GetErrorPattern() string {
	return "error"
}

func (m *MockProvider) GetPromptInjectionMethod() string {
	return "stdin"
}

func (m *MockProvider) SupportsModel(model string) bool {
	return model == "test-model"
}

func (m *MockProvider) InjectPrompt(prompt string) error {
	return nil
}

func (m *MockProvider) GetMCPServers() []MCPServer {
	return []MCPServer{}
}

func (m *MockProvider) GetTools() []Tool {
	return []Tool{}
}
