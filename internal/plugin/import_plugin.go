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
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/rizome-dev/opun/pkg/plugin"
	"gopkg.in/yaml.v3"
)

// ImportPlugin represents a YAML-based plugin that imports items
type ImportPlugin struct {
	manifest plugin.PluginManifest
	path     string
}

// NewImportPlugin creates a new import plugin from a YAML file
func NewImportPlugin(path string) (*ImportPlugin, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read plugin file: %w", err)
	}

	var manifest plugin.PluginManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse plugin manifest: %w", err)
	}

	// Validate required fields
	if manifest.Name == "" {
		return nil, fmt.Errorf("plugin name is required")
	}
	if manifest.Description == "" {
		return nil, fmt.Errorf("plugin description is required")
	}

	return &ImportPlugin{
		manifest: manifest,
		path:     path,
	}, nil
}

// Name returns the plugin name
func (p *ImportPlugin) Name() string {
	return p.manifest.Name
}

// Description returns the plugin description
func (p *ImportPlugin) Description() string {
	return p.manifest.Description
}

// Author returns the plugin author
func (p *ImportPlugin) Author() string {
	return p.manifest.Author
}

// Repository returns the plugin repository
func (p *ImportPlugin) Repository() string {
	return p.manifest.Repository
}

// GetImports returns the items to import
func (p *ImportPlugin) GetImports() *plugin.PluginImports {
	return p.manifest.Imports
}

// GetManifest returns the full manifest
func (p *ImportPlugin) GetManifest() plugin.PluginManifest {
	return p.manifest
}

// GetPath returns the plugin file path
func (p *ImportPlugin) GetPath() string {
	return p.path
}

// LoadFromURL loads a plugin from a URL (GitHub, etc.)
func LoadFromURL(url string) (*ImportPlugin, error) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "opun-plugin-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Download the file
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download plugin: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download plugin: HTTP %d", resp.StatusCode)
	}

	// Copy to temp file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to save plugin: %w", err)
	}

	// Close the file before reading it
	tmpFile.Close()

	// Load from the temp file
	return NewImportPlugin(tmpFile.Name())
}

// LoadFromDirectory loads all plugins from a directory
func LoadFromDirectory(dir string) ([]*ImportPlugin, error) {
	var plugins []*ImportPlugin

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Check for plugin.yaml in subdirectory
			pluginPath := filepath.Join(dir, entry.Name(), "plugin.yaml")
			if _, err := os.Stat(pluginPath); err == nil {
				plugin, err := NewImportPlugin(pluginPath)
				if err != nil {
					// Log error but continue
					continue
				}
				plugins = append(plugins, plugin)
			}
		} else if entry.Name() == "plugin.yaml" || entry.Name() == "plugin.yml" {
			// Load plugin from file
			pluginPath := filepath.Join(dir, entry.Name())
			plugin, err := NewImportPlugin(pluginPath)
			if err != nil {
				// Log error but continue
				continue
			}
			plugins = append(plugins, plugin)
		}
	}

	return plugins, nil
}
