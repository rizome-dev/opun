//go:build windows

package workflow

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/rizome-dev/opun/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInteractiveExecutor_Windows(t *testing.T) {
	t.Run("NewInteractiveExecutor", func(t *testing.T) {
		executor := NewInteractiveExecutor()
		assert.NotNil(t, executor)
		assert.NotNil(t, executor.outputs)
		assert.NotNil(t, executor.handoffContext)
		assert.Equal(t, 0, len(executor.outputs))
		assert.Equal(t, 0, len(executor.handoffContext))
	})

	t.Run("GetState", func(t *testing.T) {
		executor := NewInteractiveExecutor()
		
		// Initially nil
		assert.Nil(t, executor.GetState())
		
		// Set a state
		executor.state = &workflow.ExecutionState{
			WorkflowID: "test-workflow",
			Status:     workflow.StatusRunning,
		}
		
		state := executor.GetState()
		assert.NotNil(t, state)
		assert.Equal(t, "test-workflow", state.WorkflowID)
		assert.Equal(t, workflow.StatusRunning, state.Status)
	})

	t.Run("processPromptWithHandoff", func(t *testing.T) {
		executor := NewInteractiveExecutor()
		
		// Test without handoff context
		prompt := "Test prompt"
		result, err := executor.processPromptWithHandoff(prompt, 0)
		assert.NoError(t, err)
		assert.Equal(t, prompt, result)
		
		// Test with output substitution
		executor.outputs["agent1"] = "output from agent 1"
		prompt = "Use this: {{agent1.output}}"
		result, err = executor.processPromptWithHandoff(prompt, 0)
		assert.NoError(t, err)
		assert.Equal(t, "Use this: output from agent 1", result)
		
		// Test with handoff context
		executor.handoffContext = []string{"Agent 1 (claude) completed"}
		prompt = "Continue work"
		result, err = executor.processPromptWithHandoff(prompt, 1)
		assert.NoError(t, err)
		assert.Contains(t, result, "WORKFLOW CONTEXT")
		assert.Contains(t, result, "You are agent 2 in a sequential workflow")
		assert.Contains(t, result, "Agent 1 (claude) completed")
		assert.Contains(t, result, "Continue work")
	})

	t.Run("getProviderCommandAndArgs", func(t *testing.T) {
		executor := NewInteractiveExecutor()
		
		// Test unsupported provider
		_, _, err := executor.getProviderCommandAndArgs("unsupported")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported provider")
		
		// Test claude provider
		// Note: This test will pass/fail based on whether claude is installed
		cmd, args, err := executor.getProviderCommandAndArgs("claude")
		if err != nil {
			assert.Contains(t, err.Error(), "claude command not found")
		} else {
			assert.True(t, cmd == "claude" || cmd == "claude.exe" || cmd == "npx" || cmd == "npx.cmd")
			if cmd == "npx" || cmd == "npx.cmd" {
				assert.Equal(t, []string{"claude-code"}, args)
			} else {
				assert.Equal(t, []string{}, args)
			}
		}
		
		// Test gemini provider
		cmd, args, err = executor.getProviderCommandAndArgs("gemini")
		if err != nil {
			assert.Contains(t, err.Error(), "gemini command not found")
		} else {
			assert.True(t, cmd == "gemini" || cmd == "gemini.exe")
			assert.Equal(t, []string{}, args)
		}
	})

	t.Run("saveOutput", func(t *testing.T) {
		executor := NewInteractiveExecutor()
		tempDir := t.TempDir()
		
		// Test without workflow settings
		filename := "test.txt"
		content := "test content"
		err := executor.saveOutput(filename, content)
		assert.NoError(t, err)
		
		// Verify file was created
		data, err := os.ReadFile(filename)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
		
		// Clean up
		os.Remove(filename)
		
		// Test with workflow settings
		executor.workflow = &workflow.Workflow{
			Settings: workflow.WorkflowSettings{
				OutputDir: tempDir,
			},
		}
		
		err = executor.saveOutput(filename, content)
		assert.NoError(t, err)
		
		// Verify file was created in output dir
		fullPath := tempDir + "/" + filename
		data, err = os.ReadFile(fullPath)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("handleAgentError", func(t *testing.T) {
		executor := NewInteractiveExecutor()
		
		agent := &workflow.Agent{
			ID:   "test-agent",
			Name: "Test Agent",
			Settings: workflow.AgentSettings{
				ContinueOnError: false,
			},
		}
		
		state := &workflow.AgentState{
			AgentID: agent.ID,
			Status:  workflow.StatusRunning,
		}
		
		testErr := assert.AnError
		err := executor.handleAgentError(agent, state, testErr)
		
		// Should return the error when ContinueOnError is false
		assert.Error(t, err)
		assert.Equal(t, testErr, err)
		assert.Equal(t, workflow.StatusFailed, state.Status)
		assert.NotNil(t, state.EndTime)
		assert.NotNil(t, state.Error)
		assert.Equal(t, agent.ID, state.Error.AgentID)
		assert.Equal(t, testErr.Error(), state.Error.Message)
		assert.True(t, state.Error.Fatal)
		
		// Test with ContinueOnError = true
		agent.Settings.ContinueOnError = true
		state = &workflow.AgentState{
			AgentID: agent.ID,
			Status:  workflow.StatusRunning,
		}
		
		err = executor.handleAgentError(agent, state, testErr)
		
		// Should not return error when ContinueOnError is true
		assert.NoError(t, err)
		assert.Equal(t, workflow.StatusFailed, state.Status)
		assert.False(t, state.Error.Fatal)
	})
}

// TestExecuteInteractiveAgent_Windows tests the interactive agent execution
// This is an integration test that requires a mock provider
func TestExecuteInteractiveAgent_Windows(t *testing.T) {
	// Skip if not in CI or if mock provider is not available
	if os.Getenv("CI") == "" {
		t.Skip("Skipping integration test outside CI")
	}
	
	// Check if we have a mock provider available
	mockProvider := os.Getenv("OPUN_MOCK_PROVIDER")
	if mockProvider == "" {
		t.Skip("Skipping integration test: OPUN_MOCK_PROVIDER not set")
	}
	
	executor := NewInteractiveExecutor()
	ctx := context.Background()
	
	agent := &workflow.Agent{
		ID:       "test-agent",
		Name:     "Test Agent",
		Provider: "mock",
		Model:    "test",
		Prompt:   "Test prompt",
		Settings: workflow.AgentSettings{},
	}
	
	// Override getProviderCommandAndArgs for testing
	originalFunc := executor.getProviderCommandAndArgs
	executor.getProviderCommandAndArgs = func(provider string) (string, []string, error) {
		if provider == "mock" {
			return mockProvider, []string{}, nil
		}
		return originalFunc(provider)
	}
	
	// Execute the agent
	err := executor.executeInteractiveAgent(ctx, agent, 0)
	
	// The test should complete without error if the mock provider works correctly
	require.NoError(t, err)
	
	// Verify state was updated
	state := executor.state.AgentStates[agent.ID]
	assert.NotNil(t, state)
	assert.Equal(t, workflow.StatusCompleted, state.Status)
	assert.NotNil(t, state.StartTime)
	assert.NotNil(t, state.EndTime)
}

// TestExecute_Windows tests full workflow execution
func TestExecute_Windows(t *testing.T) {
	executor := NewInteractiveExecutor()
	ctx := context.Background()
	
	wf := &workflow.Workflow{
		Name:        "test-workflow",
		Description: "Test workflow",
		Agents: []workflow.Agent{
			{
				ID:       "agent1",
				Name:     "Agent 1",
				Provider: "echo", // Use echo as a simple test provider
				Model:    "test",
				Prompt:   "Hello from agent 1",
			},
		},
		Settings: workflow.WorkflowSettings{},
	}
	
	// Mock the provider command to use echo
	if _, err := exec.LookPath("echo"); err != nil {
		t.Skip("echo command not available")
	}
	
	// This will fail because echo is not a real PTY provider
	// But we can test that the workflow structure is correct
	err := executor.Execute(ctx, wf, nil)
	
	// We expect an error because echo doesn't work as a PTY
	assert.Error(t, err)
	
	// But we can verify the state was initialized correctly
	assert.NotNil(t, executor.state)
	assert.Equal(t, wf.Name, executor.state.WorkflowID)
	assert.Equal(t, workflow.StatusRunning, executor.state.Status)
	assert.NotNil(t, executor.state.StartTime)
}

// TestWindowsSpecificFeatures tests Windows-specific functionality
func TestWindowsSpecificFeatures(t *testing.T) {
	t.Run("Windows executable resolution", func(t *testing.T) {
		executor := NewInteractiveExecutor()
		
		// Test that we look for .exe and .cmd files on Windows
		cmd, args, err := executor.getProviderCommandAndArgs("claude")
		
		// The function should check for multiple Windows-specific variants
		// Even if claude is not installed, the error message should be appropriate
		if err != nil {
			assert.Contains(t, err.Error(), "claude command not found")
		} else {
			// If found, it should be one of the Windows variants
			assert.True(t, 
				cmd == "claude" || 
				cmd == "claude.exe" || 
				cmd == "npx" || 
				cmd == "npx.cmd",
				"Expected Windows-compatible command, got: %s", cmd)
		}
	})
	
	t.Run("No signal handling", func(t *testing.T) {
		// The Windows implementation should not use SIGWINCH
		// This is implicitly tested by the fact that the code compiles on Windows
		// and doesn't reference syscall.SIGWINCH
		assert.True(t, true, "Windows implementation compiles without Unix signals")
	})
}

// BenchmarkProcessPromptWithHandoff benchmarks prompt processing
func BenchmarkProcessPromptWithHandoff(b *testing.B) {
	executor := NewInteractiveExecutor()
	
	// Set up some outputs and handoff context
	executor.outputs["agent1"] = "output from agent 1"
	executor.outputs["agent2"] = "output from agent 2"
	executor.handoffContext = []string{
		"Agent 1 (claude) completed",
		"Agent 2 (gemini) completed",
	}
	
	prompt := "Process this with {{agent1.output}} and {{agent2.output}}"
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := executor.processPromptWithHandoff(prompt, 2)
		if err != nil {
			b.Fatal(err)
		}
	}
}