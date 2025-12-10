package cmd

import (
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/joelhelbling/glovebox/internal/profile"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove <snippet>",
	Aliases: []string{"rm"},
	Short:   "Remove a snippet from your profile",
	Long: `Remove a snippet from your glovebox profile.

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
	snippetID := args[0]

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

	// Remove snippet
	if !p.RemoveSnippet(snippetID) {
		fmt.Printf("Snippet '%s' is not in your profile.\n", snippetID)
		return nil
	}

	// Save profile
	if err := p.Save(); err != nil {
		return fmt.Errorf("saving profile: %w", err)
	}

	green := color.New(color.FgGreen)
	green.Printf("✓ Removed '%s' from profile\n", snippetID)
	fmt.Println("\nRun 'glovebox build' to regenerate your Dockerfile.")

	return nil
}
