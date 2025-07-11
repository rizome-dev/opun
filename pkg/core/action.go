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

// Tool represents a tool available to a provider
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Version     string                 `json:"version"`
	Enabled     bool                   `json:"enabled"`
	Parameters  []ToolParameter        `json:"parameters"`
	Config      map[string]interface{} `json:"config"`
}

// ToolParameter represents a parameter for a tool
type ToolParameter struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	Description  string      `json:"description"`
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"default_value"`
	Validation   string      `json:"validation"`
}

// StandardAction represents an action that can work across providers
type StandardAction struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Category    string `json:"category"`
	Version     string `json:"version"`

	// Execution details - pick one
	Command     string `json:"command,omitempty"`      // Direct command to execute
	WorkflowRef string `json:"workflow_ref,omitempty"` // Reference to a workflow
	PromptRef   string `json:"prompt_ref,omitempty"`   // Reference to a prompt

	// Provider support
	Providers []string `json:"providers,omitempty"` // Empty means all providers
}

// ActionRegistry manages standardized actions
type ActionRegistry interface {
	// Register a new action
	Register(action StandardAction) error

	// Get an action by ID
	Get(id string) (*StandardAction, error)

	// List all actions or filtered by provider
	List(provider string) []StandardAction

	// Remove an action
	Remove(id string) error
}
