package cli

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
	"os"
	"path/filepath"
	"strings"

	"github.com/rizome-dev/opun/internal/promptgarden"
	"github.com/rizome-dev/opun/internal/tools"
	"github.com/rizome-dev/opun/internal/utils"
	"github.com/rizome-dev/opun/internal/workflow"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// AddCmd creates the add command
func AddCmd() *cobra.Command {
	var (
		path       string
		name       string
		asWorkflow bool
		asPrompt   bool
		asAction   bool
	)

	cmd := &cobra.Command{
		Use:   "add [workflow|prompt|action]",
		Short: "Add workflows, prompts, or actions to Opun",
		Long: `Add workflows, prompts, or actions to Opun for use across AI providers.

Examples:
  # Add a workflow
  opun add workflow --path workflow.yaml --name my-workflow
  
  # Add a prompt
  opun add prompt --path prompt.txt --name my-prompt
  
  # Add an action
  opun add action --path action.yaml --name my-action
  
  # Interactive mode
  opun add`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if subcommand is provided as positional argument
			if len(args) > 0 {
				switch args[0] {
				case "workflow":
					asWorkflow = true
				case "prompt":
					asPrompt = true
				case "action":
					asAction = true
				default:
					return fmt.Errorf("unknown type: %s", args[0])
				}
			}

			// If no flags or args provided, run interactive mode
			if !asWorkflow && !asPrompt && !asAction {
				return runInteractiveAdd()
			}

			// Validate required fields
			if path == "" {
				return fmt.Errorf("--path is required")
			}

			if name == "" {
				return fmt.Errorf("--name is required")
			}

			if asWorkflow {
				return addWorkflow(path, name)
			}

			if asPrompt {
				return addPrompt(path, name)
			}

			if asAction {
				return addActionFromFile(path, name)
			}

			return fmt.Errorf("specify either workflow, prompt, or action")
		},
	}

	// Flags
	cmd.Flags().BoolVar(&asWorkflow, "workflow", false, "Add a workflow")
	cmd.Flags().BoolVar(&asPrompt, "prompt", false, "Add a prompt")
	cmd.Flags().BoolVar(&asAction, "action", false, "Add an action")
	cmd.Flags().StringVar(&path, "path", "", "path to file")
	cmd.Flags().StringVar(&name, "name", "", "name for the item")

	// Only one type can be used at a time
	cmd.MarkFlagsMutuallyExclusive("workflow", "prompt", "action")

	return cmd
}

// copyDir recursively copies a directory
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate destination path
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return utils.EnsureDir(dstPath)
		}

		// Copy file
		return copyFile(path, dstPath)
	})
}

// copyFile copies a single file
func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

// addWorkflow adds a workflow to the system
func addWorkflow(path, name string) error {
	// Read workflow file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read workflow file: %w", err)
	}

	// Get workflow directory first
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	workflowDir := filepath.Join(home, ".opun", "workflows")

	// Parse workflow to validate it
	parser := workflow.NewParser(workflowDir)
	wf, err := parser.Parse(data)
	if err != nil {
		return fmt.Errorf("invalid workflow format: %w", err)
	}

	// Set the command name
	wf.Command = name
	if err := utils.EnsureDir(workflowDir); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied: cannot create %s\nTry: sudo chown -R $USER ~/.opun", workflowDir)
		}
		return fmt.Errorf("failed to create workflow directory: %w", err)
	}

	// Save workflow
	destPath := filepath.Join(workflowDir, name+".yaml")
	if err := utils.WriteFile(destPath, data); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied: cannot write to %s\nTry: sudo chown -R $USER ~/.opun", workflowDir)
		}
		return fmt.Errorf("failed to save workflow: %w", err)
	}

	fmt.Printf("✓ Added workflow '%s' as /%s command\n", name, name)
	fmt.Printf("  Saved to: %s\n", destPath)

	// Update config to register the workflow
	workflows := viper.GetStringSlice("workflows")
	workflows = append(workflows, name)
	viper.Set("workflows", workflows)

	// Save config
	configPath := viper.ConfigFileUsed()
	if configPath == "" {
		configPath = filepath.Join(home, ".opun", "config.yaml")
	}

	if err := viper.WriteConfigAs(configPath); err != nil {
		return fmt.Errorf("failed to update config: %w", err)
	}

	return nil
}

// addPrompt adds a prompt to the prompt garden
func addPrompt(path, name string) error {
	// Read prompt file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read prompt file: %w", err)
	}

	// Get prompt garden
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	gardenPath := filepath.Join(home, ".opun", "promptgarden")
	garden, err := promptgarden.NewGarden(gardenPath)
	if err != nil {
		return fmt.Errorf("failed to access prompt garden: %w", err)
	}

	// Create prompt
	prompt := &promptgarden.Prompt{
		ID:      name,
		Name:    name,
		Content: string(data),
		Metadata: promptgarden.PromptMetadata{
			Tags:        extractTags(string(data)),
			Category:    "user",
			Version:     "1.0.0",
			Description: fmt.Sprintf("Prompt added from %s", filepath.Base(path)),
		},
	}

	// Save prompt
	if err := garden.SavePrompt(prompt); err != nil {
		return fmt.Errorf("failed to save prompt: %w", err)
	}

	fmt.Printf("✓ Added prompt '%s' to prompt garden\n", name)
	fmt.Printf("  Access with: promptgarden://%s\n", name)

	return nil
}

// extractTags extracts tags from prompt content (looks for #tag patterns)
func extractTags(content string) []string {
	var tags []string
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		if strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "##") {
			// This might be a tag line
			parts := strings.Fields(line)
			for _, part := range parts {
				if strings.HasPrefix(part, "#") && len(part) > 1 {
					tag := strings.TrimPrefix(part, "#")
					tags = append(tags, tag)
				}
			}
		}
	}

	return tags
}

// addActionFromFile adds an action to the system from a file
func addActionFromFile(path, name string) error {
	// Read tool file
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read action file: %w", err)
	}

	// Get tools directory
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	actionsDir := filepath.Join(home, ".opun", "actions")

	// Create action loader
	loader := tools.NewLoader(actionsDir)

	// Parse the tool to validate it
	tempFile := filepath.Join(os.TempDir(), "temp-action.yaml")
	if err := utils.WriteFile(tempFile, data); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}
	defer os.Remove(tempFile)

	if err := loader.LoadFile(tempFile); err != nil {
		return fmt.Errorf("invalid tool format: %w", err)
	}

	// Save tool with the given name
	destPath := filepath.Join(actionsDir, name+".yaml")
	if err := utils.EnsureDir(actionsDir); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied: cannot create %s\nTry: sudo chown -R $USER ~/.opun", actionsDir)
		}
		return fmt.Errorf("failed to create tools directory: %w", err)
	}

	if err := utils.WriteFile(destPath, data); err != nil {
		if os.IsPermission(err) {
			return fmt.Errorf("permission denied: cannot write to %s\nTry: sudo chown -R $USER ~/.opun", actionsDir)
		}
		return fmt.Errorf("failed to save tool: %w", err)
	}

	fmt.Printf("✓ Added tool '%s'\n", name)
	fmt.Printf("  Saved to: %s\n", destPath)
	fmt.Printf("  Tool will be available across all AI providers\n")

	return nil
}

// Legacy code removed - see add_interactive.go for new implementation

// runInteractiveAdd runs the interactive add flow
func runInteractiveAdd() error {
	return RunInteractiveAdd()
}
