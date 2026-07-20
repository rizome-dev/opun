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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRunner(t *testing.T) {
	for _, provider := range []string{"claude", "gemini", "qwen", "mock"} {
		runner, err := NewRunner(provider)
		require.NoError(t, err, provider)
		require.NotNil(t, runner, provider)
	}

	_, err := NewRunner("unknown")
	assert.Error(t, err)
}

func TestClaudeBuildArgs(t *testing.T) {
	r := &ClaudeRunner{}

	assert.Equal(t,
		[]string{"-p", "--output-format", "json"},
		r.buildArgs(Request{Prompt: "hi"}))

	assert.Equal(t,
		[]string{
			"-p", "--output-format", "json",
			"--model", "sonnet",
			"--session-id", "abc-123",
			"--append-system-prompt", "you are a reviewer",
			"--permission-mode", "acceptEdits",
			"--max-turns", "5",
		},
		r.buildArgs(Request{
			Prompt:         "hi",
			Model:          "sonnet",
			SessionID:      "abc-123",
			SystemPrompt:   "you are a reviewer",
			PermissionMode: "auto-edit",
			ExtraArgs:      []string{"--max-turns", "5"},
		}))

	assert.Contains(t, r.buildArgs(Request{PermissionMode: "yolo"}), "bypassPermissions")
}

func TestGeminiBuildArgs(t *testing.T) {
	r := &GeminiRunner{}

	assert.Equal(t,
		[]string{"--output-format", "json"},
		r.buildArgs(Request{Prompt: "hi"}))

	assert.Equal(t,
		[]string{
			"--output-format", "json",
			"--model", "auto",
			"--approval-mode", "auto_edit",
		},
		r.buildArgs(Request{
			Prompt:         "hi",
			Model:          "auto",
			PermissionMode: "auto-edit",
		}))
}

func TestQwenBuildArgs(t *testing.T) {
	r := &QwenRunner{}

	assert.Equal(t,
		[]string{"--output-format", "json"},
		r.buildArgs(Request{Prompt: "hi"}))

	assert.Equal(t,
		[]string{
			"--output-format", "json",
			"--model", "qwen3-coder-plus",
			"--system-prompt", "you are a coder",
			"--approval-mode", "auto-edit",
		},
		r.buildArgs(Request{
			Prompt:         "hi",
			Model:          "qwen3-coder-plus",
			SystemPrompt:   "you are a coder",
			PermissionMode: "auto-edit",
		}))
}

func TestParseClaudeResult(t *testing.T) {
	fixture := []byte(`{
		"type": "result",
		"subtype": "success",
		"is_error": false,
		"duration_ms": 4207,
		"result": "The answer is 42.",
		"session_id": "3f9c1de2-0001-4b2a-9c1d-e20001aaaa01",
		"total_cost_usd": 0.0034
	}`)

	res, err := parseClaudeResult(fixture)
	require.NoError(t, err)
	assert.Equal(t, "The answer is 42.", res.Output)
	assert.Equal(t, "3f9c1de2-0001-4b2a-9c1d-e20001aaaa01", res.SessionID)
	assert.Equal(t, int64(4207), res.DurationMS)
	assert.NotEmpty(t, res.Raw)

	errFixture := []byte(`{"type":"result","subtype":"error_max_turns","is_error":true,"result":"hit turn limit"}`)
	_, err = parseClaudeResult(errFixture)
	assert.ErrorContains(t, err, "hit turn limit")

	_, err = parseClaudeResult([]byte("not json"))
	assert.Error(t, err)
}

func TestParseGeminiResult(t *testing.T) {
	fixture := []byte(`{
		"response": "Here is the analysis you requested.",
		"stats": {"models": {"gemini-2.5-pro": {"tokens": {"prompt": 10, "candidates": 20}}}}
	}`)

	res, err := parseGeminiResult(fixture)
	require.NoError(t, err)
	assert.Equal(t, "Here is the analysis you requested.", res.Output)

	errFixture := []byte(`{"response": "", "error": {"type": "ApiError", "message": "quota exceeded"}}`)
	_, err = parseGeminiResult(errFixture)
	assert.ErrorContains(t, err, "quota exceeded")

	_, err = parseGeminiResult([]byte("[]"))
	assert.Error(t, err)
}

func TestParseQwenResult(t *testing.T) {
	fixture := []byte(`[
		{"type": "system", "subtype": "session_start", "session_id": "q-sess-1"},
		{"type": "assistant", "message": {"content": [{"type": "text", "text": "Working on it."}]}},
		{"type": "result", "subtype": "success", "is_error": false, "duration_ms": 1500, "result": "Refactor complete.", "session_id": "q-sess-1"}
	]`)

	res, err := parseQwenResult(fixture)
	require.NoError(t, err)
	assert.Equal(t, "Refactor complete.", res.Output)
	assert.Equal(t, "q-sess-1", res.SessionID)

	noResult := []byte(`[
		{"type": "system", "subtype": "session_start"},
		{"type": "assistant", "message": {"content": [{"type": "text", "text": "partial answer"}]}}
	]`)
	res, err = parseQwenResult(noResult)
	require.NoError(t, err)
	assert.Equal(t, "partial answer", res.Output)

	errFixture := []byte(`[{"type": "result", "subtype": "error", "is_error": true, "result": "budget exceeded"}]`)
	_, err = parseQwenResult(errFixture)
	assert.ErrorContains(t, err, "budget exceeded")

	_, err = parseQwenResult([]byte(`{"response": "wrong shape"}`))
	assert.Error(t, err)

	_, err = parseQwenResult([]byte(`[]`))
	assert.Error(t, err)
}

func TestMockRunner(t *testing.T) {
	r := &MockRunner{Response: "canned"}
	res, err := r.Run(context.Background(), Request{Prompt: "anything"})
	require.NoError(t, err)
	assert.Equal(t, "canned", res.Output)
	require.NotNil(t, r.LastReq)
	assert.Equal(t, "anything", r.LastReq.Prompt)
}
