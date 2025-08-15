//go:build windows

package cli

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/creack/pty"
	"github.com/rizome-dev/opun/internal/config"
	"github.com/rizome-dev/opun/internal/tools"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

func runChat(cmd *cobra.Command, provider string, providerArgs []string) error {
	provider = strings.ToLower(provider)

	// Load actions from the actions directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	actionsDir := filepath.Join(homeDir, ".opun", "actions")
	actionLoader := tools.NewLoader(actionsDir)

	// Load all actions
	if err := actionLoader.LoadAll(); err != nil {
		// Non-fatal, log and continue
		fmt.Printf("Warning: failed to load actions: %v\n", err)
	}

	// Get the action registry
	actionRegistry := actionLoader.GetRegistry()

	// Prepare provider environment with injected configuration
	injectionManager, err := config.NewInjectionManager(actionRegistry)
	if err != nil {
		return fmt.Errorf("failed to create injection manager: %w", err)
	}

	env, err := injectionManager.PrepareProviderEnvironment(provider)
	if err != nil {
		return fmt.Errorf("failed to prepare provider environment: %w", err)
	}
	defer env.Cleanup()

	// Perform health check and display results
	healthCheck, err := NewHealthCheck(provider, injectionManager)
	if err != nil {
		// Non-fatal, still show basic info
		fmt.Printf("🚀 Starting %s chat session...\n", provider)
		fmt.Printf("⚠️  Could not perform health check: %v\n\n", err)
	} else {
		services := healthCheck.CheckAllServices()
		DisplayHealthCheck(provider, services)
	}

	// Determine which command to run
	var command string
	var commandArgs []string
	switch provider {
	case "claude":
		// On Windows, we need to handle .exe and .cmd extensions
		if _, err := exec.LookPath("claude"); err == nil {
			command = "claude"
		} else if _, err := exec.LookPath("claude.exe"); err == nil {
			command = "claude.exe"
		} else if _, err := exec.LookPath("npx"); err == nil {
			command = "npx"
			commandArgs = []string{"claude-code"}
		} else if _, err := exec.LookPath("npx.cmd"); err == nil {
			command = "npx.cmd"
			commandArgs = []string{"claude-code"}
		} else {
			return fmt.Errorf("claude command not found, please install Claude CLI")
		}
	case "gemini":
		if _, err := exec.LookPath("gemini"); err == nil {
			command = "gemini"
		} else if _, err := exec.LookPath("gemini.exe"); err == nil {
			command = "gemini.exe"
		} else {
			return fmt.Errorf("gemini command not found, please install Gemini CLI")
		}
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}

	// Append provider arguments to command arguments
	commandArgs = append(commandArgs, providerArgs...)
	
	// Create command with prepared environment and provider arguments
	// #nosec G204 -- command is hardcoded based on provider type
	c := exec.Command(command, commandArgs...)
	c.Dir = env.WorkingDir

	// Apply environment variables
	c.Env = os.Environ()
	for k, v := range env.Environment {
		c.Env = append(c.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Start command with PTY
	ptmx, err := pty.Start(c)
	if err != nil {
		return fmt.Errorf("failed to start %s: %w", provider, err)
	}
	defer func() { _ = ptmx.Close() }()

	// On Windows, we don't need to handle SIGWINCH for resizing
	// The Windows Console API handles this automatically with ConPTY

	// Set initial size from current terminal
	if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
		// Non-fatal error, continue without resize
		fmt.Fprintf(os.Stderr, "warning: could not set initial PTY size: %v\n", err)
	}

	// Set stdin to raw mode if it's a terminal
	if term.IsTerminal(int(os.Stdin.Fd())) {
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("failed to set raw mode: %w", err)
		}
		defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()
	}

	// Copy stdin to pty master
	go func() {
		_, _ = io.Copy(ptmx, os.Stdin)
	}()

	// Copy pty master to stdout
	_, _ = io.Copy(os.Stdout, ptmx)

	return nil
}
