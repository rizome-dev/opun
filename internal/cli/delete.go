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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// DeleteCmd creates the delete command
func DeleteCmd() *cobra.Command {
	var (
		isWorkflow bool
		isPrompt   bool
		isAction   bool
		name       string
		force      bool
	)

	cmd := &cobra.Command{
		Use:   "delete [workflow|prompt|action]",
		Short: "Delete workflows, prompts, or actions from Opun",
		Long: `Delete workflows, prompts, or actions that have been added to Opun.

Examples:
  # Delete a workflow
  opun delete workflow my-workflow
  
  # Delete a prompt
  opun delete prompt my-prompt
  
  # Delete an action
  opun delete action my-action
  
  # Interactive mode
  opun delete`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if subcommand is provided as positional argument
			if len(args) > 0 {
				switch args[0] {
				case "workflow":
					isWorkflow = true
					if len(args) > 1 {
						name = args[1]
					}
				case "prompt":
					isPrompt = true
					if len(args) > 1 {
						name = args[1]
					}
				case "action":
					isAction = true
					if len(args) > 1 {
						name = args[1]
					}
				default:
					return fmt.Errorf("unknown type: %s", args[0])
				}
			}

			// If no flags or args provided, run interactive mode
			if !isWorkflow && !isPrompt && !isAction {
				return runInteractiveDelete()
			}

			// Validate required fields
			if name == "" {
				return fmt.Errorf("name is required")
			}

			// Determine what to delete based on flags
			if isWorkflow {
				return deleteWorkflow(name, force)
			}

			if isPrompt {
				return deletePrompt(name, force)
			}

			if isAction {
				return deleteAction(name, force)
			}

			return fmt.Errorf("specify either workflow, prompt, or action")
		},
	}

	// Flags
	cmd.Flags().BoolVar(&isWorkflow, "workflow", false, "Delete a workflow")
	cmd.Flags().BoolVar(&isPrompt, "prompt", false, "Delete a prompt")
	cmd.Flags().BoolVar(&isAction, "action", false, "Delete an action")
	cmd.Flags().StringVar(&name, "name", "", "Name of the item to delete")
	cmd.Flags().BoolVar(&force, "force", false, "Force deletion without confirmation")

	// Only one type can be used at a time
	cmd.MarkFlagsMutuallyExclusive("workflow", "prompt", "action")

	return cmd
}

// deleteWorkflow deletes a workflow from the system
func deleteWorkflow(name string, force bool) error {
	// Get workflow directory
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	workflowDir := filepath.Join(home, ".opun", "workflows")
	workflowPath := filepath.Join(workflowDir, name+".yaml")

	// Check if workflow exists
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		return fmt.Errorf("workflow '%s' not found", name)
	}

	// Confirm deletion if not forced
	if !force {
		confirm, err := Confirm(fmt.Sprintf("Are you sure you want to delete workflow '%s'?", name))
		if err != nil {
			return err
		}
		if !confirm {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	// Delete workflow file
	if err := os.Remove(workflowPath); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied: cannot delete %s\nTry: sudo chown -R $USER ~/.opun", workflowPath)
		}
		return fmt.Errorf("failed to delete workflow: %w", err)
	}

	// Remove from config
	workflows := viper.GetStringSlice("workflows")
	var updatedWorkflows []string
	for _, w := range workflows {
		if w != name {
			updatedWorkflows = append(updatedWorkflows, w)
		}
	}
	viper.Set("workflows", updatedWorkflows)

	// Save config
	configPath := viper.ConfigFileUsed()
	if configPath == "" {
		configPath = filepath.Join(home, ".opun", "config.yaml")
	}

	if err := viper.WriteConfigAs(configPath); err != nil {
		fmt.Printf("⚠️  Warning: Failed to update config: %v\n", err)
	}

	fmt.Printf("✓ Deleted workflow '%s'\n", name)

	return nil
}

// deletePrompt deletes a prompt from the prompt garden
func deletePrompt(name string, force bool) error {
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
	_, err = garden.GetPrompt(name)
	if err != nil {
		return fmt.Errorf("prompt '%s' not found", name)
	}

	// Confirm deletion if not forced
	if !force {
		confirm, err := Confirm(fmt.Sprintf("Are you sure you want to delete prompt '%s'?", name))
		if err != nil {
			return err
		}
		if !confirm {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	// Delete prompt
	if err := garden.DeletePrompt(name); err != nil {
		return fmt.Errorf("failed to delete prompt: %w", err)
	}

	fmt.Printf("✓ Deleted prompt '%s'\n", name)

	return nil
}

// deleteAction deletes an action from the system
func deleteAction(name string, force bool) error {
	// Get actions directory
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	actionsDir := filepath.Join(home, ".opun", "actions")
	actionPath := filepath.Join(actionsDir, name+".yaml")

	// Check if action exists
	if _, err := os.Stat(actionPath); os.IsNotExist(err) {
		// Try .yml extension
		actionPath = filepath.Join(actionsDir, name+".yml")
		if _, err := os.Stat(actionPath); os.IsNotExist(err) {
			return fmt.Errorf("action '%s' not found", name)
		}
	}

	// Confirm deletion if not forced
	if !force {
		confirm, err := Confirm(fmt.Sprintf("Are you sure you want to delete action '%s'?", name))
		if err != nil {
			return err
		}
		if !confirm {
			fmt.Println("Deletion cancelled.")
			return nil
		}
	}

	// Delete action file
	if err := os.Remove(actionPath); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied: cannot delete %s\nTry: sudo chown -R $USER ~/.opun", actionPath)
		}
		return fmt.Errorf("failed to delete action: %w", err)
	}

	fmt.Printf("✓ Deleted action '%s'\n", name)

	return nil
}

// addType constants for compatibility with interactive selection
type addType int

const (
	addTypeWorkflow addType = iota
	addTypePrompt
	addTypeAction
)

// item represents a selectable item in the list
type item struct {
	title       string
	description string
	addType     addType
}

func (i item) FilterValue() string { return i.title }
func (i item) Title() string       { return i.title }
func (i item) Description() string { return i.description }

// Interactive delete types
type deleteItem struct {
	name        string
	description string
	itemType    string // "workflow", "prompt", or "action"
}

func (i deleteItem) FilterValue() string { return i.name }
func (i deleteItem) Title() string       { return i.name }
func (i deleteItem) Description() string { return i.description }

type deleteModel struct {
	list     list.Model
	items    []deleteItem
	choice   *deleteItem
	state    string // "selecting", "done"
	itemType string // "workflow", "prompt", "action", or "choosing"
}

func (m deleteModel) Init() tea.Cmd {
	return nil
}

func (m deleteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if m.state == "selecting" {
				if i, ok := m.list.SelectedItem().(deleteItem); ok {
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

func (m deleteModel) View() string {
	if m.state == "selecting" {
		return m.list.View()
	}
	return ""
}

// Multi-select model for batch deletion
type multiDeleteModel struct {
	list     list.Model
	items    []deleteItem
	selected map[string]bool
	state    string
	itemType string
}

type toggleableDeleteItem struct {
	deleteItem
	selected bool
}

func (i toggleableDeleteItem) FilterValue() string { return i.name }
func (i toggleableDeleteItem) Title() string {
	checkbox := "☐"
	if i.selected {
		checkbox = "☑"
	}
	return fmt.Sprintf("%s %s", checkbox, i.name)
}
func (i toggleableDeleteItem) Description() string { return i.description }

func (m multiDeleteModel) Init() tea.Cmd {
	return nil
}

func (m multiDeleteModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		case " ":
			// Toggle selection
			if i, ok := m.list.SelectedItem().(toggleableDeleteItem); ok {
				idx := m.list.Index()
				m.selected[i.name] = !m.selected[i.name]

				// Update the item in the list
				items := m.list.Items()
				if idx < len(items) {
					if item, ok := items[idx].(toggleableDeleteItem); ok {
						item.selected = !item.selected
						items[idx] = item
						m.list.SetItems(items)
					}
				}
			}
		case "a":
			// Select all
			items := m.list.Items()
			for i, item := range items {
				if tItem, ok := item.(toggleableDeleteItem); ok {
					tItem.selected = true
					m.selected[tItem.name] = true
					items[i] = tItem
				}
			}
			m.list.SetItems(items)
		case "n":
			// Select none
			items := m.list.Items()
			for i, item := range items {
				if tItem, ok := item.(toggleableDeleteItem); ok {
					tItem.selected = false
					m.selected[tItem.name] = false
					items[i] = tItem
				}
			}
			m.list.SetItems(items)
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m multiDeleteModel) View() string {
	selectedCount := 0
	for _, selected := range m.selected {
		if selected {
			selectedCount++
		}
	}

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true)

	instructionsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	status := statusStyle.Render(fmt.Sprintf("Selected: %d/%d items", selectedCount, len(m.items)))
	instructions := instructionsStyle.Render("Space to toggle • a to select all • n to select none • Enter to confirm • q to cancel")

	return m.list.View() + "\n\n" + status + "\n" + instructions
}

// selectTypeModel is for selecting the type of item to delete
type selectTypeModel struct {
	list   list.Model
	state  string
	choice *item
}

func (m selectTypeModel) Init() tea.Cmd {
	return nil
}

func (m selectTypeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if i, ok := m.list.SelectedItem().(item); ok {
				m.choice = &i
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m selectTypeModel) View() string {
	return m.list.View()
}

// runInteractiveDelete runs the interactive delete flow
func runInteractiveDelete() error {
	// Step 1: Choose type (workflow, prompt, or action)
	typeChoice, err := selectDeleteType()
	if err != nil {
		return err
	}

	// Step 2: List available items of that type
	var items []deleteItem
	switch typeChoice {
	case "workflow":
		items, err = getWorkflowDeleteItems()
	case "prompt":
		items, err = getPromptDeleteItems()
	case "action":
		items, err = getActionDeleteItems()
	}
	if err != nil {
		return err
	}

	if len(items) == 0 {
		fmt.Printf("No %ss found to delete.\n", typeChoice)
		return nil
	}

	// Step 3: Single or multi-select mode?
	if len(items) > 1 {
		multiMode, err := Confirm("Do you want to select multiple items to delete?")
		if err != nil {
			return err
		}

		if multiMode {
			return runMultiDelete(items, typeChoice)
		}
	}

	// Step 4: Single item selection
	selectedItem, err := selectItemToDelete(items, typeChoice)
	if err != nil {
		return err
	}

	// Step 5: Execute the deletion
	fmt.Printf("\n🗑️  Deleting %s...\n", typeChoice)

	switch typeChoice {
	case "workflow":
		return deleteWorkflow(selectedItem.name, false)
	case "prompt":
		return deletePrompt(selectedItem.name, false)
	case "action":
		return deleteAction(selectedItem.name, false)
	default:
		return fmt.Errorf("unknown type: %s", typeChoice)
	}
}

// runMultiDelete handles multi-select deletion
func runMultiDelete(items []deleteItem, itemType string) error {
	// Convert to toggleable items
	listItems := make([]list.Item, len(items))
	selected := make(map[string]bool)

	for i, item := range items {
		listItems[i] = toggleableDeleteItem{
			deleteItem: item,
			selected:   false,
		}
		selected[item.name] = false
	}

	const defaultWidth = 70
	listHeight := len(items)*3 + 10
	if listHeight > 30 {
		listHeight = 30
	}

	l := list.New(listItems, list.NewDefaultDelegate(), defaultWidth, listHeight)
	l.Title = fmt.Sprintf("Select %ss to delete (multi-select)", itemType)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("196")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	model := multiDeleteModel{
		list:     l,
		items:    items,
		selected: selected,
		state:    "selecting",
		itemType: itemType,
	}

	p := tea.NewProgram(model)
	result, err := p.Run()
	if err != nil {
		return err
	}

	if m, ok := result.(multiDeleteModel); ok {
		// Count selected items
		var toDelete []string
		for name, isSelected := range m.selected {
			if isSelected {
				toDelete = append(toDelete, name)
			}
		}

		if len(toDelete) == 0 {
			fmt.Println("No items selected for deletion.")
			return nil
		}

		// Confirm batch deletion
		confirm, err := Confirm(fmt.Sprintf("Are you sure you want to delete %d %s(s)?", len(toDelete), itemType))
		if err != nil {
			return err
		}
		if !confirm {
			fmt.Println("Deletion cancelled.")
			return nil
		}

		// Execute deletions
		fmt.Printf("\n🗑️  Deleting %d %s(s)...\n", len(toDelete), itemType)

		var errors []string
		for _, name := range toDelete {
			var err error
			switch itemType {
			case "workflow":
				err = deleteWorkflow(name, true)
			case "prompt":
				err = deletePrompt(name, true)
			case "action":
				err = deleteAction(name, true)
			}

			if err != nil {
				errors = append(errors, fmt.Sprintf("  ✗ %s: %v", name, err))
			}
		}

		if len(errors) > 0 {
			fmt.Println("\n⚠️  Some deletions failed:")
			for _, e := range errors {
				fmt.Println(e)
			}
			return fmt.Errorf("%d deletion(s) failed", len(errors))
		}

		fmt.Printf("\n✓ Successfully deleted %d %s(s)\n", len(toDelete), itemType)
	}

	return nil
}

// selectDeleteType lets user choose between workflow, prompt, and action
func selectDeleteType() (string, error) {
	items := []list.Item{
		item{
			title:       "Workflow",
			description: "Delete existing workflows",
			addType:     addTypeWorkflow,
		},
		item{
			title:       "Prompt",
			description: "Delete existing prompts",
			addType:     addTypePrompt,
		},
		item{
			title:       "Action",
			description: "Delete existing actions",
			addType:     addTypeAction,
		},
	}

	const defaultWidth = 60
	listHeight := len(items)*3 + 8

	l := list.New(items, list.NewDefaultDelegate(), defaultWidth, listHeight)
	l.Title = "What would you like to delete?"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowPagination(false)
	l.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("196")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	// Create a temporary model for selection
	model := &selectTypeModel{list: l, state: "choosing"}

	p := tea.NewProgram(model)
	result, err := p.Run()
	if err != nil {
		return "", err
	}

	if m, ok := result.(*selectTypeModel); ok && m.choice != nil {
		switch m.choice.addType {
		case addTypeWorkflow:
			return "workflow", nil
		case addTypePrompt:
			return "prompt", nil
		case addTypeAction:
			return "action", nil
		}
	}

	return "", fmt.Errorf("no selection made")
}

// getWorkflowDeleteItems returns all available workflows as delete items
func getWorkflowDeleteItems() ([]deleteItem, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	workflowDir := filepath.Join(home, ".opun", "workflows")
	if _, err := os.Stat(workflowDir); os.IsNotExist(err) {
		return []deleteItem{}, nil
	}

	entries, err := os.ReadDir(workflowDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflows directory: %w", err)
	}

	var items []deleteItem
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

		items = append(items, deleteItem{
			name:        name,
			description: description,
			itemType:    "workflow",
		})
	}

	return items, nil
}

// getPromptDeleteItems returns all available prompts as delete items
func getPromptDeleteItems() ([]deleteItem, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	gardenPath := filepath.Join(home, ".opun", "promptgarden")
	if _, err := os.Stat(gardenPath); os.IsNotExist(err) {
		return []deleteItem{}, nil
	}

	garden, err := promptgarden.NewGarden(gardenPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access prompt garden: %w", err)
	}

	prompts, err := garden.ListPrompts()
	if err != nil {
		return nil, fmt.Errorf("failed to list prompts: %w", err)
	}

	var items []deleteItem
	for _, prompt := range prompts {
		description := prompt.Metadata.Description
		if description == "" {
			description = "Prompt"
		}

		items = append(items, deleteItem{
			name:        prompt.ID,
			description: description,
			itemType:    "prompt",
		})
	}

	return items, nil
}

// getActionDeleteItems returns all available actions as delete items
func getActionDeleteItems() ([]deleteItem, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	actionsDir := filepath.Join(home, ".opun", "actions")
	if _, err := os.Stat(actionsDir); os.IsNotExist(err) {
		return []deleteItem{}, nil
	}

	entries, err := os.ReadDir(actionsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read actions directory: %w", err)
	}

	var items []deleteItem
	for _, entry := range entries {
		if entry.IsDir() || (!strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml")) {
			continue
		}

		name := strings.TrimSuffix(strings.TrimSuffix(entry.Name(), ".yaml"), ".yml")
		description := "Action"

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

		items = append(items, deleteItem{
			name:        name,
			description: description,
			itemType:    "action",
		})
	}

	return items, nil
}

// selectItemToDelete lets user select which item to delete
func selectItemToDelete(items []deleteItem, itemType string) (*deleteItem, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("no items to delete")
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
	l.Title = fmt.Sprintf("Select %s to delete", itemType)
	l.SetShowStatusBar(true)
	l.SetFilteringEnabled(true)
	l.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("196")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	model := deleteModel{
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

	if m, ok := result.(deleteModel); ok && m.choice != nil {
		return m.choice, nil
	}

	return nil, fmt.Errorf("no selection made")
}
