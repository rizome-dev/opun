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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rizome-dev/opun/internal/subagent"
	"github.com/rizome-dev/opun/pkg/core"
	subagentpkg "github.com/rizome-dev/opun/pkg/subagent"
)

// ProviderFactory creates providers
type ProviderFactory struct {
	subAgentManager *subagentpkg.Manager
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{
		subAgentManager: subagentpkg.NewManager(),
	}
}

// GetSubAgentManager returns the subagent manager
func (f *ProviderFactory) GetSubAgentManager() *subagentpkg.Manager {
	return f.subAgentManager
}

// InitializeSubAgents initializes subagents from configuration
func (f *ProviderFactory) InitializeSubAgents() error {
	// Load subagent configurations
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".opun", "subagents")
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		// Create directory if it doesn't exist
		if err := os.MkdirAll(configDir, 0755); err != nil {
			return err
		}
	}

	// Register default provider adapters with the manager
	// These will be created using the subagent factory
	factory := subagent.NewFactory()
	
	// Claude adapter
	claudeConfig := core.SubAgentConfig{
		Name:        "claude-default",
		Description: "Default Claude subagent",
		Provider:    core.ProviderTypeClaude,
		Model:       "claude-3-sonnet",
		Strategy:    core.DelegationAutomatic,
		Settings:    make(map[string]interface{}),
	}
	if claudeAdapter, err := factory.CreateSubAgent(claudeConfig); err == nil {
		f.subAgentManager.Register(claudeAdapter)
	}

	// Gemini adapter
	geminiConfig := core.SubAgentConfig{
		Name:        "gemini-default",
		Description: "Default Gemini subagent",
		Provider:    core.ProviderTypeGemini,
		Model:       "gemini-pro",
		Strategy:    core.DelegationAutomatic,
		Settings:    make(map[string]interface{}),
	}
	if geminiAdapter, err := factory.CreateSubAgent(geminiConfig); err == nil {
		f.subAgentManager.Register(geminiAdapter)
	}

	// Qwen adapter
	qwenConfig := core.SubAgentConfig{
		Name:        "qwen-default",
		Description: "Default Qwen subagent",
		Provider:    core.ProviderTypeQwen,
		Model:       "qwen-coder",
		Strategy:    core.DelegationAutomatic,
		Settings:    make(map[string]interface{}),
	}
	if qwenAdapter, err := factory.CreateSubAgent(qwenConfig); err == nil {
		f.subAgentManager.Register(qwenAdapter)
	}

	return nil
}

// CreateProvider creates a provider instance with the given configuration
func (f *ProviderFactory) CreateProvider(config core.ProviderConfig) (core.Provider, error) {
	var provider core.Provider

	// Create base provider based on type
	switch config.Type {
	case core.ProviderTypeClaude:
		provider = NewClaudeProvider(config)
	case core.ProviderTypeGemini:
		provider = NewGeminiProvider(config)
	case core.ProviderTypeQwen:
		provider = NewQwenProvider(config)
	case core.ProviderTypeMock:
		provider = NewMockProvider(config.Name, config)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", config.Type)
	}

	// Validate the provider
	if err := provider.Validate(); err != nil {
		return nil, fmt.Errorf("provider validation failed: %w", err)
	}

	return provider, nil
}

// CreateProviderFromType creates a provider with default configuration
func (f *ProviderFactory) CreateProviderFromType(providerType string, name string) (core.Provider, error) {
	// Convert string to ProviderType
	var pType core.ProviderType
	switch strings.ToLower(providerType) {
	case "claude":
		pType = core.ProviderTypeClaude
	case "gemini":
		pType = core.ProviderTypeGemini
	case "qwen":
		pType = core.ProviderTypeQwen
	case "mock":
		pType = core.ProviderTypeMock
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", providerType)
	}

	config := core.ProviderConfig{
		Name: name,
		Type: pType,
		Settings: map[string]interface{}{
			"model": getDefaultModel(providerType),
		},
		Environment: make(map[string]string),
		Features:    getDefaultFeatures(providerType),
	}

	// Set provider-specific defaults
	switch strings.ToLower(providerType) {
	case "claude":
		config.Command = "claude"
		config.Args = []string{}
	case "gemini":
		config.Command = "gemini"
		config.Args = []string{"chat"}
	case "qwen":
		config.Command = "qwen"
		config.Args = []string{"chat"}
	}

	return f.CreateProvider(config)
}

// getDefaultModel returns the default model for a provider type
func getDefaultModel(providerType string) string {
	switch strings.ToLower(providerType) {
	case "claude":
		return "sonnet"
	case "gemini":
		return "gemini-pro"
	case "qwen":
		return "code"
	default:
		return ""
	}
}

// getDefaultFeatures returns the default features for a provider type
func getDefaultFeatures(providerType string) core.ProviderFeatures {
	switch strings.ToLower(providerType) {
	case "claude":
		return core.ProviderFeatures{
			Interactive:      true,
			Batch:            false,
			Streaming:        true,
			FileOutput:       false,
			MCP:              true,
			Tools:            true,
			SlashCommands:    true,
			Plugins:          true,
			QualityModes:     true,
			ContextWindowing: true,
		}
	case "gemini":
		return core.ProviderFeatures{
			Interactive:      true,
			Batch:            false,
			Streaming:        true,
			FileOutput:       false,
			MCP:              true,
			Tools:            true,
			SlashCommands:    true,
			Plugins:          true,
			QualityModes:     false,
			ContextWindowing: true,
		}
	case "qwen":
		return core.ProviderFeatures{
			Interactive:      true,
			Batch:            false,
			Streaming:        true,
			FileOutput:       false,
			MCP:              true,
			Tools:            true,
			SlashCommands:    true,
			Plugins:          true,
			QualityModes:     false,
			ContextWindowing: true,
		}
	default:
		return core.ProviderFeatures{}
	}
}

// DefaultProviderFactory is the default factory
var DefaultProviderFactory = NewProviderFactory()
