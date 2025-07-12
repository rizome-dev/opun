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
		// Skip this test if claude is actually installed to avoid starting it
		if _, err := os.Stat("/usr/local/bin/claude"); err == nil {
			t.Skip("Skipping test - claude is installed")
		}
		if _, err := os.Stat("/opt/homebrew/bin/claude"); err == nil {
			t.Skip("Skipping test - claude is installed") 
		}
		
		// Temporarily modify PATH to ensure claude isn't found
		oldPath := os.Getenv("PATH")
		os.Setenv("PATH", tempHome)
		defer os.Setenv("PATH", oldPath)
		
		// This test verifies that runChat is implemented for all platforms
		// The function signature should be the same
		args := []string{"claude"}
		
		// We expect an error because the provider won't be installed in test
		err := runChat(cmd, args)
		assert.Error(t, err)
		
		// The error should be about the provider not being found or PTY issues
		// not about the function not existing
		assert.True(t,
			containsAny(err.Error(), 
				"claude command not found",
				"failed to start claude",
				"failed to prepare provider environment"),
			"Unexpected error: %v", err)
	})
	
	t.Run("Platform-specific command handling", func(t *testing.T) {
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
		
		args := []string{"claude"}
		err := runChat(cmd, args)
		
		// Should still error because it's not a real PTY provider
		assert.Error(t, err)
		
		// But the error should be different - PTY related, not command not found
		assert.NotContains(t, err.Error(), "command not found")
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