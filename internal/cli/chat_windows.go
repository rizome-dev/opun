//go:build windows

package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func runChat(cmd *cobra.Command, args []string) error {
	return fmt.Errorf("interactive chat is not yet supported on Windows")
}
