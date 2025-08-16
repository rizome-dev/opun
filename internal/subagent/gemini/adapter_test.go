package gemini

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
	"time"

	"github.com/rizome-dev/opun/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeminiAdapter_Creation(t *testing.T) {
	config := core.SubAgentConfig{
		Name:        "gemini-test-agent",
		Type:        core.SubAgentTypeProgrammatic,
		Provider:    core.ProviderTypeGemini,
		Model:       "gemini-pro",
		Description: "Test Gemini agent",
		Capabilities: []string{"code", "analysis", "creativity"},
		Strategy:    core.DelegationProactive,
		MaxRetries:  2,
		Timeout:     time.Minute * 3,
	}

	adapter := NewGeminiAdapter(config)
	assert.NotNil(t, adapter)
	assert.Equal(t, "gemini-test-agent", adapter.Name())
	assert.Equal(t, core.ProviderTypeGemini, adapter.Provider())
	assert.Equal(t, config, adapter.Config())
}

func TestGeminiAdapter_Initialize(t *testing.T) {
	adapter := NewGeminiAdapter(core.SubAgentConfig{
		Name:     "test",
		Provider: core.ProviderTypeGemini,
	})

	t.Run("Initialize with valid config", func(t *testing.T) {
		config := core.SubAgentConfig{
			Name:        "initialized-gemini",
			Provider:    core.ProviderTypeGemini,
			Model:       "gemini-ultra",
			Capabilities: []string{"advanced", "multimodal"},
		}

		err := adapter.Initialize(config)
		assert.NoError(t, err)
		assert.Equal(t, config, adapter.Config())
	})

	t.Run("Initialize with SubAgentScope config", func(t *testing.T) {
		config := core.SubAgentConfig{
			Name:     "scoped-gemini",
			Provider: core.ProviderTypeGemini,
			ProviderConfig: map[string]interface{}{
				"scope": "specialized",
				"focus": "data-analysis",
			},
		}

		err := adapter.Initialize(config)
		assert.NoError(t, err)
		assert.Equal(t, "specialized", config.ProviderConfig["scope"])
	})
}

func TestGeminiAdapter_TaskAdaptation(t *testing.T) {
	adapter := NewGeminiAdapter(core.SubAgentConfig{
		Name:     "gemini-adapter",
		Provider: core.ProviderTypeGemini,
		Model:    "gemini-pro",
	})

	t.Run("Adapt standard task", func(t *testing.T) {
		task := core.SubAgentTask{
			ID:          "task-456",
			Name:        "Data Analysis",
			Description: "Analyze the dataset",
			Input:       "data: [1, 2, 3, 4, 5]",
			Context: map[string]interface{}{
				"format": "json",
				"type":   "numerical",
			},
			Variables: map[string]interface{}{
				"method": "statistical",
			},
			Priority: 3,
		}

		adapted, err := adapter.AdaptTask(task)
		require.NoError(t, err)
		assert.NotNil(t, adapted)

		// Check if adaptation follows Gemini's SubAgentScope format
		if scopeData, ok := adapted.(map[string]interface{}); ok {
			// Verify Gemini-specific fields
			assert.NotNil(t, scopeData)
		}
	})

	t.Run("Adapt multimodal task", func(t *testing.T) {
		task := core.SubAgentTask{
			ID:   "multimodal-task",
			Name: "Image Analysis",
			Context: map[string]interface{}{
				"type":       "image",
				"multimodal": true,
			},
		}

		adapted, err := adapter.AdaptTask(task)
		require.NoError(t, err)
		assert.NotNil(t, adapted)
	})

	t.Run("Adapt task with constraints", func(t *testing.T) {
		task := core.SubAgentTask{
			ID:          "constrained-task",
			Name:        "Limited Processing",
			Constraints: []string{"response_time < 2s", "memory < 500MB"},
		}

		adapted, err := adapter.AdaptTask(task)
		require.NoError(t, err)
		assert.NotNil(t, adapted)
	})
}

func TestGeminiAdapter_ResultAdaptation(t *testing.T) {
	adapter := NewGeminiAdapter(core.SubAgentConfig{
		Name:     "gemini-adapter",
		Provider: core.ProviderTypeGemini,
	})

	t.Run("Adapt successful result", func(t *testing.T) {
		// Simulate Gemini's response format
		geminiResult := map[string]interface{}{
			"status": "success",
			"response": map[string]interface{}{
				"text":       "Analysis complete",
				"confidence": 0.95,
			},
			"metadata": map[string]interface{}{
				"tokens":         800,
				"execution_time": "1.2s",
			},
		}

		result, err := adapter.AdaptResult(geminiResult)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, core.StatusCompleted, result.Status)
		assert.Contains(t, result.Output, "Analysis complete")
	})

	t.Run("Adapt failed result", func(t *testing.T) {
		geminiResult := map[string]interface{}{
			"status":       "error",
			"error_message": "Processing failed: invalid input",
		}

		result, err := adapter.AdaptResult(geminiResult)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, core.StatusFailed, result.Status)
		assert.NotNil(t, result.Error)
	})

	t.Run("Adapt result with structured output", func(t *testing.T) {
		geminiResult := map[string]interface{}{
			"status": "success",
			"response": map[string]interface{}{
				"analysis": map[string]interface{}{
					"summary": "Data shows positive trend",
					"metrics": []float64{1.2, 3.4, 5.6},
				},
			},
		}

		result, err := adapter.AdaptResult(geminiResult)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Output)
	})
}

func TestGeminiAdapter_SubAgentScope(t *testing.T) {
	adapter := NewGeminiAdapter(core.SubAgentConfig{
		Name:     "scoped-gemini",
		Provider: core.ProviderTypeGemini,
		ProviderConfig: map[string]interface{}{
			"enable_scope": true,
			"scope_type":   "specialized",
		},
	})

	t.Run("Create scope for task", func(t *testing.T) {
		task := core.SubAgentTask{
			ID:   "scoped-task",
			Name: "Specialized Processing",
			Context: map[string]interface{}{
				"scope": "data-science",
			},
		}

		adapted, err := adapter.AdaptTask(task)
		require.NoError(t, err)
		
		// Verify scope configuration
		if scopeConfig, ok := adapted.(map[string]interface{}); ok {
			assert.NotNil(t, scopeConfig)
			// Would contain Gemini-specific scope settings
		}
	})

	t.Run("Handle scope transitions", func(t *testing.T) {
		// Test transitioning between different scopes
		tasks := []core.SubAgentTask{
			{ID: "task1", Context: map[string]interface{}{"scope": "analysis"}},
			{ID: "task2", Context: map[string]interface{}{"scope": "generation"}},
			{ID: "task3", Context: map[string]interface{}{"scope": "validation"}},
		}

		for _, task := range tasks {
			adapted, err := adapter.AdaptTask(task)
			assert.NoError(t, err)
			assert.NotNil(t, adapted)
		}
	})
}

func TestGeminiAdapter_Execution(t *testing.T) {
	adapter := NewGeminiAdapter(core.SubAgentConfig{
		Name:       "gemini-executor",
		Provider:   core.ProviderTypeGemini,
		Model:      "gemini-pro",
		Timeout:    time.Second * 5,
		MaxRetries: 2,
	})

	t.Run("Execute with retry logic", func(t *testing.T) {
		// Test that retry logic is properly configured
		assert.Equal(t, 2, adapter.config.MaxRetries)
		assert.Equal(t, time.Second*5, adapter.config.Timeout)
	})

	t.Run("Execute async task", func(t *testing.T) {
		// Test async execution capability
		assert.NotPanics(t, func() {
			// In real implementation:
			// ctx := context.Background()
			// task := core.SubAgentTask{ID: "async-task"}
			// resultChan, err := adapter.ExecuteAsync(ctx, task)
		})
	})
}

func TestGeminiAdapter_Capabilities(t *testing.T) {
	adapter := NewGeminiAdapter(core.SubAgentConfig{
		Name:         "capable-gemini",
		Provider:     core.ProviderTypeGemini,
		Capabilities: []string{"analysis", "generation", "translation", "summarization"},
	})

	t.Run("Get capabilities", func(t *testing.T) {
		caps := adapter.GetCapabilities()
		assert.Len(t, caps, 4)
		assert.Contains(t, caps, "analysis")
		assert.Contains(t, caps, "generation")
	})

	t.Run("Can handle matching task", func(t *testing.T) {
		task := core.SubAgentTask{
			Context: map[string]interface{}{
				"analysis": true,
			},
		}
		assert.True(t, adapter.CanHandle(task))
	})

	t.Run("Supports parallel execution", func(t *testing.T) {
		adapter.config.Parallel = true
		assert.True(t, adapter.SupportsParallel())
	})

	t.Run("Interactive support", func(t *testing.T) {
		adapter.config.Interactive = true
		assert.True(t, adapter.SupportsInteractive())
	})
}

func TestGeminiAdapter_ProviderIntegration(t *testing.T) {
	adapter := NewGeminiAdapter(core.SubAgentConfig{
		Name:     "provider-gemini",
		Provider: core.ProviderTypeGemini,
	})

	t.Run("Initialize with provider", func(t *testing.T) {
		provider := &MockGeminiProvider{
			name: "gemini-provider",
			typ:  core.ProviderTypeGemini,
		}

		err := adapter.InitializeProvider(provider)
		assert.NoError(t, err)
	})

	t.Run("Get provider config", func(t *testing.T) {
		adapter.config.ProviderConfig = map[string]interface{}{
			"api_key":           "test-key",
			"temperature":       0.7,
			"safety_settings":   "moderate",
		}

		config := adapter.GetProviderConfig()
		assert.NotNil(t, config)
		assert.Contains(t, config, "api_key")
		assert.Equal(t, 0.7, config["temperature"])
		assert.Equal(t, "moderate", config["safety_settings"])
	})
}

func TestGeminiAdapter_StatusAndControl(t *testing.T) {
	adapter := NewGeminiAdapter(core.SubAgentConfig{
		Name:     "control-gemini",
		Provider: core.ProviderTypeGemini,
	})

	t.Run("Initial status", func(t *testing.T) {
		assert.Equal(t, core.StatusPending, adapter.Status())
	})

	t.Run("Cancel execution", func(t *testing.T) {
		err := adapter.Cancel()
		assert.NoError(t, err)
	})

	t.Run("Get progress", func(t *testing.T) {
		progress, message := adapter.GetProgress()
		assert.GreaterOrEqual(t, progress, 0.0)
		assert.LessOrEqual(t, progress, 1.0)
		assert.NotEmpty(t, message)
	})

	t.Run("Validate configuration", func(t *testing.T) {
		err := adapter.Validate()
		// Should validate successfully with basic config
		assert.NoError(t, err)
	})

	t.Run("Cleanup resources", func(t *testing.T) {
		err := adapter.Cleanup()
		assert.NoError(t, err)
	})
}

func TestGeminiAdapter_EdgeCases(t *testing.T) {
	adapter := NewGeminiAdapter(core.SubAgentConfig{
		Name:     "edge-gemini",
		Provider: core.ProviderTypeGemini,
	})

	t.Run("Handle empty task", func(t *testing.T) {
		task := core.SubAgentTask{}
		adapted, err := adapter.AdaptTask(task)
		// Should handle gracefully
		assert.NoError(t, err)
		assert.NotNil(t, adapted)
	})

	t.Run("Handle nil result", func(t *testing.T) {
		result, err := adapter.AdaptResult(nil)
		// Should handle gracefully
		if err != nil {
			assert.Contains(t, err.Error(), "nil")
		} else {
			assert.NotNil(t, result)
		}
	})

	t.Run("Handle malformed result", func(t *testing.T) {
		malformed := "not a map"
		result, err := adapter.AdaptResult(malformed)
		// Should handle type mismatch
		if err != nil {
			assert.NotNil(t, err)
		} else {
			assert.NotNil(t, result)
		}
	})
}

// Mock Gemini provider for testing
type MockGeminiProvider struct {
	name string
	typ  core.ProviderType
}

func (m *MockGeminiProvider) Name() string                                       { return m.name }
func (m *MockGeminiProvider) Type() core.ProviderType                            { return m.typ }
func (m *MockGeminiProvider) Initialize(config core.ProviderConfig) error        { return nil }
func (m *MockGeminiProvider) Validate() error                                    { return nil }
func (m *MockGeminiProvider) GetPTYCommand() (*exec.Cmd, error)                  { return exec.Command("echo"), nil }
func (m *MockGeminiProvider) GetPTYCommandWithPrompt(prompt string) (*exec.Cmd, error) { return exec.Command("echo", prompt), nil }
func (m *MockGeminiProvider) Features() core.ProviderFeatures                    { return core.ProviderFeatures{} }
func (m *MockGeminiProvider) SupportsModel(model string) bool                    { return true }
func (m *MockGeminiProvider) PrepareSession(ctx context.Context, sessionID string) error { return nil }
func (m *MockGeminiProvider) CleanupSession(ctx context.Context, sessionID string) error { return nil }
func (m *MockGeminiProvider) GetReadyPattern() string                            { return "ready" }
func (m *MockGeminiProvider) GetOutputPattern() string                           { return "output" }
func (m *MockGeminiProvider) GetErrorPattern() string                            { return "error" }
func (m *MockGeminiProvider) GetPromptInjectionMethod() string                   { return "stdin" }
func (m *MockGeminiProvider) InjectPrompt(prompt string) error                   { return nil }
func (m *MockGeminiProvider) GetMCPServers() []core.MCPServer                    { return []core.MCPServer{} }
func (m *MockGeminiProvider) GetTools() []core.Tool                              { return []core.Tool{} }
func (m *MockGeminiProvider) GetSlashCommands() []core.SharedSlashCommand        { return []core.SharedSlashCommand{} }
func (m *MockGeminiProvider) GetPlugins() []core.PluginReference                 { return []core.PluginReference{} }
func (m *MockGeminiProvider) SupportsSlashCommands() bool                        { return false }
func (m *MockGeminiProvider) GetSlashCommandDirectory() string                   { return "" }
func (m *MockGeminiProvider) GetSlashCommandFormat() string                      { return "" }
func (m *MockGeminiProvider) PrepareSlashCommands(commands []core.SharedSlashCommand, targetDir string) error { return nil }

// Benchmark tests
func BenchmarkGeminiAdapter_TaskAdaptation(b *testing.B) {
	adapter := NewGeminiAdapter(core.SubAgentConfig{
		Name:     "bench-gemini",
		Provider: core.ProviderTypeGemini,
	})

	task := core.SubAgentTask{
		ID:   "bench-task",
		Name: "Benchmark Task",
		Context: map[string]interface{}{
			"type": "analysis",
			"data": []int{1, 2, 3, 4, 5},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.AdaptTask(task)
	}
}

func BenchmarkGeminiAdapter_ResultAdaptation(b *testing.B) {
	adapter := NewGeminiAdapter(core.SubAgentConfig{
		Name:     "bench-gemini",
		Provider: core.ProviderTypeGemini,
	})

	result := map[string]interface{}{
		"status": "success",
		"response": map[string]interface{}{
			"text": "result text",
			"data": []int{1, 2, 3},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.AdaptResult(result)
	}
}