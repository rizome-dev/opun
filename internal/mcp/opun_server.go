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
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rizome-dev/opun/internal/command"
	"github.com/rizome-dev/opun/internal/plugin"
	"github.com/rizome-dev/opun/internal/promptgarden"
	"github.com/rizome-dev/opun/pkg/core"
	"gopkg.in/yaml.v3"
)

// OpunMCPServer implements an MCP server that exposes all Opun capabilities
type OpunMCPServer struct {
	garden   *promptgarden.Garden
	registry *command.Registry
	manager  *plugin.Manager
	port     int
	server   *http.Server
}

// NewOpunMCPServer creates a new unified MCP server for Opun
func NewOpunMCPServer(garden *promptgarden.Garden, registry *command.Registry, manager *plugin.Manager, port int) *OpunMCPServer {
	return &OpunMCPServer{
		garden:   garden,
		registry: registry,
		manager:  manager,
		port:     port,
	}
}

// Start starts the MCP server
func (s *OpunMCPServer) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// MCP protocol endpoints
	mux.HandleFunc("/", s.handleRoot)
	mux.HandleFunc("/tools", s.handleTools)
	mux.HandleFunc("/tool/call", s.handleToolCall)
	mux.HandleFunc("/prompts/list", s.handlePromptsList)
	mux.HandleFunc("/prompts/get", s.handlePromptsGet)

	s.server = &http.Server{
		Addr:              fmt.Sprintf("localhost:%d", s.port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("MCP server error: %v\n", err)
		}
	}()

	// Write MCP server config for providers
	if err := s.writeConfig(); err != nil {
		return fmt.Errorf("failed to write MCP config: %w", err)
	}

	fmt.Printf("🚀 Opun MCP server started on port %d\n", s.port)
	return nil
}

// Stop stops the MCP server
func (s *OpunMCPServer) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

// handleRoot handles the root endpoint
func (s *OpunMCPServer) handleRoot(w http.ResponseWriter, r *http.Request) {
	response := map[string]interface{}{
		"name":        "opun",
		"version":     "1.0.0",
		"description": "Opun Unified MCP Server",
		"capabilities": map[string]bool{
			"tools":   true,
			"prompts": true,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

// handleTools returns all available tools (MCP tools, actions, commands)
func (s *OpunMCPServer) handleTools(w http.ResponseWriter, r *http.Request) {
	tools := []map[string]interface{}{}

	// NOTE: Prompts are now exposed via the prompts API, not as tools

	// Add tools from ~/.opun/tools
	home, _ := os.UserHomeDir()
	toolsDir := filepath.Join(home, ".opun", "tools")
	if entries, err := os.ReadDir(toolsDir); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() && (strings.HasSuffix(entry.Name(), ".yaml") || strings.HasSuffix(entry.Name(), ".yml")) {
				toolPath := filepath.Join(toolsDir, entry.Name())
				if data, err := os.ReadFile(toolPath); err == nil {
					var toolDef map[string]interface{}
					if err := yaml.Unmarshal(data, &toolDef); err == nil {
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

						// Create MCP tool definition
						tool := map[string]interface{}{
							"name":        fmt.Sprintf("tool_%s", name),
							"description": fmt.Sprintf("[Tool] %s: %s", name, description),
						}

						// Add input schema if present
						if schema, ok := toolDef["inputSchema"]; ok {
							tool["inputSchema"] = schema
						} else if parameters, ok := toolDef["parameters"]; ok {
							tool["inputSchema"] = parameters
						}

						tools = append(tools, tool)
					}
				}
			}
		}
	}

	// Add plugin tools
	if s.manager != nil {
		// New plugins don't expose tools directly - they import actions
		// which are handled separately through the action registry
	}

	// Add slash commands as tools
	if s.registry != nil {
		commands := s.registry.List()
		for _, cmd := range commands {
			// Skip hidden commands
			if cmd.Hidden {
				continue
			}

			parameters := map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"args": map[string]interface{}{
						"type":        "string",
						"description": "Arguments for the command",
					},
				},
			}

			tool := map[string]interface{}{
				"name":        fmt.Sprintf("command_%s", cmd.Name),
				"description": fmt.Sprintf("[Command] /%s: %s", cmd.Name, cmd.Description),
				"inputSchema": parameters,
			}

			tools = append(tools, tool)
		}
	}

	response := map[string]interface{}{
		"tools": tools,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

// handleToolCall executes a tool
func (s *OpunMCPServer) handleToolCall(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Tool      string                 `json:"tool"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var result string
	var err error

	// Determine tool type and execute
	switch {
	case strings.HasPrefix(request.Tool, "prompt_"):
		result, err = s.executePrompt(request.Tool, request.Arguments)
	case strings.HasPrefix(request.Tool, "plugin_"):
		result, err = s.executePlugin(request.Tool, request.Arguments)
	case strings.HasPrefix(request.Tool, "command_"):
		result, err = s.executeCommand(request.Tool, request.Arguments)
	case strings.HasPrefix(request.Tool, "tool_"):
		result, err = s.executeMCPTool(request.Tool, request.Arguments)
	default:
		err = fmt.Errorf("unknown tool type: %s", request.Tool)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": result,
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

// executePrompt executes a prompt tool
func (s *OpunMCPServer) executePrompt(tool string, args map[string]interface{}) (string, error) {
	if s.garden == nil {
		return "", fmt.Errorf("prompt garden not available")
	}

	// Extract prompt name from tool name
	promptName := strings.TrimPrefix(tool, "prompt_")
	promptName = strings.ReplaceAll(promptName, "_", "-")

	// Execute the prompt
	result, err := s.garden.Execute(promptName, args)
	if err != nil {
		// Try by ID if name fails
		result, err = s.garden.Execute(tool, args)
		if err != nil {
			return "", err
		}
	}

	return result, nil
}

// executePlugin executes a plugin tool
func (s *OpunMCPServer) executePlugin(tool string, args map[string]interface{}) (string, error) {
	if s.manager == nil {
		return "", fmt.Errorf("plugin manager not available")
	}

	// Extract plugin name and tool name
	parts := strings.SplitN(strings.TrimPrefix(tool, "plugin_"), "_", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid plugin tool format: %s", tool)
	}

	pluginName := parts[0]
	// toolName := parts[1] // Not used in new plugin system

	// Find plugin
	pluginInfos, _ := s.manager.ListPlugins()
	var targetPluginName string
	for _, info := range pluginInfos {
		if info.Name == pluginName {
			targetPluginName = info.Name
			break
		}
	}

	if targetPluginName == "" {
		return "", fmt.Errorf("plugin not found or not running: %s", pluginName)
	}

	// New plugins don't have tools - they import actions
	return "", fmt.Errorf("plugin tools not supported in new plugin system - plugins now import actions")
}

// executeCommand executes a slash command
func (s *OpunMCPServer) executeCommand(tool string, args map[string]interface{}) (string, error) {
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

	// Execute command (this would need proper implementation)
	return fmt.Sprintf("Execute command: /%s %s", cmd.Name, argsStr), nil
}

// executeMCPTool executes an MCP tool
func (s *OpunMCPServer) executeMCPTool(tool string, args map[string]interface{}) (string, error) {
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

	// For now, just return a message indicating the tool was called
	// In a real implementation, this would execute the tool's logic
	return fmt.Sprintf("Executed MCP tool '%s' with arguments: %v", toolName, args), nil
}

// buildPromptParameters builds parameter schema for a prompt
func (s *OpunMCPServer) buildPromptParameters(p core.Prompt) map[string]interface{} {
	parameters := map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}

	for _, v := range p.Variables() {
		parameters["properties"].(map[string]interface{})[v.Name] = map[string]interface{}{
			"type":        v.Type,
			"description": v.Description,
		}

		if v.Required {
			parameters["required"] = append(parameters["required"].([]string), v.Name)
		}
	}

	return parameters
}

// writeConfig writes the MCP server configuration
func (s *OpunMCPServer) writeConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	// Create opun MCP config directory
	configDir := filepath.Join(home, ".opun", "mcp")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied: cannot create %s\nTry: sudo chown -R $USER ~/.opun", configDir)
		}
		return err
	}

	// Write server config
	config := map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"opun": map[string]interface{}{
				"command": "curl",
				"args": []string{
					"-X",
					"POST",
					fmt.Sprintf("http://localhost:%d", s.port),
				},
				"env": map[string]string{},
			},
		},
	}

	configPath := filepath.Join(configDir, "opun-server.json")
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied: cannot write to %s\nTry: sudo chown -R $USER ~/.opun", configPath)
		}
		return err
	}

	fmt.Printf("📝 MCP config written to: %s\n", configPath)
	return nil
}

// GetInfo returns server information
func (s *OpunMCPServer) GetInfo() core.MCPServer {
	return core.MCPServer{
		Name:        "opun",
		Description: "Opun Unified MCP Server",
		Version:     "1.0.0",
		Enabled:     true,
		Endpoints: []core.MCPEndpoint{
			{
				Name:   "tools",
				URL:    fmt.Sprintf("http://localhost:%d/tools", s.port),
				Method: "GET",
			},
			{
				Name:   "tool/call",
				URL:    fmt.Sprintf("http://localhost:%d/tool/call", s.port),
				Method: "POST",
			},
		},
	}
}

// handlePromptsList returns the list of available prompts
func (s *OpunMCPServer) handlePromptsList(w http.ResponseWriter, r *http.Request) {
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

	response := map[string]interface{}{
		"prompts": prompts,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}

// handlePromptsGet returns a specific prompt's content
func (s *OpunMCPServer) handlePromptsGet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if s.garden == nil {
		http.Error(w, "Prompt garden not available", http.StatusInternalServerError)
		return
	}

	// Execute the prompt with provided arguments
	result, err := s.garden.Execute(request.Name, request.Arguments)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	// Return as a prompt message
	response := map[string]interface{}{
		"messages": []map[string]interface{}{
			{
				"role": "user",
				"content": map[string]interface{}{
					"type": "text",
					"text": result,
				},
			},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, fmt.Sprintf("Failed to encode response: %v", err), http.StatusInternalServerError)
	}
}
