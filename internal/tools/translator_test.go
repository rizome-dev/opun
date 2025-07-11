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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rizome-dev/opun/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTranslator(t *testing.T) {
	t.Run("ToClaudeMarkdown", func(t *testing.T) {
		registry := NewRegistry()
		translator := NewTranslator(registry)

		tests := []struct {
			name     string
			tool     core.StandardAction
			contains []string
		}{
			{
				name: "Command tool",
				tool: core.StandardAction{
					ID:          "test-cmd",
					Name:        "Test Command",
					Description: "A test command tool",
					Category:    "testing",
					Command:     "echo hello",
				},
				contains: []string{
					"# Test Command",
					"A test command tool",
					"```bash",
					"echo hello $ARGUMENTS",
					"Category: testing",
				},
			},
			{
				name: "Workflow tool",
				tool: core.StandardAction{
					ID:          "test-wf",
					Name:        "Test Workflow",
					Description: "A test workflow tool",
					WorkflowRef: "my-workflow",
				},
				contains: []string{
					"# Test Workflow",
					"Execute the Opun workflow: `my-workflow`",
					"opun run my-workflow",
				},
			},
			{
				name: "Prompt tool",
				tool: core.StandardAction{
					ID:          "test-prompt",
					Name:        "Test Prompt",
					Description: "A test prompt tool",
					PromptRef:   "my-prompt",
				},
				contains: []string{
					"# Test Prompt",
					"Execute the Opun prompt: `my-prompt`",
					"opun prompt my-prompt",
				},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				markdown, err := translator.ToClaudeMarkdown(tt.tool)
				require.NoError(t, err)

				for _, expected := range tt.contains {
					assert.Contains(t, markdown, expected)
				}
			})
		}
	})

	t.Run("ToMCPAction", func(t *testing.T) {
		registry := NewRegistry()
		translator := NewTranslator(registry)

		tool := core.StandardAction{
			ID:          "test-tool",
			Name:        "Test Tool",
			Description: "A test tool",
			Category:    "testing",
			Command:     "echo test",
		}

		mcpTool := translator.ToMCPAction(tool)

		// Verify structure
		assert.Equal(t, "action_test-tool", mcpTool["name"])
		assert.Equal(t, "[Action] Test Tool: A test tool", mcpTool["description"])

		// Check input schema
		inputSchema, ok := mcpTool["inputSchema"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "object", inputSchema["type"])

		// Check metadata
		metadata, ok := mcpTool["metadata"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "testing", metadata["category"])
		assert.Equal(t, "test-tool", metadata["action_id"])
		assert.Equal(t, "echo test", metadata["command"])
	})

	t.Run("WriteClaudeCommand", func(t *testing.T) {
		registry := NewRegistry()
		translator := NewTranslator(registry)

		// Create temp directory
		tmpDir, err := os.MkdirTemp("", "translator-test")
		require.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		tool := core.StandardAction{
			ID:          "test-write",
			Name:        "Test Write",
			Description: "Test writing to file",
			Command:     "ls",
		}

		err = translator.WriteClaudeCommand(tool, tmpDir)
		require.NoError(t, err)

		// Verify file was created
		expectedPath := filepath.Join(tmpDir, "test-write.md")
		assert.FileExists(t, expectedPath)

		// Read and verify content
		content, err := os.ReadFile(expectedPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "# Test Write")
		assert.Contains(t, string(content), "ls $ARGUMENTS")
	})

	t.Run("GetMCPActions", func(t *testing.T) {
		registry := NewRegistry()
		translator := NewTranslator(registry)

		// Register some tools
		tools := []core.StandardAction{
			{
				ID:          "tool1",
				Name:        "Tool 1",
				Description: "First tool",
				Providers:   []string{"claude"},
			},
			{
				ID:          "tool2",
				Name:        "Tool 2",
				Description: "Second tool",
				Providers:   []string{"gemini"},
			},
			{
				ID:          "tool3",
				Name:        "Tool 3",
				Description: "Third tool",
				// No providers means all
			},
		}

		for _, tool := range tools {
			err := registry.Register(tool)
			require.NoError(t, err)
		}

		// Get MCP tools for claude
		claudeTools := translator.GetMCPActions("claude")
		assert.Len(t, claudeTools, 2) // tool1 and tool3

		// Verify tool names
		names := []string{}
		for _, t := range claudeTools {
			name, _ := t["name"].(string)
			names = append(names, name)
		}
		assert.Contains(t, names, "action_tool1")
		assert.Contains(t, names, "action_tool3")
	})
}

func TestToolMarkdownGeneration(t *testing.T) {
	// Test that generated markdown is valid and readable
	registry := NewRegistry()
	translator := NewTranslator(registry)

	tool := core.StandardAction{
		ID:          "analyzer",
		Name:        "Code Analyzer",
		Description: "Analyzes code quality and suggests improvements",
		Category:    "development",
		Command:     "opun analyze",
	}

	markdown, err := translator.ToClaudeMarkdown(tool)
	require.NoError(t, err)

	// Check markdown structure
	lines := strings.Split(markdown, "\n")
	assert.True(t, strings.HasPrefix(lines[0], "# "))

	// Ensure it has proper sections
	assert.Contains(t, markdown, "## Command")
	assert.Contains(t, markdown, "## Category")

	// Ensure it's readable
	assert.True(t, len(markdown) > 50, "Markdown should have substantial content")
}
