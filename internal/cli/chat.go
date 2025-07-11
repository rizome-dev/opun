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
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/creack/pty"
	"github.com/rizome-dev/opun/internal/config"
	"github.com/rizome-dev/opun/internal/tools"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

// ChatCmd creates the chat command
func ChatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chat [provider]",
		Short: "Start an interactive chat session with an AI provider",
		Long: `Start an interactive chat session with Claude or Gemini.

If no provider is specified, uses the default provider from your configuration.
Your promptgarden prompts and configured slash commands are available through the injection system.

Examples:
  opun chat          # Use default provider
  opun chat claude   # Chat with Claude
  opun chat gemini   # Chat with Gemini`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var provider string

			// If no provider specified, use default from config
			if len(args) == 0 {
				provider = viper.GetString("default_provider")
				if provider == "" {
					return fmt.Errorf("no provider specified and no default provider configured. Run 'opun setup' to configure a default provider")
				}
			} else {
				provider = strings.ToLower(args[0])
			}

			return runChat(cmd, []string{provider})
		},
	}

	return cmd
}

func runChat(cmd *cobra.Command, args []string) error {
	provider := strings.ToLower(args[0])

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
	switch provider {
	case "claude":
		command = "claude"
	case "gemini":
		command = "gemini"
	default:
		return fmt.Errorf("unsupported provider: %s", provider)
	}

	// Create command with prepared environment
	c := exec.Command(command)
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

	// Handle pty size changes
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
				fmt.Fprintf(os.Stderr, "error resizing pty: %v\n", err)
			}
		}
	}()
	ch <- syscall.SIGWINCH // Initial resize
	defer func() { signal.Stop(ch); close(ch) }()

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
