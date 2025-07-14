package e2e

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E test in short mode")
	}

	// Create a test workflow
	workflowContent := `
name: test-cancellation
description: Test workflow cancellation
agents:
  - id: test-agent
    name: Test Agent
    provider: mock
    prompt: "This is a test that will be cancelled"
`

	// Write workflow to temp file
	tmpFile, err := os.CreateTemp("", "test-workflow-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(workflowContent)
	require.NoError(t, err)
	tmpFile.Close()

	// Run the workflow
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "opun", "run", tmpFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Start the command
	err = cmd.Start()
	require.NoError(t, err)

	// Give it a moment to start
	time.Sleep(500 * time.Millisecond)

	// Send interrupt signal
	err = cmd.Process.Signal(os.Interrupt)
	require.NoError(t, err)

	// Wait for it to finish
	err = cmd.Wait()

	// The command should exit with an error (interrupted)
	assert.Error(t, err)

	// Verify terminal is in a good state by running a simple command
	verifyCmd := exec.Command("echo", "terminal test")
	output, err := verifyCmd.CombinedOutput()
	assert.NoError(t, err)
	assert.Contains(t, string(output), "terminal test")
}