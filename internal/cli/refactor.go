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
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/rizome-dev/opun/internal/command"
	"github.com/rizome-dev/opun/internal/mcp"
	"github.com/rizome-dev/opun/internal/plugin"
	"github.com/rizome-dev/opun/internal/promptgarden"
	"github.com/rizome-dev/opun/internal/workflow"
	wf "github.com/rizome-dev/opun/pkg/workflow"
	"github.com/spf13/cobra"
)

// RefactorCmd creates the refactor command
func RefactorCmd() *cobra.Command {
	var (
		workflowName string
		outputDir    string
		interactive  bool
		enableMCP    bool
		mcpPort      int
		variables    map[string]string
	)

	cmd := &cobra.Command{
		Use:   "refactor",
		Short: "Run AI-powered refactoring workflows",
		Long: `Run AI-powered refactoring workflows with interactive capabilities.

By default, workflows run in interactive mode, allowing you to engage with the AI
during the refactoring process. MCP server is enabled for Claude to provide
access to your prompt garden as native tools.

Examples:
  opun refactor                              # Interactive workflow selection
  opun refactor --workflow=enhance-code      # Run specific workflow
  opun refactor --interactive=false          # Batch mode execution`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// If no workflow specified, show interactive selection
			if workflowName == "" {
				return runInteractiveWorkflowSelection()
			}

			return runRefactor(workflowName, outputDir, variables, interactive, enableMCP, mcpPort)
		},
	}

	// Flags
	cmd.Flags().StringVarP(&workflowName, "workflow", "w", "", "workflow to run")
	cmd.Flags().StringVarP(&outputDir, "output", "o", "", "output directory for artifacts")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", true, "run in interactive mode")
	cmd.Flags().BoolVar(&enableMCP, "enable-mcp", true, "enable MCP server for prompt garden")
	cmd.Flags().IntVar(&mcpPort, "mcp-port", 8765, "port for MCP server")
	cmd.Flags().StringToStringVarP(&variables, "var", "v", map[string]string{}, "variables to pass to the workflow (key=value)")

	return cmd
}

// runInteractiveWorkflowSelection shows an interactive workflow selector
func runInteractiveWorkflowSelection() error {
	// List available workflows
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	workflowDir := filepath.Join(home, ".opun", "workflows")
	files, err := os.ReadDir(workflowDir)
	if err != nil {
		return fmt.Errorf("failed to read workflows: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No workflows found. Add workflows with 'opun add --workflow'")
		return nil
	}

	// For now, just list them
	fmt.Println("Available workflows:")
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".yaml" {
			name := file.Name()[:len(file.Name())-5] // Remove .yaml extension
			fmt.Printf("  - %s\n", name)
		}
	}

	// TODO: Implement interactive selection with Bubble Tea
	fmt.Println("\nRun with: opun refactor --workflow=<name>")
	return nil
}

// runRefactor executes a refactoring workflow
func runRefactor(name, outputDir string, vars map[string]string, interactive, enableMCP bool, mcpPort int) error {
	ctx := context.Background()

	// Load workflow
	wf, err := loadRefactorWorkflow(name)
	if err != nil {
		return fmt.Errorf("failed to load workflow: %w", err)
	}

	// Create output directory
	if outputDir == "" {
		outputDir = fmt.Sprintf("refactor_%s_%d", name, time.Now().Unix())
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	fmt.Printf("🚀 Starting refactor workflow: %s\n", wf.Name)
	fmt.Printf("📁 Output directory: %s\n", outputDir)
	fmt.Printf("🔧 Interactive mode: %v\n", interactive)
	fmt.Println()

	// Initialize components
	home, _ := os.UserHomeDir()
	gardenPath := filepath.Join(home, ".opun", "promptgarden")
	garden, err := promptgarden.NewGarden(gardenPath)
	if err != nil {
		return fmt.Errorf("failed to initialize prompt garden: %w", err)
	}

	// Start MCP server if enabled
	var mcpServer *mcp.OpunMCPServer
	if enableMCP && hasClaudeAgent(wf) {
		// Initialize command registry (built-ins are loaded automatically)
		cmdRegistry := command.NewRegistry()

		// Initialize plugin manager
		pluginPath := filepath.Join(home, ".opun", "plugins")
		manager := plugin.NewManager(pluginPath)

		// Create unified MCP server
		mcpServer = mcp.NewOpunMCPServer(garden, cmdRegistry, manager, mcpPort)
		if err := mcpServer.Start(ctx); err != nil {
			return fmt.Errorf("failed to start MCP server: %w", err)
		}
		defer mcpServer.Stop(ctx)

		fmt.Printf("🌐 MCP server running on port %d\n", mcpPort)
		time.Sleep(500 * time.Millisecond)
	}

	// Create workflow executor
	executor := workflow.NewExecutor()

	// Convert string vars to interface{}
	variables := make(map[string]interface{})

	// First, populate default values from workflow definition
	for _, v := range wf.Variables {
		if v.DefaultValue != nil {
			variables[v.Name] = v.DefaultValue
		}
	}

	// Then override with user-provided values
	for k, v := range vars {
		variables[k] = v
	}

	// If interactive mode, allow for pausing between phases
	if interactive {
		fmt.Println("💡 Interactive mode enabled. You can interact with agents during execution.")
		fmt.Println("Press Ctrl+C to interrupt at any time.")
		fmt.Println()
	}

	// Execute workflow
	if err := executor.Execute(ctx, wf, variables); err != nil {
		return fmt.Errorf("workflow execution failed: %w", err)
	}

	fmt.Printf("\n✅ Refactor completed successfully!\n")
	fmt.Printf("📁 Artifacts saved to: %s\n", outputDir)

	return nil
}

// hasClaudeAgent checks if workflow uses Claude
func hasClaudeAgent(wf *wf.Workflow) bool {
	for _, agent := range wf.Agents {
		if agent.Provider == "claude" {
			return true
		}
	}
	return false
}

// handleRefactorEvent handles workflow execution events
func handleRefactorEvent(event wf.WorkflowEvent, interactive bool) {
	switch event.Type {
	case wf.EventAgentStart:
		fmt.Printf("\n🤖 Starting agent: %s (%s)\n", event.AgentID, event.Message)

		if interactive {
			// In interactive mode, agents have direct PTY access
			fmt.Println("   💬 [Interactive mode: You can interact with the agent directly]")
		}

	case wf.EventAgentComplete:
		fmt.Printf("✅ Agent completed: %s\n", event.AgentID)

	case wf.EventAgentError:
		fmt.Printf("❌ Agent error: %s - %s\n", event.AgentID, event.Message)

	case wf.EventWorkflowComplete:
		fmt.Printf("\n🎉 Workflow completed in %s\n", event.Message)

	case wf.EventWorkflowError:
		fmt.Printf("\n❌ Workflow error: %s\n", event.Message)

	case wf.EventOutputCreated:
		// In interactive mode, output is shown in real-time through PTY
		// This event is more for logging/tracking
	}
}

// loadRefactorWorkflow loads a workflow by name or path
func loadRefactorWorkflow(name string) (*wf.Workflow, error) {
	var data []byte

	// Check if it's a file path
	if _, err := os.Stat(name); err == nil {
		var readErr error
		data, readErr = os.ReadFile(name)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read workflow file: %w", readErr)
		}
	} else {
		// Load from workflows directory
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}

		workflowPath := filepath.Join(home, ".opun", "workflows", name+".yaml")
		data, err = os.ReadFile(workflowPath)
		if err != nil {
			return nil, fmt.Errorf("workflow '%s' not found", name)
		}
	}

	// Parse workflow
	home, _ := os.UserHomeDir()
	workflowDir := filepath.Join(home, ".opun", "workflows")
	parser := workflow.NewParser(workflowDir)
	return parser.Parse(data)
}
