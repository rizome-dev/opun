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
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/rizome-dev/opun/pkg/workflow"
	"golang.org/x/term"
)

// InteractiveExecutor executes workflows with interactive sessions
type InteractiveExecutor struct {
	mu sync.Mutex

	// Current workflow being executed
	workflow *workflow.Workflow

	// Current execution state
	state *workflow.ExecutionState

	// Agent outputs for chaining
	outputs map[string]string

	// Handoff context between agents
	handoffContext []string
}

// NewInteractiveExecutor creates a new interactive workflow executor
func NewInteractiveExecutor() *InteractiveExecutor {
	return &InteractiveExecutor{
		outputs:        make(map[string]string),
		handoffContext: make([]string, 0),
	}
}

// Execute executes a workflow with interactive sessions
func (e *InteractiveExecutor) Execute(ctx context.Context, wf *workflow.Workflow, variables map[string]interface{}) error {
	e.workflow = wf

	// Initialize execution state
	e.state = &workflow.ExecutionState{
		WorkflowID:   wf.Name,
		StartTime:    time.Now(),
		Status:       workflow.StatusRunning,
		AgentStates:  make(map[string]*workflow.AgentState),
		Variables:    variables,
		Outputs:      make(map[string]string),
		CurrentAgent: "",
	}

	// Print workflow header
	fmt.Printf("\n🚀 Starting interactive workflow: %s\n", wf.Name)
	if wf.Description != "" {
		fmt.Printf("📝 %s\n", wf.Description)
	}
	fmt.Printf("📋 %d agents to execute sequentially\n\n", len(wf.Agents))

	// Execute agents sequentially
	for i, agent := range wf.Agents {
		fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("🤖 Agent %d/%d: %s\n", i+1, len(wf.Agents), agent.Name)
		fmt.Printf("   Provider: %s | Model: %s\n", agent.Provider, agent.Model)
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

		if err := e.executeInteractiveAgent(ctx, &agent, i); err != nil {
			return fmt.Errorf("agent %s failed: %w", agent.Name, err)
		}

		// Add handoff context for this agent
		e.handoffContext = append(e.handoffContext, fmt.Sprintf("Agent %s (%s) completed", agent.Name, agent.Provider))
	}

	// Update final state
	endTime := time.Now()
	e.state.Status = workflow.StatusCompleted
	e.state.EndTime = &endTime

	fmt.Printf("\n✨ Workflow completed successfully!\n")
	return nil
}

// executeInteractiveAgent executes a single agent interactively
func (e *InteractiveExecutor) executeInteractiveAgent(ctx context.Context, agent *workflow.Agent, agentIndex int) error {
	// Initialize agent state
	startTime := time.Now()
	agentState := &workflow.AgentState{
		AgentID:   agent.ID,
		StartTime: &startTime,
		Status:    workflow.StatusRunning,
		Attempts:  1,
	}

	// Set a default name if not provided
	if agent.Name == "" {
		agent.Name = agent.ID
	}

	e.mu.Lock()
	e.state.AgentStates[agent.ID] = agentState
	e.state.CurrentAgent = agent.Name
	e.mu.Unlock()

	// Get provider command
	providerCmd, providerArgs, err := e.getProviderCommandAndArgs(agent.Provider)
	if err != nil {
		return e.handleAgentError(agent, agentState, err)
	}

	// Process prompt template
	prompt, err := e.processPromptWithHandoff(agent.Prompt, agentIndex)
	if err != nil {
		return e.handleAgentError(agent, agentState, fmt.Errorf("failed to process prompt: %w", err))
	}

	// Create command - use direct command instead of shell
	// #nosec G204 -- providerCmd is from a hardcoded list of known AI provider commands
	cmd := exec.Command(providerCmd, providerArgs...)
	cmd.Env = os.Environ()

	// Start PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return e.handleAgentError(agent, agentState, fmt.Errorf("failed to start PTY: %w", err))
	}
	defer ptmx.Close()

	// Handle pty size changes
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				fmt.Fprintf(os.Stderr, "error resizing pty: %v\n", err)
			}
		}
	}()
	ch <- syscall.SIGWINCH // Initial resize
	defer func() { signal.Stop(ch); close(ch) }()

	// Set terminal to raw mode
	var oldState *term.State
	if term.IsTerminal(int(os.Stdin.Fd())) {
		oldState, err = term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return e.handleAgentError(agent, agentState, fmt.Errorf("failed to set raw mode: %w", err))
		}
		defer term.Restore(int(os.Stdin.Fd()), oldState)
	}

	// Schedule prompt injection after a delay
	go func() {
		// Wait for Claude to be fully ready
		time.Sleep(3 * time.Second)

		fmt.Fprintf(os.Stderr, "\n🤖 Injecting workflow prompt...\n")

		// Type the prompt
		for _, char := range prompt {
			ptmx.Write([]byte(string(char)))
			time.Sleep(5 * time.Millisecond)
		}

		fmt.Fprintf(os.Stderr, "✏️  Edit the prompt as needed, then press Enter to send.\n")
	}()

	// Simple bidirectional copy
	errChan := make(chan error, 2)

	// Copy PTY output to stdout
	go func() {
		_, err := io.Copy(os.Stdout, ptmx)
		errChan <- err
	}()

	// Copy stdin to PTY
	go func() {
		_, err := io.Copy(ptmx, os.Stdin)
		errChan <- err
	}()

	// Wait for either copy to finish
	<-errChan

	// Update state
	endTime := time.Now()
	agentState.Status = workflow.StatusCompleted
	agentState.EndTime = &endTime

	fmt.Printf("\n✅ %s session completed\n", agent.Name)
	return nil
}

// processPromptWithHandoff processes prompt template and adds handoff context
func (e *InteractiveExecutor) processPromptWithHandoff(prompt string, agentIndex int) (string, error) {
	// Process template variables
	result := prompt
	for id, output := range e.outputs {
		placeholder := fmt.Sprintf("{{%s.output}}", id)
		result = strings.ReplaceAll(result, placeholder, output)
	}

	// Add handoff context if this is not the first agent
	if agentIndex > 0 && len(e.handoffContext) > 0 {
		handoff := "\n\n---\n🤝 WORKFLOW CONTEXT:\n"
		handoff += fmt.Sprintf("You are agent %d in a sequential workflow.\n", agentIndex+1)
		handoff += "Previous agents completed:\n"
		for i, ctx := range e.handoffContext {
			handoff += fmt.Sprintf("  %d. %s\n", i+1, ctx)
		}
		handoff += "\nPlease continue the workflow with your assigned task.\n---\n\n"

		result = handoff + result
	}

	return result, nil
}

// getProviderCommandAndArgs returns the command and args to start a provider
func (e *InteractiveExecutor) getProviderCommandAndArgs(provider string) (string, []string, error) {
	switch provider {
	case "claude":
		// Try claude command first
		if _, err := exec.LookPath("claude"); err == nil {
			return "claude", []string{}, nil
		}
		// Fall back to npx claude-code
		if _, err := exec.LookPath("npx"); err == nil {
			return "npx", []string{"claude-code"}, nil
		}
		return "", nil, fmt.Errorf("claude command not found, please install Claude CLI")

	case "gemini":
		if _, err := exec.LookPath("gemini"); err == nil {
			return "gemini", []string{}, nil
		}
		return "", nil, fmt.Errorf("gemini command not found, please install Gemini CLI")

	default:
		return "", nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// saveOutput saves output to a file
func (e *InteractiveExecutor) saveOutput(filename, content string) error {
	if e.workflow != nil && e.workflow.Settings.OutputDir != "" {
		if err := os.MkdirAll(e.workflow.Settings.OutputDir, 0755); err != nil {
			return err
		}
		filename = filepath.Join(e.workflow.Settings.OutputDir, filename)
	}
	return os.WriteFile(filename, []byte(content), 0644)
}

// handleAgentError handles an agent error
func (e *InteractiveExecutor) handleAgentError(agent *workflow.Agent, state *workflow.AgentState, err error) error {
	endTime := time.Now()
	state.Status = workflow.StatusFailed
	state.EndTime = &endTime
	state.Error = &workflow.ExecutionError{
		AgentID:   agent.ID,
		Message:   err.Error(),
		Timestamp: endTime,
		Fatal:     !agent.Settings.ContinueOnError,
	}

	if agent.Settings.ContinueOnError {
		fmt.Printf("⚠️  %s failed but continuing: %v\n", agent.Name, err)
		return nil
	}

	return err
}

// GetState returns the current execution state
func (e *InteractiveExecutor) GetState() *workflow.ExecutionState {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.state
}
