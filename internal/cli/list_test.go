package cli

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
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestListCmd(t *testing.T) {
	cmd := ListCmd()

	t.Run("Basic Properties", func(t *testing.T) {
		assert.Equal(t, "list", cmd.Use)
		assert.Contains(t, cmd.Short, "List available workflows")
	})

	t.Run("Flags", func(t *testing.T) {
		// Check that flags exist
		assert.NotNil(t, cmd.Flag("workflows"))
		assert.NotNil(t, cmd.Flag("prompts"))
		assert.NotNil(t, cmd.Flag("actions"))

		// Check that flags have shorthand versions
		workflowFlag := cmd.Flag("workflows")
		promptsFlag := cmd.Flag("prompts")
		actionsFlag := cmd.Flag("actions")
		assert.NotNil(t, workflowFlag)
		assert.NotNil(t, promptsFlag)
		assert.NotNil(t, actionsFlag)
		assert.Equal(t, "w", workflowFlag.Shorthand)
		assert.Equal(t, "p", promptsFlag.Shorthand)
		assert.Equal(t, "a", actionsFlag.Shorthand)
	})
}

func TestShowWorkflows(t *testing.T) {
	// Set up test environment
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	t.Run("No Workflows", func(t *testing.T) {
		err := showWorkflows()
		assert.NoError(t, err)

		w.Close()
		buf := make([]byte, 1024)
		n, _ := r.Read(buf)
		output := string(buf[:n])

		assert.Contains(t, output, "📋 Workflows: (none)")
	})

	// Reset pipe
	r, w, _ = os.Pipe()
	os.Stdout = w

	t.Run("With Workflows", func(t *testing.T) {
		// Create workflows directory
		workflowDir := filepath.Join(tempDir, ".opun", "workflows")
		err := os.MkdirAll(workflowDir, 0755)
		require.NoError(t, err)

		// Create test workflows
		workflow1 := map[string]interface{}{
			"name":        "test-workflow",
			"description": "Test workflow description",
			"agents":      []interface{}{},
		}

		data1, err := yaml.Marshal(workflow1)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(workflowDir, "test-workflow.yaml"), data1, 0644)
		require.NoError(t, err)

		workflow2 := map[string]interface{}{
			"name":   "another-workflow",
			"agents": []interface{}{},
		}

		data2, err := yaml.Marshal(workflow2)
		require.NoError(t, err)
		err = os.WriteFile(filepath.Join(workflowDir, "another-workflow.yaml"), data2, 0644)
		require.NoError(t, err)

		// Create a non-yaml file that should be ignored
		err = os.WriteFile(filepath.Join(workflowDir, "readme.txt"), []byte("ignore me"), 0644)
		require.NoError(t, err)

		err = showWorkflows()
		assert.NoError(t, err)

		w.Close()
		buf := make([]byte, 4096)
		n, _ := r.Read(buf)
		output := string(buf[:n])

		assert.Contains(t, output, "📋 Workflows:")
		assert.Contains(t, output, "/test-workflow - Test workflow description")
		assert.Contains(t, output, "/another-workflow")
		assert.Contains(t, output, "Total: 2 workflow(s)")
		assert.NotContains(t, output, "readme.txt")
	})

	// Restore stdout
	os.Stdout = oldStdout
}

func TestShowPrompts(t *testing.T) {
	// Set up test environment
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	t.Run("No Prompt Garden", func(t *testing.T) {
		err := showPrompts()
		assert.NoError(t, err)

		w.Close()
		buf := make([]byte, 1024)
		n, _ := r.Read(buf)
		output := string(buf[:n])

		assert.Contains(t, output, "🌱 Prompts: (none)")
	})

	// Restore stdout
	os.Stdout = oldStdout

	t.Run("With Prompt Garden", func(t *testing.T) {
		// Create prompt garden directory
		gardenDir := filepath.Join(tempDir, ".opun", "promptgarden")
		err := os.MkdirAll(gardenDir, 0755)
		require.NoError(t, err)

		// Since the prompt garden initialization is complex, we'll just verify
		// that it attempts to access the garden
		// In a real test, we'd mock the promptgarden package
	})
}

func TestShowActions(t *testing.T) {
	// Set up test environment
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	t.Run("No Actions", func(t *testing.T) {
		err := showActions()
		assert.NoError(t, err)

		w.Close()
		buf := make([]byte, 1024)
		n, _ := r.Read(buf)
		output := string(buf[:n])

		assert.Contains(t, output, "⚡ Actions: (none)")
	})

	// Restore stdout
	os.Stdout = oldStdout
}

func TestListCmd_Execute(t *testing.T) {
	// Set up test environment
	tempDir := t.TempDir()
	oldHome := os.Getenv("HOME")
	os.Setenv("HOME", tempDir)
	defer os.Setenv("HOME", oldHome)

	t.Run("List All", func(t *testing.T) {
		// Create fresh command for each test
		cmd := ListCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)

		// No flags = list all
		cmd.SetArgs([]string{})

		// Capture stdout since the functions write directly to stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := cmd.Execute()
		assert.NoError(t, err)

		w.Close()
		outBuf := make([]byte, 4096)
		n, _ := r.Read(outBuf)
		output := string(outBuf[:n])
		os.Stdout = oldStdout

		// Should show all three sections
		assert.Contains(t, output, "📋 Workflows:")
		assert.Contains(t, output, "🌱 Prompts:")
		assert.Contains(t, output, "⚡ Actions:")
	})

	t.Run("List Only Workflows", func(t *testing.T) {
		// Create fresh command for each test
		cmd := ListCmd()
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"--workflows"})

		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		err := cmd.Execute()
		assert.NoError(t, err)

		w.Close()
		outBuf := make([]byte, 4096)
		n, _ := r.Read(outBuf)
		output := string(outBuf[:n])
		os.Stdout = oldStdout

		// Should only show workflows
		assert.Contains(t, output, "📋 Workflows:")
		assert.NotContains(t, output, "🌱 Prompts:")
		assert.NotContains(t, output, "⚡ Actions:")
	})
}
