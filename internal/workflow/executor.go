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
	"github.com/rizome-dev/opun/pkg/workflow"
)

// Executor is the main workflow executor that delegates to InteractiveExecutor
type Executor struct {
	*InteractiveExecutor
}

// NewExecutor creates a new workflow executor
func NewExecutor() *Executor {
	return &Executor{
		InteractiveExecutor: NewInteractiveExecutor(),
	}
}

// Execute executes a workflow using interactive sessions
func (e *Executor) Execute(ctx context.Context, wf *workflow.Workflow, variables map[string]interface{}) error {
	return e.InteractiveExecutor.Execute(ctx, wf, variables)
}

// GetState returns the current execution state
func (e *Executor) GetState() *workflow.ExecutionState {
	return e.InteractiveExecutor.GetState()
}
