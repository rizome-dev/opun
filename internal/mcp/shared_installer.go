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

// InstallServers marks the selected MCP servers as enabled; servers run on
// demand via their configured command (npx -y fetches packages as needed)
func (i *SharedMCPInstaller) InstallServers(ctx context.Context, serverNames []string) error {
	fmt.Println("🔍 Configuring MCP servers...")

	for _, name := range serverNames {
		available, version, err := i.configManager.CheckMCPServerInstalled(name)
		if err != nil {
			fmt.Printf("⚠️  Error checking %s: %v\n", name, err)
			continue
		}

		if !available {
			fmt.Printf("⚠️  %s: command not found on PATH, skipping\n", name)
			continue
		}

		if err := i.configManager.UpdateMCPServerStatus(name, true, version); err != nil {
			fmt.Printf("⚠️  Failed to update status for %s: %v\n", name, err)
			continue
		}
		fmt.Printf("✅ %s configured (version: %s)\n", name, version)
	}

	return nil
}

// SyncConfigurations syncs the shared configuration to installed providers
func (i *SharedMCPInstaller) SyncConfigurations(providers []string) error {
	fmt.Println("\n🔄 Syncing configuration to providers...")

	for _, provider := range providers {
		if _, err := exec.LookPath(provider); err != nil {
			fmt.Printf("  Skipping %s (CLI not installed)\n", provider)
			continue
		}

		fmt.Printf("  Syncing to %s...", provider)

		if err := i.configManager.SyncToProvider(provider); err != nil {
			fmt.Printf(" ❌ (failed: %v)\n", err)
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
