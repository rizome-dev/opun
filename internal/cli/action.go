package cli

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
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/rizome-dev/opun/internal/tools"
	"github.com/rizome-dev/opun/pkg/core"
	"github.com/spf13/cobra"
)

// actionCmd represents the action command
var actionCmd = &cobra.Command{
	Use:     "action",
	Aliases: []string{"actions"},
	Short:   "Manage actions (simple command wrappers)",
	Long: `Actions are lightweight, declarative definitions that wrap commands, workflows, or prompts.
They provide a simple way to expose functionality without runtime overhead.`,
}

// listActionsCmd lists all available actions
var listActionsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available actions",
	RunE: func(cmd *cobra.Command, args []string) error {
		provider, _ := cmd.Flags().GetString("provider")
		category, _ := cmd.Flags().GetString("category")

		return listActions(provider, category)
	},
}

// addActionCmd adds a new action
var addActionCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add a new action",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		description, _ := cmd.Flags().GetString("description")
		category, _ := cmd.Flags().GetString("category")
		command, _ := cmd.Flags().GetString("command")
		workflow, _ := cmd.Flags().GetString("workflow")
		prompt, _ := cmd.Flags().GetString("prompt")

		return addAction(name, description, category, command, workflow, prompt)
	},
}

// removeActionCmd removes an action
var removeActionCmd = &cobra.Command{
	Use:   "remove [id]",
	Short: "Remove an action",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return removeAction(args[0])
	},
}

// runActionCmd runs an action
var runActionCmd = &cobra.Command{
	Use:   "run [id] [args...]",
	Short: "Run an action",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		actionID := args[0]
		actionArgs := ""
		if len(args) > 1 {
			actionArgs = strings.Join(args[1:], " ")
		}

		return runAction(actionID, actionArgs)
	},
}

// testActionCmd tests an action
var testActionCmd = &cobra.Command{
	Use:   "test [file]",
	Short: "Test an action definition file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return testAction(args[0])
	},
}

func init() {
	actionCmd.AddCommand(listActionsCmd)
	actionCmd.AddCommand(addActionCmd)
	actionCmd.AddCommand(removeActionCmd)
	actionCmd.AddCommand(runActionCmd)
	actionCmd.AddCommand(testActionCmd)

	// List flags
	listActionsCmd.Flags().StringP("provider", "p", "", "Filter by provider")
	listActionsCmd.Flags().StringP("category", "c", "", "Filter by category")

	// Add flags
	addActionCmd.Flags().StringP("description", "d", "", "Action description")
	addActionCmd.Flags().StringP("category", "c", "general", "Action category")
	addActionCmd.Flags().String("command", "", "Command to execute")
	addActionCmd.Flags().String("workflow", "", "Workflow to reference")
	addActionCmd.Flags().String("prompt", "", "Prompt to reference")
	addActionCmd.MarkFlagsMutuallyExclusive("command", "workflow", "prompt")
}

func listActions(provider, category string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	actionsDir := filepath.Join(homeDir, ".opun", "actions")
	loader := tools.NewLoader(actionsDir)

	if err := loader.LoadAll(); err != nil {
		return fmt.Errorf("failed to load actions: %w", err)
	}

	registry := loader.GetRegistry()
	actions := registry.List(provider)

	// Filter by category if specified
	if category != "" {
		filtered := []core.StandardAction{}
		for _, action := range actions {
			if action.Category == category {
				filtered = append(filtered, action)
			}
		}
		actions = filtered
	}

	if len(actions) == 0 {
		fmt.Println("No actions found")
		return nil
	}

	// Display actions in a table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tNAME\tCATEGORY\tTYPE\tDESCRIPTION")
	fmt.Fprintln(w, "---\t----\t--------\t----\t-----------")

	for _, action := range actions {
		actionType := "command"
		if action.WorkflowRef != "" {
			actionType = "workflow"
		} else if action.PromptRef != "" {
			actionType = "prompt"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			action.ID,
			action.Name,
			action.Category,
			actionType,
			truncate(action.Description, 50),
		)
	}

	w.Flush()
	return nil
}

func addAction(name, description, category, command, workflow, prompt string) error {
	// Validate inputs
	if command == "" && workflow == "" && prompt == "" {
		return fmt.Errorf("must specify one of: --command, --workflow, or --prompt")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	actionsDir := filepath.Join(homeDir, ".opun", "actions")
	loader := tools.NewLoader(actionsDir)

	// Create action
	action := core.StandardAction{
		ID:          name,
		Name:        name,
		Description: description,
		Category:    category,
		Version:     "1.0.0",
		Command:     command,
		WorkflowRef: workflow,
		PromptRef:   prompt,
	}

	// Save action
	if err := loader.SaveAction(action); err != nil {
		return fmt.Errorf("failed to save action: %w", err)
	}

	fmt.Printf("Successfully created action '%s'\n", name)
	return nil
}

func removeAction(actionID string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	actionsDir := filepath.Join(homeDir, ".opun", "actions")
	loader := tools.NewLoader(actionsDir)

	if err := loader.DeleteAction(actionID); err != nil {
		return fmt.Errorf("failed to remove action: %w", err)
	}

	fmt.Printf("Successfully removed action '%s'\n", actionID)
	return nil
}

func runAction(actionID, args string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	actionsDir := filepath.Join(homeDir, ".opun", "actions")
	loader := tools.NewLoader(actionsDir)

	if err := loader.LoadAll(); err != nil {
		return fmt.Errorf("failed to load actions: %w", err)
	}

	registry := loader.GetRegistry()
	action, err := registry.Get(actionID)
	if err != nil {
		return fmt.Errorf("action not found: %w", err)
	}

	// Execute based on type
	if action.Command != "" {
		fmt.Printf("Executing command: %s %s\n", action.Command, args)
		// In a real implementation, would execute the command
		fmt.Println("(Command execution would happen here)")
	} else if action.WorkflowRef != "" {
		fmt.Printf("Running workflow: %s with args: %s\n", action.WorkflowRef, args)
		// In a real implementation, would trigger the workflow
		fmt.Println("(Workflow execution would happen here)")
	} else if action.PromptRef != "" {
		fmt.Printf("Executing prompt: %s with args: %s\n", action.PromptRef, args)
		// In a real implementation, would execute the prompt
		fmt.Println("(Prompt execution would happen here)")
	}

	return nil
}

func testAction(file string) error {
	loader := tools.NewLoader("")

	if err := loader.LoadFile(file); err != nil {
		return fmt.Errorf("failed to load action: %w", err)
	}

	fmt.Printf("Action definition is valid: %s\n", file)
	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}
