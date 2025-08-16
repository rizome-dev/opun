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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/rizome-dev/opun/internal/utils"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// RootCmd returns the root command
func RootCmd() *cobra.Command {
	var configFile string

	rootCmd := &cobra.Command{
		Use:   "opun",
		Short: "AI code agent automation framework",
		Long: `Opun automates interaction with AI code agents (Claude Code, Gemini CLI, and Qwen Code)
by managing their interactive sessions and providing workflow orchestration.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return initConfig(configFile)
		},
		// Override default help behavior to show our custom grouped commands
		Run: func(cmd *cobra.Command, args []string) {
			_ = cmd.Help()
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is $HOME/.opun/config.yaml)")

	// Set custom help template
	rootCmd.SetHelpTemplate(customHelpTemplate())

	// Disable the default help command and add our own
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
	})

	// Add custom help command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "help [command]",
		Short: "Help about any command",
		Long: `Help provides help for any command in the application.
Simply type opun help [path to command] for full details.`,
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(c *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			var completions []string
			cmd, _, e := c.Root().Find(args)
			if e != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			if cmd == nil {
				cmd = c.Root()
			}
			for _, subCmd := range cmd.Commands() {
				if subCmd.IsAvailableCommand() {
					completions = append(completions, fmt.Sprintf("%s\t%s", subCmd.Name(), subCmd.Short))
				}
			}
			return completions, cobra.ShellCompDirectiveNoFileComp
		},
		Run: func(c *cobra.Command, args []string) {
			cmd, _, e := c.Root().Find(args)
			if cmd == nil || e != nil {
				c.Printf("Unknown help topic %#q\n", args)
				_ = c.Root().Usage()
			} else {
				_ = cmd.Help()
			}
		},
	})

	// Add Registry commands (configuration management)
	rootCmd.AddCommand(
		AddCmd(),
		UpdateCmd(),
		DeleteCmd(),
		ListCmd(),
	)

	// Add Main commands (user-facing operations)
	addMainCommands(rootCmd)

	// Add Capability commands
	rootCmd.AddCommand(
		capabilityCmd,
	)

	// Add SubAgent command
	rootCmd.AddCommand(
		SubAgentCmd(),
	)

	// Add System commands (internal operations)
	rootCmd.AddCommand(
		SetupCmd(),
		MCPCmd(),
		CompletionCmd(),
	)

	return rootCmd
}

// customHelpTemplate returns a custom help template with grouped commands
func customHelpTemplate() string {
	return `{{.Long}}

Usage:
  {{.UseLine}}

Registry Commands (manage configuration):
  add         Add workflows, prompts, actions, or tools
  update      Update existing configuration
  delete      Delete from configuration
  list        List all configured items

Main Commands:
  chat        Start an interactive chat session
  run         Run a workflow
  refactor    Refactor code files
  subagent    Manage cross-provider subagents

Capability Commands:
  capability  List and search all Opun capabilities

System Commands:
  setup       Configure Opun for first use
  mcp         Manage MCP server
  completion  Generate shell completions

Flags:
{{.LocalFlags.FlagUsages | trimTrailingWhitespaces}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}

Use "{{.CommandPath}} [command] --help" for more information about a command.
`
}

// GetCustomHelp returns the formatted help text for display
func GetCustomHelp() string {
	return `Opun automates interaction with AI code agents (Claude Code, Gemini CLI, and Qwen Code)
by managing their interactive sessions and providing workflow orchestration.

Usage:
  opun [command]

Registry Commands (manage configuration):
  add         Add workflows, prompts, actions, or tools
  update      Update existing configuration  
  delete      Delete from configuration
  list        List all configured items

Main Commands:
  chat        Start an interactive chat session
  run         Run a workflow
  refactor    Refactor code files

Capability Commands:
  capability  List and search all Opun capabilities

System Commands:
  setup       Configure Opun for first use
  mcp         Manage MCP server
  completion  Generate shell completions
  help        Help about any command

Flags:
  --config string   config file (default is $HOME/.opun/config.yaml)
  -h, --help       help for opun

Use "opun [command] --help" for more information about a command.
`
}

// CompletionCmd generates shell completions
func CompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `To load completions:

Bash:
  $ source <(opun completion bash)
  # To load completions for each session, execute once:
  $ opun completion bash > /etc/bash_completion.d/opun

Zsh:
  $ source <(opun completion zsh)
  # To load completions for each session, execute once:
  $ opun completion zsh > "${fpath[1]}/_opun"

Fish:
  $ opun completion fish | source
  # To load completions for each session, execute once:
  $ opun completion fish > ~/.config/fish/completions/opun.fish

PowerShell:
  PS> opun completion powershell | Out-String | Invoke-Expression
  # To load completions for every new session, run:
  PS> opun completion powershell > opun.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletion(os.Stdout)
			}
			return nil
		},
	}
	return cmd
}

func initConfig(configFile string) error {
	if configFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(configFile)
	} else {
		// Search for config in home directory
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		opunDir := filepath.Join(home, ".opun")

		// Create opun directory if it doesn't exist with proper ownership
		if err := utils.EnsureDir(opunDir); err != nil {
			return err
		}

		// Check permissions and warn if incorrect
		if err := checkAndWarnPermissions(opunDir); err != nil {
			// Don't fail, just warn
			fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
		}

		viper.AddConfigPath(opunDir)
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	viper.AutomaticEnv()

	// Read config if it exists
	_ = viper.ReadInConfig()

	return nil
}

// checkAndWarnPermissions checks if the .opun directory has correct ownership
func checkAndWarnPermissions(opunDir string) error {
	// Get actual user info
	actualUser, err := utils.GetActualUser()
	if err != nil {
		return nil // Can't check, skip warning
	}

	// Check directory ownership using ls command
	cmd := exec.Command("ls", "-ld", opunDir)
	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err == nil {
		// Parse the output to get owner
		fields := strings.Fields(out.String())
		if len(fields) >= 3 {
			owner := fields[2]
			if owner != actualUser.Username && owner != "" {
				return fmt.Errorf("~/.opun is owned by %s, not %s. Run 'make fix-permissions' to fix", owner, actualUser.Username)
			}
		}
	}

	// Also check if we can write to the directory
	testFile := filepath.Join(opunDir, ".permission-test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("cannot write to ~/.opun directory. Run 'make fix-permissions' or 'sudo chown -R %s ~/.opun'", actualUser.Username)
	}
	_ = os.Remove(testFile)

	return nil
}
