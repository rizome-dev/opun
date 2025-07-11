package mcp

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
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/rizome-dev/opun/internal/config"
	"github.com/rizome-dev/opun/pkg/core"
)

// SharedMCPInstaller handles MCP server installation using the shared configuration
type SharedMCPInstaller struct {
	configManager *config.SharedConfigManager
}

// NewSharedMCPInstaller creates a new shared MCP installer
func NewSharedMCPInstaller() (*SharedMCPInstaller, error) {
	manager, err := config.NewSharedConfigManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create config manager: %w", err)
	}

	return &SharedMCPInstaller{
		configManager: manager,
	}, nil
}

// InstallServers installs the selected MCP servers, checking for existing installations
func (i *SharedMCPInstaller) InstallServers(ctx context.Context, serverNames []string) error {
	fmt.Println("🔍 Checking for existing MCP server installations...")

	toInstall := []string{}
	for _, name := range serverNames {
		installed, version, err := i.configManager.CheckMCPServerInstalled(name)
		if err != nil {
			fmt.Printf("⚠️  Error checking %s: %v\n", name, err)
			toInstall = append(toInstall, name)
			continue
		}

		if installed {
			fmt.Printf("✅ %s is already installed (version: %s)\n", name, version)
			// Update the config to reflect it's installed
			i.configManager.UpdateMCPServerStatus(name, true, version)
		} else {
			fmt.Printf("❌ %s needs to be installed\n", name)
			toInstall = append(toInstall, name)
		}
	}

	if len(toInstall) == 0 {
		fmt.Println("✨ All selected MCP servers are already installed!")
		return nil
	}

	// Check if npm is available
	if !i.hasNPM() {
		return fmt.Errorf("npm is required to install MCP servers. Please install Node.js and npm")
	}

	fmt.Printf("\n📦 Installing %d MCP servers...\n", len(toInstall))

	servers := i.configManager.GetMCPServers()
	for idx, name := range toInstall {
		fmt.Printf("  [%d/%d] Installing %s", idx+1, len(toInstall), name)

		// Find the server configuration
		var server *core.SharedMCPServer
		for _, s := range servers {
			if s.Name == name {
				server = &s
				break
			}
		}

		if server == nil {
			fmt.Printf(" ❌ (server not found in configuration)\n")
			continue
		}

		// Install the server
		if err := i.installSingleServer(ctx, *server); err != nil {
			fmt.Printf(" ❌ (failed: %v)\n", err)
			// Mark as failed in config
			i.configManager.UpdateMCPServerStatus(name, false, "")
			continue
		}

		// Verify installation and get version
		installed, version, _ := i.configManager.CheckMCPServerInstalled(name)
		if installed {
			fmt.Printf(" ✅ (version: %s)\n", version)
			i.configManager.UpdateMCPServerStatus(name, true, version)
		} else {
			fmt.Printf(" ⚠️ (installed but couldn't verify)\n")
		}
	}

	return nil
}

// installSingleServer installs a single MCP server
func (i *SharedMCPInstaller) installSingleServer(ctx context.Context, server core.SharedMCPServer) error {
	// Create a context with timeout if not already set
	installCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Install the package globally
	cmd := exec.CommandContext(installCtx, "npm", "install", "-g", server.Package)

	// Capture output for debugging
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("npm install failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// SyncConfigurations syncs the shared configuration to all providers
func (i *SharedMCPInstaller) SyncConfigurations(providers []string) error {
	fmt.Println("\n🔄 Syncing configuration to providers...")

	for _, provider := range providers {
		fmt.Printf("  Syncing to %s...", provider)

		if err := i.configManager.SyncToProvider(provider); err != nil {
			fmt.Printf(" ❌ (failed: %v)\n", err)
			// Continue with other providers
		} else {
			fmt.Printf(" ✅\n")
		}
	}

	return nil
}

// ValidateEnvironmentVariables checks for required environment variables
func (i *SharedMCPInstaller) ValidateEnvironmentVariables() map[string][]string {
	servers := i.configManager.GetMCPServers()
	missing := make(map[string][]string)

	for _, server := range servers {
		if !server.Installed {
			continue
		}

		var missingVars []string
		for envVar, defaultValue := range server.Env {
			value := os.Getenv(envVar)
			if value == "" && !strings.Contains(defaultValue, "YOUR_") {
				missingVars = append(missingVars, envVar)
			}
		}

		if len(missingVars) > 0 {
			missing[server.Name] = missingVars
		}
	}

	return missing
}

// hasNPM checks if npm is available
func (i *SharedMCPInstaller) hasNPM() bool {
	cmd := exec.Command("npm", "--version")
	return cmd.Run() == nil
}

// GetInstalledServers returns a list of all installed MCP servers
func (i *SharedMCPInstaller) GetInstalledServers() []core.SharedMCPServer {
	var installed []core.SharedMCPServer
	servers := i.configManager.GetMCPServers()

	for _, server := range servers {
		if server.Installed {
			installed = append(installed, server)
		}
	}

	return installed
}
