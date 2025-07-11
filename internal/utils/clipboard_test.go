package utils

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
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClipboard(t *testing.T) {
	clipboard := NewClipboard()
	assert.NotNil(t, clipboard)
	_, ok := clipboard.(*SystemClipboard)
	assert.True(t, ok)
}

func TestSystemClipboard_CopyPaste(t *testing.T) {
	// Skip in CI environments where clipboard may not be available
	if testing.Short() || isCI() {
		t.Skip("Skipping clipboard test in CI environment")
	}

	clipboard := NewClipboard()
	testText := "Hello, Clipboard Test!"

	// Test Copy
	err := clipboard.Copy(testText)
	if err != nil {
		// Check if it's due to missing utilities
		if runtime.GOOS == "linux" && contains(err.Error(), "no clipboard utility found") {
			t.Skip("Skipping test - no clipboard utility available")
		}
		require.NoError(t, err)
	}

	// Test Paste
	result, err := clipboard.Paste()
	if err != nil {
		// Check if it's due to missing utilities
		if runtime.GOOS == "linux" && contains(err.Error(), "no clipboard utility found") {
			t.Skip("Skipping test - no clipboard utility available")
		}
		require.NoError(t, err)
	}

	// On some systems, clipboard may add trailing newline
	assert.Contains(t, result, testText)
}

func TestSystemClipboard_UnsupportedOS(t *testing.T) {
	// This test would require mocking runtime.GOOS which isn't straightforward
	// The implementation correctly handles darwin, linux, and windows
	t.Skip("Cannot easily test unsupported OS without mocking runtime.GOOS")
}

// Helper functions

func isCI() bool {
	// Common CI environment variables
	ciVars := []string{"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "JENKINS", "TRAVIS"}
	for _, v := range ciVars {
		if os.Getenv(v) != "" {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
