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
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockSubAgent implements SubAgent interface for testing
type MockSubAgent struct {
	name         string
	config       SubAgentConfig
	status       ExecutionStatus
	capabilities []string
	executeFunc  func(ctx context.Context, task SubAgentTask) (*SubAgentResult, error)
}

func NewMockSubAgent(name string) *MockSubAgent {
	return &MockSubAgent{
		name:         name,
		status:       StatusPending,
		capabilities: []string{"test", "mock"},
		config: SubAgentConfig{
			Name:     name,
			Type:     SubAgentTypeWorkflow,
			Provider: ProviderTypeMock,
		},
	}
}

func (m *MockSubAgent) Name() string                     { return m.name }
func (m *MockSubAgent) Config() SubAgentConfig           { return m.config }
func (m *MockSubAgent) Provider() ProviderType           { return m.config.Provider }
func (m *MockSubAgent) Initialize(config SubAgentConfig) error {
	m.config = config
	return nil
}
func (m *MockSubAgent) Validate() error {
	if m.config.Name == "" {
		return errors.New("name is required")
	}
	return nil
}
func (m *MockSubAgent) Cleanup() error              { return nil }
func (m *MockSubAgent) Status() ExecutionStatus     { return m.status }
func (m *MockSubAgent) Cancel() error               { m.status = StatusCancelled; return nil }
func (m *MockSubAgent) GetProgress() (float64, string) { return 0.5, "processing" }
func (m *MockSubAgent) GetCapabilities() []string   { return m.capabilities }
func (m *MockSubAgent) SupportsParallel() bool      { return true }
func (m *MockSubAgent) SupportsInteractive() bool   { return false }

func (m *MockSubAgent) Execute(ctx context.Context, task SubAgentTask) (*SubAgentResult, error) {
	m.status = StatusRunning
	defer func() { m.status = StatusCompleted }()
	
	if m.executeFunc != nil {
		return m.executeFunc(ctx, task)
	}
	
	return &SubAgentResult{
		TaskID:    task.ID,
		AgentName: m.name,
		Status:    StatusCompleted,
		Output:    "test output",
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Duration:  time.Millisecond * 100,
	}, nil
}

func (m *MockSubAgent) ExecuteAsync(ctx context.Context, task SubAgentTask) (<-chan *SubAgentResult, error) {
	resultChan := make(chan *SubAgentResult, 1)
	go func() {
		result, _ := m.Execute(ctx, task)
		resultChan <- result
		close(resultChan)
	}()
	return resultChan, nil
}

func (m *MockSubAgent) CanHandle(task SubAgentTask) bool {
	for _, cap := range m.capabilities {
		for _, taskCap := range task.Context {
			if cap == taskCap {
				return true
			}
		}
	}
	return len(m.capabilities) > 0
}

func TestSubAgentConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		config  SubAgentConfig
		wantErr bool
	}{
		{
			name: "Valid config",
			config: SubAgentConfig{
				Name:        "test-agent",
				Type:        SubAgentTypeWorkflow,
				Provider:    ProviderTypeClaude,
				Model:       "sonnet",
				Strategy:    DelegationAutomatic,
				MaxRetries:  3,
				Timeout:     time.Minute * 5,
			},
			wantErr: false,
		},
		{
			name: "Config with MCP servers",
			config: SubAgentConfig{
				Name:       "mcp-agent",
				Type:       SubAgentTypeMCP,
				Provider:   ProviderTypeGemini,
				MCPServers: []string{"server1", "server2"},
			},
			wantErr: false,
		},
		{
			name: "Config with capabilities",
			config: SubAgentConfig{
				Name:         "capable-agent",
				Type:         SubAgentTypeDeclarative,
				Provider:     ProviderTypeQwen,
				Capabilities: []string{"code", "analysis", "testing"},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Basic validation - name and type should be present
			hasErr := tt.config.Name == "" || tt.config.Type == ""
			assert.Equal(t, tt.wantErr, hasErr)
		})
	}
}

func TestSubAgentTask_Creation(t *testing.T) {
	deadline := time.Now().Add(time.Hour)
	
	task := SubAgentTask{
		ID:          "task-123",
		Name:        "Test Task",
		Description: "A test task",
		Input:       "process this data",
		Context: map[string]interface{}{
			"environment": "test",
			"debug":       true,
		},
		Variables: map[string]interface{}{
			"timeout": 30,
			"retry":   3,
		},
		Constraints: []string{"memory < 1GB", "time < 5min"},
		Priority:    5,
		Deadline:    &deadline,
	}

	assert.Equal(t, "task-123", task.ID)
	assert.Equal(t, "Test Task", task.Name)
	assert.NotNil(t, task.Context)
	assert.Equal(t, true, task.Context["debug"])
	assert.NotNil(t, task.Deadline)
	assert.Equal(t, 5, task.Priority)
}

func TestSubAgentResult_Creation(t *testing.T) {
	startTime := time.Now()
	endTime := startTime.Add(time.Second * 10)
	
	result := SubAgentResult{
		TaskID:    "task-123",
		AgentName: "test-agent",
		Status:    StatusCompleted,
		Output:    "Task completed successfully",
		StartTime: startTime,
		EndTime:   endTime,
		Duration:  endTime.Sub(startTime),
		Metadata: map[string]interface{}{
			"tokens_used": 1000,
			"model":       "sonnet",
		},
		Artifacts: []SubAgentArtifact{
			{
				Name:        "output.txt",
				Type:        "file",
				Path:        "/tmp/output.txt",
				ContentType: "text/plain",
				Size:        1024,
				Created:     time.Now(),
			},
		},
	}

	assert.Equal(t, "task-123", result.TaskID)
	assert.Equal(t, StatusCompleted, result.Status)
	assert.Equal(t, time.Second*10, result.Duration)
	assert.Len(t, result.Artifacts, 1)
	assert.Equal(t, "output.txt", result.Artifacts[0].Name)
}

func TestExecutionStatus(t *testing.T) {
	tests := []struct {
		status   ExecutionStatus
		expected string
	}{
		{StatusPending, "pending"},
		{StatusRunning, "running"},
		{StatusCompleted, "completed"},
		{StatusFailed, "failed"},
		{StatusCancelled, "cancelled"},
		{StatusTimeout, "timeout"},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.status))
		})
	}
}

func TestMockSubAgent_Execution(t *testing.T) {
	agent := NewMockSubAgent("test-agent")
	
	t.Run("Basic execution", func(t *testing.T) {
		ctx := context.Background()
		task := SubAgentTask{
			ID:   "task-1",
			Name: "Test Task",
		}
		
		result, err := agent.Execute(ctx, task)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "task-1", result.TaskID)
		assert.Equal(t, "test-agent", result.AgentName)
		assert.Equal(t, StatusCompleted, result.Status)
	})

	t.Run("Async execution", func(t *testing.T) {
		ctx := context.Background()
		task := SubAgentTask{
			ID:   "task-2",
			Name: "Async Task",
		}
		
		resultChan, err := agent.ExecuteAsync(ctx, task)
		require.NoError(t, err)
		
		result := <-resultChan
		assert.NotNil(t, result)
		assert.Equal(t, "task-2", result.TaskID)
	})

	t.Run("Custom execution function", func(t *testing.T) {
		agent.executeFunc = func(ctx context.Context, task SubAgentTask) (*SubAgentResult, error) {
			return &SubAgentResult{
				TaskID:    task.ID,
				AgentName: agent.Name(),
				Status:    StatusFailed,
				Error:     errors.New("custom error"),
			}, errors.New("custom error")
		}
		
		ctx := context.Background()
		task := SubAgentTask{ID: "task-3"}
		
		result, err := agent.Execute(ctx, task)
		assert.Error(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, StatusFailed, result.Status)
	})
}

func TestMockSubAgent_Capabilities(t *testing.T) {
	agent := NewMockSubAgent("capable-agent")
	agent.capabilities = []string{"code", "test", "review"}
	
	t.Run("Get capabilities", func(t *testing.T) {
		caps := agent.GetCapabilities()
		assert.Len(t, caps, 3)
		assert.Contains(t, caps, "code")
		assert.Contains(t, caps, "test")
		assert.Contains(t, caps, "review")
	})

	t.Run("Can handle task", func(t *testing.T) {
		task := SubAgentTask{
			Context: map[string]interface{}{
				"code": true,
			},
		}
		assert.True(t, agent.CanHandle(task))
		
		task2 := SubAgentTask{
			Context: map[string]interface{}{
				"unknown": true,
			},
		}
		assert.False(t, agent.CanHandle(task2))
	})
}

func TestSubAgentType(t *testing.T) {
	tests := []struct {
		agentType SubAgentType
		expected  string
	}{
		{SubAgentTypeDeclarative, "declarative"},
		{SubAgentTypeProgrammatic, "programmatic"},
		{SubAgentTypeWorkflow, "workflow"},
		{SubAgentTypeMCP, "mcp"},
	}

	for _, tt := range tests {
		t.Run(string(tt.agentType), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.agentType))
		})
	}
}

func TestDelegationStrategy(t *testing.T) {
	tests := []struct {
		strategy DelegationStrategy
		expected string
	}{
		{DelegationAutomatic, "automatic"},
		{DelegationExplicit, "explicit"},
		{DelegationProactive, "proactive"},
	}

	for _, tt := range tests {
		t.Run(string(tt.strategy), func(t *testing.T) {
			assert.Equal(t, tt.expected, string(tt.strategy))
		})
	}
}

func TestSubAgentArtifact(t *testing.T) {
	artifact := SubAgentArtifact{
		Name:        "result.json",
		Type:        "file",
		Path:        "/tmp/result.json",
		Content:     []byte(`{"status": "ok"}`),
		ContentType: "application/json",
		Size:        17,
		Created:     time.Now(),
	}

	assert.Equal(t, "result.json", artifact.Name)
	assert.Equal(t, "file", artifact.Type)
	assert.Equal(t, int64(17), artifact.Size)
	assert.Equal(t, "application/json", artifact.ContentType)
	assert.NotNil(t, artifact.Content)
}

// Benchmark tests
func BenchmarkSubAgentExecution(b *testing.B) {
	agent := NewMockSubAgent("bench-agent")
	ctx := context.Background()
	task := SubAgentTask{
		ID:   "bench-task",
		Name: "Benchmark Task",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		agent.Execute(ctx, task)
	}
}

func BenchmarkSubAgentAsyncExecution(b *testing.B) {
	agent := NewMockSubAgent("bench-agent")
	ctx := context.Background()
	task := SubAgentTask{
		ID:   "bench-task",
		Name: "Benchmark Task",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resultChan, _ := agent.ExecuteAsync(ctx, task)
		<-resultChan
	}
}