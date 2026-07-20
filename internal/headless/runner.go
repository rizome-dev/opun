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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const DefaultTimeout = 10 * time.Minute

type Request struct {
	Prompt         string
	SystemPrompt   string
	Model          string
	WorkDir        string
	SessionID      string
	PermissionMode string
	Timeout        time.Duration
	Env            []string
	ExtraArgs      []string
}

type Result struct {
	Output     string
	SessionID  string
	DurationMS int64
	Raw        json.RawMessage
}

type Runner interface {
	Run(ctx context.Context, req Request) (*Result, error)
}

func NewRunner(provider string) (Runner, error) {
	switch provider {
	case "claude":
		return &ClaudeRunner{}, nil
	case "gemini":
		return &GeminiRunner{}, nil
	case "qwen":
		return &QwenRunner{}, nil
	case "mock":
		return &MockRunner{}, nil
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

func runCLI(ctx context.Context, req Request, bin string, args []string, stdin string) ([]byte, error) {
	timeout := req.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if _, err := exec.LookPath(bin); err != nil {
		return nil, fmt.Errorf("%s not found in PATH: %w", bin, err)
	}

	// #nosec G204 -- bin and args are constructed from the provider registry, not user shell input
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdin = strings.NewReader(stdin)
	if req.WorkDir != "" {
		cmd.Dir = req.WorkDir
	}
	cmd.Env = append(os.Environ(), req.Env...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("%s timed out after %s", bin, timeout)
		}
		return nil, fmt.Errorf("%s failed: %w\nstderr: %s", bin, err, truncateOutput(stderr.String()))
	}

	return stdout.Bytes(), nil
}

func truncateOutput(s string) string {
	const max = 2000
	s = strings.TrimSpace(s)
	if len(s) > max {
		return s[:max] + "... (truncated)"
	}
	return s
}
