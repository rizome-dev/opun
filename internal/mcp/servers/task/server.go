package task

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
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/rizome-dev/opun/pkg/core"
	"github.com/rizome-dev/opun/pkg/subagent"
)

// TaskServer implements an MCP server that provides Task tool functionality
type TaskServer struct {
	name        string
	description string
	manager     core.SubAgentManager
	mu          sync.RWMutex
	tasks       map[string]*TaskExecution
}

// TaskExecution tracks a task execution
type TaskExecution struct {
	ID        string                 `json:"id"`
	Task      core.SubAgentTask      `json:"task"`
	Agent     string                 `json:"agent,omitempty"`
	Status    core.ExecutionStatus   `json:"status"`
	Result    *core.SubAgentResult   `json:"result,omitempty"`
	StartTime time.Time              `json:"start_time"`
	EndTime   *time.Time             `json:"end_time,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// TaskRequest represents a task execution request
type TaskRequest struct {
	Task        string                 `json:"task"`
	Agent       string                 `json:"agent,omitempty"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Parallel    bool                   `json:"parallel,omitempty"`
	Timeout     int                    `json:"timeout,omitempty"` // seconds
	RetryCount  int                    `json:"retry_count,omitempty"`
}

// NewTaskServer creates a new Task MCP server
func NewTaskServer(manager core.SubAgentManager) *TaskServer {
	if manager == nil {
		// Create a default manager if none provided
		manager = subagent.NewManager()
	}
	
	return &TaskServer{
		name:        "task",
		description: "Cross-provider task delegation and subagent orchestration",
		manager:     manager,
		tasks:       make(map[string]*TaskExecution),
	}
}

// Name returns the server name
func (s *TaskServer) Name() string {
	return s.name
}

// Description returns the server description
func (s *TaskServer) Description() string {
	return s.description
}

// GetTools returns the tools provided by this server
func (s *TaskServer) GetTools() []core.Tool {
	return []core.Tool{
		{
			Name:        "task",
			Description: "Delegate a task to a subagent for execution",
			Category:    "orchestration",
			Parameters: []core.ToolParameter{
				{
					Name:        "task",
					Type:        "string",
					Description: "The task description or prompt",
					Required:    true,
				},
				{
					Name:        "agent",
					Type:        "string",
					Description: "Optional: specific agent to use",
					Required:    false,
				},
				{
					Name:        "context",
					Type:        "object",
					Description: "Optional: task context and variables",
					Required:    false,
				},
				{
					Name:        "parallel",
					Type:        "boolean",
					Description: "Execute in parallel (non-blocking)",
					Required:    false,
				},
			},
		},
		{
			Name:        "task_batch",
			Description: "Execute multiple tasks in parallel",
			Category:    "orchestration",
			Parameters: []core.ToolParameter{
				{
					Name:        "tasks",
					Type:        "array",
					Description: "Array of task requests",
					Required:    true,
				},
			},
		},
		{
			Name:        "task_status",
			Description: "Get the status of a running task",
			Category:    "orchestration",
			Parameters: []core.ToolParameter{
				{
					Name:        "task_id",
					Type:        "string",
					Description: "The task ID to check",
					Required:    true,
				},
			},
		},
		{
			Name:        "list_agents",
			Description: "List available subagents",
			Category:    "orchestration",
		},
	}
}

// ExecuteTool executes a tool request
func (s *TaskServer) ExecuteTool(ctx context.Context, tool string, params map[string]interface{}) (interface{}, error) {
	switch tool {
	case "task":
		return s.executeTask(ctx, params)
		
	case "task_batch":
		return s.executeBatch(ctx, params)
		
	case "task_status":
		return s.getTaskStatus(params)
		
	case "list_agents":
		return s.listAgents()
		
	default:
		return nil, fmt.Errorf("unknown tool: %s", tool)
	}
}

// executeTask executes a single task
func (s *TaskServer) executeTask(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	// Parse request
	taskDesc, ok := params["task"].(string)
	if !ok || taskDesc == "" {
		return nil, fmt.Errorf("task description is required")
	}
	
	// Create task
	task := core.SubAgentTask{
		ID:          uuid.New().String(),
		Name:        "task_" + time.Now().Format("20060102_150405"),
		Description: taskDesc,
		Input:       taskDesc,
		Context:     make(map[string]interface{}),
		Variables:   make(map[string]interface{}),
	}
	
	// Add context if provided
	if context, ok := params["context"].(map[string]interface{}); ok {
		task.Context = context
	}
	
	// Check for specific agent
	agentName, _ := params["agent"].(string)
	
	// Check for parallel execution
	parallel, _ := params["parallel"].(bool)
	
	// Create execution record
	execution := &TaskExecution{
		ID:        task.ID,
		Task:      task,
		Agent:     agentName,
		Status:    core.StatusPending,
		StartTime: time.Now(),
	}
	
	s.mu.Lock()
	s.tasks[task.ID] = execution
	s.mu.Unlock()
	
	// Execute task
	if parallel {
		// Async execution
		go s.executeAsync(ctx, execution, agentName)
		return map[string]interface{}{
			"task_id": task.ID,
			"status":  "running",
			"message": "Task started in background",
		}, nil
	}
	
	// Sync execution
	var result *core.SubAgentResult
	var err error
	
	if agentName != "" {
		result, err = s.manager.Execute(ctx, task, agentName)
	} else {
		result, err = s.manager.Delegate(ctx, task)
	}
	
	// Update execution record
	s.updateExecution(execution, result, err)
	
	if err != nil {
		return map[string]interface{}{
			"task_id": task.ID,
			"status":  "failed",
			"error":   err.Error(),
		}, nil
	}
	
	return map[string]interface{}{
		"task_id": task.ID,
		"status":  string(result.Status),
		"output":  result.Output,
		"agent":   result.AgentName,
		"duration": result.Duration.Seconds(),
	}, nil
}

// executeBatch executes multiple tasks in parallel
func (s *TaskServer) executeBatch(ctx context.Context, params map[string]interface{}) (interface{}, error) {
	tasksRaw, ok := params["tasks"].([]interface{})
	if !ok || len(tasksRaw) == 0 {
		return nil, fmt.Errorf("tasks array is required")
	}
	
	var tasks []core.SubAgentTask
	for i, taskRaw := range tasksRaw {
		taskMap, ok := taskRaw.(map[string]interface{})
		if !ok {
			continue
		}
		
		taskDesc, ok := taskMap["task"].(string)
		if !ok || taskDesc == "" {
			continue
		}
		
		task := core.SubAgentTask{
			ID:          fmt.Sprintf("%s_%d", uuid.New().String(), i),
			Name:        fmt.Sprintf("batch_task_%d", i),
			Description: taskDesc,
			Input:       taskDesc,
			Context:     make(map[string]interface{}),
			Variables:   make(map[string]interface{}),
		}
		
		if context, ok := taskMap["context"].(map[string]interface{}); ok {
			task.Context = context
		}
		
		tasks = append(tasks, task)
	}
	
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no valid tasks provided")
	}
	
	// Execute tasks in parallel
	results, err := s.manager.ExecuteParallel(ctx, tasks)
	if err != nil {
		return nil, fmt.Errorf("batch execution failed: %w", err)
	}
	
	// Format results
	var output []map[string]interface{}
	for i, result := range results {
		if result == nil {
			continue
		}
		
		output = append(output, map[string]interface{}{
			"task_id":  tasks[i].ID,
			"status":   string(result.Status),
			"output":   result.Output,
			"agent":    result.AgentName,
			"duration": result.Duration.Seconds(),
		})
	}
	
	return map[string]interface{}{
		"total":   len(tasks),
		"results": output,
	}, nil
}

// getTaskStatus gets the status of a task
func (s *TaskServer) getTaskStatus(params map[string]interface{}) (interface{}, error) {
	taskID, ok := params["task_id"].(string)
	if !ok || taskID == "" {
		return nil, fmt.Errorf("task_id is required")
	}
	
	s.mu.RLock()
	execution, exists := s.tasks[taskID]
	s.mu.RUnlock()
	
	if !exists {
		// Try to get from manager
		status, err := s.manager.GetStatus(taskID)
		if err != nil {
			return nil, fmt.Errorf("task not found: %s", taskID)
		}
		
		return map[string]interface{}{
			"task_id": taskID,
			"status":  string(status),
		}, nil
	}
	
	response := map[string]interface{}{
		"task_id":    execution.ID,
		"status":     string(execution.Status),
		"start_time": execution.StartTime.Format(time.RFC3339),
	}
	
	if execution.Agent != "" {
		response["agent"] = execution.Agent
	}
	
	if execution.EndTime != nil {
		response["end_time"] = execution.EndTime.Format(time.RFC3339)
		response["duration"] = execution.EndTime.Sub(execution.StartTime).Seconds()
	}
	
	if execution.Result != nil {
		response["output"] = execution.Result.Output
	}
	
	if execution.Error != "" {
		response["error"] = execution.Error
	}
	
	return response, nil
}

// listAgents lists available subagents
func (s *TaskServer) listAgents() (interface{}, error) {
	agents := s.manager.List()
	
	var agentList []map[string]interface{}
	for _, agent := range agents {
		config := agent.Config()
		agentList = append(agentList, map[string]interface{}{
			"name":         config.Name,
			"description":  config.Description,
			"provider":     string(config.Provider),
			"capabilities": config.Capabilities,
			"strategy":     string(config.Strategy),
			"interactive":  config.Interactive,
		})
	}
	
	return map[string]interface{}{
		"total":  len(agentList),
		"agents": agentList,
	}, nil
}

// executeAsync executes a task asynchronously
func (s *TaskServer) executeAsync(ctx context.Context, execution *TaskExecution, agentName string) {
	execution.Status = core.StatusRunning
	
	var result *core.SubAgentResult
	var err error
	
	if agentName != "" {
		result, err = s.manager.Execute(ctx, execution.Task, agentName)
	} else {
		result, err = s.manager.Delegate(ctx, execution.Task)
	}
	
	s.updateExecution(execution, result, err)
}

// updateExecution updates an execution record
func (s *TaskServer) updateExecution(execution *TaskExecution, result *core.SubAgentResult, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	now := time.Now()
	execution.EndTime = &now
	
	if err != nil {
		execution.Status = core.StatusFailed
		execution.Error = err.Error()
	} else if result != nil {
		execution.Status = result.Status
		execution.Result = result
		if result.AgentName != "" {
			execution.Agent = result.AgentName
		}
	}
}

// Cleanup cleans up old task records
func (s *TaskServer) Cleanup(maxAge time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	cutoff := time.Now().Add(-maxAge)
	
	for id, execution := range s.tasks {
		if execution.EndTime != nil && execution.EndTime.Before(cutoff) {
			delete(s.tasks, id)
		}
	}
}

// GetConfiguration returns the server configuration
func (s *TaskServer) GetConfiguration() map[string]interface{} {
	return map[string]interface{}{
		"name":        s.name,
		"description": s.description,
		"version":     "1.0.0",
		"tools":       len(s.GetTools()),
	}
}

// Start starts the MCP server (if needed for standalone operation)
func (s *TaskServer) Start(ctx context.Context) error {
	// Periodic cleanup of old tasks
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.Cleanup(24 * time.Hour)
			}
		}
	}()
	
	return nil
}

// ToJSON returns the server definition as JSON
func (s *TaskServer) ToJSON() ([]byte, error) {
	config := s.GetConfiguration()
	config["tools"] = s.GetTools()
	
	return json.MarshalIndent(config, "", "  ")
}