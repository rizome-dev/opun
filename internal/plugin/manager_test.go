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
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rizome-dev/opun/pkg/plugin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestManager(t *testing.T) {
	// Create temp directory for tests
	tmpDir, err := os.MkdirTemp("", "plugin-manager-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create manager
	manager := NewManager(tmpDir)

	t.Run("Load Plugin", func(t *testing.T) {
		// Create a test plugin manifest
		manifest := plugin.PluginManifest{
			Name:        "test-plugin",
			Description: "A test plugin",
			Author:      "Test Author",
			Repository:  "https://github.com/test/plugin",
			Imports: &plugin.PluginImports{
				Prompts: []plugin.PromptImport{
					{
						Name:        "test-prompt",
						Description: "A test prompt",
						Template:    "This is a test prompt",
					},
				},
			},
		}

		// Write manifest to file
		manifestPath := filepath.Join(tmpDir, "test-plugin.yaml")
		data, err := yaml.Marshal(manifest)
		require.NoError(t, err)
		err = os.WriteFile(manifestPath, data, 0644)
		require.NoError(t, err)

		// Load the plugin
		err = manager.LoadPlugin(manifestPath)
		require.NoError(t, err)

		// Get plugin info
		info, err := manager.GetPlugin("test-plugin")
		require.NoError(t, err)
		assert.Equal(t, "test-plugin", info.Name)
		assert.Equal(t, manifestPath, info.Source)
		assert.Equal(t, 1, info.ItemCount.Prompts)
		assert.True(t, time.Since(info.InstalledAt) < time.Minute)
	})

	t.Run("List Plugins", func(t *testing.T) {
		plugins, err := manager.ListPlugins()
		require.NoError(t, err)
		assert.Len(t, plugins, 1)
		assert.Equal(t, "test-plugin", plugins[0].Name)
	})

	t.Run("Uninstall Plugin", func(t *testing.T) {
		// Uninstall the plugin
		err := manager.UninstallPlugin("test-plugin")
		require.NoError(t, err)

		// Verify it's gone
		_, err = manager.GetPlugin("test-plugin")
		assert.Error(t, err)

		// List should be empty
		plugins, err := manager.ListPlugins()
		require.NoError(t, err)
		assert.Len(t, plugins, 0)
	})

	t.Run("Load Multiple Plugins", func(t *testing.T) {
		// Create multiple test plugins
		plugins := []struct {
			name      string
			prompts   int
			workflows int
			actions   int
		}{
			{"plugin-1", 2, 1, 0},
			{"plugin-2", 0, 2, 1},
			{"plugin-3", 1, 1, 1},
		}

		for _, p := range plugins {
			manifest := plugin.PluginManifest{
				Name:        p.name,
				Description: "Test plugin " + p.name,
				Imports:     &plugin.PluginImports{},
			}

			// Add prompts
			for i := 0; i < p.prompts; i++ {
				manifest.Imports.Prompts = append(manifest.Imports.Prompts, plugin.PromptImport{
					Name:     fmt.Sprintf("%s-prompt-%d", p.name, i),
					Template: "test",
				})
			}

			// Add workflows
			for i := 0; i < p.workflows; i++ {
				manifest.Imports.Workflows = append(manifest.Imports.Workflows, plugin.WorkflowImport{
					Name: fmt.Sprintf("%s-workflow-%d", p.name, i),
				})
			}

			// Add actions
			for i := 0; i < p.actions; i++ {
				manifest.Imports.Actions = append(manifest.Imports.Actions, plugin.ActionImport{
					Name:    fmt.Sprintf("%s-action-%d", p.name, i),
					Command: "echo test",
				})
			}

			// Write and load
			manifestPath := filepath.Join(tmpDir, p.name+".yaml")
			data, err := yaml.Marshal(manifest)
			require.NoError(t, err)
			err = os.WriteFile(manifestPath, data, 0644)
			require.NoError(t, err)

			err = manager.LoadPlugin(manifestPath)
			require.NoError(t, err)
		}

		// Verify all plugins are loaded
		list, err := manager.ListPlugins()
		require.NoError(t, err)
		assert.Len(t, list, 3)

		// Verify counts
		for _, p := range plugins {
			info, err := manager.GetPlugin(p.name)
			require.NoError(t, err)
			assert.Equal(t, p.prompts, info.ItemCount.Prompts, "Plugin %s prompts", p.name)
			assert.Equal(t, p.workflows, info.ItemCount.Workflows, "Plugin %s workflows", p.name)
			assert.Equal(t, p.actions, info.ItemCount.Actions, "Plugin %s actions", p.name)
		}
	})
}

func TestPluginValidation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "plugin-validation-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	manager := NewManager(tmpDir)

	t.Run("Invalid Plugin File", func(t *testing.T) {
		// Create invalid YAML
		invalidPath := filepath.Join(tmpDir, "invalid.yaml")
		err := os.WriteFile(invalidPath, []byte("invalid: yaml: content: ["), 0644)
		require.NoError(t, err)

		err = manager.LoadPlugin(invalidPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse")
	})

	t.Run("Missing Required Fields", func(t *testing.T) {
		manifest := plugin.PluginManifest{
			// Missing Name
			Description: "Missing name",
		}

		manifestPath := filepath.Join(tmpDir, "missing-name.yaml")
		data, err := yaml.Marshal(manifest)
		require.NoError(t, err)
		err = os.WriteFile(manifestPath, data, 0644)
		require.NoError(t, err)

		err = manager.LoadPlugin(manifestPath)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "plugin name is required")
	})
}
