package core

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
)

// ProviderType represents the type of AI provider
type ProviderType string

const (
	ProviderTypeClaude ProviderType = "claude"
	ProviderTypeGemini ProviderType = "gemini"
	ProviderTypeQwen   ProviderType = "qwen"
	ProviderTypeMock   ProviderType = "mock"
)

// PluginReference represents a reference to a plugin that a provider uses
type PluginReference struct {
	Name    string `json:"name"`
	Type    string `json:"type"` // go, json, script, wasm
	Version string `json:"version"`
}

// ProviderConfig holds provider-specific configuration
type ProviderConfig struct {
	Name        string                 `json:"name"`
	Type        ProviderType           `json:"type"`
	Model       string                 `json:"model"`
	Command     string                 `json:"command"`
	Args        []string               `json:"args"`
	Environment map[string]string      `json:"environment"`
	WorkingDir  string                 `json:"working_dir"`
	Features    ProviderFeatures       `json:"features"`
	Metadata    map[string]interface{} `json:"metadata"`
	Settings    map[string]interface{} `json:"settings"`
}

// ProviderFeatures defines what features a provider supports
type ProviderFeatures struct {
	Interactive      bool `json:"interactive"`
	Batch            bool `json:"batch"`
	Streaming        bool `json:"streaming"`
	FileOutput       bool `json:"file_output"`
	MCP              bool `json:"mcp"`
	Tools            bool `json:"tools"`
	SlashCommands    bool `json:"slash_commands"`
	Plugins          bool `json:"plugins"`
	QualityModes     bool `json:"quality_modes"`
	ContextWindowing bool `json:"context_windowing"`
}

// Provider defines the interface for AI providers
type Provider interface {
	// Basic information
	Name() string
	Type() ProviderType

	// Initialization and validation
	Initialize(config ProviderConfig) error
	Validate() error

	// PTY command generation
	GetPTYCommand() (*exec.Cmd, error)
	GetPTYCommandWithPrompt(prompt string) (*exec.Cmd, error)

	// Feature support
	Features() ProviderFeatures
	SupportsModel(model string) bool

	// Session management
	PrepareSession(ctx context.Context, sessionID string) error
	CleanupSession(ctx context.Context, sessionID string) error

	// Provider-specific behaviors
	GetReadyPattern() string          // Pattern to detect when provider is ready
	GetOutputPattern() string         // Pattern to detect output completion
	GetErrorPattern() string          // Pattern to detect errors
	GetPromptInjectionMethod() string // How to inject prompts (clipboard, file, stdin)

	// Prompt injection
	InjectPrompt(prompt string) error

	// Extended features
	GetMCPServers() []MCPServer
	GetTools() []Tool
	GetSlashCommands() []SharedSlashCommand
	GetPlugins() []PluginReference

	// Slash command support
	SupportsSlashCommands() bool
	GetSlashCommandDirectory() string
	GetSlashCommandFormat() string
	PrepareSlashCommands(commands []SharedSlashCommand, targetDir string) error
}

// ProviderRegistry manages available providers
type ProviderRegistry struct {
	providers map[ProviderType]Provider
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[ProviderType]Provider),
	}
}

// Register registers a new provider
func (r *ProviderRegistry) Register(providerType ProviderType, provider Provider) error {
	r.providers[providerType] = provider
	return nil
}

// Get retrieves a provider by type
func (r *ProviderRegistry) Get(providerType ProviderType) (Provider, bool) {
	provider, exists := r.providers[providerType]
	return provider, exists
}

// List returns all registered provider types
func (r *ProviderRegistry) List() []ProviderType {
	types := make([]ProviderType, 0, len(r.providers))
	for t := range r.providers {
		types = append(types, t)
	}
	return types
}

// BaseProvider provides common functionality for providers
type BaseProvider struct {
	name     string
	typ      ProviderType
	config   ProviderConfig
	features ProviderFeatures
}

// NewBaseProvider creates a new base provider
func NewBaseProvider(name string, typ ProviderType) *BaseProvider {
	return &BaseProvider{
		name: name,
		typ:  typ,
	}
}

// Name returns the provider name
func (b *BaseProvider) Name() string {
	return b.name
}

// Type returns the provider type
func (b *BaseProvider) Type() ProviderType {
	return b.typ
}

// Initialize initializes the provider with config
func (b *BaseProvider) Initialize(config ProviderConfig) error {
	b.config = config
	b.features = config.Features
	return nil
}

// Features returns provider features
func (b *BaseProvider) Features() ProviderFeatures {
	return b.features
}

// Validate validates the provider configuration
func (b *BaseProvider) Validate() error {
	// Basic validation - can be extended by specific providers
	if b.config.Command == "" {
		return ErrInvalidConfig
	}
	return nil
}

// Config returns the provider configuration
func (b *BaseProvider) Config() ProviderConfig {
	return b.config
}

// GetSlashCommands returns slash commands supported by the provider
func (b *BaseProvider) GetSlashCommands() []SharedSlashCommand {
	// Default implementation returns empty list
	// Concrete providers should override this
	return []SharedSlashCommand{}
}

// GetPlugins returns plugins used by the provider
func (b *BaseProvider) GetPlugins() []PluginReference {
	// Default implementation returns empty list
	// Concrete providers should override this
	return []PluginReference{}
}

// SupportsSlashCommands returns whether the provider supports slash commands
func (b *BaseProvider) SupportsSlashCommands() bool {
	return false
}

// GetSlashCommandDirectory returns the directory for slash commands
func (b *BaseProvider) GetSlashCommandDirectory() string {
	return ""
}

// GetSlashCommandFormat returns the format for slash commands
func (b *BaseProvider) GetSlashCommandFormat() string {
	return ""
}

// PrepareSlashCommands prepares slash commands for the provider
func (b *BaseProvider) PrepareSlashCommands(commands []SharedSlashCommand, targetDir string) error {
	return fmt.Errorf("slash commands not supported by this provider")
}
