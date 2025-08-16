package subagent

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

	"github.com/rizome-dev/opun/internal/subagent/claude"
	"github.com/rizome-dev/opun/internal/subagent/gemini"
	"github.com/rizome-dev/opun/internal/subagent/qwen"
	"github.com/rizome-dev/opun/pkg/core"
)

// Factory creates provider-specific subagent adapters
type Factory struct {
	providers map[core.ProviderType]core.Provider
}

// NewFactory creates a new subagent factory
func NewFactory() *Factory {
	return &Factory{
		providers: make(map[core.ProviderType]core.Provider),
	}
}

// RegisterProvider registers a provider with the factory
func (f *Factory) RegisterProvider(provider core.Provider) error {
	if provider == nil {
		return fmt.Errorf("provider cannot be nil")
	}
	
	f.providers[provider.Type()] = provider
	return nil
}

// CreateAdapter creates a provider-specific subagent adapter
func (f *Factory) CreateAdapter(config core.SubAgentConfig) (core.SubAgentAdapter, error) {
	if config.Provider == "" {
		return nil, fmt.Errorf("provider type not specified in config")
	}
	
	var adapter core.SubAgentAdapter
	
	switch config.Provider {
	case core.ProviderTypeClaude:
		adapter = claude.NewClaudeAdapter(config)
		
	case core.ProviderTypeGemini:
		adapter = gemini.NewGeminiAdapter(config)
		
	case core.ProviderTypeQwen:
		adapter = qwen.NewQwenAdapter(config)
		
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", config.Provider)
	}
	
	// Initialize the adapter
	if err := adapter.Initialize(config); err != nil {
		return nil, fmt.Errorf("failed to initialize adapter: %w", err)
	}
	
	// Associate provider if available
	if provider, exists := f.providers[config.Provider]; exists {
		if err := adapter.InitializeProvider(provider); err != nil {
			return nil, fmt.Errorf("failed to initialize provider: %w", err)
		}
	}
	
	// Validate the adapter
	if err := adapter.Validate(); err != nil {
		return nil, fmt.Errorf("adapter validation failed: %w", err)
	}
	
	return adapter, nil
}

// CreateSubAgent creates a complete subagent with the appropriate adapter
func (f *Factory) CreateSubAgent(config core.SubAgentConfig) (core.SubAgent, error) {
	adapter, err := f.CreateAdapter(config)
	if err != nil {
		return nil, err
	}
	
	// The adapter implements the SubAgent interface
	return adapter, nil
}

// ListSupportedProviders returns a list of supported provider types
func (f *Factory) ListSupportedProviders() []core.ProviderType {
	return []core.ProviderType{
		core.ProviderTypeClaude,
		core.ProviderTypeGemini,
		core.ProviderTypeQwen,
	}
}

// GetProviderCapabilities returns the subagent capabilities of a provider
func (f *Factory) GetProviderCapabilities(providerType core.ProviderType) (core.SubAgentType, error) {
	switch providerType {
	case core.ProviderTypeClaude:
		return core.SubAgentTypeDeclarative, nil
		
	case core.ProviderTypeGemini:
		return core.SubAgentTypeProgrammatic, nil
		
	case core.ProviderTypeQwen:
		return core.SubAgentTypeWorkflow, nil // Custom implementation via workflows
		
	default:
		return "", fmt.Errorf("unknown provider type: %s", providerType)
	}
}

// CreateFromProvider creates subagents from a provider's native configuration
func (f *Factory) CreateFromProvider(provider core.Provider) ([]core.SubAgent, error) {
	if provider == nil {
		return nil, fmt.Errorf("provider cannot be nil")
	}
	
	// Check if provider supports subagents
	capable, ok := provider.(core.SubAgentCapable)
	if !ok {
		return nil, fmt.Errorf("provider %s does not support subagents", provider.Name())
	}
	
	if !capable.SupportsSubAgents() {
		return nil, fmt.Errorf("provider %s has subagents disabled", provider.Name())
	}
	
	// Get subagent configurations from provider
	configs, err := capable.ListSubAgents()
	if err != nil {
		return nil, fmt.Errorf("failed to list subagents: %w", err)
	}
	
	// Create subagents from configurations
	var agents []core.SubAgent
	for _, config := range configs {
		// Ensure provider type is set
		if config.Provider == "" {
			config.Provider = provider.Type()
		}
		
		agent, err := f.CreateSubAgent(config)
		if err != nil {
			// Log error but continue with other agents
			continue
		}
		
		agents = append(agents, agent)
	}
	
	return agents, nil
}

// CreateDefaultConfigs creates default subagent configurations for each provider
func (f *Factory) CreateDefaultConfigs() []core.SubAgentConfig {
	return []core.SubAgentConfig{
		// Claude: Declarative research agent
		{
			Name:        "claude-researcher",
			Type:        core.SubAgentTypeDeclarative,
			Description: "Research and documentation agent using Claude's Task tool",
			Provider:    core.ProviderTypeClaude,
			Model:       "claude-3-opus",
			Strategy:    core.DelegationAutomatic,
			Context:     []string{"research", "documentation", "analysis", "report"},
			Capabilities: []string{"research", "writing", "analysis", "summarization"},
			Priority:    8,
			Timeout:     300,
			OutputFormat: "markdown",
		},
		
		// Gemini: Programmatic code generator
		{
			Name:        "gemini-coder",
			Type:        core.SubAgentTypeProgrammatic,
			Description: "Code generation and implementation agent using Gemini",
			Provider:    core.ProviderTypeGemini,
			Model:       "gemini-1.5-flash",
			Strategy:    core.DelegationAutomatic,
			Context:     []string{"code", "implementation", "programming", "development"},
			Capabilities: []string{"code_generation", "implementation", "testing", "debugging"},
			Priority:    7,
			Timeout:     180,
			Interactive: false,
			OutputFormat: "json",
			ProviderConfig: map[string]interface{}{
				"temperature":    0.3,
				"max_tokens":     4096,
				"max_iterations": 3,
			},
		},
		
		// Qwen: Specialized code review agent
		{
			Name:        "qwen-reviewer",
			Type:        core.SubAgentTypeWorkflow,
			Description: "Code review and refactoring specialist using Qwen",
			Provider:    core.ProviderTypeQwen,
			Model:       "qwen-coder-32b-instruct",
			Strategy:    core.DelegationExplicit,
			Context:     []string{"review", "refactor", "optimize", "quality"},
			Capabilities: []string{"code_review", "refactoring", "optimization", "best_practices"},
			Priority:    9,
			Timeout:     240,
			Interactive: false,
			OutputFormat: "text",
			Tools:       []string{"linter", "formatter", "analyzer"},
		},
	}
}