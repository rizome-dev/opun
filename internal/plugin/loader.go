package plugin

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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rizome-dev/opun/pkg/plugin"
	"gopkg.in/yaml.v3"
)

// Loader handles loading plugins from various sources
type Loader struct {
	pluginDir string
}

// NewLoader creates a new plugin loader
func NewLoader(pluginDir string) *Loader {
	return &Loader{
		pluginDir: pluginDir,
	}
}

// LoadManifest loads a plugin manifest from a file
func (l *Loader) LoadManifest(path string) (*plugin.PluginManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest plugin.PluginManifest

	// Try to parse as YAML first, then JSON
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		if err := json.Unmarshal(data, &manifest); err != nil {
			return nil, fmt.Errorf("failed to parse manifest: %w", err)
		}
	}

	// Validate manifest
	if err := l.validateManifest(&manifest); err != nil {
		return nil, fmt.Errorf("invalid manifest: %w", err)
	}

	return &manifest, nil
}

// LoadPlugin loads a plugin based on its manifest
func (l *Loader) LoadPlugin(manifest *plugin.PluginManifest) (plugin.Plugin, error) {
	// For now, all plugins are import-based plugins
	return l.loadJSONPlugin(manifest)
}

// loadJSONPlugin loads a JSON-defined plugin
func (l *Loader) loadJSONPlugin(manifest *plugin.PluginManifest) (plugin.Plugin, error) {
	// JSON plugins are simpler - they define workflows and commands declaratively
	return &JSONPlugin{
		manifest: manifest,
		loader:   l,
	}, nil
}

// validateManifest validates a plugin manifest
func (l *Loader) validateManifest(manifest *plugin.PluginManifest) error {
	if manifest.Name == "" {
		return fmt.Errorf("plugin name is required")
	}

	if manifest.Description == "" {
		return fmt.Errorf("plugin description is required")
	}

	return nil
}

// DiscoverPlugins discovers all plugins in the plugin directory
func (l *Loader) DiscoverPlugins() ([]*plugin.PluginManifest, error) {
	var manifests []*plugin.PluginManifest

	// Create plugin directory if it doesn't exist
	if err := os.MkdirAll(l.pluginDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create plugin directory: %w", err)
	}

	// Walk the plugin directory
	err := filepath.Walk(l.pluginDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Look for manifest files
		if info.Name() == "plugin.yaml" || info.Name() == "plugin.yml" || info.Name() == "plugin.json" {
			manifest, err := l.LoadManifest(path)
			if err != nil {
				// Log error but continue
				fmt.Printf("Warning: failed to load manifest %s: %v\n", path, err)
				return nil
			}

			manifests = append(manifests, manifest)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to discover plugins: %w", err)
	}

	return manifests, nil
}

// JSONPlugin implements the Plugin interface for JSON-defined plugins
type JSONPlugin struct {
	manifest *plugin.PluginManifest
	loader   *Loader
	config   plugin.PluginConfig
}

func (p *JSONPlugin) Name() string {
	return p.manifest.Name
}

func (p *JSONPlugin) Version() string {
	// Version no longer in manifest
	return "1.0.0"
}

func (p *JSONPlugin) Description() string {
	return p.manifest.Description
}

func (p *JSONPlugin) Author() string {
	return p.manifest.Author
}

func (p *JSONPlugin) Initialize(config plugin.PluginConfig) error {
	p.config = config
	return nil
}

func (p *JSONPlugin) Start(ctx context.Context) error {
	// JSON plugins don't need to start anything
	return nil
}

func (p *JSONPlugin) Stop(ctx context.Context) error {
	// JSON plugins don't need to stop anything
	return nil
}

func (p *JSONPlugin) Execute(ctx context.Context, input plugin.PluginInput) (plugin.PluginOutput, error) {
	// For JSON plugins, execution means running a predefined workflow
	// This would integrate with the workflow system
	return plugin.PluginOutput{
		Success: true,
		Result:  "JSON plugin execution not yet implemented",
	}, nil
}

func (p *JSONPlugin) GetCommands() []plugin.CommandDefinition {
	// Legacy support - return empty
	return nil
}

func (p *JSONPlugin) GetTools() []plugin.ToolDefinition {
	// Legacy support - return empty
	return nil
}

func (p *JSONPlugin) GetProviders() []plugin.ProviderDefinition {
	// Legacy support - return empty
	return nil
}

func (p *JSONPlugin) GetImports() *plugin.PluginImports {
	return p.manifest.Imports
}

func (p *JSONPlugin) Repository() string {
	return p.manifest.Repository
}
