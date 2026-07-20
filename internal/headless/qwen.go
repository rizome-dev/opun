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
	"strings"
)

type QwenRunner struct {
	Command string
}

func (r *QwenRunner) binary() string {
	if r.Command != "" {
		return r.Command
	}
	return "qwen"
}

func (r *QwenRunner) buildArgs(req Request) []string {
	args := []string{"--output-format", "json"}
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}
	if req.SystemPrompt != "" {
		args = append(args, "--system-prompt", req.SystemPrompt)
	}
	if mode := qwenApprovalMode(req.PermissionMode); mode != "" {
		args = append(args, "--approval-mode", mode)
	}
	return append(args, req.ExtraArgs...)
}

func qwenApprovalMode(mode string) string {
	switch mode {
	case "auto-edit":
		return "auto-edit"
	case "yolo":
		return "yolo"
	case "plan":
		return "plan"
	default:
		return ""
	}
}

type qwenMessage struct {
	Type       string `json:"type"`
	Subtype    string `json:"subtype"`
	IsError    bool   `json:"is_error"`
	DurationMS int64  `json:"duration_ms"`
	Result     string `json:"result"`
	SessionID  string `json:"session_id"`
	Message    *struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"message"`
}

func (r *QwenRunner) Run(ctx context.Context, req Request) (*Result, error) {
	out, err := runCLI(ctx, req, r.binary(), r.buildArgs(req), req.Prompt)
	if err != nil {
		return nil, err
	}
	return parseQwenResult(out)
}

func parseQwenResult(out []byte) (*Result, error) {
	var messages []qwenMessage
	if err := json.Unmarshal(out, &messages); err != nil {
		return nil, fmt.Errorf("failed to parse qwen JSON output: %w", err)
	}

	for i := len(messages) - 1; i >= 0; i-- {
		m := messages[i]
		if m.Type != "result" {
			continue
		}
		if m.IsError {
			msg := m.Result
			if msg == "" {
				msg = m.Subtype
			}
			return nil, errors.New("qwen reported an error: " + msg)
		}
		return &Result{
			Output:     m.Result,
			SessionID:  m.SessionID,
			DurationMS: m.DurationMS,
			Raw:        json.RawMessage(out),
		}, nil
	}

	var texts []string
	for _, m := range messages {
		if m.Type == "assistant" && m.Message != nil {
			for _, c := range m.Message.Content {
				if c.Type == "text" && c.Text != "" {
					texts = append(texts, c.Text)
				}
			}
		}
	}
	if len(texts) > 0 {
		return &Result{
			Output: strings.Join(texts, "\n"),
			Raw:    json.RawMessage(out),
		}, nil
	}

	return nil, errors.New("qwen output contained no result message")
}
