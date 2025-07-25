package e2e

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
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/fang"
	"github.com/rizome-dev/opun/internal/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/term"
)

func TestCLICommands(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Set up test environment
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	t.Run("Root Command", func(t *testing.T) {
		cmd := cli.RootCmd()
		assert.NotNil(t, cmd)
		assert.Equal(t, "opun", cmd.Use)
	})

	t.Run("Setup Command", func(t *testing.T) {
		// Skip interactive setup test in CI environment
		if os.Getenv("CI") == "true" || !isTerminal() {
			t.Skip("Skipping interactive setup test in non-TTY environment")
		}

		// Create a mock stdin for setup interaction
		mockStdin := bytes.NewBufferString("1\n") // Choose Claude
		oldStdin := os.Stdin
		r, w, _ := os.Pipe()
		os.Stdin = r
		defer func() {
			os.Stdin = oldStdin
			w.Close()
		}()

		go func() {
			w.Write(mockStdin.Bytes())
			w.Close()
		}()

		cmd := cli.RootCmd()
		cmd.SetArgs([]string{"setup"})

		// Execute with fang (capture output)
		err := fang.Execute(context.Background(), cmd)
		require.NoError(t, err)

		// Check that config was created
		configPath := filepath.Join(tempDir, ".opun", "config.yaml")
		assert.FileExists(t, configPath)

		// Read config and verify
		data, err := os.ReadFile(configPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "default_provider: claude")
	})

	t.Run("Add Command - Prompt", func(t *testing.T) {
		// Create a test prompt file
		promptPath := filepath.Join(tempDir, "test-prompt.md")
		promptContent := `# Test Prompt

This is a test prompt for {{name}}.`
		err := os.WriteFile(promptPath, []byte(promptContent), 0644)
		require.NoError(t, err)

		cmd := cli.RootCmd()
		cmd.SetArgs([]string{"add", "--prompt", "--path", promptPath, "--name", "test-prompt"})

		err = fang.Execute(context.Background(), cmd)
		require.NoError(t, err)

		// Check that prompt was saved
		gardenPath := filepath.Join(tempDir, ".opun", "promptgarden")
		files, err := os.ReadDir(gardenPath)
		require.NoError(t, err)
		// Should have at least one prompt file and index.json
		assert.GreaterOrEqual(t, len(files), 2)
	})

	t.Run("Add Command - Workflow", func(t *testing.T) {
		// Create a test workflow file
		workflowPath := filepath.Join(tempDir, "test-workflow.yaml")
		workflowContent := `name: test-workflow
description: Test workflow
agents:
  - id: agent1
    provider: claude
    model: opus
    prompt: Test prompt`
		err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
		require.NoError(t, err)

		cmd := cli.RootCmd()
		cmd.SetArgs([]string{"add", "--workflow", "--path", workflowPath, "--name", "test-workflow"})

		err = fang.Execute(context.Background(), cmd)
		require.NoError(t, err)

		// Check that workflow was saved
		savedPath := filepath.Join(tempDir, ".opun", "workflows", "test-workflow.yaml")
		assert.FileExists(t, savedPath)
	})

	t.Run("List Command", func(t *testing.T) {
		// Capture stdout since the list command prints directly
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		cmd := cli.RootCmd()
		cmd.SetArgs([]string{"list"})

		// Execute command
		err := fang.Execute(context.Background(), cmd)
		require.NoError(t, err)

		// Close writer and read output
		w.Close()
		var buf bytes.Buffer
		_, copyErr := buf.ReadFrom(r)
		require.NoError(t, copyErr)
		os.Stdout = oldStdout

		output := buf.String()
		assert.Contains(t, output, "📋 Workflows:")
		assert.Contains(t, output, "🌱 Prompts:")
	})
}

func TestWorkflowRun(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Set up test environment
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Create .opun directories
	opunDir := filepath.Join(tempDir, ".opun")
	os.MkdirAll(filepath.Join(opunDir, "workflows"), 0755)
	os.MkdirAll(filepath.Join(opunDir, "promptgarden"), 0755)

	// Create a test workflow
	workflowContent := `name: test-workflow
description: Test workflow for E2E
agents:
  - id: test-agent
    provider: mock
    model: test
    prompt: This is a test prompt`

	workflowPath := filepath.Join(opunDir, "workflows", "test-workflow.yaml")
	err := os.WriteFile(workflowPath, []byte(workflowContent), 0644)
	require.NoError(t, err)

	t.Run("Run Workflow Command", func(t *testing.T) {
		cmd := cli.RootCmd()
		cmd.SetArgs([]string{"run", "test-workflow"})

		// Create a context with timeout since mock provider will run
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Execute the command - should succeed with mock provider
		err := fang.Execute(ctx, cmd)
		assert.NoError(t, err)
	})
}

// Helper functions

// isTerminal checks if the current process is running in a terminal
func isTerminal() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}
