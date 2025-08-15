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
		Use:   "chat [provider] [-- additional-args...]",
		Short: "Start an interactive chat session with an AI provider",
		Long: `Start an interactive chat session with Claude or Gemini.

If no provider is specified, uses the default provider from your configuration.
Your promptgarden prompts and configured slash commands are available through the injection system.

Additional arguments after -- will be passed directly to the underlying provider.

Examples:
  opun chat                          # Use default provider
  opun chat claude                   # Chat with Claude
  opun chat gemini                   # Chat with Gemini
  opun chat claude -- --continue     # Chat with Claude using --continue flag
  opun chat gemini -- --model=pro    # Chat with Gemini using specific model
  opun chat -- --continue            # Use default provider with --continue flag`,
		Args: cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var provider string
			var providerArgs []string

			// Known provider names
			knownProviders := map[string]bool{
				"claude": true,
				"gemini": true,
			}

			// Parse provider and additional arguments
			if len(args) == 0 {
				// No provider specified, use default from config
				provider = viper.GetString("default_provider")
				if provider == "" {
					return fmt.Errorf("no provider specified and no default provider configured. Run 'opun setup' to configure a default provider")
				}
			} else {
				// Check if first argument is a known provider
				firstArg := strings.ToLower(args[0])
				if knownProviders[firstArg] {
					// First argument is a provider name
					provider = firstArg
					// Any additional args (after provider name) are passed to the provider
					if len(args) > 1 {
						providerArgs = args[1:]
					}
				} else {
					// First argument is not a provider name, use default provider
					// and treat all arguments as provider arguments
					provider = viper.GetString("default_provider")
					if provider == "" {
						return fmt.Errorf("no default provider configured and '%s' is not a known provider. Run 'opun setup' to configure a default provider", args[0])
					}
					providerArgs = args
				}
			}

			return runChat(cmd, provider, providerArgs)
		},
	}

	return cmd
}
