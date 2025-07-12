//go:build windows

package cli

import "github.com/spf13/cobra"

func addMainCommands(rootCmd *cobra.Command) {
	rootCmd.AddCommand(
		RunCmd(),
		RefactorCmd(),
	)
}
