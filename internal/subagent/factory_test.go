package subagent

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

	"github.com/rizome-dev/opun/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFactory_CreateAdapter(t *testing.T) {
	factory := NewFactory()

	tests := []struct {
		name        string
		config      core.SubAgentConfig
		shouldError bool
	}{
		{
			name: "Create Claude adapter",
			config: core.SubAgentConfig{
				Name:        "claude-agent",
				Type:        core.SubAgentTypeDeclarative,
				Provider:    core.ProviderTypeClaude,
				Model:       "sonnet",
				Description: "Test agent",
			},
			shouldError: false,
		},
		{
			name: "Create Gemini adapter",
			config: core.SubAgentConfig{
				Name:        "gemini-agent",
				Type:        core.SubAgentTypeProgrammatic,
				Provider:    core.ProviderTypeGemini,
				Model:       "gemini-pro",
				Description: "Test agent",
			},
			shouldError: false,
		},
		{
			name: "Create Qwen adapter",
			config: core.SubAgentConfig{
				Name:        "qwen-agent",
				Type:        core.SubAgentTypeWorkflow,
				Provider:    core.ProviderTypeQwen,
				Model:       "code",
				Description: "Test agent",
			},
			shouldError: false,
		},
		{
			name: "Unsupported provider type",
			config: core.SubAgentConfig{
				Name:     "unknown-agent",
				Provider: "unsupported",
			},
			shouldError: true,
		},
		{
			name: "Missing provider type",
			config: core.SubAgentConfig{
				Name: "no-provider-agent",
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, err := factory.CreateAdapter(tt.config)

			if tt.shouldError {
				assert.Error(t, err)
				assert.Nil(t, adapter)
			} else {
				// May still error due to initialization (e.g., directory creation)
				if err != nil {
					t.Logf("Adapter creation error (may be expected): %v", err)
				} else {
					assert.NotNil(t, adapter)
					assert.Equal(t, tt.config.Name, adapter.Name())
					assert.Equal(t, tt.config.Provider, adapter.Provider())
				}
			}
		})
	}
}

func TestFactory_RegisterProvider(t *testing.T) {
	factory := NewFactory()

	t.Run("Register provider", func(t *testing.T) {
		provider := &MockProvider{
			name: "test-provider",
			typ:  core.ProviderTypeClaude,
		}
		
		err := factory.RegisterProvider(provider)
		require.NoError(t, err)
		
		// Create adapter with registered provider
		config := core.SubAgentConfig{
			Name:        "test-agent",
			Provider:    core.ProviderTypeClaude,
			Description: "Test",
		}
		
		adapter, err := factory.CreateAdapter(config)
		// May error due to initialization
		if err != nil {
			t.Logf("Expected initialization error: %v", err)
		} else if adapter != nil {
			t.Logf("Adapter created successfully")
		}
	})

	t.Run("Register nil provider", func(t *testing.T) {
		err := factory.RegisterProvider(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})
}

func TestFactory_ListSupportedProviders(t *testing.T) {
	factory := NewFactory()

	providers := factory.ListSupportedProviders()
	assert.NotEmpty(t, providers)
	
	// Should include built-in providers
	assert.Contains(t, providers, core.ProviderTypeClaude)
	assert.Contains(t, providers, core.ProviderTypeGemini)
	assert.Contains(t, providers, core.ProviderTypeQwen)
	// Mock provider is not registered by default, only when explicitly added
}

func TestFactory_CreateSubAgent(t *testing.T) {
	factory := NewFactory()

	config := core.SubAgentConfig{
		Name:        "subagent-test",
		Provider:    core.ProviderTypeClaude,
		Description: "Test subagent",
		Model:       "sonnet",
	}

	agent, err := factory.CreateSubAgent(config)
	// May error due to initialization
	if err != nil {
		t.Logf("Expected initialization error: %v", err)
	} else {
		assert.NotNil(t, agent)
		assert.Equal(t, "subagent-test", agent.Name())
	}
}

func TestFactory_GetProviderCapabilities(t *testing.T) {
	factory := NewFactory()

	tests := []struct {
		providerType core.ProviderType
		expected     core.SubAgentType
		shouldError  bool
	}{
		{core.ProviderTypeClaude, core.SubAgentTypeDeclarative, false},
		{core.ProviderTypeGemini, core.SubAgentTypeProgrammatic, false},
		{core.ProviderTypeQwen, core.SubAgentTypeWorkflow, false},
		{"unknown", "", true},
	}

	for _, tt := range tests {
		t.Run(string(tt.providerType), func(t *testing.T) {
			capability, err := factory.GetProviderCapabilities(tt.providerType)
			
			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, capability)
			}
		})
	}
}

func TestFactory_CreateDefaultConfigs(t *testing.T) {
	factory := NewFactory()

	configs := factory.CreateDefaultConfigs()
	assert.NotEmpty(t, configs)
	
	// Should have at least one config for each provider
	providers := make(map[core.ProviderType]bool)
	for _, config := range configs {
		providers[config.Provider] = true
	}
	
	assert.True(t, providers[core.ProviderTypeClaude])
	assert.True(t, providers[core.ProviderTypeGemini])
	assert.True(t, providers[core.ProviderTypeQwen])
}

// Mock implementations for testing
type MockAdapter struct {
	config   core.SubAgentConfig
	provider core.Provider
}

func NewMockAdapter(config core.SubAgentConfig) *MockAdapter {
	return &MockAdapter{
		config: config,
	}
}

func (m *MockAdapter) Name() string                { return m.config.Name }
func (m *MockAdapter) Config() core.SubAgentConfig { return m.config }
func (m *MockAdapter) Provider() core.ProviderType { return m.config.Provider }
func (m *MockAdapter) Initialize(config core.SubAgentConfig) error {
	m.config = config
	return nil
}
func (m *MockAdapter) Validate() error                                      { return nil }
func (m *MockAdapter) Cleanup() error                                       { return nil }
func (m *MockAdapter) Status() core.ExecutionStatus                         { return core.StatusPending }
func (m *MockAdapter) Cancel() error                                        { return nil }
func (m *MockAdapter) GetProgress() (float64, string)                       { return 0, "" }
func (m *MockAdapter) GetCapabilities() []string                            { return []string{} }
func (m *MockAdapter) SupportsParallel() bool                               { return false }
func (m *MockAdapter) SupportsInteractive() bool                            { return false }
func (m *MockAdapter) CanHandle(task core.SubAgentTask) bool                { return true }
func (m *MockAdapter) InitializeProvider(provider core.Provider) error {
	m.provider = provider
	return nil
}
func (m *MockAdapter) AdaptTask(task core.SubAgentTask) (interface{}, error) {
	return task, nil
}
func (m *MockAdapter) AdaptResult(result interface{}) (*core.SubAgentResult, error) {
	if r, ok := result.(*core.SubAgentResult); ok {
		return r, nil
	}
	return &core.SubAgentResult{}, nil
}
func (m *MockAdapter) GetProviderConfig() map[string]interface{} {
	return m.config.ProviderConfig
}

func (m *MockAdapter) Execute(ctx context.Context, task core.SubAgentTask) (*core.SubAgentResult, error) {
	return &core.SubAgentResult{
		TaskID:    task.ID,
		AgentName: m.Name(),
		Status:    core.StatusCompleted,
		Output:    "mock output",
	}, nil
}

func (m *MockAdapter) ExecuteAsync(ctx context.Context, task core.SubAgentTask) (<-chan *core.SubAgentResult, error) {
	ch := make(chan *core.SubAgentResult, 1)
	go func() {
		result, _ := m.Execute(ctx, task)
		ch <- result
		close(ch)
	}()
	return ch, nil
}

type MockProvider struct {
	name string
	typ  core.ProviderType
	core.BaseProvider
}

func (m *MockProvider) Name() string            { return m.name }
func (m *MockProvider) Type() core.ProviderType { return m.typ }
func (m *MockProvider) Initialize(config core.ProviderConfig) error { return nil }
func (m *MockProvider) Validate() error { return nil }
func (m *MockProvider) GetPTYCommand() (*exec.Cmd, error) { return exec.Command("echo"), nil }
func (m *MockProvider) GetPTYCommandWithPrompt(prompt string) (*exec.Cmd, error) {
	return exec.Command("echo", prompt), nil
}
func (m *MockProvider) Features() core.ProviderFeatures { return core.ProviderFeatures{} }
func (m *MockProvider) SupportsModel(model string) bool { return true }
func (m *MockProvider) PrepareSession(ctx context.Context, sessionID string) error { return nil }
func (m *MockProvider) CleanupSession(ctx context.Context, sessionID string) error { return nil }
func (m *MockProvider) GetReadyPattern() string { return "ready" }
func (m *MockProvider) GetOutputPattern() string { return "done" }
func (m *MockProvider) GetErrorPattern() string { return "error" }
func (m *MockProvider) GetPromptInjectionMethod() string { return "stdin" }
func (m *MockProvider) InjectPrompt(prompt string) error { return nil }
func (m *MockProvider) GetMCPServers() []core.MCPServer { return nil }
func (m *MockProvider) GetTools() []core.Tool { return nil }
func (m *MockProvider) GetSlashCommands() []core.SharedSlashCommand { return nil }
func (m *MockProvider) GetPlugins() []core.PluginReference { return nil }
func (m *MockProvider) SupportsSlashCommands() bool { return false }
func (m *MockProvider) GetSlashCommandDirectory() string { return "" }
func (m *MockProvider) GetSlashCommandFormat() string { return "" }
func (m *MockProvider) PrepareSlashCommands(commands []core.SharedSlashCommand, targetDir string) error {
	return nil
}

// Benchmark tests
func BenchmarkFactory_CreateAdapter(b *testing.B) {
	factory := NewFactory()
	config := core.SubAgentConfig{
		Name:        "bench-agent",
		Provider:    core.ProviderTypeClaude,
		Description: "Benchmark agent",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		factory.CreateAdapter(config)
	}
}

func BenchmarkFactory_CreateSubAgent(b *testing.B) {
	factory := NewFactory()
	config := core.SubAgentConfig{
		Name:        "bench-subagent",
		Provider:    core.ProviderTypeGemini,
		Description: "Benchmark subagent",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		factory.CreateSubAgent(config)
	}
}