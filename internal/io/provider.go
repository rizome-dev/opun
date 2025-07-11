package io

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
	"strings"
)

// Provider represents an AI provider that can be run transparently
type Provider interface {
	// Name returns the provider name
	Name() string

	// Command returns the command to execute
	Command() string

	// Args returns command arguments
	Args() []string

	// Env returns additional environment variables
	Env() []string
}

// ClaudeProvider implements Provider for Claude
type ClaudeProvider struct{}

func (p *ClaudeProvider) Name() string {
	return "claude"
}

func (p *ClaudeProvider) Command() string {
	return "claude"
}

func (p *ClaudeProvider) Args() []string {
	return []string{}
}

func (p *ClaudeProvider) Env() []string {
	return []string{}
}

// GeminiProvider implements Provider for Gemini
type GeminiProvider struct{}

func (p *GeminiProvider) Name() string {
	return "gemini"
}

func (p *GeminiProvider) Command() string {
	return "gemini"
}

func (p *GeminiProvider) Args() []string {
	return []string{}
}

func (p *GeminiProvider) Env() []string {
	return []string{}
}

// GetProvider returns a provider by name
func GetProvider(name string) (Provider, error) {
	switch strings.ToLower(name) {
	case "claude":
		return &ClaudeProvider{}, nil
	case "gemini":
		return &GeminiProvider{}, nil
	default:
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
}

// RunProviderSession runs a transparent session with the given provider
func RunProviderSession(ctx context.Context, provider Provider, config *TransparentSessionConfig) error {
	if config == nil {
		config = &TransparentSessionConfig{}
	}

	config.Provider = provider.Name()
	config.Command = provider.Command()
	config.Args = provider.Args()
	config.Env = provider.Env()

	session, err := NewTransparentSession(*config)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	return session.RunInteractive()
}
