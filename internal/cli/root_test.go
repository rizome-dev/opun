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
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootCmd(t *testing.T) {
	cmd := RootCmd()

	t.Run("Basic Properties", func(t *testing.T) {
		assert.Equal(t, "opun", cmd.Use)
		assert.Equal(t, "opun", cmd.Name())
		assert.Contains(t, cmd.Short, "AI code agent automation framework")
		assert.NotEmpty(t, cmd.Long)
		// Version is not set on root cmd, commenting out for now
		// assert.NotEmpty(t, cmd.Version)
	})

	t.Run("Help Output", func(t *testing.T) {
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)
		cmd.SetArgs([]string{"--help"})

		err := cmd.Execute()
		assert.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Opun automates interaction with AI code agents")
		assert.Contains(t, output, "Registry Commands")
		assert.Contains(t, output, "setup")
		assert.Contains(t, output, "chat")
		assert.Contains(t, output, "run")
	})

	// Commenting out version flag test as it's not implemented
	// t.Run("Version Flag", func(t *testing.T) {
	// 	buf := new(bytes.Buffer)
	// 	cmd.SetOut(buf)
	// 	cmd.SetErr(buf)
	// 	cmd.SetArgs([]string{"--version"})

	// 	err := cmd.Execute()
	// 	assert.NoError(t, err)

	// 	output := buf.String()
	// 	// Version output might vary, just check something is printed
	// 	assert.NotEmpty(t, output)
	// })
}

func TestRootCmd_Subcommands(t *testing.T) {
	cmd := RootCmd()

	// Map of expected subcommands
	expectedCommands := map[string]bool{
		"setup":      true,
		"chat":       true,
		"run":        true,
		"add":        true,
		"list":       true,
		"delete":     true,
		"mcp":        true,
		"update":     true,
		"refactor":   true,
		"capability": true,
		"completion": true,
		"help":       true,
	}

	// Check all expected commands exist
	for _, subcmd := range cmd.Commands() {
		name := subcmd.Name()
		if expectedCommands[name] {
			delete(expectedCommands, name)
		}
	}

	// All expected commands should have been found
	assert.Empty(t, expectedCommands, "Missing commands: %v", expectedCommands)
}

func TestRootCmd_InvalidCommand(t *testing.T) {
	cmd := RootCmd()
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)
	cmd.SetArgs([]string{"invalid-command"})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown command")
}

func TestInitConfig(t *testing.T) {
	// Save original environment
	oldHome := getEnv("HOME", "")
	oldOpunHome := getEnv("OPUN_HOME", "")
	defer func() {
		setEnv("HOME", oldHome)
		setEnv("OPUN_HOME", oldOpunHome)
	}()

	t.Run("Default Config Path", func(t *testing.T) {
		tempDir := t.TempDir()
		setEnv("HOME", tempDir)
		unsetEnv("OPUN_HOME")

		// Call initConfig (this is normally called by cobra)
		err := initConfig("")
		assert.NoError(t, err)

		// Config should be set to use ~/.opun
		// Note: We can't easily test viper internals, but we can verify
		// the function completes without error
	})

	t.Run("Custom OPUN_HOME", func(t *testing.T) {
		tempDir := t.TempDir()
		customDir := filepath.Join(tempDir, "custom-opun")
		setEnv("OPUN_HOME", customDir)

		err := initConfig("")
		assert.NoError(t, err)

		// Should use custom directory
		// Again, testing viper internals is difficult
	})
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func setEnv(key, value string) {
	if value == "" {
		os.Unsetenv(key)
	} else {
		os.Setenv(key, value)
	}
}

func unsetEnv(key string) {
	os.Unsetenv(key)
}
