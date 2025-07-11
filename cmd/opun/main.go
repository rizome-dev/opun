package main

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
	"os/signal"
	"syscall"
	"time"

	"github.com/charmbracelet/fang"
	"github.com/rizome-dev/opun/internal/cli"
	"github.com/rizome-dev/opun/internal/utils"
)

var (
	version   = "dev"
	commit    = "none"
	buildTime = "unknown"
)

func main() {
	// Set up signal handling for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Fprintln(os.Stderr, "\nReceived interrupt signal, shutting down gracefully...")
		cancel()

		// Run all registered cleanup functions
		cleanupDone := make(chan struct{})
		go func() {
			utils.RunCleanup()
			close(cleanupDone)
		}()

		// Wait for cleanup with timeout
		select {
		case <-cleanupDone:
			// Cleanup completed
		case <-time.After(3 * time.Second):
			fmt.Fprintln(os.Stderr, "Cleanup timeout exceeded, forcing exit")
		}

		os.Exit(130) // Standard exit code for SIGINT
	}()

	// Get root command
	rootCmd := cli.RootCmd()

	// Check if user is requesting help
	if len(os.Args) == 1 || (len(os.Args) == 2 && (os.Args[1] == "--help" || os.Args[1] == "-h" || os.Args[1] == "help")) {
		// Display our custom grouped help
		fmt.Print(cli.GetCustomHelp())
		os.Exit(0)
	}

	// Use fang for enhanced CLI experience for all other commands
	if err := fang.Execute(ctx, rootCmd); err != nil {
		// Don't print error if context was cancelled (user interrupted)
		if ctx.Err() != context.Canceled {
			os.Exit(1)
		}
	}
}
