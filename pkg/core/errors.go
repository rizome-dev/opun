package core

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

import "errors"

var (
	// Provider errors
	ErrProviderNotFound     = errors.New("provider not found")
	ErrProviderNotSupported = errors.New("provider not supported")
	ErrInvalidConfig        = errors.New("invalid provider configuration")
	ErrProviderInitFailed   = errors.New("provider initialization failed")

	// PTY errors
	ErrPTYCreationFailed = errors.New("failed to create PTY")
	ErrPTYNotReady       = errors.New("PTY session not ready")
	ErrPTYTimeout        = errors.New("PTY operation timed out")
	ErrPTYDisconnected   = errors.New("PTY session disconnected")

	// Session errors
	ErrSessionNotFound = errors.New("session not found")
	ErrSessionExists   = errors.New("session already exists")
	ErrSessionInvalid  = errors.New("invalid session state")

	// Automation errors
	ErrAutomationFailed = errors.New("automation operation failed")
	ErrClipboardFailed  = errors.New("clipboard operation failed")
	ErrPatternNotFound  = errors.New("expected pattern not found")

	// Prompt errors
	ErrPromptNotFound = errors.New("prompt not found")
	ErrPromptInvalid  = errors.New("invalid prompt format")
	ErrTemplateError  = errors.New("template processing error")

	// Tool errors
	ErrToolNotFound        = errors.New("tool not found")
	ErrToolExecutionFailed = errors.New("tool execution failed")

	// Command errors
	ErrCommandNotFound = errors.New("command not found")
	ErrCommandFailed   = errors.New("command execution failed")
	ErrInvalidArgs     = errors.New("invalid command arguments")
)
