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
	"fmt"
	"sort"
	"strings"
	"sync"

	cmdpkg "github.com/rizome-dev/opun/pkg/command"
)

// Registry manages registered commands
type Registry struct {
	mu       sync.RWMutex
	commands map[string]*cmdpkg.Command
	aliases  map[string]string // alias -> command name
}

// NewRegistry creates a new command registry
func NewRegistry() *Registry {
	r := &Registry{
		commands: make(map[string]*cmdpkg.Command),
		aliases:  make(map[string]string),
	}

	// Register built-in commands
	r.registerBuiltinCommands()

	return r
}

// Register registers a new command
func (r *Registry) Register(cmd *cmdpkg.Command) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Validate command
	if err := r.validateCommand(cmd); err != nil {
		return fmt.Errorf("invalid command: %w", err)
	}

	// Check if command already exists
	if _, exists := r.commands[cmd.Name]; exists {
		return fmt.Errorf("command already registered: %s", cmd.Name)
	}

	// Check if any alias conflicts
	for _, alias := range cmd.Aliases {
		if existing, exists := r.aliases[alias]; exists {
			return fmt.Errorf("alias '%s' already used by command '%s'", alias, existing)
		}
		if _, exists := r.commands[alias]; exists {
			return fmt.Errorf("alias '%s' conflicts with existing command", alias)
		}
	}

	// Register command
	r.commands[cmd.Name] = cmd

	// Register aliases
	for _, alias := range cmd.Aliases {
		r.aliases[alias] = cmd.Name
	}

	return nil
}

// Get retrieves a command by name or alias
func (r *Registry) Get(name string) (*cmdpkg.Command, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Check direct command name
	if cmd, exists := r.commands[name]; exists {
		return cmd, true
	}

	// Check aliases
	if cmdName, exists := r.aliases[name]; exists {
		return r.commands[cmdName], true
	}

	return nil, false
}

// List returns all registered commands
func (r *Registry) List() []*cmdpkg.Command {
	r.mu.RLock()
	defer r.mu.RUnlock()

	commands := make([]*cmdpkg.Command, 0, len(r.commands))
	for _, cmd := range r.commands {
		if !cmd.Hidden {
			commands = append(commands, cmd)
		}
	}

	// Sort by name
	sort.Slice(commands, func(i, j int) bool {
		return commands[i].Name < commands[j].Name
	})

	return commands
}

// ListByCategory returns commands grouped by category
func (r *Registry) ListByCategory() map[string][]*cmdpkg.Command {
	r.mu.RLock()
	defer r.mu.RUnlock()

	categories := make(map[string][]*cmdpkg.Command)

	for _, cmd := range r.commands {
		if cmd.Hidden {
			continue
		}

		category := cmd.Category
		if category == "" {
			category = "Other"
		}

		categories[category] = append(categories[category], cmd)
	}

	// Sort commands within each category
	for _, commands := range categories {
		sort.Slice(commands, func(i, j int) bool {
			return commands[i].Name < commands[j].Name
		})
	}

	return categories
}

// Search searches for commands matching a query
func (r *Registry) Search(query string) []*cmdpkg.Command {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query = strings.ToLower(query)
	var matches []*cmdpkg.Command

	for _, cmd := range r.commands {
		if cmd.Hidden {
			continue
		}

		// Check name
		if strings.Contains(strings.ToLower(cmd.Name), query) {
			matches = append(matches, cmd)
			continue
		}

		// Check description
		if strings.Contains(strings.ToLower(cmd.Description), query) {
			matches = append(matches, cmd)
			continue
		}

		// Check aliases
		for _, alias := range cmd.Aliases {
			if strings.Contains(strings.ToLower(alias), query) {
				matches = append(matches, cmd)
				break
			}
		}
	}

	// Sort by relevance (name matches first)
	sort.Slice(matches, func(i, j int) bool {
		iNameMatch := strings.Contains(strings.ToLower(matches[i].Name), query)
		jNameMatch := strings.Contains(strings.ToLower(matches[j].Name), query)

		if iNameMatch && !jNameMatch {
			return true
		}
		if !iNameMatch && jNameMatch {
			return false
		}

		return matches[i].Name < matches[j].Name
	})

	return matches
}

// Remove removes a command from the registry
func (r *Registry) Remove(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cmd, exists := r.commands[name]
	if !exists {
		return fmt.Errorf("command not found: %s", name)
	}

	// Remove aliases
	for _, alias := range cmd.Aliases {
		delete(r.aliases, alias)
	}

	// Remove command
	delete(r.commands, name)

	return nil
}

// validateCommand validates a command definition
func (r *Registry) validateCommand(cmd *cmdpkg.Command) error {
	if cmd.Name == "" {
		return fmt.Errorf("command name is required")
	}

	if cmd.Type == "" {
		return fmt.Errorf("command type is required")
	}

	if cmd.Handler == "" {
		return fmt.Errorf("command handler is required")
	}

	// Validate arguments
	argNames := make(map[string]bool)
	for _, arg := range cmd.Arguments {
		if arg.Name == "" {
			return fmt.Errorf("argument name is required")
		}

		if argNames[arg.Name] {
			return fmt.Errorf("duplicate argument name: %s", arg.Name)
		}
		argNames[arg.Name] = true

		if arg.Type == "" {
			arg.Type = "string" // Default type
		}
	}

	return nil
}

// registerBuiltinCommands registers built-in commands
func (r *Registry) registerBuiltinCommands() {
	// Help command
	helpCmd := &cmdpkg.Command{
		Name:        "help",
		Description: "Show help information",
		Category:    "System",
		Type:        cmdpkg.CommandTypeBuiltin,
		Handler:     "help",
		Arguments: []cmdpkg.Argument{
			{
				Name:        "command",
				Description: "Command to get help for",
				Type:        "string",
				Required:    false,
			},
		},
		Aliases: []string{"h", "?"},
	}
	r.commands["help"] = helpCmd
	for _, alias := range helpCmd.Aliases {
		r.aliases[alias] = helpCmd.Name
	}

	// List command
	listCmd := &cmdpkg.Command{
		Name:        "list",
		Description: "List available commands",
		Category:    "System",
		Type:        cmdpkg.CommandTypeBuiltin,
		Handler:     "list",
		Arguments: []cmdpkg.Argument{
			{
				Name:        "category",
				Description: "Filter by category",
				Type:        "string",
				Required:    false,
			},
		},
		Aliases: []string{"ls"},
	}
	r.commands["list"] = listCmd
	for _, alias := range listCmd.Aliases {
		r.aliases[alias] = listCmd.Name
	}

	// Add command
	addCmd := &cmdpkg.Command{
		Name:        "add",
		Description: "Add a new command",
		Category:    "System",
		Type:        cmdpkg.CommandTypeBuiltin,
		Handler:     "add",
		Arguments: []cmdpkg.Argument{
			{
				Name:        "type",
				Description: "Command type (workflow, plugin)",
				Type:        "string",
				Required:    true,
				Choices:     []string{"workflow", "plugin"},
			},
			{
				Name:        "name",
				Description: "Command name",
				Type:        "string",
				Required:    true,
			},
			{
				Name:        "handler",
				Description: "Handler (workflow or plugin name)",
				Type:        "string",
				Required:    true,
			},
		},
	}
	r.commands["add"] = addCmd

	// Remove command
	removeCmd := &cmdpkg.Command{
		Name:        "remove",
		Description: "Remove a command",
		Category:    "System",
		Type:        cmdpkg.CommandTypeBuiltin,
		Handler:     "remove",
		Arguments: []cmdpkg.Argument{
			{
				Name:        "name",
				Description: "Command name to remove",
				Type:        "string",
				Required:    true,
			},
		},
		Aliases: []string{"rm"},
	}
	r.commands["remove"] = removeCmd
	for _, alias := range removeCmd.Aliases {
		r.aliases[alias] = removeCmd.Name
	}

	// Clear command
	clearCmd := &cmdpkg.Command{
		Name:        "clear",
		Description: "Clear the screen",
		Category:    "System",
		Type:        cmdpkg.CommandTypeBuiltin,
		Handler:     "clear",
		Aliases:     []string{"cls"},
	}
	r.commands["clear"] = clearCmd
	for _, alias := range clearCmd.Aliases {
		r.aliases[alias] = clearCmd.Name
	}

	// Exit command
	exitCmd := &cmdpkg.Command{
		Name:        "exit",
		Description: "Exit the application",
		Category:    "System",
		Type:        cmdpkg.CommandTypeBuiltin,
		Handler:     "exit",
		Aliases:     []string{"quit", "q"},
	}
	r.commands["exit"] = exitCmd
	for _, alias := range exitCmd.Aliases {
		r.aliases[alias] = exitCmd.Name
	}
}
