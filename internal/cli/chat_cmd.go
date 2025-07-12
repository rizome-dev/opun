package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// ChatCmd creates the chat command
func ChatCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "chat [provider]",
		Short: "Start an interactive chat session with an AI provider",
		Long: `Start an interactive chat session with Claude or Gemini.

If no provider is specified, uses the default provider from your configuration.
Your promptgarden prompts and configured slash commands are available through the injection system.

Examples:
  opun chat          # Use default provider
  opun chat claude   # Chat with Claude
  opun chat gemini   # Chat with Gemini`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var provider string

			// If no provider specified, use default from config
			if len(args) == 0 {
				provider = viper.GetString("default_provider")
				if provider == "" {
					return fmt.Errorf("no provider specified and no default provider configured. Run 'opun setup' to configure a default provider")
				}
			} else {
				provider = strings.ToLower(args[0])
			}

			return runChat(cmd, []string{provider})
		},
	}

	return cmd
}
