package utils

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
	"os/exec"
	"runtime"
)

// Clipboard provides clipboard operations
type Clipboard interface {
	Copy(text string) error
	Paste() (string, error)
}

// SystemClipboard implements clipboard operations using system commands
type SystemClipboard struct{}

// NewClipboard creates a new clipboard instance
func NewClipboard() Clipboard {
	return &SystemClipboard{}
}

// Copy copies text to the system clipboard
func (c *SystemClipboard) Copy(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		// macOS
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try different Linux clipboard utilities
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return fmt.Errorf("no clipboard utility found (xclip or xsel)")
		}
	case "windows":
		// Windows PowerShell
		cmd = exec.Command("powershell", "-command", "Set-Clipboard", "-Value", text)
		return cmd.Run()
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start clipboard command: %w", err)
	}

	_, err = io.WriteString(stdin, text)
	stdin.Close()

	if err != nil {
		return fmt.Errorf("failed to write to clipboard: %w", err)
	}

	return cmd.Wait()
}

// Paste retrieves text from the system clipboard
func (c *SystemClipboard) Paste() (string, error) {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		// macOS
		cmd = exec.Command("pbpaste")
	case "linux":
		// Try different Linux clipboard utilities
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard", "-out")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--output")
		} else {
			return "", fmt.Errorf("no clipboard utility found (xclip or xsel)")
		}
	case "windows":
		// Windows PowerShell
		cmd = exec.Command("powershell", "-command", "Get-Clipboard")
	default:
		return "", fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to read from clipboard: %w", err)
	}

	return string(output), nil
}
