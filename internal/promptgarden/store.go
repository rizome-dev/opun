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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/rizome-dev/opun/internal/utils"
	"github.com/rizome-dev/opun/pkg/core"
)

// FileStore implements PromptStore using the filesystem
type FileStore struct {
	basePath string
	index    map[string]*indexEntry
	mu       sync.RWMutex
}

// indexEntry stores metadata for quick lookups
type indexEntry struct {
	ID       string
	Name     string
	Category string
	Tags     []string
	FilePath string
}

// NewFileStore creates a new file-based prompt store
func NewFileStore(basePath string) *FileStore {
	store := &FileStore{
		basePath: basePath,
		index:    make(map[string]*indexEntry),
	}

	// Load index
	store.loadIndex()

	return store
}

// Create creates a new prompt
func (s *FileStore) Create(prompt core.Prompt) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Generate file path
	fileName := fmt.Sprintf("%s.json", prompt.ID())
	filePath := filepath.Join(s.basePath, fileName)

	// Check if already exists
	if _, err := os.Stat(filePath); err == nil {
		return fmt.Errorf("prompt with ID %s already exists", prompt.ID())
	}

	// Save to file
	if err := s.savePrompt(prompt, filePath); err != nil {
		return err
	}

	// Update index
	s.updateIndex(prompt, filePath)

	return s.saveIndex()
}

// Get retrieves a prompt by ID
func (s *FileStore) Get(id string) (core.Prompt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.index[id]
	if !ok {
		return nil, fmt.Errorf("prompt not found: %s", id)
	}

	return s.loadPrompt(entry.FilePath)
}

// GetByName retrieves a prompt by name
func (s *FileStore) GetByName(name string) (core.Prompt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, entry := range s.index {
		if entry.Name == name {
			return s.loadPrompt(entry.FilePath)
		}
	}

	return nil, fmt.Errorf("prompt not found: %s", name)
}

// Update updates an existing prompt
func (s *FileStore) Update(prompt core.Prompt) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.index[prompt.ID()]
	if !ok {
		return fmt.Errorf("prompt not found: %s", prompt.ID())
	}

	// Save to file
	if err := s.savePrompt(prompt, entry.FilePath); err != nil {
		return err
	}

	// Update index
	s.updateIndex(prompt, entry.FilePath)

	return s.saveIndex()
}

// Delete deletes a prompt
func (s *FileStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.index[id]
	if !ok {
		return fmt.Errorf("prompt not found: %s", id)
	}

	// Delete file
	if err := os.Remove(entry.FilePath); err != nil {
		return err
	}

	// Remove from index
	delete(s.index, id)

	return s.saveIndex()
}

// List returns all prompts
func (s *FileStore) List() ([]core.Prompt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prompts := []core.Prompt{}
	for _, entry := range s.index {
		prompt, err := s.loadPrompt(entry.FilePath)
		if err != nil {
			continue // Skip failed loads
		}
		prompts = append(prompts, prompt)
	}

	return prompts, nil
}

// ListByType returns prompts of a specific type
func (s *FileStore) ListByType(promptType core.PromptType) ([]core.Prompt, error) {
	// For now, load all and filter
	// Could optimize with type in index
	all, err := s.List()
	if err != nil {
		return nil, err
	}

	filtered := []core.Prompt{}
	for _, prompt := range all {
		if prompt.Metadata().Type == promptType {
			filtered = append(filtered, prompt)
		}
	}

	return filtered, nil
}

// ListByCategory returns prompts in a category
func (s *FileStore) ListByCategory(category string) ([]core.Prompt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prompts := []core.Prompt{}
	for _, entry := range s.index {
		if entry.Category == category {
			prompt, err := s.loadPrompt(entry.FilePath)
			if err != nil {
				continue
			}
			prompts = append(prompts, prompt)
		}
	}

	return prompts, nil
}

// ListByTags returns prompts with matching tags
func (s *FileStore) ListByTags(tags []string) ([]core.Prompt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	prompts := []core.Prompt{}
	tagSet := make(map[string]bool)
	for _, tag := range tags {
		tagSet[tag] = true
	}

	for _, entry := range s.index {
		// Check if any tag matches
		hasMatch := false
		for _, tag := range entry.Tags {
			if tagSet[tag] {
				hasMatch = true
				break
			}
		}

		if hasMatch {
			prompt, err := s.loadPrompt(entry.FilePath)
			if err != nil {
				continue
			}
			prompts = append(prompts, prompt)
		}
	}

	return prompts, nil
}

// Search searches for prompts
func (s *FileStore) Search(query string) ([]core.Prompt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query = strings.ToLower(query)
	prompts := []core.Prompt{}

	for _, entry := range s.index {
		// Search in name, category, and tags
		if strings.Contains(strings.ToLower(entry.Name), query) ||
			strings.Contains(strings.ToLower(entry.Category), query) {
			prompt, err := s.loadPrompt(entry.FilePath)
			if err != nil {
				continue
			}
			prompts = append(prompts, prompt)
			continue
		}

		// Check tags
		for _, tag := range entry.Tags {
			if strings.Contains(strings.ToLower(tag), query) {
				prompt, err := s.loadPrompt(entry.FilePath)
				if err != nil {
					continue
				}
				prompts = append(prompts, prompt)
				break
			}
		}
	}

	return prompts, nil
}

// Export exports a prompt to JSON
func (s *FileStore) Export(id string) ([]byte, error) {
	prompt, err := s.Get(id)
	if err != nil {
		return nil, err
	}

	// Create export structure
	export := map[string]interface{}{
		"metadata": prompt.Metadata(),
		"content":  prompt.Content(),
	}

	return json.MarshalIndent(export, "", "  ")
}

// Import imports a prompt from JSON
func (s *FileStore) Import(data []byte) (core.Prompt, error) {
	var export map[string]interface{}
	if err := json.Unmarshal(data, &export); err != nil {
		return nil, err
	}

	// Extract metadata
	metadataData, err := json.Marshal(export["metadata"])
	if err != nil {
		return nil, err
	}

	var metadata core.PromptMetadata
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		return nil, err
	}

	// Extract content
	content, ok := export["content"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid content in import data")
	}

	// Create prompt
	prompt := NewTemplatePrompt(metadata, content)

	// Store it
	if err := s.Create(prompt); err != nil {
		return nil, err
	}

	return prompt, nil
}

// Helper methods

func (s *FileStore) loadIndex() error {
	indexPath := filepath.Join(s.basePath, "index.json")

	data, err := os.ReadFile(indexPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No index yet
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &s.index)
}

func (s *FileStore) saveIndex() error {
	indexPath := filepath.Join(s.basePath, "index.json")

	data, err := json.MarshalIndent(s.index, "", "  ")
	if err != nil {
		return err
	}

	return utils.WriteFile(indexPath, data)
}

func (s *FileStore) updateIndex(prompt core.Prompt, filePath string) {
	metadata := prompt.Metadata()
	s.index[prompt.ID()] = &indexEntry{
		ID:       prompt.ID(),
		Name:     prompt.Name(),
		Category: metadata.Category,
		Tags:     metadata.Tags,
		FilePath: filePath,
	}
}

func (s *FileStore) savePrompt(prompt core.Prompt, filePath string) error {
	// Prepare data
	data := map[string]interface{}{
		"metadata": prompt.Metadata(),
		"content":  prompt.Content(),
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// Use utils.WriteFile which handles directory creation and permissions
	return utils.WriteFile(filePath, jsonData)
}

func (s *FileStore) loadPrompt(filePath string) (core.Prompt, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var promptData map[string]interface{}
	if err := json.Unmarshal(data, &promptData); err != nil {
		return nil, err
	}

	// Extract metadata
	metadataData, err := json.Marshal(promptData["metadata"])
	if err != nil {
		return nil, err
	}

	var metadata core.PromptMetadata
	if err := json.Unmarshal(metadataData, &metadata); err != nil {
		return nil, err
	}

	// Extract content
	content, ok := promptData["content"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid content in prompt file")
	}

	return NewTemplatePrompt(metadata, content), nil
}
