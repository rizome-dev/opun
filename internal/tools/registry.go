package tools

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
	"sync"

	"github.com/rizome-dev/opun/pkg/core"
)

// Registry is a simple in-memory action registry
type Registry struct {
	mu      sync.RWMutex
	actions map[string]core.StandardAction
}

// NewRegistry creates a new action registry
func NewRegistry() *Registry {
	return &Registry{
		actions: make(map[string]core.StandardAction),
	}
}

// Register adds a new action to the registry
func (r *Registry) Register(action core.StandardAction) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if action.ID == "" {
		return fmt.Errorf("action ID cannot be empty")
	}

	if _, exists := r.actions[action.ID]; exists {
		return fmt.Errorf("action with ID %s already exists", action.ID)
	}

	r.actions[action.ID] = action
	return nil
}

// Get retrieves an action by ID
func (r *Registry) Get(id string) (*core.StandardAction, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	action, exists := r.actions[id]
	if !exists {
		return nil, fmt.Errorf("action with ID %s not found", id)
	}

	return &action, nil
}

// List returns all actions or filtered by provider
func (r *Registry) List(provider string) []core.StandardAction {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []core.StandardAction

	for _, action := range r.actions {
		// If provider is specified, filter by it
		if provider != "" {
			if !r.supportsProvider(action, provider) {
				continue
			}
		}
		result = append(result, action)
	}

	return result
}

// Remove deletes an action from the registry
func (r *Registry) Remove(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.actions[id]; !exists {
		return fmt.Errorf("action with ID %s not found", id)
	}

	delete(r.actions, id)
	return nil
}

// supportsProvider checks if an action supports a specific provider
func (r *Registry) supportsProvider(action core.StandardAction, provider string) bool {
	// If no providers specified, it supports all
	if len(action.Providers) == 0 {
		return true
	}

	// Check if provider is in the list
	for _, p := range action.Providers {
		if p == provider {
			return true
		}
	}

	return false
}

// LoadBuiltinActions registers common built-in actions
func (r *Registry) LoadBuiltinActions() error {
	builtinActions := []core.StandardAction{
		{
			ID:          "list-files",
			Name:        "List Files",
			Description: "List files in a directory",
			Category:    "file",
			Command:     "ls -la",
			Version:     "1.0.0",
		},
		{
			ID:          "search-code",
			Name:        "Search Code",
			Description: "Search for patterns in code files",
			Category:    "search",
			Command:     "rg --type-add 'code:*.{js,ts,go,py,java,rs}' -t code",
			Version:     "1.0.0",
		},
		{
			ID:          "run-tests",
			Name:        "Run Tests",
			Description: "Run tests in the current project",
			Category:    "development",
			Command:     "make test",
			Version:     "1.0.0",
		},
	}

	for _, action := range builtinActions {
		if err := r.Register(action); err != nil {
			return fmt.Errorf("failed to register builtin action %s: %w", action.ID, err)
		}
	}

	return nil
}
