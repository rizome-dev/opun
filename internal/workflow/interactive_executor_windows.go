//go:build windows

package workflow

import (
	"context"
	"fmt"
)

// InteractiveExecutor executes workflows with interactive sessions
type InteractiveExecutor struct{}

// NewInteractiveExecutor creates a new interactive workflow executor
func NewInteractiveExecutor() *InteractiveExecutor {
	return &InteractiveExecutor{}
}

// Execute executes a workflow with interactive sessions
func (e *InteractiveExecutor) Execute(ctx context.Context, wf *workflow.Workflow, variables map[string]interface{}) error {
	return fmt.Errorf("interactive workflows are not yet supported on Windows")
}
