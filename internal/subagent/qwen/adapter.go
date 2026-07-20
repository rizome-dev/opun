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
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rizome-dev/opun/internal/headless"
	"github.com/rizome-dev/opun/pkg/core"
)

// QwenAdapter adapts Qwen Code to support subagents via custom implementation
type QwenAdapter struct {
	config   core.SubAgentConfig
	provider core.Provider
	status   core.ExecutionStatus
	executor *ToolExecutor
	Runner   headless.Runner
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
		Runner: &headless.QwenRunner{},
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
		a.executor.Model = "qwen3-coder-plus"
	}

	return nil
}

// Validate validates the adapter configuration
func (a *QwenAdapter) Validate() error {
	if a.config.Name == "" {
		return fmt.Errorf("agent name is required")
	}

	return nil
}

// Cleanup cleans up the adapter
func (a *QwenAdapter) Cleanup() error {
	// No specific cleanup needed for Qwen
	return nil
}

// Execute executes a task via Qwen's headless print mode
func (a *QwenAdapter) Execute(ctx context.Context, task core.SubAgentTask) (*core.SubAgentResult, error) {
	startTime := time.Now()
	a.status = core.StatusRunning

	result, err := a.Runner.Run(ctx, headless.Request{
		Prompt:       a.buildExecutionPrompt(task),
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
			"provider": "qwen",
			"method":   "headless",
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
	return core.MatchesCapabilities(a.config.Capabilities, task)
}

// GetCapabilities returns agent capabilities
func (a *QwenAdapter) GetCapabilities() []string {
	return a.config.Capabilities
}

// SupportsParallel checks if parallel execution is supported
func (a *QwenAdapter) SupportsParallel() bool {
	return a.config.Parallel
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

	res := &core.SubAgentResult{
		Status: core.StatusCompleted,
		Output: output,
		Metadata: map[string]interface{}{
			"raw_result": result,
			"provider":   "qwen",
		},
	}

	if m, ok := result.(map[string]interface{}); ok {
		if status, _ := m["status"].(string); status == "error" || status == "failed" {
			res.Status = core.StatusFailed
			if msg, _ := m["error"].(string); msg != "" {
				res.Error = errors.New(msg)
			} else {
				res.Error = errors.New("task failed")
			}
		}

		for _, raw := range mapSlice(m["files"]) {
			name, _ := raw["name"].(string)
			content, _ := raw["content"].(string)
			language, _ := raw["language"].(string)
			res.Artifacts = append(res.Artifacts, core.SubAgentArtifact{
				Name:        name,
				Type:        "file",
				Content:     []byte(content),
				ContentType: language,
				Size:        int64(len(content)),
			})
		}
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
