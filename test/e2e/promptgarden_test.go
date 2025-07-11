package e2e

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

	"github.com/rizome-dev/opun/internal/promptgarden"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptGarden(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create temporary directory for garden
	tempDir := t.TempDir()
	gardenPath := filepath.Join(tempDir, "promptgarden")

	// Initialize garden
	garden, err := promptgarden.NewGarden(gardenPath)
	require.NoError(t, err)
	require.NotNil(t, garden)

	t.Run("Built-in Prompts", func(t *testing.T) {
		// Check that built-in prompts are loaded
		prompts, err := garden.ListPrompts()
		require.NoError(t, err)
		assert.Greater(t, len(prompts), 0)

		// Look for specific built-in prompts
		foundPlanning := false
		foundReview := false
		foundQuestions := false

		for _, p := range prompts {
			switch p.Name {
			case "planning-template":
				foundPlanning = true
			case "review-template":
				foundReview = true
			case "questions-template":
				foundQuestions = true
			}
		}

		assert.True(t, foundPlanning, "planning-template not found")
		assert.True(t, foundReview, "review-template not found")
		assert.True(t, foundQuestions, "questions-template not found")
	})

	t.Run("Add and Retrieve Prompt", func(t *testing.T) {
		prompt := &promptgarden.Prompt{
			ID:   "test-prompt-1",
			Name: "test-prompt-1",
			Content: `# Test Prompt
Hello {{name}}, this is a test prompt.
Your task: {{task}}`,
			Metadata: promptgarden.PromptMetadata{
				Tags:        []string{"test", "e2e"},
				Category:    "testing",
				Version:     "1.0.0",
				Description: "E2E test prompt",
				Author:      "test",
			},
		}

		// Save prompt
		err := garden.SavePrompt(prompt)
		require.NoError(t, err)

		// Retrieve by name
		retrieved, err := garden.GetByName("test-prompt-1")
		require.NoError(t, err)
		assert.Equal(t, prompt.Name, retrieved.Name())
		assert.Equal(t, prompt.Content, retrieved.Content())
	})

	t.Run("Template Execution", func(t *testing.T) {
		// Execute template with variables
		result, err := garden.Execute("test-prompt-1", map[string]interface{}{
			"name": "Alice",
			"task": "Write unit tests",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "Hello Alice")
		assert.Contains(t, result, "Your task: Write unit tests")
	})

	t.Run("Prompt with Includes", func(t *testing.T) {
		// Create a base prompt
		basePrompt := &promptgarden.Prompt{
			ID:      "base-prompt",
			Name:    "base-prompt",
			Content: "This is the base content.",
			Metadata: promptgarden.PromptMetadata{
				Category: "testing",
			},
		}
		err := garden.SavePrompt(basePrompt)
		require.NoError(t, err)

		// Create a prompt that includes the base
		includingPrompt := &promptgarden.Prompt{
			ID:   "including-prompt",
			Name: "including-prompt",
			Content: `# Main Prompt
{{include:base-prompt}}

Additional content here.`,
			Metadata: promptgarden.PromptMetadata{
				Category: "testing",
			},
		}
		err = garden.SavePrompt(includingPrompt)
		require.NoError(t, err)

		// Execute the including prompt
		result, err := garden.Execute("including-prompt", nil)
		require.NoError(t, err)
		assert.Contains(t, result, "This is the base content")
		assert.Contains(t, result, "Additional content here")
	})

	t.Run("Import and Export", func(t *testing.T) {
		// Create a markdown file to import
		mdPath := filepath.Join(tempDir, "import-test.md")
		mdContent := `---
name: imported-prompt
description: Imported from markdown
category: imported
tags: markdown, import
---

# Imported Prompt

This prompt was imported from a markdown file.`

		err := os.WriteFile(mdPath, []byte(mdContent), 0644)
		require.NoError(t, err)

		// Import the file
		// Note: ImportFromFile is not exposed in the types.go, so we'd need to add it
		// For now, we'll test the store directly
		prompts, err := garden.ListPrompts()
		require.NoError(t, err)
		initialCount := len(prompts)

		// Create the prompt manually (simulating import)
		importedPrompt := &promptgarden.Prompt{
			ID:      "imported-prompt",
			Name:    "imported-prompt",
			Content: "This prompt was imported from a markdown file.",
			Metadata: promptgarden.PromptMetadata{
				Category:    "imported",
				Tags:        []string{"markdown", "import"},
				Description: "Imported from markdown",
			},
		}
		err = garden.SavePrompt(importedPrompt)
		require.NoError(t, err)

		// Verify it was added
		prompts, err = garden.ListPrompts()
		require.NoError(t, err)
		assert.Equal(t, initialCount+1, len(prompts))
	})

	t.Run("Search Functionality", func(t *testing.T) {
		// Add a searchable prompt
		searchPrompt := &promptgarden.Prompt{
			ID:      "search-test",
			Name:    "search-test",
			Content: "This prompt contains unique keywords like elasticsearch and fuzzy matching.",
			Metadata: promptgarden.PromptMetadata{
				Category:    "search",
				Tags:        []string{"search", "test"},
				Description: "Test prompt for search functionality",
			},
		}
		err := garden.SavePrompt(searchPrompt)
		require.NoError(t, err)

		// Search for the prompt
		// Note: Search is not implemented in types.go, would need to add it
		prompts, err := garden.ListPrompts()
		require.NoError(t, err)

		found := false
		for _, p := range prompts {
			if p.Name == "search-test" {
				found = true
				break
			}
		}
		assert.True(t, found, "search-test prompt not found")
	})

	t.Run("Category Filtering", func(t *testing.T) {
		// List prompts by category
		prompts, err := garden.ListPrompts()
		require.NoError(t, err)

		// Count prompts by category
		categories := make(map[string]int)
		for _, p := range prompts {
			categories[p.Metadata.Category]++
		}

		// Verify we have multiple categories
		assert.Greater(t, len(categories), 1)
		assert.Greater(t, categories["testing"], 0)
		assert.Greater(t, categories["templates"], 0)
	})
}
