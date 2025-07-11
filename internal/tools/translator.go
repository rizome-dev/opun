package tools

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
	"path/filepath"
	"strings"

	"github.com/rizome-dev/opun/internal/utils"
	"github.com/rizome-dev/opun/pkg/core"
)

// Translator converts StandardAction to provider-specific formats
type Translator struct {
	registry *Registry
}

// NewTranslator creates a new tool translator
func NewTranslator(registry *Registry) *Translator {
	return &Translator{
		registry: registry,
	}
}

// ToClaudeMarkdown converts a StandardAction to Claude's markdown format
func (t *Translator) ToClaudeMarkdown(action core.StandardAction) (string, error) {
	var content strings.Builder

	content.WriteString(fmt.Sprintf("# %s\n\n", action.Name))
	content.WriteString(fmt.Sprintf("%s\n\n", action.Description))

	// Determine how to execute the action
	if action.Command != "" {
		content.WriteString("## Command\n\n")
		content.WriteString("```bash\n")
		content.WriteString(action.Command)
		content.WriteString(" $ARGUMENTS\n")
		content.WriteString("```\n\n")
		content.WriteString("This action executes a system command.\n")
	} else if action.WorkflowRef != "" {
		content.WriteString("## Workflow\n\n")
		content.WriteString(fmt.Sprintf("Execute the Opun workflow: `%s`\n\n", action.WorkflowRef))
		content.WriteString("Use the following command:\n")
		content.WriteString("```bash\n")
		content.WriteString(fmt.Sprintf("opun run %s $ARGUMENTS\n", action.WorkflowRef))
		content.WriteString("```\n")
	} else if action.PromptRef != "" {
		content.WriteString("## Prompt\n\n")
		content.WriteString(fmt.Sprintf("Execute the Opun prompt: `%s`\n\n", action.PromptRef))
		content.WriteString("Use the following command:\n")
		content.WriteString("```bash\n")
		content.WriteString(fmt.Sprintf("opun prompt %s $ARGUMENTS\n", action.PromptRef))
		content.WriteString("```\n")
	} else {
		return "", fmt.Errorf("action %s has no execution method defined", action.ID)
	}

	content.WriteString("\n## Category\n\n")
	content.WriteString(fmt.Sprintf("Category: %s\n", action.Category))

	return content.String(), nil
}

// WriteClaudeCommand writes an action as a Claude command markdown file
func (t *Translator) WriteClaudeCommand(action core.StandardAction, commandDir string) error {
	content, err := t.ToClaudeMarkdown(action)
	if err != nil {
		return err
	}

	// Use action ID as filename
	filename := filepath.Join(commandDir, fmt.Sprintf("%s.md", action.ID))

	return utils.WriteFile(filename, []byte(content))
}

// ToMCPAction converts a StandardAction to MCP tool format
func (t *Translator) ToMCPAction(action core.StandardAction) map[string]interface{} {
	// Build the input schema
	inputSchema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"arguments": map[string]interface{}{
				"type":        "string",
				"description": "Arguments to pass to the action",
			},
		},
	}

	// MCP tool format
	mcpTool := map[string]interface{}{
		"name":        fmt.Sprintf("action_%s", action.ID),
		"description": fmt.Sprintf("[Action] %s: %s", action.Name, action.Description),
		"inputSchema": inputSchema,
		"metadata": map[string]interface{}{
			"source":    "action",
			"provider":  "opun",
			"version":   action.Version,
			"category":  action.Category,
			"action_id": action.ID,
		},
	}

	// Add execution metadata
	if action.Command != "" {
		mcpTool["metadata"].(map[string]interface{})["command"] = action.Command
	} else if action.WorkflowRef != "" {
		mcpTool["metadata"].(map[string]interface{})["workflow"] = action.WorkflowRef
	} else if action.PromptRef != "" {
		mcpTool["metadata"].(map[string]interface{})["prompt"] = action.PromptRef
	}

	return mcpTool
}

// GetMCPActions returns all actions in MCP format for a provider
func (t *Translator) GetMCPActions(provider string) []map[string]interface{} {
	actions := t.registry.List(provider)
	mcpActions := make([]map[string]interface{}, 0, len(actions))

	for _, action := range actions {
		mcpActions = append(mcpActions, t.ToMCPAction(action))
	}

	return mcpActions
}

// PrepareProviderActions prepares actions for a specific provider
func (t *Translator) PrepareProviderActions(provider core.Provider) error {
	providerType := provider.Type()

	switch providerType {
	case "claude":
		return t.prepareClaudeActions(provider)
	case "gemini":
		// Gemini uses MCP, actions are exposed through the MCP server
		return nil
	default:
		return fmt.Errorf("unsupported provider type: %s", providerType)
	}
}

// prepareClaudeActions writes markdown files for Claude
func (t *Translator) prepareClaudeActions(provider core.Provider) error {
	commandDir := provider.GetSlashCommandDirectory()
	if commandDir == "" {
		return fmt.Errorf("Claude provider has no command directory")
	}

	// Ensure directory exists
	if err := utils.EnsureDir(commandDir); err != nil {
		return fmt.Errorf("failed to create command directory: %w", err)
	}

	// Get actions for this provider
	actions := t.registry.List(string(provider.Type()))

	// Write each action as a markdown file
	for _, action := range actions {
		if err := t.WriteClaudeCommand(action, commandDir); err != nil {
			return fmt.Errorf("failed to write action %s: %w", action.ID, err)
		}
	}

	return nil
}
