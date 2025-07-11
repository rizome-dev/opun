package command

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
	"time"
)

// Command represents a slash command that can be executed
type Command struct {
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description" yaml:"description"`
	Category    string                 `json:"category" yaml:"category"`
	Type        CommandType            `json:"type" yaml:"type"`
	Handler     string                 `json:"handler" yaml:"handler"` // workflow name, plugin name, or builtin function
	Arguments   []Argument             `json:"arguments" yaml:"arguments"`
	Aliases     []string               `json:"aliases" yaml:"aliases"`
	Hidden      bool                   `json:"hidden" yaml:"hidden"`
	Metadata    map[string]interface{} `json:"metadata" yaml:"metadata"`
}

// CommandType defines the type of command
type CommandType string

const (
	CommandTypeWorkflow CommandType = "workflow"
	CommandTypePlugin   CommandType = "plugin"
	CommandTypeBuiltin  CommandType = "builtin"
	CommandTypePrompt   CommandType = "prompt"
)

// Argument defines a command argument
type Argument struct {
	Name         string      `json:"name" yaml:"name"`
	Description  string      `json:"description" yaml:"description"`
	Type         string      `json:"type" yaml:"type"` // string, number, boolean, file
	Required     bool        `json:"required" yaml:"required"`
	DefaultValue interface{} `json:"default" yaml:"default"`
	Choices      []string    `json:"choices" yaml:"choices"`
}

// CommandExecution represents an execution of a command
type CommandExecution struct {
	ID          string                 `json:"id"`
	CommandName string                 `json:"command_name"`
	Arguments   map[string]interface{} `json:"arguments"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     *time.Time             `json:"end_time"`
	Status      ExecutionStatus        `json:"status"`
	Output      string                 `json:"output"`
	Error       string                 `json:"error"`
}

// ExecutionStatus represents the status of a command execution
type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "pending"
	StatusRunning   ExecutionStatus = "running"
	StatusCompleted ExecutionStatus = "completed"
	StatusFailed    ExecutionStatus = "failed"
	StatusCancelled ExecutionStatus = "cancelled"
)

// CommandEvent represents an event during command execution
type CommandEvent struct {
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	CommandID string                 `json:"command_id"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// EventType represents the type of command event
type EventType string

const (
	EventCommandStart    EventType = "command_start"
	EventCommandComplete EventType = "command_complete"
	EventCommandError    EventType = "command_error"
	EventCommandOutput   EventType = "command_output"
)
