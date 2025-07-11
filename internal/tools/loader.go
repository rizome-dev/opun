package tools

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

	"github.com/rizome-dev/opun/pkg/core"
	"gopkg.in/yaml.v3"
)

// ToolConfig represents a tool configuration file
type ToolConfig struct {
	ID          string `yaml:"id"`
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Category    string `yaml:"category"`

	// Execution method (one of these)
	Command     string `yaml:"command,omitempty"`
	WorkflowRef string `yaml:"workflow,omitempty"`
	PromptRef   string `yaml:"prompt,omitempty"`

	// Provider constraints
	Providers []string `yaml:"providers,omitempty"`
}

// Loader handles loading actions from the filesystem
type Loader struct {
	toolsDir string
	registry *Registry
}

// NewLoader creates a new action loader
func NewLoader(toolsDir string) *Loader {
	return &Loader{
		toolsDir: toolsDir,
		registry: NewRegistry(),
	}
}

// LoadAll loads all actions from the actions directory
func (l *Loader) LoadAll() error {
	// Ensure actions directory exists
	if err := os.MkdirAll(l.toolsDir, 0755); err != nil {
		return fmt.Errorf("failed to create actions directory: %w", err)
	}

	// Read all YAML files
	entries, err := os.ReadDir(l.toolsDir)
	if err != nil {
		return fmt.Errorf("failed to read actions directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process .yaml files
		if !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		path := filepath.Join(l.toolsDir, entry.Name())
		if err := l.LoadFile(path); err != nil {
			// Log error but continue loading other tools
			fmt.Fprintf(os.Stderr, "Warning: failed to load tool %s: %v\n", entry.Name(), err)
		}
	}

	// Also load builtin actions
	return l.registry.LoadBuiltinActions()
}

// LoadFile loads a single tool from a file
func (l *Loader) LoadFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var config ToolConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Convert to StandardAction
	action := core.StandardAction{
		ID:          config.ID,
		Name:        config.Name,
		Description: config.Description,
		Category:    config.Category,
		Version:     "1.0.0", // Default version
		Command:     config.Command,
		WorkflowRef: config.WorkflowRef,
		PromptRef:   config.PromptRef,
		Providers:   config.Providers,
	}

	// Validate action
	if action.ID == "" {
		// Use filename without extension as ID
		base := filepath.Base(path)
		action.ID = strings.TrimSuffix(base, filepath.Ext(base))
	}

	if action.Name == "" {
		action.Name = action.ID
	}

	if action.Category == "" {
		action.Category = "general"
	}

	// Ensure at least one execution method is defined
	if action.Command == "" && action.WorkflowRef == "" && action.PromptRef == "" {
		return fmt.Errorf("action must have at least one execution method (command, workflow, or prompt)")
	}

	// Register the action
	return l.registry.Register(action)
}

// SaveAction saves an action configuration to a file
func (l *Loader) SaveAction(action core.StandardAction) error {
	// Convert to config format
	config := ToolConfig{
		ID:          action.ID,
		Name:        action.Name,
		Description: action.Description,
		Category:    action.Category,
		Command:     action.Command,
		WorkflowRef: action.WorkflowRef,
		PromptRef:   action.PromptRef,
		Providers:   action.Providers,
	}

	// Marshal to YAML
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal action: %w", err)
	}

	// Write to file
	filename := fmt.Sprintf("%s.yaml", action.ID)
	path := filepath.Join(l.toolsDir, filename)

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	// Register in memory
	return l.registry.Register(action)
}

// DeleteAction deletes an action configuration file
func (l *Loader) DeleteAction(id string) error {
	// Remove from registry first
	if err := l.registry.Remove(id); err != nil {
		return err
	}

	// Remove file
	filename := fmt.Sprintf("%s.yaml", id)
	path := filepath.Join(l.toolsDir, filename)

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

// GetRegistry returns the action registry
func (l *Loader) GetRegistry() *Registry {
	return l.registry
}
