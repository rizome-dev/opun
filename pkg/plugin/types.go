package plugin

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
	"time"
)

// Plugin represents a plugin that imports prompts, workflows, and actions
type Plugin interface {
	// Metadata
	Name() string
	Description() string
	Author() string
	Repository() string

	// Get imported items
	GetImports() *PluginImports
}

// PluginConfig represents plugin configuration
type PluginConfig struct {
	Settings  map[string]interface{} `json:"settings" yaml:"settings"`
	Resources ResourceConfig         `json:"resources" yaml:"resources"`
}

// ResourceConfig defines resource limits for a plugin
type ResourceConfig struct {
	MaxMemoryMB   int           `json:"max_memory_mb" yaml:"max_memory_mb"`
	MaxCPUPercent int           `json:"max_cpu_percent" yaml:"max_cpu_percent"`
	Timeout       time.Duration `json:"timeout" yaml:"timeout"`
}

// PluginInput represents input to a plugin execution
type PluginInput struct {
	Command   string                 `json:"command"`
	Arguments map[string]interface{} `json:"arguments"`
	Context   map[string]interface{} `json:"context"`
}

// PluginOutput represents output from a plugin execution
type PluginOutput struct {
	Success bool                   `json:"success"`
	Result  interface{}            `json:"result"`
	Error   string                 `json:"error,omitempty"`
	Logs    []string               `json:"logs,omitempty"`
	Metrics map[string]interface{} `json:"metrics,omitempty"`
}

// CommandDefinition defines a command provided by a plugin
type CommandDefinition struct {
	Name        string               `json:"name" yaml:"name"`
	Description string               `json:"description" yaml:"description"`
	Arguments   []ArgumentDefinition `json:"arguments" yaml:"arguments"`
	Examples    []string             `json:"examples" yaml:"examples"`
}

// ArgumentDefinition defines a command argument
type ArgumentDefinition struct {
	Name        string      `json:"name" yaml:"name"`
	Description string      `json:"description" yaml:"description"`
	Type        string      `json:"type" yaml:"type"`
	Required    bool        `json:"required" yaml:"required"`
	Default     interface{} `json:"default" yaml:"default"`
	Choices     []string    `json:"choices" yaml:"choices"`
}

// ToolDefinition defines a tool provided by a plugin
type ToolDefinition struct {
	Name         string                 `json:"name" yaml:"name"`
	Description  string                 `json:"description" yaml:"description"`
	InputSchema  map[string]interface{} `json:"input_schema" yaml:"input_schema"`
	OutputSchema map[string]interface{} `json:"output_schema" yaml:"output_schema"`
}

// ProviderDefinition defines an AI provider provided by a plugin
type ProviderDefinition struct {
	Name         string   `json:"name" yaml:"name"`
	Description  string   `json:"description" yaml:"description"`
	Models       []string `json:"models" yaml:"models"`
	Capabilities []string `json:"capabilities" yaml:"capabilities"`
}

// PluginManifest represents the metadata about a plugin
type PluginManifest struct {
	Name        string         `json:"name" yaml:"name"`
	Description string         `json:"description" yaml:"description"`
	Author      string         `json:"author" yaml:"author"`
	Repository  string         `json:"repository" yaml:"repository"`
	Imports     *PluginImports `json:"imports" yaml:"imports"`
}

// PluginImports defines what a plugin imports
type PluginImports struct {
	Prompts   []PromptImport   `json:"prompts" yaml:"prompts"`
	Workflows []WorkflowImport `json:"workflows" yaml:"workflows"`
	Actions   []ActionImport   `json:"actions" yaml:"actions"`
}

// PromptImport represents an imported prompt
type PromptImport struct {
	Name        string           `json:"name" yaml:"name"`
	Description string           `json:"description" yaml:"description"`
	Category    string           `json:"category" yaml:"category"`
	Template    string           `json:"template" yaml:"template"`
	Providers   []string         `json:"providers" yaml:"providers"`
	Variables   []PromptVariable `json:"variables" yaml:"variables"`
}

// PromptVariable represents a variable in a prompt
type PromptVariable struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description" yaml:"description"`
	Default     string `json:"default" yaml:"default"`
}

// WorkflowImport represents an imported workflow
type WorkflowImport struct {
	Name        string                   `json:"name" yaml:"name"`
	Description string                   `json:"description" yaml:"description"`
	Agents      []map[string]interface{} `json:"agents" yaml:"agents"`
}

// ActionImport represents an imported action
type ActionImport struct {
	Name        string               `json:"name" yaml:"name"`
	Description string               `json:"description" yaml:"description"`
	Category    string               `json:"category" yaml:"category"`
	Command     string               `json:"command" yaml:"command"`
	Arguments   []ArgumentDefinition `json:"arguments" yaml:"arguments"`
	Providers   []string             `json:"providers" yaml:"providers"`
}

// PluginType defines the type of plugin
type PluginType string

const (
	// PluginTypeImport is the new standard plugin type that imports items
	PluginTypeImport PluginType = "import"

	// Legacy plugin types (deprecated)
	PluginTypeCode   PluginType = "code"   // Native Go plugin
	PluginTypeScript PluginType = "script" // Script-based (Python, JS, etc.)
	PluginTypeJSON   PluginType = "json"   // JSON-defined workflows
	PluginTypeWASM   PluginType = "wasm"   // WebAssembly plugin
)

// RuntimeType defines the runtime for a plugin
type RuntimeType string

const (
	RuntimeYAML RuntimeType = "yaml" // New YAML-based import plugins

	// Legacy runtimes (deprecated)
	RuntimeGo         RuntimeType = "go"
	RuntimePython     RuntimeType = "python"
	RuntimeJavaScript RuntimeType = "javascript"
	RuntimeJSON       RuntimeType = "json"
	RuntimeWASM       RuntimeType = "wasm"
)

// PluginState represents the current state of a plugin
type PluginState string

const (
	StateUnloaded    PluginState = "unloaded"
	StateLoading     PluginState = "loading"
	StateLoaded      PluginState = "loaded"
	StateInitialized PluginState = "initialized"
	StateRunning     PluginState = "running"
	StateStopped     PluginState = "stopped"
	StateError       PluginState = "error"
)

// PluginInfo represents runtime information about a plugin
type PluginInfo struct {
	Manifest    PluginManifest         `json:"manifest"`
	State       PluginState            `json:"state"`
	InstalledAt *time.Time             `json:"installed_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Metrics     map[string]interface{} `json:"metrics"`
}

// InstalledPlugin tracks an installed plugin
type InstalledPlugin struct {
	Name        string    `json:"name" yaml:"name"`
	Source      string    `json:"source" yaml:"source"` // file path or URL
	InstalledAt time.Time `json:"installed_at" yaml:"installed_at"`
	ItemCount   ItemCount `json:"item_count" yaml:"item_count"`
}

// ItemCount tracks how many items were imported
type ItemCount struct {
	Prompts   int `json:"prompts" yaml:"prompts"`
	Workflows int `json:"workflows" yaml:"workflows"`
	Actions   int `json:"actions" yaml:"actions"`
}

// PluginEvent represents an event from a plugin
type PluginEvent struct {
	Type       EventType              `json:"type"`
	Timestamp  time.Time              `json:"timestamp"`
	PluginName string                 `json:"plugin_name"`
	Message    string                 `json:"message"`
	Data       map[string]interface{} `json:"data,omitempty"`
}

// EventType represents the type of plugin event
type EventType string

const (
	EventLoaded      EventType = "loaded"
	EventInitialized EventType = "initialized"
	EventStarted     EventType = "started"
	EventStopped     EventType = "stopped"
	EventError       EventType = "error"
	EventExecuting   EventType = "executing"
	EventExecuted    EventType = "executed"
)
