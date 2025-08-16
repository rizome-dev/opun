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
	"strings"

	"github.com/rizome-dev/opun/pkg/core"
)

// ProviderFactory creates providers
type ProviderFactory struct {
}

// NewProviderFactory creates a new provider factory
func NewProviderFactory() *ProviderFactory {
	return &ProviderFactory{}
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
