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
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"

	"github.com/rizome-dev/opun/internal/utils"

	"github.com/creack/pty"
	"golang.org/x/term"
)

// TransparentSession represents a transparent PTY session with an AI provider
type TransparentSession struct {
	provider string
	cmd      *exec.Cmd
	ptmx     *os.File
	mu       sync.Mutex
	closed   bool

	// Optional hooks for intercepting I/O
	onInput  func([]byte) []byte // Transform input before sending
	onOutput func([]byte) []byte // Transform output before displaying

	// Channels for controlled I/O when needed
	inputChan  chan []byte
	outputChan chan []byte

	// For capturing output when needed
	captureOutput bool
	outputBuffer  []byte
}

// TransparentSessionConfig holds configuration for creating a transparent session
type TransparentSessionConfig struct {
	Provider string
	Command  string
	Args     []string
	Env      []string

	// Optional hooks
	OnInput  func([]byte) []byte
	OnOutput func([]byte) []byte

	// Whether to start in capture mode
	CaptureOutput bool
}

// NewTransparentSession creates a new transparent PTY session
func NewTransparentSession(config TransparentSessionConfig) (*TransparentSession, error) {
	cmd := exec.Command(config.Command, config.Args...)

	if len(config.Env) > 0 {
		cmd.Env = append(os.Environ(), config.Env...)
	}

	s := &TransparentSession{
		provider:      config.Provider,
		cmd:           cmd,
		onInput:       config.OnInput,
		onOutput:      config.OnOutput,
		inputChan:     make(chan []byte, 100),
		outputChan:    make(chan []byte, 100),
		captureOutput: config.CaptureOutput,
	}

	// Register for cleanup on shutdown
	utils.RegisterCloser(s)

	return s, nil
}

// Start starts the transparent session
func (s *TransparentSession) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("session already closed")
	}

	// Start command with PTY
	ptmx, err := pty.Start(s.cmd)
	if err != nil {
		return fmt.Errorf("failed to start PTY: %w", err)
	}
	s.ptmx = ptmx

	// Handle PTY size changes
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	go func() {
		for range ch {
			if err := pty.InheritSize(os.Stdin, s.ptmx); err != nil {
				// Ignore resize errors
			}
		}
	}()
	ch <- syscall.SIGWINCH // Initial resize

	// Start I/O goroutines
	go s.handleInput()
	go s.handleOutput()

	return nil
}

// RunInteractive runs the session in fully interactive mode
func (s *TransparentSession) RunInteractive() error {
	// Set stdin to raw mode
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return fmt.Errorf("failed to set raw mode: %w", err)
	}
	defer func() { _ = term.Restore(int(os.Stdin.Fd()), oldState) }()

	// Start the session
	if err := s.Start(); err != nil {
		return err
	}

	// Wait for the process to exit
	return s.cmd.Wait()
}

// handleInput handles input from stdin to PTY
func (s *TransparentSession) handleInput() {
	// Create a scanner for stdin
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := os.Stdin.Read(buf)
			if err != nil {
				if err != io.EOF {
					fmt.Fprintf(os.Stderr, "stdin read error: %v\n", err)
				}
				return
			}

			data := buf[:n]

			// Apply input transformation if configured
			if s.onInput != nil {
				data = s.onInput(data)
			}

			// Write to PTY
			if _, err := s.ptmx.Write(data); err != nil {
				fmt.Fprintf(os.Stderr, "pty write error: %v\n", err)
				return
			}
		}
	}()

	// Also handle programmatic input
	for data := range s.inputChan {
		if _, err := s.ptmx.Write(data); err != nil {
			fmt.Fprintf(os.Stderr, "pty write error: %v\n", err)
			return
		}
	}
}

// handleOutput handles output from PTY to stdout
func (s *TransparentSession) handleOutput() {
	scanner := bufio.NewScanner(s.ptmx)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // Large buffer for tool output

	for scanner.Scan() {
		data := scanner.Bytes()

		// Capture output if needed
		if s.captureOutput {
			s.mu.Lock()
			s.outputBuffer = append(s.outputBuffer, data...)
			s.outputBuffer = append(s.outputBuffer, '\n')
			s.mu.Unlock()
		}

		// Send to output channel for programmatic access
		select {
		case s.outputChan <- data:
		default:
			// Don't block if nobody is reading
		}

		// Apply output transformation if configured
		outputData := data
		if s.onOutput != nil {
			outputData = s.onOutput(data)
		}

		// Write to stdout
		os.Stdout.Write(outputData)
		os.Stdout.Write([]byte{'\n'})
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "pty read error: %v\n", err)
	}
}

// SendInput sends input programmatically
func (s *TransparentSession) SendInput(data []byte) error {
	select {
	case s.inputChan <- data:
		return nil
	default:
		return fmt.Errorf("input channel full")
	}
}

// GetOutput returns captured output
func (s *TransparentSession) GetOutput() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()

	output := make([]byte, len(s.outputBuffer))
	copy(output, s.outputBuffer)
	return output
}

// ClearOutput clears the output buffer
func (s *TransparentSession) ClearOutput() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.outputBuffer = nil
}

// SetCaptureOutput enables or disables output capture
func (s *TransparentSession) SetCaptureOutput(capture bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.captureOutput = capture
}

// Close closes the session
func (s *TransparentSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	s.closed = true
	close(s.inputChan)

	if s.ptmx != nil {
		s.ptmx.Close()
	}

	// Terminate the process if still running
	if s.cmd.Process != nil {
		s.cmd.Process.Signal(syscall.SIGTERM)
		// Give it a moment to exit cleanly
		done := make(chan error, 1)
		go func() {
			done <- s.cmd.Wait()
		}()

		select {
		case <-done:
			// Process exited
		case <-context.Background().Done():
			// Force kill if needed
			s.cmd.Process.Kill()
		}
	}

	return nil
}
