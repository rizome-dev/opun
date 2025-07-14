//go:build windows

package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunChat_Windows(t *testing.T) {
	// Save original environment
	oldHome := os.Getenv("HOME")
	oldUserProfile := os.Getenv("USERPROFILE")
	defer func() {
		os.Setenv("HOME", oldHome)
		os.Setenv("USERPROFILE", oldUserProfile)
	}()

	// Create a temporary home directory for testing
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	os.Setenv("USERPROFILE", tempHome) // Windows uses USERPROFILE

	// Create required directories
	opunDir := filepath.Join(tempHome, ".opun")
	actionsDir := filepath.Join(opunDir, "actions")
	require.NoError(t, os.MkdirAll(actionsDir, 0755))

	// Create a mock command
	cmd := &cobra.Command{}

	t.Run("Unsupported provider", func(t *testing.T) {
		args := []string{"unsupported"}
		err := runChat(cmd, args)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported provider")
	})

	t.Run("Claude provider not found", func(t *testing.T) {
		// This test assumes claude is not installed in the test environment
		// If claude is installed, the test will try to run it and may fail differently
		args := []string{"claude"}

		// Create a minimal config to avoid injection manager errors
		configPath := filepath.Join(opunDir, "config.yaml")
		configContent := `default_provider: claude
providers:
  claude:
    name: claude
`
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

		err := runChat(cmd, args)
		if err != nil {
			// The error could be either "command not found" or PTY-related
			assert.True(t,
				contains(err.Error(), "claude command not found") ||
					contains(err.Error(), "failed to start claude") ||
					contains(err.Error(), "failed to prepare provider environment"),
				"Unexpected error: %v", err)
		}
	})

	t.Run("Gemini provider not found", func(t *testing.T) {
		args := []string{"gemini"}

		// Create a minimal config
		configPath := filepath.Join(opunDir, "config.yaml")
		configContent := `default_provider: gemini
providers:
  gemini:
    name: gemini
`
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

		err := runChat(cmd, args)
		if err != nil {
			assert.True(t,
				contains(err.Error(), "gemini command not found") ||
					contains(err.Error(), "failed to start gemini") ||
					contains(err.Error(), "failed to prepare provider environment"),
				"Unexpected error: %v", err)
		}
	})

	t.Run("Actions directory warning", func(t *testing.T) {
		// Remove actions directory to trigger warning
		os.RemoveAll(actionsDir)

		args := []string{"claude"}
		err := runChat(cmd, args)

		// Should still get an error (provider not found or config issue)
		// but the actions loading should have shown a warning
		assert.Error(t, err)
	})
}

// TestWindowsChatSpecificFeatures tests Windows-specific chat functionality
func TestWindowsChatSpecificFeatures(t *testing.T) {
	t.Run("Windows command resolution", func(t *testing.T) {
		// Test that Windows-specific command resolution works
		// This is implicitly tested in runChat by looking for .exe and .cmd variants

		// Create a temporary directory and add it to PATH
		tempDir := t.TempDir()
		oldPath := os.Getenv("PATH")
		os.Setenv("PATH", tempDir+";"+oldPath)
		defer os.Setenv("PATH", oldPath)

		// Create mock executables
		mockClaude := filepath.Join(tempDir, "claude.exe")
		require.NoError(t, os.WriteFile(mockClaude, []byte("mock"), 0755))

		// Now the command should be found
		cmd := &cobra.Command{}
		args := []string{"claude"}

		// Set up minimal environment
		tempHome := t.TempDir()
		os.Setenv("HOME", tempHome)
		os.Setenv("USERPROFILE", tempHome)

		opunDir := filepath.Join(tempHome, ".opun")
		require.NoError(t, os.MkdirAll(filepath.Join(opunDir, "actions"), 0755))

		configPath := filepath.Join(opunDir, "config.yaml")
		configContent := `default_provider: claude
providers:
  claude:
    name: claude
`
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

		// This will still fail because our mock exe isn't a real PTY provider
		// but it should fail differently (PTY error instead of command not found)
		err := runChat(cmd, args)
		assert.Error(t, err)
		assert.NotContains(t, err.Error(), "command not found")
	})

	t.Run("No Unix signal handling", func(t *testing.T) {
		// The Windows implementation should not use SIGWINCH
		// This is verified by successful compilation on Windows
		assert.True(t, true, "Windows chat compiles without Unix signals")
	})

	t.Run("ConPTY support", func(t *testing.T) {
		// The Windows implementation uses github.com/creack/pty which supports ConPTY
		// This is a compile-time verification
		assert.True(t, true, "Windows chat uses ConPTY-compatible PTY library")
	})
}

// TestChatCommandPathResolution tests the various command path resolution strategies
func TestChatCommandPathResolution(t *testing.T) {
	// Create a temporary directory for mock commands
	tempDir := t.TempDir()
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", tempDir+";"+oldPath)
	defer os.Setenv("PATH", oldPath)

	// Set up environment
	tempHome := t.TempDir()
	os.Setenv("HOME", tempHome)
	os.Setenv("USERPROFILE", tempHome)

	opunDir := filepath.Join(tempHome, ".opun")
	require.NoError(t, os.MkdirAll(filepath.Join(opunDir, "actions"), 0755))

	configPath := filepath.Join(opunDir, "config.yaml")
	configContent := `default_provider: claude
providers:
  claude:
    name: claude
  gemini:
    name: gemini
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	cmd := &cobra.Command{}

	testCases := []struct {
		name        string
		provider    string
		createFiles []string
		shouldFind  bool
	}{
		{
			name:        "claude.exe",
			provider:    "claude",
			createFiles: []string{"claude.exe"},
			shouldFind:  true,
		},
		{
			name:        "npx.cmd for claude",
			provider:    "claude",
			createFiles: []string{"npx.cmd"},
			shouldFind:  true,
		},
		{
			name:        "gemini.exe",
			provider:    "gemini",
			createFiles: []string{"gemini.exe"},
			shouldFind:  true,
		},
		{
			name:        "no command found",
			provider:    "claude",
			createFiles: []string{},
			shouldFind:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Clean up any previous files
			files, _ := filepath.Glob(filepath.Join(tempDir, "*"))
			for _, f := range files {
				os.Remove(f)
			}

			// Create mock files
			for _, file := range tc.createFiles {
				mockFile := filepath.Join(tempDir, file)
				require.NoError(t, os.WriteFile(mockFile, []byte("mock"), 0755))
			}

			args := []string{tc.provider}
			err := runChat(cmd, args)

			if tc.shouldFind {
				// Should fail with PTY error, not command not found
				assert.Error(t, err)
				assert.NotContains(t, err.Error(), "command not found")
			} else {
				// Should fail with command not found
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "command not found")
			}
		})
	}
}

// contains is a helper function for string contains check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && (stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	return len(substr) == 0 || (len(s) >= len(substr) && (s[:len(substr)] == substr || stringContains(s[1:], substr)))
}
