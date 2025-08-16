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
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/rizome-dev/opun/pkg/core"
)

// SimpleRouter implements a basic task routing strategy
type SimpleRouter struct {
	mu        sync.RWMutex
	stats     map[string]*agentStats
	weights   map[string]float64
}

// agentStats tracks agent performance
type agentStats struct {
	totalTasks   int
	successTasks int
	failedTasks  int
	totalTime    float64
	avgTime      float64
}

// NewSimpleRouter creates a new simple router
func NewSimpleRouter() *SimpleRouter {
	return &SimpleRouter{
		stats:   make(map[string]*agentStats),
		weights: make(map[string]float64),
	}
}

// Route finds the best agent for a task
func (r *SimpleRouter) Route(task core.SubAgentTask, agents []core.SubAgent) (core.SubAgent, error) {
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents available")
	}
	
	// Score all agents
	scores := make(map[core.SubAgent]float64)
	for _, agent := range agents {
		if agent.CanHandle(task) {
			scores[agent] = r.Score(task, agent)
		}
	}
	
	if len(scores) == 0 {
		return nil, fmt.Errorf("no capable agents found for task %s", task.Name)
	}
	
	// Sort agents by score
	type agentScore struct {
		agent core.SubAgent
		score float64
	}
	
	var sortedAgents []agentScore
	for agent, score := range scores {
		sortedAgents = append(sortedAgents, agentScore{agent, score})
	}
	
	sort.Slice(sortedAgents, func(i, j int) bool {
		return sortedAgents[i].score > sortedAgents[j].score
	})
	
	return sortedAgents[0].agent, nil
}

// Score calculates a score for an agent handling a task
func (r *SimpleRouter) Score(task core.SubAgentTask, agent core.SubAgent) float64 {
	// If agent can't handle the task, return 0
	if !agent.CanHandle(task) {
		return 0.0
	}
	
	score := 0.0
	config := agent.Config()
	
	// Base score from priority
	score += float64(config.Priority) * 10
	
	// Context matching score
	contextScore := r.calculateContextScore(task, config.Context)
	score += contextScore * 20
	
	// Capability matching score
	capScore := r.calculateCapabilityScore(task, agent.GetCapabilities())
	score += capScore * 30
	
	// Performance history score
	r.mu.RLock()
	if stats, exists := r.stats[agent.Name()]; exists {
		if stats.totalTasks > 0 {
			successRate := float64(stats.successTasks) / float64(stats.totalTasks)
			score += successRate * 25
			
			// Penalize slow agents
			if stats.avgTime > 0 {
				speedScore := 100.0 / stats.avgTime
				if speedScore > 15 {
					speedScore = 15
				}
				score += speedScore
			}
		}
	}
	r.mu.RUnlock()
	
	// Provider preference (if specified in task context)
	if preferredProvider, ok := task.Context["preferred_provider"].(string); ok {
		if string(agent.Provider()) == preferredProvider {
			score += 10
		}
	}
	
	// Parallel execution bonus
	if agent.SupportsParallel() && task.Priority > 5 {
		score += 5
	}
	
	return score
}

// Learn updates router knowledge based on execution results
func (r *SimpleRouter) Learn(task core.SubAgentTask, agent core.SubAgent, result *core.SubAgentResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Initialize stats if needed
	if _, exists := r.stats[agent.Name()]; !exists {
		r.stats[agent.Name()] = &agentStats{}
	}
	
	stats := r.stats[agent.Name()]
	stats.totalTasks++
	
	if result.Status == core.StatusCompleted {
		stats.successTasks++
	} else if result.Status == core.StatusFailed {
		stats.failedTasks++
	}
	
	// Update average time
	duration := result.Duration.Seconds()
	stats.totalTime += duration
	stats.avgTime = stats.totalTime / float64(stats.totalTasks)
	
	// Update weights based on performance
	successRate := float64(stats.successTasks) / float64(stats.totalTasks)
	r.weights[agent.Name()] = successRate
}

// GetStats returns routing statistics
func (r *SimpleRouter) GetStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	stats := make(map[string]interface{})
	
	// Copy agent stats
	agentStats := make(map[string]interface{})
	for name, stat := range r.stats {
		agentStats[name] = map[string]interface{}{
			"total_tasks":   stat.totalTasks,
			"success_tasks": stat.successTasks,
			"failed_tasks":  stat.failedTasks,
			"avg_time":      stat.avgTime,
			"success_rate":  float64(stat.successTasks) / float64(stat.totalTasks),
		}
	}
	stats["agents"] = agentStats
	
	// Copy weights
	weights := make(map[string]float64)
	for name, weight := range r.weights {
		weights[name] = weight
	}
	stats["weights"] = weights
	
	return stats
}

// calculateContextScore calculates how well task context matches agent context patterns
func (r *SimpleRouter) calculateContextScore(task core.SubAgentTask, agentContextPatterns []string) float64 {
	if len(agentContextPatterns) == 0 {
		return 0.5 // Neutral score if no patterns defined
	}
	
	// Combine task description and context into searchable text
	taskText := strings.ToLower(task.Description + " " + task.Name)
	for _, value := range task.Context {
		if str, ok := value.(string); ok {
			taskText += " " + strings.ToLower(str)
		}
	}
	
	matches := 0
	for _, pattern := range agentContextPatterns {
		pattern = strings.ToLower(pattern)
		if strings.Contains(taskText, pattern) {
			matches++
		}
	}
	
	return float64(matches) / float64(len(agentContextPatterns))
}

// calculateCapabilityScore calculates how well agent capabilities match task requirements
func (r *SimpleRouter) calculateCapabilityScore(task core.SubAgentTask, agentCapabilities []string) float64 {
	if len(agentCapabilities) == 0 {
		return 0.5 // Neutral score if no capabilities defined
	}
	
	// Extract required capabilities from task
	requiredCaps := make(map[string]bool)
	
	// Check task constraints for capability hints
	for _, constraint := range task.Constraints {
		constraint = strings.ToLower(constraint)
		for _, cap := range agentCapabilities {
			if strings.Contains(constraint, strings.ToLower(cap)) {
				requiredCaps[cap] = true
			}
		}
	}
	
	// Check task description for capability keywords
	taskText := strings.ToLower(task.Description + " " + task.Name)
	for _, cap := range agentCapabilities {
		if strings.Contains(taskText, strings.ToLower(cap)) {
			requiredCaps[cap] = true
		}
	}
	
	if len(requiredCaps) == 0 {
		return 0.5 // No specific requirements found
	}
	
	// Calculate match percentage
	matches := 0
	for cap := range requiredCaps {
		for _, agentCap := range agentCapabilities {
			if strings.EqualFold(cap, agentCap) {
				matches++
				break
			}
		}
	}
	
	return float64(matches) / float64(len(requiredCaps))
}

// Reset clears all learned statistics
func (r *SimpleRouter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.stats = make(map[string]*agentStats)
	r.weights = make(map[string]float64)
}