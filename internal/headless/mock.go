package headless

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
)

type MockRunner struct {
	Response string
	Err      error
	LastReq  *Request
}

func (r *MockRunner) Run(ctx context.Context, req Request) (*Result, error) {
	r.LastReq = &req
	if r.Err != nil {
		return nil, r.Err
	}
	output := r.Response
	if output == "" {
		output = fmt.Sprintf("mock response for: %s", req.Prompt)
	}
	return &Result{Output: output}, nil
}
