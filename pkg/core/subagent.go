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
	"time"
)

// SubAgentType represents different delegation approaches
type SubAgentType string

const (
	// SubAgentTypeDeclarative uses provider's native declarative system (e.g., Claude Task tool)
	SubAgentTypeDeclarative SubAgentType = "declarative"
	// SubAgentTypeProgrammatic uses programmatic delegation (e.g., Gemini SubAgentScope)
	SubAgentTypeProgrammatic SubAgentType = "programmatic"
	// SubAgentTypeWorkflow uses Opun's workflow system for delegation
	SubAgentTypeWorkflow SubAgentType = "workflow"
	// SubAgentTypeMCP uses MCP Task tool for cross-provider delegation
	SubAgentTypeMCP SubAgentType = "mcp"
)

// DelegationStrategy defines how tasks are delegated to subagents
type DelegationStrategy string

const (
	// DelegationAutomatic delegates based on context matching
	DelegationAutomatic DelegationStrategy = "automatic"
	// DelegationExplicit requires explicit delegation calls
	DelegationExplicit DelegationStrategy = "explicit"
	// DelegationProactive proactively suggests delegation
	DelegationProactive DelegationStrategy = "proactive"
)

// SubAgentConfig defines configuration for a subagent
type SubAgentConfig struct {
	// Basic information
	Name        string                 `json:"name" yaml:"name"`
	Type        SubAgentType           `json:"type" yaml:"type"`
	Description string                 `json:"description" yaml:"description"`
	Provider    ProviderType           `json:"provider" yaml:"provider"`
	Model       string                 `json:"model" yaml:"model"`

	// Delegation settings
	Strategy    DelegationStrategy     `json:"strategy" yaml:"strategy"`
	Context     []string               `json:"context" yaml:"context"`       // Context patterns for automatic delegation
	Capabilities []string              `json:"capabilities" yaml:"capabilities"` // What the agent can do
	Priority    int                    `json:"priority" yaml:"priority"`     // Priority for conflict resolution

	// Execution settings
	MaxRetries  int                    `json:"max_retries" yaml:"max_retries"`
	Timeout     time.Duration          `json:"timeout" yaml:"timeout"`
	Parallel    bool                   `json:"parallel" yaml:"parallel"`
	Interactive bool                   `json:"interactive" yaml:"interactive"`

	// Provider-specific settings
	ProviderConfig map[string]interface{} `json:"provider_config" yaml:"provider_config"`
	Settings       map[string]interface{} `json:"settings" yaml:"settings"`
	SystemPrompt   string                 `json:"system_prompt" yaml:"system_prompt"`

	// Tool and MCP configuration
	Tools       []string               `json:"tools" yaml:"tools"`
	MCPServers  []string               `json:"mcp_servers" yaml:"mcp_servers"`

	// Output handling
	OutputFormat string                 `json:"output_format" yaml:"output_format"` // json, text, markdown
	OutputPath   string                 `json:"output_path" yaml:"output_path"`     // Where to save output

	// Metadata
	Metadata    map[string]interface{} `json:"metadata" yaml:"metadata"`
}

// SubAgentTask represents a task to be executed by a subagent
type SubAgentTask struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Input       string                 `json:"input"`
	Context     map[string]interface{} `json:"context"`
	Variables   map[string]interface{} `json:"variables"`
	Constraints []string               `json:"constraints"`
	Priority    int                    `json:"priority"`
	Deadline    *time.Time             `json:"deadline,omitempty"`
}

// SubAgentResult represents the result of a subagent execution
type SubAgentResult struct {
	TaskID      string                 `json:"task_id"`
	AgentName   string                 `json:"agent_name"`
	Status      ExecutionStatus        `json:"status"`
	Output      string                 `json:"output"`
	Error       error                  `json:"error,omitempty"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time"`
	Duration    time.Duration          `json:"duration"`
	Metadata    map[string]interface{} `json:"metadata"`
	Artifacts   []SubAgentArtifact     `json:"artifacts,omitempty"`
}

// SubAgentArtifact represents an artifact produced by a subagent
type SubAgentArtifact struct {
	Name        string    `json:"name"`
	Type        string    `json:"type"` // file, url, data
	Path        string    `json:"path,omitempty"`
	Content     []byte    `json:"content,omitempty"`
	ContentType string    `json:"content_type"`
	Size        int64     `json:"size"`
	Created     time.Time `json:"created"`
}

// ExecutionStatus represents the status of subagent execution
type ExecutionStatus string

const (
	StatusPending    ExecutionStatus = "pending"
	StatusRunning    ExecutionStatus = "running"
	StatusCompleted  ExecutionStatus = "completed"
	StatusFailed     ExecutionStatus = "failed"
	StatusCancelled  ExecutionStatus = "cancelled"
	StatusTimeout    ExecutionStatus = "timeout"
)

// SubAgent defines the interface for a subagent
type SubAgent interface {
	// Information
	Name() string
	Config() SubAgentConfig
	Provider() ProviderType

	// Lifecycle
	Initialize(config SubAgentConfig) error
	Validate() error
	Cleanup() error

	// Task execution
	Execute(ctx context.Context, task SubAgentTask) (*SubAgentResult, error)
	ExecuteAsync(ctx context.Context, task SubAgentTask) (<-chan *SubAgentResult, error)

	// Status and control
	Status() ExecutionStatus
	Cancel() error
	GetProgress() (float64, string) // Returns progress percentage and message

	// Capabilities
	CanHandle(task SubAgentTask) bool
	GetCapabilities() []string
	SupportsParallel() bool
	SupportsInteractive() bool
}

// SubAgentCapable indicates a provider supports subagents
type SubAgentCapable interface {
	// Check if provider supports subagents
	SupportsSubAgents() bool

	// Get subagent implementation type
	GetSubAgentType() SubAgentType

	// Create a subagent with given configuration
	CreateSubAgent(config SubAgentConfig) (SubAgent, error)

	// List available subagents
	ListSubAgents() ([]SubAgentConfig, error)

	// Get subagent by name
	GetSubAgent(name string) (SubAgent, error)

	// Delegate a task to appropriate subagent
	Delegate(ctx context.Context, task SubAgentTask) (*SubAgentResult, error)

	// Register a new subagent
	RegisterSubAgent(agent SubAgent) error

	// Unregister a subagent
	UnregisterSubAgent(name string) error
}

// SubAgentManager manages subagents across providers
type SubAgentManager interface {
	// Registration
	Register(agent SubAgent) error
	Unregister(name string) error

	// Discovery
	List() []SubAgent
	Get(name string) (SubAgent, error)
	Find(capabilities []string) []SubAgent

	// Execution
	Execute(ctx context.Context, task SubAgentTask, agentName string) (*SubAgentResult, error)
	ExecuteParallel(ctx context.Context, tasks []SubAgentTask) ([]*SubAgentResult, error)

	// Delegation
	Delegate(ctx context.Context, task SubAgentTask) (*SubAgentResult, error)
	DelegateWithStrategy(ctx context.Context, task SubAgentTask, strategy DelegationStrategy) (*SubAgentResult, error)

	// Monitoring
	GetStatus(taskID string) (ExecutionStatus, error)
	GetResults(taskID string) (*SubAgentResult, error)
	ListActiveTasks() []string

	// Cross-provider coordination
	CoordinateAcrossProviders(ctx context.Context, tasks []SubAgentTask) ([]*SubAgentResult, error)
}

// SubAgentAdapter adapts provider-specific implementations to the SubAgent interface
type SubAgentAdapter interface {
	SubAgent

	// Provider-specific initialization
	InitializeProvider(provider Provider) error

	// Adapt provider-specific task format
	AdaptTask(task SubAgentTask) (interface{}, error)

	// Adapt provider-specific result format
	AdaptResult(result interface{}) (*SubAgentResult, error)

	// Get provider-specific configuration
	GetProviderConfig() map[string]interface{}
}

// TaskRouter routes tasks to appropriate subagents
type TaskRouter interface {
	// Route a task to the best subagent
	Route(task SubAgentTask, agents []SubAgent) (SubAgent, error)

	// Score agents for a given task
	Score(task SubAgentTask, agent SubAgent) float64

	// Learn from execution results
	Learn(task SubAgentTask, agent SubAgent, result *SubAgentResult)

	// Get routing statistics
	GetStats() map[string]interface{}
}