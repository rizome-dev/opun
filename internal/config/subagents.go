package config

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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rizome-dev/opun/pkg/core"
	"gopkg.in/yaml.v3"
)

// SubAgentConfigLoader loads subagent configurations
type SubAgentConfigLoader struct {
	configDir string
}

// NewSubAgentConfigLoader creates a new config loader
func NewSubAgentConfigLoader() (*SubAgentConfigLoader, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	configDir := filepath.Join(home, ".opun", "subagents")
	return &SubAgentConfigLoader{
		configDir: configDir,
	}, nil
}

// LoadAll loads all subagent configurations
func (l *SubAgentConfigLoader) LoadAll() ([]core.SubAgentConfig, error) {
	// Ensure directory exists
	if err := os.MkdirAll(l.configDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	entries, err := os.ReadDir(l.configDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read config directory: %w", err)
	}

	var configs []core.SubAgentConfig
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process config files
		name := entry.Name()
		if !strings.HasSuffix(name, ".yaml") && 
		   !strings.HasSuffix(name, ".yml") && 
		   !strings.HasSuffix(name, ".json") {
			continue
		}

		config, err := l.LoadFile(filepath.Join(l.configDir, name))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to load %s: %v\n", name, err)
			continue
		}

		configs = append(configs, config)
	}

	return configs, nil
}

// LoadFile loads a single configuration file
func (l *SubAgentConfigLoader) LoadFile(path string) (core.SubAgentConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return core.SubAgentConfig{}, fmt.Errorf("failed to read file: %w", err)
	}

	var config core.SubAgentConfig
	
	// Try YAML first, then JSON
	if strings.HasSuffix(path, ".json") {
		err = json.Unmarshal(data, &config)
	} else {
		err = yaml.Unmarshal(data, &config)
	}

	if err != nil {
		return core.SubAgentConfig{}, fmt.Errorf("failed to parse config: %w", err)
	}

	// Validate the configuration
	if err := l.ValidateConfig(config); err != nil {
		return core.SubAgentConfig{}, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// Save saves a subagent configuration
func (l *SubAgentConfigLoader) Save(config core.SubAgentConfig) error {
	// Validate before saving
	if err := l.ValidateConfig(config); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Ensure directory exists
	if err := os.MkdirAll(l.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Use YAML by default
	filename := fmt.Sprintf("%s.yaml", config.Name)
	path := filepath.Join(l.configDir, filename)

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Delete removes a subagent configuration
func (l *SubAgentConfigLoader) Delete(name string) error {
	// Try different extensions
	for _, ext := range []string{".yaml", ".yml", ".json"} {
		path := filepath.Join(l.configDir, name+ext)
		if _, err := os.Stat(path); err == nil {
			return os.Remove(path)
		}
	}

	return fmt.Errorf("configuration not found: %s", name)
}

// ValidateConfig validates a subagent configuration
func (l *SubAgentConfigLoader) ValidateConfig(config core.SubAgentConfig) error {
	// Required fields
	if config.Name == "" {
		return fmt.Errorf("name is required")
	}

	if config.Provider == "" {
		return fmt.Errorf("provider is required")
	}

	// Validate provider type
	validProviders := []core.ProviderType{
		core.ProviderTypeClaude,
		core.ProviderTypeGemini,
		core.ProviderTypeQwen,
		core.ProviderTypeMock,
	}

	valid := false
	for _, p := range validProviders {
		if config.Provider == p {
			valid = true
			break
		}
	}

	if !valid {
		return fmt.Errorf("invalid provider: %s", config.Provider)
	}

	// Validate delegation strategy
	if config.Strategy != "" {
		validStrategies := []core.DelegationStrategy{
			core.DelegationAutomatic,
			core.DelegationExplicit,
			core.DelegationProactive,
		}

		valid = false
		for _, s := range validStrategies {
			if config.Strategy == s {
				valid = true
				break
			}
		}

		if !valid {
			return fmt.Errorf("invalid strategy: %s", config.Strategy)
		}
	}

	return nil
}

// List returns a list of all configuration names
func (l *SubAgentConfigLoader) List() ([]string, error) {
	configs, err := l.LoadAll()
	if err != nil {
		return nil, err
	}

	names := make([]string, len(configs))
	for i, config := range configs {
		names[i] = config.Name
	}

	return names, nil
}