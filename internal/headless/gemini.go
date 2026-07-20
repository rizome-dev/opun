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

type GeminiRunner struct {
	Command string
}

func (r *GeminiRunner) binary() string {
	if r.Command != "" {
		return r.Command
	}
	return "gemini"
}

func (r *GeminiRunner) buildArgs(req Request) []string {
	args := []string{"--output-format", "json"}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}
	if mode := geminiApprovalMode(req.PermissionMode); mode != "" {
		args = append(args, "--approval-mode", mode)
	}
	return append(args, req.ExtraArgs...)
}

func geminiApprovalMode(mode string) string {
	switch mode {
	case "auto-edit":
		return "auto_edit"
	case "yolo":
		return "yolo"
	case "plan":
		return "plan"
	default:
		return ""
	}
}

type geminiResult struct {
	Response string `json:"response"`
	Error    *struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func (r *GeminiRunner) Run(ctx context.Context, req Request) (*Result, error) {
	prompt := req.Prompt
	if req.SystemPrompt != "" {
		prompt = req.SystemPrompt + "\n\n---\n\n" + prompt
	}
	out, err := runCLI(ctx, req, r.binary(), r.buildArgs(req), prompt)
	if err != nil {
		return nil, err
	}
	return parseGeminiResult(out)
}

func parseGeminiResult(out []byte) (*Result, error) {
	var parsed geminiResult
	if err := json.Unmarshal(out, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse gemini JSON output: %w", err)
	}
	if parsed.Error != nil {
		return nil, errors.New("gemini reported an error: " + parsed.Error.Message)
	}
	return &Result{
		Output: parsed.Response,
		Raw:    json.RawMessage(out),
	}, nil
}
