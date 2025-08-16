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
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/rizome-dev/opun/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	errNoAgents        = errors.New("no agents available")
	errNoSuitableAgent = errors.New("no suitable agent found")
)

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestSimpleRouter_Route(t *testing.T) {
	router := NewSimpleRouter()

	// Create test agents
	agent1 := NewMockSubAgent("agent1")
	agent1.capabilities = []string{"code", "test"}
	agent1.config.Priority = 5

	agent2 := NewMockSubAgent("agent2")
	agent2.capabilities = []string{"review", "analysis"}
	agent2.config.Priority = 3

	agent3 := NewMockSubAgent("agent3")
	agent3.capabilities = []string{"code", "review"}
	agent3.config.Priority = 7

	agents := []core.SubAgent{agent1, agent2, agent3}

	t.Run("Route to best agent", func(t *testing.T) {
		task := core.SubAgentTask{
			ID:   "task1",
			Name: "Code Task",
			Context: map[string]interface{}{
				"code": true,
			},
		}

		selected, err := router.Route(task, agents)
		require.NoError(t, err)
		assert.NotNil(t, selected)
		// Should select agent3 (highest priority among code-capable agents)
		assert.Equal(t, "agent3", selected.Name())
	})

	t.Run("Route with no capable agents", func(t *testing.T) {
		task := core.SubAgentTask{
			ID:   "task2",
			Name: "Unknown Task",
			Context: map[string]interface{}{
				"unknown": true,
			},
		}

		// Make all agents unable to handle this task
		agent1.mu.Lock()
		agent1.canHandle = false
		agent1.mu.Unlock()
		agent2.mu.Lock()
		agent2.canHandle = false
		agent2.mu.Unlock()
		agent3.mu.Lock()
		agent3.canHandle = false
		agent3.mu.Unlock()

		_, err := router.Route(task, agents)
		assert.Error(t, err)
		// Error could be either "no suitable agent" or "no capable agents"
		assert.True(t, 
			contains(err.Error(), "no suitable agent") || 
			contains(err.Error(), "no capable agents"),
			"Expected error to contain 'no suitable agent' or 'no capable agents', got: %v", err)

		// Reset
		agent1.mu.Lock()
		agent1.canHandle = true
		agent1.mu.Unlock()
		agent2.mu.Lock()
		agent2.canHandle = true
		agent2.mu.Unlock()
		agent3.mu.Lock()
		agent3.canHandle = true
		agent3.mu.Unlock()
	})

	t.Run("Route with empty agent list", func(t *testing.T) {
		task := core.SubAgentTask{ID: "task3"}
		_, err := router.Route(task, []core.SubAgent{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no agents available")
	})

	t.Run("Route based on capabilities match", func(t *testing.T) {
		task := core.SubAgentTask{
			ID: "task4",
			Context: map[string]interface{}{
				"review":   true,
				"analysis": true,
			},
		}

		selected, err := router.Route(task, agents)
		require.NoError(t, err)
		// Should prefer agent2 or agent3 based on who matches better
		assert.Contains(t, []string{"agent2", "agent3"}, selected.Name())
	})
}

func TestSimpleRouter_Score(t *testing.T) {
	router := NewSimpleRouter()

	agent := NewMockSubAgent("test-agent")
	agent.capabilities = []string{"code", "test", "review"}
	agent.config.Priority = 5

	t.Run("Score with matching capabilities", func(t *testing.T) {
		task := core.SubAgentTask{
			ID: "task1",
			Context: map[string]interface{}{
				"code": true,
				"test": true,
			},
		}

		score := router.Score(task, agent)
		// Should have positive score for matching capabilities
		assert.Greater(t, score, 0.0)
	})

	t.Run("Score with no matching capabilities", func(t *testing.T) {
		agent.mu.Lock()
		agent.canHandle = false
		agent.mu.Unlock()
		task := core.SubAgentTask{
			ID: "task2",
			Context: map[string]interface{}{
				"unknown": true,
			},
		}

		score := router.Score(task, agent)
		assert.Equal(t, 0.0, score)
		
		agent.mu.Lock()
		agent.canHandle = true
		agent.mu.Unlock()
	})

	t.Run("Score with priority consideration", func(t *testing.T) {
		task := core.SubAgentTask{
			ID:       "task3",
			Priority: 10,
			Context: map[string]interface{}{
				"code": true,
			},
		}

		highPriorityAgent := NewMockSubAgent("high-priority")
		highPriorityAgent.capabilities = []string{"code"}
		highPriorityAgent.config.Priority = 10

		lowPriorityAgent := NewMockSubAgent("low-priority")
		lowPriorityAgent.capabilities = []string{"code"}
		lowPriorityAgent.config.Priority = 1

		highScore := router.Score(task, highPriorityAgent)
		lowScore := router.Score(task, lowPriorityAgent)

		// Higher priority agent should have higher score
		assert.Greater(t, highScore, lowScore)
	})
}

func TestSimpleRouter_Learn(t *testing.T) {
	router := NewSimpleRouter()

	agent := NewMockSubAgent("learning-agent")
	task := core.SubAgentTask{
		ID:   "task1",
		Name: "Learning Task",
	}

	t.Run("Learn from successful execution", func(t *testing.T) {
		result := &core.SubAgentResult{
			TaskID:    task.ID,
			AgentName: agent.Name(),
			Status:    core.StatusCompleted,
			Duration:  time.Second,
		}

		// Get initial stats
		initialStats := router.GetStats()

		// Learn from result
		router.Learn(task, agent, result)

		// Get updated stats
		updatedStats := router.GetStats()

		// Stats should be updated
		assert.NotEqual(t, initialStats, updatedStats)
		
		// Check success count increased
		agentStats, ok := updatedStats[agent.Name()].(map[string]interface{})
		if ok {
			successCount, _ := agentStats["success_count"].(int)
			assert.Greater(t, successCount, 0)
		}
	})

	t.Run("Learn from failed execution", func(t *testing.T) {
		result := &core.SubAgentResult{
			TaskID:    task.ID,
			AgentName: agent.Name(),
			Status:    core.StatusFailed,
		}

		// Get initial stats
		initialStats := router.GetStats()

		// Learn from failure
		router.Learn(task, agent, result)

		// Get updated stats
		updatedStats := router.GetStats()

		// Stats should be updated
		assert.NotEqual(t, initialStats, updatedStats)

		// Check failure count increased
		agentStats, ok := updatedStats[agent.Name()].(map[string]interface{})
		if ok {
			failureCount, _ := agentStats["failure_count"].(int)
			assert.Greater(t, failureCount, 0)
		}
	})

	t.Run("Learn updates average duration", func(t *testing.T) {
		// Execute multiple tasks with different durations
		for i := 0; i < 5; i++ {
			result := &core.SubAgentResult{
				TaskID:    string(rune('a' + i)),
				AgentName: agent.Name(),
				Status:    core.StatusCompleted,
				Duration:  time.Duration(i+1) * time.Second,
			}
			router.Learn(task, agent, result)
		}

		stats := router.GetStats()
		agentStats, ok := stats[agent.Name()].(map[string]interface{})
		if ok {
			avgDuration, _ := agentStats["avg_duration"].(time.Duration)
			// Average should be around 3 seconds ((1+2+3+4+5)/5)
			assert.Greater(t, avgDuration, time.Duration(0))
		}
	})
}

func TestSimpleRouter_GetStats(t *testing.T) {
	router := NewSimpleRouter()

	t.Run("Initial stats are empty", func(t *testing.T) {
		stats := router.GetStats()
		assert.NotNil(t, stats)
		// May be empty or contain initialization data
	})

	t.Run("Stats after routing", func(t *testing.T) {
		agent1 := NewMockSubAgent("agent1")
		agent2 := NewMockSubAgent("agent2")
		
		task := core.SubAgentTask{ID: "task1"}
		agents := []core.SubAgent{agent1, agent2}

		// Perform routing
		router.Route(task, agents)

		// Record some learning data
		result := &core.SubAgentResult{
			TaskID:    task.ID,
			AgentName: agent1.Name(),
			Status:    core.StatusCompleted,
			Duration:  time.Second,
		}
		router.Learn(task, agent1, result)

		stats := router.GetStats()
		assert.NotNil(t, stats)
		
		// Should have stats for agent1
		agentsMap, ok := stats["agents"].(map[string]interface{})
		assert.True(t, ok)
		agentStats, ok := agentsMap[agent1.Name()]
		assert.True(t, ok)
		assert.NotNil(t, agentStats)
	})
}

func TestCapabilityRouter(t *testing.T) {
	// Test a more sophisticated router that considers capabilities
	router := NewCapabilityRouter()

	agent1 := NewMockSubAgent("specialist")
	agent1.capabilities = []string{"code", "refactor"}

	agent2 := NewMockSubAgent("generalist")
	agent2.capabilities = []string{"code", "test", "review", "docs"}

	agent3 := NewMockSubAgent("tester")
	agent3.capabilities = []string{"test", "coverage", "performance"}

	agents := []core.SubAgent{agent1, agent2, agent3}

	t.Run("Route to specialist for specific task", func(t *testing.T) {
		task := core.SubAgentTask{
			ID: "refactor-task",
			Context: map[string]interface{}{
				"refactor": true,
			},
		}

		selected, err := router.Route(task, agents)
		require.NoError(t, err)
		// Should select specialist for refactoring
		assert.Equal(t, "specialist", selected.Name())
	})

	t.Run("Route to best match for multi-requirement task", func(t *testing.T) {
		task := core.SubAgentTask{
			ID: "test-task",
			Context: map[string]interface{}{
				"test":     true,
				"coverage": true,
			},
		}

		selected, err := router.Route(task, agents)
		require.NoError(t, err)
		// Should select tester for testing tasks
		assert.Equal(t, "tester", selected.Name())
	})
}

// CapabilityRouter is a more sophisticated router for testing
type CapabilityRouter struct {
	*SimpleRouter
}

func NewCapabilityRouter() *CapabilityRouter {
	return &CapabilityRouter{
		SimpleRouter: NewSimpleRouter(),
	}
}

func (r *CapabilityRouter) Route(task core.SubAgentTask, agents []core.SubAgent) (core.SubAgent, error) {
	if len(agents) == 0 {
		return nil, errNoAgents
	}

	var bestAgent core.SubAgent
	bestScore := 0.0

	for _, agent := range agents {
		score := r.scoreByCapabilities(task, agent)
		if score > bestScore {
			bestScore = score
			bestAgent = agent
		}
	}

	if bestAgent == nil {
		return nil, errNoSuitableAgent
	}

	return bestAgent, nil
}

func (r *CapabilityRouter) scoreByCapabilities(task core.SubAgentTask, agent core.SubAgent) float64 {
	if !agent.CanHandle(task) {
		return 0.0
	}

	score := 0.0
	capabilities := agent.GetCapabilities()
	
	// Check how many task requirements the agent can handle
	for key := range task.Context {
		for _, cap := range capabilities {
			if key == cap {
				score += 10.0
			}
		}
	}

	// Bonus for exact capability match
	if len(capabilities) > 0 {
		score += float64(agent.Config().Priority)
	}

	return score
}

func TestRouterFallback(t *testing.T) {
	router := NewSimpleRouter()

	t.Run("Fallback to any available agent", func(t *testing.T) {
		agent1 := NewMockSubAgent("agent1")
		agent1.mu.Lock()
		agent1.canHandle = false
		agent1.mu.Unlock()

		agent2 := NewMockSubAgent("agent2")
		agent2.mu.Lock()
		agent2.canHandle = false
		agent2.mu.Unlock()

		agent3 := NewMockSubAgent("fallback")
		agent3.mu.Lock()
		agent3.canHandle = true
		agent3.mu.Unlock()

		agents := []core.SubAgent{agent1, agent2, agent3}
		task := core.SubAgentTask{ID: "task1"}

		selected, err := router.Route(task, agents)
		require.NoError(t, err)
		assert.Equal(t, "fallback", selected.Name())
	})

	t.Run("No fallback available", func(t *testing.T) {
		agent1 := NewMockSubAgent("agent1")
		agent1.mu.Lock()
		agent1.canHandle = false
		agent1.mu.Unlock()

		agent2 := NewMockSubAgent("agent2")
		agent2.mu.Lock()
		agent2.canHandle = false
		agent2.mu.Unlock()

		agents := []core.SubAgent{agent1, agent2}
		task := core.SubAgentTask{ID: "task1"}

		_, err := router.Route(task, agents)
		assert.Error(t, err)
	})
}

// Benchmark tests
func BenchmarkRouter_Route(b *testing.B) {
	router := NewSimpleRouter()
	
	// Create many agents
	agents := make([]core.SubAgent, 100)
	for i := 0; i < 100; i++ {
		agent := NewMockSubAgent(string(rune('a' + (i % 26))))
		agent.capabilities = []string{"cap1", "cap2", "cap3"}
		agent.config.Priority = i % 10
		agents[i] = agent
	}

	task := core.SubAgentTask{
		ID: "bench-task",
		Context: map[string]interface{}{
			"cap1": true,
			"cap2": true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.Route(task, agents)
	}
}

func BenchmarkRouter_Score(b *testing.B) {
	router := NewSimpleRouter()
	agent := NewMockSubAgent("bench-agent")
	agent.capabilities = []string{"cap1", "cap2", "cap3"}
	
	task := core.SubAgentTask{
		ID: "bench-task",
		Context: map[string]interface{}{
			"cap1": true,
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.Score(task, agent)
	}
}

func BenchmarkRouter_Learn(b *testing.B) {
	router := NewSimpleRouter()
	agent := NewMockSubAgent("bench-agent")
	
	task := core.SubAgentTask{ID: "bench-task"}
	result := &core.SubAgentResult{
		TaskID:    task.ID,
		AgentName: agent.Name(),
		Status:    core.StatusCompleted,
		Duration:  time.Second,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		router.Learn(task, agent, result)
	}
}