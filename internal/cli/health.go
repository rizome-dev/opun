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

	"github.com/charmbracelet/lipgloss"
	"github.com/rizome-dev/opun/internal/config"
	"github.com/rizome-dev/opun/internal/promptgarden"
	"github.com/rizome-dev/opun/internal/tools"
	"github.com/rizome-dev/opun/internal/workflow"
)

// ServiceStatus represents the health status of a service
type ServiceStatus struct {
	Name   string
	Status bool
	Count  int // -1 for non-countable items
}

// HealthCheck performs a comprehensive health check of all Opun services
type HealthCheck struct {
	provider         string
	injectionManager *config.InjectionManager
	sharedManager    *config.SharedConfigManager
	homeDir          string
}

// NewHealthCheck creates a new health check instance
func NewHealthCheck(provider string, injectionManager *config.InjectionManager) (*HealthCheck, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	sharedManager, err := config.NewSharedConfigManager()
	if err != nil {
		// Non-fatal, some services might still work
		sharedManager = nil
	}

	return &HealthCheck{
		provider:         provider,
		injectionManager: injectionManager,
		sharedManager:    sharedManager,
		homeDir:          homeDir,
	}, nil
}

// CheckAllServices performs health checks on all services
func (h *HealthCheck) CheckAllServices() []ServiceStatus {
	var services []ServiceStatus

	// Core services that apply to all providers
	services = append(services, h.checkConfigSync())
	services = append(services, h.checkMCPServer())
	services = append(services, h.checkMemoryManagement())

	// Provider-specific services
	if h.provider == "claude" {
		services = append(services, h.checkPromptGardenClaude())
		services = append(services, h.checkSlashCommandsClaude())
	} else if h.provider == "gemini" {
		services = append(services, h.checkPromptGardenGemini())
		services = append(services, h.checkSlashCommandsGemini())
	}

	// Additional services
	services = append(services, h.checkWorkflows())
	services = append(services, h.checkActions())
	services = append(services, h.checkTools())

	return services
}

// checkConfigSync checks if configuration synchronization is working
func (h *HealthCheck) checkConfigSync() ServiceStatus {
	return ServiceStatus{
		Name:   "Config Sync",
		Status: h.injectionManager != nil && h.sharedManager != nil,
		Count:  -1,
	}
}

// checkMCPServer checks MCP server status
func (h *HealthCheck) checkMCPServer() ServiceStatus {
	status := ServiceStatus{
		Name:  "MCP Server",
		Count: -1,
	}

	// Check if unified MCP server is configured
	mcpServerPath := filepath.Join(h.homeDir, ".opun", "mcp", "opun-server.js")
	if _, err := os.Stat(mcpServerPath); err == nil {
		status.Status = true
		return status
	}

	// Check if any MCP servers are installed
	if h.sharedManager != nil {
		servers := h.sharedManager.GetMCPServers()
		installed := 0
		for _, server := range servers {
			if server.Installed {
				installed++
			}
		}
		status.Status = installed > 0
	}

	return status
}

// checkPromptGardenClaude checks PromptGarden for Claude (via slash commands)
func (h *HealthCheck) checkPromptGardenClaude() ServiceStatus {
	status := ServiceStatus{
		Name: "PromptGarden",
	}

	gardenPath := filepath.Join(h.homeDir, ".opun", "promptgarden")
	if garden, err := promptgarden.NewGarden(gardenPath); err == nil {
		prompts, _ := garden.ListPrompts()
		status.Status = true
		status.Count = len(prompts)
	} else {
		status.Status = false
		status.Count = 0
	}

	return status
}

// checkPromptGardenGemini checks PromptGarden for Gemini (via MCP)
func (h *HealthCheck) checkPromptGardenGemini() ServiceStatus {
	status := ServiceStatus{
		Name: "PromptGarden (MCP)",
	}

	// For Gemini, PromptGarden is accessed through MCP
	if mcpStatus := h.checkMCPServer(); mcpStatus.Status {
		gardenPath := filepath.Join(h.homeDir, ".opun", "promptgarden")
		if garden, err := promptgarden.NewGarden(gardenPath); err == nil {
			prompts, _ := garden.ListPrompts()
			status.Status = true
			status.Count = len(prompts)
		} else {
			status.Status = false
			status.Count = 0
		}
	} else {
		status.Status = false
		status.Count = 0
	}

	return status
}

// checkSlashCommandsClaude checks slash commands for Claude
func (h *HealthCheck) checkSlashCommandsClaude() ServiceStatus {
	status := ServiceStatus{
		Name: "Slash Commands",
	}

	if h.sharedManager != nil {
		commands := h.sharedManager.GetSlashCommands()
		status.Status = true
		status.Count = len(commands)
	} else {
		status.Status = false
		status.Count = 0
	}

	return status
}

// checkSlashCommandsGemini checks slash commands for Gemini
func (h *HealthCheck) checkSlashCommandsGemini() ServiceStatus {
	status := ServiceStatus{
		Name: "Slash Commands (MCP)",
	}

	// Gemini accesses slash commands through MCP
	if mcpStatus := h.checkMCPServer(); mcpStatus.Status {
		if h.sharedManager != nil {
			commands := h.sharedManager.GetSlashCommands()
			status.Status = true
			status.Count = len(commands)
		} else {
			status.Status = true
			status.Count = 0
		}
	} else {
		status.Status = false
		status.Count = 0
	}

	return status
}

// checkWorkflows checks workflow system
func (h *HealthCheck) checkWorkflows() ServiceStatus {
	status := ServiceStatus{
		Name: "Workflows",
	}

	workflowDir := filepath.Join(h.homeDir, ".opun", "workflows")
	if manager, err := workflow.NewManager(workflowDir); err == nil {
		workflows, _ := manager.ListWorkflows()
		status.Status = true
		status.Count = len(workflows)
	} else {
		status.Status = true
		status.Count = 0
	}

	return status
}

// checkActions checks actions system
func (h *HealthCheck) checkActions() ServiceStatus {
	status := ServiceStatus{
		Name: "Actions",
	}

	actionsDir := filepath.Join(h.homeDir, ".opun", "actions")
	loader := tools.NewLoader(actionsDir)
	if err := loader.LoadAll(); err == nil {
		actionList := loader.GetRegistry().List("")
		status.Status = true
		status.Count = len(actionList)
	} else {
		// Actions directory might not exist yet
		status.Status = true
		status.Count = 0
	}

	return status
}

// checkTools checks tools system
func (h *HealthCheck) checkTools() ServiceStatus {
	status := ServiceStatus{
		Name: "Tools",
	}

	// Check tools directory
	toolsDir := filepath.Join(h.homeDir, ".opun", "tools")
	if entries, err := os.ReadDir(toolsDir); err == nil {
		count := 0
		for _, entry := range entries {
			if !entry.IsDir() && (filepath.Ext(entry.Name()) == ".yaml" || filepath.Ext(entry.Name()) == ".yml") {
				count++
			}
		}
		status.Status = true
		status.Count = count
	} else {
		// Tools directory might not exist yet
		status.Status = true
		status.Count = 0
	}

	return status
}

// checkMemoryManagement checks memory management
func (h *HealthCheck) checkMemoryManagement() ServiceStatus {
	sessionDir := filepath.Join(h.homeDir, ".opun", "sessions")
	info, err := os.Stat(sessionDir)

	return ServiceStatus{
		Name:   "Memory Management",
		Status: err == nil && info.IsDir(),
		Count:  -1,
	}
}

// DisplayHealthCheck displays the health check results with visual indicators
func DisplayHealthCheck(provider string, services []ServiceStatus) {
	// Define styles
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("86")).
		MarginTop(1).
		MarginBottom(1)

	serviceStyle := lipgloss.NewStyle().
		PaddingLeft(2)

	successStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("42"))

	failureStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("161"))

	countStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("244"))

	// Capitalize provider name
	providerName := provider
	if len(provider) > 0 {
		providerName = string(provider[0]&^32) + provider[1:]
	}

	// Display title
	fmt.Println(titleStyle.Render(fmt.Sprintf("%s Health Check", providerName)))

	// Display services
	for _, service := range services {
		var output string

		// Choose indicator based on status
		if service.Status {
			output = successStyle.Render("●") + " "
		} else {
			output = failureStyle.Render("●") + " "
		}

		// Add service name
		output += service.Name

		// Add count if applicable
		if service.Count >= 0 {
			countText := fmt.Sprintf(" (%d)", service.Count)
			output += countStyle.Render(countText)
		}

		fmt.Println(serviceStyle.Render(output))
	}

	fmt.Println()
}
