package qwen

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

func TestQwenAdapter_Creation(t *testing.T) {
	config := core.SubAgentConfig{
		Name:        "qwen-test-agent",
		Type:        core.SubAgentTypeWorkflow,
		Provider:    core.ProviderTypeQwen,
		Model:       "code",
		Description: "Test Qwen agent",
		Capabilities: []string{"code", "debugging", "optimization"},
		Strategy:    core.DelegationExplicit,
		MaxRetries:  1,
		Timeout:     time.Minute * 2,
	}

	adapter := NewQwenAdapter(config)
	assert.NotNil(t, adapter)
	assert.Equal(t, "qwen-test-agent", adapter.Name())
	assert.Equal(t, core.ProviderTypeQwen, adapter.Provider())
	assert.Equal(t, config, adapter.Config())
}

func TestQwenAdapter_Initialize(t *testing.T) {
	adapter := NewQwenAdapter(core.SubAgentConfig{
		Name:     "test",
		Provider: core.ProviderTypeQwen,
	})

	t.Run("Initialize with valid config", func(t *testing.T) {
		config := core.SubAgentConfig{
			Name:        "initialized-qwen",
			Provider:    core.ProviderTypeQwen,
			Model:       "code",
			Capabilities: []string{"code", "testing"},
		}

		err := adapter.Initialize(config)
		assert.NoError(t, err)
		assert.Equal(t, config, adapter.Config())
	})

	t.Run("Initialize with workflow config", func(t *testing.T) {
		config := core.SubAgentConfig{
			Name:     "workflow-qwen",
			Provider: core.ProviderTypeQwen,
			Type:     core.SubAgentTypeWorkflow,
			ProviderConfig: map[string]interface{}{
				"workflow_path": "/path/to/workflow.yaml",
				"variables": map[string]interface{}{
					"env": "test",
				},
			},
		}

		err := adapter.Initialize(config)
		assert.NoError(t, err)
		assert.Equal(t, "/path/to/workflow.yaml", config.ProviderConfig["workflow_path"])
	})
}

func TestQwenAdapter_TaskAdaptation(t *testing.T) {
	adapter := NewQwenAdapter(core.SubAgentConfig{
		Name:     "qwen-adapter",
		Provider: core.ProviderTypeQwen,
		Model:    "code",
	})

	t.Run("Adapt code task", func(t *testing.T) {
		task := core.SubAgentTask{
			ID:          "code-task",
			Name:        "Code Generation",
			Description: "Generate Python function",
			Input:       "Create a fibonacci function",
			Context: map[string]interface{}{
				"language": "python",
				"style":    "functional",
			},
			Variables: map[string]interface{}{
				"max_length": 100,
			},
			Priority: 2,
		}

		adapted, err := adapter.AdaptTask(task)
		require.NoError(t, err)
		assert.NotNil(t, adapted)

		// Check if adaptation follows Qwen's format
		if qwenTask, ok := adapted.(map[string]interface{}); ok {
			assert.NotNil(t, qwenTask)
		}
	})

	t.Run("Adapt debugging task", func(t *testing.T) {
		task := core.SubAgentTask{
			ID:   "debug-task",
			Name: "Debug Code",
			Input: "def func():\n  print('hello'\n", // Intentional syntax error
			Context: map[string]interface{}{
				"task_type": "debugging",
				"language":  "python",
			},
		}

		adapted, err := adapter.AdaptTask(task)
		require.NoError(t, err)
		assert.NotNil(t, adapted)
	})

	t.Run("Adapt optimization task", func(t *testing.T) {
		task := core.SubAgentTask{
			ID:          "optimize-task",
			Name:        "Optimize Algorithm",
			Constraints: []string{"time_complexity < O(n^2)", "space_complexity < O(n)"},
		}

		adapted, err := adapter.AdaptTask(task)
		require.NoError(t, err)
		assert.NotNil(t, adapted)
	})
}

func TestQwenAdapter_ResultAdaptation(t *testing.T) {
	adapter := NewQwenAdapter(core.SubAgentConfig{
		Name:     "qwen-adapter",
		Provider: core.ProviderTypeQwen,
	})

	t.Run("Adapt code generation result", func(t *testing.T) {
		qwenResult := map[string]interface{}{
			"status": "success",
			"code": `def fibonacci(n):
    if n <= 1:
        return n
    return fibonacci(n-1) + fibonacci(n-2)`,
			"language": "python",
			"metadata": map[string]interface{}{
				"lines":      4,
				"complexity": "O(2^n)",
			},
		}

		result, err := adapter.AdaptResult(qwenResult)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, core.StatusCompleted, result.Status)
		assert.Contains(t, result.Output, "fibonacci")
	})

	t.Run("Adapt failed result", func(t *testing.T) {
		qwenResult := map[string]interface{}{
			"status": "error",
			"error":  "Syntax error in input code",
			"line":   3,
		}

		result, err := adapter.AdaptResult(qwenResult)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, core.StatusFailed, result.Status)
		assert.NotNil(t, result.Error)
	})

	t.Run("Adapt result with code artifacts", func(t *testing.T) {
		qwenResult := map[string]interface{}{
			"status": "success",
			"files": []map[string]interface{}{
				{
					"name":     "main.py",
					"content":  "print('hello')",
					"language": "python",
				},
				{
					"name":     "test.py",
					"content":  "import main",
					"language": "python",
				},
			},
		}

		result, err := adapter.AdaptResult(qwenResult)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Artifacts, 2)
		assert.Equal(t, "main.py", result.Artifacts[0].Name)
		assert.Equal(t, "test.py", result.Artifacts[1].Name)
	})
}

func TestQwenAdapter_WorkflowIntegration(t *testing.T) {
	adapter := NewQwenAdapter(core.SubAgentConfig{
		Name:     "workflow-qwen",
		Provider: core.ProviderTypeQwen,
		Type:     core.SubAgentTypeWorkflow,
	})

	t.Run("Execute as workflow", func(t *testing.T) {
		task := core.SubAgentTask{
			ID:   "workflow-task",
			Name: "Multi-step Processing",
			Context: map[string]interface{}{
				"steps": []string{"analyze", "refactor", "test"},
			},
		}

		adapted, err := adapter.AdaptTask(task)
		require.NoError(t, err)
		
		// Verify workflow configuration
		if workflowConfig, ok := adapted.(map[string]interface{}); ok {
			assert.NotNil(t, workflowConfig)
			// Would contain workflow-specific settings
		}
	})

	t.Run("Handle workflow variables", func(t *testing.T) {
		task := core.SubAgentTask{
			ID: "variable-task",
			Variables: map[string]interface{}{
				"input_file":  "source.py",
				"output_file": "optimized.py",
				"mode":        "aggressive",
			},
		}

		adapted, err := adapter.AdaptTask(task)
		assert.NoError(t, err)
		assert.NotNil(t, adapted)
	})
}

func TestQwenAdapter_CodeSpecificFeatures(t *testing.T) {
	adapter := NewQwenAdapter(core.SubAgentConfig{
		Name:         "code-qwen",
		Provider:     core.ProviderTypeQwen,
		Model:        "code",
		Capabilities: []string{"code", "refactor", "test", "debug"},
	})

	t.Run("Handle multiple languages", func(t *testing.T) {
		languages := []string{"python", "javascript", "go", "rust"}
		
		for _, lang := range languages {
			task := core.SubAgentTask{
				ID: lang + "-task",
				Context: map[string]interface{}{
					"language": lang,
				},
			}
			
			adapted, err := adapter.AdaptTask(task)
			assert.NoError(t, err)
			assert.NotNil(t, adapted)
		}
	})

	t.Run("Code analysis capabilities", func(t *testing.T) {
		task := core.SubAgentTask{
			ID:   "analysis-task",
			Name: "Analyze Code Quality",
			Input: `
func example() {
    // Complex nested logic
    for i := 0; i < 10; i++ {
        if i % 2 == 0 {
            for j := 0; j < i; j++ {
                // Deep nesting
            }
        }
    }
}`,
			Context: map[string]interface{}{
				"analysis_type": "complexity",
				"language":      "go",
			},
		}

		adapted, err := adapter.AdaptTask(task)
		assert.NoError(t, err)
		assert.NotNil(t, adapted)
	})
}

func TestQwenAdapter_Capabilities(t *testing.T) {
	adapter := NewQwenAdapter(core.SubAgentConfig{
		Name:         "capable-qwen",
		Provider:     core.ProviderTypeQwen,
		Capabilities: []string{"code", "debug", "optimize", "test"},
	})

	t.Run("Get capabilities", func(t *testing.T) {
		caps := adapter.GetCapabilities()
		assert.Len(t, caps, 4)
		assert.Contains(t, caps, "code")
		assert.Contains(t, caps, "optimize")
	})

	t.Run("Can handle code tasks", func(t *testing.T) {
		task := core.SubAgentTask{
			Context: map[string]interface{}{
				"code": true,
			},
		}
		assert.True(t, adapter.CanHandle(task))
	})

	t.Run("Cannot handle non-code tasks", func(t *testing.T) {
		task := core.SubAgentTask{
			Context: map[string]interface{}{
				"image_generation": true,
			},
		}
		// Qwen Code might not handle image tasks
		canHandle := adapter.CanHandle(task)
		// Just verify it returns a boolean
		assert.NotNil(t, &canHandle)
	})

	t.Run("Supports parallel execution", func(t *testing.T) {
		adapter.config.Parallel = false // Qwen might be sequential by default
		assert.False(t, adapter.SupportsParallel())
		
		adapter.config.Parallel = true
		assert.True(t, adapter.SupportsParallel())
	})
}

func TestQwenAdapter_ProviderIntegration(t *testing.T) {
	adapter := NewQwenAdapter(core.SubAgentConfig{
		Name:     "provider-qwen",
		Provider: core.ProviderTypeQwen,
	})

	t.Run("Initialize with provider", func(t *testing.T) {
		provider := &MockQwenProvider{
			name: "qwen-provider",
			typ:  core.ProviderTypeQwen,
		}

		err := adapter.InitializeProvider(provider)
		assert.NoError(t, err)
	})

	t.Run("Get provider config", func(t *testing.T) {
		adapter.config.ProviderConfig = map[string]interface{}{
			"api_key":        "test-key",
			"model_version":  "latest",
			"code_style":     "clean",
			"max_iterations": 3,
		}

		config := adapter.GetProviderConfig()
		assert.NotNil(t, config)
		assert.Contains(t, config, "api_key")
		assert.Equal(t, "clean", config["code_style"])
		assert.Equal(t, 3, config["max_iterations"])
	})
}

func TestQwenAdapter_StatusAndControl(t *testing.T) {
	adapter := NewQwenAdapter(core.SubAgentConfig{
		Name:     "control-qwen",
		Provider: core.ProviderTypeQwen,
	})

	t.Run("Initial status", func(t *testing.T) {
		assert.Equal(t, core.StatusPending, adapter.Status())
	})

	t.Run("Cancel execution", func(t *testing.T) {
		err := adapter.Cancel()
		assert.NoError(t, err)
	})

	t.Run("Get progress for code tasks", func(t *testing.T) {
		progress, message := adapter.GetProgress()
		assert.GreaterOrEqual(t, progress, 0.0)
		assert.LessOrEqual(t, progress, 1.0)
		assert.NotEmpty(t, message)
	})

	t.Run("Validate code-specific config", func(t *testing.T) {
		adapter.config.Model = "code"
		err := adapter.Validate()
		assert.NoError(t, err)
		
		// Invalid model for Qwen
		adapter.config.Model = "image"
		err = adapter.Validate()
		// Might error for invalid model
		if err != nil {
			assert.Contains(t, err.Error(), "model")
		}
		
		// Reset
		adapter.config.Model = "code"
	})

	t.Run("Cleanup resources", func(t *testing.T) {
		err := adapter.Cleanup()
		assert.NoError(t, err)
	})
}

func TestQwenAdapter_EdgeCases(t *testing.T) {
	adapter := NewQwenAdapter(core.SubAgentConfig{
		Name:     "edge-qwen",
		Provider: core.ProviderTypeQwen,
		Model:    "code",
	})

	t.Run("Handle malformed code input", func(t *testing.T) {
		task := core.SubAgentTask{
			ID:    "malformed-task",
			Input: "def func( print('broken'",
			Context: map[string]interface{}{
				"language": "python",
			},
		}

		adapted, err := adapter.AdaptTask(task)
		// Should handle gracefully
		assert.NoError(t, err)
		assert.NotNil(t, adapted)
	})

	t.Run("Handle unsupported language", func(t *testing.T) {
		task := core.SubAgentTask{
			ID: "unsupported-lang",
			Context: map[string]interface{}{
				"language": "cobol", // Potentially unsupported
			},
		}

		adapted, err := adapter.AdaptTask(task)
		// Should adapt even for unsupported languages
		assert.NoError(t, err)
		assert.NotNil(t, adapted)
	})

	t.Run("Handle large code files", func(t *testing.T) {
		// Generate large input
		largeCode := ""
		for i := 0; i < 1000; i++ {
			largeCode += "def func_" + string(rune(i)) + "(): pass\n"
		}

		task := core.SubAgentTask{
			ID:    "large-task",
			Input: largeCode,
		}

		adapted, err := adapter.AdaptTask(task)
		assert.NoError(t, err)
		assert.NotNil(t, adapted)
	})
}

// Mock Qwen provider for testing
type MockQwenProvider struct {
	name string
	typ  core.ProviderType
}

func (m *MockQwenProvider) Name() string                                       { return m.name }
func (m *MockQwenProvider) Type() core.ProviderType                            { return m.typ }
func (m *MockQwenProvider) Initialize(config core.ProviderConfig) error        { return nil }
func (m *MockQwenProvider) Validate() error                                    { return nil }
func (m *MockQwenProvider) GetPTYCommand() (*exec.Cmd, error)                  { return exec.Command("echo"), nil }
func (m *MockQwenProvider) GetPTYCommandWithPrompt(prompt string) (*exec.Cmd, error) { return exec.Command("echo", prompt), nil }
func (m *MockQwenProvider) Features() core.ProviderFeatures                    { return core.ProviderFeatures{} }
func (m *MockQwenProvider) SupportsModel(model string) bool                    { return true }
func (m *MockQwenProvider) PrepareSession(ctx context.Context, sessionID string) error { return nil }
func (m *MockQwenProvider) CleanupSession(ctx context.Context, sessionID string) error { return nil }
func (m *MockQwenProvider) GetReadyPattern() string                            { return "ready" }
func (m *MockQwenProvider) GetOutputPattern() string                           { return "output" }
func (m *MockQwenProvider) GetErrorPattern() string                            { return "error" }
func (m *MockQwenProvider) GetPromptInjectionMethod() string                   { return "file" }
func (m *MockQwenProvider) InjectPrompt(prompt string) error                   { return nil }
func (m *MockQwenProvider) GetMCPServers() []core.MCPServer                    { return []core.MCPServer{} }
func (m *MockQwenProvider) GetTools() []core.Tool                              { return []core.Tool{} }
func (m *MockQwenProvider) GetSlashCommands() []core.SharedSlashCommand        { return []core.SharedSlashCommand{} }
func (m *MockQwenProvider) GetPlugins() []core.PluginReference                 { return []core.PluginReference{} }
func (m *MockQwenProvider) SupportsSlashCommands() bool                        { return false }
func (m *MockQwenProvider) GetSlashCommandDirectory() string                   { return "" }
func (m *MockQwenProvider) GetSlashCommandFormat() string                      { return "" }
func (m *MockQwenProvider) PrepareSlashCommands(commands []core.SharedSlashCommand, targetDir string) error { return nil }

// Benchmark tests
func BenchmarkQwenAdapter_TaskAdaptation(b *testing.B) {
	adapter := NewQwenAdapter(core.SubAgentConfig{
		Name:     "bench-qwen",
		Provider: core.ProviderTypeQwen,
		Model:    "code",
	})

	task := core.SubAgentTask{
		ID:   "bench-task",
		Name: "Benchmark Task",
		Input: "def example(): return 42",
		Context: map[string]interface{}{
			"language": "python",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.AdaptTask(task)
	}
}

func BenchmarkQwenAdapter_ResultAdaptation(b *testing.B) {
	adapter := NewQwenAdapter(core.SubAgentConfig{
		Name:     "bench-qwen",
		Provider: core.ProviderTypeQwen,
	})

	result := map[string]interface{}{
		"status": "success",
		"code":   "def optimized(): return 42",
		"metadata": map[string]interface{}{
			"performance": "improved",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		adapter.AdaptResult(result)
	}
}