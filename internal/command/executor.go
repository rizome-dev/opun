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
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/rizome-dev/opun/internal/workflow"
	cmdpkg "github.com/rizome-dev/opun/pkg/command"
)

// Executor handles command execution
type Executor struct {
	registry         *Registry
	workflowParser   *workflow.Parser
	workflowExecutor *workflow.Executor
	promptGarden     interface {
		Execute(string, map[string]interface{}) (string, error)
	}
	eventChan chan cmdpkg.CommandEvent
}

// NewExecutor creates a new command executor
func NewExecutor(
	registry *Registry,
	workflowParser *workflow.Parser,
	workflowExecutor *workflow.Executor,
	promptGarden interface {
		Execute(string, map[string]interface{}) (string, error)
	},
) *Executor {
	return &Executor{
		registry:         registry,
		workflowParser:   workflowParser,
		workflowExecutor: workflowExecutor,
		promptGarden:     promptGarden,
		eventChan:        make(chan cmdpkg.CommandEvent, 100),
	}
}

// Execute executes a command with the given arguments
func (e *Executor) Execute(ctx context.Context, commandName string, args map[string]interface{}) (*cmdpkg.CommandExecution, error) {
	// Get command
	cmd, exists := e.registry.Get(commandName)
	if !exists {
		return nil, fmt.Errorf("command not found: %s", commandName)
	}

	// Create execution record
	execution := &cmdpkg.CommandExecution{
		ID:          fmt.Sprintf("cmd-%d", time.Now().Unix()),
		CommandName: cmd.Name,
		Arguments:   args,
		StartTime:   time.Now(),
		Status:      cmdpkg.StatusRunning,
	}

	// Send start event
	e.sendEvent(cmdpkg.CommandEvent{
		Type:      cmdpkg.EventCommandStart,
		Timestamp: time.Now(),
		CommandID: execution.ID,
		Message:   fmt.Sprintf("Executing command: %s", cmd.Name),
		Data: map[string]interface{}{
			"command": cmd.Name,
			"args":    args,
		},
	})

	// Execute based on type
	var err error
	switch cmd.Type {
	case cmdpkg.CommandTypeBuiltin:
		err = e.executeBuiltin(ctx, cmd, args, execution)
	case cmdpkg.CommandTypeWorkflow:
		err = e.executeWorkflow(ctx, cmd, args, execution)
	case cmdpkg.CommandTypePlugin:
		err = e.executePlugin(ctx, cmd, args, execution)
	case cmdpkg.CommandTypePrompt:
		err = e.executePrompt(ctx, cmd, args, execution)
	default:
		err = fmt.Errorf("unknown command type: %s", cmd.Type)
	}

	// Update execution status
	endTime := time.Now()
	execution.EndTime = &endTime

	if err != nil {
		execution.Status = cmdpkg.StatusFailed
		execution.Error = err.Error()

		e.sendEvent(cmdpkg.CommandEvent{
			Type:      cmdpkg.EventCommandError,
			Timestamp: time.Now(),
			CommandID: execution.ID,
			Message:   fmt.Sprintf("Command failed: %v", err),
		})
	} else {
		execution.Status = cmdpkg.StatusCompleted

		e.sendEvent(cmdpkg.CommandEvent{
			Type:      cmdpkg.EventCommandComplete,
			Timestamp: time.Now(),
			CommandID: execution.ID,
			Message:   "Command completed successfully",
			Data: map[string]interface{}{
				"duration": execution.EndTime.Sub(execution.StartTime).String(),
			},
		})
	}

	return execution, err
}

// ParseCommand parses a command line into command name and arguments
func (e *Executor) ParseCommand(input string) (string, map[string]interface{}, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", nil, fmt.Errorf("empty command")
	}

	// Handle slash prefix
	if strings.HasPrefix(input, "/") {
		input = input[1:]
	}

	// Split into parts
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return "", nil, fmt.Errorf("empty command")
	}

	commandName := parts[0]
	args := make(map[string]interface{})

	// Get command definition
	cmd, exists := e.registry.Get(commandName)
	if !exists {
		return commandName, args, nil // Let Execute handle unknown command
	}

	// Parse arguments
	argValues := parts[1:]

	// Simple positional argument parsing
	for i, argDef := range cmd.Arguments {
		if i < len(argValues) {
			args[argDef.Name] = argValues[i]
		} else if argDef.Required {
			return "", nil, fmt.Errorf("missing required argument: %s", argDef.Name)
		} else if argDef.DefaultValue != nil {
			args[argDef.Name] = argDef.DefaultValue
		}
	}

	// TODO: Implement more sophisticated argument parsing (flags, named args, etc.)

	return commandName, args, nil
}

// executeBuiltin executes a built-in command
func (e *Executor) executeBuiltin(ctx context.Context, cmd *cmdpkg.Command, args map[string]interface{}, execution *cmdpkg.CommandExecution) error {
	switch cmd.Handler {
	case "help":
		return e.executeHelp(args, execution)
	case "list":
		return e.executeList(args, execution)
	case "add":
		return e.executeAdd(args, execution)
	case "remove":
		return e.executeRemove(args, execution)
	case "clear":
		execution.Output = "\033[2J\033[H" // ANSI clear screen
		return nil
	case "exit":
		// This should be handled by the caller
		execution.Output = "Exiting..."
		return nil
	default:
		return fmt.Errorf("unknown builtin handler: %s", cmd.Handler)
	}
}

// executeHelp executes the help command
func (e *Executor) executeHelp(args map[string]interface{}, execution *cmdpkg.CommandExecution) error {
	cmdName, _ := args["command"].(string)

	if cmdName == "" {
		// General help
		output := "Available commands:\n\n"

		categories := e.registry.ListByCategory()
		for category, commands := range categories {
			output += fmt.Sprintf("%s:\n", category)
			for _, cmd := range commands {
				output += fmt.Sprintf("  /%s", cmd.Name)
				if len(cmd.Aliases) > 0 {
					output += fmt.Sprintf(" (aliases: %s)", strings.Join(cmd.Aliases, ", "))
				}
				output += fmt.Sprintf(" - %s\n", cmd.Description)
			}
			output += "\n"
		}

		output += "Use /help <command> for detailed help on a specific cmdpkg."
		execution.Output = output
	} else {
		// Specific command help
		cmd, exists := e.registry.Get(cmdName)
		if !exists {
			return fmt.Errorf("command not found: %s", cmdName)
		}

		output := fmt.Sprintf("Command: /%s\n", cmd.Name)
		output += fmt.Sprintf("Description: %s\n", cmd.Description)

		if len(cmd.Aliases) > 0 {
			output += fmt.Sprintf("Aliases: %s\n", strings.Join(cmd.Aliases, ", "))
		}

		if len(cmd.Arguments) > 0 {
			output += "\nArguments:\n"
			for _, arg := range cmd.Arguments {
				output += fmt.Sprintf("  %s", arg.Name)
				if arg.Required {
					output += " (required)"
				}
				output += fmt.Sprintf(" - %s\n", arg.Description)

				if arg.Type != "string" {
					output += fmt.Sprintf("    Type: %s\n", arg.Type)
				}

				if len(arg.Choices) > 0 {
					output += fmt.Sprintf("    Choices: %s\n", strings.Join(arg.Choices, ", "))
				}

				if arg.DefaultValue != nil {
					output += fmt.Sprintf("    Default: %v\n", arg.DefaultValue)
				}
			}
		}

		execution.Output = output
	}

	return nil
}

// executeList executes the list command
func (e *Executor) executeList(args map[string]interface{}, execution *cmdpkg.CommandExecution) error {
	category, _ := args["category"].(string)

	var output string

	if category == "" {
		// List all commands by category
		categories := e.registry.ListByCategory()
		for cat, commands := range categories {
			output += fmt.Sprintf("%s:\n", cat)
			for _, cmd := range commands {
				output += fmt.Sprintf("  /%s - %s\n", cmd.Name, cmd.Description)
			}
			output += "\n"
		}
	} else {
		// List commands in specific category
		categories := e.registry.ListByCategory()
		commands, exists := categories[category]
		if !exists {
			return fmt.Errorf("category not found: %s", category)
		}

		output = fmt.Sprintf("%s commands:\n", category)
		for _, cmd := range commands {
			output += fmt.Sprintf("  /%s - %s\n", cmd.Name, cmd.Description)
		}
	}

	execution.Output = output
	return nil
}

// executeAdd executes the add command
func (e *Executor) executeAdd(args map[string]interface{}, execution *cmdpkg.CommandExecution) error {
	cmdType, _ := args["type"].(string)
	name, _ := args["name"].(string)
	handler, _ := args["handler"].(string)

	// Create new command
	newCmd := &cmdpkg.Command{
		Name:        name,
		Description: fmt.Sprintf("%s command: %s", cmdType, handler),
		Category:    "User",
		Type:        cmdpkg.CommandType(cmdType),
		Handler:     handler,
	}

	// Register command
	if err := e.registry.Register(newCmd); err != nil {
		return err
	}

	execution.Output = fmt.Sprintf("Command '/%s' added successfully", name)
	return nil
}

// executeRemove executes the remove command
func (e *Executor) executeRemove(args map[string]interface{}, execution *cmdpkg.CommandExecution) error {
	name, _ := args["name"].(string)

	if err := e.registry.Remove(name); err != nil {
		return err
	}

	execution.Output = fmt.Sprintf("Command '/%s' removed successfully", name)
	return nil
}

// executeWorkflow executes a workflow command
func (e *Executor) executeWorkflow(ctx context.Context, cmd *cmdpkg.Command, args map[string]interface{}, execution *cmdpkg.CommandExecution) error {
	// Load workflow
	wf, err := e.workflowParser.LoadWorkflow(cmd.Handler)
	if err != nil {
		return fmt.Errorf("failed to load workflow: %w", err)
	}

	// Execute workflow
	if err := e.workflowExecutor.Execute(ctx, wf, args); err != nil {
		return fmt.Errorf("workflow execution failed: %w", err)
	}

	execution.Output = fmt.Sprintf("Workflow '%s' executed successfully", cmd.Handler)
	return nil
}

// executePlugin executes a plugin command
func (e *Executor) executePlugin(ctx context.Context, cmd *cmdpkg.Command, args map[string]interface{}, execution *cmdpkg.CommandExecution) error {
	// TODO: Implement plugin execution
	return fmt.Errorf("plugin execution not implemented yet")
}

// executePrompt executes a prompt command from the prompt garden
func (e *Executor) executePrompt(ctx context.Context, cmd *cmdpkg.Command, args map[string]interface{}, execution *cmdpkg.CommandExecution) error {
	if e.promptGarden == nil {
		return fmt.Errorf("prompt garden not available")
	}

	// Execute the prompt using the prompt ID stored in the handler
	result, err := e.promptGarden.Execute(cmd.Handler, args)
	if err != nil {
		return fmt.Errorf("failed to execute prompt: %w", err)
	}

	execution.Output = result
	return nil
}

// sendEvent sends a command event
func (e *Executor) sendEvent(event cmdpkg.CommandEvent) {
	select {
	case e.eventChan <- event:
	default:
		// Channel full, drop event
	}
}

// GetEventChannel returns the event channel for monitoring
func (e *Executor) GetEventChannel() <-chan cmdpkg.CommandEvent {
	return e.eventChan
}
