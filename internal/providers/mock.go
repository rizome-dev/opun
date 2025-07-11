package providers

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
	"os/exec"

	"github.com/rizome-dev/opun/internal/config"
	"github.com/rizome-dev/opun/pkg/core"
)

// MockProvider is a mock provider for testing
type MockProvider struct {
	name             string
	config           core.ProviderConfig
	injectionManager *config.InjectionManager
	environment      *config.ProviderEnvironment
}

// NewMockProvider creates a new mock provider
func NewMockProvider(name string, providerConfig core.ProviderConfig) *MockProvider {
	// Create injection manager (optional)
	injectionManager, _ := config.NewInjectionManager(nil)

	return &MockProvider{
		name:             name,
		config:           providerConfig,
		injectionManager: injectionManager,
	}
}

// Name returns the provider name
func (p *MockProvider) Name() string {
	return p.name
}

// Type returns the provider type
func (p *MockProvider) Type() core.ProviderType {
	return core.ProviderTypeMock
}

// Initialize initializes the provider
func (p *MockProvider) Initialize(config core.ProviderConfig) error {
	p.config = config
	return nil
}

// Validate validates the provider configuration
func (p *MockProvider) Validate() error {
	return nil
}

// GetPTYCommand returns a mock command
func (p *MockProvider) GetPTYCommand() (*exec.Cmd, error) {
	// Use echo as a simple mock command
	return exec.Command("echo", "Mock provider ready"), nil
}

// GetPTYCommandWithPrompt returns a mock command with prompt
func (p *MockProvider) GetPTYCommandWithPrompt(prompt string) (*exec.Cmd, error) {
	// Echo the prompt as output
	return exec.Command("echo", fmt.Sprintf("Mock response to: %s", prompt)), nil
}

// PrepareSession prepares a mock session
func (p *MockProvider) PrepareSession(ctx context.Context, sessionID string) error {
	// Mock implementation - nothing to prepare
	return nil
}

// CleanupSession cleans up a mock session
func (p *MockProvider) CleanupSession(ctx context.Context, sessionID string) error {
	// Mock implementation - nothing to cleanup
	return nil
}

// GetReadyPattern returns the pattern that indicates the provider is ready
func (p *MockProvider) GetReadyPattern() string {
	return "Mock provider ready"
}

// GetOutputPattern returns the pattern for provider output
func (p *MockProvider) GetOutputPattern() string {
	return "Mock response"
}

// GetErrorPattern returns the pattern for provider errors
func (p *MockProvider) GetErrorPattern() string {
	return "Mock error"
}

// GetPromptInjectionMethod returns the method for injecting prompts
func (p *MockProvider) GetPromptInjectionMethod() string {
	return "direct"
}

// Features returns provider features
func (p *MockProvider) Features() core.ProviderFeatures {
	return core.ProviderFeatures{
		Interactive:      true,
		Batch:            false,
		Streaming:        false,
		FileOutput:       false,
		MCP:              false,
		Tools:            false,
		QualityModes:     false,
		ContextWindowing: false,
	}
}

// SupportsModel returns whether the provider supports a model
func (p *MockProvider) SupportsModel(model string) bool {
	models := []string{"test", "mock"}
	for _, m := range models {
		if m == model {
			return true
		}
	}
	return false
}

// InjectPrompt injects a prompt (mock implementation)
func (p *MockProvider) InjectPrompt(prompt string) error {
	// Mock implementation - just log it
	return nil
}

// GetMCPServers returns MCP servers (none for mock)
func (p *MockProvider) GetMCPServers() []core.MCPServer {
	return []core.MCPServer{}
}

// GetTools returns available tools (none for mock)
func (p *MockProvider) GetTools() []core.Tool {
	return []core.Tool{}
}

// GetSlashCommands returns slash commands (none for mock)
func (p *MockProvider) GetSlashCommands() []core.SharedSlashCommand {
	return []core.SharedSlashCommand{}
}

// GetPlugins returns plugins (none for mock)
func (p *MockProvider) GetPlugins() []core.PluginReference {
	return []core.PluginReference{}
}

// SupportsSlashCommands returns false for mock
func (p *MockProvider) SupportsSlashCommands() bool {
	return false
}

// GetSlashCommandDirectory returns empty for mock
func (p *MockProvider) GetSlashCommandDirectory() string {
	return ""
}

// GetSlashCommandFormat returns empty for mock
func (p *MockProvider) GetSlashCommandFormat() string {
	return ""
}

// PrepareSlashCommands does nothing for mock
func (p *MockProvider) PrepareSlashCommands(commands []core.SharedSlashCommand, targetDir string) error {
	return nil
}
