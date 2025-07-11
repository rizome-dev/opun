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
	"bytes"
	"fmt"
	"os"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/rizome-dev/opun/pkg/core"
)

// TemplateEngine handles prompt templating
type TemplateEngine struct {
	includeResolver core.IncludeResolver
	funcMap         template.FuncMap
}

// NewTemplateEngine creates a new template engine
func NewTemplateEngine() *TemplateEngine {
	engine := &TemplateEngine{
		funcMap: make(template.FuncMap),
	}

	// Register default template functions
	engine.registerDefaultFuncs()

	return engine
}

// SetIncludeResolver sets the include resolver
func (e *TemplateEngine) SetIncludeResolver(resolver core.IncludeResolver) {
	e.includeResolver = resolver
}

// Execute executes a template with variables
func (e *TemplateEngine) Execute(templateContent string, vars map[string]interface{}) (string, error) {
	// First, process includes
	processed, err := e.processIncludes(templateContent, vars)
	if err != nil {
		return "", fmt.Errorf("failed to process includes: %w", err)
	}

	// Then, process variables
	result, err := e.processVariables(processed, vars)
	if err != nil {
		return "", fmt.Errorf("failed to process variables: %w", err)
	}

	return result, nil
}

// processIncludes processes {{include:prompt-name}} directives
func (e *TemplateEngine) processIncludes(content string, vars map[string]interface{}) (string, error) {
	if e.includeResolver == nil {
		return content, nil
	}

	// Regex to match {{include:prompt-name}} or {{promptgarden://prompt-name}}
	includeRegex := regexp.MustCompile(`\{\{(?:include:|promptgarden://)([^}]+)\}\}`)

	// Track included prompts to prevent circular dependencies
	included := make(map[string]bool)

	// Process includes recursively
	var processContent func(string) (string, error)
	processContent = func(text string) (string, error) {
		result := includeRegex.ReplaceAllStringFunc(text, func(match string) string {
			// Extract prompt name
			promptName := includeRegex.FindStringSubmatch(match)[1]
			promptName = strings.TrimSpace(promptName)

			// Check for circular dependency
			if included[promptName] {
				return fmt.Sprintf("[ERROR: Circular dependency detected for prompt '%s']", promptName)
			}
			included[promptName] = true

			// Resolve prompt
			prompt, err := e.includeResolver.Resolve(promptName)
			if err != nil {
				return fmt.Sprintf("[ERROR: Failed to resolve prompt '%s': %v]", promptName, err)
			}

			// Get prompt content
			includedContent := prompt.Content()

			// Process nested includes
			processed, err := processContent(includedContent)
			if err != nil {
				return fmt.Sprintf("[ERROR: Failed to process nested includes: %v]", err)
			}

			// Remove from included set (allows the same prompt in different branches)
			delete(included, promptName)

			return processed
		})
		return result, nil
	}

	return processContent(content)
}

// processVariables processes template variables
func (e *TemplateEngine) processVariables(content string, vars map[string]interface{}) (string, error) {
	// First, handle simple {{variable}} syntax
	content = e.processSimpleVariables(content, vars)

	// Then, handle complex template syntax using Go templates
	tmpl, err := template.New("prompt").Funcs(e.funcMap).Parse(content)
	if err != nil {
		// If template parsing fails, return content with simple substitution only
		return content, nil
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		// If template execution fails, return content with simple substitution only
		return content, nil
	}

	return buf.String(), nil
}

// processSimpleVariables handles simple {{variable}} substitution
func (e *TemplateEngine) processSimpleVariables(content string, vars map[string]interface{}) string {
	// Handle {{variable}} syntax - must handle spaces in file paths
	varRegex := regexp.MustCompile(`\{\{([^}]+)\}\}`)

	return varRegex.ReplaceAllStringFunc(content, func(match string) string {
		// Skip template directives but not file: directives
		if (strings.Contains(match, "#") ||
			strings.Contains(match, "include:") ||
			strings.Contains(match, "promptgarden://")) &&
			!strings.Contains(match, "file:") {
			return match
		}

		// Extract variable name
		varName := strings.TrimSpace(varRegex.FindStringSubmatch(match)[1])

		// Handle file references {{file:path/to/file.txt}}
		if strings.HasPrefix(varName, "file:") {
			// Extract file path after "file:"
			filePath := strings.TrimSpace(varName[5:])

			// #nosec G304 -- include path is from template under user control
			content, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Sprintf("[ERROR: Failed to read file '%s': %v]", filePath, err)
			}
			return string(content)
		}

		// Look up variable value
		if value, ok := vars[varName]; ok {
			return fmt.Sprintf("%v", value)
		}

		// Return original if not found
		return match
	})
}

// registerDefaultFuncs registers default template functions
func (e *TemplateEngine) registerDefaultFuncs() {
	e.funcMap["upper"] = strings.ToUpper
	e.funcMap["lower"] = strings.ToLower
	e.funcMap["title"] = strings.Title
	e.funcMap["trim"] = strings.TrimSpace
	e.funcMap["replace"] = strings.ReplaceAll
	e.funcMap["contains"] = strings.Contains
	e.funcMap["hasPrefix"] = strings.HasPrefix
	e.funcMap["hasSuffix"] = strings.HasSuffix
	e.funcMap["join"] = strings.Join
	e.funcMap["split"] = strings.Split

	// Date/time functions
	e.funcMap["now"] = func() string {
		return time.Now().Format(time.RFC3339)
	}
	e.funcMap["date"] = func(format string) string {
		return time.Now().Format(format)
	}

	// File functions
	e.funcMap["readFile"] = func(path string) (string, error) {
		// #nosec G304 -- template function for user-specified files
		content, err := os.ReadFile(path)
		if err != nil {
			return "", err
		}
		return string(content), nil
	}

	// Include function for template syntax
	e.funcMap["include"] = func(promptName string) (string, error) {
		if e.includeResolver == nil {
			return "", fmt.Errorf("include resolver not set")
		}

		prompt, err := e.includeResolver.Resolve(promptName)
		if err != nil {
			return "", err
		}

		return prompt.Content(), nil
	}
}

// TemplatePrompt implements the Prompt interface with template support
type TemplatePrompt struct {
	*core.BasePrompt
	engine *TemplateEngine
}

// NewTemplatePrompt creates a new template prompt
func NewTemplatePrompt(metadata core.PromptMetadata, content string) *TemplatePrompt {
	// Extract variables from content if not already defined
	if len(metadata.Variables) == 0 {
		metadata.Variables = extractVariables(content)
	}

	return &TemplatePrompt{
		BasePrompt: core.NewBasePrompt(metadata, content),
		engine:     NewTemplateEngine(),
	}
}

// Template executes the prompt template with variables
func (p *TemplatePrompt) Template(vars map[string]interface{}) (string, error) {
	// Initialize vars if nil
	if vars == nil {
		vars = make(map[string]interface{})
	}

	// Apply defaults first
	vars = p.applyDefaults(vars)

	// Validate variables after defaults are applied
	if err := p.Validate(vars); err != nil {
		return "", err
	}

	// Execute template
	return p.engine.Execute(p.Content(), vars)
}

// Validate validates the provided variables
func (p *TemplatePrompt) Validate(vars map[string]interface{}) error {
	for _, variable := range p.Variables() {
		if variable.Required {
			if _, ok := vars[variable.Name]; !ok {
				return fmt.Errorf("required variable '%s' not provided", variable.Name)
			}
		}

		// TODO: Add type and regex validation
	}

	return nil
}

// SetIncludeResolver sets the include resolver
func (p *TemplatePrompt) SetIncludeResolver(resolver core.IncludeResolver) {
	p.BasePrompt.SetIncludeResolver(resolver)
	p.engine.SetIncludeResolver(resolver)
}

// applyDefaults applies default values to variables
func (p *TemplatePrompt) applyDefaults(vars map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})

	// Copy existing variables
	for k, v := range vars {
		result[k] = v
	}

	// Apply defaults for missing variables
	for _, variable := range p.Variables() {
		if _, ok := result[variable.Name]; !ok && variable.DefaultValue != nil {
			result[variable.Name] = variable.DefaultValue
		}
	}

	return result
}

// extractVariables extracts variables from template content
func extractVariables(content string) []core.PromptVariable {
	variables := make(map[string]bool)
	varList := []core.PromptVariable{}

	// Simple {{variable}} pattern
	simpleVarRegex := regexp.MustCompile(`\{\{([^}#/]+)\}\}`)
	matches := simpleVarRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		varName := strings.TrimSpace(match[1])

		// Skip special directives
		if strings.Contains(varName, ":") {
			continue
		}

		if !variables[varName] {
			variables[varName] = true
			varList = append(varList, core.PromptVariable{
				Name:        varName,
				Description: fmt.Sprintf("Variable %s", varName),
				Type:        "string",
				Required:    false, // Default to optional
			})
		}
	}

	// TODO: Also extract variables from template syntax {{.variable}}

	return varList
}
