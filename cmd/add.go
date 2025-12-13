package cmd

import (
	"fmt"
	"os"

	"github.com/joelhelbling/glovebox/internal/mod"
	"github.com/joelhelbling/glovebox/internal/profile"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add <mod>",
	Short: "Add a mod to your profile",
	Long: `Add a mod to your glovebox profile.

Run 'glovebox mod list' to see available mods.

To create your own custom mod, run:
  glovebox mod create <name>

Examples:
  glovebox add shells/fish
  glovebox add ai/claude-code
  glovebox add custom/my-tool`,
	Args: cobra.ExactArgs(1),
	RunE: runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	modID := args[0]

	// Verify mod exists
	if _, err := mod.Load(modID); err != nil {
		return fmt.Errorf("mod '%s' not found. Run 'glovebox mod list' to see available mods", modID)
	}

	// Load effective profile
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	p, err := profile.LoadEffective(cwd)
	if err != nil {
		return err
	}

	if p == nil {
		return fmt.Errorf("no profile found. Run 'glovebox init' first")
	}

	// Add mod
	if !p.AddMod(modID) {
		fmt.Printf("Mod '%s' is already in your profile.\n", modID)
		return nil
	}

	// Save profile
	if err := p.Save(); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}

	colorGreen.Printf("✓ Added '%s' to profile\n", modID)
	fmt.Println("\nRun 'glovebox build' to regenerate your Dockerfile.")

	return nil
}
