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
	"encoding/json"
	"errors"
	"fmt"
)

type ClaudeRunner struct {
	Command string
}

func (r *ClaudeRunner) binary() string {
	if r.Command != "" {
		return r.Command
	}
	return "claude"
}

func (r *ClaudeRunner) buildArgs(req Request) []string {
	args := []string{"-p", "--output-format", "json"}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}
	if req.SessionID != "" {
		args = append(args, "--session-id", req.SessionID)
	}
	if req.SystemPrompt != "" {
		args = append(args, "--append-system-prompt", req.SystemPrompt)
	}
	if mode := claudePermissionMode(req.PermissionMode); mode != "" {
		args = append(args, "--permission-mode", mode)
	}
	return append(args, req.ExtraArgs...)
}

func claudePermissionMode(mode string) string {
	switch mode {
	case "auto-edit":
		return "acceptEdits"
	case "yolo":
		return "bypassPermissions"
	case "plan":
		return "plan"
	default:
		return ""
	}
}

type claudeResult struct {
	Type       string `json:"type"`
	Subtype    string `json:"subtype"`
	IsError    bool   `json:"is_error"`
	DurationMS int64  `json:"duration_ms"`
	Result     string `json:"result"`
	SessionID  string `json:"session_id"`
}

func (r *ClaudeRunner) Run(ctx context.Context, req Request) (*Result, error) {
	out, err := runCLI(ctx, req, r.binary(), r.buildArgs(req), req.Prompt)
	if err != nil {
		return nil, err
	}
	return parseClaudeResult(out)
}

func parseClaudeResult(out []byte) (*Result, error) {
	var parsed claudeResult
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse claude JSON output: %w", err)
	}
	if parsed.IsError {
		msg := parsed.Result
		if msg == "" {
			msg = parsed.Subtype
		}
		return nil, errors.New("claude reported an error: " + msg)
	}
	return &Result{
		Output:     parsed.Result,
		SessionID:  parsed.SessionID,
		DurationMS: parsed.DurationMS,
		Raw:        json.RawMessage(out),
	}, nil
}
