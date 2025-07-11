package workflow

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
	"strings"

	"github.com/rizome-dev/opun/pkg/workflow"
	"gopkg.in/yaml.v3"
)

// Manager manages workflows
type Manager struct {
	workflowDir string
}

// NewManager creates a new workflow manager
func NewManager(workflowDir string) (*Manager, error) {
	if workflowDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		workflowDir = filepath.Join(home, ".opun", "workflows")
	}

	// Ensure directory exists
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create workflow directory: %w", err)
	}

	return &Manager{
		workflowDir: workflowDir,
	}, nil
}

// ListWorkflows returns a list of available workflows
func (m *Manager) ListWorkflows() ([]*workflow.Workflow, error) {
	entries, err := os.ReadDir(m.workflowDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow directory: %w", err)
	}

	var workflows []*workflow.Workflow
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		path := filepath.Join(m.workflowDir, entry.Name())
		wf, err := m.loadWorkflow(path)
		if err != nil {
			// Skip invalid workflows
			continue
		}

		workflows = append(workflows, wf)
	}

	return workflows, nil
}

// Execute runs a workflow by name
func (m *Manager) Execute(ctx context.Context, name string, variables map[string]interface{}) (interface{}, error) {
	// Find workflow file
	workflowPath := filepath.Join(m.workflowDir, name+".yaml")
	if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
		// Try without extension
		workflowPath = filepath.Join(m.workflowDir, name)
		if _, err := os.Stat(workflowPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("workflow not found: %s", name)
		}
	}

	// Load workflow
	wf, err := m.loadWorkflow(workflowPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load workflow: %w", err)
	}

	// Create executor
	executor := NewExecutor()

	// Convert variables to string map if needed
	stringVars := make(map[string]interface{})
	if variables != nil {
		stringVars = variables
	}

	// Execute workflow
	result := executor.Execute(ctx, wf, stringVars)

	return result, nil
}

// loadWorkflow loads a workflow from a file
func (m *Manager) loadWorkflow(path string) (*workflow.Workflow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var wf workflow.Workflow
	if err := yaml.Unmarshal(data, &wf); err != nil {
		return nil, fmt.Errorf("failed to parse workflow: %w", err)
	}

	// Set name from filename if not specified
	if wf.Name == "" {
		base := filepath.Base(path)
		wf.Name = strings.TrimSuffix(base, filepath.Ext(base))
	}

	return &wf, nil
}
