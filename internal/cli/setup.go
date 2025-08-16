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
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rizome-dev/opun/internal/mcp"
	"github.com/rizome-dev/opun/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// SetupCmd creates the setup command
func SetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Configure Opun with your preferred AI provider and MCP servers",
		Long: `Interactive setup to configure Opun with your default AI provider and MCP servers.

This command will:
- Set your default AI provider (Claude or Gemini)
- Install and configure MCP servers
- Set up shell integration
- Configure completion`,
		RunE: runInteractiveSetup,
	}

	return cmd
}

// Provider selection item
type providerItem struct {
	name        string
	description string
	value       string
}

func (i providerItem) FilterValue() string { return i.name }
func (i providerItem) Title() string       { return i.name }
func (i providerItem) Description() string { return i.description }

// MCP server selection item
type mcpItem struct {
	server   mcp.DefaultMCPServer
	selected bool
}

func (i mcpItem) FilterValue() string { return i.server.Name }
func (i mcpItem) Title() string {
	checkbox := "☐"
	if i.selected {
		checkbox = "☑"
	}
	required := ""
	if i.server.Required {
		required = " (required)"
	}
	return fmt.Sprintf("%s %s%s", checkbox, i.server.Name, required)
}
func (i mcpItem) Description() string { return i.server.Description }

// Provider selection model
type providerModel struct {
	list   list.Model
	choice *providerItem
}

func (m providerModel) Init() tea.Cmd {
	return nil
}

func (m providerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			if i, ok := m.list.SelectedItem().(providerItem); ok {
				m.choice = &i
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m providerModel) View() string {
	return m.list.View()
}

// MCP server selection model
type mcpModel struct {
	list    list.Model
	servers []mcp.DefaultMCPServer
	choices []string
}

func (m mcpModel) Init() tea.Cmd {
	return nil
}

func (m mcpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "enter":
			return m, tea.Quit
		case " ":
			// Toggle selection
			if i, ok := m.list.SelectedItem().(mcpItem); ok {
				idx := m.list.Index()
				if contains(m.choices, i.server.Name) {
					m.choices = remove(m.choices, i.server.Name)
				} else {
					m.choices = append(m.choices, i.server.Name)
				}

				// Update the item in the list
				items := m.list.Items()
				if idx < len(items) {
					if item, ok := items[idx].(mcpItem); ok {
						item.selected = !item.selected
						items[idx] = item
						m.list.SetItems(items)
					}
				}
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m mcpModel) View() string {
	selectedCount := len(m.choices)
	totalCount := len(m.servers)

	statusStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true)

	instructionsStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	status := statusStyle.Render(fmt.Sprintf("Selected: %d/%d servers", selectedCount, totalCount))
	instructions := instructionsStyle.Render("Space to toggle selection • ↑↓ to navigate • Enter to continue")

	return m.list.View() + "\n\n" + status + "\n" + instructions
}

func runInteractiveSetup(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 Welcome to Opun Setup!")
	fmt.Println("This interactive setup will configure Opun for your system.")
	fmt.Println()

	// 1. Provider Selection
	fmt.Println("📋 Step 1: Choose your default AI provider")
	provider, err := selectProvider()
	if err != nil {
		return err
	}
	fmt.Printf("✅ Selected provider: %s\n\n", provider)

	// 2. MCP Server Selection
	fmt.Println("🔧 Step 2: Select MCP servers to install")
	selectedServers, err := selectMCPServers()
	if err != nil {
		return err
	}
	fmt.Printf("✅ Selected %d MCP servers\n\n", len(selectedServers))

	// 3. Install MCP servers using shared configuration
	if len(selectedServers) > 0 {
		fmt.Println("📦 Step 3: Installing MCP servers...")
		if err := installMCPServersShared(selectedServers, provider); err != nil {
			fmt.Printf("⚠️  Some MCP servers failed to install: %v\n", err)
		} else {
			fmt.Println("✅ MCP servers installed and configured successfully")
		}
	}

	// 4. Save configuration
	fmt.Println("💾 Step 4: Saving configuration...")
	if err := saveSetupConfig(provider, selectedServers); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}
	fmt.Println("✅ Configuration saved")

	// 5. Setup completion
	fmt.Println("\n🎉 Setup completed successfully!")
	fmt.Println("📝 Next steps:")
	fmt.Println("  • Run 'opun chat' to start chatting with your default provider")
	fmt.Println("  • Run 'opun add' to add workflows and prompts")

	return nil
}

func selectProvider() (string, error) {
	items := []list.Item{
		providerItem{
			name:        "Claude",
			description: "Anthropic's Claude - excellent for coding and reasoning",
			value:       "claude",
		},
		providerItem{
			name:        "Gemini",
			description: "Google's Gemini - powerful multimodal AI",
			value:       "gemini",
		},
		providerItem{
			name:        "Qwen",
			description: "Qwen Code - optimized for coding tasks",
			value:       "qwen",
		},
	}

	// Calculate height to show all items with sufficient space
	listHeight := len(items)*3 + 10 // generous spacing for items + title + padding
	l := list.New(items, list.NewDefaultDelegate(), 70, listHeight)
	l.Title = "Select your default AI provider"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowPagination(false) // Show all items at once
	l.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	model := providerModel{list: l}

	p := tea.NewProgram(model)
	result, err := p.Run()
	if err != nil {
		return "", err
	}

	if m, ok := result.(providerModel); ok && m.choice != nil {
		return m.choice.value, nil
	}

	return "claude", nil // Default fallback
}

func selectMCPServers() ([]string, error) {
	servers := mcp.GetDefaultMCPServers()
	items := make([]list.Item, len(servers))

	for i, server := range servers {
		items[i] = mcpItem{
			server:   server,
			selected: true, // Pre-select all servers by default
		}
	}

	// Calculate height to show all items with sufficient space
	listHeight := len(servers)*3 + 12 // generous spacing for servers + title + padding + status
	l := list.New(items, list.NewDefaultDelegate(), 85, listHeight)
	l.Title = "Select MCP servers to install"
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowPagination(false) // Show all items at once
	l.Styles.Title = lipgloss.NewStyle().
		Background(lipgloss.Color("205")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1)

	// Pre-select all servers by default
	var preSelected []string
	for _, server := range servers {
		preSelected = append(preSelected, server.Name)
	}

	model := mcpModel{
		list:    l,
		servers: servers,
		choices: preSelected,
	}

	p := tea.NewProgram(model)
	result, err := p.Run()
	if err != nil {
		return nil, err
	}

	if m, ok := result.(mcpModel); ok {
		return m.choices, nil
	}

	return preSelected, nil
}

func installMCPServersShared(serverNames []string, provider string) error {
	if len(serverNames) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Use the new shared installer
	installer, err := mcp.NewSharedMCPInstaller()
	if err != nil {
		return fmt.Errorf("failed to create MCP installer: %w", err)
	}

	// Install servers (checks for existing installations)
	if err := installer.InstallServers(ctx, serverNames); err != nil {
		return fmt.Errorf("failed to install servers: %w", err)
	}

	// Sync configuration to all providers
	providers := []string{provider}
	// Also sync to the other providers if user might use them
	allProviders := []string{"claude", "gemini", "qwen"}
	for _, p := range allProviders {
		if p != provider {
			providers = append(providers, p)
		}
	}

	if err := installer.SyncConfigurations(providers); err != nil {
		return fmt.Errorf("failed to sync configurations: %w", err)
	}

	// Check for missing environment variables
	missing := installer.ValidateEnvironmentVariables()
	if len(missing) > 0 {
		fmt.Println("\n⚠️  Some MCP servers require environment variables:")
		for server, vars := range missing {
			fmt.Printf("  %s: %s\n", server, strings.Join(vars, ", "))
		}
		fmt.Println("  Set these in your shell configuration or provider config")
	}

	return nil
}

func saveSetupConfig(provider string, mcpServers []string) error {
	// Set up viper config
	viper.Set("default_provider", provider)
	viper.Set("mcp_servers", mcpServers)

	// Ensure config directory exists
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".opun")
	if err := utils.EnsureDir(configDir); err != nil {
		return err
	}

	// Write config file
	configPath := filepath.Join(configDir, "config.yaml")
	if err := viper.WriteConfigAs(configPath); err != nil {
		return err
	}

	return nil
}

// Utility functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func remove(slice []string, item string) []string {
	var result []string
	for _, s := range slice {
		if s != item {
			result = append(result, s)
		}
	}
	return result
}
