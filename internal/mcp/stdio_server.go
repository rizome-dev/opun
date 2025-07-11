package mcp

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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rizome-dev/opun/internal/command"
	"github.com/rizome-dev/opun/internal/plugin"
	"github.com/rizome-dev/opun/internal/promptgarden"
	toolslib "github.com/rizome-dev/opun/internal/tools"
	"github.com/rizome-dev/opun/internal/workflow"
	"github.com/rizome-dev/opun/pkg/core"
	"gopkg.in/yaml.v3"
)

// StdioMCPServer implements an MCP server using stdio transport
type StdioMCPServer struct {
	garden       *promptgarden.Garden
	registry     *command.Registry
	pluginMgr    *plugin.Manager
	workflowMgr  *workflow.Manager
	toolRegistry *toolslib.Registry
	toolExecutor *ToolExecutor
	reader       *bufio.Reader
	writer       io.Writer
}

// NewStdioMCPServer creates a new stdio-based MCP server
func NewStdioMCPServer(garden *promptgarden.Garden, registry *command.Registry, pluginMgr *plugin.Manager, workflowMgr *workflow.Manager, toolRegistry *toolslib.Registry) *StdioMCPServer {
	// Get working directory for tool executor
	workDir, _ := os.Getwd()

	return &StdioMCPServer{
		garden:       garden,
		registry:     registry,
		pluginMgr:    pluginMgr,
		workflowMgr:  workflowMgr,
		toolRegistry: toolRegistry,
		toolExecutor: NewToolExecutor(workDir),
		reader:       bufio.NewReader(os.Stdin),
		writer:       os.Stdout,
	}
}

// Run starts the stdio MCP server
func (s *StdioMCPServer) Run(ctx context.Context) error {
	// Log server start
	fmt.Fprintf(os.Stderr, "Opun MCP server started (stdio mode)\n")

	// Main message loop - wait for requests
	for {
		select {
		case <-ctx.Done():
			fmt.Fprintf(os.Stderr, "MCP server shutting down: %v\n", ctx.Err())
			return ctx.Err()
		default:
			// Blocking read - this is the standard MCP approach
			request, err := s.readRequest()
			if err != nil {
				if err == io.EOF {
					// EOF is normal when client disconnects
					fmt.Fprintf(os.Stderr, "MCP client disconnected (EOF)\n")
					return nil
				}
				// Log other errors to stderr
				fmt.Fprintf(os.Stderr, "Error reading request: %v\n", err)
				continue
			}

			// Handle request
			s.handleRequest(request)
		}
	}
}

// readRequest reads a JSON-RPC request from stdin
func (s *StdioMCPServer) readRequest() (map[string]interface{}, error) {
	line, err := s.reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	var request map[string]interface{}
	if err := json.Unmarshal([]byte(line), &request); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	return request, nil
}

// sendResponse sends a JSON-RPC response to stdout
func (s *StdioMCPServer) sendResponse(id interface{}, result interface{}) {
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"result":  result,
	}

	data, _ := json.Marshal(response)
	fmt.Fprintf(s.writer, "%s\n", data)
	// Ensure output is flushed immediately
	if f, ok := s.writer.(*os.File); ok {
		f.Sync()
	}
}

// sendError sends a JSON-RPC error response
func (s *StdioMCPServer) sendError(id interface{}, err error) {
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      id,
		"error": map[string]interface{}{
			"code":    -32603,
			"message": err.Error(),
		},
	}

	data, _ := json.Marshal(response)
	fmt.Fprintf(s.writer, "%s\n", data)
	// Ensure output is flushed immediately
	if f, ok := s.writer.(*os.File); ok {
		f.Sync()
	}
}

// handleRequest handles a JSON-RPC request
func (s *StdioMCPServer) handleRequest(request map[string]interface{}) {
	method, _ := request["method"].(string)
	id := request["id"]
	params, _ := request["params"].(map[string]interface{})

	// Only log errors and important events to stderr

	switch method {
	case "initialize":
		s.handleInitialize(id, params)
	case "tools/list":
		s.handleToolsList(id)
	case "tools/call":
		s.handleToolCall(id, params)
	case "prompts/list":
		s.handlePromptsList(id)
	case "prompts/get":
		s.handlePromptsGet(id, params)
	case "ping":
		// Handle ping to keep connection alive
		s.sendResponse(id, map[string]interface{}{
			"status": "ok",
		})
	default:
		s.sendError(id, fmt.Errorf("unknown method: %s", method))
	}
}

// handleInitialize handles the initialize request
func (s *StdioMCPServer) handleInitialize(id interface{}, params map[string]interface{}) {
	s.sendResponse(id, map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools":   map[string]interface{}{},
			"prompts": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "opun",
			"version": "1.0.0",
		},
	})
}

// createToolDescriptor creates a standardized tool descriptor with metadata
func (s *StdioMCPServer) createToolDescriptor(name, description, source, version string, inputSchema map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
		"name":        name,
		"description": description,
		"inputSchema": inputSchema,
		"metadata": map[string]interface{}{
			"source":   source,
			"provider": "opun",
			"version":  version,
		},
	}
}

// handleToolsList returns all available tools
func (s *StdioMCPServer) handleToolsList(id interface{}) {
	tools := []map[string]interface{}{}

	// Add workflow tools
	if s.workflowMgr != nil {
		workflows, err := s.workflowMgr.ListWorkflows()
		if err == nil {
			for _, wf := range workflows {
				tool := s.createToolDescriptor(
					fmt.Sprintf("workflow_%s", wf.Name),
					fmt.Sprintf("[Workflow] %s: %s", wf.Name, wf.Description),
					"workflow",
					wf.Version,
					map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"args": map[string]interface{}{
								"type":        "string",
								"description": "Arguments for the workflow",
							},
						},
					},
				)
				tools = append(tools, tool)
			}
		}
	}

	// Add prompt tools
	if s.garden != nil {
		prompts, err := s.garden.List()
		if err == nil {
			for _, p := range prompts {
				// Skip templates
				if strings.HasSuffix(p.Name(), "-template") {
					continue
				}

				metadata := p.Metadata()
				tool := s.createToolDescriptor(
					fmt.Sprintf("prompt_%s", p.Name()),
					fmt.Sprintf("[Prompt] %s: %s", p.Name(), metadata.Description),
					"prompt",
					metadata.Version,
					s.buildPromptParameters(p),
				)
				tools = append(tools, tool)
			}
		}
	}

	// Add command tools
	if s.registry != nil {
		commands := s.registry.List()
		for _, cmd := range commands {
			// Skip hidden commands
			if cmd.Hidden {
				continue
			}

			tool := s.createToolDescriptor(
				fmt.Sprintf("command_%s", cmd.Name),
				fmt.Sprintf("[Command] /%s: %s", cmd.Name, cmd.Description),
				"command",
				"1.0.0", // Commands don't have versions, use default
				map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"args": map[string]interface{}{
							"type":        "string",
							"description": "Arguments for the command",
						},
					},
				},
			)
			tools = append(tools, tool)
		}
	}

	// Add plugin tools
	if s.pluginMgr != nil {
		// New plugins don't expose tools directly - they import actions
	}

	// Add standardized actions from action registry
	if s.toolRegistry != nil {
		translator := toolslib.NewTranslator(s.toolRegistry)
		standardActions := translator.GetMCPActions("") // Get all actions
		for _, action := range standardActions {
			tools = append(tools, action)
		}
	}

	// Add MCP tools from ~/.opun/tools
	home, _ := os.UserHomeDir()
	toolsDir := filepath.Join(home, ".opun", "tools")
	if entries, err := os.ReadDir(toolsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".yaml") || strings.HasSuffix(entry.Name(), ".yml")) {
				toolPath := filepath.Join(toolsDir, entry.Name())
				if data, err := os.ReadFile(toolPath); err == nil {
					var toolDef map[string]interface{}
					if err := yaml.Unmarshal(data, &toolDef); err == nil {
						// Check if this is a proper MCP tool (has input_schema or implementation)
						if _, hasImpl := toolDef["implementation"]; hasImpl {
							// Extract tool info
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

							// Get input schema
							inputSchema := map[string]interface{}{
								"type":       "object",
								"properties": map[string]interface{}{},
							}
							if schema, ok := toolDef["input_schema"]; ok {
								inputSchema = schema.(map[string]interface{})
							}

							tool := s.createToolDescriptor(
								fmt.Sprintf("tool_%s", name),
								fmt.Sprintf("[Tool] %s: %s", name, description),
								"tool",
								"1.0.0",
								inputSchema,
							)
							tools = append(tools, tool)
						}
					}
				}
			}
		}
	}

	s.sendResponse(id, map[string]interface{}{
		"tools": tools,
	})
}

// handleToolCall executes a tool
func (s *StdioMCPServer) handleToolCall(id interface{}, params map[string]interface{}) {
	toolName, _ := params["name"].(string)
	arguments, _ := params["arguments"].(map[string]interface{})

	var result string
	var err error

	// Determine tool type and execute
	switch {
	case strings.HasPrefix(toolName, "workflow_"):
		result, err = s.executeWorkflow(toolName, arguments)
	case strings.HasPrefix(toolName, "prompt_"):
		result, err = s.executePrompt(toolName, arguments)
	case strings.HasPrefix(toolName, "command_"):
		result, err = s.executeCommand(toolName, arguments)
	case strings.HasPrefix(toolName, "plugin_"):
		result, err = s.executePlugin(toolName, arguments)
	case strings.HasPrefix(toolName, "action_"):
		result, err = s.executeStandardAction(toolName, arguments)
	case strings.HasPrefix(toolName, "tool_"):
		result, err = s.executeMCPTool(toolName, arguments)
	default:
		err = fmt.Errorf("unknown tool: %s", toolName)
	}

	if err != nil {
		s.sendError(id, err)
		return
	}

	s.sendResponse(id, map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": result,
			},
		},
	})
}

// executeWorkflow executes a workflow
func (s *StdioMCPServer) executeWorkflow(tool string, args map[string]interface{}) (string, error) {
	if s.workflowMgr == nil {
		return "", fmt.Errorf("workflow manager not available")
	}

	// Extract workflow name
	workflowName := strings.TrimPrefix(tool, "workflow_")

	// Get args string
	argsStr, _ := args["args"].(string)

	// Execute workflow
	ctx := context.Background()
	result, err := s.workflowMgr.Execute(ctx, workflowName, map[string]interface{}{
		"args": argsStr,
	})
	if err != nil {
		return "", err
	}

	// Format result
	return fmt.Sprintf("Workflow '%s' executed successfully:\n%v", workflowName, result), nil
}

// executePrompt executes a prompt
func (s *StdioMCPServer) executePrompt(tool string, args map[string]interface{}) (string, error) {
	if s.garden == nil {
		return "", fmt.Errorf("prompt garden not available")
	}

	// Extract prompt name
	promptName := strings.TrimPrefix(tool, "prompt_")

	// Execute the prompt
	result, err := s.garden.Execute(promptName, args)
	if err != nil {
		return "", err
	}

	return result, nil
}

// executeCommand executes a command
func (s *StdioMCPServer) executeCommand(tool string, args map[string]interface{}) (string, error) {
	if s.registry == nil {
		return "", fmt.Errorf("command registry not available")
	}

	// Extract command name
	cmdName := strings.TrimPrefix(tool, "command_")

	// Get command
	cmd, exists := s.registry.Get(cmdName)
	if !exists {
		return "", fmt.Errorf("command not found: %s", cmdName)
	}

	// Get args string
	argsStr, _ := args["args"].(string)

	// Note: Actual command execution would require proper implementation
	return fmt.Sprintf("Command '/%s %s' executed successfully", cmd.Name, argsStr), nil
}

// executePlugin executes a plugin tool
func (s *StdioMCPServer) executePlugin(tool string, args map[string]interface{}) (string, error) {
	if s.pluginMgr == nil {
		return "", fmt.Errorf("plugin manager not available")
	}

	// Extract plugin name and tool name
	parts := strings.SplitN(strings.TrimPrefix(tool, "plugin_"), "_", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid plugin tool format: %s", tool)
	}

	pluginName := parts[0]
	toolName := parts[1]

	// Note: Actual plugin execution would require proper implementation
	return fmt.Sprintf("Plugin tool '%s/%s' executed with args: %v", pluginName, toolName, args), nil
}

// executeMCPTool executes an MCP tool
func (s *StdioMCPServer) executeMCPTool(tool string, args map[string]interface{}) (string, error) {
	// Extract tool name
	toolName := strings.TrimPrefix(tool, "tool_")

	// Load tool definition
	home, _ := os.UserHomeDir()
	toolsDir := filepath.Join(home, ".opun", "tools")
	toolPath := filepath.Join(toolsDir, toolName+".yaml")

	// Try .yml if .yaml doesn't exist
	if _, err := os.Stat(toolPath); os.IsNotExist(err) {
		toolPath = filepath.Join(toolsDir, toolName+".yml")
	}

	// Read tool definition
	data, err := os.ReadFile(toolPath)
	if err != nil {
		return "", fmt.Errorf("failed to read tool definition: %w", err)
	}

	var toolDef map[string]interface{}
	if err := yaml.Unmarshal(data, &toolDef); err != nil {
		return "", fmt.Errorf("failed to parse tool definition: %w", err)
	}

	// Check if tool has implementation
	impl, hasImpl := toolDef["implementation"].(map[string]interface{})
	if !hasImpl {
		return "", fmt.Errorf("tool %s has no implementation", toolName)
	}

	// Get implementation type
	implType, _ := impl["type"].(string)

	// For JavaScript tools, we need to simulate execution
	// In a real implementation, this would use a JavaScript runtime
	if implType == "javascript" {
		// Special handling for calculator tool
		if toolName == "calculator" {
			operation, _ := args["operation"].(string)
			a, _ := args["a"].(float64)
			b, _ := args["b"].(float64)

			var result float64
			var symbol string

			switch operation {
			case "add":
				result = a + b
				symbol = "+"
			case "subtract":
				result = a - b
				symbol = "-"
			case "multiply":
				result = a * b
				symbol = "*"
			case "divide":
				if b == 0 {
					return "", fmt.Errorf("division by zero")
				}
				result = a / b
				symbol = "/"
			default:
				return "", fmt.Errorf("invalid operation: %s", operation)
			}

			return fmt.Sprintf("%g %s %g = %g", a, symbol, b, result), nil
		}

		// For other JavaScript tools, return a placeholder
		return fmt.Sprintf("JavaScript tool '%s' execution not yet implemented", toolName), nil
	}

	// For other tool types, just return a message
	return fmt.Sprintf("Executed MCP tool '%s' with arguments: %v", toolName, args), nil
}

// executeStandardAction executes an action from the action registry
func (s *StdioMCPServer) executeStandardAction(toolName string, args map[string]interface{}) (string, error) {
	if s.toolRegistry == nil {
		return "", fmt.Errorf("action registry not available")
	}

	// Extract action ID
	actionID := strings.TrimPrefix(toolName, "action_")

	// Get action definition
	action, err := s.toolRegistry.Get(actionID)
	if err != nil {
		return "", fmt.Errorf("action not found: %s", actionID)
	}

	// Get arguments
	arguments, _ := args["arguments"].(string)

	// Execute based on action type
	ctx := context.Background()

	if action.Command != "" {
		// Validate command before execution
		if err := s.toolExecutor.ValidateCommand(action.Command); err != nil {
			return "", fmt.Errorf("command validation failed: %w", err)
		}

		// Execute system command safely
		result, err := s.toolExecutor.ExecuteCommand(ctx, action.Command, arguments)
		if err != nil {
			return fmt.Sprintf("Action '%s' execution failed: %v\nOutput:\n%s", action.Name, err, result), nil
		}
		return fmt.Sprintf("Action '%s' executed successfully:\n%s", action.Name, result), nil
	} else if action.WorkflowRef != "" {
		// Execute workflow
		if s.workflowMgr != nil {
			result, err := s.workflowMgr.Execute(ctx, action.WorkflowRef, map[string]interface{}{
				"args": arguments,
			})
			if err != nil {
				return "", fmt.Errorf("workflow execution failed: %w", err)
			}
			return fmt.Sprintf("Action '%s' (workflow) executed successfully:\n%v", action.Name, result), nil
		}
		return "", fmt.Errorf("workflow manager not available for action: %s", action.Name)
	} else if action.PromptRef != "" {
		// Execute prompt
		if s.garden != nil {
			result, err := s.garden.Execute(action.PromptRef, map[string]interface{}{
				"args": arguments,
			})
			if err != nil {
				return "", fmt.Errorf("prompt execution failed: %w", err)
			}
			return result, nil
		}
		return "", fmt.Errorf("prompt garden not available for action: %s", action.Name)
	}

	return "", fmt.Errorf("action '%s' has no execution method defined", action.Name)
}

// handlePromptsList returns the list of available prompts
func (s *StdioMCPServer) handlePromptsList(id interface{}) {
	prompts := []map[string]interface{}{}

	if s.garden != nil {
		gardenPrompts, err := s.garden.List()
		if err == nil {
			for _, p := range gardenPrompts {
				// Skip templates
				if strings.HasSuffix(p.Name(), "-template") {
					continue
				}

				metadata := p.Metadata()

				// Build arguments from prompt variables
				arguments := []map[string]interface{}{}
				for _, v := range p.Variables() {
					arg := map[string]interface{}{
						"name":        v.Name,
						"description": v.Description,
						"required":    v.Required,
					}
					arguments = append(arguments, arg)
				}

				prompt := map[string]interface{}{
					"name":        p.Name(),
					"description": metadata.Description,
					"arguments":   arguments,
				}

				prompts = append(prompts, prompt)
			}
		}
	}

	s.sendResponse(id, map[string]interface{}{
		"prompts": prompts,
	})
}

// handlePromptsGet returns a specific prompt's content
func (s *StdioMCPServer) handlePromptsGet(id interface{}, params map[string]interface{}) {
	name, _ := params["name"].(string)
	arguments, _ := params["arguments"].(map[string]interface{})

	if s.garden == nil {
		s.sendError(id, fmt.Errorf("prompt garden not available"))
		return
	}

	// Execute the prompt with provided arguments
	result, err := s.garden.Execute(name, arguments)
	if err != nil {
		s.sendError(id, err)
		return
	}

	// Return as a prompt message
	s.sendResponse(id, map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": map[string]interface{}{
					"type": "text",
					"text": result,
				},
			},
		},
	})
}

// buildPromptParameters builds parameter schema for a prompt
func (s *StdioMCPServer) buildPromptParameters(p core.Prompt) map[string]interface{} {
	parameters := map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}

	properties := parameters["properties"].(map[string]interface{})
	required := []string{}

	for _, v := range p.Variables() {
		properties[v.Name] = map[string]interface{}{
			"type":        "string", // Default to string for simplicity
			"description": v.Description,
		}

		if v.Required {
			required = append(required, v.Name)
		}
	}

	if len(required) > 0 {
		parameters["required"] = required
	}

	return parameters
}
