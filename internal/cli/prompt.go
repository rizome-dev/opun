package cli

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
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// promptModel is a simple text input model for prompts
type promptModel struct {
	textInput textinput.Model
	question  string
	err       error
}

func initialPromptModel(question string) promptModel {
	ti := textinput.New()
	ti.Placeholder = "Type your answer..."
	ti.Focus()
	ti.CharLimit = 1000
	ti.Width = 50

	return promptModel{
		textInput: ti,
		question:  question,
		err:       nil,
	}
}

func (m promptModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m promptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			return m, tea.Quit
		}

	case error:
		m.err = msg
		return m, nil
	}

	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m promptModel) View() string {
	// Style for the question
	questionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))

	// Style for the input area
	inputStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1)

	return fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		questionStyle.Render(m.question),
		inputStyle.Render(m.textInput.View()),
		"(Press Enter to submit, Esc to cancel)",
	)
}

// Prompt asks the user a question and returns their answer
func Prompt(question string) (string, error) {
	p := tea.NewProgram(initialPromptModel(question))
	m, err := p.Run()
	if err != nil {
		return "", err
	}

	if m, ok := m.(promptModel); ok {
		return strings.TrimSpace(m.textInput.Value()), nil
	}

	return "", fmt.Errorf("unexpected model type")
}

// confirmModel is a simple yes/no confirmation model
type confirmModel struct {
	question string
	answer   bool
}

func initialConfirmModel(question string) confirmModel {
	return confirmModel{
		question: question,
		answer:   false,
	}
}

func (m confirmModel) Init() tea.Cmd {
	return nil
}

func (m confirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			m.answer = true
			return m, tea.Quit
		case "n", "N":
			m.answer = false
			return m, tea.Quit
		case "ctrl+c", "esc":
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m confirmModel) View() string {
	// Style for the question
	questionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))

	// Style for the options
	optionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240"))

	return fmt.Sprintf(
		"%s\n\n%s",
		questionStyle.Render(m.question),
		optionStyle.Render("(y/n)"),
	)
}

// Confirm asks the user a yes/no question
func Confirm(question string) (bool, error) {
	p := tea.NewProgram(initialConfirmModel(question))
	m, err := p.Run()
	if err != nil {
		return false, err
	}

	if m, ok := m.(confirmModel); ok {
		return m.answer, nil
	}

	return false, fmt.Errorf("unexpected model type")
}

// fileItem represents a file or directory in the file selector
type fileItem struct {
	name   string
	path   string
	isDir  bool
	hidden bool
}

func (f fileItem) FilterValue() string { return f.name }
func (f fileItem) Title() string {
	// Show directory structure with visual hierarchy
	depth := strings.Count(f.name, string(filepath.Separator))
	if depth > 0 {
		// Add tree-like visual indication of depth
		parts := strings.Split(f.name, string(filepath.Separator))
		lastPart := parts[len(parts)-1]
		prefix := strings.Repeat("  ", depth-1) + "└─ "
		title := prefix + lastPart
		if f.isDir {
			title += "/"
		}
		return title
	}

	// Root level items
	title := f.name
	if f.isDir {
		title += "/"
	}
	return title
}
func (f fileItem) Description() string {
	if f.isDir {
		return "directory"
	}
	// Show file size or other metadata if available
	return ""
}

// filePromptModel is a file selection model with autocomplete and fuzzy find
type filePromptModel struct {
	textInput       textinput.Model
	question        string
	currentDir      string
	files           []fileItem
	filteredFiles   []fileItem
	selectedIndex   int
	showSuggestions bool
	err             error
}

func initialFilePromptModel(question string) filePromptModel {
	ti := textinput.New()
	ti.Placeholder = "Enter path, URL, or type to search files..."
	ti.Focus()
	ti.CharLimit = 1000
	ti.Width = 60

	// Start in current directory
	currentDir, _ := os.Getwd()

	m := filePromptModel{
		textInput:       ti,
		question:        question,
		currentDir:      currentDir,
		files:           []fileItem{},
		filteredFiles:   []fileItem{},
		selectedIndex:   0,
		showSuggestions: false,
		err:             nil,
	}

	m.loadFiles()
	return m
}

func (m *filePromptModel) loadFiles() {
	m.files = []fileItem{}

	// Recursively walk through all subdirectories
	maxDepth := 5 // Limit recursion depth to avoid performance issues
	err := filepath.WalkDir(m.currentDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip the current directory itself
		if path == m.currentDir {
			return nil
		}

		// Check depth to avoid going too deep
		relPath, _ := filepath.Rel(m.currentDir, path)
		depth := strings.Count(relPath, string(filepath.Separator))
		if depth > maxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden directories (and their contents)
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && d.Name() != "." {
			return filepath.SkipDir
		}

		// Skip common directories that shouldn't be searched
		if d.IsDir() {
			name := d.Name()
			if name == "node_modules" || name == ".git" || name == "vendor" ||
				name == ".next" || name == "dist" || name == "build" ||
				name == "__pycache__" || name == ".venv" || name == "venv" {
				return filepath.SkipDir
			}
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		// Create relative path for display
		displayPath := relPath
		if displayPath == "" {
			displayPath = d.Name()
		}

		hidden := strings.HasPrefix(d.Name(), ".")

		m.files = append(m.files, fileItem{
			name:   displayPath,
			path:   path,
			isDir:  info.IsDir(),
			hidden: hidden,
		})

		return nil
	})

	if err != nil {
		m.err = err
		return
	}

	// Sort files: directories first, then files, alphabetically
	sort.Slice(m.files, func(i, j int) bool {
		if m.files[i].isDir && !m.files[j].isDir {
			return true
		}
		if !m.files[i].isDir && m.files[j].isDir {
			return false
		}
		return m.files[i].name < m.files[j].name
	})

	m.filterFiles()
}

func (m *filePromptModel) filterFiles() {
	input := strings.ToLower(m.textInput.Value())

	// If input is empty, show non-hidden files in current directory only
	if input == "" {
		m.filteredFiles = []fileItem{}
		for _, f := range m.files {
			if !f.hidden && !strings.Contains(f.name, string(filepath.Separator)) {
				m.filteredFiles = append(m.filteredFiles, f)
			}
		}
		m.showSuggestions = len(m.filteredFiles) > 0
		return
	}

	// Filter files based on fuzzy matching
	m.filteredFiles = []fileItem{}

	// First pass: fuzzy match on the file/directory name (not full path)
	for _, f := range m.files {
		// Extract just the base name for fuzzy matching
		baseName := filepath.Base(f.name)
		if m.fuzzyMatch(strings.ToLower(baseName), input) {
			m.filteredFiles = append(m.filteredFiles, f)
		}
	}

	// Second pass: if we don't have many results, also match on full path
	if len(m.filteredFiles) < 5 {
		for _, f := range m.files {
			// Skip if already in results
			alreadyAdded := false
			for _, existing := range m.filteredFiles {
				if existing.path == f.path {
					alreadyAdded = true
					break
				}
			}
			if alreadyAdded {
				continue
			}

			// Match against the relative path
			if m.fuzzyMatch(strings.ToLower(f.name), input) || strings.Contains(strings.ToLower(f.path), input) {
				m.filteredFiles = append(m.filteredFiles, f)
			}
		}
	}

	// Sort results by relevance (shorter paths first, then alphabetically)
	sort.Slice(m.filteredFiles, func(i, j int) bool {
		// Directories first
		if m.filteredFiles[i].isDir && !m.filteredFiles[j].isDir {
			return true
		}
		if !m.filteredFiles[i].isDir && m.filteredFiles[j].isDir {
			return false
		}

		// Then by path depth (fewer separators = higher up)
		depthI := strings.Count(m.filteredFiles[i].name, string(filepath.Separator))
		depthJ := strings.Count(m.filteredFiles[j].name, string(filepath.Separator))
		if depthI != depthJ {
			return depthI < depthJ
		}

		// Finally alphabetically
		return m.filteredFiles[i].name < m.filteredFiles[j].name
	})

	m.showSuggestions = len(m.filteredFiles) > 0
	m.selectedIndex = 0
}

func (m *filePromptModel) fuzzyMatch(text, pattern string) bool {
	if pattern == "" {
		return true
	}

	patternIdx := 0
	for i := 0; i < len(text) && patternIdx < len(pattern); i++ {
		if text[i] == pattern[patternIdx] {
			patternIdx++
		}
	}

	return patternIdx == len(pattern)
}

func (m *filePromptModel) selectFile() {
	if m.selectedIndex >= 0 && m.selectedIndex < len(m.filteredFiles) {
		selected := m.filteredFiles[m.selectedIndex]

		if selected.isDir {
			// For directories in recursive mode, just set the path
			// Don't navigate since we're already showing everything recursively
			m.textInput.SetValue(selected.path + string(filepath.Separator))
			m.filterFiles() // Re-filter to show contents of this directory
		} else {
			// Select file
			m.textInput.SetValue(selected.path)
			m.showSuggestions = false
		}
	}
}

func (m filePromptModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m filePromptModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if m.showSuggestions && m.selectedIndex >= 0 && m.selectedIndex < len(m.filteredFiles) {
				selected := m.filteredFiles[m.selectedIndex]
				m.selectFile()
				// If a file was selected (not a directory), quit
				if !selected.isDir {
					return m, tea.Quit
				}
				return m, nil
			}
			return m, tea.Quit
		case tea.KeyUp:
			if m.showSuggestions && m.selectedIndex > 0 {
				m.selectedIndex--
			}
			return m, nil
		case tea.KeyDown:
			if m.showSuggestions && m.selectedIndex < len(m.filteredFiles)-1 {
				m.selectedIndex++
			}
			return m, nil
		case tea.KeyTab:
			if m.showSuggestions && len(m.filteredFiles) > 0 {
				// If no file is selected, select the first one
				if m.selectedIndex < 0 || m.selectedIndex >= len(m.filteredFiles) {
					m.selectedIndex = 0
				}
				selected := m.filteredFiles[m.selectedIndex]
				m.selectFile()
				// If a file was selected (not a directory), quit
				if !selected.isDir {
					return m, tea.Quit
				}
				return m, nil
			}
		}

	case error:
		m.err = msg
		return m, nil
	}

	// Update text input
	oldValue := m.textInput.Value()
	m.textInput, cmd = m.textInput.Update(msg)

	// If text changed, update filters
	if m.textInput.Value() != oldValue {
		m.filterFiles()
	}

	return m, cmd
}

func (m filePromptModel) View() string {
	// Style for the question
	questionStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("205"))

	// Style for the input area
	inputStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(0, 1)

	// Style for current directory
	dirStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true)

	// Style for suggestions
	suggestionStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1).
		MaxHeight(20) // Increased to show more suggestions

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("62")).
		Foreground(lipgloss.Color("230"))

	view := fmt.Sprintf(
		"%s\n\n%s\n%s\n\n%s",
		questionStyle.Render(m.question),
		dirStyle.Render(fmt.Sprintf("📂 Searching recursively from: %s", m.currentDir)),
		inputStyle.Render(m.textInput.View()),
		"(Enter absolute path/URL or type to search • Tab/Enter to select • ↑↓ to navigate • Esc to cancel)",
	)

	// Add suggestions if any
	if m.showSuggestions && len(m.filteredFiles) > 0 {
		var suggestions []string
		maxSuggestions := 20 // Show more suggestions for recursive search

		// Add a header showing the number of matches
		headerStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

		matchText := "match"
		if len(m.filteredFiles) > 1 {
			matchText = "matches"
		}
		suggestions = append(suggestions, headerStyle.Render(fmt.Sprintf("Found %d %s:", len(m.filteredFiles), matchText)))
		suggestions = append(suggestions, "") // Empty line for spacing

		for i, file := range m.filteredFiles {
			if i >= maxSuggestions {
				break
			}

			title := file.Title()
			if i == m.selectedIndex {
				suggestions = append(suggestions, selectedStyle.Render(title))
			} else {
				suggestions = append(suggestions, title)
			}
		}

		if len(m.filteredFiles) > maxSuggestions {
			suggestions = append(suggestions, fmt.Sprintf("... and %d more matches", len(m.filteredFiles)-maxSuggestions))
		}

		view += "\n\n" + suggestionStyle.Render(strings.Join(suggestions, "\n"))
	} else if m.textInput.Value() != "" && len(m.filteredFiles) == 0 {
		// Show "no matches" message
		noMatchStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("243")).
			Italic(true)
		view += "\n\n" + noMatchStyle.Render("No matching files found")
	}

	return view
}

// FilePrompt asks the user to select a file with autocomplete and fuzzy find
func FilePrompt(question string) (string, error) {
	p := tea.NewProgram(initialFilePromptModel(question))
	m, err := p.Run()
	if err != nil {
		return "", err
	}

	if m, ok := m.(filePromptModel); ok {
		value := strings.TrimSpace(m.textInput.Value())

		// Expand relative paths to absolute paths
		if value != "" && !filepath.IsAbs(value) {
			if strings.HasPrefix(value, "~/") {
				home, err := os.UserHomeDir()
				if err != nil {
					return "", err
				}
				value = filepath.Join(home, value[2:])
			} else {
				abs, err := filepath.Abs(value)
				if err != nil {
					return "", err
				}
				value = abs
			}
		}

		return value, nil
	}

	return "", fmt.Errorf("unexpected model type")
}
