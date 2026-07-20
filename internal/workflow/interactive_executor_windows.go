//go:build windows

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
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"time"

	"github.com/rizome-dev/opun/pkg/workflow"
)

// Execute executes a workflow
func (e *InteractiveExecutor) Execute(ctx context.Context, wf *workflow.Workflow, variables map[string]interface{}) error {
	e.workflow = wf

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	e.cancelFunc = cancel

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)
	defer signal.Stop(sigChan)

	e.state = &workflow.ExecutionState{
		WorkflowID:   wf.Name,
		StartTime:    time.Now(),
		Status:       workflow.StatusRunning,
		AgentStates:  make(map[string]*workflow.AgentState),
		Variables:    variables,
		Outputs:      make(map[string]string),
		CurrentAgent: "",
	}

	if wf.Settings.OutputDir != "" {
		outputDir := wf.Settings.OutputDir
		timestamp := time.Now().Format("20060102-150405")
		outputDir = strings.ReplaceAll(outputDir, "{{timestamp}}", timestamp)
		e.outputDir = outputDir

		if err := os.MkdirAll(e.outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
		fmt.Printf("📁 Output directory: %s\n", e.outputDir)
	}

	fmt.Printf("\n🚀 Starting workflow: %s\n", wf.Name)
	if wf.Description != "" {
		fmt.Printf("📝 %s\n", wf.Description)
	}
	fmt.Printf("📋 %d agents to execute sequentially\n\n", len(wf.Agents))

	go func() {
		select {
		case sig := <-sigChan:
			fmt.Printf("\n\n⚠️  Received %s signal, stopping workflow...\n", sig)
			cancel()
		case <-ctx.Done():
		}
	}()

	for i, agent := range wf.Agents {
		select {
		case <-ctx.Done():
			e.state.Status = workflow.StatusAborted
			return fmt.Errorf("workflow canceled by user")
		default:
		}

		fmt.Printf("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Printf("🤖 Agent %d/%d: %s\n", i+1, len(wf.Agents), agent.Name)
		fmt.Printf("   Provider: %s | Model: %s\n", agent.Provider, agent.Model)
		fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

		usedVars := e.extractVariablesFromPrompt(agent.Prompt)

		if len(wf.Variables) > 0 && len(usedVars) > 0 {
			var promptVars []promptVariable
			for _, v := range wf.Variables {
				if !v.Internal && contains(usedVars, v.Name) {
					currentVal, exists := e.state.Variables[v.Name]
					if !exists {
						currentVal = v.DefaultValue
					}

					promptVars = append(promptVars, promptVariable{
						Name:         v.Name,
						Description:  v.Description,
						Type:         v.Type,
						Required:     v.Required,
						DefaultValue: v.DefaultValue,
						CurrentValue: currentVal,
					})
				}
			}

			if len(promptVars) > 0 {
				fmt.Printf("📝 Configure variables for %s:\n\n", agent.Name)

				updatedVars, err := promptForVariables(promptVars)
				if err != nil {
					fmt.Printf("⚠️  Using existing variable values\n\n")
				} else {
					for k, v := range updatedVars {
						e.state.Variables[k] = v
					}
					fmt.Printf("✅ Variables updated\n\n")
				}
			}
		}

		if err := e.executeAgent(ctx, &agent, i); err != nil {
			if ctx.Err() != nil {
				e.state.Status = workflow.StatusAborted
				return fmt.Errorf("workflow canceled during agent %s", agent.Name)
			}
			return fmt.Errorf("agent %s failed: %w", agent.Name, err)
		}

		if agent.Output != "" && e.outputDir != "" {
			outputPath := filepath.Join(e.outputDir, agent.Output)
			e.outputs[agent.ID] = outputPath
			fmt.Printf("💾 Output saved to: %s\n", outputPath)
			fmt.Printf("📌 Next agents can reference this as: {{%s.output}}\n", agent.ID)
		}

		e.handoffContext = append(e.handoffContext, fmt.Sprintf("Agent %s (%s) completed", agent.Name, agent.Provider))
	}

	endTime := time.Now()
	e.state.Status = workflow.StatusCompleted
	e.state.EndTime = &endTime

	fmt.Printf("\n✨ Workflow completed successfully!\n")
	return nil
}

// executeInteractiveAgent is not supported on Windows; agents run headless
func (e *InteractiveExecutor) executeInteractiveAgent(ctx context.Context, agent *workflow.Agent, agentIndex int) error {
	fmt.Printf("⚠️  Interactive mode is not supported on Windows, running %s headless\n", agent.Name)
	return e.executeHeadlessAgent(ctx, agent, agentIndex)
}

// getProviderCommandAndArgs returns the command and args to start a provider interactively
func (e *InteractiveExecutor) getProviderCommandAndArgs(provider string) (string, []string, error) {
	switch provider {
	case "claude", "gemini", "qwen":
		for _, candidate := range []string{provider, provider + ".exe", provider + ".cmd"} {
			if _, err := exec.LookPath(candidate); err == nil {
				return candidate, []string{}, nil
			}
		}
		return "", nil, fmt.Errorf("%s command not found, please install the %s CLI", provider, provider)

	case "mock":
		return "cmd.exe", []string{"/c", "echo Mock provider ready && more"}, nil

	default:
		return "", nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}
