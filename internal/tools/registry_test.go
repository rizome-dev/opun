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
	"testing"

	"github.com/rizome-dev/opun/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry(t *testing.T) {
	t.Run("RegisterAndGet", func(t *testing.T) {
		registry := NewRegistry()

		tool := core.StandardAction{
			ID:          "test-tool",
			Name:        "Test Tool",
			Description: "A test tool",
			Category:    "testing",
			Command:     "echo 'test'",
		}

		// Register tool
		err := registry.Register(tool)
		require.NoError(t, err)

		// Get tool
		retrieved, err := registry.Get("test-tool")
		require.NoError(t, err)
		assert.Equal(t, tool.ID, retrieved.ID)
		assert.Equal(t, tool.Name, retrieved.Name)
	})

	t.Run("DuplicateRegistration", func(t *testing.T) {
		registry := NewRegistry()

		tool := core.StandardAction{
			ID:          "test-tool",
			Name:        "Test Tool",
			Description: "A test tool",
		}

		// First registration should succeed
		err := registry.Register(tool)
		require.NoError(t, err)

		// Second registration should fail
		err = registry.Register(tool)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("ListByProvider", func(t *testing.T) {
		registry := NewRegistry()

		// Register tools with different provider support
		tools := []core.StandardAction{
			{
				ID:          "claude-only",
				Name:        "Claude Only",
				Description: "Only for Claude",
				Providers:   []string{"claude"},
			},
			{
				ID:          "gemini-only",
				Name:        "Gemini Only",
				Description: "Only for Gemini",
				Providers:   []string{"gemini"},
			},
			{
				ID:          "both",
				Name:        "Both Providers",
				Description: "For both providers",
				Providers:   []string{"claude", "gemini"},
			},
			{
				ID:          "all",
				Name:        "All Providers",
				Description: "For all providers",
				Providers:   []string{}, // Empty means all
			},
		}

		for _, tool := range tools {
			err := registry.Register(tool)
			require.NoError(t, err)
		}

		// Test filtering by Claude
		claudeTools := registry.List("claude")
		assert.Len(t, claudeTools, 3) // claude-only, both, all

		// Test filtering by Gemini
		geminiTools := registry.List("gemini")
		assert.Len(t, geminiTools, 3) // gemini-only, both, all

		// Test no filter (all tools)
		allTools := registry.List("")
		assert.Len(t, allTools, 4)
	})

	t.Run("Remove", func(t *testing.T) {
		registry := NewRegistry()

		tool := core.StandardAction{
			ID:          "test-tool",
			Name:        "Test Tool",
			Description: "A test tool",
		}

		// Register and verify
		err := registry.Register(tool)
		require.NoError(t, err)

		_, err = registry.Get("test-tool")
		require.NoError(t, err)

		// Remove
		err = registry.Remove("test-tool")
		require.NoError(t, err)

		// Verify removal
		_, err = registry.Get("test-tool")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("LoadBuiltinActions", func(t *testing.T) {
		registry := NewRegistry()

		err := registry.LoadBuiltinActions()
		require.NoError(t, err)

		// Check that some builtin tools exist
		tools := registry.List("")
		assert.NotEmpty(t, tools)

		// Verify specific builtin tool
		tool, err := registry.Get("list-files")
		require.NoError(t, err)
		assert.Equal(t, "List Files", tool.Name)
		assert.Equal(t, "ls -la", tool.Command)
	})
}
