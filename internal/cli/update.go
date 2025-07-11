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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rizome-dev/opun/internal/promptgarden"
	"github.com/rizome-dev/opun/internal/tools"
	"github.com/rizome-dev/opun/internal/workflow"
	"github.com/spf13/cobra"
)

// UpdateCmd creates the update command
func UpdateCmd() *cobra.Command {
	var (
		isWorkflow bool
		isPrompt   bool
		isAction   bool
		name       string
		path       string
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update existing workflows, prompts, or tools",
		Long:  `Update existing workflows, prompts, or tools that have been added to Opun.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If no flags provided, run interactive mode
			if !isWorkflow && !isPrompt && !isAction {
				return runInteractiveUpdate()
			}

			// Validate required fields
			if name == "" {
				return fmt.Errorf("--name is required")
			}

			if path == "" {
				return fmt.Errorf("--path is required")
			}

			// Determine what to update based on flags
			if isWorkflow {
				return updateWorkflow(name, path)
			}

			if isPrompt {
				return updatePrompt(name, path)
			}

			if isAction {
				return updateAction(name, path)
			}

			return fmt.Errorf("specify either --workflow, --prompt, or --action")
		},
	}

	// Flags
	cmd.Flags().BoolVar(&isWorkflow, "workflow", false, "Update a workflow")
	cmd.Flags().BoolVar(&isPrompt, "prompt", false, "Update a prompt")
	cmd.Flags().BoolVar(&isAction, "action", false, "Update an action")
	cmd.Flags().StringVar(&name, "name", "", "Name of the workflow, prompt, or action to update")
	cmd.Flags().StringVar(&path, "path", "", "New file path for the workflow, prompt, or action")

	// Only one of workflow, prompt, or tool can be used at a time
	cmd.MarkFlagsMutuallyExclusive("workflow", "prompt", "action")

	return cmd
}

// updateWorkflow updates an existing workflow
func updateWorkflow(name, path string) error {
	// Read new workflow file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read workflow file: %w", err)
	}

	// Get workflow directory
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	workflowDir := filepath.Join(home, ".opun", "workflows")

	// Check if workflow exists
	workflowPath := filepath.Join(workflowDir, name+".yaml")
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		return fmt.Errorf("workflow '%s' not found", name)
	}

	// Parse workflow to validate it
	parser := workflow.NewParser(workflowDir)
	wf, err := parser.Parse(data)
	if err != nil {
		return fmt.Errorf("invalid workflow format: %w", err)
	}

	// Set the command name
	wf.Command = name

	// Update workflow
	if err := os.WriteFile(workflowPath, data, 0644); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied: cannot write to %s\nTry: sudo chown -R $USER ~/.opun", workflowDir)
		}
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	fmt.Printf("✓ Updated workflow '%s'\n", name)
	fmt.Printf("  Path: %s\n", workflowPath)

	return nil
}

// updatePrompt updates an existing prompt
func updatePrompt(name, path string) error {
	// Read new prompt file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read prompt file: %w", err)
	}

	// Get prompt garden
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	gardenPath := filepath.Join(home, ".opun", "promptgarden")
	garden, err := promptgarden.NewGarden(gardenPath)
	if err != nil {
		return fmt.Errorf("failed to access prompt garden: %w", err)
	}

	// Check if prompt exists
	existingPrompt, err := garden.GetPrompt(name)
	if err != nil {
		return fmt.Errorf("prompt '%s' not found", name)
	}

	// Update prompt content while preserving metadata
	existingPrompt.Content = string(data)

	// Update tags if present in new content
	newTags := extractTags(string(data))
	if len(newTags) > 0 {
		existingPrompt.Metadata.Tags = newTags
	}

	// Update version
	version := existingPrompt.Metadata.Version
	parts := strings.Split(version, ".")
	if len(parts) == 3 {
		// Increment patch version
		patch := 0
		fmt.Sscanf(parts[2], "%d", &patch)
		existingPrompt.Metadata.Version = fmt.Sprintf("%s.%s.%d", parts[0], parts[1], patch+1)
	}

	// Save updated prompt
	if err := garden.SavePrompt(existingPrompt); err != nil {
		return fmt.Errorf("failed to update prompt: %w", err)
	}

	fmt.Printf("✓ Updated prompt '%s' (version %s)\n", name, existingPrompt.Metadata.Version)
	fmt.Printf("  Access with: promptgarden://%s\n", name)

	return nil
}

// updateAction updates an existing action
func updateAction(name, path string) error {
	// Read new action file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read action file: %w", err)
	}

	// Get actions directory
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	actionsDir := filepath.Join(home, ".opun", "actions")

	// Check if action exists
	actionPath := filepath.Join(actionsDir, name+".yaml")
	if _, err := os.Stat(actionPath); os.IsNotExist(err) {
		return fmt.Errorf("action '%s' not found", name)
	}

	// Create tool loader to validate
	loader := tools.NewLoader(actionsDir)

	// Parse tool to validate it
	tempFile := filepath.Join(os.TempDir(), "temp-tool.yaml")
	if err := os.WriteFile(tempFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	defer os.Remove(tempFile)

	if err := loader.LoadFile(tempFile); err != nil {
		return fmt.Errorf("invalid tool format: %w", err)
	}

	// Update tool file
	if err := os.WriteFile(actionPath, data, 0644); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied: cannot update %s\nTry: sudo chown -R $USER ~/.opun", actionPath)
		}
		return fmt.Errorf("failed to update tool: %w", err)
	}

	fmt.Printf("✓ Updated action '%s'\n", name)
	fmt.Printf("  Tool changes will be available across all AI providers\n")

	return nil
}

// updateAddType constants for compatibility with interactive selection
type updateAddType int

const (
	updateAddTypeWorkflow updateAddType = iota
	updateAddTypePrompt
	updateAddTypeAction
)

// updateTypeItem represents a selectable item in the list
type updateTypeItem struct {
	title       string
	description string
	addType     updateAddType
}

func (i updateTypeItem) FilterValue() string { return i.title }
func (i updateTypeItem) Title() string       { return i.title }
func (i updateTypeItem) Description() string { return i.description }

// addModel is for selecting the type of item to add
type addModel struct {
	list   list.Model
	state  string
	choice *updateTypeItem
}

func (m addModel) Init() tea.Cmd {
	return nil
}

func (m addModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if i, ok := m.list.SelectedItem().(updateTypeItem); ok {
				m.choice = &i
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m addModel) View() string {
	return m.list.View()
}

// Interactive update types
type updateItem struct {
	name        string
	description string
	itemType    string // "workflow", "prompt", or "tool"
}

func (i updateItem) FilterValue() string { return i.name }
func (i updateItem) Title() string       { return i.name }
func (i updateItem) Description() string { return i.description }

type updateModel struct {
	list     list.Model
	items    []updateItem
	choice   *updateItem
	state    string // "selecting", "done"
	itemType string // "workflow", "prompt", or "choosing"
}

func (m updateModel) Init() tea.Cmd {
	return nil
}

func (m updateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if m.state == "selecting" {
				if i, ok := m.list.SelectedItem().(updateItem); ok {
					m.choice = &i
					return m, tea.Quit
				}
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m updateModel) View() string {
	if m.state == "selecting" {
		return m.list.View()
	}
	return ""
}

// runInteractiveUpdate runs the interactive update flow
func runInteractiveUpdate() error {
	// Step 1: Choose type (workflow, prompt, or tool)
	typeChoice, err := selectUpdateType()
	if err != nil {
		return err
	}

	// Step 2: List available items of that type
	var items []updateItem
	switch typeChoice {
	case "workflow":
		items, err = getWorkflowItems()
	case "prompt":
		items, err = getPromptItems()
	case "tool":
		items, err = getToolItems()
	}
	if err != nil {
		return err
	}

	if len(items) == 0 {
		fmt.Printf("No %ss found to update.\n", typeChoice)
		return nil
	}

	// Step 3: Let user select which item to update
	selectedItem, err := selectItemToUpdate(items, typeChoice)
	if err != nil {
		return err
	}

	// Step 4: Get new file path
	path, err := FilePrompt(fmt.Sprintf("Select the new file for %s '%s':", typeChoice, selectedItem.name))
	if err != nil {
		return err
	}
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("file not found: %s", path)
	}

	// Step 5: Confirm update
	confirm, err := Confirm(fmt.Sprintf("Update %s '%s' with contents from %s?", typeChoice, selectedItem.name, filepath.Base(path)))
	if err != nil {
		return err
	}
	if !confirm {
		fmt.Println("Update cancelled.")
		return nil
	}

	// Step 6: Execute the update
	fmt.Printf("\n📝 Updating %s...\n", typeChoice)

	switch typeChoice {
	case "workflow":
		return updateWorkflow(selectedItem.name, path)
	case "prompt":
		return updatePrompt(selectedItem.name, path)
	case "tool":
		return updateAction(selectedItem.name, path)
	default:
		return fmt.Errorf("unknown type: %s", typeChoice)
	}
}

// selectUpdateType lets user choose between workflow, prompt, and tool
func selectUpdateType() (string, error) {
	items := []list.Item{
		updateTypeItem{
			title:       "Workflow",
			description: "Update an existing workflow",
			addType:     updateAddTypeWorkflow,
		},
		updateTypeItem{
			title:       "Prompt",
			description: "Update an existing prompt",
			addType:     updateAddTypePrompt,
		},
		updateTypeItem{
			title:       "Action",
			description: "Update an existing action",
			addType:     updateAddTypeAction,
		},
	}

	const defaultWidth = 60
	listHeight := len(items)*3 + 8

	l := list.New(items, list.NewDefaultDelegate(), defaultWidth, listHeight)
	l.Title = "What would you like to update?"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowPagination(false)
	l.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	model := addModel{list: l, state: "choosing"}

	p := tea.NewProgram(model)
	result, err := p.Run()
	if err != nil {
		return "", err
	}

	if m, ok := result.(addModel); ok && m.choice != nil {
		switch m.choice.addType {
		case updateAddTypeWorkflow:
			return "workflow", nil
		case updateAddTypePrompt:
			return "prompt", nil
		case updateAddTypeAction:
			return "action", nil
		}
	}

	return "", fmt.Errorf("no selection made")
}

// getWorkflowItems returns all available workflows as update items
func getWorkflowItems() ([]updateItem, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	workflowDir := filepath.Join(home, ".opun", "workflows")
	if _, err := os.Stat(workflowDir); os.IsNotExist(err) {
		return []updateItem{}, nil
	}

	entries, err := os.ReadDir(workflowDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflows directory: %w", err)
	}

	var items []updateItem
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".yaml")
		description := "Workflow"

		// Try to read workflow to get description
		data, err := os.ReadFile(filepath.Join(workflowDir, entry.Name()))
		if err == nil {
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

		items = append(items, updateItem{
			name:        name,
			description: description,
			itemType:    "workflow",
		})
	}

	return items, nil
}

// getPromptItems returns all available prompts as update items
func getPromptItems() ([]updateItem, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	gardenPath := filepath.Join(home, ".opun", "promptgarden")
	if _, err := os.Stat(gardenPath); os.IsNotExist(err) {
		return []updateItem{}, nil
	}

	garden, err := promptgarden.NewGarden(gardenPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access prompt garden: %w", err)
	}

	prompts, err := garden.ListPrompts()
	if err != nil {
		return nil, fmt.Errorf("failed to list prompts: %w", err)
	}

	var items []updateItem
	for _, prompt := range prompts {
		description := prompt.Metadata.Description
		if description == "" {
			description = "Prompt"
		}

		items = append(items, updateItem{
			name:        prompt.ID,
			description: description,
			itemType:    "prompt",
		})
	}

	return items, nil
}

// getToolItems returns all available tools as update items
func getToolItems() ([]updateItem, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	actionsDir := filepath.Join(home, ".opun", "actions")
	if _, err := os.Stat(actionsDir); os.IsNotExist(err) {
		return []updateItem{}, nil
	}

	entries, err := os.ReadDir(actionsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read tools directory: %w", err)
	}

	var items []updateItem
	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml")) {
			continue
		}

		name := strings.TrimSuffix(strings.TrimSuffix(entry.Name(), ".yaml"), ".yml")
		description := "Tool"

		// Try to read tool to get description
		data, err := os.ReadFile(filepath.Join(actionsDir, entry.Name()))
		if err == nil {
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

		items = append(items, updateItem{
			name:        name,
			description: description,
			itemType:    "tool",
		})
	}

	return items, nil
}

// selectItemToUpdate lets user select which item to update
func selectItemToUpdate(items []updateItem, itemType string) (*updateItem, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("no items to update")
	}

	// Convert to list items
	listItems := make([]list.Item, len(items))
	for i, item := range items {
		listItems[i] = item
	}

	const defaultWidth = 70
	listHeight := len(items)*3 + 10
	if listHeight > 30 {
		listHeight = 30
	}

	l := list.New(listItems, list.NewDefaultDelegate(), defaultWidth, listHeight)
	l.Title = fmt.Sprintf("Select %s to update", itemType)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("205")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	model := updateModel{
		list:     l,
		items:    items,
		state:    "selecting",
		itemType: itemType,
	}

	p := tea.NewProgram(model)
	result, err := p.Run()
	if err != nil {
		return nil, err
	}

	if m, ok := result.(updateModel); ok && m.choice != nil {
		return m.choice, nil
	}

	return nil, fmt.Errorf("no selection made")
}
