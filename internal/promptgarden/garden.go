package promptgarden

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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rizome-dev/opun/internal/utils"
	"github.com/rizome-dev/opun/pkg/command"
	"github.com/rizome-dev/opun/pkg/core"
)

// Garden represents the prompt garden manager
type Garden struct {
	storePath string
	store     core.PromptStore
	templates *TemplateEngine
	mu        sync.RWMutex
}

// NewGarden creates a new prompt garden
func NewGarden(storePath string) (*Garden, error) {
	// Ensure store path exists with proper ownership
	if err := utils.EnsureDir(storePath); err != nil {
		return nil, fmt.Errorf("failed to create store path: %w", err)
	}

	// Create file store
	store := NewFileStore(storePath)

	// Create template engine
	templates := NewTemplateEngine()

	garden := &Garden{
		storePath: storePath,
		store:     store,
		templates: templates,
	}

	// Set the garden as include resolver for templates
	templates.SetIncludeResolver(garden)

	// Load built-in prompts
	if err := garden.loadBuiltinPrompts(); err != nil {
		return nil, fmt.Errorf("failed to load builtin prompts: %w", err)
	}

	return garden, nil
}

// Add adds a prompt to the garden
func (g *Garden) Add(prompt core.Prompt) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Validate prompt
	if prompt.Name() == "" {
		return fmt.Errorf("prompt name is required")
	}

	// Check for duplicates
	if existing, err := g.store.GetByName(prompt.Name()); err == nil && existing != nil {
		return fmt.Errorf("prompt with name '%s' already exists", prompt.Name())
	}

	// Store prompt
	return g.store.Create(prompt)
}

// Get retrieves a prompt by ID
func (g *Garden) Get(id string) (core.Prompt, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.store.Get(id)
}

// GetByName retrieves a prompt by name
func (g *Garden) GetByName(name string) (core.Prompt, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.store.GetByName(name)
}

// List returns all prompts
func (g *Garden) List() ([]core.Prompt, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.store.List()
}

// ListByCategory returns prompts in a category
func (g *Garden) ListByCategory(category string) ([]core.Prompt, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.store.ListByCategory(category)
}

// ListByTags returns prompts with matching tags
func (g *Garden) ListByTags(tags []string) ([]core.Prompt, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.store.ListByTags(tags)
}

// Search searches for prompts
func (g *Garden) Search(query string) ([]core.Prompt, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.store.Search(query)
}

// Execute executes a prompt with variables
func (g *Garden) Execute(nameOrID string, vars map[string]interface{}) (string, error) {
	// Try to get by name first
	prompt, err := g.GetByName(nameOrID)
	if err != nil {
		// Try by ID
		prompt, err = g.Get(nameOrID)
		if err != nil {
			return "", fmt.Errorf("prompt not found: %s", nameOrID)
		}
	}

	// Set include resolver
	prompt.SetIncludeResolver(g)

	// Execute template
	return prompt.Template(vars)
}

// ImportFromFile imports a prompt from a file
func (g *Garden) ImportFromFile(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Determine format based on extension
	ext := strings.ToLower(filepath.Ext(filePath))

	switch ext {
	case ".json":
		return g.importJSON(data)
	case ".md", ".txt":
		return g.importMarkdown(filePath, data)
	default:
		return fmt.Errorf("unsupported file format: %s", ext)
	}
}

// ExportToFile exports a prompt to a file
func (g *Garden) ExportToFile(nameOrID string, filePath string) error {
	prompt, err := g.GetByName(nameOrID)
	if err != nil {
		prompt, err = g.Get(nameOrID)
		if err != nil {
			return fmt.Errorf("prompt not found: %s", nameOrID)
		}
	}

	data, err := g.store.Export(prompt.ID())
	if err != nil {
		return fmt.Errorf("failed to export prompt: %w", err)
	}

	return os.WriteFile(filePath, data, 0644)
}

// Resolve implements IncludeResolver interface
func (g *Garden) Resolve(promptID string) (core.Prompt, error) {
	// Handle different prompt reference formats
	if strings.HasPrefix(promptID, "promptgarden://") {
		promptID = strings.TrimPrefix(promptID, "promptgarden://")
	}

	// Try by name first
	prompt, err := g.GetByName(promptID)
	if err == nil {
		return prompt, nil
	}

	// Try by ID
	return g.Get(promptID)
}

// loadBuiltinPrompts loads built-in prompts
func (g *Garden) loadBuiltinPrompts() error {
	builtins := []struct {
		name     string
		category string
		content  string
		tags     []string
	}{
		{
			name:     "planning-template",
			category: "templates",
			content: `# Planning Phase

## Overview
{{description}}

## Current State
{{current_state}}

## Target State
{{target_state}}

## Implementation Steps
1. [Add your implementation steps here]

## Success Criteria
{{success_criteria}}

## Risks and Mitigations
[Identify potential risks and how to mitigate them]`,
			tags: []string{"planning", "template", "refactor"},
		},
		{
			name:     "review-template",
			category: "templates",
			content: `# Code Review

## Changes Summary
{{changes_summary}}

## Review Checklist
- [ ] Code follows project conventions
- [ ] Tests are comprehensive and passing
- [ ] Documentation is updated
- [ ] No security vulnerabilities introduced
- [ ] Performance impact is acceptable

## Detailed Review

### Architecture
[Review architectural decisions]

### Code Quality
[Review code quality aspects]

### Testing
[Review test coverage and quality]

## Recommendations
[Provide specific recommendations for improvement]`,
			tags: []string{"review", "template", "quality"},
		},
		{
			name:     "questions-template",
			category: "templates",
			content: `# Clarifying Questions

Based on the provided {{document_type}}, I have the following questions:

## Technical Questions
{{#each technical_questions}}
- {{this}}
{{/each}}

## Business/Requirements Questions
{{#each business_questions}}
- {{this}}
{{/each}}

## Implementation Questions
{{#each implementation_questions}}
- {{this}}
{{/each}}

Please provide answers to help create a comprehensive implementation plan.`,
			tags: []string{"questions", "template", "discovery"},
		},
	}

	for _, builtin := range builtins {
		metadata := core.PromptMetadata{
			Name:        builtin.name,
			Description: fmt.Sprintf("Built-in %s", builtin.name),
			Type:        core.PromptTypeTemplate,
			Category:    builtin.category,
			Tags:        builtin.tags,
			Author:      "system",
			Version:     "1.0.0",
		}

		prompt := NewTemplatePrompt(metadata, builtin.content)
		if err := g.Add(prompt); err != nil {
			// Ignore duplicate errors for built-ins
			if !strings.Contains(err.Error(), "already exists") {
				return err
			}
		}
	}

	return nil
}

// importJSON imports a prompt from JSON
func (g *Garden) importJSON(data []byte) error {
	var metadata core.PromptMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Extract content from metadata
	content, ok := metadata.Extra["content"].(string)
	if !ok {
		return fmt.Errorf("missing content in prompt data")
	}

	prompt := NewTemplatePrompt(metadata, content)
	return g.Add(prompt)
}

// importMarkdown imports a prompt from markdown
func (g *Garden) importMarkdown(filePath string, data []byte) error {
	// Extract metadata from filename
	name := strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath))

	// Parse front matter if present
	content := string(data)
	metadata := core.PromptMetadata{
		Name:        name,
		Description: fmt.Sprintf("Imported from %s", filePath),
		Type:        core.PromptTypeTemplate,
		Category:    "imported",
		Tags:        []string{"imported"},
		Author:      "user",
		Version:     "1.0.0",
	}

	// Simple front matter parsing
	if strings.HasPrefix(content, "---\n") {
		parts := strings.SplitN(content, "---\n", 3)
		if len(parts) >= 3 {
			// Parse YAML front matter (simplified)
			frontMatter := parts[1]
			content = parts[2]

			// Extract basic fields
			for _, line := range strings.Split(frontMatter, "\n") {
				if strings.HasPrefix(line, "name:") {
					metadata.Name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
				} else if strings.HasPrefix(line, "description:") {
					metadata.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
				} else if strings.HasPrefix(line, "category:") {
					metadata.Category = strings.TrimSpace(strings.TrimPrefix(line, "category:"))
				} else if strings.HasPrefix(line, "tags:") {
					tagStr := strings.TrimSpace(strings.TrimPrefix(line, "tags:"))
					metadata.Tags = strings.Split(tagStr, ",")
					for i := range metadata.Tags {
						metadata.Tags[i] = strings.TrimSpace(metadata.Tags[i])
					}
				}
			}
		}
	}

	prompt := NewTemplatePrompt(metadata, content)
	return g.Add(prompt)
}

// RegisterAsCommands registers all prompts in the garden as slash commands
func (g *Garden) RegisterAsCommands(registry interface{ Register(*command.Command) error }) error {
	prompts, err := g.List()
	if err != nil {
		return fmt.Errorf("failed to list prompts: %w", err)
	}

	for _, p := range prompts {
		// Skip built-in templates that are not meant to be commands
		if strings.HasSuffix(p.Name(), "-template") {
			continue
		}

		// Get metadata
		metadata := p.Metadata()

		// Create command from prompt
		cmd := &command.Command{
			Name:        p.Name(),
			Description: metadata.Description,
			Category:    metadata.Category,
			Type:        command.CommandTypePrompt,
			Handler:     p.ID(),        // Use prompt ID as handler
			Aliases:     metadata.Tags, // Use tags as aliases
			Arguments:   convertVariablesToArguments(p.Variables()),
		}

		// Register the command
		if err := registry.Register(cmd); err != nil {
			// Log error but continue registering other prompts
			fmt.Printf("Warning: failed to register prompt '%s' as command: %v\n", p.Name(), err)
		}
	}

	return nil
}

// convertVariablesToArguments converts prompt variables to command arguments
func convertVariablesToArguments(vars []core.PromptVariable) []command.Argument {
	args := make([]command.Argument, len(vars))
	for i, v := range vars {
		args[i] = command.Argument{
			Name:         v.Name,
			Description:  v.Description,
			Type:         v.Type,
			Required:     v.Required,
			DefaultValue: v.DefaultValue,
		}
	}
	return args
}
