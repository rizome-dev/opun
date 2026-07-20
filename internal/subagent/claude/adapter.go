package claude

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
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rizome-dev/opun/internal/headless"
	"github.com/rizome-dev/opun/pkg/core"
	"gopkg.in/yaml.v3"
)

// ClaudeAdapter adapts Claude's file-based declarative system
type ClaudeAdapter struct {
	config   core.SubAgentConfig
	provider core.Provider
	status   core.ExecutionStatus
	agentDir string
	Runner   headless.Runner
}

// ClaudeAgentDefinition represents the structure of a Claude agent definition file
type ClaudeAgentDefinition struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	Capabilities []string `yaml:"capabilities"`
	Context      []string `yaml:"context"`
	Instructions string   `yaml:"instructions"`
	Tools        []string `yaml:"tools,omitempty"`
	MCPServers   []string `yaml:"mcp_servers,omitempty"`
}

// NewClaudeAdapter creates a new Claude subagent adapter
func NewClaudeAdapter(config core.SubAgentConfig) *ClaudeAdapter {
	return &ClaudeAdapter{
		config: config,
		status: core.StatusPending,
		Runner: &headless.ClaudeRunner{},
	}
}

// Name returns the agent name
func (a *ClaudeAdapter) Name() string {
	return a.config.Name
}

// Config returns the agent configuration
func (a *ClaudeAdapter) Config() core.SubAgentConfig {
	return a.config
}

// Provider returns the provider type
func (a *ClaudeAdapter) Provider() core.ProviderType {
	return core.ProviderTypeClaude
}

// Initialize initializes the adapter
func (a *ClaudeAdapter) Initialize(config core.SubAgentConfig) error {
	a.config = config

	// Set up Claude agent directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	// Claude uses .claude/agents/ for agent definitions
	a.agentDir = filepath.Join(homeDir, ".claude", "agents")

	// Ensure directory exists
	if err := os.MkdirAll(a.agentDir, 0755); err != nil {
		return fmt.Errorf("failed to create agent directory: %w", err)
	}

	// Create agent definition file
	return a.createAgentDefinition()
}

// Validate validates the adapter configuration
func (a *ClaudeAdapter) Validate() error {
	if a.config.Name == "" {
		return fmt.Errorf("agent name is required")
	}

	if a.config.Description == "" {
		return fmt.Errorf("agent description is required")
	}

	return nil
}

// Cleanup cleans up the adapter
func (a *ClaudeAdapter) Cleanup() error {
	// Remove agent definition file if it exists
	if a.agentDir != "" {
		agentFile := filepath.Join(a.agentDir, a.config.Name+".md")
		if err := os.Remove(agentFile); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to remove agent file: %w", err)
		}
	}

	return nil
}

// Execute executes a task via Claude's headless print mode
func (a *ClaudeAdapter) Execute(ctx context.Context, task core.SubAgentTask) (*core.SubAgentResult, error) {
	startTime := time.Now()
	a.status = core.StatusRunning

	result, err := a.Runner.Run(ctx, headless.Request{
		Prompt:       a.buildTaskPrompt(task),
		SystemPrompt: a.config.SystemPrompt,
		Model:        a.config.Model,
	})
	if err != nil {
		a.status = core.StatusFailed
		return &core.SubAgentResult{
			TaskID:    task.ID,
			AgentName: a.config.Name,
			Status:    core.StatusFailed,
			Error:     err,
			StartTime: startTime,
			EndTime:   time.Now(),
			Duration:  time.Since(startTime),
		}, err
	}

	a.status = core.StatusCompleted

	return &core.SubAgentResult{
		TaskID:    task.ID,
		AgentName: a.config.Name,
		Status:    core.StatusCompleted,
		Output:    result.Output,
		StartTime: startTime,
		EndTime:   time.Now(),
		Duration:  time.Since(startTime),
		Metadata: map[string]interface{}{
			"provider":   "claude",
			"method":     "headless",
			"session_id": result.SessionID,
		},
	}, nil
}

// ExecuteAsync executes a task asynchronously
func (a *ClaudeAdapter) ExecuteAsync(ctx context.Context, task core.SubAgentTask) (<-chan *core.SubAgentResult, error) {
	resultChan := make(chan *core.SubAgentResult, 1)

	go func() {
		result, _ := a.Execute(ctx, task)
		resultChan <- result
		close(resultChan)
	}()

	return resultChan, nil
}

// Status returns the current execution status
func (a *ClaudeAdapter) Status() core.ExecutionStatus {
	return a.status
}

// Cancel cancels the current execution
func (a *ClaudeAdapter) Cancel() error {
	a.status = core.StatusCancelled
	return nil
}

// GetProgress returns progress information
func (a *ClaudeAdapter) GetProgress() (float64, string) {
	switch a.status {
	case core.StatusPending:
		return 0, "Pending"
	case core.StatusRunning:
		return 50, "Running"
	case core.StatusCompleted:
		return 100, "Completed"
	case core.StatusFailed:
		return 0, "Failed"
	default:
		return 0, string(a.status)
	}
}

// CanHandle checks if the agent can handle a task
func (a *ClaudeAdapter) CanHandle(task core.SubAgentTask) bool {
	if core.MatchesCapabilities(a.config.Capabilities, task) {
		return true
	}

	for _, pattern := range a.config.Context {
		if matchesContext(task, pattern) {
			return true
		}
	}

	return false
}

// GetCapabilities returns agent capabilities
func (a *ClaudeAdapter) GetCapabilities() []string {
	return a.config.Capabilities
}

// SupportsParallel checks if parallel execution is supported
func (a *ClaudeAdapter) SupportsParallel() bool {
	return true
}

// SupportsInteractive checks if interactive mode is supported
func (a *ClaudeAdapter) SupportsInteractive() bool {
	return a.config.Interactive
}

// InitializeProvider initializes with a provider instance
func (a *ClaudeAdapter) InitializeProvider(provider core.Provider) error {
	a.provider = provider
	return nil
}

// AdaptTask adapts a task to Claude-specific format
func (a *ClaudeAdapter) AdaptTask(task core.SubAgentTask) (interface{}, error) {
	// Convert to Claude Task tool format
	claudeTask := map[string]interface{}{
		"task":        task.Description,
		"agent":       a.config.Name,
		"context":     task.Context,
		"constraints": task.Constraints,
	}

	return claudeTask, nil
}

// AdaptResult adapts Claude output to standard result format
func (a *ClaudeAdapter) AdaptResult(result interface{}) (*core.SubAgentResult, error) {
	m, ok := result.(map[string]interface{})
	if !ok {
		return &core.SubAgentResult{
			Status: core.StatusCompleted,
			Output: fmt.Sprintf("%v", result),
		}, nil
	}

	res := &core.SubAgentResult{Status: core.StatusCompleted}

	if status, _ := m["status"].(string); status == "failed" || status == "error" {
		res.Status = core.StatusFailed
		if msg, _ := m["error"].(string); msg != "" {
			res.Error = errors.New(msg)
		} else {
			res.Error = errors.New("task failed")
		}
	}

	if output, _ := m["output"].(string); output != "" {
		res.Output = output
	}

	if meta, _ := m["metadata"].(map[string]interface{}); meta != nil {
		res.Metadata = meta
	}

	for _, raw := range mapSlice(m["artifacts"]) {
		name, _ := raw["name"].(string)
		typ, _ := raw["type"].(string)
		content, _ := raw["content"].(string)
		res.Artifacts = append(res.Artifacts, core.SubAgentArtifact{
			Name:    name,
			Type:    typ,
			Content: []byte(content),
			Size:    int64(len(content)),
		})
	}

	return res, nil
}

func mapSlice(v interface{}) []map[string]interface{} {
	switch s := v.(type) {
	case []map[string]interface{}:
		return s
	case []interface{}:
		out := make([]map[string]interface{}, 0, len(s))
		for _, item := range s {
			if m, ok := item.(map[string]interface{}); ok {
				out = append(out, m)
			}
		}
		return out
	default:
		return nil
	}
}

// GetProviderConfig returns provider-specific configuration
func (a *ClaudeAdapter) GetProviderConfig() map[string]interface{} {
	return a.config.ProviderConfig
}

// createAgentDefinition creates the Claude agent definition file
func (a *ClaudeAdapter) createAgentDefinition() error {
	// Create agent definition
	def := ClaudeAgentDefinition{
		Name:         a.config.Name,
		Description:  a.config.Description,
		Capabilities: a.config.Capabilities,
		Context:      a.config.Context,
		Instructions: a.buildInstructions(),
		Tools:        a.config.Tools,
		MCPServers:   a.config.MCPServers,
	}

	// Write YAML front matter
	yamlData, err := yaml.Marshal(def)
	if err != nil {
		return fmt.Errorf("failed to marshal agent definition: %w", err)
	}

	// Create agent file with Markdown content
	agentFile := filepath.Join(a.agentDir, a.config.Name+".md")
	content := fmt.Sprintf("---\n%s---\n\n# %s\n\n%s\n\n## Instructions\n\n%s\n",
		string(yamlData),
		def.Name,
		def.Description,
		def.Instructions,
	)

	if err := os.WriteFile(agentFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write agent file: %w", err)
	}

	return nil
}

// buildInstructions builds agent instructions from config
func (a *ClaudeAdapter) buildInstructions() string {
	instructions := fmt.Sprintf("You are %s, a specialized agent with the following capabilities:\n\n", a.config.Name)

	for _, cap := range a.config.Capabilities {
		instructions += fmt.Sprintf("- %s\n", cap)
	}

	instructions += "\n## Context Patterns\n\n"
	for _, ctx := range a.config.Context {
		instructions += fmt.Sprintf("- %s\n", ctx)
	}

	if len(a.config.Tools) > 0 {
		instructions += "\n## Available Tools\n\n"
		for _, tool := range a.config.Tools {
			instructions += fmt.Sprintf("- %s\n", tool)
		}
	}

	return instructions
}

// buildTaskPrompt builds the prompt for Task tool invocation
func (a *ClaudeAdapter) buildTaskPrompt(task core.SubAgentTask) string {
	prompt := fmt.Sprintf("Task: %s\n", task.Name)
	prompt += fmt.Sprintf("Description: %s\n", task.Description)

	if task.Input != "" {
		prompt += fmt.Sprintf("\nInput:\n%s\n", task.Input)
	}

	if len(task.Constraints) > 0 {
		prompt += "\nConstraints:\n"
		for _, constraint := range task.Constraints {
			prompt += fmt.Sprintf("- %s\n", constraint)
		}
	}

	return prompt
}

// Helper functions

func matchesContext(task core.SubAgentTask, pattern string) bool {
	// Simple pattern matching - could be enhanced with regex or NLP
	// Check task description and name
	if contains(task.Description, pattern) || contains(task.Name, pattern) {
		return true
	}

	// Check context values
	for _, value := range task.Context {
		if str, ok := value.(string); ok {
			if contains(str, pattern) {
				return true
			}
		}
	}

	return false
}

func contains(text, pattern string) bool {
	// Case-insensitive contains
	return len(text) >= len(pattern) &&
		(text == pattern ||
			len(text) > len(pattern) &&
				(text[:len(pattern)] == pattern ||
					text[len(text)-len(pattern):] == pattern))
}
