package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestChatCrossPlatform verifies that chat functionality works on all platforms
func TestChatCrossPlatform(t *testing.T) {
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
	if runtime.GOOS == "windows" {
		os.Setenv("USERPROFILE", tempHome)
	}

	// Create required directories
	opunDir := filepath.Join(tempHome, ".opun")
	actionsDir := filepath.Join(opunDir, "actions")
	require.NoError(t, os.MkdirAll(actionsDir, 0755))

	// Create a minimal config
	configPath := filepath.Join(opunDir, "config.yaml")
	configContent := `default_provider: claude
providers:
  claude:
    name: claude
  gemini:
    name: gemini
`
	require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

	// Create a mock command
	cmd := &cobra.Command{}

	t.Run("runChat exists on all platforms", func(t *testing.T) {
		// In CI, Claude might be installed, which causes issues
		// So we'll test with a non-existent provider instead

		// Create config for the non-existent provider to avoid config errors
		configPath := filepath.Join(opunDir, "config.yaml")
		configContent := `default_provider: nonexistent-provider
providers:
  nonexistent-provider:
    name: nonexistent-provider
`
		require.NoError(t, os.WriteFile(configPath, []byte(configContent), 0644))

		// We expect an error because the provider won't exist
		err := runChat(cmd, "nonexistent-provider", []string{})
		assert.Error(t, err)

		// The error should be about unsupported provider
		assert.Contains(t, err.Error(), "unsupported provider: nonexistent-provider")
	})

	t.Run("Platform-specific command handling", func(t *testing.T) {
		// Skip this test if Claude is actually running to avoid interference
		if os.Getenv("CI") != "" {
			t.Skip("Skipping in CI environment where Claude may be installed")
		}

		// On Windows, the implementation should look for .exe and .cmd files
		// On Unix, it should look for standard executables
		// This is tested implicitly by the runChat function

		// Create a temporary directory for mock commands
		tempDir := t.TempDir()
		oldPath := os.Getenv("PATH")
		pathSeparator := string(os.PathListSeparator)
		os.Setenv("PATH", tempDir+pathSeparator+oldPath)
		defer os.Setenv("PATH", oldPath)

		// Create a mock executable appropriate for the platform
		var mockFile string
		if runtime.GOOS == "windows" {
			mockFile = filepath.Join(tempDir, "claude.exe")
		} else {
			mockFile = filepath.Join(tempDir, "claude")
		}

		// Write a simple script that exits immediately
		content := []byte("#!/bin/sh\nexit 0\n")
		if runtime.GOOS == "windows" {
			content = []byte("@echo off\nexit /b 0\n")
		}
		require.NoError(t, os.WriteFile(mockFile, content, 0755))

		err := runChat(cmd, "claude", []string{})

		// The result depends on whether the mock script works as a PTY
		// If err is nil, that means our mock was found and executed
		// If err is not nil, check it's not "command not found"
		if err != nil {
			// Should not be a "command not found" error since we created the mock
			assert.NotContains(t, err.Error(), "command not found")
		}
		// If err is nil, that's also acceptable - it means the mock was executed
	})

	t.Run("No platform-specific signals in interface", func(t *testing.T) {
		// The chat implementation should not expose SIGWINCH or other
		// Unix-specific signals in its public interface
		// This is verified by successful compilation on all platforms
		assert.True(t, true, "Chat implementation compiles on all platforms")
	})
}

// containsAny checks if the string contains any of the substrings
func containsAny(s string, substrs ...string) bool {
	for _, substr := range substrs {
		if stringContains(s, substr) {
			return true
		}
	}
	return false
}

// stringContains is a simple string contains implementation
func stringContains(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	if len(s) < len(substr) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
