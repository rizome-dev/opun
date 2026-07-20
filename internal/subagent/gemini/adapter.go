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
	"errors"
	"fmt"
	"time"

	"github.com/rizome-dev/opun/internal/headless"
	"github.com/rizome-dev/opun/pkg/core"
)

// GeminiAdapter adapts Gemini's programmatic SubAgentScope system
type GeminiAdapter struct {
	config   core.SubAgentConfig
	provider core.Provider
	status   core.ExecutionStatus
	scope    *SubAgentScope
	Runner   headless.Runner
}

// SubAgentScope represents Gemini's subagent configuration
type SubAgentScope struct {
	Name         string       `json:"name"`
	Description  string       `json:"description"`
	PromptConfig PromptConfig `json:"prompt_config"`
	ModelConfig  ModelConfig  `json:"model_config"`
	RunConfig    RunConfig    `json:"run_config"`
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
	Interactive         bool   `json:"interactive"`
	TerminateOnComplete bool   `json:"terminate_on_complete"`
	OutputCollector     string `json:"output_collector"`
	MaxIterations       int    `json:"max_iterations"`
	Timeout             int    `json:"timeout"`
}

// NewGeminiAdapter creates a new Gemini subagent adapter
func NewGeminiAdapter(config core.SubAgentConfig) *GeminiAdapter {
	return &GeminiAdapter{
		config: config,
		status: core.StatusPending,
		Runner: &headless.GeminiRunner{},
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
		a.config.Model = "auto"
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
			"provider": "gemini",
			"method":   "headless",
		},
	}, nil
}

func (a *GeminiAdapter) buildTaskPrompt(task core.SubAgentTask) string {
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
	return core.MatchesCapabilities(a.config.Capabilities, task)
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

	res := &core.SubAgentResult{
		Status: core.StatusCompleted,
		Output: output,
		Metadata: map[string]interface{}{
			"raw_result": result,
		},
	}

	if m, ok := result.(map[string]interface{}); ok {
		if status, _ := m["status"].(string); status == "error" || status == "failed" {
			res.Status = core.StatusFailed
			if msg, _ := m["error_message"].(string); msg != "" {
				res.Error = errors.New(msg)
			} else if msg, _ := m["error"].(string); msg != "" {
				res.Error = errors.New(msg)
			} else {
				res.Error = errors.New("task failed")
			}
		}
	}

	return res, nil
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
