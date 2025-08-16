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
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/rizome-dev/opun/internal/subagent"
	"github.com/rizome-dev/opun/pkg/core"
	subagentpkg "github.com/rizome-dev/opun/pkg/subagent"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	// Global subagent manager instance
	globalSubAgentManager *subagentpkg.Manager
)

// InitSubAgentManager initializes the global subagent manager
func InitSubAgentManager() error {
	if globalSubAgentManager == nil {
		globalSubAgentManager = subagentpkg.NewManager()
		
		// Load subagent configurations from disk
		if err := loadSubAgentConfigs(); err != nil {
			return fmt.Errorf("failed to load subagent configs: %w", err)
		}
	}
	return nil
}

// GetSubAgentManager returns the global subagent manager instance
func GetSubAgentManager() *subagentpkg.Manager {
	if globalSubAgentManager == nil {
		// Initialize if not already done
		if err := InitSubAgentManager(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to initialize subagent manager: %v\n", err)
			return subagentpkg.NewManager() // Return empty manager
		}
	}
	return globalSubAgentManager
}

// SubAgentCmd creates the subagent command with subcommands
func SubAgentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subagent",
		Short: "Manage cross-provider subagents",
		Long: `Manage subagents that can delegate tasks across different AI providers.
Subagents enable sophisticated task delegation and cross-provider coordination.`,
	}

	// Add subcommands
	cmd.AddCommand(
		subAgentListCmd(),
		subAgentCreateCmd(),
		subAgentDeleteCmd(),
		subAgentExecuteCmd(),
		subAgentInfoCmd(),
	)

	return cmd
}

// subAgentListCmd lists all registered subagents
func subAgentListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all registered subagents",
		Long:  `Display a list of all currently registered subagents with their capabilities and status.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr := GetSubAgentManager()
			agents := mgr.List()

			if len(agents) == 0 {
				fmt.Println("No subagents registered.")
				return nil
			}

			// Create table writer
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NAME\tPROVIDER\tSTATUS\tCAPABILITIES\tSTRATEGY")
			fmt.Fprintln(w, "----\t--------\t------\t------------\t--------")

			for _, agent := range agents {
				config := agent.Config()
				caps := strings.Join(agent.GetCapabilities(), ", ")
				if caps == "" {
					caps = "none"
				}
				
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
					agent.Name(),
					agent.Provider(),
					agent.Status(),
					caps,
					config.Strategy,
				)
			}

			return w.Flush()
		},
	}
}

// subAgentCreateCmd creates a new subagent from configuration
func subAgentCreateCmd() *cobra.Command {
	var (
		name         string
		provider     string
		capabilities []string
		strategy     string
		model        string
	)

	cmd := &cobra.Command{
		Use:   "create [config-file]",
		Short: "Create a new subagent",
		Long: `Create a new subagent from a configuration file or using command-line flags.
If a config file is provided, it will be used. Otherwise, flags can specify the configuration.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var agentConfig core.SubAgentConfig

			if len(args) > 0 {
				// Load from config file
				configFile := args[0]
				data, err := os.ReadFile(configFile)
				if err != nil {
					return fmt.Errorf("failed to read config file: %w", err)
				}

				// Try YAML first, then JSON
				if err := yaml.Unmarshal(data, &agentConfig); err != nil {
					if err := json.Unmarshal(data, &agentConfig); err != nil {
						return fmt.Errorf("failed to parse config file (tried YAML and JSON): %w", err)
					}
				}
			} else {
				// Create from flags
				if name == "" || provider == "" {
					return fmt.Errorf("name and provider are required when not using a config file")
				}

				// Parse provider type
				var providerType core.ProviderType
				switch strings.ToLower(provider) {
				case "claude":
					providerType = core.ProviderTypeClaude
				case "gemini":
					providerType = core.ProviderTypeGemini
				case "qwen":
					providerType = core.ProviderTypeQwen
				default:
					return fmt.Errorf("unsupported provider: %s", provider)
				}

				// Parse delegation strategy
				var delegationStrategy core.DelegationStrategy
				switch strings.ToLower(strategy) {
				case "automatic", "":
					delegationStrategy = core.DelegationAutomatic
				case "explicit":
					delegationStrategy = core.DelegationExplicit
				case "proactive":
					delegationStrategy = core.DelegationProactive
				default:
					return fmt.Errorf("unsupported strategy: %s", strategy)
				}

				agentConfig = core.SubAgentConfig{
					Name:         name,
					Provider:     providerType,
					Model:        model,
					Capabilities: capabilities,
					Strategy:     delegationStrategy,
					Settings:     make(map[string]interface{}),
				}
			}

			// Create the subagent using the factory
			factory := subagent.NewFactory()
			agent, err := factory.CreateSubAgent(agentConfig)
			if err != nil {
				return fmt.Errorf("failed to create subagent: %w", err)
			}

			// Register with the manager
			mgr := GetSubAgentManager()
			if err := mgr.Register(agent); err != nil {
				return fmt.Errorf("failed to register subagent: %w", err)
			}

			// Save configuration to disk
			if err := saveSubAgentConfig(agentConfig); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: subagent created but config not saved: %v\n", err)
			}

			fmt.Printf("✅ Subagent '%s' created successfully\n", agent.Name())
			return nil
		},
	}

	// Add flags
	cmd.Flags().StringVarP(&name, "name", "n", "", "Subagent name")
	cmd.Flags().StringVarP(&provider, "provider", "p", "", "Provider type (claude, gemini, qwen)")
	cmd.Flags().StringSliceVarP(&capabilities, "capabilities", "c", nil, "List of capabilities")
	cmd.Flags().StringVarP(&strategy, "strategy", "s", "automatic", "Delegation strategy (automatic, explicit, proactive)")
	cmd.Flags().StringVarP(&model, "model", "m", "", "Model to use")

	return cmd
}

// subAgentDeleteCmd deletes a subagent
func subAgentDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <name>",
		Short: "Delete a subagent",
		Long:  `Remove a subagent from the registry and delete its configuration.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			mgr := GetSubAgentManager()

			// Unregister from manager
			if err := mgr.Unregister(name); err != nil {
				return fmt.Errorf("failed to unregister subagent: %w", err)
			}

			// Delete configuration file
			if err := deleteSubAgentConfig(name); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: subagent unregistered but config file not deleted: %v\n", err)
			}

			fmt.Printf("✅ Subagent '%s' deleted successfully\n", name)
			return nil
		},
	}
}

// subAgentExecuteCmd executes a task on a specific subagent
func subAgentExecuteCmd() *cobra.Command {
	var (
		timeout     int
		taskContext map[string]string
		inputFile   string
		outputFile  string
	)

	cmd := &cobra.Command{
		Use:   "execute <name> <task>",
		Short: "Execute a task on a specific subagent",
		Long: `Execute a task on a named subagent. The task can be provided as a string argument
or loaded from a file using the --input flag.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			var taskContent string

			if inputFile != "" {
				// Load task from file
				data, err := os.ReadFile(inputFile)
				if err != nil {
					return fmt.Errorf("failed to read input file: %w", err)
				}
				taskContent = string(data)
			} else if len(args) > 1 {
				taskContent = args[1]
			} else {
				return fmt.Errorf("task content required (either as argument or via --input)")
			}

			// Create task
			taskCtx := make(map[string]interface{})
			for k, v := range taskContext {
				taskCtx[k] = v
			}

			task := core.SubAgentTask{
				ID:          fmt.Sprintf("task-%d", time.Now().Unix()),
				Name:        "CLI Task",
				Description: taskContent,
				Input:       taskContent,
				Priority:    1,
				Context:     taskCtx,
				Variables:   make(map[string]interface{}),
			}

			// Get manager and execute
			mgr := GetSubAgentManager()
			
			// Create context with timeout
			ctx := cmd.Context()
			if timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
				defer cancel()
			}

			fmt.Printf("🚀 Executing task on subagent '%s'...\n", name)
			
			result, err := mgr.Execute(ctx, task, name)
			if err != nil {
				return fmt.Errorf("task execution failed: %w", err)
			}

			// Display result
			fmt.Printf("\n📊 Task Result:\n")
			fmt.Printf("Status: %s\n", result.Status)
			fmt.Printf("Duration: %s\n", result.EndTime.Sub(result.StartTime))
			
			if result.Output != "" {
				fmt.Printf("\n📝 Output:\n%s\n", result.Output)
				
				// Save to file if requested
				if outputFile != "" {
					if err := os.WriteFile(outputFile, []byte(result.Output), 0644); err != nil {
						fmt.Fprintf(os.Stderr, "Warning: failed to save output to file: %v\n", err)
					} else {
						fmt.Printf("\n💾 Output saved to: %s\n", outputFile)
					}
				}
			}

			if result.Error != nil {
				fmt.Printf("\n❌ Error: %v\n", result.Error)
			}

			return nil
		},
	}

	// Add flags
	cmd.Flags().IntVarP(&timeout, "timeout", "t", 60, "Execution timeout in seconds")
	cmd.Flags().StringToStringVarP(&taskContext, "context", "c", nil, "Context key-value pairs")
	cmd.Flags().StringVarP(&inputFile, "input", "i", "", "Input file containing the task")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file to save results")

	return cmd
}

// subAgentInfoCmd shows detailed information about a subagent
func subAgentInfoCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "info <name>",
		Short: "Show detailed information about a subagent",
		Long:  `Display comprehensive information about a specific subagent including its configuration, capabilities, and current status.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			mgr := GetSubAgentManager()

			agent, err := mgr.Get(name)
			if err != nil {
				return fmt.Errorf("subagent not found: %w", err)
			}

			config := agent.Config()

			fmt.Printf("📋 Subagent Information\n")
			fmt.Printf("=======================\n\n")
			
			fmt.Printf("Name:        %s\n", agent.Name())
			fmt.Printf("Provider:    %s\n", agent.Provider())
			fmt.Printf("Model:       %s\n", config.Model)
			fmt.Printf("Status:      %s\n", agent.Status())
			fmt.Printf("Strategy:    %s\n", config.Strategy)
			
			if len(agent.GetCapabilities()) > 0 {
				fmt.Printf("\n🎯 Capabilities:\n")
				for _, cap := range agent.GetCapabilities() {
					fmt.Printf("  • %s\n", cap)
				}
			}

			if len(config.Settings) > 0 {
				fmt.Printf("\n⚙️  Settings:\n")
				for key, value := range config.Settings {
					fmt.Printf("  • %s: %v\n", key, value)
				}
			}

			if config.SystemPrompt != "" {
				fmt.Printf("\n📝 System Prompt:\n%s\n", config.SystemPrompt)
			}

			// Show active tasks if any
			activeTasks := mgr.ListActiveTasks()
			if len(activeTasks) > 0 {
				fmt.Printf("\n🔄 Active Tasks:\n")
				for _, taskID := range activeTasks {
					status, _ := mgr.GetStatus(taskID)
					fmt.Printf("  • %s [%s]\n", taskID, status)
				}
			}

			return nil
		},
	}
}

// Helper functions for configuration management

func loadSubAgentConfigs() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".opun", "subagents")
	
	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	// Load all config files
	entries, err := os.ReadDir(configDir)
	if err != nil {
		return err
	}

	factory := subagent.NewFactory()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Only process YAML and JSON files
		if !strings.HasSuffix(entry.Name(), ".yaml") && 
		   !strings.HasSuffix(entry.Name(), ".yml") && 
		   !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		configFile := filepath.Join(configDir, entry.Name())
		data, err := os.ReadFile(configFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to read config %s: %v\n", entry.Name(), err)
			continue
		}

		var config core.SubAgentConfig
		if strings.HasSuffix(entry.Name(), ".json") {
			err = json.Unmarshal(data, &config)
		} else {
			err = yaml.Unmarshal(data, &config)
		}

		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to parse config %s: %v\n", entry.Name(), err)
			continue
		}

		// Create and register the subagent
		agent, err := factory.CreateSubAgent(config)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to create subagent from %s: %v\n", entry.Name(), err)
			continue
		}

		if err := globalSubAgentManager.Register(agent); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to register subagent %s: %v\n", config.Name, err)
		}
	}

	return nil
}

func saveSubAgentConfig(config core.SubAgentConfig) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".opun", "subagents")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configFile := filepath.Join(configDir, fmt.Sprintf("%s.yaml", config.Name))
	
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}

	return os.WriteFile(configFile, data, 0644)
}

func deleteSubAgentConfig(name string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	configDir := filepath.Join(home, ".opun", "subagents")
	
	// Try different extensions
	for _, ext := range []string{".yaml", ".yml", ".json"} {
		configFile := filepath.Join(configDir, name+ext)
		if _, err := os.Stat(configFile); err == nil {
			return os.Remove(configFile)
		}
	}

	// Config file not found, but that's okay
	return nil
}