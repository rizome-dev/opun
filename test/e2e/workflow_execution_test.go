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
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/rizome-dev/opun/internal/promptgarden"
	workflowexec "github.com/rizome-dev/opun/internal/workflow"
	"github.com/rizome-dev/opun/pkg/workflow"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create executor
	executor := workflowexec.NewExecutor()
	require.NotNil(t, executor)

	t.Run("Simple Workflow Execution", func(t *testing.T) {
		// Create a simple workflow
		wf := &workflow.Workflow{
			Name:        "test-workflow",
			Description: "Test workflow for E2E testing",
			Agents: []workflow.Agent{
				{
					ID:       "agent1",
					Provider: "mock",
					Model:    "test",
					Prompt:   "Test prompt",
					Output:   "test-output.txt",
					Settings: workflow.AgentSettings{
						Timeout: 30,
					},
				},
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Execute workflow with mock provider
		err := executor.Execute(ctx, wf, map[string]interface{}{})

		// Should succeed now that mock provider is supported
		assert.NoError(t, err)
	})

	t.Run("Workflow with Prompt Garden", func(t *testing.T) {
		// Create temporary directory and garden for this test
		tempDir := t.TempDir()
		gardenPath := filepath.Join(tempDir, "promptgarden")

		garden, err := promptgarden.NewGarden(gardenPath)
		require.NoError(t, err)

		// Add a test prompt to the garden
		prompt := &promptgarden.Prompt{
			ID:      "test-prompt",
			Name:    "test-prompt",
			Content: "This is a test prompt with {{variable}}",
			Metadata: promptgarden.PromptMetadata{
				Category: "test",
				Tags:     []string{"test"},
			},
		}
		err = garden.SavePrompt(prompt)
		require.NoError(t, err)

		// Create workflow using prompt garden
		wf := &workflow.Workflow{
			Name:        "test-workflow-garden",
			Description: "Test workflow with prompt garden",
			Agents: []workflow.Agent{
				{
					ID:       "agent1",
					Provider: "mock",
					Model:    "test",
					Prompt:   "promptgarden://test-prompt",
					Input: map[string]interface{}{
						"variable": "test-value",
					},
					Settings: workflow.AgentSettings{
						Timeout: 30,
					},
				},
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// Execute workflow with mock provider
		err = executor.Execute(ctx, wf, map[string]interface{}{})

		// Should succeed now that mock provider is supported
		assert.NoError(t, err)
	})
}

func TestWorkflowParser(t *testing.T) {
	t.Run("Parse YAML Workflow", func(t *testing.T) {
		yamlContent := `
name: test-workflow
description: Test workflow parsing

agents:
  - id: questioner
    provider: claude
    model: opus
    prompt: |
      Analyze this and generate questions
    output: QUESTIONS.md

  - id: planner
    provider: gemini
    model: pro
    prompt: promptgarden://planning-template
    input:
      questions: "{{file:QUESTIONS.md}}"
    output: PLAN.md
`

		parser := workflowexec.NewParser(t.TempDir())
		wf, err := parser.Parse([]byte(yamlContent))
		require.NoError(t, err)

		assert.Equal(t, "test-workflow", wf.Name)
		assert.Equal(t, "Test workflow parsing", wf.Description)
		assert.Len(t, wf.Agents, 2)

		// Check first agent
		assert.Equal(t, "questioner", wf.Agents[0].ID)
		assert.Equal(t, "claude", wf.Agents[0].Provider)
		assert.Equal(t, "opus", wf.Agents[0].Model)
		assert.Equal(t, "QUESTIONS.md", wf.Agents[0].Output)

		// Check second agent
		assert.Equal(t, "planner", wf.Agents[1].ID)
		assert.Equal(t, "gemini", wf.Agents[1].Provider)
		assert.Equal(t, "pro", wf.Agents[1].Model)
		assert.Equal(t, "promptgarden://planning-template", wf.Agents[1].Prompt)
		assert.Equal(t, "PLAN.md", wf.Agents[1].Output)
	})
}
