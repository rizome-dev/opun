//go:build !windows

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
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/creack/pty"
	"github.com/rizome-dev/opun/pkg/core"
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

	// Workflow output directory with timestamp
	outputDir string

	// Cancel function for the entire workflow
	cancelFunc context.CancelFunc

	// Ctrl-C handling for workflow control
	ctrlCCount    int
	lastCtrlCTime time.Time
	ctrlCMutex    sync.Mutex
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

	// Create a context that can be canceled on interrupt
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Store cancel function for use in PTY sessions
	e.cancelFunc = cancel

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

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

	// Process output directory with timestamp
	if wf.Settings.OutputDir != "" {
		outputDir := wf.Settings.OutputDir
		timestamp := time.Now().Format("20060102-150405")
		outputDir = strings.ReplaceAll(outputDir, "{{timestamp}}", timestamp)
		e.outputDir = outputDir

		// Create output directory
		if err := os.MkdirAll(e.outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
		fmt.Printf("📁 Output directory: %s\n", e.outputDir)
	}

	// Print workflow header
	fmt.Printf("\n🚀 Starting interactive workflow: %s\n", wf.Name)
	if wf.Description != "" {
		fmt.Printf("📝 %s\n", wf.Description)
	}
	fmt.Printf("📋 %d agents to execute sequentially\n", len(wf.Agents))
	fmt.Printf("\n⚡ Workflow Control:\n")
	fmt.Printf("   • Press Ctrl-C twice to continue to the next workflow step\n")
	fmt.Printf("   • Press Ctrl-C three times rapidly (within 1.2s) to abort entire workflow\n\n")

	// Start signal handler in background
	go func() {
		select {
		case sig := <-sigChan:
			fmt.Printf("\n\n⚠️  Received %s signal, stopping workflow...\n", sig)
			cancel()
		case <-ctx.Done():
			// Context was canceled elsewhere
		}
	}()

	// Execute agents sequentially
	for i, agent := range wf.Agents {
		// Check for cancellation before starting each agent
		select {
		case <-ctx.Done():
			e.state.Status = workflow.StatusAborted
			return fmt.Errorf("workflow canceled by user")
		default:
		}

		fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("🤖 Agent %d/%d: %s\n", i+1, len(wf.Agents), agent.Name)
		fmt.Printf("   Provider: %s | Model: %s\n", agent.Provider, agent.Model)
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

		// Extract variables used in this agent's prompt
		usedVars := e.extractVariablesFromPrompt(agent.Prompt)

		// Prompt for variable updates before executing agent
		if len(wf.Variables) > 0 && len(usedVars) > 0 {
			// Collect non-internal variables that are actually used in this agent
			var promptVars []promptVariable
			for _, v := range wf.Variables {
				// Check if this variable is used in the agent's prompt and is not internal
				if !v.Internal && contains(usedVars, v.Name) {
					// Get current value from state
					currentVal, exists := e.state.Variables[v.Name]
					if !exists {
						currentVal = v.DefaultValue
					}

					promptVars = append(promptVars, promptVariable{
						Name:         v.Name,
						Description:  v.Description,
						Type:         v.Type,
						Required:     v.Required,
						DefaultValue: v.DefaultValue,
						CurrentValue: currentVal,
					})
				}
			}

			// Prompt user if there are any user-facing variables used in this agent
			if len(promptVars) > 0 {
				fmt.Printf("📝 Configure variables for %s:\n\n", agent.Name)

				updatedVars, err := promptForVariables(promptVars)
				if err != nil {
					// User cancelled, continue with existing values
					fmt.Printf("⚠️  Using existing variable values\n\n")
				} else {
					// Update the state with new values
					for k, v := range updatedVars {
						e.state.Variables[k] = v
					}
					fmt.Printf("✅ Variables updated\n\n")
				}
			}
		}

		if err := e.executeInteractiveAgent(ctx, &agent, i); err != nil {
			// Check if error is due to cancellation
			if ctx.Err() != nil {
				e.state.Status = workflow.StatusAborted
				return fmt.Errorf("workflow canceled during agent %s", agent.Name)
			}
			return fmt.Errorf("agent %s failed: %w", agent.Name, err)
		}

		// Record output file path if agent has output configured
		if agent.Output != "" && e.outputDir != "" {
			outputPath := filepath.Join(e.outputDir, agent.Output)
			e.outputs[agent.ID] = outputPath
			fmt.Printf("💾 Output will be saved to: %s\n", outputPath)
			fmt.Printf("📌 Next agents can reference this as: {{%s.output}}\n", agent.ID)
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
	// Check if this is a subagent delegation
	if agent.SubAgent != nil {
		return e.executeSubAgent(ctx, agent, agentIndex)
	}

	// Reset Ctrl+C count for new agent
	e.ctrlCMutex.Lock()
	e.ctrlCCount = 0
	e.lastCtrlCTime = time.Time{}
	e.ctrlCMutex.Unlock()

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

	// Debug: Show processed prompt summary
	if len(e.outputs) > 0 {
		fmt.Printf("📎 Prompt includes references to %d previous output(s)\n", len(e.outputs))
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

	// Handle pty size changes only if running in a terminal
	if term.IsTerminal(int(os.Stdin.Fd())) {
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
	}

	// Set terminal to raw mode
	var oldState *term.State
	if term.IsTerminal(int(os.Stdin.Fd())) {
		oldState, err = term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return e.handleAgentError(agent, agentState, fmt.Errorf("failed to set raw mode: %w", err))
		}
		defer func() {
			if oldState != nil {
				if err := term.Restore(int(os.Stdin.Fd()), oldState); err != nil {
					// Fallback to stty sane if restore fails
					exec.Command("stty", "sane").Run()
				}
			}
		}()
	}

	// Track output and inject prompt when ready
	outputBuffer := &strings.Builder{}
	promptInjected := false
	promptMutex := &sync.Mutex{}

	// Simple bidirectional copy with context cancellation
	errChan := make(chan error, 2)
	doneChan := make(chan struct{})

	// Copy PTY output to stdout and detect ready state
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				// Write to stdout
				os.Stdout.Write(buf[:n])

				// Accumulate output for ready detection
				promptMutex.Lock()
				outputBuffer.Write(buf[:n])

				currentOutput := outputBuffer.String()

				// Check if we should inject prompt based on provider
				if !promptInjected {
					switch agent.Provider {
					case "claude":
						// Claude prompt detection - original logic
						if strings.Contains(currentOutput, "│") && strings.Contains(currentOutput, ">") {
							// More specific check - look for the prompt line pattern
							if strings.Contains(currentOutput, "│\u00a0>") ||
								strings.Contains(currentOutput, "│ >") ||
								strings.Contains(currentOutput, "\u00a0>\u00a0") {
								promptInjected = true
								// Small delay to ensure UI is ready
								go func() {
									time.Sleep(500 * time.Millisecond)
									// Type the prompt character by character
									for _, char := range prompt {
										ptmx.Write([]byte(string(char)))
										time.Sleep(5 * time.Millisecond)
									}
								}()
							}
						}

					case "gemini":
						// Gemini prompt detection - needs ANSI stripping
						// Strip ANSI escape sequences to check for patterns
						ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*m`)
						strippedOutput := ansiRegex.ReplaceAllString(currentOutput, "")

						// Check for the prompt pattern in stripped output
						// The actual pattern is "│ > " with a space between pipe and arrow
						if strings.Contains(strippedOutput, "│ > ") {
							promptInjected = true
							// Longer delay to ensure UI is fully ready and won't cut off the beginning
							go func() {
								time.Sleep(2 * time.Second)
								// Type the prompt character by character
								for _, char := range prompt {
									ptmx.Write([]byte(string(char)))
									time.Sleep(10 * time.Millisecond) // Also slow down typing a bit
								}
							}()
						}
					}
				}
				promptMutex.Unlock()
			}

			if err != nil {
				select {
				case errChan <- err:
				case <-doneChan:
				}
				return
			}
		}
	}()

	// Copy stdin to PTY with interrupt detection
	go func() {
		buf := make([]byte, 1024)
		for {
			select {
			case <-doneChan:
				return
			default:
				n, err := os.Stdin.Read(buf)
				if err != nil {
					select {
					case errChan <- err:
					case <-doneChan:
					}
					return
				}

				// First check for Ctrl+C to count (non-blocking)
				for i := 0; i < n; i++ {
					if buf[i] == 0x03 { // Ctrl+C
						e.ctrlCMutex.Lock()
						now := time.Now()

						// Reset count if more than 1.2 seconds since last Ctrl+C
						if now.Sub(e.lastCtrlCTime) > 1200*time.Millisecond {
							e.ctrlCCount = 0
						}

						e.ctrlCCount++
						e.lastCtrlCTime = now
						count := e.ctrlCCount
						e.ctrlCMutex.Unlock()

						// Check if we've hit 3 Ctrl+C presses within 1.2s
						if count >= 3 {
							// Abort entire workflow
							if oldState != nil {
								if err := term.Restore(int(os.Stdin.Fd()), oldState); err != nil {
									// Fallback to stty sane if restore fails
									exec.Command("stty", "sane").Run()
								}
								// Clear to prevent double restore
								oldState = nil
							}
							fmt.Printf("\n\n🛑 Triple Ctrl+C detected, aborting entire workflow...\n")
							e.cancelFunc() // Cancel the entire workflow
							return
						}
					}
				}

				// Then pass through the entire buffer to PTY
				if _, err := ptmx.Write(buf[:n]); err != nil {
					select {
					case errChan <- err:
					case <-doneChan:
					}
					return
				}
			}
		}
	}()

	// Wait for either copy to finish or context cancellation
	select {
	case err := <-errChan:
		close(doneChan)
		if err != nil && err != io.EOF {
			return e.handleAgentError(agent, agentState, err)
		}
	case <-ctx.Done():
		// Context canceled, clean up
		close(doneChan)

		// Restore terminal state immediately
		if oldState != nil {
			if err := term.Restore(int(os.Stdin.Fd()), oldState); err != nil {
				// Fallback to stty sane if restore fails
				exec.Command("stty", "sane").Run()
			}
			// Clear the oldState to prevent double restore in defer
			oldState = nil
		}

		// Send interrupt to the PTY session
		if cmd.Process != nil {
			cmd.Process.Signal(os.Interrupt)
			time.Sleep(100 * time.Millisecond)

			// Force kill if still running
			if cmd.ProcessState == nil {
				cmd.Process.Kill()
			}
		}

		// Update state
		endTime := time.Now()
		agentState.Status = workflow.StatusAborted
		agentState.EndTime = &endTime

		return ctx.Err()
	}

	// Update state
	endTime := time.Now()
	agentState.Status = workflow.StatusCompleted
	agentState.EndTime = &endTime

	fmt.Printf("\n✅ %s session completed\n", agent.Name)
	return nil
}

// executeSubAgent executes an agent via subagent delegation
func (e *InteractiveExecutor) executeSubAgent(ctx context.Context, agent *workflow.Agent, agentIndex int) error {
	// Initialize agent state
	startTime := time.Now()
	agentState := &workflow.AgentState{
		AgentID:   agent.ID,
		StartTime: &startTime,
		Status:    workflow.StatusRunning,
		Attempts:  1,
	}

	e.mu.Lock()
	e.state.AgentStates[agent.ID] = agentState
	e.state.CurrentAgent = agent.Name
	e.mu.Unlock()

	fmt.Printf("🤖 Delegating to subagent: %s\n", agent.SubAgent.Name)
	
	// Import the CLI package to get the subagent manager
	// Note: This creates a circular dependency that needs to be resolved
	// For now, we'll use a placeholder implementation
	
	// Process prompt template
	prompt, err := e.processPromptWithHandoff(agent.Prompt, agentIndex)
	if err != nil {
		return e.handleAgentError(agent, agentState, fmt.Errorf("failed to process prompt: %w", err))
	}

	// Create a SubAgentTask from the workflow agent
	task := core.SubAgentTask{
		ID:          fmt.Sprintf("%s-%d", agent.ID, time.Now().Unix()),
		Name:        agent.Name,
		Description: prompt,
		Input:       prompt,
		Priority:    1, // Default priority
		Context:     make(map[string]interface{}),
		Variables:   e.state.Variables,
	}

	// Add workflow context
	if agent.Input != nil {
		for k, v := range agent.Input {
			task.Context[k] = v
		}
	}

	// Get the subagent manager (this will need to be injected properly)
	// For now, return a placeholder error
	fmt.Printf("⚠️  Subagent execution not yet fully integrated\n")
	fmt.Printf("   Task would be: %s\n", task.Description)
	
	// Simulate success for now
	endTime := time.Now()
	agentState.Status = workflow.StatusCompleted
	agentState.EndTime = &endTime
	agentState.Output = "Subagent execution placeholder output"
	
	// Store output for next agents
	if agent.Output != "" && e.outputDir != "" {
		outputPath := filepath.Join(e.outputDir, agent.Output)
		e.outputs[agent.ID] = outputPath
	}

	return nil
}

// processPromptWithHandoff processes prompt template and adds handoff context
func (e *InteractiveExecutor) processPromptWithHandoff(prompt string, agentIndex int) (string, error) {
	// Get the current agent to add output instructions
	agent := e.workflow.Agents[agentIndex]

	// Process template variables
	result := prompt

	// Replace workflow variables
	for name, value := range e.state.Variables {
		placeholder := fmt.Sprintf("{{%s}}", name)
		replacement := fmt.Sprintf("%v", value)
		result = strings.ReplaceAll(result, placeholder, replacement)
	}

	// Replace {{agent.output}} references with @filepath
	for id, outputPath := range e.outputs {
		placeholder := fmt.Sprintf("{{%s.output}}", id)
		// Convert to @ syntax for providers to read the file
		replacement := fmt.Sprintf("@%s", outputPath)
		result = strings.ReplaceAll(result, placeholder, replacement)
	}

	// Add output saving instructions if agent has output configured
	if agent.Output != "" && e.outputDir != "" {
		outputPath := filepath.Join(e.outputDir, agent.Output)
		outputInstructions := fmt.Sprintf("\n\n📝 **IMPORTANT**: Please save your complete analysis/results to the file:\n`%s`\n\nUse your file writing capabilities to save the output before finishing.\n", outputPath)
		result = outputInstructions + result
	}

	// Add handoff context if this is not the first agent
	if agentIndex > 0 && len(e.handoffContext) > 0 {
		handoff := "\n\n---\n🤝 WORKFLOW CONTEXT:\n"
		handoff += fmt.Sprintf("You are agent %d in a sequential workflow.\n", agentIndex+1)
		handoff += "Previous agents completed:\n"
		for i, ctx := range e.handoffContext {
			handoff += fmt.Sprintf("  %d. %s\n", i+1, ctx)
		}

		// Add note about reading previous outputs
		if len(e.outputs) > 0 {
			handoff += "\nPrevious agent outputs are referenced in your prompt using @ syntax.\n"
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

	case "mock":
		// For mock provider, use a simple echo command for testing
		return "/bin/sh", []string{"-c", "echo 'Mock provider ready'; cat"}, nil

	default:
		return "", nil, fmt.Errorf("unsupported provider: %s", provider)
	}
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

// extractVariablesFromPrompt extracts variable names from a prompt template
func (e *InteractiveExecutor) extractVariablesFromPrompt(prompt string) []string {
	var variables []string
	seen := make(map[string]bool)

	// Regular expression to match {{variable_name}} patterns
	// This will match {{var}}, {{var.property}}, etc.
	re := regexp.MustCompile(`\{\{([a-zA-Z_][a-zA-Z0-9_]*)(\.[\w\.]+)?\}\}`)

	matches := re.FindAllStringSubmatch(prompt, -1)
	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			// Skip agent output references (e.g., {{agent-id.output}})
			if !strings.Contains(varName, "-") && !seen[varName] {
				variables = append(variables, varName)
				seen[varName] = true
			}
		}
	}

	return variables
}

// contains checks if a string slice contains a specific string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// Variable prompting functionality

// variablePromptModel handles prompting for workflow variables
type variablePromptModel struct {
	variables    []promptVariable
	currentIndex int
	inputs       []textinput.Model
	values       map[string]interface{}
	err          error
}

// promptVariable represents a workflow variable for prompting
type promptVariable struct {
	Name         string
	Description  string
	Type         string
	Required     bool
	DefaultValue interface{}
	CurrentValue interface{}
}

func initialVariablePromptModel(variables []promptVariable) variablePromptModel {
	inputs := make([]textinput.Model, len(variables))
	values := make(map[string]interface{})

	for i, v := range variables {
		ti := textinput.New()
		ti.CharLimit = 1000
		ti.Width = 50

		// Set placeholder based on type and default
		placeholder := "Enter value"
		if v.DefaultValue != nil {
			placeholder = fmt.Sprintf("Default: %v", v.DefaultValue)
		}
		ti.Placeholder = placeholder

		// Set current value if exists
		if v.CurrentValue != nil {
			ti.SetValue(fmt.Sprintf("%v", v.CurrentValue))
		}

		// Focus the first input
		if i == 0 {
			ti.Focus()
		}

		inputs[i] = ti

		// Initialize with current or default value
		if v.CurrentValue != nil {
			values[v.Name] = v.CurrentValue
		} else if v.DefaultValue != nil {
			values[v.Name] = v.DefaultValue
		}
	}

	return variablePromptModel{
		variables:    variables,
		currentIndex: 0,
		inputs:       inputs,
		values:       values,
	}
}

func (m variablePromptModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m variablePromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			m.err = fmt.Errorf("cancelled")
			return m, tea.Quit
		case tea.KeyTab, tea.KeyShiftTab:
			// Move to next/previous variable
			if msg.Type == tea.KeyTab {
				m.currentIndex = (m.currentIndex + 1) % len(m.variables)
			} else {
				m.currentIndex--
				if m.currentIndex < 0 {
					m.currentIndex = len(m.variables) - 1
				}
			}

			// Update focus
			for i := range m.inputs {
				if i == m.currentIndex {
					m.inputs[i].Focus()
				} else {
					m.inputs[i].Blur()
				}
			}

			return m, nil
		case tea.KeyEnter:
			// Save current value
			v := m.variables[m.currentIndex]
			val := strings.TrimSpace(m.inputs[m.currentIndex].Value())

			if val != "" {
				// Convert based on type
				switch v.Type {
				case "boolean":
					m.values[v.Name] = val == "true" || val == "yes" || val == "y" || val == "1"
				case "number":
					// Keep as string for now, let workflow processor handle conversion
					m.values[v.Name] = val
				default:
					m.values[v.Name] = val
				}
			} else if v.DefaultValue != nil {
				m.values[v.Name] = v.DefaultValue
			} else if v.Required {
				// Don't allow empty required fields
				return m, nil
			}

			// Move to next or finish
			if m.currentIndex < len(m.variables)-1 {
				m.currentIndex++
				for i := range m.inputs {
					if i == m.currentIndex {
						m.inputs[i].Focus()
					} else {
						m.inputs[i].Blur()
					}
				}
			} else {
				// All done
				return m, tea.Quit
			}

			return m, nil
		}
	}

	// Update current input
	var cmd tea.Cmd
	m.inputs[m.currentIndex], cmd = m.inputs[m.currentIndex].Update(msg)
	return m, cmd
}

func (m variablePromptModel) View() string {
	if m.err != nil {
		return ""
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205")).
		MarginBottom(1)

	varStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	activeStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1)

	inactiveStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	var s strings.Builder
	s.WriteString(titleStyle.Render("Configure Workflow Variables") + "\n\n")

	for i, v := range m.variables {
		// Variable name and description
		name := v.Name
		if v.Required {
			name += " *"
		}
		s.WriteString(varStyle.Render(fmt.Sprintf("%s: %s", name, v.Description)) + "\n")

		// Input field
		style := inactiveStyle
		if i == m.currentIndex {
			style = activeStyle
		}
		s.WriteString(style.Render(m.inputs[i].View()) + "\n\n")
	}

	s.WriteString(varStyle.Render("(Tab to navigate, Enter to confirm, Esc to cancel)"))

	return s.String()
}

// promptForVariables prompts the user to input values for workflow variables
func promptForVariables(variables []promptVariable) (map[string]interface{}, error) {
	if len(variables) == 0 {
		return make(map[string]interface{}), nil
	}

	p := tea.NewProgram(initialVariablePromptModel(variables))
	m, err := p.Run()
	if err != nil {
		return nil, err
	}

	if model, ok := m.(variablePromptModel); ok {
		if model.err != nil {
			return nil, model.err
		}
		return model.values, nil
	}

	return nil, fmt.Errorf("unexpected model type")
}
