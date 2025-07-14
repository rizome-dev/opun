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
	// "context"
	"testing"
	"time"

	"github.com/rizome-dev/opun/internal/pty"
	// "github.com/rizome-dev/opun/internal/pty/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPTYAutomation(t *testing.T) {
	// Skip if not in CI or explicit e2e mode
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	t.Run("PTY Session Creation", func(t *testing.T) {
		config := pty.SessionConfig{
			Provider: "test",
			Command:  "/bin/sh",
			Args:     []string{"-c", "echo 'hello world'; sleep 0.5"},
		}

		session, err := pty.NewSession(config)
		require.NoError(t, err)
		defer session.Close()

		// Wait for output with retry logic
		var output []byte
		for i := 0; i < 10; i++ {
			time.Sleep(100 * time.Millisecond)
			output = session.GetOutput()
			if len(output) > 0 && string(output) != "" {
				break
			}
		}

		assert.Contains(t, string(output), "hello world")
	})

	t.Run("PTY Automator Copy/Paste", func(t *testing.T) {
		// This test requires a more complex setup with a PTY that accepts input
		// Skip for now as it requires interactive shell
		t.Skip("Requires interactive shell setup")
	})
}

func TestClaudeProvider(t *testing.T) {
	// Skip this test as it requires actual Claude CLI to be installed
	t.Skip("Skipping Claude provider test - requires Claude CLI installation")

	// // Skip if Claude CLI is not available
	// if testing.Short() {
	// 	t.Skip("Skipping Claude provider test in short mode")
	// }

	// t.Run("Claude Provider Initialization", func(t *testing.T) {
	// 	provider := providers.NewClaudePTYProvider()
	// 	assert.NotNil(t, provider)

	// 	// Try to start a session (will fail if Claude CLI not installed)
	// 	ctx := context.Background()
	// 	err := provider.StartSession(ctx, ".")

	// 	if err != nil {
	// 		t.Logf("Claude CLI not available: %v", err)
	// 		t.Skip("Claude CLI not installed")
	// 	}

	// 	defer provider.StopSession()

	// 	// Check if ready
	// 	assert.Eventually(t, func() bool {
	// 		return provider.IsReady()
	// 	}, 5*time.Second, 100*time.Millisecond)
	// })
}

func TestGeminiProvider(t *testing.T) {
	// Skip this test as it requires actual Gemini CLI to be installed
	t.Skip("Skipping Gemini provider test - requires Gemini CLI installation")

	// // Skip if Gemini CLI is not available
	// if testing.Short() {
	// 	t.Skip("Skipping Gemini provider test in short mode")
	// }

	// t.Run("Gemini Provider Initialization", func(t *testing.T) {
	// 	provider := providers.NewGeminiPTYProvider()
	// 	assert.NotNil(t, provider)

	// 	// Try to start a session (will fail if Gemini CLI not installed)
	// 	ctx := context.Background()
	// 	err := provider.StartSession(ctx, ".")

	// 	if err != nil {
	// 		t.Logf("Gemini CLI not available: %v", err)
	// 		t.Skip("Gemini CLI not installed")
	// 	}

	// 	defer provider.StopSession()

	// 	// Check if ready
	// 	assert.Eventually(t, func() bool {
	// 		return provider.IsReady()
	// 	}, 5*time.Second, 100*time.Millisecond)
	// })
}
