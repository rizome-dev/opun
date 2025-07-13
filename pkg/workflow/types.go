package workflow

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
	"time"
)

// Workflow represents a complete workflow definition
type Workflow struct {
	Name        string                 `yaml:"name" json:"name"`
	Command     string                 `yaml:"command" json:"command"` // Slash command to trigger
	Description string                 `yaml:"description" json:"description"`
	Version     string                 `yaml:"version" json:"version"`
	Author      string                 `yaml:"author" json:"author"`
	Variables   []Variable             `yaml:"variables" json:"variables"`
	Agents      []Agent                `yaml:"agents" json:"agents"`
	Settings    Settings               `yaml:"settings" json:"settings"`
	Metadata    map[string]interface{} `yaml:"metadata" json:"metadata"`
}

// Variable defines a workflow-level variable
type Variable struct {
	Name         string      `yaml:"name" json:"name"`
	Description  string      `yaml:"description" json:"description"`
	Type         string      `yaml:"type" json:"type"` // string, number, boolean, file
	Required     bool        `yaml:"required" json:"required"`
	DefaultValue interface{} `yaml:"default" json:"default"`
	Internal     bool        `yaml:"internal" json:"internal"` // If true, don't prompt user for this variable
}

// Agent represents a single agent in the workflow
type Agent struct {
	ID        string                 `yaml:"id" json:"id"`
	Name      string                 `yaml:"name" json:"name"`
	Provider  string                 `yaml:"provider" json:"provider"`
	Model     string                 `yaml:"model" json:"model"`
	Prompt    string                 `yaml:"prompt" json:"prompt"`
	Input     map[string]interface{} `yaml:"input" json:"input"`
	Output    string                 `yaml:"output" json:"output"`
	DependsOn []string               `yaml:"depends_on" json:"depends_on"`
	Condition string                 `yaml:"condition" json:"condition"`
	Settings  AgentSettings          `yaml:"settings" json:"settings"`
	OnSuccess []Action               `yaml:"on_success" json:"on_success"`
	OnFailure []Action               `yaml:"on_failure" json:"on_failure"`
}

// AgentSettings contains agent-specific settings
type AgentSettings struct {
	Temperature     float64  `yaml:"temperature" json:"temperature"`
	MaxTokens       int      `yaml:"max_tokens" json:"max_tokens"`
	Timeout         int      `yaml:"timeout" json:"timeout"` // seconds
	RetryCount      int      `yaml:"retry_count" json:"retry_count"`
	QualityMode     string   `yaml:"quality_mode" json:"quality_mode"`
	Tools           []string `yaml:"tools" json:"tools"`
	MCPServers      []string `yaml:"mcp_servers" json:"mcp_servers"`
	WaitForFile     string   `yaml:"wait_for_file" json:"wait_for_file"`
	Interactive     bool     `yaml:"interactive" json:"interactive"`
	ContinueOnError bool     `yaml:"continue_on_error" json:"continue_on_error"`
}

// Settings contains workflow-level settings
type Settings struct {
	Parallel      bool   `yaml:"parallel" json:"parallel"`
	MaxConcurrent int    `yaml:"max_concurrent" json:"max_concurrent"`
	StopOnError   bool   `yaml:"stop_on_error" json:"stop_on_error"`
	OutputDir     string `yaml:"output_dir" json:"output_dir"`
	LogLevel      string `yaml:"log_level" json:"log_level"`
}

// Action represents an action to take on success/failure
type Action struct {
	Type    string                 `yaml:"type" json:"type"` // notify, log, execute, abort
	Message string                 `yaml:"message" json:"message"`
	Data    map[string]interface{} `yaml:"data" json:"data"`
}

// ExecutionState represents the current state of a workflow execution
type ExecutionState struct {
	WorkflowID   string                 `json:"workflow_id"`
	SessionID    string                 `json:"session_id"`
	StartTime    time.Time              `json:"start_time"`
	EndTime      *time.Time             `json:"end_time"`
	Status       ExecutionStatus        `json:"status"`
	CurrentAgent string                 `json:"current_agent"`
	Variables    map[string]interface{} `json:"variables"`
	AgentStates  map[string]*AgentState `json:"agent_states"`
	Outputs      map[string]string      `json:"outputs"`
	Errors       []ExecutionError       `json:"errors"`
}

// AgentState represents the state of a single agent execution
type AgentState struct {
	AgentID   string          `json:"agent_id"`
	Status    ExecutionStatus `json:"status"`
	StartTime *time.Time      `json:"start_time"`
	EndTime   *time.Time      `json:"end_time"`
	Attempts  int             `json:"attempts"`
	Output    string          `json:"output"`
	Error     *ExecutionError `json:"error"`
}

// ExecutionStatus represents the status of execution
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusSkipped   ExecutionStatus = "skipped"
	StatusAborted   ExecutionStatus = "aborted"
)

// ExecutionError represents an error during execution
type ExecutionError struct {
	AgentID   string    `json:"agent_id"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Fatal     bool      `json:"fatal"`
}

// WorkflowEvent represents an event during workflow execution
type WorkflowEvent struct {
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	AgentID   string                 `json:"agent_id,omitempty"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// EventType represents the type of workflow event
type EventType string

const (
	EventWorkflowStart    EventType = "workflow_start"
	EventWorkflowComplete EventType = "workflow_complete"
	EventWorkflowError    EventType = "workflow_error"
	EventAgentStart       EventType = "agent_start"
	EventAgentComplete    EventType = "agent_complete"
	EventAgentError       EventType = "agent_error"
	EventAgentRetry       EventType = "agent_retry"
	EventVariableSet      EventType = "variable_set"
	EventOutputCreated    EventType = "output_created"
)
