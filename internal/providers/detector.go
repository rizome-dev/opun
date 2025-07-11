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
	"fmt"
	"os/exec"
)

// Detector detects available provider commands
type Detector struct{}

// DetectCommand detects the command to use for a provider
func (d *Detector) DetectCommand(provider string) (string, error) {
	switch provider {
	case "claude":
		// Try claude command first
		if _, err := exec.LookPath("claude"); err == nil {
			return "claude", nil
		}
		// Fall back to npx claude-code
		if _, err := exec.LookPath("npx"); err == nil {
			return "npx claude-code", nil
		}
		return "", fmt.Errorf("claude command not found, please install Claude CLI")

	case "gemini":
		if _, err := exec.LookPath("gemini"); err == nil {
			return "gemini", nil
		}
		return "", fmt.Errorf("gemini command not found, please install Gemini CLI")

	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
}
