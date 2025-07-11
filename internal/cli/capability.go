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

	"github.com/rizome-dev/opun/internal/command"
	"github.com/rizome-dev/opun/internal/promptgarden"
	"github.com/rizome-dev/opun/internal/tools"
	"github.com/rizome-dev/opun/internal/workflow"
	pkgcommand "github.com/rizome-dev/opun/pkg/command"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// capabilityCmd represents the capability command
var capabilityCmd = &cobra.Command{
	Use:     "capability",
	Aliases: []string{"capabilities", "cap"},
	Short:   "Manage and list all Opun capabilities",
	Long: `Unified interface for managing all Opun capabilities including:
- Actions (simple command wrappers)
- Tools (MCP tools)
- Workflows (multi-agent orchestrations)
- Prompts (reusable prompt templates)
- Commands (slash commands)`,
}

// listCapabilitiesCmd lists all capabilities
var listCapabilitiesCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available capabilities",
	RunE: func(cmd *cobra.Command, args []string) error {
		capType, _ := cmd.Flags().GetString("type")
		provider, _ := cmd.Flags().GetString("provider")

		return listCapabilities(capType, provider)
	},
}

// searchCapabilitiesCmd searches capabilities
var searchCapabilitiesCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search capabilities by name or description",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		capType, _ := cmd.Flags().GetString("type")

		return searchCapabilities(query, capType)
	},
}

func init() {
	capabilityCmd.AddCommand(listCapabilitiesCmd)
	capabilityCmd.AddCommand(searchCapabilitiesCmd)

	// List flags
	listCapabilitiesCmd.Flags().StringP("type", "t", "all", "Type of capability (all, action, tool, workflow, prompt, command)")
	listCapabilitiesCmd.Flags().StringP("provider", "p", "", "Filter by provider")

	// Search flags
	searchCapabilitiesCmd.Flags().StringP("type", "t", "all", "Type of capability to search")
}

func listCapabilities(capType, provider string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	opunDir := filepath.Join(homeDir, ".opun")

	// Create output table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TYPE\tID\tNAME\tDESCRIPTION")
	fmt.Fprintln(w, "----\t--\t----\t-----------")

	// List actions
	if capType == "all" || capType == "action" {
		actionsDir := filepath.Join(opunDir, "actions")
		actionLoader := tools.NewLoader(actionsDir)

		if err := actionLoader.LoadAll(); err == nil {
			registry := actionLoader.GetRegistry()
			actions := registry.List(provider)

			for _, action := range actions {
				fmt.Fprintf(w, "Action\t%s\t%s\t%s\n",
					action.ID,
					action.Name,
					truncate(action.Description, 50),
				)
			}
		}
	}

	// List tools
	if capType == "all" || capType == "tool" {
		toolsDir := filepath.Join(opunDir, "tools")
		if entries, err := os.ReadDir(toolsDir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".yaml") || strings.HasSuffix(entry.Name(), ".yml")) {
					toolPath := filepath.Join(toolsDir, entry.Name())
					if data, err := os.ReadFile(toolPath); err == nil {
						var toolDef map[string]interface{}
						if err := yaml.Unmarshal(data, &toolDef); err == nil {
							name := ""
							if n, ok := toolDef["name"].(string); ok {
								name = n
							} else {
								name = strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
							}
							
							description := ""
							if d, ok := toolDef["description"].(string); ok {
								description = d
							}
							
							fmt.Fprintf(w, "Tool\t%s\t%s\t%s\n",
								name,
								name,
								truncate(description, 50),
							)
						}
					}
				}
			}
		}
	}

	// List workflows
	if capType == "all" || capType == "workflow" {
		workflowDir := filepath.Join(opunDir, "workflows")
		workflowMgr, err := workflow.NewManager(workflowDir)
		if err != nil {
			return fmt.Errorf("failed to create workflow manager: %w", err)
		}

		workflows, err := workflowMgr.ListWorkflows()
		if err == nil {
			for _, wf := range workflows {
				fmt.Fprintf(w, "Workflow\t%s\t%s\t%s\n",
					wf.Name,
					wf.Name,
					truncate(wf.Description, 50),
				)
			}
		}
	}

	// List prompts
	if capType == "all" || capType == "prompt" {
		promptDir := filepath.Join(opunDir, "promptgarden")
		garden, err := promptgarden.NewGarden(promptDir)
		if err != nil {
			return fmt.Errorf("failed to create prompt garden: %w", err)
		}

		prompts, err := garden.List()
		if err == nil {
			for _, p := range prompts {
				metadata := p.Metadata()
				fmt.Fprintf(w, "Prompt\t%s\t%s\t%s\n",
					p.Name(),
					p.Name(),
					truncate(metadata.Description, 50),
				)
			}
		}
	}

	// List commands
	if capType == "all" || capType == "command" {
		registry := command.NewRegistry()
		// Load default commands
		registry.Register(&pkgcommand.Command{
			Name:        "help",
			Description: "Show help information",
		})
		registry.Register(&pkgcommand.Command{
			Name:        "clear",
			Description: "Clear the screen",
		})
		registry.Register(&pkgcommand.Command{
			Name:        "exit",
			Description: "Exit the application",
		})

		commands := registry.List()
		for _, cmd := range commands {
			fmt.Fprintf(w, "Command\t%s\t/%s\t%s\n",
				cmd.Name,
				cmd.Name,
				truncate(cmd.Description, 50),
			)
		}
	}

	w.Flush()
	return nil
}

func searchCapabilities(query, capType string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	opunDir := filepath.Join(homeDir, ".opun")
	queryLower := strings.ToLower(query)

	// Create output table
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "TYPE\tID\tNAME\tDESCRIPTION")
	fmt.Fprintln(w, "----\t--\t----\t-----------")

	found := 0

	// Search actions
	if capType == "all" || capType == "action" {
		actionsDir := filepath.Join(opunDir, "actions")
		actionLoader := tools.NewLoader(actionsDir)

		if err := actionLoader.LoadAll(); err == nil {
			registry := actionLoader.GetRegistry()
			actions := registry.List("")

			for _, action := range actions {
				if matchesQuery(action.ID, action.Name, action.Description, queryLower) {
					fmt.Fprintf(w, "Action\t%s\t%s\t%s\n",
						action.ID,
						action.Name,
						truncate(action.Description, 50),
					)
					found++
				}
			}
		}
	}

	// Search tools
	if capType == "all" || capType == "tool" {
		toolsDir := filepath.Join(opunDir, "tools")
		if entries, err := os.ReadDir(toolsDir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".yaml") || strings.HasSuffix(entry.Name(), ".yml")) {
					toolPath := filepath.Join(toolsDir, entry.Name())
					if data, err := os.ReadFile(toolPath); err == nil {
						var toolDef map[string]interface{}
						if err := yaml.Unmarshal(data, &toolDef); err == nil {
							name := ""
							if n, ok := toolDef["name"].(string); ok {
								name = n
							} else {
								name = strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
							}
							
							description := ""
							if d, ok := toolDef["description"].(string); ok {
								description = d
							}
							
							if matchesQuery(name, name, description, queryLower) {
								fmt.Fprintf(w, "Tool\t%s\t%s\t%s\n",
									name,
									name,
									truncate(description, 50),
								)
								found++
							}
						}
					}
				}
			}
		}
	}

	// Search workflows
	if capType == "all" || capType == "workflow" {
		workflowDir := filepath.Join(opunDir, "workflows")
		workflowMgr, err := workflow.NewManager(workflowDir)
		if err != nil {
			return fmt.Errorf("failed to create workflow manager: %w", err)
		}

		workflows, err := workflowMgr.ListWorkflows()
		if err == nil {
			for _, wf := range workflows {
				if matchesQuery(wf.Name, wf.Name, wf.Description, queryLower) {
					fmt.Fprintf(w, "Workflow\t%s\t%s\t%s\n",
						wf.Name,
						wf.Name,
						truncate(wf.Description, 50),
					)
					found++
				}
			}
		}
	}

	// Search prompts
	if capType == "all" || capType == "prompt" {
		promptDir := filepath.Join(opunDir, "promptgarden")
		garden, err := promptgarden.NewGarden(promptDir)
		if err != nil {
			return fmt.Errorf("failed to create prompt garden: %w", err)
		}

		prompts, err := garden.List()
		if err == nil {
			for _, p := range prompts {
				metadata := p.Metadata()
				if matchesQuery(p.Name(), p.Name(), metadata.Description, queryLower) {
					fmt.Fprintf(w, "Prompt\t%s\t%s\t%s\n",
						p.Name(),
						p.Name(),
						truncate(metadata.Description, 50),
					)
					found++
				}
			}
		}
	}

	// Search commands
	if capType == "all" || capType == "command" {
		registry := command.NewRegistry()
		// Load default commands
		registry.Register(&pkgcommand.Command{
			Name:        "help",
			Description: "Show help information",
		})
		registry.Register(&pkgcommand.Command{
			Name:        "clear",
			Description: "Clear the screen",
		})
		registry.Register(&pkgcommand.Command{
			Name:        "exit",
			Description: "Exit the application",
		})

		commands := registry.List()
		for _, cmd := range commands {
			if matchesQuery(cmd.Name, cmd.Name, cmd.Description, queryLower) {
				fmt.Fprintf(w, "Command\t%s\t/%s\t%s\n",
					cmd.Name,
					cmd.Name,
					truncate(cmd.Description, 50),
				)
				found++
			}
		}
	}

	w.Flush()

	if found == 0 {
		fmt.Printf("No capabilities found matching '%s'\n", query)
	} else {
		fmt.Printf("\nFound %d capabilities matching '%s'\n", found, query)
	}

	return nil
}

func matchesQuery(id, name, description, query string) bool {
	idLower := strings.ToLower(id)
	nameLower := strings.ToLower(name)
	descLower := strings.ToLower(description)

	return strings.Contains(idLower, query) ||
		strings.Contains(nameLower, query) ||
		strings.Contains(descLower, query)
}
