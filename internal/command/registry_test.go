package command

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
	"testing"

	cmdpkg "github.com/rizome-dev/opun/pkg/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistry(t *testing.T) {
	t.Run("New Registry", func(t *testing.T) {
		registry := NewRegistry()
		assert.NotNil(t, registry)

		// Should have built-in commands
		commands := registry.List()
		assert.Greater(t, len(commands), 0)

		// Check for specific built-in commands
		help, exists := registry.Get("help")
		assert.True(t, exists)
		assert.Equal(t, "help", help.Name)

		// Check aliases work
		h, exists := registry.Get("h")
		assert.True(t, exists)
		assert.Equal(t, "help", h.Name)
	})

	t.Run("Register Command", func(t *testing.T) {
		registry := NewRegistry()

		cmd := &cmdpkg.Command{
			Name:        "test",
			Description: "Test command",
			Type:        cmdpkg.CommandTypeWorkflow,
			Handler:     "test-workflow",
			Aliases:     []string{"t"},
		}

		err := registry.Register(cmd)
		require.NoError(t, err)

		// Should be able to get by name
		retrieved, exists := registry.Get("test")
		assert.True(t, exists)
		assert.Equal(t, cmd.Name, retrieved.Name)

		// Should be able to get by alias
		retrieved, exists = registry.Get("t")
		assert.True(t, exists)
		assert.Equal(t, cmd.Name, retrieved.Name)
	})

	t.Run("Register Duplicate", func(t *testing.T) {
		registry := NewRegistry()

		cmd := &cmdpkg.Command{
			Name:    "test",
			Type:    cmdpkg.CommandTypeWorkflow,
			Handler: "test-workflow",
		}

		err := registry.Register(cmd)
		require.NoError(t, err)

		// Should fail to register duplicate
		err = registry.Register(cmd)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
	})

	t.Run("Register Conflicting Alias", func(t *testing.T) {
		registry := NewRegistry()

		cmd1 := &cmdpkg.Command{
			Name:    "test1",
			Type:    cmdpkg.CommandTypeWorkflow,
			Handler: "workflow1",
			Aliases: []string{"t"},
		}

		cmd2 := &cmdpkg.Command{
			Name:    "test2",
			Type:    cmdpkg.CommandTypeWorkflow,
			Handler: "workflow2",
			Aliases: []string{"t"},
		}

		err := registry.Register(cmd1)
		require.NoError(t, err)

		err = registry.Register(cmd2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "alias")
	})

	t.Run("List Commands", func(t *testing.T) {
		registry := NewRegistry()

		// Add test commands
		for i := 0; i < 3; i++ {
			cmd := &cmdpkg.Command{
				Name:     fmt.Sprintf("test%d", i),
				Type:     cmdpkg.CommandTypeWorkflow,
				Handler:  fmt.Sprintf("workflow%d", i),
				Category: "Test",
			}
			err := registry.Register(cmd)
			require.NoError(t, err)
		}

		// Add hidden command
		hidden := &cmdpkg.Command{
			Name:    "hidden",
			Type:    cmdpkg.CommandTypeWorkflow,
			Handler: "hidden-workflow",
			Hidden:  true,
		}
		err := registry.Register(hidden)
		require.NoError(t, err)

		// List should not include hidden
		commands := registry.List()
		for _, cmd := range commands {
			assert.False(t, cmd.Hidden)
		}
	})

	t.Run("List By Category", func(t *testing.T) {
		registry := NewRegistry()

		// Add commands in different categories
		categories := []string{"Cat1", "Cat2", "Cat1"}
		for i, cat := range categories {
			cmd := &cmdpkg.Command{
				Name:     fmt.Sprintf("cmd%d", i),
				Type:     cmdpkg.CommandTypeWorkflow,
				Handler:  fmt.Sprintf("workflow%d", i),
				Category: cat,
			}
			err := registry.Register(cmd)
			require.NoError(t, err)
		}

		byCategory := registry.ListByCategory()
		assert.Len(t, byCategory["Cat1"], 2)
		assert.Len(t, byCategory["Cat2"], 1)

		// Built-in commands should be in System category
		assert.Greater(t, len(byCategory["System"]), 0)
	})

	t.Run("Search Commands", func(t *testing.T) {
		registry := NewRegistry()

		// Add test commands
		commands := []*cmdpkg.Command{
			{
				Name:        "refactor",
				Description: "Refactor code",
				Type:        cmdpkg.CommandTypeWorkflow,
				Handler:     "refactor-workflow",
			},
			{
				Name:        "review",
				Description: "Review changes",
				Type:        cmdpkg.CommandTypeWorkflow,
				Handler:     "review-workflow",
				Aliases:     []string{"rev"},
			},
			{
				Name:        "test",
				Description: "Run tests",
				Type:        cmdpkg.CommandTypeWorkflow,
				Handler:     "test-workflow",
			},
		}

		for _, cmd := range commands {
			err := registry.Register(cmd)
			require.NoError(t, err)
		}

		// Search by name (should find refactor, review, remove, and clear (via "screen"))
		results := registry.Search("re")
		assert.GreaterOrEqual(t, len(results), 2) // At least refactor and review

		// Check that our test commands are in the results
		foundRefactor := false
		foundReview := false
		for _, r := range results {
			if r.Name == "refactor" {
				foundRefactor = true
			}
			if r.Name == "review" {
				foundReview = true
			}
		}
		assert.True(t, foundRefactor)
		assert.True(t, foundReview)

		// Search by description
		results = registry.Search("code")
		assert.Len(t, results, 1)
		assert.Equal(t, "refactor", results[0].Name)

		// Search by alias
		results = registry.Search("rev")
		assert.Len(t, results, 1)
		assert.Equal(t, "review", results[0].Name)
	})

	t.Run("Remove Command", func(t *testing.T) {
		registry := NewRegistry()

		cmd := &cmdpkg.Command{
			Name:    "test",
			Type:    cmdpkg.CommandTypeWorkflow,
			Handler: "test-workflow",
			Aliases: []string{"t"},
		}

		err := registry.Register(cmd)
		require.NoError(t, err)

		// Should exist
		_, exists := registry.Get("test")
		assert.True(t, exists)

		// Remove
		err = registry.Remove("test")
		require.NoError(t, err)

		// Should not exist
		_, exists = registry.Get("test")
		assert.False(t, exists)

		// Alias should also be removed
		_, exists = registry.Get("t")
		assert.False(t, exists)

		// Should error on removing non-existent
		err = registry.Remove("test")
		assert.Error(t, err)
	})
}

func TestCommandValidation(t *testing.T) {
	registry := NewRegistry()

	tests := []struct {
		name    string
		command *cmdpkg.Command
		wantErr bool
		errMsg  string
	}{
		{
			name: "Valid Command",
			command: &cmdpkg.Command{
				Name:    "test",
				Type:    cmdpkg.CommandTypeWorkflow,
				Handler: "test-workflow",
			},
			wantErr: false,
		},
		{
			name: "Missing Name",
			command: &cmdpkg.Command{
				Type:    cmdpkg.CommandTypeWorkflow,
				Handler: "test-workflow",
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "Missing Type",
			command: &cmdpkg.Command{
				Name:    "test",
				Handler: "test-workflow",
			},
			wantErr: true,
			errMsg:  "type is required",
		},
		{
			name: "Missing Handler",
			command: &cmdpkg.Command{
				Name: "test",
				Type: cmdpkg.CommandTypeWorkflow,
			},
			wantErr: true,
			errMsg:  "handler is required",
		},
		{
			name: "Duplicate Argument Names",
			command: &cmdpkg.Command{
				Name:    "test",
				Type:    cmdpkg.CommandTypeWorkflow,
				Handler: "test-workflow",
				Arguments: []cmdpkg.Argument{
					{Name: "arg1"},
					{Name: "arg1"},
				},
			},
			wantErr: true,
			errMsg:  "duplicate argument",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := registry.Register(tt.command)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
