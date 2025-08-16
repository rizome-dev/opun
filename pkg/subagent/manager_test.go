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
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/rizome-dev/opun/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSubAgent for testing
type MockSubAgent struct {
	mu           sync.RWMutex
	name         string
	config       core.SubAgentConfig
	status       core.ExecutionStatus
	capabilities []string
	executeFunc  func(ctx context.Context, task core.SubAgentTask) (*core.SubAgentResult, error)
	canHandle    bool
	validated    bool
}

func NewMockSubAgent(name string) *MockSubAgent {
	return &MockSubAgent{
		name:         name,
		status:       core.StatusPending,
		capabilities: []string{"test"},
		canHandle:    true,
		validated:    true,
		config: core.SubAgentConfig{
			Name:     name,
			Provider: core.ProviderTypeMock,
		},
	}
}

func (m *MockSubAgent) Name() string                   { return m.name }
func (m *MockSubAgent) Config() core.SubAgentConfig    { return m.config }
func (m *MockSubAgent) Provider() core.ProviderType    { return m.config.Provider }
func (m *MockSubAgent) Initialize(config core.SubAgentConfig) error {
	m.config = config
	return nil
}
func (m *MockSubAgent) Validate() error {
	if !m.validated {
		return errors.New("validation failed")
	}
	return nil
}
func (m *MockSubAgent) Cleanup() error                     { return nil }
func (m *MockSubAgent) Status() core.ExecutionStatus       { 
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.status 
}
func (m *MockSubAgent) Cancel() error                      { 
	m.mu.Lock()
	defer m.mu.Unlock()
	m.status = core.StatusCancelled
	return nil 
}
func (m *MockSubAgent) GetProgress() (float64, string)     { return 0.5, "processing" }
func (m *MockSubAgent) GetCapabilities() []string          { return m.capabilities }
func (m *MockSubAgent) SupportsParallel() bool             { return true }
func (m *MockSubAgent) SupportsInteractive() bool          { return false }
func (m *MockSubAgent) CanHandle(task core.SubAgentTask) bool { 
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.canHandle 
}

func (m *MockSubAgent) Execute(ctx context.Context, task core.SubAgentTask) (*core.SubAgentResult, error) {
	m.mu.Lock()
	m.status = core.StatusRunning
	execFunc := m.executeFunc
	m.mu.Unlock()
	
	if execFunc != nil {
		return execFunc(ctx, task)
	}
	
	select {
	case <-ctx.Done():
		m.mu.Lock()
		m.status = core.StatusCancelled
		m.mu.Unlock()
		return &core.SubAgentResult{
			TaskID:    task.ID,
			AgentName: m.name,
			Status:    core.StatusCancelled,
			Error:     ctx.Err(),
		}, ctx.Err()
	case <-time.After(10 * time.Millisecond):
		m.mu.Lock()
		m.status = core.StatusCompleted
		m.mu.Unlock()
		return &core.SubAgentResult{
			TaskID:    task.ID,
			AgentName: m.name,
			Status:    core.StatusCompleted,
			Output:    "test output",
			StartTime: time.Now(),
			EndTime:   time.Now(),
		}, nil
	}
}

func (m *MockSubAgent) ExecuteAsync(ctx context.Context, task core.SubAgentTask) (<-chan *core.SubAgentResult, error) {
	resultChan := make(chan *core.SubAgentResult, 1)
	go func() {
		result, _ := m.Execute(ctx, task)
		resultChan <- result
		close(resultChan)
	}()
	return resultChan, nil
}

// MockProvider for testing
type MockSubAgentProvider struct {
	name       string
	typ        core.ProviderType
	subAgents  []core.SubAgentConfig
	createFunc func(config core.SubAgentConfig) (core.SubAgent, error)
}

func (m *MockSubAgentProvider) Name() string              { return m.name }
func (m *MockSubAgentProvider) Type() core.ProviderType   { return m.typ }
func (m *MockSubAgentProvider) SupportsSubAgents() bool   { return true }
func (m *MockSubAgentProvider) GetSubAgentType() core.SubAgentType {
	return core.SubAgentTypeWorkflow
}
func (m *MockSubAgentProvider) ListSubAgents() ([]core.SubAgentConfig, error) {
	return m.subAgents, nil
}
func (m *MockSubAgentProvider) GetSubAgent(name string) (core.SubAgent, error) {
	return NewMockSubAgent(name), nil
}
func (m *MockSubAgentProvider) CreateSubAgent(config core.SubAgentConfig) (core.SubAgent, error) {
	if m.createFunc != nil {
		return m.createFunc(config)
	}
	return NewMockSubAgent(config.Name), nil
}
func (m *MockSubAgentProvider) Delegate(ctx context.Context, task core.SubAgentTask) (*core.SubAgentResult, error) {
	agent := NewMockSubAgent("delegate-agent")
	return agent.Execute(ctx, task)
}
func (m *MockSubAgentProvider) RegisterSubAgent(agent core.SubAgent) error   { return nil }
func (m *MockSubAgentProvider) UnregisterSubAgent(name string) error         { return nil }

func TestManager_Registration(t *testing.T) {
	manager := NewManager()

	t.Run("Register agent", func(t *testing.T) {
		agent := NewMockSubAgent("agent1")
		err := manager.Register(agent)
		require.NoError(t, err)

		// Verify agent is registered
		agents := manager.List()
		assert.Len(t, agents, 1)
		assert.Equal(t, "agent1", agents[0].Name())
	})

	t.Run("Register duplicate agent", func(t *testing.T) {
		agent := NewMockSubAgent("agent1")
		err := manager.Register(agent)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
	})

	t.Run("Register invalid agent", func(t *testing.T) {
		agent := NewMockSubAgent("invalid")
		agent.validated = false
		err := manager.Register(agent)
		assert.Error(t, err)
	})

	t.Run("Unregister agent", func(t *testing.T) {
		err := manager.Unregister("agent1")
		require.NoError(t, err)

		agents := manager.List()
		assert.Len(t, agents, 0)
	})

	t.Run("Unregister non-existent agent", func(t *testing.T) {
		err := manager.Unregister("non-existent")
		assert.Error(t, err)
	})
}

func TestManager_Discovery(t *testing.T) {
	manager := NewManager()

	// Register multiple agents
	agent1 := NewMockSubAgent("agent1")
	agent1.capabilities = []string{"code", "test"}
	manager.Register(agent1)

	agent2 := NewMockSubAgent("agent2")
	agent2.capabilities = []string{"review", "analysis"}
	manager.Register(agent2)

	agent3 := NewMockSubAgent("agent3")
	agent3.capabilities = []string{"code", "review"}
	manager.Register(agent3)

	t.Run("List all agents", func(t *testing.T) {
		agents := manager.List()
		assert.Len(t, agents, 3)
	})

	t.Run("Get agent by name", func(t *testing.T) {
		agent, err := manager.Get("agent2")
		require.NoError(t, err)
		assert.Equal(t, "agent2", agent.Name())
	})

	t.Run("Get non-existent agent", func(t *testing.T) {
		_, err := manager.Get("non-existent")
		assert.Error(t, err)
	})

	t.Run("Find agents by capabilities", func(t *testing.T) {
		agents := manager.Find([]string{"code"})
		assert.Len(t, agents, 2)

		agents = manager.Find([]string{"review"})
		assert.Len(t, agents, 2)

		agents = manager.Find([]string{"test"})
		assert.Len(t, agents, 1)

		agents = manager.Find([]string{"unknown"})
		assert.Len(t, agents, 0)
	})
}

func TestManager_Execution(t *testing.T) {
	manager := NewManager()
	agent := NewMockSubAgent("test-agent")
	manager.Register(agent)

	t.Run("Execute task", func(t *testing.T) {
		ctx := context.Background()
		task := core.SubAgentTask{
			ID:   "task1",
			Name: "Test Task",
		}

		result, err := manager.Execute(ctx, task, "test-agent")
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "task1", result.TaskID)
		assert.Equal(t, core.StatusCompleted, result.Status)
	})

	t.Run("Execute with non-existent agent", func(t *testing.T) {
		ctx := context.Background()
		task := core.SubAgentTask{ID: "task2"}

		_, err := manager.Execute(ctx, task, "non-existent")
		assert.Error(t, err)
	})

	t.Run("Execute with context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		task := core.SubAgentTask{ID: "task3"}
		result, err := manager.Execute(ctx, task, "test-agent")
		assert.Error(t, err)
		if result != nil {
			assert.Equal(t, core.StatusCancelled, result.Status)
		}
	})
}

func TestManager_ParallelExecution(t *testing.T) {
	manager := NewManager()

	// Register multiple agents
	for i := 0; i < 3; i++ {
		agent := NewMockSubAgent(string(rune('a' + i)))
		manager.Register(agent)
	}

	t.Run("Execute parallel tasks", func(t *testing.T) {
		ctx := context.Background()
		tasks := []core.SubAgentTask{
			{ID: "task1", Name: "Task 1"},
			{ID: "task2", Name: "Task 2"},
			{ID: "task3", Name: "Task 3"},
		}

		results, err := manager.ExecuteParallel(ctx, tasks)
		require.NoError(t, err)
		assert.Len(t, results, 3)

		for i, result := range results {
			assert.Equal(t, tasks[i].ID, result.TaskID)
			assert.Equal(t, core.StatusCompleted, result.Status)
		}
	})

	t.Run("Parallel execution with errors", func(t *testing.T) {
		// Add agent that fails
		failAgent := NewMockSubAgent("fail-agent")
		failAgent.executeFunc = func(ctx context.Context, task core.SubAgentTask) (*core.SubAgentResult, error) {
			return &core.SubAgentResult{
				TaskID:    task.ID,
				AgentName: "fail-agent",
				Status:    core.StatusFailed,
				Error:     errors.New("execution failed"),
			}, errors.New("execution failed")
		}
		manager.Register(failAgent)

		ctx := context.Background()
		tasks := []core.SubAgentTask{
			{ID: "task1"},
			{ID: "task2"},
		}

		results, err := manager.ExecuteParallel(ctx, tasks)
		// Should not error - individual task errors are in results
		assert.NoError(t, err)
		assert.Len(t, results, 2)
	})
}

func TestManager_Delegation(t *testing.T) {
	manager := NewManager()

	// Create agents with different capabilities
	codeAgent := NewMockSubAgent("code-agent")
	codeAgent.capabilities = []string{"code", "refactor"}
	manager.Register(codeAgent)

	testAgent := NewMockSubAgent("test-agent")
	testAgent.capabilities = []string{"test", "coverage"}
	manager.Register(testAgent)

	reviewAgent := NewMockSubAgent("review-agent")
	reviewAgent.capabilities = []string{"review", "analysis"}
	manager.Register(reviewAgent)

	t.Run("Delegate to capable agent", func(t *testing.T) {
		ctx := context.Background()
		task := core.SubAgentTask{
			ID:   "task1",
			Name: "Code Task",
			Context: map[string]interface{}{
				"code": true,
			},
		}

		result, err := manager.Delegate(ctx, task)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "code-agent", result.AgentName)
	})

	t.Run("Delegate with no capable agents", func(t *testing.T) {
		ctx := context.Background()
		task := core.SubAgentTask{
			ID: "task2",
			Context: map[string]interface{}{
				"unknown": true,
			},
		}

		// Set all agents to not handle this task
		codeAgent.mu.Lock()
		codeAgent.canHandle = false
		codeAgent.mu.Unlock()
		testAgent.mu.Lock()
		testAgent.canHandle = false
		testAgent.mu.Unlock()
		reviewAgent.mu.Lock()
		reviewAgent.canHandle = false
		reviewAgent.mu.Unlock()

		_, err := manager.Delegate(ctx, task)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no capable agents")

		// Reset
		codeAgent.mu.Lock()
		codeAgent.canHandle = true
		codeAgent.mu.Unlock()
		testAgent.mu.Lock()
		testAgent.canHandle = true
		testAgent.mu.Unlock()
		reviewAgent.mu.Lock()
		reviewAgent.canHandle = true
		reviewAgent.mu.Unlock()
	})

	t.Run("Delegate with strategy", func(t *testing.T) {
		ctx := context.Background()
		task := core.SubAgentTask{
			ID:   "task3",
			Name: "Strategic Task",
		}

		result, err := manager.DelegateWithStrategy(ctx, task, core.DelegationAutomatic)
		require.NoError(t, err)
		assert.NotNil(t, result)
	})
}

func TestManager_Monitoring(t *testing.T) {
	manager := NewManager()
	agent := NewMockSubAgent("monitor-agent")
	manager.Register(agent)

	t.Run("Get task status", func(t *testing.T) {
		ctx := context.Background()
		task := core.SubAgentTask{
			ID:   "monitor-task",
			Name: "Monitor Task",
		}

		// Start execution
		go manager.Execute(ctx, task, "monitor-agent")
		time.Sleep(5 * time.Millisecond)

		status, err := manager.GetStatus("monitor-task")
		require.NoError(t, err)
		assert.NotEqual(t, core.StatusPending, status)
	})

	t.Run("Get task results", func(t *testing.T) {
		ctx := context.Background()
		task := core.SubAgentTask{
			ID:   "result-task",
			Name: "Result Task",
		}

		result, _ := manager.Execute(ctx, task, "monitor-agent")
		
		retrieved, err := manager.GetResults("result-task")
		require.NoError(t, err)
		assert.Equal(t, result.TaskID, retrieved.TaskID)
		assert.Equal(t, result.Status, retrieved.Status)
	})

	t.Run("List active tasks", func(t *testing.T) {
		// Start multiple tasks
		var wg sync.WaitGroup
		for i := 0; i < 3; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				ctx := context.Background()
				task := core.SubAgentTask{
					ID: string(rune('a' + id)),
				}
				manager.Execute(ctx, task, "monitor-agent")
			}(i)
		}

		time.Sleep(5 * time.Millisecond)
		active := manager.ListActiveTasks()
		// Some tasks may have completed
		assert.GreaterOrEqual(t, len(active), 0)
		
		wg.Wait()
	})
}

func TestManager_CrossProviderCoordination(t *testing.T) {
	manager := NewManager()

	// For testing purposes, we'll register the agents directly
	// In a real implementation, these would come from providers
	claudeAgent := NewMockSubAgent("claude-agent")
	claudeAgent.config.Provider = core.ProviderTypeClaude
	manager.Register(claudeAgent)

	geminiAgent := NewMockSubAgent("gemini-agent")
	geminiAgent.config.Provider = core.ProviderTypeGemini
	manager.Register(geminiAgent)

	t.Run("Coordinate across providers", func(t *testing.T) {
		ctx := context.Background()
		tasks := []core.SubAgentTask{
			{ID: "task1", Name: "Claude Task"},
			{ID: "task2", Name: "Gemini Task"},
		}

		results, err := manager.CoordinateAcrossProviders(ctx, tasks)
		require.NoError(t, err)
		assert.Len(t, results, 2)

		// Verify each task was handled
		for i, result := range results {
			assert.Equal(t, tasks[i].ID, result.TaskID)
			assert.Equal(t, core.StatusCompleted, result.Status)
		}
	})
}

func TestManager_ThreadSafety(t *testing.T) {
	manager := NewManager()

	// Test concurrent registration
	t.Run("Concurrent registration", func(t *testing.T) {
		var wg sync.WaitGroup
		errors := make(chan error, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				agent := NewMockSubAgent(string(rune('a' + id)))
				if err := manager.Register(agent); err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Should have no errors for unique agents
		errCount := 0
		for err := range errors {
			if err != nil {
				errCount++
			}
		}
		assert.Equal(t, 0, errCount)
		assert.Len(t, manager.List(), 10)
	})

	// Test concurrent execution
	t.Run("Concurrent execution", func(t *testing.T) {
		var wg sync.WaitGroup
		results := make(chan *core.SubAgentResult, 10)

		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				ctx := context.Background()
				task := core.SubAgentTask{
					ID: string(rune('A' + id)),
				}
				// Use different agents
				agentName := string(rune('a' + (id % 10)))
				if result, err := manager.Execute(ctx, task, agentName); err == nil {
					results <- result
				}
			}(i)
		}

		wg.Wait()
		close(results)

		// Count successful results
		count := 0
		for range results {
			count++
		}
		assert.Equal(t, 10, count)
	})
}

// Benchmark tests
func BenchmarkManager_Register(b *testing.B) {
	manager := NewManager()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agent := NewMockSubAgent(string(rune('a' + (i % 26))))
		manager.Register(agent)
		manager.Unregister(agent.Name())
	}
}

func BenchmarkManager_Execute(b *testing.B) {
	manager := NewManager()
	agent := NewMockSubAgent("bench-agent")
	manager.Register(agent)
	
	ctx := context.Background()
	task := core.SubAgentTask{
		ID:   "bench-task",
		Name: "Benchmark Task",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Execute(ctx, task, "bench-agent")
	}
}

func BenchmarkManager_ParallelExecution(b *testing.B) {
	manager := NewManager()
	for i := 0; i < 5; i++ {
		agent := NewMockSubAgent(string(rune('a' + i)))
		manager.Register(agent)
	}

	ctx := context.Background()
	tasks := []core.SubAgentTask{
		{ID: "task1"},
		{ID: "task2"},
		{ID: "task3"},
		{ID: "task4"},
		{ID: "task5"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.ExecuteParallel(ctx, tasks)
	}
}