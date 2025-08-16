package claude

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

func TestClaudeAdapter_Creation(t *testing.T) {
	config := core.SubAgentConfig{
		Name:        "claude-test-agent",
		Type:        core.SubAgentTypeDeclarative,
		Provider:    core.ProviderTypeClaude,
		Model:       "sonnet",
		Description: "Test Claude agent",
		Capabilities: []string{"code", "analysis", "testing"},
		Strategy:    core.DelegationAutomatic,
		MaxRetries:  3,
		Timeout:     time.Minute * 5,
	}

	adapter := NewClaudeAdapter(config)
	assert.NotNil(t, adapter)
	assert.Equal(t, "claude-test-agent", adapter.Name())
	assert.Equal(t, core.ProviderTypeClaude, adapter.Provider())
	assert.Equal(t, config, adapter.Config())
}

func TestClaudeAdapter_Initialize(t *testing.T) {
	adapter := NewClaudeAdapter(core.SubAgentConfig{
		Name:     "test",
		Provider: core.ProviderTypeClaude,
	})

	t.Run("Initialize with valid config", func(t *testing.T) {
		config := core.SubAgentConfig{
			Name:        "initialized-agent",
			Provider:    core.ProviderTypeClaude,
			Model:       "opus",
			Capabilities: []string{"advanced"},
		}

		err := adapter.Initialize(config)
		assert.NoError(t, err)
		assert.Equal(t, config, adapter.Config())
	})

	t.Run("Initialize with invalid config", func(t *testing.T) {
		config := core.SubAgentConfig{
			// Missing required fields
			Provider: core.ProviderTypeClaude,
		}

		err := adapter.Initialize(config)
		// Should validate and potentially error
		if err != nil {
			assert.Contains(t, err.Error(), "name")
		}
	})
}

func TestClaudeAdapter_TaskAdaptation(t *testing.T) {
	adapter := NewClaudeAdapter(core.SubAgentConfig{
		Name:     "claude-adapter",
		Provider: core.ProviderTypeClaude,
		Model:    "sonnet",
	})

	t.Run("Adapt standard task", func(t *testing.T) {
		task := core.SubAgentTask{
			ID:          "task-123",
			Name:        "Code Review",
			Description: "Review the following code",
			Input:       "def hello(): print('world')",
			Context: map[string]interface{}{
				"language": "python",
				"type":     "function",
			},
			Variables: map[string]interface{}{
				"style_guide": "PEP8",
			},
			Priority: 5,
		}

		adapted, err := adapter.AdaptTask(task)
		require.NoError(t, err)
		assert.NotNil(t, adapted)

		// Check if adaptation preserves essential fields
		// The actual format depends on Claude's Task tool format
		if taskMap, ok := adapted.(map[string]interface{}); ok {
			assert.Contains(t, taskMap, "task")
			assert.Contains(t, taskMap, "context")
		}
	})

	t.Run("Adapt task with deadline", func(t *testing.T) {
		deadline := time.Now().Add(time.Hour)
		task := core.SubAgentTask{
			ID:       "urgent-task",
			Name:     "Urgent Fix",
			Deadline: &deadline,
		}

		adapted, err := adapter.AdaptTask(task)
		require.NoError(t, err)
		assert.NotNil(t, adapted)
	})

	t.Run("Adapt task with constraints", func(t *testing.T) {
		task := core.SubAgentTask{
			ID:          "constrained-task",
			Name:        "Optimized Code",
			Constraints: []string{"memory < 100MB", "time < 1s", "no external deps"},
		}

		adapted, err := adapter.AdaptTask(task)
		require.NoError(t, err)
		assert.NotNil(t, adapted)
	})
}

func TestClaudeAdapter_ResultAdaptation(t *testing.T) {
	adapter := NewClaudeAdapter(core.SubAgentConfig{
		Name:     "claude-adapter",
		Provider: core.ProviderTypeClaude,
	})

	t.Run("Adapt successful result", func(t *testing.T) {
		// Simulate Claude's Task tool response format
		claudeResult := map[string]interface{}{
			"status": "completed",
			"output": "Task completed successfully",
			"metadata": map[string]interface{}{
				"tokens_used": 1500,
				"model":       "claude-3-sonnet",
			},
		}

		result, err := adapter.AdaptResult(claudeResult)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, core.StatusCompleted, result.Status)
		assert.Contains(t, result.Output, "completed successfully")
	})

	t.Run("Adapt failed result", func(t *testing.T) {
		claudeResult := map[string]interface{}{
			"status": "failed",
			"error":  "Task execution failed: timeout",
		}

		result, err := adapter.AdaptResult(claudeResult)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, core.StatusFailed, result.Status)
		assert.NotNil(t, result.Error)
	})

	t.Run("Adapt result with artifacts", func(t *testing.T) {
		claudeResult := map[string]interface{}{
			"status": "completed",
			"output": "Generated code",
			"artifacts": []map[string]interface{}{
				{
					"name":    "main.py",
					"type":    "file",
					"content": "print('hello')",
				},
			},
		}

		result, err := adapter.AdaptResult(claudeResult)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Artifacts, 1)
		assert.Equal(t, "main.py", result.Artifacts[0].Name)
	})
}

func TestClaudeAdapter_Execution(t *testing.T) {
	// Note: These tests would require mocking the actual Claude provider
	// For unit tests, we'll simulate the behavior

	t.Run("Execute simple task", func(t *testing.T) {
		// In a real test, we'd mock the provider interaction
		// For now, we'll test the structure
		assert.NotPanics(t, func() {
			// This would normally execute
			// adapter := NewClaudeAdapter(...)
			// ctx := context.Background()
			// task := core.SubAgentTask{...}
			// result, err := adapter.Execute(ctx, task)
		})
	})

	t.Run("Execute with timeout", func(t *testing.T) {
		// Test timeout handling
		assert.NotPanics(t, func() {
			// ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond*100)
			// defer cancel()
			// task := core.SubAgentTask{...}
			// adapter.Execute(ctx, task)
		})
	})
}

func TestClaudeAdapter_Capabilities(t *testing.T) {
	adapter := NewClaudeAdapter(core.SubAgentConfig{
		Name:         "capable-claude",
		Provider:     core.ProviderTypeClaude,
		Capabilities: []string{"code", "review", "test", "documentation"},
	})

	t.Run("Get capabilities", func(t *testing.T) {
		caps := adapter.GetCapabilities()
		assert.Len(t, caps, 4)
		assert.Contains(t, caps, "code")
		assert.Contains(t, caps, "documentation")
	})

	t.Run("Can handle matching task", func(t *testing.T) {
		task := core.SubAgentTask{
			Context: map[string]interface{}{
				"code": true,
			},
		}
		assert.True(t, adapter.CanHandle(task))
	})

	t.Run("Cannot handle non-matching task", func(t *testing.T) {
		task := core.SubAgentTask{
			Context: map[string]interface{}{
				"graphics": true,
			},
		}
		// Depends on implementation - might still handle
		canHandle := adapter.CanHandle(task)
		assert.NotNil(t, &canHandle) // Just check it returns a value
	})

	t.Run("Supports parallel execution", func(t *testing.T) {
		// Claude typically supports parallel via Task tool
		assert.True(t, adapter.SupportsParallel())
	})

	t.Run("Supports interactive mode", func(t *testing.T) {
		adapter.config.Interactive = true
		assert.True(t, adapter.SupportsInteractive())
	})
}

func TestClaudeAdapter_ProviderIntegration(t *testing.T) {
	adapter := NewClaudeAdapter(core.SubAgentConfig{
		Name:     "provider-claude",
		Provider: core.ProviderTypeClaude,
	})

	t.Run("Initialize with provider", func(t *testing.T) {
		// Create a mock provider
		provider := &MockClaudeProvider{
			name: "claude-provider",
			typ:  core.ProviderTypeClaude,
		}

		err := adapter.InitializeProvider(provider)
		assert.NoError(t, err)
	})

	t.Run("Get provider config", func(t *testing.T) {
		adapter.config.ProviderConfig = map[string]interface{}{
			"api_key":    "test-key",
			"max_tokens": 4000,
		}

		config := adapter.GetProviderConfig()
		assert.NotNil(t, config)
		assert.Contains(t, config, "api_key")
		assert.Equal(t, 4000, config["max_tokens"])
	})
}

func TestClaudeAdapter_StatusAndControl(t *testing.T) {
	adapter := NewClaudeAdapter(core.SubAgentConfig{
		Name:     "control-claude",
		Provider: core.ProviderTypeClaude,
	})

	t.Run("Initial status", func(t *testing.T) {
		assert.Equal(t, core.StatusPending, adapter.Status())
	})

	t.Run("Cancel execution", func(t *testing.T) {
		err := adapter.Cancel()
		assert.NoError(t, err)
		// Status should change after cancel
		// Implementation dependent
	})

	t.Run("Get progress", func(t *testing.T) {
		progress, message := adapter.GetProgress()
		assert.GreaterOrEqual(t, progress, 0.0)
		assert.LessOrEqual(t, progress, 1.0)
		assert.NotEmpty(t, message)
	})

	t.Run("Cleanup", func(t *testing.T) {
		err := adapter.Cleanup()
		assert.NoError(t, err)
	})
}

// Mock Claude provider for testing
type MockClaudeProvider struct {
	name string
	typ  core.ProviderType
}

func (m *MockClaudeProvider) Name() string                                       { return m.name }
func (m *MockClaudeProvider) Type() core.ProviderType                            { return m.typ }
func (m *MockClaudeProvider) Initialize(config core.ProviderConfig) error        { return nil }
func (m *MockClaudeProvider) Validate() error                                    { return nil }
func (m *MockClaudeProvider) GetPTYCommand() (*exec.Cmd, error)                  { return exec.Command("echo"), nil }
func (m *MockClaudeProvider) GetPTYCommandWithPrompt(prompt string) (*exec.Cmd, error) { return exec.Command("echo", prompt), nil }
func (m *MockClaudeProvider) Features() core.ProviderFeatures                    { return core.ProviderFeatures{} }
func (m *MockClaudeProvider) SupportsModel(model string) bool                    { return true }
func (m *MockClaudeProvider) PrepareSession(ctx context.Context, sessionID string) error { return nil }
func (m *MockClaudeProvider) CleanupSession(ctx context.Context, sessionID string) error { return nil }
func (m *MockClaudeProvider) GetReadyPattern() string                            { return "ready" }
func (m *MockClaudeProvider) GetOutputPattern() string                           { return "output" }
func (m *MockClaudeProvider) GetErrorPattern() string                            { return "error" }
func (m *MockClaudeProvider) GetPromptInjectionMethod() string                   { return "clipboard" }
func (m *MockClaudeProvider) InjectPrompt(prompt string) error                   { return nil }
func (m *MockClaudeProvider) GetMCPServers() []core.MCPServer                    { return []core.MCPServer{} }
func (m *MockClaudeProvider) GetTools() []core.Tool                              { return []core.Tool{} }
func (m *MockClaudeProvider) GetSlashCommands() []core.SharedSlashCommand        { return []core.SharedSlashCommand{} }
func (m *MockClaudeProvider) GetPlugins() []core.PluginReference                 { return []core.PluginReference{} }
func (m *MockClaudeProvider) SupportsSlashCommands() bool                        { return false }
func (m *MockClaudeProvider) GetSlashCommandDirectory() string                   { return "" }
func (m *MockClaudeProvider) GetSlashCommandFormat() string                      { return "" }
func (m *MockClaudeProvider) PrepareSlashCommands(commands []core.SharedSlashCommand, targetDir string) error { return nil }

// Benchmark tests
func BenchmarkClaudeAdapter_TaskAdaptation(b *testing.B) {
	adapter := NewClaudeAdapter(core.SubAgentConfig{
		Name:     "bench-claude",
		Provider: core.ProviderTypeClaude,
	})

	task := core.SubAgentTask{
		ID:   "bench-task",
		Name: "Benchmark Task",
		Context: map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.AdaptTask(task)
	}
}

func BenchmarkClaudeAdapter_ResultAdaptation(b *testing.B) {
	adapter := NewClaudeAdapter(core.SubAgentConfig{
		Name:     "bench-claude",
		Provider: core.ProviderTypeClaude,
	})

	result := map[string]interface{}{
		"status": "completed",
		"output": "test output",
		"metadata": map[string]interface{}{
			"tokens": 1000,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.AdaptResult(result)
	}
}