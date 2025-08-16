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
	"fmt"
	"sync"
	"time"

	"github.com/rizome-dev/opun/pkg/core"
)

// Manager implements the SubAgentManager interface
type Manager struct {
	mu         sync.RWMutex
	agents     map[string]core.SubAgent
	tasks      map[string]*taskExecution
	router     core.TaskRouter
	providers  map[core.ProviderType]core.Provider
}

// taskExecution tracks an executing task
type taskExecution struct {
	task      core.SubAgentTask
	agent     core.SubAgent
	result    *core.SubAgentResult
	status    core.ExecutionStatus
	startTime time.Time
	cancel    context.CancelFunc
}

// NewManager creates a new subagent manager
func NewManager() *Manager {
	return &Manager{
		agents:    make(map[string]core.SubAgent),
		tasks:     make(map[string]*taskExecution),
		providers: make(map[core.ProviderType]core.Provider),
		router:    NewSimpleRouter(),
	}
}

// SetRouter sets a custom task router
func (m *Manager) SetRouter(router core.TaskRouter) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.router = router
}

// RegisterProvider registers a provider for cross-provider coordination
func (m *Manager) RegisterProvider(provider core.Provider) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.providers[provider.Type()] = provider
	
	// If provider supports subagents, register its agents
	if capable, ok := provider.(core.SubAgentCapable); ok && capable.SupportsSubAgents() {
		configs, err := capable.ListSubAgents()
		if err != nil {
			return fmt.Errorf("failed to list subagents for provider %s: %w", provider.Name(), err)
		}
		
		for _, config := range configs {
			agent, err := capable.CreateSubAgent(config)
			if err != nil {
				// Log error but continue with other agents
				continue
			}
			m.agents[agent.Name()] = agent
		}
	}
	
	return nil
}

// Register registers a new subagent
func (m *Manager) Register(agent core.SubAgent) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.agents[agent.Name()]; exists {
		return fmt.Errorf("agent %s already registered", agent.Name())
	}
	
	if err := agent.Validate(); err != nil {
		return fmt.Errorf("agent validation failed: %w", err)
	}
	
	m.agents[agent.Name()] = agent
	return nil
}

// Unregister removes a subagent
func (m *Manager) Unregister(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	agent, exists := m.agents[name]
	if !exists {
		return fmt.Errorf("agent %s not found", name)
	}
	
	// Cleanup the agent
	if err := agent.Cleanup(); err != nil {
		return fmt.Errorf("failed to cleanup agent: %w", err)
	}
	
	delete(m.agents, name)
	return nil
}

// List returns all registered subagents
func (m *Manager) List() []core.SubAgent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	agents := make([]core.SubAgent, 0, len(m.agents))
	for _, agent := range m.agents {
		agents = append(agents, agent)
	}
	return agents
}

// Get retrieves a subagent by name
func (m *Manager) Get(name string) (core.SubAgent, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	agent, exists := m.agents[name]
	if !exists {
		return nil, fmt.Errorf("agent %s not found", name)
	}
	return agent, nil
}

// Find finds subagents with specific capabilities
func (m *Manager) Find(capabilities []string) []core.SubAgent {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var matched []core.SubAgent
	for _, agent := range m.agents {
		agentCaps := agent.GetCapabilities()
		if hasAllCapabilities(agentCaps, capabilities) {
			matched = append(matched, agent)
		}
	}
	return matched
}

// Execute executes a task with a specific agent
func (m *Manager) Execute(ctx context.Context, task core.SubAgentTask, agentName string) (*core.SubAgentResult, error) {
	agent, err := m.Get(agentName)
	if err != nil {
		return nil, err
	}
	
	if !agent.CanHandle(task) {
		return nil, fmt.Errorf("agent %s cannot handle task %s", agentName, task.Name)
	}
	
	// Track execution
	ctx, cancel := context.WithCancel(ctx)
	execution := &taskExecution{
		task:      task,
		agent:     agent,
		status:    core.StatusRunning,
		startTime: time.Now(),
		cancel:    cancel,
	}
	
	m.mu.Lock()
	m.tasks[task.ID] = execution
	m.mu.Unlock()
	
	// Execute the task
	result, err := agent.Execute(ctx, task)
	
	// Update execution tracking
	m.mu.Lock()
	if execution, exists := m.tasks[task.ID]; exists {
		execution.result = result
		if err != nil {
			execution.status = core.StatusFailed
		} else {
			execution.status = result.Status
		}
	}
	m.mu.Unlock()
	
	// Learn from the result if router supports it
	if m.router != nil && result != nil {
		m.router.Learn(task, agent, result)
	}
	
	return result, err
}

// ExecuteParallel executes multiple tasks in parallel
func (m *Manager) ExecuteParallel(ctx context.Context, tasks []core.SubAgentTask) ([]*core.SubAgentResult, error) {
	results := make([]*core.SubAgentResult, len(tasks))
	errors := make([]error, len(tasks))
	var wg sync.WaitGroup
	
	for i, task := range tasks {
		wg.Add(1)
		go func(idx int, t core.SubAgentTask) {
			defer wg.Done()
			
			// Delegate to find best agent
			result, err := m.Delegate(ctx, t)
			results[idx] = result
			errors[idx] = err
		}(i, task)
	}
	
	wg.Wait()
	
	// Check for errors
	for _, err := range errors {
		if err != nil {
			return results, fmt.Errorf("one or more tasks failed")
		}
	}
	
	return results, nil
}

// Delegate automatically delegates a task to the best agent
func (m *Manager) Delegate(ctx context.Context, task core.SubAgentTask) (*core.SubAgentResult, error) {
	return m.DelegateWithStrategy(ctx, task, core.DelegationAutomatic)
}

// DelegateWithStrategy delegates a task using a specific strategy
func (m *Manager) DelegateWithStrategy(ctx context.Context, task core.SubAgentTask, strategy core.DelegationStrategy) (*core.SubAgentResult, error) {
	m.mu.RLock()
	agents := make([]core.SubAgent, 0, len(m.agents))
	for _, agent := range m.agents {
		agents = append(agents, agent)
	}
	m.mu.RUnlock()
	
	if len(agents) == 0 {
		return nil, fmt.Errorf("no agents available")
	}
	
	var selectedAgent core.SubAgent
	var err error
	
	switch strategy {
	case core.DelegationAutomatic:
		// Use router to find best agent
		if m.router != nil {
			selectedAgent, err = m.router.Route(task, agents)
			if err != nil {
				return nil, fmt.Errorf("routing failed: %w", err)
			}
		} else {
			// Fallback to first capable agent
			for _, agent := range agents {
				if agent.CanHandle(task) {
					selectedAgent = agent
					break
				}
			}
		}
		
	case core.DelegationExplicit:
		// This strategy should be handled by Execute with specific agent name
		return nil, fmt.Errorf("explicit delegation requires agent name")
		
	case core.DelegationProactive:
		// Find agents that match task context
		for _, agent := range agents {
			config := agent.Config()
			if config.Strategy == core.DelegationProactive && agent.CanHandle(task) {
				selectedAgent = agent
				break
			}
		}
	}
	
	if selectedAgent == nil {
		return nil, fmt.Errorf("no suitable agent found for task %s", task.Name)
	}
	
	return m.Execute(ctx, task, selectedAgent.Name())
}

// GetStatus gets the status of a task
func (m *Manager) GetStatus(taskID string) (core.ExecutionStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	execution, exists := m.tasks[taskID]
	if !exists {
		return "", fmt.Errorf("task %s not found", taskID)
	}
	
	// Check agent status if still running
	if execution.status == core.StatusRunning {
		return execution.agent.Status(), nil
	}
	
	return execution.status, nil
}

// GetResults gets the results of a task
func (m *Manager) GetResults(taskID string) (*core.SubAgentResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	execution, exists := m.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("task %s not found", taskID)
	}
	
	return execution.result, nil
}

// ListActiveTasks lists all active task IDs
func (m *Manager) ListActiveTasks() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var active []string
	for id, execution := range m.tasks {
		if execution.status == core.StatusRunning || execution.status == core.StatusPending {
			active = append(active, id)
		}
	}
	return active
}

// CoordinateAcrossProviders coordinates tasks across different providers
func (m *Manager) CoordinateAcrossProviders(ctx context.Context, tasks []core.SubAgentTask) ([]*core.SubAgentResult, error) {
	// Group tasks by preferred provider
	providerTasks := make(map[core.ProviderType][]core.SubAgentTask)
	
	for _, task := range tasks {
		// Find best agent for task
		agents := m.List()
		if len(agents) == 0 {
			return nil, fmt.Errorf("no agents available")
		}
		
		agent, err := m.router.Route(task, agents)
		if err != nil {
			return nil, fmt.Errorf("failed to route task %s: %w", task.Name, err)
		}
		
		provider := agent.Provider()
		providerTasks[provider] = append(providerTasks[provider], task)
	}
	
	// Execute tasks grouped by provider
	var allResults []*core.SubAgentResult
	var mu sync.Mutex
	var wg sync.WaitGroup
	
	for provider, tasks := range providerTasks {
		wg.Add(1)
		go func(p core.ProviderType, taskList []core.SubAgentTask) {
			defer wg.Done()
			
			for _, task := range taskList {
				result, err := m.Delegate(ctx, task)
				if err != nil {
					// Create error result
					result = &core.SubAgentResult{
						TaskID:    task.ID,
						Status:    core.StatusFailed,
						Error:     err,
						StartTime: time.Now(),
						EndTime:   time.Now(),
					}
				}
				
				mu.Lock()
				allResults = append(allResults, result)
				mu.Unlock()
			}
		}(provider, tasks)
	}
	
	wg.Wait()
	return allResults, nil
}

// CancelTask cancels a running task
func (m *Manager) CancelTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	execution, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}
	
	if execution.cancel != nil {
		execution.cancel()
	}
	
	execution.status = core.StatusCancelled
	return execution.agent.Cancel()
}

// CleanupTask removes a completed task from tracking
func (m *Manager) CleanupTask(taskID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	execution, exists := m.tasks[taskID]
	if !exists {
		return fmt.Errorf("task %s not found", taskID)
	}
	
	if execution.status == core.StatusRunning || execution.status == core.StatusPending {
		return fmt.Errorf("cannot cleanup running task")
	}
	
	delete(m.tasks, taskID)
	return nil
}

// hasAllCapabilities checks if agent has all required capabilities
func hasAllCapabilities(agentCaps, required []string) bool {
	capMap := make(map[string]bool)
	for _, cap := range agentCaps {
		capMap[cap] = true
	}
	
	for _, req := range required {
		if !capMap[req] {
			return false
		}
	}
	return true
}