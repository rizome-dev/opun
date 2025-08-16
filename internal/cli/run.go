package cli

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
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rizome-dev/opun/internal/workflow"
	wf "github.com/rizome-dev/opun/pkg/workflow"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// RunCmd creates the run command
func RunCmd() *cobra.Command {
	var (
		workflowName string
		variables    map[string]string
	)

	cmd := &cobra.Command{
		Use:   "run [workflow]",
		Short: "Run a workflow",
		Long:  `Run a workflow by name or from a file path. If no workflow is specified, shows an interactive selection.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// If no workflow specified, run interactive selection
			if len(args) == 0 {
				selectedWorkflow, err := selectWorkflowInteractively()
				if err != nil {
					return err
				}
				workflowName = selectedWorkflow
			} else {
				workflowName = args[0]
			}

			return runWorkflow(workflowName, variables)
		},
	}

	// Flags
	cmd.Flags().StringToStringVarP(&variables, "var", "v", map[string]string{}, "variables to pass to the workflow (key=value)")

	return cmd
}

// runWorkflow executes a workflow
func runWorkflow(name string, vars map[string]string) error {
	ctx := context.Background()

	// Load workflow
	wf, err := loadWorkflow(name)
	if err != nil {
		return fmt.Errorf("failed to load workflow: %w", err)
	}

	// Workflow header is printed by the executor

	// Initialize components

	// Create workflow executor
	executor := workflow.NewExecutor()

	// Convert string vars to interface{}
	variables := make(map[string]interface{})

	// First, populate default values from workflow definition
	for _, v := range wf.Variables {
		if v.DefaultValue != nil {
			variables[v.Name] = v.DefaultValue
		}
	}

	// Then override with user-provided values
	for k, v := range vars {
		variables[k] = v
	}

	// Handle ctrl+c gracefully
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Ensure terminal is restored on interrupt
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigChan)

	// Track if we're handling a signal
	signalHandled := make(chan struct{})
	go func() {
		<-sigChan
		cancel()
		close(signalHandled)
		// Don't exit immediately - let the workflow executor handle cleanup
	}()

	// Execute workflow
	execErr := executor.Execute(ctx, wf, variables)

	// Always ensure terminal is restored, whether we succeeded or failed
	if term.IsTerminal(int(os.Stdin.Fd())) {
		_ = exec.Command("stty", "sane").Run()
	}

	// Check if we were interrupted
	select {
	case <-signalHandled:
		// Give a moment for the executor to finish cleanup
		time.Sleep(200 * time.Millisecond)
		return fmt.Errorf("interrupted")
	default:
		// Normal completion or error
	}

	if execErr != nil {
		return fmt.Errorf("workflow execution failed: %w", execErr)
	}

	// Completion message is printed by the executor
	return nil
}

// loadWorkflow loads a workflow by name or path
func loadWorkflow(name string) (*wf.Workflow, error) {
	var data []byte

	// Check if it's a file path
	if _, err := os.Stat(name); err == nil {
		var readErr error
		data, readErr = os.ReadFile(name)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read workflow file: %w", readErr)
		}
	} else {
		// Load from workflows directory
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}

		workflowPath := filepath.Join(home, ".opun", "workflows", name+".yaml")
		data, err = os.ReadFile(workflowPath)
		if err != nil {
			return nil, fmt.Errorf("workflow '%s' not found", name)
		}
	}

	// Parse workflow
	home, _ := os.UserHomeDir()
	workflowDir := filepath.Join(home, ".opun", "workflows")
	parser := workflow.NewParser(workflowDir)
	return parser.Parse(data)
}

// handleWorkflowEvent handles workflow execution events
func handleWorkflowEvent(event wf.WorkflowEvent) {
	switch event.Type {
	case wf.EventAgentStart:
		fmt.Printf("\n🤖 Agent: %s\n", event.AgentID)

	case wf.EventAgentComplete:
		fmt.Printf("✅ %s completed\n", event.AgentID)

	case wf.EventAgentError:
		fmt.Printf("❌ %s failed: %s\n", event.AgentID, event.Message)

	case wf.EventWorkflowComplete:
		// Handled in main function

	case wf.EventWorkflowError:
		fmt.Printf("\n❌ Workflow error: %s\n", event.Message)
	}
}

// Interactive workflow selection types and functions

type workflowItem struct {
	name        string
	description string
	path        string
}

func (i workflowItem) FilterValue() string { return i.name }
func (i workflowItem) Title() string       { return i.name }
func (i workflowItem) Description() string { return i.description }

type workflowSelectionModel struct {
	list     list.Model
	choice   *workflowItem
	quitting bool
}

func (m workflowSelectionModel) Init() tea.Cmd {
	return nil
}

func (m workflowSelectionModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if i, ok := m.list.SelectedItem().(workflowItem); ok {
				m.choice = &i
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m workflowSelectionModel) View() string {
	if m.quitting || m.choice != nil {
		return ""
	}
	return m.list.View()
}

// selectWorkflowInteractively shows an interactive workflow selection
func selectWorkflowInteractively() (string, error) {
	workflows, err := getAvailableWorkflows()
	if err != nil {
		return "", err
	}

	if len(workflows) == 0 {
		return "", fmt.Errorf("no workflows found. Add workflows using 'opun add --workflow'")
	}

	// Convert to list items
	items := make([]list.Item, len(workflows))
	for i, w := range workflows {
		items[i] = w
	}

	// Calculate height to show 4 entries per page
	// Each item takes about 3 lines (title + description + spacing)
	itemHeight := 3
	entriesPerPage := 4
	listHeight := (entriesPerPage * itemHeight) + 8 // +8 for title, borders, and status bar

	// Create list
	l := list.New(items, list.NewDefaultDelegate(), 80, listHeight)
	l.Title = "Select a workflow to run"
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.SetShowPagination(true)
	l.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	model := workflowSelectionModel{list: l}

	// Run the program
	p := tea.NewProgram(model)
	result, err := p.Run()
	if err != nil {
		return "", err
	}

	// Check if user selected a workflow
	if m, ok := result.(workflowSelectionModel); ok {
		if m.choice != nil {
			return m.choice.name, nil
		}
		if m.quitting {
			return "", fmt.Errorf("workflow selection cancelled")
		}
	}

	return "", fmt.Errorf("no workflow selected")
}

// getAvailableWorkflows returns all available workflows
func getAvailableWorkflows() ([]workflowItem, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	workflowDir := filepath.Join(home, ".opun", "workflows")
	if _, err := os.Stat(workflowDir); os.IsNotExist(err) {
		return []workflowItem{}, nil
	}

	entries, err := os.ReadDir(workflowDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflows directory: %w", err)
	}

	var workflows []workflowItem
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".yaml")
		description := "Workflow"

		// Try to read workflow to get description
		data, err := os.ReadFile(filepath.Join(workflowDir, entry.Name()))
		if err == nil {
			// Simple extraction of description from YAML
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "description:") {
					desc := strings.TrimSpace(strings.TrimPrefix(line, "description:"))
					desc = strings.Trim(desc, "\"'")
					if desc != "" {
						description = desc
					}
					break
				}
			}
		}

		workflows = append(workflows, workflowItem{
			name:        name,
			description: description,
			path:        filepath.Join(workflowDir, entry.Name()),
		})
	}

	return workflows, nil
}
