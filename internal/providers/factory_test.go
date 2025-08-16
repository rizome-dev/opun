package providers

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
	"testing"

	"github.com/rizome-dev/opun/pkg/core"
	"github.com/stretchr/testify/assert"
)

func TestProviderFactory_CreateProvider(t *testing.T) {
	factory := NewProviderFactory()

	tests := []struct {
		name        string
		config      core.ProviderConfig
		wantType    core.ProviderType
		shouldError bool
	}{
		{
			name: "Create Claude provider",
			config: core.ProviderConfig{
				Name:    "test-claude",
				Type:    core.ProviderTypeClaude,
				Command: "claude",
			},
			wantType:    core.ProviderTypeClaude,
			shouldError: true, // Will error because claude command doesn't exist in test
		},
		{
			name: "Create Gemini provider",
			config: core.ProviderConfig{
				Name:    "test-gemini",
				Type:    core.ProviderTypeGemini,
				Command: "gemini",
			},
			wantType:    core.ProviderTypeGemini,
			shouldError: true, // Will error because gemini command doesn't exist in test
		},
		{
			name: "Create Qwen provider",
			config: core.ProviderConfig{
				Name:    "test-qwen",
				Type:    core.ProviderTypeQwen,
				Command: "qwen",
			},
			wantType:    core.ProviderTypeQwen,
			shouldError: true, // Will error because qwen command doesn't exist in test
		},
		{
			name: "Create Mock provider",
			config: core.ProviderConfig{
				Name:    "test-mock",
				Type:    core.ProviderTypeMock,
				Command: "mock",
			},
			wantType:    core.ProviderTypeMock,
			shouldError: false, // Mock provider doesn't validate command existence
		},
		{
			name: "Unsupported provider type",
			config: core.ProviderConfig{
				Name:    "test-unknown",
				Type:    "unknown",
				Command: "unknown",
			},
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := factory.CreateProvider(tt.config)

			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
				assert.Equal(t, tt.wantType, provider.Type())
			}
		})
	}
}

func TestProviderFactory_CreateProviderFromType(t *testing.T) {
	factory := NewProviderFactory()

	tests := []struct {
		name         string
		providerType string
		providerName string
		wantCommand  string
		wantModel    string
		shouldError  bool
	}{
		{
			name:         "Create Claude from type",
			providerType: "claude",
			providerName: "test-claude",
			wantCommand:  "claude",
			wantModel:    "sonnet",
			shouldError:  true, // Will error due to validation
		},
		{
			name:         "Create Gemini from type",
			providerType: "gemini",
			providerName: "test-gemini",
			wantCommand:  "gemini",
			wantModel:    "gemini-pro",
			shouldError:  true, // Will error due to validation
		},
		{
			name:         "Create Qwen from type",
			providerType: "qwen",
			providerName: "test-qwen",
			wantCommand:  "qwen",
			wantModel:    "code",
			shouldError:  true, // Will error due to validation
		},
		{
			name:         "Create Mock from type",
			providerType: "mock",
			providerName: "test-mock",
			shouldError:  false, // Mock doesn't validate
		},
		{
			name:         "Unsupported provider type",
			providerType: "unknown",
			providerName: "test-unknown",
			shouldError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := factory.CreateProviderFromType(tt.providerType, tt.providerName)

			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, provider)
				assert.Equal(t, tt.providerName, provider.Name())
			}
		})
	}
}

func TestGetDefaultModel(t *testing.T) {
	tests := []struct {
		providerType string
		wantModel    string
	}{
		{"claude", "sonnet"},
		{"gemini", "gemini-pro"},
		{"qwen", "code"},
		{"unknown", ""},
	}

	for _, tt := range tests {
		t.Run(tt.providerType, func(t *testing.T) {
			model := getDefaultModel(tt.providerType)
			assert.Equal(t, tt.wantModel, model)
		})
	}
}

func TestGetDefaultFeatures(t *testing.T) {
	tests := []struct {
		providerType     string
		wantInteractive  bool
		wantMCP          bool
		wantQualityModes bool
	}{
		{"claude", true, true, true},
		{"gemini", true, true, false},
		{"qwen", true, true, false},
		{"unknown", false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.providerType, func(t *testing.T) {
			features := getDefaultFeatures(tt.providerType)
			assert.Equal(t, tt.wantInteractive, features.Interactive)
			assert.Equal(t, tt.wantMCP, features.MCP)
			assert.Equal(t, tt.wantQualityModes, features.QualityModes)
		})
	}
}

