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
	"github.com/rizome-dev/opun/internal/plugin"
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

// Step 1: Choose source (Local or Remote)
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
	step         string // "source", "type", "path", "done"
	source       sourceType
	itemType     itemType
	path         string
	url          string
	list         list.Model
	sourceChoice *sourceChoice
	itemChoice   *itemChoice
	err          error
}

func initialInteractiveAddModel() interactiveAddModel {
	// Start with source selection
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

	return interactiveAddModel{
		step: "source",
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

	// Now handle based on source and type
	if model.source == sourceLocal {
		return handleLocalAdd(model.itemType)
	} else {
		return handleRemoteAdd(model.itemType)
	}
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
	baseDir := filepath.Join(home, ".opun")
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
