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
	"fmt"
	"os"
	"path/filepath"
	"strings"

	wf "github.com/rizome-dev/opun/pkg/workflow"
	"gopkg.in/yaml.v3"
)

// Parser handles parsing of workflow definitions
type Parser struct {
	workflowDir string
}

// NewParser creates a new workflow parser
func NewParser(workflowDir string) *Parser {
	return &Parser{
		workflowDir: workflowDir,
	}
}

// ParseFile parses a workflow from a YAML file
func (p *Parser) ParseFile(filePath string) (*wf.Workflow, error) {
	// #nosec G304 -- file path is provided by user for their workflow files
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow file: %w", err)
	}

	return p.Parse(data)
}

// Parse parses a workflow from YAML data
func (p *Parser) Parse(data []byte) (*wf.Workflow, error) {
	var workflow wf.Workflow

	// Parse YAML
	if err := yaml.Unmarshal(data, &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	// Validate workflow
	if err := p.validate(&workflow); err != nil {
		return nil, fmt.Errorf("workflow validation failed: %w", err)
	}

	// Process agents
	if err := p.processAgents(&workflow); err != nil {
		return nil, fmt.Errorf("failed to process agents: %w", err)
	}

	return &workflow, nil
}

// ParseYAMLExample parses the example from INIT.md
func (p *Parser) ParseYAMLExample(data string) (*wf.Workflow, error) {
	// Handle the simplified format from INIT.md
	// This format uses a more concise agent definition

	var rawData map[string]interface{}
	if err := yaml.Unmarshal([]byte(data), &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	wf := &wf.Workflow{
		Agents: []wf.Agent{},
	}

	// Extract command
	if cmd, ok := rawData["command"].(string); ok {
		wf.Command = cmd
		wf.Name = cmd
	}

	// Process agents in order
	agentOrder := []string{}
	for key := range rawData {
		if strings.HasPrefix(key, "agent") {
			agentOrder = append(agentOrder, key)
		}
	}

	// Sort agent keys
	for i := 1; i <= len(agentOrder); i++ {
		key := fmt.Sprintf("agent%d", i)
		if agentData, ok := rawData[key].(map[string]interface{}); ok {
			agent := p.parseSimpleAgent(key, agentData)
			wf.Agents = append(wf.Agents, agent)
		}
	}

	return wf, nil
}

// parseSimpleAgent parses a simplified agent definition
func (p *Parser) parseSimpleAgent(id string, data map[string]interface{}) wf.Agent {
	agent := wf.Agent{
		ID: id,
	}

	if provider, ok := data["provider"].(string); ok {
		agent.Provider = provider
	}

	if model, ok := data["model"].(string); ok {
		agent.Model = model
	}

	if prompt, ok := data["prompt"].(string); ok {
		agent.Prompt = prompt
	}

	// Handle output specification
	if output, ok := data["output"].(string); ok {
		agent.Output = output
	}

	// Handle input specification
	if input, ok := data["input"].(map[string]interface{}); ok {
		agent.Input = input
	}

	return agent
}

// validate validates a workflow definition
func (p *Parser) validate(wf *wf.Workflow) error {
	// Validate basic fields
	if wf.Name == "" {
		return fmt.Errorf("workflow name is required")
	}

	if len(wf.Agents) == 0 {
		return fmt.Errorf("workflow must have at least one agent")
	}

	// Validate agents
	agentIDs := make(map[string]bool)
	for i, agent := range wf.Agents {
		// Generate ID if not provided
		if agent.ID == "" {
			agent.ID = fmt.Sprintf("agent%d", i+1)
			wf.Agents[i].ID = agent.ID
		}

		// Check for duplicate IDs
		if agentIDs[agent.ID] {
			return fmt.Errorf("duplicate agent ID: %s", agent.ID)
		}
		agentIDs[agent.ID] = true

		// Validate agent fields
		if agent.Provider == "" {
			return fmt.Errorf("agent %s: provider is required", agent.ID)
		}

		if agent.Prompt == "" {
			return fmt.Errorf("agent %s: prompt is required", agent.ID)
		}

		// Validate dependencies
		for _, dep := range agent.DependsOn {
			if !agentIDs[dep] {
				return fmt.Errorf("agent %s: unknown dependency %s", agent.ID, dep)
			}
		}
	}

	return nil
}

// processAgents processes agent definitions
func (p *Parser) processAgents(wf *wf.Workflow) error {
	for i := range wf.Agents {
		agent := &wf.Agents[i]

		// Set default values
		if agent.Settings.Temperature == 0 {
			agent.Settings.Temperature = 0.7
		}

		if agent.Settings.Timeout == 0 {
			agent.Settings.Timeout = 300 // 5 minutes default
		}

		// Process prompt references
		agent.Prompt = p.processPromptReference(agent.Prompt)

		// If no explicit dependencies, depend on previous agent
		if len(agent.DependsOn) == 0 && i > 0 {
			agent.DependsOn = []string{wf.Agents[i-1].ID}
		}
	}

	return nil
}

// processPromptReference processes prompt references like promptgarden://name
func (p *Parser) processPromptReference(prompt string) string {
	// For now, just return as-is
	// Later this can resolve prompt garden references
	return prompt
}

// LoadWorkflow loads a workflow by name
func (p *Parser) LoadWorkflow(name string) (*wf.Workflow, error) {
	// Try different file extensions
	extensions := []string{".yaml", ".yml", ".json"}

	for _, ext := range extensions {
		filePath := filepath.Join(p.workflowDir, name+ext)
		if _, err := os.Stat(filePath); err == nil {
			return p.ParseFile(filePath)
		}
	}

	return nil, fmt.Errorf("workflow not found: %s", name)
}

// ListWorkflows lists all available workflows
func (p *Parser) ListWorkflows() ([]string, error) {
	workflows := []string{}

	entries, err := os.ReadDir(p.workflowDir)
	if err != nil {
		if os.IsNotExist(err) {
			return workflows, nil
		}
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		ext := filepath.Ext(name)
		if ext == ".yaml" || ext == ".yml" || ext == ".json" {
			workflows = append(workflows, strings.TrimSuffix(name, ext))
		}
	}

	return workflows, nil
}
