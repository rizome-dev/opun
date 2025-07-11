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
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/rizome-dev/opun/internal/command"
	"github.com/rizome-dev/opun/internal/mcp"
	"github.com/rizome-dev/opun/internal/plugin"
	"github.com/rizome-dev/opun/internal/promptgarden"
	"github.com/rizome-dev/opun/internal/tools"
	"github.com/rizome-dev/opun/internal/workflow"
	"github.com/spf13/cobra"
)

// MCPCmd creates the MCP command
func MCPCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "MCP server commands",
		Long:  `Commands for running MCP (Model Context Protocol) servers that integrate with AI providers.`,
	}

	cmd.AddCommand(
		mcpServeCmd(),
		mcpStdioCmd(),
	)

	return cmd
}

// mcpServeCmd creates the serve command for unified Opun MCP
func mcpServeCmd() *cobra.Command {
	var port int

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run the Opun MCP server",
		Long: `Starts a unified MCP server that exposes all Opun capabilities:
- Prompts from the PromptGarden
- Slash commands
- Plugins and tools

This server can be used by Claude, Gemini, and other MCP-compatible clients.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initialize components
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}

			// Initialize prompt garden
			gardenPath := filepath.Join(home, ".opun", "promptgarden")
			garden, err := promptgarden.NewGarden(gardenPath)
			if err != nil {
				return fmt.Errorf("failed to initialize prompt garden: %w", err)
			}

			// Initialize command registry (built-ins are loaded automatically)
			registry := command.NewRegistry()

			// Initialize plugin manager
			pluginPath := filepath.Join(home, ".opun", "plugins")
			manager := plugin.NewManager(pluginPath)

			// Create unified server
			server := mcp.NewOpunMCPServer(garden, registry, manager, port)

			// Handle graceful shutdown
			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

			go func() {
				<-sigChan
				fmt.Println("\nShutting down MCP server...")
				os.Exit(0)
			}()

			fmt.Printf("Starting Opun MCP server on port %d...\n", port)
			ctx := context.Background()
			return server.Start(ctx)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "p", 3000, "Port to run the MCP server on")

	return cmd
}

// mcpStdioCmd creates the stdio serve command for MCP
func mcpStdioCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stdio",
		Short: "Run the Opun MCP server in stdio mode",
		Long: `Starts a stdio-based MCP server that can be used by Gemini and other providers.

This server communicates via stdin/stdout using the MCP protocol and exposes:
- Workflows from ~/.opun/workflows
- Prompts from the PromptGarden
- Built-in commands
- Plugins and tools

To use with Gemini, add this to ~/.gemini/settings.json:
{
  "mcpServers": {
    "opun": {
      "command": "opun",
      "args": ["mcp", "stdio"]
    }
  }
}`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initialize components
			home, err := os.UserHomeDir()
			if err != nil {
				return err
			}

			// Initialize prompt garden
			gardenPath := filepath.Join(home, ".opun", "promptgarden")
			garden, err := promptgarden.NewGarden(gardenPath)
			if err != nil {
				// Log to stderr since stdout is used for MCP protocol
				fmt.Fprintf(os.Stderr, "Warning: failed to initialize prompt garden: %v\n", err)
			}

			// Initialize command registry
			registry := command.NewRegistry()

			// Initialize plugin manager
			pluginPath := filepath.Join(home, ".opun", "plugins")
			manager := plugin.NewManager(pluginPath)

			// Initialize workflow manager
			workflowPath := filepath.Join(home, ".opun", "workflows")
			workflowMgr, err := workflow.NewManager(workflowPath)
			if err != nil {
				// Log to stderr
				fmt.Fprintf(os.Stderr, "Warning: failed to initialize workflow manager: %v\n", err)
			}

			// Initialize tool registry
			toolsPath := filepath.Join(home, ".opun", "tools")
			toolLoader := tools.NewLoader(toolsPath)
			if err := toolLoader.LoadAll(); err != nil {
				// Log to stderr
				fmt.Fprintf(os.Stderr, "Warning: failed to load tools: %v\n", err)
			}
			toolRegistry := toolLoader.GetRegistry()

			// Create stdio server
			server := mcp.NewStdioMCPServer(garden, registry, manager, workflowMgr, toolRegistry)

			// Setup signal handling
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			sigChan := make(chan os.Signal, 1)
			signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
			go func() {
				<-sigChan
				cancel()
			}()

			// Run the server
			return server.Run(ctx)
		},
	}

	return cmd
}
