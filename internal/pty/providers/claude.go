package providers

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
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/rizome-dev/opun/internal/pty"
)

// ClaudePTYProvider handles Claude-specific PTY interactions
type ClaudePTYProvider struct {
	session   *pty.Session
	automator *pty.Automator
}

// NewClaudePTYProvider creates a new Claude PTY provider
func NewClaudePTYProvider() *ClaudePTYProvider {
	return &ClaudePTYProvider{}
}

// StartSession starts a new Claude session
func (p *ClaudePTYProvider) StartSession(ctx context.Context, workingDir string) error {
	// Try to find claude command
	command := "claude"
	var args []string

	if _, err := exec.LookPath(command); err != nil {
		// Try npx fallback
		command = "npx"
		args = []string{"claude-code"}
	}

	config := pty.SessionConfig{
		Provider:   "claude",
		Command:    command,
		Args:       args,
		WorkingDir: workingDir,
		OnOutput: func(data []byte) {
			// Could log output here if needed
		},
	}

	session, err := pty.NewSession(config)
	if err != nil {
		return fmt.Errorf("failed to create Claude session: %w", err)
	}

	p.session = session
	p.automator = pty.NewAutomator(session)

	// Wait for Claude to be ready
	// Claude Code shows different patterns
	readyPatterns := []string{
		"Type /help", // Part of Claude Code's welcome message
		"shortcuts",  // "? for shortcuts"
		"> Try",      // The input prompt line
		"│ >",        // Alternative prompt format
		"Claude>",    // Legacy pattern
		"Human:",     // Legacy pattern
	}

	// First check if there's an authentication error
	time.Sleep(500 * time.Millisecond) // Give Claude a moment to start
	output := string(p.session.GetOutput())

	// Check for common error messages
	if strings.Contains(output, "Invalid API key") ||
		strings.Contains(output, "Fix external API key") ||
		strings.Contains(output, "Authentication") ||
		strings.Contains(output, "unauthorized") {
		return fmt.Errorf("Claude authentication error: %s", output)
	}

	if err := p.automator.WaitForReady(ctx, readyPatterns); err != nil {
		// Include the actual output in the error for debugging
		currentOutput := string(p.session.GetOutput())
		return fmt.Errorf("Claude did not become ready: %w\nOutput: %s", err, currentOutput)
	}

	return nil
}

// SendPrompt sends a prompt to Claude and waits for the response
func (p *ClaudePTYProvider) SendPrompt(ctx context.Context, prompt string) (string, error) {
	if p.session == nil {
		return "", fmt.Errorf("session not started")
	}

	// Clear any previous output
	p.session.ClearOutput()

	// Send the prompt using copy/paste method
	if err := p.automator.SendPromptWithCopy(ctx, prompt); err != nil {
		return "", fmt.Errorf("failed to send prompt: %w", err)
	}

	// Wait for Claude to finish responding
	// Look for patterns that indicate Claude is ready for the next input
	responseComplete := []string{
		"> Try",     // Claude Code's prompt
		"│ >",       // Alternative prompt format
		"shortcuts", // "? for shortcuts"
		"Human:",    // Legacy pattern
		"Claude>",   // Legacy pattern
	}

	// Give Claude time to start responding
	time.Sleep(1 * time.Second)

	// Capture output until we see a completion pattern
	output, err := p.automator.CaptureOutput(ctx, strings.Join(responseComplete, "|"), 5*time.Minute)
	if err != nil {
		// Even on timeout, return what we have
		output = string(p.session.GetOutput())
	}

	// Extract just the assistant's response
	response := p.automator.ExtractLastResponse(output, "claude")

	return response, nil
}

// StopSession stops the current Claude session
func (p *ClaudePTYProvider) StopSession() error {
	if p.session == nil {
		return nil
	}

	// Send exit command
	p.session.SendKeys("exit")
	p.session.SendEnter()

	// Give it a moment to process
	time.Sleep(500 * time.Millisecond)

	// Close the session
	return p.session.Close()
}

// IsReady checks if Claude is ready for input
func (p *ClaudePTYProvider) IsReady() bool {
	if p.session == nil {
		return false
	}

	output := string(p.session.GetOutput())
	readyPatterns := []string{"Human:", "Claude>", ">>>"}

	for _, pattern := range readyPatterns {
		if strings.Contains(output, pattern) {
			return true
		}
	}

	return false
}
