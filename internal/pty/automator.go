package pty

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
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/rizome-dev/opun/internal/utils"
)

// Automator handles automated interactions with PTY sessions
type Automator struct {
	session   *Session
	clipboard utils.Clipboard
}

// NewAutomator creates a new automator for a PTY session
func NewAutomator(session *Session) *Automator {
	return &Automator{
		session:   session,
		clipboard: utils.NewClipboard(),
	}
}

// SendPromptWithCopy sends a prompt using copy/paste method
func (a *Automator) SendPromptWithCopy(ctx context.Context, prompt string) error {
	fmt.Fprintf(os.Stderr, "[AUTOMATOR DEBUG] SendPromptWithCopy called with prompt length: %d\n", len(prompt))
	
	// Copy prompt to clipboard
	if err := a.clipboard.Copy(prompt); err != nil {
		return fmt.Errorf("failed to copy to clipboard: %w", err)
	}
	fmt.Fprintf(os.Stderr, "[AUTOMATOR DEBUG] Copied to clipboard successfully\n")

	// Small delay to ensure clipboard is ready
	time.Sleep(100 * time.Millisecond)

	// Send paste command
	if err := a.sendPasteCommand(); err != nil {
		return fmt.Errorf("failed to send paste command: %w", err)
	}
	fmt.Fprintf(os.Stderr, "[AUTOMATOR DEBUG] Sent paste command\n")

	// Wait for the prompt to appear in the output
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	// Wait a bit for the paste to complete
	time.Sleep(500 * time.Millisecond)

	// Send enter
	err := a.session.SendEnter()
	fmt.Fprintf(os.Stderr, "[AUTOMATOR DEBUG] Sent enter key, error: %v\n", err)
	return err
}

// sendPasteCommand sends the appropriate paste command for the OS
func (a *Automator) sendPasteCommand() error {
	var pasteCmd string

	switch runtime.GOOS {
	case "darwin":
		// macOS: Cmd+V
		pasteCmd = "\x16" // Ctrl+V in terminal
	case "linux":
		// Linux: Ctrl+Shift+V in most terminals
		pasteCmd = "\x16" // Ctrl+V
	case "windows":
		// Windows: Ctrl+V
		pasteCmd = "\x16"
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	return a.session.Write([]byte(pasteCmd))
}

// WaitForReady waits for the provider to be ready for input
func (a *Automator) WaitForReady(ctx context.Context, readyPatterns []string) error {
	return a.WaitForReadyWithTimeout(ctx, readyPatterns, 30*time.Second)
}

// WaitForReadyWithTimeout waits for the provider to be ready with a custom timeout
func (a *Automator) WaitForReadyWithTimeout(ctx context.Context, readyPatterns []string, timeout time.Duration) error {
	// Convert patterns to bytes
	patterns := make([][]byte, len(readyPatterns))
	for i, p := range readyPatterns {
		patterns[i] = []byte(p)
	}

	// Wait for any of the patterns
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timer := time.After(timeout)
	lastLen := 0 // DEBUG: Track output length changes

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timer:
			return fmt.Errorf("timeout waiting for ready prompt after %v", timeout)
		case <-ticker.C:
			output := a.session.GetOutput()
			// DEBUG: Show what we're checking periodically (only if output changes)
			if len(output) != lastLen {
				lastLen = len(output)
				fmt.Fprintf(os.Stderr, "[AUTOMATOR DEBUG] Buffer update (len=%d): %q\n", len(output), string(output))
			}
			for _, pattern := range patterns {
				if ContainsPattern(output, pattern) {
					fmt.Fprintf(os.Stderr, "[AUTOMATOR DEBUG] Found pattern: %q\n", string(pattern))
					return nil
				}
			}
		}
	}
}

// CaptureOutput captures output until a specific pattern is found
func (a *Automator) CaptureOutput(ctx context.Context, untilPattern string, timeout time.Duration) (string, error) {
	// Clear existing output
	a.session.ClearOutput()

	// Wait for the pattern
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	pattern := []byte(untilPattern)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var lastOutput []byte

	for {
		select {
		case <-ctx.Done():
			// Return what we have so far
			return string(lastOutput), fmt.Errorf("timeout waiting for pattern: %s", untilPattern)
		case <-ticker.C:
			output := a.session.GetOutput()
			if len(output) > len(lastOutput) {
				lastOutput = output
			}

			if ContainsPattern(output, pattern) {
				// Found the pattern, return the output up to it
				index := indexOfPattern(output, pattern)
				if index >= 0 {
					return string(output[:index]), nil
				}
			}
		}
	}
}

// SendInterrupt sends Ctrl+C to interrupt the current operation
func (a *Automator) SendInterrupt() error {
	return a.session.Write([]byte("\x03"))
}

// SendEOF sends Ctrl+D to signal end of input
func (a *Automator) SendEOF() error {
	return a.session.Write([]byte("\x04"))
}

// ContainsPattern checks if data contains the pattern
func ContainsPattern(data, pattern []byte) bool {
	return indexOfPattern(data, pattern) >= 0
}

// indexOfPattern finds the index of pattern in data
func indexOfPattern(data, pattern []byte) int {
	if len(pattern) == 0 {
		return 0
	}
	if len(data) < len(pattern) {
		return -1
	}

	for i := 0; i <= len(data)-len(pattern); i++ {
		found := true
		for j := 0; j < len(pattern); j++ {
			if data[i+j] != pattern[j] {
				found = false
				break
			}
		}
		if found {
			return i
		}
	}

	return -1
}

// ExtractLastResponse extracts the last AI response from the output
func (a *Automator) ExtractLastResponse(output string, provider string) string {
	lines := strings.Split(output, "\n")

	// Find the last response based on provider-specific patterns
	switch provider {
	case "claude":
		// Look for Claude's response pattern
		inResponse := false
		var response []string

		for i := len(lines) - 1; i >= 0; i-- {
			line := lines[i]

			// Check for response end patterns
			if strings.Contains(line, "Human:") || strings.Contains(line, ">") {
				if inResponse {
					break
				}
			}

			// Check for response start patterns
			if strings.Contains(line, "Assistant:") {
				inResponse = true
				continue
			}

			if inResponse {
				response = append([]string{line}, response...)
			}
		}

		return strings.Join(response, "\n")

	case "gemini":
		// Look for Gemini's response pattern
		// Implement Gemini-specific parsing
		return output

	default:
		// Return the full output for unknown providers
		return output
	}
}
