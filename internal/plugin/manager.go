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
	"sync"

	"github.com/rizome-dev/opun/pkg/plugin"
)

// Manager manages import-based plugins
type Manager struct {
	mu        sync.RWMutex
	pluginDir string
	installer *Installer
}

// NewManager creates a new plugin manager
func NewManager(baseDir string) *Manager {
	installer, _ := NewInstaller(baseDir)
	return &Manager{
		pluginDir: baseDir,
		installer: installer,
	}
}

// LoadPlugin loads an import plugin from a file
func (m *Manager) LoadPlugin(pluginPath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Load the plugin
	p, err := NewImportPlugin(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	// Install it
	if m.installer != nil {
		return m.installer.Install(p)
	}

	return fmt.Errorf("installer not initialized")
}

// ListPlugins returns all installed plugins
func (m *Manager) ListPlugins() ([]plugin.InstalledPlugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.installer != nil {
		return m.installer.List()
	}

	return nil, fmt.Errorf("installer not initialized")
}

// UninstallPlugin removes a plugin
func (m *Manager) UninstallPlugin(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.installer != nil {
		return m.installer.Uninstall(name)
	}

	return fmt.Errorf("installer not initialized")
}

// GetPlugin returns information about an installed plugin
func (m *Manager) GetPlugin(name string) (*plugin.InstalledPlugin, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.installer == nil {
		return nil, fmt.Errorf("installer not initialized")
	}

	plugins, err := m.installer.List()
	if err != nil {
		return nil, err
	}

	for _, p := range plugins {
		if p.Name == name {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("plugin not found: %s", name)
}

// LoadFromURL loads a plugin from a URL
func (m *Manager) LoadFromURL(url string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// TODO: Implement URL loading
	return fmt.Errorf("URL loading not yet implemented")
}

// UpdatePlugin updates an installed plugin
func (m *Manager) UpdatePlugin(name string) error {
	// Get existing plugin info without lock first
	plugin, err := m.GetPlugin(name)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Reload from source
	p, err := NewImportPlugin(plugin.Source)
	if err != nil {
		return fmt.Errorf("failed to reload plugin: %w", err)
	}

	// Reinstall
	if m.installer != nil {
		return m.installer.Install(p)
	}

	return fmt.Errorf("installer not initialized")
}
