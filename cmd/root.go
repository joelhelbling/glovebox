package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "glovebox",
	Short: "A composable, sandboxed development environment",
	Long: `Glovebox creates sandboxed Docker containers for running untrusted or
experimental code. It uses a snippet-based system to compose your perfect
development environment from modular, reusable pieces.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
