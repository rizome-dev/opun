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
	"strings"
	"time"

	"github.com/rizome-dev/opun/internal/pty"
)

// GeminiPTYProvider handles Gemini-specific PTY interactions
type GeminiPTYProvider struct {
	session   *pty.Session
	automator *pty.Automator
}

// NewGeminiPTYProvider creates a new Gemini PTY provider
func NewGeminiPTYProvider() *GeminiPTYProvider {
	return &GeminiPTYProvider{}
}

// StartSession starts a new Gemini session
func (p *GeminiPTYProvider) StartSession(ctx context.Context, workingDir string) error {
	config := pty.SessionConfig{
		Provider:   "gemini",
		Command:    "gemini",
		Args:       []string{},
		WorkingDir: workingDir,
		OnOutput: func(data []byte) {
			// Could log output here if needed
		},
	}

	session, err := pty.NewSession(config)
	if err != nil {
		return fmt.Errorf("failed to create Gemini session: %w", err)
	}

	p.session = session
	p.automator = pty.NewAutomator(session)

	// Wait for Gemini to be ready
	readyPatterns := []string{
		"Gemini>",
		"gemini>",
		">",
		"$",
	}

	if err := p.automator.WaitForReady(ctx, readyPatterns); err != nil {
		return fmt.Errorf("Gemini did not become ready: %w", err)
	}

	return nil
}

// SendPrompt sends a prompt to Gemini and waits for the response
func (p *GeminiPTYProvider) SendPrompt(ctx context.Context, prompt string) (string, error) {
	if p.session == nil {
		return "", fmt.Errorf("session not started")
	}

	// Clear any previous output
	p.session.ClearOutput()

	// Send the prompt using copy/paste method
	if err := p.automator.SendPromptWithCopy(ctx, prompt); err != nil {
		return "", fmt.Errorf("failed to send prompt: %w", err)
	}

	// Wait for Gemini to finish responding
	// Look for patterns that indicate Gemini is ready for the next input
	responseComplete := []string{
		"Gemini>",
		"gemini>",
		">",
		"$",
	}

	// Give Gemini time to start responding
	time.Sleep(1 * time.Second)

	// Capture output until we see a completion pattern
	output, err := p.automator.CaptureOutput(ctx, strings.Join(responseComplete, "|"), 5*time.Minute)
	if err != nil {
		// Even on timeout, return what we have
		output = string(p.session.GetOutput())
	}

	// Extract just the model's response
	response := p.extractGeminiResponse(output)

	return response, nil
}

// StopSession stops the current Gemini session
func (p *GeminiPTYProvider) StopSession() error {
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

// IsReady checks if Gemini is ready for input
func (p *GeminiPTYProvider) IsReady() bool {
	if p.session == nil {
		return false
	}

	output := string(p.session.GetOutput())
	readyPatterns := []string{"Gemini>", "gemini>", ">", "$"}

	for _, pattern := range readyPatterns {
		if strings.Contains(output, pattern) {
			return true
		}
	}

	return false
}

// extractGeminiResponse extracts the Gemini response from the output
func (p *GeminiPTYProvider) extractGeminiResponse(output string) string {
	lines := strings.Split(output, "\n")

	// Find where the prompt ends and response begins
	promptFound := false
	var responseLines []string

	for _, line := range lines {
		// Skip empty lines at the beginning
		if !promptFound && strings.TrimSpace(line) == "" {
			continue
		}

		// Look for the end of our prompt (it should be echoed back)
		if !promptFound && strings.Contains(line, "```") {
			promptFound = true
			continue
		}

		// Once we've found the prompt, collect response lines
		if promptFound {
			// Stop at prompt indicators
			if strings.Contains(line, "Gemini>") ||
				strings.Contains(line, "gemini>") ||
				strings.HasPrefix(strings.TrimSpace(line), ">") ||
				strings.HasPrefix(strings.TrimSpace(line), "$") {
				break
			}
			responseLines = append(responseLines, line)
		}
	}

	// Join the response lines
	response := strings.Join(responseLines, "\n")

	// Trim any trailing whitespace
	return strings.TrimSpace(response)
}
