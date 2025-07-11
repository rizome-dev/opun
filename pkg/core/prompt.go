package core

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
	"fmt"
	"time"
)

// PromptType represents the type of prompt
type PromptType string

const (
	PromptTypeSystem    PromptType = "system"
	PromptTypeUser      PromptType = "user"
	PromptTypeAssistant PromptType = "assistant"
	PromptTypeTemplate  PromptType = "template"
	PromptTypeWorkflow  PromptType = "workflow"
)

// PromptMetadata contains metadata about a prompt
type PromptMetadata struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        PromptType             `json:"type"`
	Tags        []string               `json:"tags"`
	Category    string                 `json:"category"`
	Version     string                 `json:"version"`
	Author      string                 `json:"author"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Variables   []PromptVariable       `json:"variables"`
	Includes    []string               `json:"includes"`
	Extra       map[string]interface{} `json:"extra"`
}

// PromptVariable defines a variable in a prompt template
type PromptVariable struct {
	Name         string      `json:"name"`
	Description  string      `json:"description"`
	Type         string      `json:"type"` // string, number, boolean, file, prompt
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"default_value"`
	Validation   string      `json:"validation"` // regex or validation rule
}

// Prompt defines the interface for prompts
type Prompt interface {
	// Basic information
	ID() string
	Name() string
	Content() string

	// Metadata
	Metadata() PromptMetadata

	// Template operations
	Template(vars map[string]interface{}) (string, error)
	Variables() []PromptVariable
	Validate(vars map[string]interface{}) error

	// Includes
	Includes() []string
	SetIncludeResolver(resolver IncludeResolver)
}

// IncludeResolver resolves included prompts
type IncludeResolver interface {
	Resolve(promptID string) (Prompt, error)
}

// PromptStore defines the interface for prompt storage
type PromptStore interface {
	// CRUD operations
	Create(prompt Prompt) error
	Get(id string) (Prompt, error)
	GetByName(name string) (Prompt, error)
	Update(prompt Prompt) error
	Delete(id string) error

	// Query operations
	List() ([]Prompt, error)
	ListByType(promptType PromptType) ([]Prompt, error)
	ListByCategory(category string) ([]Prompt, error)
	ListByTags(tags []string) ([]Prompt, error)
	Search(query string) ([]Prompt, error)

	// Import/Export
	Export(id string) ([]byte, error)
	Import(data []byte) (Prompt, error)
}

// BasePrompt provides common prompt functionality
type BasePrompt struct {
	metadata        PromptMetadata
	content         string
	includeResolver IncludeResolver
}

// NewBasePrompt creates a new base prompt
func NewBasePrompt(metadata PromptMetadata, content string) *BasePrompt {
	if metadata.ID == "" {
		metadata.ID = GenerateID()
	}
	if metadata.CreatedAt.IsZero() {
		metadata.CreatedAt = time.Now()
	}
	metadata.UpdatedAt = time.Now()

	return &BasePrompt{
		metadata: metadata,
		content:  content,
	}
}

// ID returns the prompt ID
func (p *BasePrompt) ID() string {
	return p.metadata.ID
}

// Name returns the prompt name
func (p *BasePrompt) Name() string {
	return p.metadata.Name
}

// Content returns the prompt content
func (p *BasePrompt) Content() string {
	return p.content
}

// Metadata returns the prompt metadata
func (p *BasePrompt) Metadata() PromptMetadata {
	return p.metadata
}

// Variables returns the prompt variables
func (p *BasePrompt) Variables() []PromptVariable {
	return p.metadata.Variables
}

// Includes returns the included prompt IDs
func (p *BasePrompt) Includes() []string {
	return p.metadata.Includes
}

// SetIncludeResolver sets the include resolver
func (p *BasePrompt) SetIncludeResolver(resolver IncludeResolver) {
	p.includeResolver = resolver
}

// GenerateID generates a unique prompt ID
func GenerateID() string {
	// Simple implementation - in production use UUID
	return fmt.Sprintf("prompt-%d", time.Now().UnixNano())
}
