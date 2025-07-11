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

	"github.com/rizome-dev/opun/internal/promptgarden"
	"github.com/rizome-dev/opun/internal/tools"
	"github.com/spf13/cobra"
)

// ListCmd creates the list command
func ListCmd() *cobra.Command {
	var (
		listWorkflows bool
		listPrompts   bool
		listActions   bool
		listAll       bool
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available workflows, prompts, and tools",
		Long:  `List all available workflows, prompts, and tools that can be used with Opun.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default to listing all if no specific flag is set
			if !listWorkflows && !listPrompts && !listActions {
				listAll = true
			}

			shown := false

			if listAll || listWorkflows {
				if err := showWorkflows(); err != nil {
					return err
				}
				shown = true
			}

			if listAll || listPrompts {
				if shown {
					fmt.Println() // Add spacing
				}
				if err := showPrompts(); err != nil {
					return err
				}
				shown = true
			}

			if listAll || listActions {
				if shown {
					fmt.Println() // Add spacing
				}
				if err := showActions(); err != nil {
					return err
				}
			}

			return nil
		},
	}

	// Flags
	cmd.Flags().BoolVarP(&listWorkflows, "workflows", "w", false, "List only workflows")
	cmd.Flags().BoolVarP(&listPrompts, "prompts", "p", false, "List only prompts")
	cmd.Flags().BoolVarP(&listActions, "actions", "a", false, "List only actions")

	return cmd
}

// showWorkflows lists all available workflows
func showWorkflows() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	workflowDir := filepath.Join(home, ".opun", "workflows")

	// Check if directory exists
	if _, err := os.Stat(workflowDir); os.IsNotExist(err) {
		fmt.Println("📋 Workflows: (none)")
		return nil
	}

	// List workflow files
	entries, err := os.ReadDir(workflowDir)
	if err != nil {
		return fmt.Errorf("failed to read workflows directory: %w", err)
	}

	fmt.Println("📋 Workflows:")
	fmt.Println(strings.Repeat("-", 50))

	workflowCount := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".yaml")

		// Try to read workflow to get description
		data, err := os.ReadFile(filepath.Join(workflowDir, entry.Name()))
		if err == nil {
			// Simple extraction of description from YAML
			lines := strings.Split(string(data), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "description:") {
					desc := strings.TrimSpace(strings.TrimPrefix(line, "description:"))
					desc = strings.Trim(desc, "\"'")
					fmt.Printf("  /%s - %s\n", name, desc)
					workflowCount++
					break
				}
			}
		} else {
			fmt.Printf("  /%s\n", name)
			workflowCount++
		}
	}

	if workflowCount == 0 {
		fmt.Println("  (none)")
	}

	fmt.Printf("\nTotal: %d workflow(s)\n", workflowCount)
	return nil
}

// showPrompts lists all available prompts
func showPrompts() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	gardenPath := filepath.Join(home, ".opun", "promptgarden")

	// Check if garden exists
	if _, err := os.Stat(gardenPath); os.IsNotExist(err) {
		fmt.Println("🌱 Prompts: (none)")
		return nil
	}

	// Initialize garden
	garden, err := promptgarden.NewGarden(gardenPath)
	if err != nil {
		return fmt.Errorf("failed to access prompt garden: %w", err)
	}

	// List prompts
	prompts, err := garden.ListPrompts()
	if err != nil {
		return fmt.Errorf("failed to list prompts: %w", err)
	}

	fmt.Println("🌱 Prompts:")
	fmt.Println(strings.Repeat("-", 50))

	// Group by category
	categories := make(map[string][]*promptgarden.Prompt)
	for _, prompt := range prompts {
		category := prompt.Metadata.Category
		if category == "" {
			category = "uncategorized"
		}
		categories[category] = append(categories[category], prompt)
	}

	// Display by category
	for category, categoryPrompts := range categories {
		fmt.Printf("\n  %s:\n", strings.Title(category))
		for _, prompt := range categoryPrompts {
			desc := prompt.Metadata.Description
			if desc == "" {
				desc = "No description"
			}
			if len(desc) > 50 {
				desc = desc[:47] + "..."
			}

			fmt.Printf("    promptgarden://%s - %s\n", prompt.ID, desc)

			// Show tags if any
			if len(prompt.Metadata.Tags) > 0 {
				fmt.Printf("      Tags: %s\n", strings.Join(prompt.Metadata.Tags, ", "))
			}
		}
	}

	fmt.Printf("\nTotal: %d prompt(s)\n", len(prompts))
	return nil
}

// showActions lists all available actions
func showActions() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	actionsDir := filepath.Join(home, ".opun", "actions")

	// Check if directory exists
	if _, err := os.Stat(actionsDir); os.IsNotExist(err) {
		fmt.Println("⚡ Actions: (none)")
		return nil
	}

	// Create loader to get actions
	loader := tools.NewLoader(actionsDir)
	if err := loader.LoadAll(); err != nil {
		return fmt.Errorf("failed to load actions: %w", err)
	}

	// Get all actions from registry
	actionList := loader.GetRegistry().List("")

	fmt.Println("⚡ Actions:")
	fmt.Println(strings.Repeat("-", 50))

	if len(actionList) == 0 {
		fmt.Println("  (none)")
	} else {
		// Group by category
		categories := make(map[string][]string)
		for _, action := range actionList {
			category := action.Category
			if category == "" {
				category = "general"
			}

			// Format action info
			var execType string
			if action.Command != "" {
				execType = " [command]"
			} else if action.WorkflowRef != "" {
				execType = " [workflow]"
			} else if action.PromptRef != "" {
				execType = " [prompt]"
			}

			desc := action.Description
			if len(desc) > 40 {
				desc = desc[:37] + "..."
			}

			actionInfo := fmt.Sprintf("  /%s%s - %s", action.ID, execType, desc)
			categories[category] = append(categories[category], actionInfo)
		}

		// Display by category
		for category, toolInfos := range categories {
			fmt.Printf("\n[%s]\n", category)
			for _, info := range toolInfos {
				fmt.Println(info)
			}
		}
	}

	fmt.Printf("\nTotal: %d tool(s)\n", len(actionList))
	return nil
}
