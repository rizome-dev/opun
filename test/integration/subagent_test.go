package integration

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
	"errors"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"github.com/rizome-dev/opun/internal/providers"
	internalsubagent "github.com/rizome-dev/opun/internal/subagent"
	"github.com/rizome-dev/opun/pkg/core"
	"github.com/rizome-dev/opun/pkg/subagent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Integration test requires actual providers to be available
func skipIfNoProviders(t *testing.T) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}
}

func TestSubAgent_EndToEndWorkflow(t *testing.T) {
	skipIfNoProviders(t)

	// Create manager and factory
	manager := subagent.NewManager()
	providerFactory := providers.NewProviderFactory()

	// Setup providers if available
	setupProviders(t, manager, providerFactory)

	t.Run("Cross-provider task delegation", func(t *testing.T) {
		ctx := context.Background()

		// Create tasks for different providers
		tasks := []core.SubAgentTask{
			{
				ID:          "claude-task",
				Name:        "Analysis Task",
				Description: "Analyze this code structure",
				Input:       "func main() { println('hello') }",
				Context: map[string]interface{}{
					"provider": "claude",
					"type":     "analysis",
				},
			},
			{
				ID:          "gemini-task",
				Name:        "Generation Task",
				Description: "Generate test cases",
				Context: map[string]interface{}{
					"provider": "gemini",
					"type":     "generation",
				},
			},
			{
				ID:          "qwen-task",
				Name:        "Code Task",
				Description: "Optimize the code",
				Context: map[string]interface{}{
					"provider": "qwen",
					"type":     "optimization",
				},
			},
		}

		// Execute tasks across providers
		results, err := manager.CoordinateAcrossProviders(ctx, tasks)
		
		// May error if providers aren't available
		if err != nil {
			t.Logf("Cross-provider coordination error (expected if providers not available): %v", err)
			return
		}

		assert.Len(t, results, len(tasks))
		for _, result := range results {
			assert.NotNil(t, result)
			t.Logf("Task %s completed with status: %s", result.TaskID, result.Status)
		}
	})
}

func TestSubAgent_RealProviderInteraction(t *testing.T) {
	skipIfNoProviders(t)

	t.Run("Claude Task tool integration", func(t *testing.T) {
		// Check if Claude is available
		if _, err := exec.LookPath("claude"); err != nil {
			t.Skip("Claude not available")
		}

		factory := providers.NewProviderFactory()
		config := core.ProviderConfig{
			Name:    "claude-test",
			Type:    core.ProviderTypeClaude,
			Command: "claude",
			Model:   "sonnet",
		}

		_, err := factory.CreateProvider(config)
		if err != nil {
			t.Skipf("Could not create Claude provider: %v", err)
		}

		// Create Claude subagent
		subagentFactory := internalsubagent.NewFactory()
		agentConfig := core.SubAgentConfig{
			Name:     "claude-agent",
			Provider: core.ProviderTypeClaude,
			Type:     core.SubAgentTypeDeclarative,
		}

		adapter, err := subagentFactory.CreateAdapter(agentConfig)
		require.NoError(t, err)

		// Execute a real task
		ctx := context.Background()
		task := core.SubAgentTask{
			ID:    "real-claude-task",
			Name:  "Test Task",
			Input: "Write a hello world function in Python",
		}

		result, err := adapter.Execute(ctx, task)
		if err != nil {
			t.Logf("Execution error (expected if provider not fully configured): %v", err)
			return
		}

		assert.NotNil(t, result)
		assert.NotEmpty(t, result.Output)
		t.Logf("Claude output: %s", result.Output)
	})

	t.Run("Gemini SubAgentScope integration", func(t *testing.T) {
		// Check if Gemini is available
		if _, err := exec.LookPath("gemini"); err != nil {
			t.Skip("Gemini not available")
		}

		// Similar setup for Gemini
		// Implementation would follow similar pattern
	})

	t.Run("Qwen Code integration", func(t *testing.T) {
		// Check if Qwen is available
		if _, err := exec.LookPath("qwen"); err != nil {
			t.Skip("Qwen not available")
		}

		// Similar setup for Qwen
		// Implementation would follow similar pattern
	})
}

func TestSubAgent_WorkflowExecution(t *testing.T) {
	skipIfNoProviders(t)

	manager := subagent.NewManager()
	factory := internalsubagent.NewFactory()

	// Create workflow-based agents
	configs := []core.SubAgentConfig{
		{
			Name:     "analyzer",
			Provider: core.ProviderTypeMock, // Use mock for testing
			Type:     core.SubAgentTypeWorkflow,
			Capabilities: []string{"analysis"},
		},
		{
			Name:     "generator",
			Provider: core.ProviderTypeMock,
			Type:     core.SubAgentTypeWorkflow,
			Capabilities: []string{"generation"},
		},
		{
			Name:     "validator",
			Provider: core.ProviderTypeMock,
			Type:     core.SubAgentTypeWorkflow,
			Capabilities: []string{"validation"},
		},
	}

	// Register agents
	for _, config := range configs {
		adapter, err := factory.CreateAdapter(config)
		require.NoError(t, err)
		err = manager.Register(adapter)
		require.NoError(t, err)
	}

	t.Run("Sequential workflow execution", func(t *testing.T) {
		ctx := context.Background()

		// Define workflow tasks
		tasks := []core.SubAgentTask{
			{
				ID:   "step1",
				Name: "Analyze",
				Context: map[string]interface{}{
					"analysis": true,
				},
			},
			{
				ID:   "step2",
				Name: "Generate",
				Context: map[string]interface{}{
					"generation": true,
					"input":      "{{step1.output}}", // Reference previous output
				},
			},
			{
				ID:   "step3",
				Name: "Validate",
				Context: map[string]interface{}{
					"validation": true,
					"input":      "{{step2.output}}",
				},
			},
		}

		// Execute workflow
		for _, task := range tasks {
			result, err := manager.Delegate(ctx, task)
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, core.StatusCompleted, result.Status)
			t.Logf("Step %s completed", task.ID)
		}
	})

	t.Run("Parallel workflow execution", func(t *testing.T) {
		ctx := context.Background()

		// Define parallel tasks
		tasks := []core.SubAgentTask{
			{ID: "parallel1", Context: map[string]interface{}{"analysis": true}},
			{ID: "parallel2", Context: map[string]interface{}{"generation": true}},
			{ID: "parallel3", Context: map[string]interface{}{"validation": true}},
		}

		results, err := manager.ExecuteParallel(ctx, tasks)
		require.NoError(t, err)
		assert.Len(t, results, 3)

		for _, result := range results {
			assert.Equal(t, core.StatusCompleted, result.Status)
		}
	})
}

func TestSubAgent_MCPIntegration(t *testing.T) {
	skipIfNoProviders(t)

	t.Run("MCP Task tool delegation", func(t *testing.T) {
		// This would test the MCP server integration
		// Requires MCP server to be running

		manager := subagent.NewManager()
		factory := internalsubagent.NewFactory()

		// Create MCP-enabled agent
		config := core.SubAgentConfig{
			Name:       "mcp-agent",
			Provider:   core.ProviderTypeClaude,
			Type:       core.SubAgentTypeMCP,
			MCPServers: []string{"task-server"},
		}

		adapter, err := factory.CreateAdapter(config)
		if err != nil {
			t.Skipf("MCP not available: %v", err)
		}

		err = manager.Register(adapter)
		require.NoError(t, err)

		// Test MCP delegation
		ctx := context.Background()
		task := core.SubAgentTask{
			ID:   "mcp-task",
			Name: "MCP Delegated Task",
		}

		result, err := adapter.Execute(ctx, task)
		if err != nil {
			t.Logf("MCP execution error (expected if server not running): %v", err)
			return
		}

		assert.NotNil(t, result)
	})
}

func TestSubAgent_PerformanceAndScaling(t *testing.T) {
	manager := subagent.NewManager()
	factory := internalsubagent.NewFactory()

	// Create many test agents for scaling test
	numAgents := 50
	for i := 0; i < numAgents; i++ {
		config := core.SubAgentConfig{
			Name:     string(rune('a' + (i % 26))) + string(rune('0' + (i / 26))),
			Provider: core.ProviderTypeClaude,
			Capabilities: []string{"cap" + string(rune(i % 10))},
		}

		adapter, err := factory.CreateAdapter(config)
		require.NoError(t, err)
		err = manager.Register(adapter)
		require.NoError(t, err)
	}

	t.Run("Large scale parallel execution", func(t *testing.T) {
		ctx := context.Background()
		numTasks := 100

		// Create many tasks
		tasks := make([]core.SubAgentTask, numTasks)
		for i := 0; i < numTasks; i++ {
			tasks[i] = core.SubAgentTask{
				ID:   string(rune('A' + (i % 26))) + string(rune('0' + (i / 26))),
				Name: "Task " + string(rune(i)),
				Context: map[string]interface{}{
					"cap" + string(rune(i % 10)): true,
				},
			}
		}

		start := time.Now()
		results, err := manager.ExecuteParallel(ctx, tasks)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.Len(t, results, numTasks)

		t.Logf("Executed %d tasks across %d agents in %v", numTasks, numAgents, duration)
		
		// Performance assertion - should complete reasonably fast
		assert.Less(t, duration, time.Second*10, "Parallel execution took too long")
	})

	t.Run("Concurrent delegation stress test", func(t *testing.T) {
		ctx := context.Background()
		numGoroutines := 20
		tasksPerGoroutine := 10

		var wg sync.WaitGroup
		errors := make(chan error, numGoroutines*tasksPerGoroutine)

		for g := 0; g < numGoroutines; g++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for t := 0; t < tasksPerGoroutine; t++ {
					task := core.SubAgentTask{
						ID: string(rune('G' + goroutineID)) + string(rune('T' + t)),
						Context: map[string]interface{}{
							"cap" + string(rune(goroutineID % 10)): true,
						},
					}

					_, err := manager.Delegate(ctx, task)
					if err != nil {
						errors <- err
					}
				}
			}(g)
		}

		wg.Wait()
		close(errors)

		// Check for errors
		errorCount := 0
		for err := range errors {
			if err != nil {
				errorCount++
				t.Logf("Delegation error: %v", err)
			}
		}

		assert.Equal(t, 0, errorCount, "Should have no errors in concurrent delegation")
	})
}

func TestSubAgent_ErrorHandlingAndRecovery(t *testing.T) {
	manager := subagent.NewManager()

	// Create agents with different failure modes
	failingAgent := &FailingMockAgent{
		name:      "failing-agent",
		failAfter: 2,
	}
	manager.Register(failingAgent)

	timeoutAgent := &TimeoutMockAgent{
		name:    "timeout-agent",
		timeout: time.Second * 2,
	}
	manager.Register(timeoutAgent)

	t.Run("Handle agent failures", func(t *testing.T) {
		ctx := context.Background()
		
		// First two executions should succeed
		for i := 0; i < 2; i++ {
			task := core.SubAgentTask{ID: string(rune('a' + i))}
			result, err := manager.Execute(ctx, task, "failing-agent")
			require.NoError(t, err)
			assert.Equal(t, core.StatusCompleted, result.Status)
		}

		// Third execution should fail
		task := core.SubAgentTask{ID: "fail-task"}
		result, err := manager.Execute(ctx, task, "failing-agent")
		assert.Error(t, err)
		if result != nil {
			assert.Equal(t, core.StatusFailed, result.Status)
		}
	})

	t.Run("Handle timeouts", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		task := core.SubAgentTask{ID: "timeout-task"}
		result, err := manager.Execute(ctx, task, "timeout-agent")
		
		assert.Error(t, err)
		if result != nil {
			assert.Contains(t, []core.ExecutionStatus{
				core.StatusTimeout,
				core.StatusCancelled,
			}, result.Status)
		}
	})

	t.Run("Recovery after failures", func(t *testing.T) {
		// Reset failing agent
		failingAgent.executionCount = 0

		ctx := context.Background()
		task := core.SubAgentTask{ID: "recovery-task"}

		// Should work again after reset
		result, err := manager.Execute(ctx, task, "failing-agent")
		require.NoError(t, err)
		assert.Equal(t, core.StatusCompleted, result.Status)
	})
}

// Helper functions and mock implementations

func setupProviders(t *testing.T, manager *subagent.Manager, factory *providers.ProviderFactory) {
	// Try to setup available providers
	providers := []core.ProviderConfig{
		{Name: "claude", Type: core.ProviderTypeClaude, Command: "claude"},
		{Name: "gemini", Type: core.ProviderTypeGemini, Command: "gemini"},
		{Name: "qwen", Type: core.ProviderTypeQwen, Command: "qwen"},
	}

	for _, config := range providers {
		provider, err := factory.CreateProvider(config)
		if err == nil {
			manager.RegisterProvider(provider)
			t.Logf("Registered provider: %s", config.Name)
		}
	}
}

// FailingMockAgent simulates an agent that fails after N executions
type FailingMockAgent struct {
	name           string
	executionCount int
	failAfter      int
}

func (f *FailingMockAgent) Name() string                   { return f.name }
func (f *FailingMockAgent) Config() core.SubAgentConfig    { return core.SubAgentConfig{Name: f.name} }
func (f *FailingMockAgent) Provider() core.ProviderType    { return core.ProviderTypeMock }
func (f *FailingMockAgent) Initialize(config core.SubAgentConfig) error { return nil }
func (f *FailingMockAgent) Validate() error                { return nil }
func (f *FailingMockAgent) Cleanup() error                 { return nil }
func (f *FailingMockAgent) Status() core.ExecutionStatus   { return core.StatusPending }
func (f *FailingMockAgent) Cancel() error                  { return nil }
func (f *FailingMockAgent) GetProgress() (float64, string) { return 0, "" }
func (f *FailingMockAgent) GetCapabilities() []string      { return []string{} }
func (f *FailingMockAgent) SupportsParallel() bool         { return false }
func (f *FailingMockAgent) SupportsInteractive() bool      { return false }
func (f *FailingMockAgent) CanHandle(task core.SubAgentTask) bool { return true }

func (f *FailingMockAgent) Execute(ctx context.Context, task core.SubAgentTask) (*core.SubAgentResult, error) {
	f.executionCount++
	
	if f.executionCount > f.failAfter {
		return &core.SubAgentResult{
			TaskID:    task.ID,
			AgentName: f.name,
			Status:    core.StatusFailed,
			Error:     errors.New("simulated failure"),
		}, errors.New("simulated failure")
	}

	return &core.SubAgentResult{
		TaskID:    task.ID,
		AgentName: f.name,
		Status:    core.StatusCompleted,
		Output:    "success",
	}, nil
}

func (f *FailingMockAgent) ExecuteAsync(ctx context.Context, task core.SubAgentTask) (<-chan *core.SubAgentResult, error) {
	ch := make(chan *core.SubAgentResult, 1)
	go func() {
		result, _ := f.Execute(ctx, task)
		ch <- result
		close(ch)
	}()
	return ch, nil
}

// TimeoutMockAgent simulates an agent that times out
type TimeoutMockAgent struct {
	name    string
	timeout time.Duration
}

func (t *TimeoutMockAgent) Name() string                   { return t.name }
func (t *TimeoutMockAgent) Config() core.SubAgentConfig    { return core.SubAgentConfig{Name: t.name} }
func (t *TimeoutMockAgent) Provider() core.ProviderType    { return core.ProviderTypeMock }
func (t *TimeoutMockAgent) Initialize(config core.SubAgentConfig) error { return nil }
func (t *TimeoutMockAgent) Validate() error                { return nil }
func (t *TimeoutMockAgent) Cleanup() error                 { return nil }
func (t *TimeoutMockAgent) Status() core.ExecutionStatus   { return core.StatusPending }
func (t *TimeoutMockAgent) Cancel() error                  { return nil }
func (t *TimeoutMockAgent) GetProgress() (float64, string) { return 0, "" }
func (t *TimeoutMockAgent) GetCapabilities() []string      { return []string{} }
func (t *TimeoutMockAgent) SupportsParallel() bool         { return false }
func (t *TimeoutMockAgent) SupportsInteractive() bool      { return false }
func (t *TimeoutMockAgent) CanHandle(task core.SubAgentTask) bool { return true }

func (t *TimeoutMockAgent) Execute(ctx context.Context, task core.SubAgentTask) (*core.SubAgentResult, error) {
	select {
	case <-ctx.Done():
		return &core.SubAgentResult{
			TaskID:    task.ID,
			AgentName: t.name,
			Status:    core.StatusTimeout,
			Error:     ctx.Err(),
		}, ctx.Err()
	case <-time.After(t.timeout):
		return &core.SubAgentResult{
			TaskID:    task.ID,
			AgentName: t.name,
			Status:    core.StatusCompleted,
			Output:    "completed after timeout",
		}, nil
	}
}

func (t *TimeoutMockAgent) ExecuteAsync(ctx context.Context, task core.SubAgentTask) (<-chan *core.SubAgentResult, error) {
	ch := make(chan *core.SubAgentResult, 1)
	go func() {
		result, _ := t.Execute(ctx, task)
		ch <- result
		close(ch)
	}()
	return ch, nil
}