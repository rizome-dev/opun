package gemini

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
	"time"

	"github.com/rizome-dev/opun/pkg/core"
)

// GeminiAdapter adapts Gemini's programmatic SubAgentScope system
type GeminiAdapter struct {
	config   core.SubAgentConfig
	provider core.Provider
	status   core.ExecutionStatus
	scope    *SubAgentScope
}

// SubAgentScope represents Gemini's subagent configuration
type SubAgentScope struct {
	Name         string        `json:"name"`
	Description  string        `json:"description"`
	PromptConfig PromptConfig  `json:"prompt_config"`
	ModelConfig  ModelConfig   `json:"model_config"`
	RunConfig    RunConfig     `json:"run_config"`
}

// PromptConfig defines the prompt configuration
type PromptConfig struct {
	SystemPrompt string            `json:"system_prompt"`
	Variables    map[string]string `json:"variables"`
	Temperature  float64           `json:"temperature"`
	MaxTokens    int               `json:"max_tokens"`
}

// ModelConfig defines the model configuration
type ModelConfig struct {
	Model          string   `json:"model"`
	Provider       string   `json:"provider"`
	APIKey         string   `json:"api_key,omitempty"`
	ResponseFormat string   `json:"response_format"`
	Tools          []string `json:"tools,omitempty"`
}

// RunConfig defines execution configuration
type RunConfig struct {
	Interactive         bool     `json:"interactive"`
	TerminateOnComplete bool     `json:"terminate_on_complete"`
	OutputCollector     string   `json:"output_collector"`
	MaxIterations       int      `json:"max_iterations"`
	Timeout             int      `json:"timeout"`
}

// NewGeminiAdapter creates a new Gemini subagent adapter
func NewGeminiAdapter(config core.SubAgentConfig) *GeminiAdapter {
	return &GeminiAdapter{
		config: config,
		status: core.StatusPending,
	}
}

// Name returns the agent name
func (a *GeminiAdapter) Name() string {
	return a.config.Name
}

// Config returns the agent configuration
func (a *GeminiAdapter) Config() core.SubAgentConfig {
	return a.config
}

// Provider returns the provider type
func (a *GeminiAdapter) Provider() core.ProviderType {
	return core.ProviderTypeGemini
}

// Initialize initializes the adapter
func (a *GeminiAdapter) Initialize(config core.SubAgentConfig) error {
	a.config = config
	
	// Create SubAgentScope configuration
	a.scope = &SubAgentScope{
		Name:        config.Name,
		Description: config.Description,
		PromptConfig: PromptConfig{
			SystemPrompt: a.buildSystemPrompt(),
			Variables:    make(map[string]string),
			Temperature:  0.7,
			MaxTokens:    4096,
		},
		ModelConfig: ModelConfig{
			Model:          config.Model,
			Provider:       "gemini",
			ResponseFormat: config.OutputFormat,
			Tools:          config.Tools,
		},
		RunConfig: RunConfig{
			Interactive:         config.Interactive,
			TerminateOnComplete: true,
			OutputCollector:     "emitvalue",
			MaxIterations:       3,
			Timeout:             int(config.Timeout.Seconds()),
		},
	}
	
	// Apply provider-specific configuration
	if config.ProviderConfig != nil {
		if temp, ok := config.ProviderConfig["temperature"].(float64); ok {
			a.scope.PromptConfig.Temperature = temp
		}
		if maxTokens, ok := config.ProviderConfig["max_tokens"].(int); ok {
			a.scope.PromptConfig.MaxTokens = maxTokens
		}
		if maxIter, ok := config.ProviderConfig["max_iterations"].(int); ok {
			a.scope.RunConfig.MaxIterations = maxIter
		}
	}
	
	return nil
}

// Validate validates the adapter configuration
func (a *GeminiAdapter) Validate() error {
	if a.config.Name == "" {
		return fmt.Errorf("agent name is required")
	}
	
	if a.config.Model == "" {
		a.config.Model = "gemini-1.5-flash" // Default model
	}
	
	return nil
}

// Cleanup cleans up the adapter
func (a *GeminiAdapter) Cleanup() error {
	// No specific cleanup needed for Gemini
	return nil
}

// Execute executes a task using Gemini's SubAgentScope
func (a *GeminiAdapter) Execute(ctx context.Context, task core.SubAgentTask) (*core.SubAgentResult, error) {
	startTime := time.Now()
	a.status = core.StatusRunning
	
	// Build the execution command for Gemini
	execCommand := a.buildExecutionCommand(task)
	
	// Execute via provider (this would use PTY in real implementation)
	if a.provider != nil {
		if err := a.provider.InjectPrompt(execCommand); err != nil {
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
	// 1. Monitor PTY output for emitvalue calls
	// 2. Collect output until termination
	// 3. Parse the collected values
	
	a.status = core.StatusCompleted
	
	return &core.SubAgentResult{
		TaskID:    task.ID,
		AgentName: a.config.Name,
		Status:    core.StatusCompleted,
		Output:    fmt.Sprintf("Task %s completed by Gemini agent %s", task.Name, a.config.Name),
		StartTime: startTime,
		EndTime:   time.Now(),
		Duration:  time.Since(startTime),
		Metadata: map[string]interface{}{
			"provider":         "gemini",
			"method":           "subagent_scope",
			"output_collector": "emitvalue",
		},
	}, nil
}

// ExecuteAsync executes a task asynchronously
func (a *GeminiAdapter) ExecuteAsync(ctx context.Context, task core.SubAgentTask) (<-chan *core.SubAgentResult, error) {
	resultChan := make(chan *core.SubAgentResult, 1)
	
	go func() {
		result, _ := a.Execute(ctx, task)
		resultChan <- result
		close(resultChan)
	}()
	
	return resultChan, nil
}

// Status returns the current execution status
func (a *GeminiAdapter) Status() core.ExecutionStatus {
	return a.status
}

// Cancel cancels the current execution
func (a *GeminiAdapter) Cancel() error {
	a.status = core.StatusCancelled
	// In real implementation, would send interrupt signal to PTY
	return nil
}

// GetProgress returns progress information
func (a *GeminiAdapter) GetProgress() (float64, string) {
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
func (a *GeminiAdapter) CanHandle(task core.SubAgentTask) bool {
	// Check capabilities match
	for _, taskCap := range extractCapabilities(task) {
		for _, agentCap := range a.config.Capabilities {
			if taskCap == agentCap {
				return true
			}
		}
	}
	
	// Check if task type matches
	if taskType, ok := task.Context["type"].(string); ok {
		for _, cap := range a.config.Capabilities {
			if cap == taskType {
				return true
			}
		}
	}
	
	return len(a.config.Capabilities) == 0 // Accept all if no specific capabilities
}

// GetCapabilities returns agent capabilities
func (a *GeminiAdapter) GetCapabilities() []string {
	return a.config.Capabilities
}

// SupportsParallel checks if parallel execution is supported
func (a *GeminiAdapter) SupportsParallel() bool {
	return !a.config.Interactive // Non-interactive agents can run in parallel
}

// SupportsInteractive checks if interactive mode is supported
func (a *GeminiAdapter) SupportsInteractive() bool {
	return a.config.Interactive
}

// InitializeProvider initializes with a provider instance
func (a *GeminiAdapter) InitializeProvider(provider core.Provider) error {
	a.provider = provider
	return nil
}

// AdaptTask adapts a task to Gemini-specific format
func (a *GeminiAdapter) AdaptTask(task core.SubAgentTask) (interface{}, error) {
	// Create Gemini-specific task format
	geminiTask := map[string]interface{}{
		"task_id":     task.ID,
		"name":        task.Name,
		"description": task.Description,
		"input":       task.Input,
		"variables":   task.Variables,
		"context":     task.Context,
	}
	
	// Add to prompt variables
	if a.scope != nil {
		a.scope.PromptConfig.Variables["task"] = task.Description
		a.scope.PromptConfig.Variables["input"] = task.Input
	}
	
	return geminiTask, nil
}

// AdaptResult adapts Gemini output to standard result format
func (a *GeminiAdapter) AdaptResult(result interface{}) (*core.SubAgentResult, error) {
	// Parse Gemini emitvalue output
	var output string
	
	switch v := result.(type) {
	case string:
		output = v
	case map[string]interface{}:
		// Extract emitted values
		if emitted, ok := v["emitted_values"].([]interface{}); ok {
			jsonBytes, _ := json.Marshal(emitted)
			output = string(jsonBytes)
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
		},
	}, nil
}

// GetProviderConfig returns provider-specific configuration
func (a *GeminiAdapter) GetProviderConfig() map[string]interface{} {
	return a.config.ProviderConfig
}

// buildSystemPrompt builds the system prompt for the subagent
func (a *GeminiAdapter) buildSystemPrompt() string {
	prompt := fmt.Sprintf("You are %s, a specialized AI agent.\n\n", a.config.Name)
	prompt += fmt.Sprintf("Description: %s\n\n", a.config.Description)
	
	if len(a.config.Capabilities) > 0 {
		prompt += "Your capabilities include:\n"
		for _, cap := range a.config.Capabilities {
			prompt += fmt.Sprintf("- %s\n", cap)
		}
		prompt += "\n"
	}
	
	prompt += "Instructions:\n"
	prompt += "1. Process the given task carefully\n"
	prompt += "2. Use self.emitvalue() to output results\n"
	prompt += "3. Emit structured data when possible\n"
	prompt += "4. Complete the task and terminate\n"
	
	return prompt
}

// buildExecutionCommand builds the command to execute a subagent
func (a *GeminiAdapter) buildExecutionCommand(task core.SubAgentTask) string {
	// Build Python code to create and run SubAgentScope
	code := fmt.Sprintf(`
# Create subagent for task: %s
from gemini_cli import SubAgentScope

scope = SubAgentScope(
    name="%s",
    description="%s",
    prompt_config={
        "system_prompt": """%s""",
        "variables": %s,
        "temperature": %f,
        "max_tokens": %d
    },
    model_config={
        "model": "%s",
        "response_format": "%s"
    },
    run_config={
        "interactive": %t,
        "terminate_on_complete": true,
        "output_collector": "emitvalue",
        "max_iterations": %d
    }
)

# Execute task
scope.run("""%s""")
`,
		task.Name,
		a.scope.Name,
		a.scope.Description,
		a.scope.PromptConfig.SystemPrompt,
		formatVariables(task.Variables),
		a.scope.PromptConfig.Temperature,
		a.scope.PromptConfig.MaxTokens,
		a.scope.ModelConfig.Model,
		a.scope.ModelConfig.ResponseFormat,
		a.scope.RunConfig.Interactive,
		a.scope.RunConfig.MaxIterations,
		task.Input,
	)
	
	return code
}

// Helper functions

func extractCapabilities(task core.SubAgentTask) []string {
	var caps []string
	
	if taskCaps, ok := task.Context["capabilities"].([]string); ok {
		caps = append(caps, taskCaps...)
	}
	
	if taskType, ok := task.Context["type"].(string); ok {
		caps = append(caps, taskType)
	}
	
	return caps
}

func formatVariables(vars map[string]interface{}) string {
	if len(vars) == 0 {
		return "{}"
	}
	
	jsonBytes, err := json.Marshal(vars)
	if err != nil {
		return "{}"
	}
	
	return string(jsonBytes)
}