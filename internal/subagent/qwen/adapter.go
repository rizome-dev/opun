package qwen

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
	"strings"
	"time"

	"github.com/rizome-dev/opun/pkg/core"
)

// QwenAdapter adapts Qwen Code to support subagents via custom implementation
type QwenAdapter struct {
	config   core.SubAgentConfig
	provider core.Provider
	status   core.ExecutionStatus
	executor *ToolExecutor
}

// ToolExecutor represents Qwen's non-interactive tool executor
type ToolExecutor struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Tools       []string               `json:"tools"`
	Model       string                 `json:"model"`
	Config      map[string]interface{} `json:"config"`
}

// NewQwenAdapter creates a new Qwen subagent adapter
func NewQwenAdapter(config core.SubAgentConfig) *QwenAdapter {
	return &QwenAdapter{
		config: config,
		status: core.StatusPending,
	}
}

// Name returns the agent name
func (a *QwenAdapter) Name() string {
	return a.config.Name
}

// Config returns the agent configuration
func (a *QwenAdapter) Config() core.SubAgentConfig {
	return a.config
}

// Provider returns the provider type
func (a *QwenAdapter) Provider() core.ProviderType {
	return core.ProviderTypeQwen
}

// Initialize initializes the adapter
func (a *QwenAdapter) Initialize(config core.SubAgentConfig) error {
	a.config = config
	
	// Create tool executor configuration
	a.executor = &ToolExecutor{
		Name:        config.Name,
		Description: config.Description,
		Tools:       config.Tools,
		Model:       config.Model,
		Config:      config.ProviderConfig,
	}
	
	// Set default model if not specified
	if a.executor.Model == "" {
		a.executor.Model = "qwen-coder-32b-instruct"
	}
	
	return nil
}

// Validate validates the adapter configuration
func (a *QwenAdapter) Validate() error {
	if a.config.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	
	if a.config.Description == "" {
		return fmt.Errorf("agent description is required")
	}
	
	return nil
}

// Cleanup cleans up the adapter
func (a *QwenAdapter) Cleanup() error {
	// No specific cleanup needed for Qwen
	return nil
}

// Execute executes a task using Qwen's tool executor pattern
func (a *QwenAdapter) Execute(ctx context.Context, task core.SubAgentTask) (*core.SubAgentResult, error) {
	startTime := time.Now()
	a.status = core.StatusRunning
	
	// Build the execution prompt for Qwen
	execPrompt := a.buildExecutionPrompt(task)
	
	// Execute via provider (this would use PTY in real implementation)
	if a.provider != nil {
		// For Qwen, we use the non-interactive mode with specific termination
		if err := a.provider.InjectPrompt(execPrompt); err != nil {
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
	}
	
	// In a real implementation, we would:
	// 1. Monitor PTY output
	// 2. Parse structured output
	// 3. Detect completion patterns
	
	a.status = core.StatusCompleted
	
	return &core.SubAgentResult{
		TaskID:    task.ID,
		AgentName: a.config.Name,
		Status:    core.StatusCompleted,
		Output:    fmt.Sprintf("Task %s completed by Qwen agent %s", task.Name, a.config.Name),
		StartTime: startTime,
		EndTime:   time.Now(),
		Duration:  time.Since(startTime),
		Metadata: map[string]interface{}{
			"provider": "qwen",
			"method":   "tool_executor",
			"model":    a.executor.Model,
		},
	}, nil
}

// ExecuteAsync executes a task asynchronously
func (a *QwenAdapter) ExecuteAsync(ctx context.Context, task core.SubAgentTask) (<-chan *core.SubAgentResult, error) {
	resultChan := make(chan *core.SubAgentResult, 1)
	
	go func() {
		result, _ := a.Execute(ctx, task)
		resultChan <- result
		close(resultChan)
	}()
	
	return resultChan, nil
}

// Status returns the current execution status
func (a *QwenAdapter) Status() core.ExecutionStatus {
	return a.status
}

// Cancel cancels the current execution
func (a *QwenAdapter) Cancel() error {
	a.status = core.StatusCancelled
	// In real implementation, would send interrupt signal to PTY
	return nil
}

// GetProgress returns progress information
func (a *QwenAdapter) GetProgress() (float64, string) {
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
func (a *QwenAdapter) CanHandle(task core.SubAgentTask) bool {
	// Check if task type is code-related (Qwen specializes in code)
	taskType := extractTaskType(task)
	codeRelated := []string{"code", "coding", "programming", "debug", "refactor", "test", "implement"}
	
	for _, codeType := range codeRelated {
		if strings.Contains(strings.ToLower(taskType), codeType) {
			return true
		}
		if strings.Contains(strings.ToLower(task.Description), codeType) {
			return true
		}
	}
	
	// Check capabilities match
	for _, cap := range a.config.Capabilities {
		capLower := strings.ToLower(cap)
		if strings.Contains(strings.ToLower(task.Description), capLower) {
			return true
		}
		for _, constraint := range task.Constraints {
			if strings.Contains(strings.ToLower(constraint), capLower) {
				return true
			}
		}
	}
	
	return false
}

// GetCapabilities returns agent capabilities
func (a *QwenAdapter) GetCapabilities() []string {
	// Add Qwen-specific capabilities
	caps := append([]string{}, a.config.Capabilities...)
	caps = append(caps, 
		"code_generation",
		"code_review",
		"debugging",
		"refactoring",
		"test_generation",
		"documentation",
	)
	return caps
}

// SupportsParallel checks if parallel execution is supported
func (a *QwenAdapter) SupportsParallel() bool {
	return !a.config.Interactive // Non-interactive mode supports parallel
}

// SupportsInteractive checks if interactive mode is supported
func (a *QwenAdapter) SupportsInteractive() bool {
	return a.config.Interactive
}

// InitializeProvider initializes with a provider instance
func (a *QwenAdapter) InitializeProvider(provider core.Provider) error {
	a.provider = provider
	return nil
}

// AdaptTask adapts a task to Qwen-specific format
func (a *QwenAdapter) AdaptTask(task core.SubAgentTask) (interface{}, error) {
	// Create Qwen-specific task format
	qwenTask := map[string]interface{}{
		"task_id":     task.ID,
		"name":        task.Name,
		"description": task.Description,
		"input":       task.Input,
		"type":        "code_task",
		"language":    extractLanguage(task),
		"context":     task.Context,
		"variables":   task.Variables,
	}
	
	return qwenTask, nil
}

// AdaptResult adapts Qwen output to standard result format
func (a *QwenAdapter) AdaptResult(result interface{}) (*core.SubAgentResult, error) {
	var output string
	
	switch v := result.(type) {
	case string:
		output = v
	case map[string]interface{}:
		// Extract code blocks or structured output
		if code, ok := v["code"].(string); ok {
			output = code
		} else if result, ok := v["result"].(string); ok {
			output = result
		} else {
			jsonBytes, _ := json.Marshal(v)
			output = string(jsonBytes)
		}
	default:
		output = fmt.Sprintf("%v", result)
	}
	
	return &core.SubAgentResult{
		Status: core.StatusCompleted,
		Output: output,
		Metadata: map[string]interface{}{
			"raw_result": result,
			"provider":   "qwen",
		},
	}, nil
}

// GetProviderConfig returns provider-specific configuration
func (a *QwenAdapter) GetProviderConfig() map[string]interface{} {
	return a.config.ProviderConfig
}

// buildExecutionPrompt builds the execution prompt for Qwen
func (a *QwenAdapter) buildExecutionPrompt(task core.SubAgentTask) string {
	var prompt strings.Builder
	
	// Add role definition
	prompt.WriteString(fmt.Sprintf("You are %s, a specialized code assistant.\n", a.config.Name))
	prompt.WriteString(fmt.Sprintf("Description: %s\n\n", a.config.Description))
	
	// Add task context
	prompt.WriteString("## Task\n")
	prompt.WriteString(fmt.Sprintf("Name: %s\n", task.Name))
	prompt.WriteString(fmt.Sprintf("Description: %s\n\n", task.Description))
	
	// Add input if provided
	if task.Input != "" {
		prompt.WriteString("## Input\n")
		prompt.WriteString(fmt.Sprintf("```\n%s\n```\n\n", task.Input))
	}
	
	// Add constraints
	if len(task.Constraints) > 0 {
		prompt.WriteString("## Requirements\n")
		for _, constraint := range task.Constraints {
			prompt.WriteString(fmt.Sprintf("- %s\n", constraint))
		}
		prompt.WriteString("\n")
	}
	
	// Add output instructions
	prompt.WriteString("## Instructions\n")
	prompt.WriteString("1. Analyze the task carefully\n")
	prompt.WriteString("2. Provide a complete solution\n")
	prompt.WriteString("3. Include code in markdown code blocks\n")
	prompt.WriteString("4. Explain your approach\n")
	prompt.WriteString("5. End with '## TASK_COMPLETE' when done\n\n")
	
	// Add tools if specified
	if len(a.config.Tools) > 0 {
		prompt.WriteString("## Available Tools\n")
		for _, tool := range a.config.Tools {
			prompt.WriteString(fmt.Sprintf("- %s\n", tool))
		}
		prompt.WriteString("\n")
	}
	
	prompt.WriteString("Please complete the task now:\n")
	
	return prompt.String()
}

// Helper functions

func extractTaskType(task core.SubAgentTask) string {
	if taskType, ok := task.Context["type"].(string); ok {
		return taskType
	}
	
	// Try to infer from description
	desc := strings.ToLower(task.Description)
	if strings.Contains(desc, "code") || strings.Contains(desc, "implement") {
		return "code"
	}
	if strings.Contains(desc, "test") {
		return "test"
	}
	if strings.Contains(desc, "debug") {
		return "debug"
	}
	if strings.Contains(desc, "refactor") {
		return "refactor"
	}
	
	return "general"
}

func extractLanguage(task core.SubAgentTask) string {
	// Check context for language
	if lang, ok := task.Context["language"].(string); ok {
		return lang
	}
	
	// Check variables
	if lang, ok := task.Variables["language"].(string); ok {
		return lang
	}
	
	// Try to detect from input or description
	languages := map[string][]string{
		"python":     {"python", "py", "def ", "import ", "from "},
		"go":         {"golang", "go", "func ", "package ", "import ("},
		"javascript": {"javascript", "js", "const ", "let ", "var ", "function "},
		"typescript": {"typescript", "ts", "interface ", "type ", ": string", ": number"},
		"java":       {"java", "public class", "private ", "protected "},
		"rust":       {"rust", "fn ", "let mut", "impl ", "trait "},
		"c++":        {"c++", "cpp", "#include", "std::", "template"},
		"c":          {"c", "#include", "void ", "int main"},
	}
	
	text := strings.ToLower(task.Input + " " + task.Description)
	for lang, patterns := range languages {
		for _, pattern := range patterns {
			if strings.Contains(text, pattern) {
				return lang
			}
		}
	}
	
	return "unknown"
}