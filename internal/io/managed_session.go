package io

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
	"strings"
	"sync"
	"time"

	"github.com/rizome-dev/opun/internal/utils"
)

// ManagedSession provides higher-level control over a transparent session
type ManagedSession struct {
	session  *TransparentSession
	provider Provider
	mu       sync.RWMutex

	// State tracking
	isReady      bool
	lastActivity time.Time

	// Output handling
	outputLines []string
	outputMu    sync.Mutex

	// Callbacks
	onReady    func()
	onResponse func(response string)
}

// ManagedSessionConfig configures a managed session
type ManagedSessionConfig struct {
	Provider      Provider
	OnReady       func()
	OnResponse    func(response string)
	CaptureOutput bool
}

// NewManagedSession creates a new managed session
func NewManagedSession(config ManagedSessionConfig) (*ManagedSession, error) {
	m := &ManagedSession{
		provider:     config.Provider,
		onReady:      config.OnReady,
		onResponse:   config.OnResponse,
		outputLines:  make([]string, 0),
		lastActivity: time.Now(),
	}

	// Create transparent session with output hook
	sessionConfig := &TransparentSessionConfig{
		Provider:      config.Provider.Name(),
		Command:       config.Provider.Command(),
		Args:          config.Provider.Args(),
		Env:           config.Provider.Env(),
		CaptureOutput: config.CaptureOutput,
		OnOutput: func(data []byte) []byte {
			m.handleOutput(string(data))
			return data // Pass through unchanged
		},
	}

	session, err := NewTransparentSession(*sessionConfig)
	if err != nil {
		return nil, err
	}

	m.session = session

	// Register for cleanup on shutdown
	utils.RegisterCloser(m)

	return m, nil
}

// Start starts the managed session
func (m *ManagedSession) Start(ctx context.Context) error {
	if err := m.session.Start(); err != nil {
		return err
	}

	// Monitor for readiness
	go m.monitorReadiness(ctx)

	return nil
}

// RunInteractive runs the session in interactive mode
func (m *ManagedSession) RunInteractive() error {
	return m.session.RunInteractive()
}

// SendPrompt sends a prompt and optionally waits for response
func (m *ManagedSession) SendPrompt(prompt string, waitForResponse bool) (string, error) {
	// Clear output tracking
	m.outputMu.Lock()
	m.outputLines = m.outputLines[:0]
	m.outputMu.Unlock()

	// Send the prompt
	if err := m.session.SendInput([]byte(prompt + "\n")); err != nil {
		return "", err
	}

	if !waitForResponse {
		return "", nil
	}

	// Wait for response with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	return m.waitForResponse(ctx)
}

// handleOutput processes output lines
func (m *ManagedSession) handleOutput(line string) {
	m.mu.Lock()
	m.lastActivity = time.Now()
	m.mu.Unlock()

	m.outputMu.Lock()
	defer m.outputMu.Unlock()

	// Store output line
	m.outputLines = append(m.outputLines, line)

	// Check for readiness patterns
	if !m.isReady && m.isReadyPattern(line) {
		m.mu.Lock()
		m.isReady = true
		m.mu.Unlock()

		if m.onReady != nil {
			go m.onReady()
		}
	}

	// Check for response completion
	if m.isResponseComplete(line) && m.onResponse != nil {
		response := m.extractResponse()
		if response != "" {
			go m.onResponse(response)
		}
	}
}

// isReadyPattern checks if the line indicates the provider is ready
func (m *ManagedSession) isReadyPattern(line string) bool {
	readyPatterns := []string{
		"Human:",
		"Assistant:",
		"Claude>",
		"Gemini>",
		">>>",
		">",
	}

	for _, pattern := range readyPatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}

	return false
}

// isResponseComplete checks if the response is complete
func (m *ManagedSession) isResponseComplete(line string) bool {
	// Same patterns indicate completion
	return m.isReadyPattern(line)
}

// extractResponse extracts the last response from output
func (m *ManagedSession) extractResponse() string {
	if len(m.outputLines) < 2 {
		return ""
	}

	var responseLines []string
	foundStart := false

	// Work backwards to find the response
	for i := len(m.outputLines) - 2; i >= 0; i-- {
		line := m.outputLines[i]

		// Skip empty lines at the end
		if !foundStart && strings.TrimSpace(line) == "" {
			continue
		}
		foundStart = true

		// Stop at prompt patterns
		if m.isReadyPattern(line) {
			break
		}

		responseLines = append([]string{line}, responseLines...)
	}

	return strings.Join(responseLines, "\n")
}

// waitForResponse waits for a complete response
func (m *ManagedSession) waitForResponse(ctx context.Context) (string, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	lastLen := 0
	noChangeCount := 0

	for {
		select {
		case <-ctx.Done():
			return m.extractResponse(), ctx.Err()
		case <-ticker.C:
			m.outputMu.Lock()
			currentLen := len(m.outputLines)
			m.outputMu.Unlock()

			// Check if output has stabilized
			if currentLen == lastLen {
				noChangeCount++
				if noChangeCount > 20 { // 2 seconds of no change
					response := m.extractResponse()
					if response != "" {
						return response, nil
					}
				}
			} else {
				noChangeCount = 0
				lastLen = currentLen
			}

			// Check if we see a completion pattern
			m.outputMu.Lock()
			if len(m.outputLines) > 0 {
				lastLine := m.outputLines[len(m.outputLines)-1]
				if m.isResponseComplete(lastLine) {
					m.outputMu.Unlock()
					return m.extractResponse(), nil
				}
			}
			m.outputMu.Unlock()
		}
	}
}

// monitorReadiness monitors for provider readiness
func (m *ManagedSession) monitorReadiness(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	timeout := time.After(10 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return
		case <-timeout:
			// Timeout waiting for readiness
			return
		case <-ticker.C:
			m.mu.RLock()
			ready := m.isReady
			m.mu.RUnlock()

			if ready {
				return
			}
		}
	}
}

// IsReady returns whether the provider is ready
func (m *ManagedSession) IsReady() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.isReady
}

// Close closes the managed session
func (m *ManagedSession) Close() error {
	return m.session.Close()
}
