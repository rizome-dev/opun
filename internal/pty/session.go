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
	"io"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/creack/pty"
)

// Session represents a PTY session with an AI provider
type Session struct {
	provider  string
	cmd       *exec.Cmd
	pty       *os.File
	mu        sync.RWMutex
	closed    bool
	outputBuf []byte
	onOutput  func([]byte)
	readyChan chan bool
	errChan   chan error
}

// SessionConfig holds configuration for creating a PTY session
type SessionConfig struct {
	Provider   string
	Command    string
	Args       []string
	WorkingDir string
	Env        []string
	OnOutput   func([]byte)
}

// NewSession creates a new PTY session
func NewSession(config SessionConfig) (*Session, error) {
	// Execute the command - this is a framework function that accepts commands by design
	// The command should be validated by the caller
	// #nosec G204 -- framework function that executes configured commands
	cmd := exec.Command(config.Command, config.Args...)

	if config.WorkingDir != "" {
		cmd.Dir = config.WorkingDir
	}

	if len(config.Env) > 0 {
		cmd.Env = append(os.Environ(), config.Env...)
	}

	// Create PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, fmt.Errorf("failed to start PTY: %w", err)
	}

	// Set PTY size
	if err := pty.Setsize(ptmx, &pty.Winsize{
		Rows: 40,
		Cols: 120,
	}); err != nil {
		ptmx.Close()
		return nil, fmt.Errorf("failed to set PTY size: %w", err)
	}

	s := &Session{
		provider:  config.Provider,
		cmd:       cmd,
		pty:       ptmx,
		onOutput:  config.OnOutput,
		readyChan: make(chan bool, 1),
		errChan:   make(chan error, 1),
	}

	// Start output monitoring
	go s.monitorOutput()

	return s, nil
}

// Write sends data to the PTY
func (s *Session) Write(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("session is closed")
	}

	_, err := s.pty.Write(data)
	return err
}

// SendKeys sends a string as keyboard input
func (s *Session) SendKeys(keys string) error {
	return s.Write([]byte(keys))
}

// SendEnter sends an enter key press
func (s *Session) SendEnter() error {
	return s.Write([]byte("\r"))
}

// SendPrompt sends a prompt followed by enter
func (s *Session) SendPrompt(prompt string) error {
	if err := s.SendKeys(prompt); err != nil {
		return err
	}
	return s.SendEnter()
}

// GetOutput returns the accumulated output
func (s *Session) GetOutput() []byte {
	s.mu.RLock()
	defer s.mu.RUnlock()
	output := make([]byte, len(s.outputBuf))
	copy(output, s.outputBuf)
	return output
}

// ClearOutput clears the output buffer
func (s *Session) ClearOutput() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.outputBuf = nil
}

// WaitForPattern waits for a specific pattern in the output
func (s *Session) WaitForPattern(ctx context.Context, pattern []byte, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for pattern: %s", pattern)
		case <-ticker.C:
			output := s.GetOutput()
			if ContainsPattern(output, pattern) {
				return nil
			}
		}
	}
}

// Close closes the PTY session
func (s *Session) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true

	// Close PTY
	if err := s.pty.Close(); err != nil {
		return fmt.Errorf("failed to close PTY: %w", err)
	}

	// Wait for command to exit
	if err := s.cmd.Wait(); err != nil {
		// Ignore exit errors for now
		if _, ok := err.(*exec.ExitError); !ok {
			return fmt.Errorf("command failed: %w", err)
		}
	}

	return nil
}

// monitorOutput monitors PTY output
func (s *Session) monitorOutput() {
	buf := make([]byte, 4096)

	for {
		n, err := s.pty.Read(buf)
		if err != nil {
			if err != io.EOF {
				s.errChan <- err
			}
			return
		}

		if n > 0 {
			data := buf[:n]

			s.mu.Lock()
			s.outputBuf = append(s.outputBuf, data...)
			s.mu.Unlock()

			if s.onOutput != nil {
				s.onOutput(data)
			}
		}
	}
}
