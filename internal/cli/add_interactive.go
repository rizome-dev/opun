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
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rizome-dev/opun/internal/plugin"
	"github.com/rizome-dev/opun/internal/promptgarden"
	"gopkg.in/yaml.v3"
)

// Source types
type sourceType int

const (
	sourceLocal sourceType = iota
	sourceRemote
)

// Item types
type itemType int

const (
	itemTypeWorkflow itemType = iota
	itemTypePrompt
	itemTypeAction
	itemTypeTool
)

// Step 1: Choose add method (Interactive or From File)
type addMethodChoice struct {
	title       string
	description string
	isInteractive bool
}

func (s addMethodChoice) FilterValue() string { return s.title }
func (s addMethodChoice) Title() string       { return s.title }
func (s addMethodChoice) Description() string { return s.description }

type sourceChoice struct {
	title       string
	description string
	source      sourceType
}

func (s sourceChoice) FilterValue() string { return s.title }
func (s sourceChoice) Title() string       { return s.title }
func (s sourceChoice) Description() string { return s.description }

// Step 2: Choose item type
type itemChoice struct {
	title       string
	description string
	itemType    itemType
}

func (i itemChoice) FilterValue() string { return i.title }
func (i itemChoice) Title() string       { return i.title }
func (i itemChoice) Description() string { return i.description }

// Main interactive add model
type interactiveAddModel struct {
	step         string // "method", "source", "type", "path", "done"
	isInteractive bool
	source       sourceType
	itemType     itemType
	path         string
	url          string
	list         list.Model
	methodChoice *addMethodChoice
	sourceChoice *sourceChoice
	itemChoice   *itemChoice
	err          error
}

func initialInteractiveAddModel() interactiveAddModel {
	// Start with method selection
	methods := []list.Item{
		addMethodChoice{
			title:       "Create interactively",
			description: "Create a new configuration interactively",
			isInteractive: true,
		},
		addMethodChoice{
			title:       "From a file",
			description: "Add from a local or remote file",
			isInteractive: false,
		},
	}

	const defaultWidth = 60
	listHeight := len(methods)*3 + 8

	l := list.New(methods, list.NewDefaultDelegate(), defaultWidth, listHeight)
	l.Title = "How would you like to add a component?"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowPagination(false)
	l.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	return interactiveAddModel{
		step: "method",
		list: l,
	}
}

func (m interactiveAddModel) Init() tea.Cmd {
	return nil
}

func (m interactiveAddModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			switch m.step {
			case "method":
				// Get selected method
				if i, ok := m.list.SelectedItem().(addMethodChoice); ok {
					m.methodChoice = &i
					m.isInteractive = i.isInteractive
					if m.isInteractive {
						m.step = "type"
						return m.createItemList(), nil
					} else {
						m.step = "source"
						return m.createSourceList(), nil
					}
				}
			case "source":
				// Get selected source
				if i, ok := m.list.SelectedItem().(sourceChoice); ok {
					m.sourceChoice = &i
					m.source = i.source
					m.step = "type"
					// Create item type list
					return m.createItemList(), nil
				}
			case "type":
				// Get selected item type
				if i, ok := m.list.SelectedItem().(itemChoice); ok {
					m.itemChoice = &i
					m.itemType = i.itemType
					m.step = "done"
					return m, tea.Quit
				}
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m interactiveAddModel) createSourceList() interactiveAddModel {
	sources := []list.Item{
		sourceChoice{
			title:       "Local",
			description: "Add from files on your local filesystem",
			source:      sourceLocal,
		},
		sourceChoice{
			title:       "Remote",
			description: "Add from a URL (GitHub, web, etc.)",
			source:      sourceRemote,
		},
	}

	const defaultWidth = 60
	listHeight := len(sources)*3 + 8

	l := list.New(sources, list.NewDefaultDelegate(), defaultWidth, listHeight)
	l.Title = "Where is the configuration located?"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowPagination(false)
	l.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	m.list = l
	return m
}

func (m interactiveAddModel) createItemList() interactiveAddModel {
	items := []list.Item{
		itemChoice{
			title:       "Prompt",
			description: "Reusable prompt template for AI agents",
			itemType:    itemTypePrompt,
		},
		itemChoice{
			title:       "Workflow",
			description: "Multi-agent orchestration workflow",
			itemType:    itemTypeWorkflow,
		},
		itemChoice{
			title:       "Action",
			description: "Command that AI agents can execute",
			itemType:    itemTypeAction,
		},
		itemChoice{
			title:       "Tool",
			description: "MCP tool for Opun's MCP server",
			itemType:    itemTypeTool,
		},
	}

	const defaultWidth = 60
	listHeight := len(items)*3 + 8

	l := list.New(items, list.NewDefaultDelegate(), defaultWidth, listHeight)
	l.Title = "What type of configuration?"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowPagination(false)
	l.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	m.list = l
	return m
}

func (m interactiveAddModel) View() string {
	if m.err != nil {
		return fmt.Sprintf("\n❌ Error: %v\n", m.err)
	}
	return "\n" + m.list.View()
}

// RunInteractiveAdd runs the new interactive add flow
func RunInteractiveAdd() error {
	p := tea.NewProgram(initialInteractiveAddModel())
	result, err := p.Run()
	if err != nil {
		return err
	}

	model, ok := result.(interactiveAddModel)
	if !ok || model.step != "done" {
		return fmt.Errorf("cancelled")
	}

	if model.isInteractive {
		return handleInteractiveCreate(model.itemType)
	}

	// Now handle based on source and type
	if model.source == sourceLocal {
		return handleLocalAdd(model.itemType)
	} else {
		return handleRemoteAdd(model.itemType)
	}
}

// handleInteractiveCreate handles the interactive creation of components
func handleInteractiveCreate(itemType itemType) error {
	switch itemType {
	case itemTypeAction:
		return handleInteractiveActionCreate()
	case itemTypeTool:
		return handleInteractiveToolCreate()
	case itemTypePrompt:
		return handleInteractivePromptCreate()
	case itemTypeWorkflow:
		return handleInteractiveWorkflowCreate()
	}
	return nil
}

func handleInteractiveWorkflowCreate() error {
	fmt.Println("Creating a new workflow interactively...")
	
	// Step 1: Basic workflow info
	name, err := Prompt("Enter workflow name:")
	if err != nil {
		return err
	}
	
	description, err := Prompt("Enter workflow description:")
	if err != nil {
		return err
	}
	
	command, err := Prompt("Enter command name (for /command usage):")
	if err != nil {
		return err
	}
	
	// Step 2: Create first agent
	fmt.Println("\nCreating the first agent...")
	agents := []map[string]interface{}{}
	
	agent, err := createInteractiveAgent("agent1")
	if err != nil {
		return err
	}
	agents = append(agents, agent)
	
	// Step 3: Ask for additional agents
	for {
		addMore, err := Confirm("Do you want to add another agent?")
		if err != nil {
			return err
		}
		
		if !addMore {
			break
		}
		
		agentID := fmt.Sprintf("agent%d", len(agents)+1)
		agent, err := createInteractiveAgent(agentID)
		if err != nil {
			return err
		}
		agents = append(agents, agent)
	}
	
	// Step 4: Create workflow structure
	workflow := map[string]interface{}{
		"name":        name,
		"description": description,
		"command":     command,
		"version":     "1.0.0",
		"agents":      agents,
		"settings": map[string]interface{}{
			"stop_on_error": true,
		},
	}
	
	// Step 5: Save workflow
	data, err := yaml.Marshal(workflow)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow to yaml: %w", err)
	}
	
	// Get workflows directory
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	
	workflowsDir := filepath.Join(home, ".opun", "workflows")
	if err := os.MkdirAll(workflowsDir, 0755); err != nil {
		return fmt.Errorf("failed to create workflows directory: %w", err)
	}
	
	// Save workflow
	destPath := filepath.Join(workflowsDir, name+".yaml")
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return fmt.Errorf("failed to save workflow: %w", err)
	}
	
	fmt.Printf("✓ Added workflow '%s'\n", name)
	fmt.Printf("  Saved to: %s\n", destPath)
	fmt.Printf("  Access with: /%s\n", command)
	
	return nil
}

func createInteractiveAgent(agentID string) (map[string]interface{}, error) {
	provider, err := Prompt("Enter agent provider (claude/gemini):")
	if err != nil {
		return nil, err
	}
	
	model, err := Prompt("Enter agent model (e.g., sonnet, opus, flash):")
	if err != nil {
		return nil, err
	}
	
	prompt, err := MultilinePrompt("Enter agent prompt:")
	if err != nil {
		return nil, err
	}
	
	agent := map[string]interface{}{
		"id":       agentID,
		"provider": provider,
		"model":    model,
		"prompt":   prompt,
		"settings": map[string]interface{}{
			"temperature": 0.7,
			"timeout":     300,
		},
	}
	
	return agent, nil
}

func handleInteractivePromptCreate() error {
	name, err := Prompt("Enter a name for the prompt:")
	if err != nil {
		return err
	}

	description, err := Prompt("Enter a description for the prompt:")
	if err != nil {
		return err
	}

	content, err := MultilinePrompt("Enter the prompt content:")
	if err != nil {
		return fmt.Errorf("failed to read prompt content: %w", err)
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

	// Create prompt
	prompt := &promptgarden.Prompt{
		ID:      name,
		Name:    name,
		Content: content,
		Metadata: promptgarden.PromptMetadata{
			Tags:        extractTags(content),
			Category:    "user",
			Version:     "1.0.0",
			Description: description,
		},
	}

	// Save prompt
	if err := garden.SavePrompt(prompt); err != nil {
		return fmt.Errorf("failed to save prompt: %w", err)
	}

	fmt.Printf("✓ Added prompt '%s' to prompt garden\n", name)
	fmt.Printf("  Access with: promptgarden://%s\n", name)

	return nil
}

func handleInteractiveToolCreate() error {
	name, err := Prompt("Enter a name for the tool:")
	if err != nil {
		return err
	}

	description, err := Prompt("Enter a description for the tool:")
	if err != nil {
		return err
	}

	// Create the tool file content
	tool := make(map[string]interface{})
	tool["name"] = name
	tool["description"] = description

	data, err := yaml.Marshal(tool)
	if err != nil {
		return fmt.Errorf("failed to marshal tool to yaml: %w", err)
	}

	// Get tools directory
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	toolsDir := filepath.Join(home, ".opun", "tools")
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		return fmt.Errorf("failed to create tools directory: %w", err)
	}

	// Save tool
	destPath := filepath.Join(toolsDir, name+".yaml")
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return fmt.Errorf("failed to save tool: %w", err)
	}

	fmt.Printf("✓ Added tool '%s'\n", name)
	fmt.Printf("  Saved to: %s\n", destPath)

	return nil
}

func handleInteractiveActionCreate() error {
	name, err := Prompt("Enter a name for the action:")
	if err != nil {
		return err
	}

	description, err := Prompt("Enter a description for the action:")
	if err != nil {
		return err
	}

	command, err := Prompt("Enter the command to execute:")
	if err != nil {
		return err
	}

	args, err := Prompt("Enter the arguments for the command (space-separated):")
	if err != nil {
		return err
	}

	// Create the action file content
	action := make(map[string]interface{})
	action["name"] = name
	action["description"] = description
	action["command"] = command
	action["args"] = strings.Split(args, " ")

	data, err := yaml.Marshal(action)
	if err != nil {
		return fmt.Errorf("failed to marshal action to yaml: %w", err)
	}

	// Get actions directory
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	actionsDir := filepath.Join(home, ".opun", "actions")
	if err := os.MkdirAll(actionsDir, 0755); err != nil {
		return fmt.Errorf("failed to create actions directory: %w", err)
	}

	// Save action
	destPath := filepath.Join(actionsDir, name+".yaml")
	if err := os.WriteFile(destPath, data, 0644); err != nil {
		return fmt.Errorf("failed to save action: %w", err)
	}

	fmt.Printf("✓ Added action '%s'\n", name)
	fmt.Printf("  Saved to: %s\n", destPath)

	return nil
}

// handleLocalAdd handles adding from local filesystem
func handleLocalAdd(itemType itemType) error {
	var prompt, fileExt string

	switch itemType {
	case itemTypePrompt:
		prompt = "Select prompt file (.md, .txt, .yaml):"
		fileExt = "prompt file"
	case itemTypeWorkflow:
		prompt = "Select workflow file (.yaml, .yml):"
		fileExt = "workflow"
	case itemTypeAction:
		prompt = "Select action file (.yaml, .yml):"
		fileExt = "action"
	case itemTypeTool:
		prompt = "Select tool definition file (.yaml, .yml):"
		fileExt = "tool"
	}

	// Use file prompt to get path
	path, err := FilePrompt(prompt)
	if err != nil {
		return err
	}
	if path == "" {
		return fmt.Errorf("no file selected")
	}

	// Check if file exists
	if _, err := os.Stat(path); err != nil {
		return fmt.Errorf("file not found: %s", path)
	}

	// Get name for the item
	defaultName := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	name, err := Prompt(fmt.Sprintf("Enter a name (default: %s):", defaultName))
	if err != nil {
		return err
	}
	if name == "" {
		name = defaultName
	}

	// Execute the add based on type
	fmt.Printf("\n📝 Adding %s...\n", fileExt)

	switch itemType {
	case itemTypePrompt:
		return addPrompt(path, name)
	case itemTypeWorkflow:
		return addWorkflow(path, name)
	case itemTypeAction:
		return addActionFromFile(path, name)
	case itemTypeTool:
		return addTool(path, name)
	}

	return nil
}

// handleRemoteAdd handles adding from remote URL
func handleRemoteAdd(itemType itemType) error {
	// Get URL from user
	url, err := Prompt("Enter the URL:")
	if err != nil {
		return err
	}
	if url == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	// Validate URL
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("invalid URL: must start with http:// or https://")
	}

	fmt.Printf("\n📦 Fetching from %s...\n", url)

	// Use the plugin system to fetch and install
	return installFromURL(url, itemType)
}

// installFromURL uses the remote installer to fetch and install items
func installFromURL(url string, itemType itemType) error {
	// Get home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Initialize remote installer
	baseDir := filepath.Join(home, ".opun", "plugins")
	installer, err := plugin.NewRemoteInstaller(baseDir)
	if err != nil {
		return fmt.Errorf("failed to create installer: %w", err)
	}

	// Use the installer to fetch and install from URL
	if err := installer.InstallFromURL(url); err != nil {
		return fmt.Errorf("failed to install from URL: %w", err)
	}

	fmt.Printf("\n✅ Successfully installed from %s\n", url)

	return nil
}

// addTool adds a tool to the MCP server configuration
func addTool(path, name string) error {
	// Get home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Tools directory
	toolsDir := filepath.Join(home, ".opun", "tools")
	if err := os.MkdirAll(toolsDir, 0755); err != nil {
		return fmt.Errorf("failed to create tools directory: %w", err)
	}

	// Read the tool definition
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read tool file: %w", err)
	}

	// Parse to validate it's valid YAML
	var toolDef map[string]interface{}
	if err := yaml.Unmarshal(data, &toolDef); err != nil {
		return fmt.Errorf("invalid tool definition: %w", err)
	}

	// Ensure required fields
	if toolDef["name"] == nil {
		toolDef["name"] = name
	}
	if toolDef["description"] == nil {
		return fmt.Errorf("tool description is required")
	}

	// Save to tools directory
	destPath := filepath.Join(toolsDir, name+".yaml")

	// Marshal back to YAML with proper formatting
	output, err := yaml.Marshal(toolDef)
	if err != nil {
		return fmt.Errorf("failed to format tool definition: %w", err)
	}

	if err := os.WriteFile(destPath, output, 0644); err != nil {
		return fmt.Errorf("failed to save tool: %w", err)
	}

	fmt.Printf("\n✅ Successfully added tool '%s'\n", name)
	fmt.Printf("   Saved to: %s\n", destPath)
	fmt.Printf("   Tool will be available in Opun's MCP server\n")

	return nil
}


// MultilinePrompt prompts for multi-line input with Enter to add new lines and ctrl+d to finish
func MultilinePrompt(prompt string) (string, error) {
	fmt.Printf("%s\n", prompt)
	fmt.Println("(Press Enter to add new lines, Ctrl+D to finish)")
	
	var lines []string
	reader := bufio.NewReader(os.Stdin)
	
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				// User pressed Ctrl+D, finish input
				break
			}
			return "", err
		}
		
		// Remove the trailing newline for processing
		line = strings.TrimSuffix(line, "\n")
		lines = append(lines, line)
	}
	
	return strings.Join(lines, "\n"), nil
}

