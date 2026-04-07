package cmd

import (
	"fmt"
	"os"

	"github.com/joelhelbling/glovebox/internal/runtime"
	"github.com/spf13/cobra"
)

var (
	// rt is the active container runtime, set during PersistentPreRunE.
	rt runtime.Runtime

	// runtimeOverride is set via the --runtime flag.
	runtimeOverride string
)

var rootCmd = &cobra.Command{
	Use:   "glovebox",
	Short: "A composable, sandboxed development environment",
	Long: `Glovebox creates sandboxed containers for running untrusted or
experimental code. It uses a mod-based system to compose your perfect
development environment from modular, reusable pieces.

Supports multiple container runtimes. Use --runtime to override auto-detection.`,
	Version: Version,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip runtime detection for commands that don't need it
		switch cmd.Name() {
		case "help", "version", "init", "mod":
			return nil
		}
		// Also skip if this is a child of "mod" (e.g., "mod list")
		if cmd.Parent() != nil && cmd.Parent().Name() == "mod" {
			return nil
		}

		result, err := runtime.Detect(runtimeOverride, runtime.Stdio{
			Stdin:  os.Stdin,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		})
		if err != nil {
			return err
		}
		if result.FellBack {
			colorYellow.Println(result.FallbackMsg)
			fmt.Println()
		}
		rt = result.Runtime
		return nil
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&runtimeOverride, "runtime", "", "Container runtime to use (e.g., docker)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
