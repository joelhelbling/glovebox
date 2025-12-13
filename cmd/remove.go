package cmd

import (
	"fmt"
	"os"

	"github.com/joelhelbling/glovebox/internal/profile"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove <mod>",
	Aliases: []string{"rm"},
	Short:   "Remove a mod from your profile",
	Long: `Remove a mod from your glovebox profile.

Example:
  glovebox remove ai/opencode
  glovebox rm shells/zsh`,
	Args: cobra.ExactArgs(1),
	RunE: runRemove,
}

func init() {
	rootCmd.AddCommand(removeCmd)
}

func runRemove(cmd *cobra.Command, args []string) error {
	modID := args[0]

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

	// Remove mod
	if !p.RemoveMod(modID) {
		fmt.Printf("Mod '%s' is not in your profile.\n", modID)
		return nil
	}

	// Save profile
	if err := p.Save(); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}

	colorGreen.Printf("✓ Removed '%s' from profile\n", modID)
	fmt.Println("\nRun 'glovebox build' to regenerate your Dockerfile.")

	return nil
}
