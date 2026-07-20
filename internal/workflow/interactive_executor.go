//go:build !windows

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
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/creack/pty"
	"github.com/rizome-dev/opun/pkg/workflow"
	"golang.org/x/term"
)

// Execute executes a workflow
func (e *InteractiveExecutor) Execute(ctx context.Context, wf *workflow.Workflow, variables map[string]interface{}) error {
	e.workflow = wf

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	e.cancelFunc = cancel

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
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

// executeInteractiveAgent executes a single agent in an interactive PTY session
func (e *InteractiveExecutor) executeInteractiveAgent(ctx context.Context, agent *workflow.Agent, agentIndex int) error {
	e.ctrlCMutex.Lock()
	e.ctrlCCount = 0
	e.lastCtrlCTime = time.Time{}
	e.ctrlCMutex.Unlock()

	agentState := e.beginAgent(agent)

	fmt.Printf("⚡ Interactive session control:\n")
	fmt.Printf("   • Press Ctrl-C three times rapidly (within 1.2s) to abort the workflow\n\n")

	providerCmd, providerArgs, err := e.getProviderCommandAndArgs(agent.Provider)
	if err != nil {
		return e.handleAgentError(agent, agentState, err)
	}

	prompt, err := e.processPromptWithHandoff(agent.Prompt, agentIndex)
	if err != nil {
		return e.handleAgentError(agent, agentState, fmt.Errorf("failed to process prompt: %w", err))
	}

	if len(e.outputs) > 0 {
		fmt.Printf("📎 Prompt includes references to %d previous output(s)\n", len(e.outputs))
	}

	// #nosec G204 -- providerCmd is from a hardcoded list of known AI provider commands
	cmd := exec.Command(providerCmd, providerArgs...)
	cmd.Env = os.Environ()

	ptmx, err := pty.Start(cmd)
	if err != nil {
		return e.handleAgentError(agent, agentState, fmt.Errorf("failed to start PTY: %w", err))
	}
	defer ptmx.Close()

	if term.IsTerminal(int(os.Stdin.Fd())) {
		ch := make(chan os.Signal, 1)
		signal.Notify(ch, syscall.SIGWINCH)
		go func() {
			for range ch {
				if err := pty.InheritSize(os.Stdin, ptmx); err != nil {
					fmt.Fprintf(os.Stderr, "error resizing pty: %v\n", err)
				}
			}
		}()
		ch <- syscall.SIGWINCH
		defer func() { signal.Stop(ch); close(ch) }()
	}

	var oldState *term.State
	if term.IsTerminal(int(os.Stdin.Fd())) {
		oldState, err = term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			return e.handleAgentError(agent, agentState, fmt.Errorf("failed to set raw mode: %w", err))
		}
		defer func() {
			if oldState != nil {
				if err := term.Restore(int(os.Stdin.Fd()), oldState); err != nil {
					exec.Command("stty", "sane").Run()
				}
			}
		}()
	}

	outputBuffer := &strings.Builder{}
	promptInjected := false
	promptMutex := &sync.Mutex{}

	errChan := make(chan error, 2)
	doneChan := make(chan struct{})

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				os.Stdout.Write(buf[:n])

				promptMutex.Lock()
				outputBuffer.Write(buf[:n])

				currentOutput := outputBuffer.String()

				if !promptInjected && providerReady(agent.Provider, currentOutput) {
					promptInjected = true
					go injectPrompt(ptmx, agent.Provider, prompt)
				}
				promptMutex.Unlock()
			}

			if err != nil {
				select {
				case errChan <- err:
				case <-doneChan:
				}
				return
			}
		}
	}()

	go func() {
		buf := make([]byte, 1024)
		for {
			select {
			case <-doneChan:
				return
			default:
				n, err := os.Stdin.Read(buf)
				if err != nil {
					select {
					case errChan <- err:
					case <-doneChan:
					}
					return
				}

				for i := 0; i < n; i++ {
					if buf[i] == 0x03 {
						e.ctrlCMutex.Lock()
						now := time.Now()

						if now.Sub(e.lastCtrlCTime) > 1200*time.Millisecond {
							e.ctrlCCount = 0
						}

						e.ctrlCCount++
						e.lastCtrlCTime = now
						count := e.ctrlCCount
						e.ctrlCMutex.Unlock()

						if count >= 3 {
							if oldState != nil {
								if err := term.Restore(int(os.Stdin.Fd()), oldState); err != nil {
									exec.Command("stty", "sane").Run()
								}
								oldState = nil
							}
							fmt.Printf("\n\n🛑 Triple Ctrl+C detected, aborting entire workflow...\n")
							e.cancelFunc()
							return
						}
					}
				}

				if _, err := ptmx.Write(buf[:n]); err != nil {
					select {
					case errChan <- err:
					case <-doneChan:
					}
					return
				}
			}
		}
	}()

	select {
	case err := <-errChan:
		close(doneChan)
		if err != nil && err != io.EOF {
			return e.handleAgentError(agent, agentState, err)
		}
	case <-ctx.Done():
		close(doneChan)

		if oldState != nil {
			if err := term.Restore(int(os.Stdin.Fd()), oldState); err != nil {
				exec.Command("stty", "sane").Run()
			}
			oldState = nil
		}

		if cmd.Process != nil {
			cmd.Process.Signal(os.Interrupt)
			time.Sleep(100 * time.Millisecond)

			if cmd.ProcessState == nil {
				cmd.Process.Kill()
			}
		}

		endTime := time.Now()
		agentState.Status = workflow.StatusAborted
		agentState.EndTime = &endTime

		return ctx.Err()
	}

	e.completeAgent(agentState, "")

	fmt.Printf("\n✅ %s session completed\n", agent.Name)
	return nil
}

var ansiEscapes = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func providerReady(provider, output string) bool {
	switch provider {
	case "claude":
		if !strings.Contains(output, "│") || !strings.Contains(output, ">") {
			return false
		}
		return strings.Contains(output, "│\u00a0>") ||
			strings.Contains(output, "│ >") ||
			strings.Contains(output, "\u00a0>\u00a0")
	case "gemini", "qwen":
		stripped := ansiEscapes.ReplaceAllString(output, "")
		return strings.Contains(stripped, "Type your message") ||
			strings.Contains(stripped, "│ > ")
	case "mock":
		return strings.Contains(output, "Mock provider ready")
	default:
		return false
	}
}

func injectPrompt(ptmx *os.File, provider, prompt string) {
	settleDelay := 500 * time.Millisecond
	charDelay := 5 * time.Millisecond
	if provider == "gemini" || provider == "qwen" {
		settleDelay = 2 * time.Second
		charDelay = 10 * time.Millisecond
	}

	time.Sleep(settleDelay)
	for _, char := range prompt {
		ptmx.Write([]byte(string(char)))
		time.Sleep(charDelay)
	}
}

// getProviderCommandAndArgs returns the command and args to start a provider interactively
func (e *InteractiveExecutor) getProviderCommandAndArgs(provider string) (string, []string, error) {
	switch provider {
	case "claude", "gemini", "qwen":
		if _, err := exec.LookPath(provider); err == nil {
			return provider, []string{}, nil
		}
		return "", nil, fmt.Errorf("%s command not found, please install the %s CLI", provider, provider)

	case "mock":
		return "/bin/sh", []string{"-c", "echo 'Mock provider ready'; cat"}, nil

	default:
		return "", nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}
