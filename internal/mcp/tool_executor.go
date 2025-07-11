package mcp

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
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ToolExecutor handles safe execution of tool commands
type ToolExecutor struct {
	workingDir string
	timeout    time.Duration
}

// NewToolExecutor creates a new tool executor
func NewToolExecutor(workingDir string) *ToolExecutor {
	return &ToolExecutor{
		workingDir: workingDir,
		timeout:    30 * time.Second, // Default timeout
	}
}

// ExecuteCommand safely executes a command with arguments
func (te *ToolExecutor) ExecuteCommand(ctx context.Context, command string, args string) (string, error) {
	// Parse command and arguments
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return "", fmt.Errorf("empty command")
	}

	cmdName := parts[0]
	cmdArgs := parts[1:]

	// Add user-provided arguments
	if args != "" {
		cmdArgs = append(cmdArgs, strings.Fields(args)...)
	}

	// Create command with timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, te.timeout)
	defer cancel()

	cmd := exec.CommandContext(timeoutCtx, cmdName, cmdArgs...)
	cmd.Dir = te.workingDir

	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	err := cmd.Run()

	// Build result
	result := stdout.String()
	if stderr.Len() > 0 {
		if result != "" {
			result += "\n"
		}
		result += fmt.Sprintf("Errors:\n%s", stderr.String())
	}

	if err != nil {
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return result, fmt.Errorf("command timed out after %v", te.timeout)
		}
		return result, fmt.Errorf("command failed: %w", err)
	}

	return result, nil
}

// SetTimeout sets the execution timeout
func (te *ToolExecutor) SetTimeout(timeout time.Duration) {
	te.timeout = timeout
}

// ValidateCommand performs basic validation on a command
func (te *ToolExecutor) ValidateCommand(command string) error {
	// Basic validation - check for dangerous patterns
	dangerous := []string{
		"rm -rf /",
		"dd if=/dev/zero",
		":(){ :|:& };:", // Fork bomb
		"> /dev/sda",
		"mkfs.",
	}

	cmdLower := strings.ToLower(command)
	for _, pattern := range dangerous {
		if strings.Contains(cmdLower, strings.ToLower(pattern)) {
			return fmt.Errorf("potentially dangerous command pattern detected: %s", pattern)
		}
	}

	// Check if command starts with allowed prefixes (configurable)
	allowedPrefixes := []string{
		"ls", "grep", "find", "cat", "echo", "pwd", "date",
		"git", "npm", "yarn", "make", "go", "python", "node",
		"rg", "ag", "fd", "bat", "jq", "curl", "wget",
	}

	parts := strings.Fields(command)
	if len(parts) == 0 {
		return fmt.Errorf("empty command")
	}

	cmdBase := parts[0]
	allowed := false
	for _, prefix := range allowedPrefixes {
		if cmdBase == prefix || strings.HasPrefix(cmdBase, prefix+"/") {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("command '%s' is not in the allowed list", cmdBase)
	}

	return nil
}
