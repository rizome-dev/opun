package promptgarden

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
	"github.com/rizome-dev/opun/pkg/core"
)

// Prompt represents a prompt in the garden
type Prompt struct {
	ID       string         `json:"id"`
	Name     string         `json:"name"`
	Content  string         `json:"content"`
	Metadata PromptMetadata `json:"metadata"`
}

// PromptMetadata contains metadata about a prompt
type PromptMetadata struct {
	Tags        []string `json:"tags,omitempty"`
	Category    string   `json:"category,omitempty"`
	Version     string   `json:"version,omitempty"`
	Description string   `json:"description,omitempty"`
	Author      string   `json:"author,omitempty"`
}

// ListPrompts is a helper method for Garden
func (g *Garden) ListPrompts() ([]*Prompt, error) {
	corePrompts, err := g.List()
	if err != nil {
		return nil, err
	}

	// Convert core.Prompt to our Prompt type
	prompts := make([]*Prompt, len(corePrompts))
	for i, cp := range corePrompts {
		metadata := cp.Metadata()
		prompts[i] = &Prompt{
			ID:      cp.ID(),
			Name:    cp.Name(),
			Content: cp.Content(),
			Metadata: PromptMetadata{
				Tags:        metadata.Tags,
				Category:    metadata.Category,
				Version:     metadata.Version,
				Description: metadata.Description,
				Author:      metadata.Author,
			},
		}
	}

	return prompts, nil
}

// SavePrompt saves a prompt to the garden (creates or updates)
func (g *Garden) SavePrompt(prompt *Prompt) error {
	// Convert to core prompt
	metadata := core.PromptMetadata{
		ID:          prompt.ID,
		Name:        prompt.Name,
		Description: prompt.Metadata.Description,
		Type:        core.PromptTypeTemplate,
		Category:    prompt.Metadata.Category,
		Tags:        prompt.Metadata.Tags,
		Author:      prompt.Metadata.Author,
		Version:     prompt.Metadata.Version,
	}

	corePrompt := NewTemplatePrompt(metadata, prompt.Content)

	// Try to update first if it exists
	if _, err := g.Get(prompt.ID); err == nil {
		// Prompt exists, update it
		g.mu.Lock()
		defer g.mu.Unlock()
		return g.store.Update(corePrompt)
	}

	// Otherwise create new
	return g.Add(corePrompt)
}

// GetPrompt retrieves a prompt by ID from the garden
func (g *Garden) GetPrompt(id string) (*Prompt, error) {
	corePrompt, err := g.Get(id)
	if err != nil {
		return nil, err
	}

	metadata := corePrompt.Metadata()
	return &Prompt{
		ID:      corePrompt.ID(),
		Name:    corePrompt.Name(),
		Content: corePrompt.Content(),
		Metadata: PromptMetadata{
			Tags:        metadata.Tags,
			Category:    metadata.Category,
			Version:     metadata.Version,
			Description: metadata.Description,
			Author:      metadata.Author,
		},
	}, nil
}

// DeletePrompt deletes a prompt from the garden
func (g *Garden) DeletePrompt(id string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.store.Delete(id)
}
