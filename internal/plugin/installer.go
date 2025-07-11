package plugin

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
	"time"

	"github.com/rizome-dev/opun/internal/promptgarden"
	"github.com/rizome-dev/opun/internal/utils"
	"github.com/rizome-dev/opun/pkg/core"
	"github.com/rizome-dev/opun/pkg/plugin"
	"github.com/rizome-dev/opun/pkg/workflow"
	"gopkg.in/yaml.v3"
)

// Installer handles plugin installation
type Installer struct {
	baseDir string
}

// NewInstaller creates a new plugin installer
func NewInstaller(baseDir string) (*Installer, error) {
	return &Installer{
		baseDir: baseDir,
	}, nil
}

// Install installs a plugin and imports all its items
func (i *Installer) Install(p *ImportPlugin) error {
	imports := p.GetImports()
	if imports == nil {
		return fmt.Errorf("plugin has no imports section. Import plugins must have an 'imports' section with prompts, workflows, or actions")
	}

	counts := plugin.ItemCount{}

	// Import prompts
	for _, prompt := range imports.Prompts {
		if err := i.importPrompt(prompt); err != nil {
			return fmt.Errorf("failed to import prompt %s: %w", prompt.Name, err)
		}
		counts.Prompts++
	}

	// Import workflows
	for _, wf := range imports.Workflows {
		if err := i.importWorkflow(wf); err != nil {
			return fmt.Errorf("failed to import workflow %s: %w", wf.Name, err)
		}
		counts.Workflows++
	}

	// Import actions
	for _, action := range imports.Actions {
		if err := i.importAction(action); err != nil {
			return fmt.Errorf("failed to import action %s: %w", action.Name, err)
		}
		counts.Actions++
	}

	// Record installation
	if err := i.recordInstallation(p, counts); err != nil {
		return fmt.Errorf("failed to record installation: %w", err)
	}

	return nil
}

// Uninstall removes a plugin and all its imported items
func (i *Installer) Uninstall(pluginName string) error {
	// Load installation record
	record, err := i.getInstallationRecord(pluginName)
	if err != nil {
		return fmt.Errorf("plugin not found: %w", err)
	}

	// TODO: Track which items belong to which plugin for proper cleanup
	// For now, we'll just remove the installation record
	_ = record // Suppress unused variable warning
	return i.removeInstallationRecord(pluginName)
}

// List returns all installed plugins
func (i *Installer) List() ([]plugin.InstalledPlugin, error) {
	installedFile := filepath.Join(i.baseDir, "plugins", "installed.yaml")

	data, err := os.ReadFile(installedFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []plugin.InstalledPlugin{}, nil
		}
		return nil, fmt.Errorf("failed to read installed plugins: %w", err)
	}

	var installed struct {
		Plugins []plugin.InstalledPlugin `yaml:"plugins"`
	}
	if err := yaml.Unmarshal(data, &installed); err != nil {
		return nil, fmt.Errorf("failed to parse installed plugins: %w", err)
	}

	return installed.Plugins, nil
}

// importPrompt imports a prompt into the promptgarden
func (i *Installer) importPrompt(p plugin.PromptImport) error {
	promptGardenPath := filepath.Join(i.baseDir, "promptgarden")
	promptStore := promptgarden.NewFileStore(promptGardenPath)

	// Convert variables
	var variables []core.PromptVariable
	for _, v := range p.Variables {
		variables = append(variables, core.PromptVariable{
			Name:         v.Name,
			Description:  v.Description,
			DefaultValue: v.Default,
		})
	}

	// Create metadata
	metadata := core.PromptMetadata{
		ID:          fmt.Sprintf("plugin-%s", sanitizeName(p.Name)),
		Name:        p.Name,
		Description: p.Description,
		Type:        core.PromptTypeTemplate,
		Category:    p.Category,
		Version:     "1.0.0",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Variables:   variables,
	}

	// Create template prompt
	templatePrompt := promptgarden.NewTemplatePrompt(metadata, p.Template)

	return promptStore.Create(templatePrompt)
}

// importWorkflow imports a workflow
func (i *Installer) importWorkflow(wf plugin.WorkflowImport) error {
	workflowDir := filepath.Join(i.baseDir, "workflows")
	if err := utils.EnsureDir(workflowDir); err != nil {
		return fmt.Errorf("failed to create workflows directory: %w", err)
	}

	// Create workflow definition
	workflowDef := workflow.Workflow{
		Name:        wf.Name,
		Description: wf.Description,
		Agents:      convertAgents(wf.Agents),
	}

	// Save workflow to file
	filename := fmt.Sprintf("%s.yaml", sanitizeName(wf.Name))
	path := filepath.Join(workflowDir, filename)

	data, err := yaml.Marshal(workflowDef)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}

	return utils.WriteFile(path, data)
}

// importAction imports an action
func (i *Installer) importAction(a plugin.ActionImport) error {
	actionsDir := filepath.Join(i.baseDir, "actions")
	if err := utils.EnsureDir(actionsDir); err != nil {
		return fmt.Errorf("failed to create actions directory: %w", err)
	}

	// Create action definition as a map for YAML marshaling
	action := map[string]interface{}{
		"id":          fmt.Sprintf("plugin-%s", sanitizeName(a.Name)),
		"name":        a.Name,
		"description": a.Description,
		"category":    a.Category,
		"command":     a.Command,
		"arguments":   a.Arguments,
		"providers":   a.Providers,
		"version":     "1.0.0",
	}

	// Save action to file
	filename := fmt.Sprintf("%s.yaml", sanitizeName(a.Name))
	path := filepath.Join(actionsDir, filename)

	data, err := yaml.Marshal(action)
	if err != nil {
		return fmt.Errorf("failed to marshal action: %w", err)
	}

	return utils.WriteFile(path, data)
}

// recordInstallation records that a plugin was installed
func (i *Installer) recordInstallation(p *ImportPlugin, counts plugin.ItemCount) error {
	pluginDir := filepath.Join(i.baseDir, "plugins")
	if err := utils.EnsureDir(pluginDir); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}

	installedFile := filepath.Join(pluginDir, "installed.yaml")

	// Load existing installations
	var installed struct {
		Plugins []plugin.InstalledPlugin `yaml:"plugins"`
	}

	data, err := os.ReadFile(installedFile)
	if err == nil {
		yaml.Unmarshal(data, &installed)
	}

	// Add new installation
	record := plugin.InstalledPlugin{
		Name:        p.Name(),
		Source:      p.GetPath(),
		InstalledAt: time.Now(),
		ItemCount:   counts,
	}

	// Remove existing record if present
	filtered := []plugin.InstalledPlugin{}
	for _, existing := range installed.Plugins {
		if existing.Name != p.Name() {
			filtered = append(filtered, existing)
		}
	}
	installed.Plugins = append(filtered, record)

	// Save updated list
	data, err = yaml.Marshal(installed)
	if err != nil {
		return fmt.Errorf("failed to marshal installation record: %w", err)
	}

	return utils.WriteFile(installedFile, data)
}

// getInstallationRecord retrieves installation info for a plugin
func (i *Installer) getInstallationRecord(pluginName string) (*plugin.InstalledPlugin, error) {
	plugins, err := i.List()
	if err != nil {
		return nil, err
	}

	for _, p := range plugins {
		if p.Name == pluginName {
			return &p, nil
		}
	}

	return nil, fmt.Errorf("plugin not found: %s", pluginName)
}

// removeInstallationRecord removes a plugin from the installed list
func (i *Installer) removeInstallationRecord(pluginName string) error {
	plugins, err := i.List()
	if err != nil {
		return err
	}

	filtered := []plugin.InstalledPlugin{}
	found := false
	for _, p := range plugins {
		if p.Name != pluginName {
			filtered = append(filtered, p)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("plugin not found: %s", pluginName)
	}

	// Save updated list
	installedFile := filepath.Join(i.baseDir, "plugins", "installed.yaml")
	installed := struct {
		Plugins []plugin.InstalledPlugin `yaml:"plugins"`
	}{
		Plugins: filtered,
	}

	data, err := yaml.Marshal(installed)
	if err != nil {
		return fmt.Errorf("failed to marshal installation record: %w", err)
	}

	return utils.WriteFile(installedFile, data)
}

// sanitizeName converts a name to a filesystem-safe string
func sanitizeName(name string) string {
	// Convert to lowercase and replace spaces/special chars with hyphens
	sanitized := strings.ToLower(name)
	sanitized = strings.ReplaceAll(sanitized, " ", "-")
	sanitized = strings.ReplaceAll(sanitized, "_", "-")
	sanitized = strings.ReplaceAll(sanitized, "/", "-")
	sanitized = strings.ReplaceAll(sanitized, ".", "-")

	// Remove any double hyphens
	for strings.Contains(sanitized, "--") {
		sanitized = strings.ReplaceAll(sanitized, "--", "-")
	}

	// Trim hyphens from start/end
	sanitized = strings.Trim(sanitized, "-")

	return sanitized
}

// convertAgents converts plugin workflow agents to workflow agents
func convertAgents(agents []map[string]interface{}) []workflow.Agent {
	var result []workflow.Agent
	for _, a := range agents {
		agent := workflow.Agent{
			Name:      getStringValue(a, "name"),
			Prompt:    getStringValue(a, "prompt"),
			Provider:  getStringValue(a, "provider"),
			Model:     getStringValue(a, "model"),
			DependsOn: getStringSliceValue(a, "depends_on"),
			Input:     getMapValue(a, "variables"), // Variables map to Input
		}
		result = append(result, agent)
	}
	return result
}

// getStringValue safely gets a string value from a map
func getStringValue(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// getStringSliceValue safely gets a string slice from a map
func getStringSliceValue(m map[string]interface{}, key string) []string {
	if v, ok := m[key]; ok {
		if slice, ok := v.([]string); ok {
			return slice
		}
		if iface, ok := v.([]interface{}); ok {
			var result []string
			for _, item := range iface {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return nil
}

// getMapValue safely gets a map value from a map
func getMapValue(m map[string]interface{}, key string) map[string]interface{} {
	if v, ok := m[key]; ok {
		if mapVal, ok := v.(map[string]interface{}); ok {
			return mapVal
		}
	}
	return nil
}
