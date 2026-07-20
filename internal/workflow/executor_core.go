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
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rizome-dev/opun/internal/config"
	"github.com/rizome-dev/opun/internal/headless"
	"github.com/rizome-dev/opun/pkg/core"
	"github.com/rizome-dev/opun/pkg/workflow"
)

// InteractiveExecutor executes workflows with headless or interactive sessions
type InteractiveExecutor struct {
	mu sync.Mutex

	workflow *workflow.Workflow

	state *workflow.ExecutionState

	outputs map[string]string

	handoffContext []string

	outputDir string

	cancelFunc context.CancelFunc

	ctrlCCount    int
	lastCtrlCTime time.Time
	ctrlCMutex    sync.Mutex
}

// NewInteractiveExecutor creates a new workflow executor
func NewInteractiveExecutor() *InteractiveExecutor {
	return &InteractiveExecutor{
		outputs:        make(map[string]string),
		handoffContext: make([]string, 0),
	}
}

// executeAgent dispatches an agent to the appropriate execution path
func (e *InteractiveExecutor) executeAgent(ctx context.Context, agent *workflow.Agent, agentIndex int) error {
	if agent.SubAgent != nil {
		return e.executeSubAgent(ctx, agent, agentIndex)
	}
	if agent.Settings.Interactive {
		return e.executeInteractiveAgent(ctx, agent, agentIndex)
	}
	return e.executeHeadlessAgent(ctx, agent, agentIndex)
}

// executeHeadlessAgent executes a single agent via the provider's headless mode
func (e *InteractiveExecutor) executeHeadlessAgent(ctx context.Context, agent *workflow.Agent, agentIndex int) error {
	agentState := e.beginAgent(agent)

	prompt, err := e.processPromptWithHandoff(agent.Prompt, agentIndex)
	if err != nil {
		return e.handleAgentError(agent, agentState, fmt.Errorf("failed to process prompt: %w", err))
	}

	runner, err := headless.NewRunner(agent.Provider)
	if err != nil {
		return e.handleAgentError(agent, agentState, err)
	}

	fmt.Printf("⏳ Running %s headless...\n", agent.Provider)

	result, err := runner.Run(ctx, headless.Request{
		Prompt:         prompt,
		Model:          agent.Model,
		PermissionMode: agent.Settings.PermissionMode,
		Timeout:        time.Duration(agent.Settings.Timeout) * time.Second,
	})
	if err != nil {
		return e.handleAgentError(agent, agentState, err)
	}

	fmt.Println(result.Output)

	e.storeAgentOutput(agent, result.Output)
	e.completeAgent(agentState, result.Output)

	fmt.Printf("\n✅ %s completed\n", agent.Name)
	return nil
}

// executeSubAgent executes an agent via subagent delegation
func (e *InteractiveExecutor) executeSubAgent(ctx context.Context, agent *workflow.Agent, agentIndex int) error {
	agentState := e.beginAgent(agent)

	fmt.Printf("🤖 Delegating to subagent: %s\n", agent.SubAgent.Name)

	prompt, err := e.processPromptWithHandoff(agent.Prompt, agentIndex)
	if err != nil {
		return e.handleAgentError(agent, agentState, fmt.Errorf("failed to process prompt: %w", err))
	}

	subCfg, err := loadSubAgentConfig(agent.SubAgent.Name)
	if err != nil {
		return e.handleAgentError(agent, agentState, err)
	}

	runner, err := headless.NewRunner(string(subCfg.Provider))
	if err != nil {
		return e.handleAgentError(agent, agentState, err)
	}

	result, err := runner.Run(ctx, headless.Request{
		Prompt:         prompt,
		SystemPrompt:   subCfg.SystemPrompt,
		Model:          subCfg.Model,
		PermissionMode: agent.Settings.PermissionMode,
		Timeout:        time.Duration(agent.Settings.Timeout) * time.Second,
	})
	if err != nil {
		return e.handleAgentError(agent, agentState, err)
	}

	fmt.Println(result.Output)

	e.storeAgentOutput(agent, result.Output)
	e.completeAgent(agentState, result.Output)

	fmt.Printf("\n✅ Subagent %s completed\n", agent.SubAgent.Name)
	return nil
}

func loadSubAgentConfig(name string) (*core.SubAgentConfig, error) {
	loader, err := config.NewSubAgentConfigLoader()
	if err != nil {
		return nil, fmt.Errorf("failed to create subagent loader: %w", err)
	}

	configs, err := loader.LoadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to load subagent configs: %w", err)
	}

	for i := range configs {
		if configs[i].Name == name {
			return &configs[i], nil
		}
	}

	return nil, fmt.Errorf("subagent %q not found; create it with 'opun subagent create'", name)
}

func (e *InteractiveExecutor) beginAgent(agent *workflow.Agent) *workflow.AgentState {
	startTime := time.Now()
	agentState := &workflow.AgentState{
		AgentID:   agent.ID,
		StartTime: &startTime,
		Status:    workflow.StatusRunning,
		Attempts:  1,
	}

	if agent.Name == "" {
		agent.Name = agent.ID
	}

	e.mu.Lock()
	e.state.AgentStates[agent.ID] = agentState
	e.state.CurrentAgent = agent.Name
	e.mu.Unlock()

	return agentState
}

func (e *InteractiveExecutor) completeAgent(agentState *workflow.AgentState, output string) {
	endTime := time.Now()
	agentState.Status = workflow.StatusCompleted
	agentState.EndTime = &endTime
	agentState.Output = output
}

func (e *InteractiveExecutor) storeAgentOutput(agent *workflow.Agent, output string) {
	if agent.Output == "" || e.outputDir == "" {
		return
	}
	outputPath := filepath.Join(e.outputDir, agent.Output)
	if err := os.WriteFile(outputPath, []byte(output), 0644); err != nil {
		fmt.Printf("⚠️  Failed to write output file %s: %v\n", outputPath, err)
		return
	}
	e.outputs[agent.ID] = outputPath
}

// processPromptWithHandoff processes prompt template and adds handoff context
func (e *InteractiveExecutor) processPromptWithHandoff(prompt string, agentIndex int) (string, error) {
	agent := e.workflow.Agents[agentIndex]

	result := prompt

	for name, value := range e.state.Variables {
		placeholder := fmt.Sprintf("{{%s}}", name)
		replacement := fmt.Sprintf("%v", value)
		result = strings.ReplaceAll(result, placeholder, replacement)
	}

	for id, outputPath := range e.outputs {
		placeholder := fmt.Sprintf("{{%s.output}}", id)
		replacement := fmt.Sprintf("@%s", outputPath)
		result = strings.ReplaceAll(result, placeholder, replacement)
	}

	if agent.Settings.Interactive && agent.Output != "" && e.outputDir != "" {
		outputPath := filepath.Join(e.outputDir, agent.Output)
		outputInstructions := fmt.Sprintf("\n\n📝 **IMPORTANT**: Please save your complete analysis/results to the file:\n`%s`\n\nUse your file writing capabilities to save the output before finishing.\n", outputPath)
		result = outputInstructions + result
	}

	if agentIndex > 0 && len(e.handoffContext) > 0 {
		handoff := "\n\n---\n🤝 WORKFLOW CONTEXT:\n"
		handoff += fmt.Sprintf("You are agent %d in a sequential workflow.\n", agentIndex+1)
		handoff += "Previous agents completed:\n"
		for i, ctx := range e.handoffContext {
			handoff += fmt.Sprintf("  %d. %s\n", i+1, ctx)
		}

		if len(e.outputs) > 0 {
			handoff += "\nPrevious agent outputs are referenced in your prompt using @ syntax.\n"
		}

		handoff += "\nPlease continue the workflow with your assigned task.\n---\n\n"

		result = handoff + result
	}

	return result, nil
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

	re := regexp.MustCompile(`\{\{([a-zA-Z_][a-zA-Z0-9_]*)(\.[\w\.]+)?\}\}`)

	matches := re.FindAllStringSubmatch(prompt, -1)
	for _, match := range matches {
		if len(match) > 1 {
			varName := match[1]
			if !strings.Contains(varName, "-") && !seen[varName] {
				variables = append(variables, varName)
				seen[varName] = true
			}
		}
	}

	return variables
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

type variablePromptModel struct {
	variables    []promptVariable
	currentIndex int
	inputs       []textinput.Model
	values       map[string]interface{}
	err          error
}

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

		placeholder := "Enter value"
		if v.DefaultValue != nil {
			placeholder = fmt.Sprintf("Default: %v", v.DefaultValue)
		}
		ti.Placeholder = placeholder

		if v.CurrentValue != nil {
			ti.SetValue(fmt.Sprintf("%v", v.CurrentValue))
		}

		if i == 0 {
			ti.Focus()
		}

		inputs[i] = ti

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
			if msg.Type == tea.KeyTab {
				m.currentIndex = (m.currentIndex + 1) % len(m.variables)
			} else {
				m.currentIndex--
				if m.currentIndex < 0 {
					m.currentIndex = len(m.variables) - 1
				}
			}

			for i := range m.inputs {
				if i == m.currentIndex {
					m.inputs[i].Focus()
				} else {
					m.inputs[i].Blur()
				}
			}

			return m, nil
		case tea.KeyEnter:
			v := m.variables[m.currentIndex]
			val := strings.TrimSpace(m.inputs[m.currentIndex].Value())

			if val != "" {
				switch v.Type {
				case "boolean":
					m.values[v.Name] = val == "true" || val == "yes" || val == "y" || val == "1"
				case "number":
					m.values[v.Name] = val
				default:
					m.values[v.Name] = val
				}
			} else if v.DefaultValue != nil {
				m.values[v.Name] = v.DefaultValue
			} else if v.Required {
				return m, nil
			}

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
				return m, tea.Quit
			}

			return m, nil
		}
	}

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
		name := v.Name
		if v.Required {
			name += " *"
		}
		s.WriteString(varStyle.Render(fmt.Sprintf("%s: %s", name, v.Description)) + "\n")

		style := inactiveStyle
		if i == m.currentIndex {
			style = activeStyle
		}
		s.WriteString(style.Render(m.inputs[i].View()) + "\n\n")
	}

	s.WriteString(varStyle.Render("(Tab to navigate, Enter to confirm, Esc to cancel)"))

	return s.String()
}

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
