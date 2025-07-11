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
	"context"
	"testing"
	"time"

	"github.com/rizome-dev/opun/internal/workflow"
	cmdpkg "github.com/rizome-dev/opun/pkg/command"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutor(t *testing.T) {
	// Create test dependencies
	registry := NewRegistry()
	workflowParser := workflow.NewParser("./test-workflows")

	// We'll use nil for workflowExecutor and promptGarden in these tests since we're not testing workflow execution
	executor := NewExecutor(registry, workflowParser, nil, nil)

	t.Run("Parse Command", func(t *testing.T) {
		tests := []struct {
			input    string
			wantCmd  string
			wantArgs map[string]interface{}
			wantErr  bool
		}{
			{
				input:    "/help",
				wantCmd:  "help",
				wantArgs: map[string]interface{}{},
			},
			{
				input:   "/help test",
				wantCmd: "help",
				wantArgs: map[string]interface{}{
					"command": "test",
				},
			},
			{
				input:   "help test",
				wantCmd: "help",
				wantArgs: map[string]interface{}{
					"command": "test",
				},
			},
			{
				input:   "/add workflow myworkflow handler",
				wantCmd: "add",
				wantArgs: map[string]interface{}{
					"type":    "workflow",
					"name":    "myworkflow",
					"handler": "handler",
				},
			},
			{
				input:   "",
				wantErr: true,
			},
			{
				input:   "/",
				wantErr: true,
			},
		}

		for _, tt := range tests {
			t.Run(tt.input, func(t *testing.T) {
				cmd, args, err := executor.ParseCommand(tt.input)

				if tt.wantErr {
					assert.Error(t, err)
				} else {
					require.NoError(t, err)
					assert.Equal(t, tt.wantCmd, cmd)
					assert.Equal(t, tt.wantArgs, args)
				}
			})
		}
	})

	t.Run("Execute Builtin Commands", func(t *testing.T) {
		ctx := context.Background()

		// Test help command
		exec, err := executor.Execute(ctx, "help", map[string]interface{}{})
		require.NoError(t, err)
		assert.Equal(t, cmdpkg.StatusCompleted, exec.Status)
		assert.Contains(t, exec.Output, "Available commands")

		// Test help with specific command
		exec, err = executor.Execute(ctx, "help", map[string]interface{}{
			"command": "list",
		})
		require.NoError(t, err)
		assert.Equal(t, cmdpkg.StatusCompleted, exec.Status)
		assert.Contains(t, exec.Output, "Command: /list")

		// Test list command
		exec, err = executor.Execute(ctx, "list", map[string]interface{}{})
		require.NoError(t, err)
		assert.Equal(t, cmdpkg.StatusCompleted, exec.Status)
		assert.Contains(t, exec.Output, "System:")

		// Test clear command
		exec, err = executor.Execute(ctx, "clear", map[string]interface{}{})
		require.NoError(t, err)
		assert.Equal(t, cmdpkg.StatusCompleted, exec.Status)
		assert.Contains(t, exec.Output, "\033[2J") // ANSI clear screen
	})

	t.Run("Execute Add Command", func(t *testing.T) {
		ctx := context.Background()

		// Add a new command
		exec, err := executor.Execute(ctx, "add", map[string]interface{}{
			"type":    "workflow",
			"name":    "test",
			"handler": "test-workflow",
		})
		require.NoError(t, err)
		assert.Equal(t, cmdpkg.StatusCompleted, exec.Status)
		assert.Contains(t, exec.Output, "added successfully")

		// Verify command was added
		cmd, exists := registry.Get("test")
		assert.True(t, exists)
		assert.Equal(t, "test", cmd.Name)
		assert.Equal(t, cmdpkg.CommandTypeWorkflow, cmd.Type)
		assert.Equal(t, "test-workflow", cmd.Handler)
	})

	t.Run("Execute Remove Command", func(t *testing.T) {
		ctx := context.Background()

		// First add a command
		err := registry.Register(&cmdpkg.Command{
			Name:    "toremove",
			Type:    cmdpkg.CommandTypeWorkflow,
			Handler: "workflow",
		})
		require.NoError(t, err)

		// Remove it
		exec, err := executor.Execute(ctx, "remove", map[string]interface{}{
			"name": "toremove",
		})
		require.NoError(t, err)
		assert.Equal(t, cmdpkg.StatusCompleted, exec.Status)
		assert.Contains(t, exec.Output, "removed successfully")

		// Verify it was removed
		_, exists := registry.Get("toremove")
		assert.False(t, exists)
	})

	t.Run("Execute Unknown Command", func(t *testing.T) {
		ctx := context.Background()

		exec, err := executor.Execute(ctx, "nonexistent", map[string]interface{}{})
		assert.Error(t, err)
		assert.Nil(t, exec)
		assert.Contains(t, err.Error(), "command not found")
	})

	t.Run("Event Channel", func(t *testing.T) {
		eventChan := executor.GetEventChannel()
		assert.NotNil(t, eventChan)

		// Execute a command and check for events
		ctx := context.Background()
		go func() {
			executor.Execute(ctx, "help", map[string]interface{}{})
		}()

		// Should receive start event
		select {
		case event := <-eventChan:
			assert.Equal(t, cmdpkg.EventCommandStart, event.Type)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for event")
		}

		// Should receive complete event
		select {
		case event := <-eventChan:
			assert.Equal(t, cmdpkg.EventCommandComplete, event.Type)
		case <-time.After(100 * time.Millisecond):
			t.Fatal("timeout waiting for event")
		}
	})
}

func TestExecutorWorkflow(t *testing.T) {
	t.Skip("Skipping workflow execution tests until workflow executor is properly mocked")

	// TODO: Add tests for workflow execution once we have proper mocking
}
