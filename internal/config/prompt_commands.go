package config

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
	"github.com/rizome-dev/opun/internal/utils"
	"github.com/rizome-dev/opun/pkg/core"
)

// PromptCommandGenerator generates slash commands from prompts
type PromptCommandGenerator struct {
	garden        *promptgarden.Garden
	sharedManager *SharedConfigManager
}

// NewPromptCommandGenerator creates a new prompt command generator
func NewPromptCommandGenerator() (*PromptCommandGenerator, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	gardenPath := filepath.Join(homeDir, ".opun", "promptgarden")
	garden, err := promptgarden.NewGarden(gardenPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize prompt garden: %w", err)
	}

	sharedManager, err := NewSharedConfigManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create shared config manager: %w", err)
	}

	return &PromptCommandGenerator{
		garden:        garden,
		sharedManager: sharedManager,
	}, nil
}

// GeneratePromptCommands creates slash commands for all prompts
func (g *PromptCommandGenerator) GeneratePromptCommands() error {
	// Get all prompts from the garden
	prompts, err := g.garden.List()
	if err != nil {
		return fmt.Errorf("failed to list prompts: %w", err)
	}

	// Create slash commands for each prompt
	for _, prompt := range prompts {
		// Skip system prompts that shouldn't be exposed
		if strings.HasPrefix(prompt.Name(), "_") || strings.HasPrefix(prompt.Name(), ".") {
			continue
		}

		command := g.createPromptCommand(prompt)
		if err := g.sharedManager.AddSlashCommand(command); err != nil {
			// Log but continue with other prompts
			fmt.Printf("Warning: failed to add prompt command %s: %v\n", prompt.Name(), err)
		}
	}

	return g.sharedManager.Save()
}

// createPromptCommand creates a slash command from a prompt
func (g *PromptCommandGenerator) createPromptCommand(prompt core.Prompt) core.SharedSlashCommand {
	// Get metadata
	metadata := prompt.Metadata()

	// Generate command name from prompt name
	cmdName := g.sanitizeCommandName(metadata.Name)

	// Generate description
	description := metadata.Description
	if description == "" {
		description = fmt.Sprintf("Execute the %s prompt", metadata.Name)
	}

	return core.SharedSlashCommand{
		Name:        cmdName,
		Description: description,
		Type:        "prompt",
		Handler:     fmt.Sprintf("promptgarden://%s", metadata.Name),
		Aliases:     g.generateAliases(metadata.Name),
		Hidden:      false,
	}
}

// GenerateClaudePromptFiles generates .claude/commands files for prompts
func (g *PromptCommandGenerator) GenerateClaudePromptFiles(commandsDir string) error {
	// Create prompts subdirectory with proper ownership
	promptsDir := filepath.Join(commandsDir, "prompts")
	if err := utils.EnsureDir(promptsDir); err != nil {
		return err
	}

	prompts, err := g.garden.List()
	if err != nil {
		return fmt.Errorf("failed to list prompts: %w", err)
	}

	for _, prompt := range prompts {
		// Skip system prompts
		if strings.HasPrefix(prompt.Name(), "_") || strings.HasPrefix(prompt.Name(), ".") {
			continue
		}

		// Generate command file
		if err := g.generatePromptFile(promptsDir, prompt); err != nil {
			fmt.Printf("Warning: failed to generate prompt file for %s: %v\n", prompt.Name(), err)
		}
	}

	// Create a general prompt runner command
	runnerContent := `# Run Prompt

Execute a prompt from the PromptGarden by name.

Usage: /prompts:run <prompt-name> [arguments]

The prompt will be loaded from the PromptGarden and executed with any provided arguments substituted into the template.

Available prompts can be found in the other files in this directory.`

	runnerFile := filepath.Join(promptsDir, "run.md")
	return utils.WriteFile(runnerFile, []byte(runnerContent))
}

// generatePromptFile creates a command file for a specific prompt
func (g *PromptCommandGenerator) generatePromptFile(promptsDir string, prompt core.Prompt) error {
	metadata := prompt.Metadata()
	cmdName := g.sanitizeCommandName(metadata.Name)
	filename := fmt.Sprintf("%s.md", cmdName)
	cmdFilePath := filepath.Join(promptsDir, filename)

	// Get the prompt content
	content := prompt.Content()

	// Generate the command markdown
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# %s\n\n", metadata.Description))

	if metadata.Author != "" {
		sb.WriteString(fmt.Sprintf("*Author: %s*\n\n", metadata.Author))
	}

	if len(metadata.Tags) > 0 {
		sb.WriteString(fmt.Sprintf("*Tags: %s*\n\n", strings.Join(metadata.Tags, ", ")))
	}

	sb.WriteString("## Prompt Template\n\n")
	sb.WriteString("```\n")
	sb.WriteString(content)
	sb.WriteString("\n```\n\n")

	sb.WriteString("## Usage\n\n")
	sb.WriteString("This command will execute the above prompt template. ")
	sb.WriteString("Any occurrences of `$ARGUMENTS` in the template will be replaced with your input.\n")

	return utils.WriteFile(cmdFilePath, []byte(sb.String()))
}

// sanitizeCommandName converts a prompt name to a valid command name
func (g *PromptCommandGenerator) sanitizeCommandName(name string) string {
	// Convert to lowercase and replace spaces/special chars with hyphens
	cmdName := strings.ToLower(name)
	cmdName = strings.ReplaceAll(cmdName, " ", "-")
	cmdName = strings.ReplaceAll(cmdName, "_", "-")
	cmdName = strings.ReplaceAll(cmdName, "/", "-")
	cmdName = strings.ReplaceAll(cmdName, ".", "-")

	// Remove any double hyphens
	for strings.Contains(cmdName, "--") {
		cmdName = strings.ReplaceAll(cmdName, "--", "-")
	}

	// Trim hyphens from start/end
	cmdName = strings.Trim(cmdName, "-")

	return cmdName
}

// generateAliases creates aliases for a prompt command
func (g *PromptCommandGenerator) generateAliases(name string) []string {
	aliases := []string{}

	// Create an abbreviated alias if the name has multiple words
	parts := strings.Fields(name)
	if len(parts) > 1 {
		var abbrev strings.Builder
		for _, part := range parts {
			if len(part) > 0 {
				abbrev.WriteString(string(part[0]))
			}
		}
		if abbrev.Len() > 1 {
			aliases = append(aliases, strings.ToLower(abbrev.String()))
		}
	}

	return aliases
}

// UpdatePromptCommands updates all prompt-based commands
func UpdatePromptCommands() error {
	generator, err := NewPromptCommandGenerator()
	if err != nil {
		return err
	}

	return generator.GeneratePromptCommands()
}
