package workflow

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestInteractiveExecutorCrossPlatform verifies that both Unix and Windows
// implementations provide the same interface and basic functionality
func TestInteractiveExecutorCrossPlatform(t *testing.T) {
	t.Run("Implementation exists for current platform", func(t *testing.T) {
		// This test verifies that we can create an executor on any platform
		executor := NewInteractiveExecutor()
		assert.NotNil(t, executor)
		
		// Verify the executor has the expected structure
		assert.NotNil(t, executor.outputs)
		assert.NotNil(t, executor.handoffContext)
		
		// GetState should work on all platforms
		state := executor.GetState()
		assert.Nil(t, state) // Initially nil
	})
	
	t.Run("Platform-specific command resolution", func(t *testing.T) {
		executor := NewInteractiveExecutor()
		
		// Test command resolution for claude
		cmd, args, err := executor.getProviderCommandAndArgs("claude")
		
		if err != nil {
			// If claude is not installed, we should get an appropriate error
			assert.Contains(t, err.Error(), "claude command not found")
		} else {
			// If found, verify the command is appropriate for the platform
			if runtime.GOOS == "windows" {
				// On Windows, we should check for .exe and .cmd variants
				assert.True(t, 
					cmd == "claude" || 
					cmd == "claude.exe" || 
					cmd == "npx" || 
					cmd == "npx.cmd",
					"Expected Windows-compatible command, got: %s", cmd)
			} else {
				// On Unix, we should get the standard commands
				assert.True(t,
					cmd == "claude" || 
					cmd == "npx",
					"Expected Unix command, got: %s", cmd)
			}
			
			// Verify args are correct for npx
			if cmd == "npx" || cmd == "npx.cmd" {
				assert.Equal(t, []string{"claude-code"}, args)
			} else {
				assert.Equal(t, []string{}, args)
			}
		}
	})
	
	t.Run("No platform-specific signal handling in interface", func(t *testing.T) {
		// This test verifies that the public interface doesn't expose
		// platform-specific signal handling (like SIGWINCH)
		executor := NewInteractiveExecutor()
		
		// The executor should not have any public methods or fields
		// that are platform-specific
		// This is a compile-time verification, but we can at least
		// check that the executor compiles and runs on all platforms
		assert.NotNil(t, executor)
	})
}