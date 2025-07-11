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
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/rizome-dev/opun/internal/promptgarden"
	"github.com/rizome-dev/opun/internal/utils"
	"github.com/rizome-dev/opun/pkg/plugin"
	"gopkg.in/yaml.v3"
)

// RemoteManifest represents a manifest file that can be fetched from a URL
// It contains collections of prompts, workflows, actions, and tools to install
type RemoteManifest struct {
	Name        string                `yaml:"name"`
	Version     string                `yaml:"version"`
	Description string                `yaml:"description"`
	Author      string                `yaml:"author"`
	Repository  string                `yaml:"repository"`
	Imports     *plugin.PluginImports `yaml:"imports"`
	sourceURL   string                // URL it was loaded from
}

// LoadManifestFromURL downloads and parses a manifest from a URL
func LoadManifestFromURL(url string) (*RemoteManifest, error) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "opun-manifest-*.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Download the file
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download manifest: HTTP %d", resp.StatusCode)
	}

	// Copy to temp file
	_, err = io.Copy(tmpFile, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to save manifest: %w", err)
	}

	// Close the file before reading it
	tmpFile.Close()

	// Read and parse the manifest
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest RemoteManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	// Validate required fields
	if manifest.Name == "" {
		return nil, fmt.Errorf("manifest name is required")
	}
	if manifest.Imports == nil {
		return nil, fmt.Errorf("manifest must have an 'imports' section")
	}

	manifest.sourceURL = url
	return &manifest, nil
}

// RemoteInstaller handles installing items from remote manifests
type RemoteInstaller struct {
	baseDir string
}

// NewRemoteInstaller creates a new remote installer
func NewRemoteInstaller(baseDir string) (*RemoteInstaller, error) {
	if err := utils.EnsureDir(baseDir); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}

	return &RemoteInstaller{
		baseDir: baseDir,
	}, nil
}

// InstallFromURL downloads and installs items from a manifest URL
func (r *RemoteInstaller) InstallFromURL(url string) error {
	// Load the manifest
	manifest, err := LoadManifestFromURL(url)
	if err != nil {
		return err
	}

	return r.InstallManifest(manifest)
}

// InstallManifest installs all items from a manifest
func (r *RemoteInstaller) InstallManifest(manifest *RemoteManifest) error {
	if manifest.Imports == nil {
		return fmt.Errorf("manifest has no imports")
	}

	counts := plugin.ItemCount{}

	// Import prompts
	for _, prompt := range manifest.Imports.Prompts {
		if err := r.installPrompt(prompt); err != nil {
			return fmt.Errorf("failed to import prompt %s: %w", prompt.Name, err)
		}
		counts.Prompts++
	}

	// Import workflows
	for _, workflow := range manifest.Imports.Workflows {
		if err := r.installWorkflow(workflow); err != nil {
			return fmt.Errorf("failed to import workflow %s: %w", workflow.Name, err)
		}
		counts.Workflows++
	}

	// Import actions
	for _, action := range manifest.Imports.Actions {
		if err := r.installAction(action); err != nil {
			return fmt.Errorf("failed to import action %s: %w", action.Name, err)
		}
		counts.Actions++
	}

	// TODO: Import tools when MCP server support is added

	// Record the installation
	if err := r.recordInstallation(manifest, counts); err != nil {
		return fmt.Errorf("failed to record installation: %w", err)
	}

	return nil
}

// installPrompt installs a prompt to the promptgarden
func (r *RemoteInstaller) installPrompt(prompt plugin.PromptImport) error {
	gardenPath := filepath.Join(r.baseDir, "promptgarden")
	garden, err := promptgarden.NewGarden(gardenPath)
	if err != nil {
		return fmt.Errorf("failed to initialize prompt garden: %w", err)
	}

	// Create prompt object
	promptObj := &promptgarden.Prompt{
		ID:      prompt.Name,
		Content: prompt.Template,
		Metadata: promptgarden.PromptMetadata{
			Description: prompt.Description,
			Category:    prompt.Category,
			Version:     "1.0.0",
			Tags:        []string{prompt.Category}, // Use category as a tag
		},
	}

	// Save the prompt
	if err := garden.SavePrompt(promptObj); err != nil {
		return fmt.Errorf("failed to save prompt: %w", err)
	}

	return nil
}

// installWorkflow installs a workflow
func (r *RemoteInstaller) installWorkflow(wf plugin.WorkflowImport) error {
	workflowsDir := filepath.Join(r.baseDir, "workflows")
	if err := utils.EnsureDir(workflowsDir); err != nil {
		return fmt.Errorf("failed to create workflows directory: %w", err)
	}

	// Create workflow structure
	workflowData := map[string]interface{}{
		"command":     wf.Name,
		"description": wf.Description,
		"agents":      wf.Agents, // Already in the right format
	}

	// Marshal to YAML
	data, err := yaml.Marshal(workflowData)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow: %w", err)
	}

	// Save workflow file
	path := filepath.Join(workflowsDir, wf.Name+".yaml")
	return utils.WriteFile(path, data)
}

// installAction installs an action
func (r *RemoteInstaller) installAction(action plugin.ActionImport) error {
	actionsDir := filepath.Join(r.baseDir, "actions")
	if err := utils.EnsureDir(actionsDir); err != nil {
		return fmt.Errorf("failed to create actions directory: %w", err)
	}

	// Create action data structure
	actionData := map[string]interface{}{
		"id":          action.Name,
		"name":        action.Name,
		"description": action.Description,
		"category":    action.Category,
		"command":     action.Command,
	}

	// Add arguments if present
	if len(action.Arguments) > 0 {
		actionData["arguments"] = action.Arguments
	}

	// Add providers if specified
	if len(action.Providers) > 0 {
		actionData["providers"] = action.Providers
	}

	// Marshal to YAML
	data, err := yaml.Marshal(actionData)
	if err != nil {
		return fmt.Errorf("failed to marshal action: %w", err)
	}

	// Save action file
	path := filepath.Join(actionsDir, action.Name+".yaml")
	return utils.WriteFile(path, data)
}

// recordInstallation records that a manifest was installed
func (r *RemoteInstaller) recordInstallation(manifest *RemoteManifest, counts plugin.ItemCount) error {
	installsDir := filepath.Join(r.baseDir, "installs")
	if err := utils.EnsureDir(installsDir); err != nil {
		return fmt.Errorf("failed to create installs directory: %w", err)
	}

	installRecord := map[string]interface{}{
		"name":         manifest.Name,
		"version":      manifest.Version,
		"description":  manifest.Description,
		"author":       manifest.Author,
		"repository":   manifest.Repository,
		"source_url":   manifest.sourceURL,
		"installed_at": time.Now().Format(time.RFC3339),
		"counts": map[string]int{
			"prompts":   counts.Prompts,
			"workflows": counts.Workflows,
			"actions":   counts.Actions,
		},
	}

	// Save installation record
	data, err := yaml.Marshal(installRecord)
	if err != nil {
		return fmt.Errorf("failed to marshal install record: %w", err)
	}

	filename := fmt.Sprintf("%s-%s.yaml", manifest.Name, time.Now().Format("20060102-150405"))
	path := filepath.Join(installsDir, filename)
	return utils.WriteFile(path, data)
}
