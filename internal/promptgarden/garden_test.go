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
	"os"
	"path/filepath"
	"testing"

	"github.com/rizome-dev/opun/pkg/core"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGarden(t *testing.T) {
	// Create temp directory for test
	tempDir := t.TempDir()

	t.Run("Create Garden", func(t *testing.T) {
		garden, err := NewGarden(tempDir)
		require.NoError(t, err)
		assert.NotNil(t, garden)

		// Check that built-in prompts are loaded
		prompts, err := garden.List()
		assert.NoError(t, err)
		assert.GreaterOrEqual(t, len(prompts), 3) // At least 3 built-in prompts
	})

	t.Run("Add and Get Prompt", func(t *testing.T) {
		garden, _ := NewGarden(tempDir)

		// Create a prompt
		metadata := core.PromptMetadata{
			Name:        "test-prompt",
			Description: "Test prompt",
			Type:        core.PromptTypeTemplate,
			Category:    "test",
			Tags:        []string{"test", "example"},
		}
		prompt := NewTemplatePrompt(metadata, "Hello {{name}}!")

		// Add prompt
		err := garden.Add(prompt)
		require.NoError(t, err)

		// Get by ID
		retrieved, err := garden.Get(prompt.ID())
		require.NoError(t, err)
		assert.Equal(t, prompt.Name(), retrieved.Name())
		assert.Equal(t, prompt.Content(), retrieved.Content())

		// Get by name
		retrieved, err = garden.GetByName("test-prompt")
		require.NoError(t, err)
		assert.Equal(t, prompt.ID(), retrieved.ID())
	})

	t.Run("Execute Template", func(t *testing.T) {
		garden, _ := NewGarden(tempDir)

		// Create and add a prompt
		metadata := core.PromptMetadata{
			Name: "greeting-template",
		}
		prompt := NewTemplatePrompt(metadata, "Hello {{name}}, welcome to {{place}}!")
		garden.Add(prompt)

		// Execute with variables
		result, err := garden.Execute("greeting-template", map[string]interface{}{
			"name":  "Alice",
			"place": "Wonderland",
		})
		require.NoError(t, err)
		assert.Equal(t, "Hello Alice, welcome to Wonderland!", result)
	})

	t.Run("Template with Includes", func(t *testing.T) {
		garden, _ := NewGarden(tempDir)

		// Create base prompt
		basePrompt := NewTemplatePrompt(
			core.PromptMetadata{Name: "base-prompt"},
			"This is the base content.",
		)
		garden.Add(basePrompt)

		// Create prompt with include
		includePrompt := NewTemplatePrompt(
			core.PromptMetadata{Name: "include-prompt"},
			"Header\n{{include:base-prompt}}\nFooter",
		)
		garden.Add(includePrompt)

		// Execute
		result, err := garden.Execute("include-prompt", nil)
		require.NoError(t, err)
		assert.Contains(t, result, "Header")
		assert.Contains(t, result, "This is the base content.")
		assert.Contains(t, result, "Footer")
	})

	t.Run("Search Prompts", func(t *testing.T) {
		garden, _ := NewGarden(tempDir)

		// Add test prompts
		prompt1 := NewTemplatePrompt(
			core.PromptMetadata{
				Name:     "search-test-1",
				Category: "search",
				Tags:     []string{"findme", "test"},
			},
			"Content 1",
		)
		prompt2 := NewTemplatePrompt(
			core.PromptMetadata{
				Name:     "search-test-2",
				Category: "other",
				Tags:     []string{"different", "findme"},
			},
			"Content 2",
		)

		garden.Add(prompt1)
		garden.Add(prompt2)

		// Search by tag
		results, err := garden.Search("findme")
		require.NoError(t, err)
		assert.Len(t, results, 2)

		// Search by category
		results, err = garden.Search("search")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 1)

		// List by category
		results, err = garden.ListByCategory("search")
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(results), 1)

		// List by tags
		results, err = garden.ListByTags([]string{"findme"})
		require.NoError(t, err)
		assert.Len(t, results, 2)
	})

	t.Run("Import and Export", func(t *testing.T) {
		garden, _ := NewGarden(tempDir)

		// Create a markdown file
		mdContent := `---
name: imported-prompt
description: Imported from markdown
category: imported
tags: test, import
---
# My Imported Prompt

This is the content with {{variable}}.`

		mdPath := filepath.Join(tempDir, "test.md")
		err := os.WriteFile(mdPath, []byte(mdContent), 0644)
		require.NoError(t, err)

		// Import
		err = garden.ImportFromFile(mdPath)
		require.NoError(t, err)

		// Verify import
		prompt, err := garden.GetByName("imported-prompt")
		require.NoError(t, err)
		assert.Equal(t, "Imported from markdown", prompt.Metadata().Description)
		assert.Contains(t, prompt.Content(), "This is the content")

		// Export
		exportPath := filepath.Join(tempDir, "export.json")
		err = garden.ExportToFile("imported-prompt", exportPath)
		require.NoError(t, err)

		// Verify export file exists
		_, err = os.Stat(exportPath)
		assert.NoError(t, err)
	})
}

func TestTemplateEngine(t *testing.T) {
	t.Run("Simple Variables", func(t *testing.T) {
		engine := NewTemplateEngine()

		template := "Hello {{name}}, you are {{age}} years old."
		vars := map[string]interface{}{
			"name": "Bob",
			"age":  25,
		}

		result, err := engine.Execute(template, vars)
		require.NoError(t, err)
		assert.Equal(t, "Hello Bob, you are 25 years old.", result)
	})

	t.Run("File References", func(t *testing.T) {
		engine := NewTemplateEngine()

		// Create test file
		tempFile := filepath.Join(t.TempDir(), "test.txt")
		os.WriteFile(tempFile, []byte("File content"), 0644)

		template := "Content: {{file:" + tempFile + "}}"
		result, err := engine.Execute(template, nil)
		require.NoError(t, err)
		assert.Equal(t, "Content: File content", result)
	})

	t.Run("Template Functions", func(t *testing.T) {
		engine := NewTemplateEngine()

		template := `{{upper "hello"}} {{lower "WORLD"}}`
		result, err := engine.Execute(template, nil)
		require.NoError(t, err)
		assert.Contains(t, result, "HELLO")
		assert.Contains(t, result, "world")
	})

	t.Run("Complex Templates", func(t *testing.T) {
		engine := NewTemplateEngine()

		template := `{{range .items}}
- {{.name}}: {{.value}}
{{end}}`

		vars := map[string]interface{}{
			"items": []map[string]interface{}{
				{"name": "Item1", "value": 10},
				{"name": "Item2", "value": 20},
			},
		}

		result, err := engine.Execute(template, vars)
		require.NoError(t, err)
		assert.Contains(t, result, "Item1: 10")
		assert.Contains(t, result, "Item2: 20")
	})
}

func TestTemplatePrompt(t *testing.T) {
	t.Run("Variable Extraction", func(t *testing.T) {
		content := "Hello {{name}}, welcome to {{place}}! Your score is {{score}}."
		metadata := core.PromptMetadata{Name: "test"}

		prompt := NewTemplatePrompt(metadata, content)

		vars := prompt.Variables()
		assert.Len(t, vars, 3)

		varNames := make(map[string]bool)
		for _, v := range vars {
			varNames[v.Name] = true
		}

		assert.True(t, varNames["name"])
		assert.True(t, varNames["place"])
		assert.True(t, varNames["score"])
	})

	t.Run("Validation", func(t *testing.T) {
		metadata := core.PromptMetadata{
			Name: "test",
			Variables: []core.PromptVariable{
				{Name: "required_var", Required: true},
				{Name: "optional_var", Required: false},
			},
		}

		prompt := NewTemplatePrompt(metadata, "{{required_var}} {{optional_var}}")

		// Missing required variable
		err := prompt.Validate(map[string]interface{}{
			"optional_var": "value",
		})
		assert.Error(t, err)

		// All required variables provided
		err = prompt.Validate(map[string]interface{}{
			"required_var": "value",
		})
		assert.NoError(t, err)
	})

	t.Run("Default Values", func(t *testing.T) {
		metadata := core.PromptMetadata{
			Name: "test",
			Variables: []core.PromptVariable{
				{Name: "var1", DefaultValue: "default1"},
				{Name: "var2", DefaultValue: "default2"},
			},
		}

		prompt := NewTemplatePrompt(metadata, "{{var1}} {{var2}}")

		// Execute with partial variables
		result, err := prompt.Template(map[string]interface{}{
			"var1": "custom",
		})
		require.NoError(t, err)
		assert.Equal(t, "custom default2", result)
	})
}
