package providers

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rizome-dev/opun/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaudeSubAgentBridge(t *testing.T) {
	// Create a temporary home directory for testing
	tempHome := t.TempDir()
	originalHome := os.Getenv("HOME")
	os.Setenv("HOME", tempHome)
	defer os.Setenv("HOME", originalHome)

	// Create test directories
	opunDir := filepath.Join(tempHome, ".opun", "subagents")
	claudeDir := filepath.Join(tempHome, ".claude", "agents")
	
	err := os.MkdirAll(opunDir, 0755)
	require.NoError(t, err)

	// Create a test subagent config in YAML format
	testConfig := `name: test-agent
description: Test agent for unit testing
provider: claude
model: sonnet
capabilities:
  - code-review
  - testing
system_prompt: You are a test agent
context:
  - When testing is needed
  - When code review is requested
`
	
	configPath := filepath.Join(opunDir, "test-agent.yaml")
	err = os.WriteFile(configPath, []byte(testConfig), 0644)
	require.NoError(t, err)

	// Create Claude provider
	providerConfig := core.ProviderConfig{
		Name:    "test-claude",
		Type:    core.ProviderTypeClaude,
		Command: "echo",
		Args:    []string{"test"},
	}
	
	provider := NewClaudeProvider(providerConfig)
	ctx := context.Background()

	t.Run("PrepareSession creates markdown files", func(t *testing.T) {
		// Prepare session should create the markdown files
		err := provider.PrepareSession(ctx, "test-session")
		assert.NoError(t, err)

		// Check if .claude/agents directory was created
		_, err = os.Stat(claudeDir)
		assert.NoError(t, err, "~/.claude/agents directory should be created")

		// Check if markdown file was created
		mdPath := filepath.Join(claudeDir, "test-agent.md")
		_, err = os.Stat(mdPath)
		assert.NoError(t, err, "test-agent.md should be created")

		// Read and verify content
		content, err := os.ReadFile(mdPath)
		require.NoError(t, err)
		
		// Check for expected content
		assert.Contains(t, string(content), "# test-agent")
		assert.Contains(t, string(content), "Test agent for unit testing")
		assert.Contains(t, string(content), "## Capabilities")
		assert.Contains(t, string(content), "- code-review")
		assert.Contains(t, string(content), "- testing")
		assert.Contains(t, string(content), "## Instructions")
		assert.Contains(t, string(content), "You are a test agent")
		assert.Contains(t, string(content), "## Context Patterns")
		assert.Contains(t, string(content), "When testing is needed")
		assert.Contains(t, string(content), "## Usage")
		assert.Contains(t, string(content), "`subagent_type`: `test-agent`")
	})

	t.Run("CleanupSession removes markdown files", func(t *testing.T) {
		// Ensure file exists before cleanup
		mdPath := filepath.Join(claudeDir, "test-agent.md")
		_, err = os.Stat(mdPath)
		require.NoError(t, err, "File should exist before cleanup")

		// Cleanup should remove the files
		err = provider.CleanupSession(ctx, "test-session")
		assert.NoError(t, err)

		// Check if markdown file was removed
		_, err = os.Stat(mdPath)
		assert.True(t, os.IsNotExist(err), "test-agent.md should be removed after cleanup")
	})

	t.Run("Only processes Claude-compatible agents", func(t *testing.T) {
		// Create a non-Claude agent config
		geminiConfig := `name: gemini-agent
description: Gemini-only agent
provider: gemini
model: pro
`
		geminiPath := filepath.Join(opunDir, "gemini-agent.yaml")
		err = os.WriteFile(geminiPath, []byte(geminiConfig), 0644)
		require.NoError(t, err)

		// Create a new provider instance
		provider2 := NewClaudeProvider(providerConfig)
		
		// Prepare session
		err = provider2.PrepareSession(ctx, "test-session-2")
		assert.NoError(t, err)

		// Check that only test-agent.md was created, not gemini-agent.md
		testMdPath := filepath.Join(claudeDir, "test-agent.md")
		geminiMdPath := filepath.Join(claudeDir, "gemini-agent.md")
		
		_, err = os.Stat(testMdPath)
		assert.NoError(t, err, "test-agent.md should be created")
		
		_, err = os.Stat(geminiMdPath)
		assert.True(t, os.IsNotExist(err), "gemini-agent.md should NOT be created")

		// Cleanup
		provider2.CleanupSession(ctx, "test-session-2")
	})
}